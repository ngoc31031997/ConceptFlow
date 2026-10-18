// Command authoring is authoring-service's composition root: the prompt library,
// the 1a/1b/1c authoring chain (story → storyboard → code) with its LLM calls,
// the metadata suggestions, and the LLM usage log.
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
	"authoring/internal/domain"
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

	// The prompt library. Legacy override rows/tables are purged first so
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

	// The one path to a language model.
	llmClient := llm.NewClient(cfg.LLMServiceURL, cfg.LLMServiceTimeout)
	var llmProvider application.LLMProviderPort = llmClient
	logger.Info("llm-service configured", "url", cfg.LLMServiceURL, "model", cfg.HiveModel)
	llmUsageRepo := postgres.NewLLMUsageRepository(pool)
	llmUsageRecorder := application.NewLLMUsageRecorder(llmUsageRepo, logger)

	suggestPublishMetadata := application.NewSuggestPublishMetadataUseCase(projects, llmClient).WithModels(authoringRepo)
	defaultModel := application.NewDefaultModelUseCase(authoringRepo)
	llmClient.WithDefaultModel(authoringRepo)

	if err := authoringRepo.SeedVideoArchetypes(ctx); err != nil {
		logger.Warn("could not seed system video archetypes", "error", err)
	}
	archetypes := application.NewVideoArchetypesUseCase(authoringRepo)

	// The visual languages: how the channel's videos are drawn. Seeded before
	// anything can create a project, so every authoring row pins a language.
	if err := authoringRepo.SeedVisualLanguages(ctx); err != nil {
		logger.Error("could not seed the visual languages", "error", err)
		os.Exit(1)
	}
	renderingClient := rendering.NewClient(cfg.RenderingURL, cfg.RenderingTimeout)
	visualLanguages := application.NewVisualLanguagesUseCase(authoringRepo, renderingClient)
	go keepLanguageSamples(ctx, logger, visualLanguages)

	// The illustration library, previews rendered by the rendering service.
	if err := authoringRepo.SeedIllustrations(ctx); err != nil {
		logger.Warn("could not seed the illustration library", "error", err)
	}
	illustrations := application.NewIllustrationsUseCase(authoringRepo, renderingClient).
		WithDrawer(llmProvider, llmUsageRecorder, cfg.HiveMaxOutputTokens).
		// A drawing a video still working uses cannot be deleted.
		WithProjectStatus(projects).
		// The rules and the kit a drawing is made under are its language's.
		WithLanguages(visualLanguages)
	// Each Remotion video's drawings, gated before the code step.
	projectIllustrations := application.NewProjectIllustrationsUseCase(
		authoringRepo, illustrations, authoringRepo, llmProvider, llmUsageRecorder, cfg.HiveMaxOutputTokens).
		WithDrawConcurrency(cfg.IllustrationDrawConcurrency)

	prompts := application.NewPromptsUseCase(authoringRepo)
	// Every authoring save shares the same lock check (the project
	// must still be a draft, read from the orchestrator), and clears the steps
	// built on the one it overwrote.
	saveAuthoringStory := application.NewSaveAuthoringStoryUseCase(authoringRepo, projects, authoringRepo)
	saveAuthoringStoryboard := application.NewSaveAuthoringStoryboardUseCase(authoringRepo, projects, authoringRepo)
	saveAuthoringCode := application.NewSaveAuthoringCodeUseCase(authoringRepo, projects, authoringRepo)
	getAuthoringState := application.NewGetAuthoringStateUseCase(authoringRepo)
	saveAuthoringMode := application.NewSaveAuthoringModeUseCase(authoringRepo)
	saveAuthoringModels := application.NewSaveAuthoringModelsUseCase(authoringRepo)

	// The idea step's creative brief, the first kind of review unit; every
	// review unit is read, edited, redone and approved through reviewUnits.
	creativeBrief := application.NewCreativeBriefUseCase(
		authoringRepo, authoringRepo, projects, projects, authoringRepo,
		llmProvider, llmUsageRecorder, cfg.HiveMaxOutputTokens)
	reviewUnits := application.NewReviewUnitsUseCase(authoringRepo, projects).
		Register(domain.UnitKindBrief, creativeBrief)

	renderContext := orchestrator.PromptRenderContext{Projects: projects, Authoring: authoringRepo}
	// ONE renderer, shared by the Copy button and the generate endpoint.
	renderPrompt := application.NewRenderPromptUseCase(authoringRepo, renderContext, projects, projects).
		WithArchetypes(authoringRepo).
		WithBrief(creativeBrief).
		WithLanguages(visualLanguages)
	generateAuthoring := application.NewGenerateAuthoringUseCase(
		renderPrompt, llmProvider, llmUsageRecorder,
		renderContext,
		authoringRepo,
		saveAuthoringStory, saveAuthoringStoryboard, saveAuthoringCode,
		cfg.HiveMaxInputChars, cfg.HiveMaxOutputTokens,
	).WithErrorLog(projects).WithEvents(projects).WithPipeline(llmClient, llmClient).
		WithIllustrations(projectIllustrations).
		WithLanguages(visualLanguages).
		// The post-1b length check reads the same format and voice
		// calibration the outline prompt's beat sheet is built from.
		WithStoryboardChecks(projects, projects).
		// The code step stored segment by segment. See ADR-0030.
		WithSegments(authoringRepo)
	// NFR-3: nothing can be running yet, so a segment still marked running was
	// cut off when this service last stopped.
	if n, err := generateAuthoring.FailInterruptedSegments(ctx); err != nil {
		logger.Warn("could not mark interrupted code segments failed", "error", err)
	} else if n > 0 {
		logger.Warn("marked interrupted code segments failed", "count", n)
	}

	router := httpadapter.NewRouter(suggestPublishMetadata, authoringRepo).
		WithOperations(application.NewOperations()).
		WithPrompts(prompts).
		WithArchetypes(archetypes).
		WithVisualLanguages(visualLanguages).
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
		WithDefaultModelSetting(defaultModel).
		WithUsageStats(llmUsageRepo).
		WithGenerateAuthoring(generateAuthoring).
		WithReviewUnits(reviewUnits).
		WithCreativeBrief(creativeBrief)
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

const (
	// languageSampleRetryDelay is how long to wait before trying to render the
	// language samples again after rendering was not reachable.
	languageSampleRetryDelay = 30 * time.Second
	// stillPruneInterval and stillMinAge: stills no language page shows are
	// deleted daily once they are a day old.
	stillPruneInterval = 24 * time.Hour
	stillMinAge        = 24 * time.Hour
)

// keepLanguageSamples renders the sample scenes of every language that has
// none (or whose kit changed), retrying while rendering is not up, then prunes
// the shared stills once a day.
func keepLanguageSamples(ctx context.Context, logger *slog.Logger, languages *application.VisualLanguagesUseCase) {
	for {
		err := languages.EnsureSamples(ctx)
		if err == nil {
			break
		}
		logger.Warn("could not render the visual language samples yet", "error", err)
		select {
		case <-ctx.Done():
			return
		case <-time.After(languageSampleRetryDelay):
		}
	}
	ticker := time.NewTicker(stillPruneInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			removed, err := languages.PruneStills(ctx, stillMinAge)
			if err != nil {
				logger.Warn("could not prune the shared stills", "error", err)
				continue
			}
			logger.Info("pruned the shared stills", "removed", len(removed))
		}
	}
}
