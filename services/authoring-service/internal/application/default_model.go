package application

import (
	"context"
	"fmt"
	"log/slog"

	"authoring/internal/domain"
)

// DefaultModelPort stores the model each step runs on when nothing more
// specific was chosen.
type DefaultModelPort interface {
	StepDefaultModelsPort
	SaveStepDefaultModel(ctx context.Context, step, model string) error
}

// StepDefaultModelsPort reads the model each step runs on when nothing more
// specific was chosen. Every configurable step has an entry: the model saved
// for it, else the first model of the catalog.
type StepDefaultModelsPort interface {
	GetStepDefaultModels(ctx context.Context) (map[string]string, error)
}

// DefaultModelUseCase reads and changes those defaults.
type DefaultModelUseCase struct {
	store DefaultModelPort
}

// NewDefaultModelUseCase builds the use case over its store.
func NewDefaultModelUseCase(store DefaultModelPort) *DefaultModelUseCase {
	return &DefaultModelUseCase{store: store}
}

// StepModels returns the default model of every configurable step.
func (uc *DefaultModelUseCase) StepModels(ctx context.Context) (map[string]string, error) {
	return uc.store.GetStepDefaultModels(ctx)
}

// SetStep saves the default of one step. An unknown step, or a model that is
// not in the catalog, is refused rather than replaced, so the Creator is never
// told one model is the default while another runs.
func (uc *DefaultModelUseCase) SetStep(ctx context.Context, step, model string) error {
	if !domain.ValidModelStep(step) {
		return fmt.Errorf("%w: bước %q không cấu hình được model", domain.ErrInvalidWizardInput, step)
	}
	if model == "" || !domain.ValidAuthoringModel(model) {
		return fmt.Errorf("%w: model %q không có trong danh sách", domain.ErrInvalidWizardInput, model)
	}
	return uc.store.SaveStepDefaultModel(ctx, step, model)
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
