package application

import (
	"context"
	"log/slog"

	"authoring/internal/domain"
)

// StepDefaultModelsPort reads the model each step runs on when nothing more
// specific was chosen. llm-service owns the setting and the list of steps; every
// configurable step has an entry.
type StepDefaultModelsPort interface {
	GetStepDefaultModels(ctx context.Context) (map[string]string, error)
}

// ModelCatalogPort reads the models a step can run on from llm-service, the
// service that routes a model id to a provider.
type ModelCatalogPort interface {
	ModelCatalog(ctx context.Context) ([]domain.AuthoringModelOption, error)
}

// stepDefaultModel is the default model of step, "" when the setting cannot be
// read. An unreadable setting is logged, and the call then runs on the model
// llm-service itself defaults to.
func stepDefaultModel(ctx context.Context, src StepDefaultModelsPort, step string) string {
	if src == nil {
		return ""
	}
	defaults, err := src.GetStepDefaultModels(ctx)
	if err != nil {
		slog.Warn("could not read the step default models", "step", step, "error", err.Error())
		return ""
	}
	return defaults[step]
}
