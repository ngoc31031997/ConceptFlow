// Command visual-service is visual-service's composition root. It owns the
// visual pre-production steps (8–14) and their database (ADR-0049).
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"visual/internal/adapters/authoring"
	httpadapter "visual/internal/adapters/http"
	"visual/internal/adapters/llm"
	"visual/internal/adapters/postgres"
	"visual/internal/adapters/rendering"
	"visual/internal/application"
	"visual/internal/config"
	"visual/internal/domain"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
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

	deleteProject := application.NewDeleteProjectUseCase(postgres.NewProjectDataRepository(pool))
	router := httpadapter.NewRouter(deleteProject)
	router.WithReferenceLibrary(application.NewReferenceLibraryUseCase(postgres.NewReferenceStyleRepository(pool)))

	llmClient := llm.NewClient(cfg.LLMServiceURL, time.Duration(cfg.LLMTimeoutSeconds)*time.Second)
	conceptRepo := postgres.NewConceptRepository(pool)
	conceptDesk := application.NewConceptDesk(
		authoring.NewClient(cfg.AuthoringServiceURL, 30*time.Second),
		conceptRepo, conceptRepo,
		application.NewExternalAIProvider(llmClient, llmClient),
		llmClient, llmClient, rendering.NewClient(cfg.RenderingURL, 10*time.Second), cfg.MaxOutputTokens,
	)
	router.WithConcept(conceptDesk)
	router.WithPlan(application.NewPlanDesk(authoring.NewClient(cfg.AuthoringServiceURL, 30*time.Second), postgres.NewPlanRepository(pool)).
		WithAI(application.NewExternalAIProvider(llmClient, llmClient), llmClient, llmClient, cfg.MaxOutputTokens))
	go pushSystemPrompts(ctx, logger, llmClient)

	server := &http.Server{Addr: ":" + cfg.HTTPPort, Handler: router.Handler(), ReadHeaderTimeout: 10 * time.Second}
	go func() {
		logger.Info("visual-service http server starting", "port", cfg.HTTPPort)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
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

// pushSystemPrompts sends the shipped prompts of steps 8 and 9 to llm-service,
// retrying with a growing pause until it accepts them or the process stops.
func pushSystemPrompts(ctx context.Context, logger *slog.Logger, client *llm.Client) {
	pause := 2 * time.Second
	for {
		err := client.PushSystemPrompts(ctx, domain.PromptOwner, domain.SystemPrompts())
		if err == nil {
			logger.Info("system prompts pushed to llm-service", "count", len(domain.SystemPrompts()))
			return
		}
		logger.Error("could not push the system prompts to llm-service; retrying", "error", err.Error(), "in", pause.String())
		select {
		case <-ctx.Done():
			return
		case <-time.After(pause):
		}
		if pause < time.Minute {
			pause *= 2
		}
	}
}
