package application

import (
	"context"
	"fmt"

	"authoring/internal/domain"
)

// DefaultModelPort stores the model every AI call runs on when the project and
// the button name none.
type DefaultModelPort interface {
	GetDefaultModel(ctx context.Context) (string, error)
	SaveDefaultModel(ctx context.Context, model string) error
}

// DefaultModelUseCase reads and changes that default.
type DefaultModelUseCase struct {
	store DefaultModelPort
}

// NewDefaultModelUseCase builds the use case over its store.
func NewDefaultModelUseCase(store DefaultModelPort) *DefaultModelUseCase {
	return &DefaultModelUseCase{store: store}
}

// Get returns the saved default, "" when none was set.
func (uc *DefaultModelUseCase) Get(ctx context.Context) (string, error) {
	return uc.store.GetDefaultModel(ctx)
}

// Set saves the default. The default also serves the code step and the
// drawings, so a model that cannot write code is refused; "" clears it. An
// unknown id is refused rather than replaced, so the Creator is never told one
// model is the default while another runs.
func (uc *DefaultModelUseCase) Set(ctx context.Context, model string) error {
	if !domain.ValidAuthoringModel(model) {
		return fmt.Errorf("%w: model %q không có trong danh sách", domain.ErrInvalidWizardInput, model)
	}
	if err := checkCodeModel("code", model); err != nil {
		return err
	}
	return uc.store.SaveDefaultModel(ctx, model)
}
