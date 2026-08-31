// Package logging provides log/slog helpers that inject correlation fields
// (saga_id/project_id/step for AMQP processing, X-Request-ID for REST) into
// structured log records, so a single Saga's log lines can be filtered
// across both transports.
package logging

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"net/http"
	"os"
)

// requestIDHeader is the correlation header REST callers (API Gateway) set;
// Orchestrator does not generate its own REST correlation id — the Gateway
// owns that (interface-contracts.md "Correlation ID").
const requestIDHeader = "X-Request-ID"

// NewLogger returns the base structured logger for the service (JSON
// handler on stdout, suitable for container log collection).
func NewLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, nil))
}

// WithSagaFields returns a logger with saga_id/project_id/step attached as
// structured fields, for use around AMQP event processing.
func WithSagaFields(logger *slog.Logger, sagaID, projectID, step string) *slog.Logger {
	return logger.With("saga_id", sagaID, "project_id", projectID, "step", step)
}

// WithRequestID returns a logger with the X-Request-ID header (if present)
// attached as a structured field, for use around REST handlers.
func WithRequestID(logger *slog.Logger, r *http.Request) *slog.Logger {
	requestID := r.Header.Get(requestIDHeader)
	if requestID == "" {
		return logger
	}
	return logger.With("request_id", requestID)
}

// contextKey is an unexported type to avoid context key collisions across
// packages (standard Go idiom).
type contextKey string

const loggerContextKey contextKey = "logger"

// IntoContext stores logger in ctx for retrieval by FromContext further
// down a call chain (e.g. from a use case that doesn't otherwise receive
// the logger).
func IntoContext(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerContextKey, logger)
}

// FromContext retrieves a logger previously stored by IntoContext, falling
// back to slog.Default() if none was stored.
func FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerContextKey).(*slog.Logger); ok {
		return logger
	}
	return slog.Default()
}

// NewCorrelationID generates a fresh correlation identifier — used as a
// fallback when a REST caller does not send X-Request-ID. Implemented
// locally (UUIDv4) to avoid adding a dependency beyond chi/pgx/amqp091-go.
func NewCorrelationID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("NewCorrelationID: failed to read random bytes: %v", err))
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
