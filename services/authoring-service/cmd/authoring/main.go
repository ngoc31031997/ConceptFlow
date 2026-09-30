// Command authoring is authoring-service's composition root: the prompt library,
// the 1a/1b/1c authoring chain (story → storyboard → code) with its LLM calls,
// the metadata and short-script suggestions, and the LLM usage log (CR-040 FR111).
//
// It owns its own database. Everything about the project itself — status,
// settings, video formats, voice calibration and the project journal — is read
// and appended through the orchestrator's internal HTTP API.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpadapter "authoring/internal/adapters/http"
	"authoring/internal/adapters/llm"
	"authoring/internal/adapters/orchestrator"
	"authoring/internal/adapters/postgres"
	"authoring/internal/adapters/rendering"
	"authoring/internal/application"
	"authoring/internal/config"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL, cfg.DatabaseMaxConns)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// CR-031: the prompt library. Legacy override rows/tables are purged first so
	// none of them can hold an active slot when the system rows are seeded.
	// (Per the re-seed memory: seeding upserts, so a changed seed needs a rebuild
	// and restart of this service, and prompt_overrides must not shadow it.)
	authoringRepo := postgres.NewPromptTemplateRepository(pool)
	if n, err := authoringRepo.PurgeLegacyPrompts(ctx); err != nil {
		logger.Warn("could not purge legacy prompts", "error", err)
	} else if n > 0 {
		logger.Warn("purged legacy prompts", "count", n)
	}
	if err := authoringRepo.SeedPrompts(ctx); err != nil {
		logger.Warn("could not seed system prompts", "error", err)
	}

	projects := orchestrator.NewClient(cfg.OrchestratorURL, cfg.OrchestratorTimeout)

	// CR-039 — the one path to a language model.
	llmClient := llm.NewClient(cfg.LLMServiceURL, cfg.LLMServiceTimeout)
	var llmProvider application.LLMProviderPort = llmClient
	logger.Info("llm-service configured", "url", cfg.LLMServiceURL, "model", cfg.HiveModel)
	llmUsageRepo := postgres.NewLLMUsageRepository(pool)
	llmUsageRecorder := application.NewLLMUsageRecorder(llmUsageRepo, logger)

	suggestPublishMetadata := application.NewSuggestPublishMetadataUseCase(projects, llmClient)
	suggestShortScript := application.NewSuggestShortScriptUseCase(llmClient)

	if err := authoringRepo.SeedVideoArchetypes(ctx); err != nil {
		logger.Warn("could not seed system video archetypes", "error", err)
	}
	archetypes := application.NewVideoArchetypesUseCase(authoringRepo)

	// CR-044 — the illustration library, previews rendered by the rendering service.
	if err := authoringRepo.SeedIllustrations(ctx); err != nil {
		logger.Warn("could not seed the illustration library", "error", err)
	}
	illustrations := application.NewIllustrationsUseCase(
		authoringRepo, rendering.NewClient(cfg.RenderingURL, cfg.RenderingTimeout)).
		WithDrawer(llmProvider, llmUsageRecorder, cfg.HiveMaxOutputTokens).
		// CR-052: a drawing a video still working uses cannot be deleted.
		WithProjectStatus(projects)
	// CR-044 — each Remotion video's drawings, gated before the code step.
	projectIllustrations := application.NewProjectIllustrationsUseCase(
		authoringRepo, illustrations, authoringRepo, llmProvider, llmUsageRecorder, cfg.HiveMaxOutputTokens).
		WithDrawConcurrency(cfg.IllustrationDrawConcurrency)

	prompts := application.NewPromptsUseCase(authoringRepo)
	// CR-028 FR84.2: every authoring save shares the same lock check (the project
	// must still be a draft, read from the orchestrator), and clears the steps
	// built on the one it overwrote.
	saveAuthoringStory := application.NewSaveAuthoringStoryUseCase(authoringRepo, projects, authoringRepo)
	saveAuthoringStoryboard := application.NewSaveAuthoringStoryboardUseCase(authoringRepo, projects, authoringRepo)
	saveAuthoringCode := application.NewSaveAuthoringCodeUseCase(authoringRepo, projects, authoringRepo)
	getAuthoringState := application.NewGetAuthoringStateUseCase(authoringRepo)
	saveAuthoringMode := application.NewSaveAuthoringModeUseCase(authoringRepo)
	saveAuthoringModels := application.NewSaveAuthoringModelsUseCase(authoringRepo)

	renderContext := orchestrator.PromptRenderContext{Projects: projects, Authoring: authoringRepo}
	// CR-027 FR77.1 — ONE renderer, shared by the Copy button and the generate endpoint.
	renderPrompt := application.NewRenderPromptUseCase(authoringRepo, renderContext, projects, projects).WithArchetypes(authoringRepo)
	generateAuthoring := application.NewGenerateAuthoringUseCase(
		renderPrompt, llmProvider, llmUsageRecorder,
		renderContext,
		authoringRepo,
		saveAuthoringStory, saveAuthoringStoryboard, saveAuthoringCode,
		cfg.HiveMaxInputChars, cfg.HiveMaxOutputTokens,
	).WithErrorLog(projects).WithEvents(projects).WithPipeline(llmClient, llmClient).
		WithIllustrations(projectIllustrations).
		// CR-048 T8 — the post-1b length check reads the same format and voice
		// calibration the outline prompt's beat sheet is built from.
		WithStoryboardChecks(projects, projects).
		// CR-050 Unit 2 (ADR-0030) — the code step stored segment by segment.
		WithSegments(authoringRepo)
	// NFR-3: nothing can be running yet, so a segment still marked running was
	// cut off when this service last stopped.
	if n, err := generateAuthoring.FailInterruptedSegments(ctx); err != nil {
		logger.Warn("could not mark interrupted code segments failed", "error", err)
	} else if n > 0 {
		logger.Warn("marked interrupted code segments failed", "count", n)
	}

	router := httpadapter.NewRouter(suggestPublishMetadata, authoringRepo).
		WithShortScriptSuggester(suggestShortScript).
		WithOperations(application.NewOperations()).
		WithPrompts(prompts).
		WithArchetypes(archetypes).
		WithIllustrations(illustrations).
		WithProjectIllustrations(projectIllustrations).
		WithRenderPrompt(renderPrompt).
		WithAuthoringStory(saveAuthoringStory).
		WithAuthoringStoryboard(saveAuthoringStoryboard).
		WithAuthoringCode(saveAuthoringCode).
		WithAuthoringState(getAuthoringState).
		WithAuthoringMode(saveAuthoringMode).
		WithAuthoringModels(saveAuthoringModels).
		WithDefaultModel(cfg.HiveModel).
		WithUsageStats(llmUsageRepo).
		WithGenerateAuthoring(generateAuthoring)
	router = router.WithAuthoringChain(application.NewAuthoringChainRunner(generateAuthoring, func(err error) string {
		_, msg := httpadapter.DescribeGenerateError(err)
		return msg
	}))

	server := &http.Server{Addr: ":" + cfg.HTTPPort, Handler: router.Handler()}
	go func() {
		logger.Info("authoring-service http server starting", "port", cfg.HTTPPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("http server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("http server shutdown error", "error", err)
	}
}
