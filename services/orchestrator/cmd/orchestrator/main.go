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
	"orchestrator/internal/adapters/authoring"
	httpadapter "orchestrator/internal/adapters/http"
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
	// CR-040 FR111: prompts, the 1a/1b/1c chain, LLM calls and usage live in
	// authoring-service. The orchestrator reaches it only for the few things it
	// still needs: the topic search, list summaries, a fork's copy and cleanup.
	authoringClient := authoring.NewClient(cfg.AuthoringServiceURL, cfg.AuthoringServiceTimeout)

	channelAssetPointers := postgres.NewChannelAssetPointerRepository(pool)
	handleStepEvent := application.NewHandleStepEventUseCase(projectRepo, outboxRepo, realPublisher, channelAssetPointers, logger).
		WithQCReports(qcReportRepo).
		WithErrorLog(projectRepo).
		WithAuthoringCleanup(authoringClient)
	retryStep := application.NewRetryStepUseCase(projectRepo, outboxRepo)
	// Cancel goes straight to the broker (a fanout to every worker), not through
	// the outbox: it must reach a worker that is busy right now, and there is no
	// state to keep consistent with it if the publish fails (the use case then
	// leaves the project untouched).
	cancelStep := application.NewCancelStepUseCase(projectRepo, realPublisher)
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
	// CR-028 FR83: the project row is created here, at wizard step 1
	// (POST /v1/projects); the topic and its collision search live in
	// authoring-service (see projectDraftAdapter).
	draftPort := projectDraftAdapter{projects: projectRepo, authoring: authoringClient}
	createProjectDraft := application.NewCreateProjectDraftUseCase(draftPort)
	updateProjectTopic := application.NewUpdateProjectTopicUseCase(draftPort)
	// Wizard steps 1-2: "Tiếp tục" stores the step's data and how far the
	// Creator got, so a reload or another browser resumes in place.
	saveWizardSettings := application.NewSaveWizardSettingsUseCase(wizardAdapter{projects: projectRepo})

	deleteProject := application.NewDeleteProjectUseCase(projectRepo, projectRepo, outboxRepo)
	// The list needs each project's topic and what a draft holds, both of which
	// authoring-service owns; the store below adds them to the repository's rows.
	projectStore := projectStoreWithAuthoring{ProjectRepository: projectRepo, authoring: authoringClient}
	router := httpadapter.NewRouter(startRenderSaga, startPublishSaga, retryStep, projectStore, reviewOutline, channelAssets).
		WithDeleteProject(deleteProject).
		WithQCReports(qcReportRepo).
		WithProjectErrors(projectRepo).
		WithProjectEvents(projectRepo).
		WithAuthoredContent(authoringClient).
		WithProjectDrafts(createProjectDraft, updateProjectTopic).
		WithWizard(saveWizardSettings)
	router = router.WithCancelStep(cancelStep)
	router = router.WithForkProject(application.NewForkProjectUseCase(projectRepo, authoringClient))

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

// projectStoreWithAuthoring is the repository plus the two things the project
// list takes from authoring-service: the topic to name a project by, and which
// of the 1a/1b/1c outputs a draft holds (to place it among steps 3-5).
type projectStoreWithAuthoring struct {
	*postgres.ProjectRepository
	authoring application.AuthoringLookupPort
}

func (s projectStoreWithAuthoring) List(ctx context.Context) ([]domain.ProjectSummary, error) {
	summaries, err := s.ProjectRepository.List(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(summaries))
	for _, p := range summaries {
		ids = append(ids, p.ProjectID)
	}
	// Best effort: a list without topics beats no list when authoring-service is down.
	authored, err := s.authoring.Summaries(ctx, ids)
	if err != nil {
		slog.Warn("could not read authoring summaries for the project list", "error", err)
		return summaries, nil
	}
	for i := range summaries {
		a, ok := authored[summaries[i].ProjectID]
		if !ok {
			continue
		}
		summaries[i].Topic = a.Topic
		content := domain.AuthoredContent{Story: a.Story, Storyboard: a.Storyboard, Code: a.Code}
		summaries[i].FlowStep = domain.FlowStateFor(summaries[i].Status, summaries[i].WizardStep, content).Step
	}
	return summaries, nil
}

// projectDraftAdapter joins what CR-028's early-draft use cases read/write: the
// projects row (this service) and the topic with its collision search
// (authoring-service).
type projectDraftAdapter struct {
	projects  *postgres.ProjectRepository
	authoring application.AuthoringLookupPort
}

func (a projectDraftAdapter) Save(ctx context.Context, project *domain.Project) error {
	return a.projects.Save(ctx, project)
}

func (a projectDraftAdapter) GetStatus(ctx context.Context, projectID string) (domain.ProjectStatus, error) {
	return a.projects.GetStatus(ctx, projectID)
}

func (a projectDraftAdapter) GetStatusAndLanguage(ctx context.Context, projectID string) (domain.ProjectStatus, domain.ContentLanguage, error) {
	return a.projects.GetStatusAndLanguage(ctx, projectID)
}

func (a projectDraftAdapter) SaveAuthoringTopic(ctx context.Context, projectID, topic string, language domain.ContentLanguage) error {
	return a.authoring.SaveAuthoringTopic(ctx, projectID, topic, language)
}

// FindSimilarTopics asks authoring-service for the matches, then fills in each
// one's current status from the projects it owns; a match whose project no
// longer exists here is dropped.
func (a projectDraftAdapter) FindSimilarTopics(ctx context.Context, language domain.ContentLanguage, normalizedTopic, excludeProjectID string) ([]application.SimilarProject, error) {
	found, err := a.authoring.FindSimilarTopics(ctx, language, normalizedTopic, excludeProjectID)
	if err != nil || len(found) == 0 {
		return found, err
	}
	ids := make([]string, 0, len(found))
	for _, f := range found {
		ids = append(ids, f.ProjectID)
	}
	statuses, err := a.projects.GetStatuses(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := found[:0]
	for _, f := range found {
		if status, ok := statuses[f.ProjectID]; ok {
			f.Status = status
			out = append(out, f)
		}
	}
	return out, nil
}

func (a projectDraftAdapter) SaveRenderEngine(ctx context.Context, projectID string, engine domain.RenderEngine) error {
	return a.projects.SaveRenderEngine(ctx, projectID, engine)
}

// wizardAdapter is the projects row's settings, plus its status lookup.
type wizardAdapter struct {
	projects *postgres.ProjectRepository
}

func (a wizardAdapter) GetStatus(ctx context.Context, projectID string) (domain.ProjectStatus, error) {
	return a.projects.GetStatus(ctx, projectID)
}

func (a wizardAdapter) SaveWizardSettings(ctx context.Context, projectID string, s domain.WizardSettings) error {
	return a.projects.SaveWizardSettings(ctx, projectID, s)
}
