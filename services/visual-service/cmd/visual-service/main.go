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

	httpadapter "visual/internal/adapters/http"
	"visual/internal/adapters/postgres"
	"visual/internal/application"
	"visual/internal/config"
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
