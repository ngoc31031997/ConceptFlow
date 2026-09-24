// Command orchestrator is the composition root — wires config, PostgreSQL,
// RabbitMQ, the 4 use cases, the HTTP server, the AMQP consumer, and the
// OutboxRelay in the order documented in dependency-injection.md's
// "Composition Root" (10 steps, numbered below).
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	amqplib "github.com/rabbitmq/amqp091-go"

	"orchestrator/internal/adapters/amqp"
	httpadapter "orchestrator/internal/adapters/http"
	"orchestrator/internal/adapters/llm"
	"orchestrator/internal/adapters/postgres"
	"orchestrator/internal/application"
	"orchestrator/internal/config"
	"orchestrator/internal/domain"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 1. Load config from env vars.
	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// 2. Connect PostgreSQL (pgx.Pool), bootstrap schema.
	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL, cfg.DatabaseMaxConns)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// 3. Connect RabbitMQ (amqp091-go) via ConnectionManager, which owns
	// reconnect-with-backoff for the lifetime of the process (ADR-0022) —
	// a bare channel does not recover on its own once the broker closes it.
	connMgr := amqp.NewConnectionManager(cfg.RabbitMQURL, cfg.RabbitMQReconnectInitialDelay, cfg.RabbitMQReconnectMaxDelay, logger)
	if err := connMgr.Connect(ctx); err != nil {
		logger.Error("failed to connect to rabbitmq", "error", err)
		os.Exit(1)
	}

	// 4. Construct postgres.ProjectRepository, InboxRepository, OutboxRepository.
	projectRepo := postgres.NewProjectRepository(pool)
	// CR-019 FR51.2: gieo các format dựng sẵn nếu chưa có. Insert-if-absent,
	// không upsert — khi Creator đã sửa một format rồi thì lần khởi động sau
	// không được lặng lẽ khôi phục lại số liệu gốc bên dưới.
	if err := projectRepo.SeedVideoFormats(ctx); err != nil {
		logger.Warn("could not seed video formats", "error", err)
	}
	// CR-031: the prompt library. Legacy override rows/tables are purged first
	// so none of them can hold an active slot when the system rows are seeded.
	promptTemplateRepo := postgres.NewPromptTemplateRepository(pool)
	if n, err := promptTemplateRepo.PurgeLegacyPrompts(ctx); err != nil {
		logger.Warn("could not purge legacy prompts", "error", err)
	} else if n > 0 {
		logger.Warn("purged legacy prompts", "count", n)
	}
	if err := promptTemplateRepo.SeedPrompts(ctx); err != nil {
		logger.Warn("could not seed system prompts", "error", err)
	}
	inboxRepo := postgres.NewInboxRepository(pool)
	outboxRepo := postgres.NewOutboxRepository(pool)

	// 5. Construct amqp.Publisher (implements CommandPublisherPort + ProgressPublisherPort).
	realPublisher := amqp.NewPublisher(connMgr)

	// 6. Construct the 4 use cases. Use cases that dispatch commands are
	// injected with outboxRepo (Outbox-backed CommandPublisherPort — writes
	// to outbox_events, no direct AMQP call, ADR-0019) rather than
	// realPublisher directly, so a command dispatch survives a crash between
	// the state update and the network send.
	startRenderSaga := application.NewStartRenderSagaUseCase(projectRepo, outboxRepo)
	// CR-021 D5/D6: the publish gate reads the QC report the saga stored, and
	// QC_ENFORCE (default false) decides whether a blocking finding actually
	// stops the Saga or is only shown.
	qcReportRepo := postgres.NewQCReportRepository(pool)
	startPublishSaga := application.NewStartPublishSagaUseCase(projectRepo, outboxRepo).
		WithQCGate(qcReportRepo, cfg.QCEnforce)
	// CR-023 correction: there is no HTTP server between backend services —
	// Orchestrator resolves the active intro/outro from its own local
	// channel_asset_pointers projection (kept current by subscribing to
	// channel_asset_rendered/channel_asset_normalized events in
	// handle_step_event.go), not by calling video-assembly over HTTP.
	channelAssetPointers := postgres.NewChannelAssetPointerRepository(pool)
	handleStepEvent := application.NewHandleStepEventUseCase(projectRepo, outboxRepo, realPublisher, channelAssetPointers, logger).
		WithQCReports(qcReportRepo)
	retryStep := application.NewRetryStepUseCase(projectRepo, outboxRepo)
	ollamaClient := llm.NewOllamaClient(cfg.OllamaURL, cfg.OllamaModel, cfg.OllamaTimeout)
	suggestPublishMetadata := application.NewSuggestPublishMetadataUseCase(projectRepo, ollamaClient)
	suggestShortScript := application.NewSuggestShortScriptUseCase(ollamaClient)

	// CR-027 — the LLM provider the authoring pipeline talks to. Both
	// adapters are built regardless of LLM_PROVIDER: the Ollama one is the
	// declared fallback for the light tasks, and building it costs an http
	// client. Without a Hive key the provider is Ollama and the pipeline's
	// generate endpoints simply stay unavailable, which is how the system
	// ran before CR-027 (FR83.2).
	ollamaProvider := llm.NewOllamaProvider(ollamaClient)
	var llmProvider application.LLMProviderPort = ollamaProvider
	if cfg.LLMProvider == "hive" && cfg.HiveAPIKey != "" {
		llmProvider = llm.NewHiveClient(cfg.HiveBaseURL, cfg.HiveAPIKey, cfg.HiveModel, cfg.HiveTimeout, cfg.HiveMaxRetries)
	}
	logger.Info("llm provider selected", "provider", llmProvider.Name(), "model", cfg.HiveModel)

	// CR-027 FR82 — the usage ledger. Constructed before the first billable
	// call can happen, so no call ever runs unmeasured.
	llmUsageRecorder := application.NewLLMUsageRecorder(postgres.NewLLMUsageRepository(pool), logger)


	// 7. Construct amqp.Consumer, register orchestrator.events + 6 DLQ queues,
	// wire HandleStepEventUseCase. Re-run Start after every reconnect
	// (ADR-0022) — a broker reconnect implicitly drops all consumers, and
	// Consumer.Start is safe to call again (its old delivery loops already
	// exited on their own when the previous channel closed).
	consumer := amqp.NewConsumer(connMgr, inboxRepo, handleStepEvent, projectRepo, logger)
	if err := consumer.Start(ctx); err != nil {
		logger.Error("failed to start amqp consumer", "error", err)
		os.Exit(1)
	}
	connMgr.OnReconnect(func(ch *amqplib.Channel) error {
		return consumer.Start(ctx)
	})

	// 8. Start postgres.OutboxRelay as a background goroutine.
	relay := postgres.NewOutboxRelay(outboxRepo, realPublisher, time.Duration(cfg.OutboxPollIntervalMS)*time.Millisecond, logger)
	go relay.Run(ctx)

	// 9. Construct chi router, wire the 4 REST handlers to their use cases.
	// CR-024: cổng duyệt dàn ý dùng lại đúng nhánh chọn TTS mà handleStepEvent
	// đã sở hữu, thay vì dựng một bản thứ hai của cùng quyết định.
	reviewOutline := application.NewReviewOutlineUseCase(projectRepo, handleStepEvent, realPublisher, outboxRepo, logger)
	// CR-023 correction: Normalize dispatches normalize_channel_asset via the
	// Outbox (same durability guarantee as every other command), Preview
	// reads the local channel_asset_pointers projection.
	channelAssets := application.NewChannelAssetsUseCase(outboxRepo, channelAssetPointers)
	// CR-025: prompt-template CRUD (admin editor + web-gui runtime read) and
	// step 1's story-save endpoint.
	prompts := application.NewPromptsUseCase(promptTemplateRepo)
	// CR-028 FR84.2: every authoring save shares the same lock check (project
	// must still be status=draft), and clears the steps built on the one it
	// overwrote — both live on promptTemplateRepo, right alongside the
	// project_authoring table itself.
	saveAuthoringStory := application.NewSaveAuthoringStoryUseCase(promptTemplateRepo, promptTemplateRepo, promptTemplateRepo)
	// CR-025 step 2: Visual Director's storyboard save, and the shared
	// read-side use case both steps' rehydration relies on.
	saveAuthoringStoryboard := application.NewSaveAuthoringStoryboardUseCase(promptTemplateRepo, promptTemplateRepo, promptTemplateRepo)
	// CR-025 step 3: Manim Engineer's code save, sharing the same read-side
	// use case. CR-030 đã bỏ hẳn bước 4 (Script Reviewer).
	saveAuthoringCode := application.NewSaveAuthoringCodeUseCase(promptTemplateRepo, promptTemplateRepo, promptTemplateRepo)
	getAuthoringState := application.NewGetAuthoringStateUseCase(promptTemplateRepo)
	// CR-027 FR79 — the step-1 working mode, stored per project so the choice
	// survives a reload, another browser, and a restart of this service.
	saveAuthoringMode := application.NewSaveAuthoringModeUseCase(promptTemplateRepo)
	// Model-per-step picker (follow-up to CR-027 FR79) — which Hive model
	// each of 1a/1b/1c calls, stored the same place and the same way as the
	// mode above.
	saveAuthoringModels := application.NewSaveAuthoringModelsUseCase(promptTemplateRepo)
	// CR-028 FR83: the project row is created here, at wizard step 1
	// (POST /v1/projects), instead of at POST /v1/sagas/render — see
	// projectDraftAdapter below for why this needs both repositories.
	draftPort := projectDraftAdapter{projects: projectRepo, authoring: promptTemplateRepo}
	createProjectDraft := application.NewCreateProjectDraftUseCase(draftPort)
	updateProjectTopic := application.NewUpdateProjectTopicUseCase(draftPort)
	// Wizard steps 1-2: "Tiếp tục" stores the step's data and how far the
	// Creator got, so a reload or another browser resumes in place.
	wizardPort := wizardAdapter{projects: projectRepo, authoring: promptTemplateRepo}
	saveWizardSettings := application.NewSaveWizardSettingsUseCase(wizardPort)
	// CR-027 FR77.1 — ONE renderer, shared by the Copy button (FR77.2) and the
	// generate endpoint (FR78.1). Two instances would be two chances for the
	// manual path and the API path to send different text for the same role.
	renderPrompt := application.NewRenderPromptUseCase(
		promptTemplateRepo, promptRenderContext{projects: projectRepo, authoring: promptTemplateRepo}, projectRepo, projectRepo)

	// CR-027 FR78/FR79 — the API option, beside the copy-out one. Without a
	// Hive key llmProvider is the Ollama fallback, whose models are not up to
	// writing a Manim scene, so the option is simply not offered and the GUI
	// says so (FR79.4/FR83.2) rather than serving a button that fails.
	var generateAuthoring *application.GenerateAuthoringUseCase
	if cfg.LLMProvider == "hive" && cfg.HiveAPIKey != "" {
		generateAuthoring = application.NewGenerateAuthoringUseCase(
			renderPrompt, llmProvider, llmUsageRecorder,
			promptRenderContext{projects: projectRepo, authoring: promptTemplateRepo},
			promptTemplateRepo,
			saveAuthoringStory, saveAuthoringStoryboard, saveAuthoringCode,
			cfg.HiveMaxInputChars, cfg.HiveMaxOutputTokens,
		).WithClearer(promptTemplateRepo)
	}

	router := httpadapter.NewRouter(startRenderSaga, startPublishSaga, retryStep, projectRepo, suggestPublishMetadata, reviewOutline, channelAssets).
		WithQCReports(qcReportRepo).
		WithShortScriptSuggester(suggestShortScript).
		WithPrompts(prompts).
		WithRenderPrompt(renderPrompt).
		WithAuthoringStory(saveAuthoringStory).
		WithAuthoringStoryboard(saveAuthoringStoryboard).
		WithAuthoringCode(saveAuthoringCode).
		WithAuthoringState(getAuthoringState).
		WithAuthoringMode(saveAuthoringMode).
		WithAuthoringModels(saveAuthoringModels).
		WithProjectDrafts(createProjectDraft, updateProjectTopic).
		WithDefaultModel(cfg.HiveModel).
		WithWizardPosition(projectRepo).
		WithWizard(saveWizardSettings)
	if generateAuthoring != nil {
		router = router.WithGenerateAuthoring(generateAuthoring)
	}

	// 10. Start the HTTP server; the AMQP consumer loop is already running
	// (started in step 7 via goroutines spawned inside consumer.Start).
	server := &http.Server{
		Addr:    ":" + cfg.HTTPPort,
		Handler: router.Handler(),
	}

	go func() {
		logger.Info("orchestrator http server starting", "port", cfg.HTTPPort)
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

// promptRenderContext joins the two repositories CR-027's prompt renderer
// reads from: the project itself lives in ProjectRepository, while the
// authoring artefacts live in PromptTemplateRepository.
//
// A four-line struct here rather than a new method on either repository —
// the split is an accident of which table each row sits in, and neither
// repository should grow a dependency on the other to paper over it.
// Embedding both would be shorter but the two repositories each have a Get,
// so the selector is ambiguous — and forwarding explicitly says which store
// each field of a prompt comes from.
// projectDraftAdapter joins the two repositories CR-028's early-draft use
// cases read/write: the projects row itself lives in ProjectRepository
// (Save, the same upsert StartRenderSaga already calls), while the topic
// and its collision search live in PromptTemplateRepository alongside the
// rest of project_authoring — same split as promptRenderContext above, for
// the same reason.
type projectDraftAdapter struct {
	projects  *postgres.ProjectRepository
	authoring *postgres.PromptTemplateRepository
}

func (a projectDraftAdapter) Save(ctx context.Context, project *domain.Project) error {
	return a.projects.Save(ctx, project)
}

func (a projectDraftAdapter) GetStatus(ctx context.Context, projectID string) (domain.ProjectStatus, error) {
	return a.authoring.GetStatus(ctx, projectID)
}

func (a projectDraftAdapter) GetStatusAndLanguage(ctx context.Context, projectID string) (domain.ProjectStatus, domain.ContentLanguage, error) {
	return a.authoring.GetStatusAndLanguage(ctx, projectID)
}

func (a projectDraftAdapter) SaveAuthoringTopic(ctx context.Context, projectID, topic string) error {
	return a.authoring.SaveAuthoringTopic(ctx, projectID, topic)
}

func (a projectDraftAdapter) FindSimilarTopics(ctx context.Context, language domain.ContentLanguage, normalizedTopic, excludeProjectID string) ([]application.SimilarProject, error) {
	return a.authoring.FindSimilarTopics(ctx, language, normalizedTopic, excludeProjectID)
}

func (a projectDraftAdapter) SaveRenderEngine(ctx context.Context, projectID string, engine domain.RenderEngine) error {
	return a.projects.SaveRenderEngine(ctx, projectID, engine)
}

// wizardAdapter joins the projects row (settings, wizard_step) with the
// status lookup that lives beside project_authoring, same split as
// projectDraftAdapter above.
type wizardAdapter struct {
	projects  *postgres.ProjectRepository
	authoring *postgres.PromptTemplateRepository
}

func (a wizardAdapter) GetStatus(ctx context.Context, projectID string) (domain.ProjectStatus, error) {
	return a.authoring.GetStatus(ctx, projectID)
}

func (a wizardAdapter) SaveWizardSettings(ctx context.Context, projectID string, s domain.WizardSettings) error {
	return a.projects.SaveWizardSettings(ctx, projectID, s)
}

type promptRenderContext struct {
	projects  *postgres.ProjectRepository
	authoring *postgres.PromptTemplateRepository
}

func (c promptRenderContext) Get(ctx context.Context, projectID string) (*domain.Project, error) {
	return c.projects.Get(ctx, projectID)
}

func (c promptRenderContext) GetAuthoringTopic(ctx context.Context, projectID string) (string, error) {
	return c.authoring.GetAuthoringTopic(ctx, projectID)
}

func (c promptRenderContext) GetAuthoringStory(ctx context.Context, projectID string) (string, error) {
	return c.authoring.GetAuthoringStory(ctx, projectID)
}

func (c promptRenderContext) GetAuthoringStoryboard(ctx context.Context, projectID string) (string, error) {
	return c.authoring.GetAuthoringStoryboard(ctx, projectID)
}

func (c promptRenderContext) GetAuthoringCode(ctx context.Context, projectID string) (string, error) {
	return c.authoring.GetAuthoringCode(ctx, projectID)
}
