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
	startPublishSaga := application.NewStartPublishSagaUseCase(projectRepo, outboxRepo)
	handleStepEvent := application.NewHandleStepEventUseCase(projectRepo, outboxRepo, realPublisher, logger)
	retryStep := application.NewRetryStepUseCase(projectRepo, outboxRepo)

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
	router := httpadapter.NewRouter(startRenderSaga, startPublishSaga, retryStep, projectRepo)

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
