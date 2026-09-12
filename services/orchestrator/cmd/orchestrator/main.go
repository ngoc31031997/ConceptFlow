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
	// CR-025: seed the 4-role authoring-pipeline prompt templates, same
	// insert-if-absent posture as SeedVideoFormats above — an editor's saved
	// wording must survive a restart.
	promptTemplateRepo := postgres.NewPromptTemplateRepository(pool)
	if err := promptTemplateRepo.SeedPromptTemplates(ctx); err != nil {
		logger.Warn("could not seed prompt templates", "error", err)
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
	promptTemplates := application.NewPromptTemplatesUseCase(promptTemplateRepo)
	saveAuthoringStory := application.NewSaveAuthoringStoryUseCase(promptTemplateRepo)
	// CR-025 step 2: Visual Director's storyboard save, and the shared
	// read-side use case both steps' rehydration relies on.
	saveAuthoringStoryboard := application.NewSaveAuthoringStoryboardUseCase(promptTemplateRepo)
	// CR-025 step 3/4: Manim Engineer's code save and Script Reviewer's
	// verdict save, sharing the same read-side use case.
	saveAuthoringCode := application.NewSaveAuthoringCodeUseCase(promptTemplateRepo)
	saveAuthoringReview := application.NewSaveAuthoringReviewUseCase(promptTemplateRepo)
	getAuthoringState := application.NewGetAuthoringStateUseCase(promptTemplateRepo)
	router := httpadapter.NewRouter(startRenderSaga, startPublishSaga, retryStep, projectRepo, suggestPublishMetadata, reviewOutline, channelAssets).
		WithQCReports(qcReportRepo).
		WithShortScriptSuggester(suggestShortScript).
		WithPromptTemplates(promptTemplates).
		WithAuthoringStory(saveAuthoringStory).
		WithAuthoringStoryboard(saveAuthoringStoryboard).
		WithAuthoringCode(saveAuthoringCode).
		WithAuthoringReview(saveAuthoringReview).
		WithAuthoringState(getAuthoringState)

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
