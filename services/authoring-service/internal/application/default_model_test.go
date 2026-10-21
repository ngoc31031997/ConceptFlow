package application

import (
	"context"
	"errors"
	"testing"

	"authoring/internal/domain"
)

type memoryDefault struct {
	steps map[string]string
	err   error
}

func (m *memoryDefault) GetStepDefaultModels(context.Context) (map[string]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	return domain.WithStepModelFallback(m.steps), nil
}

func (m *memoryDefault) SaveStepDefaultModel(_ context.Context, step, model string) error {
	if m.steps == nil {
		m.steps = map[string]string{}
	}
	m.steps[step] = model
	return nil
}

func TestStepDefaultModelStoresAnyListedModel(t *testing.T) {
	uc := NewDefaultModelUseCase(&memoryDefault{})
	ctx := context.Background()
	for _, c := range []struct{ step, model string }{
		{"brief", "ollama"},
		{"research", "deepseek-ai/deepseek-v4.1-flash"},
		{"code", "ollama"},
		{"illustrations", "zai-org/glm-5.3-flash"},
	} {
		if err := uc.SetStep(ctx, c.step, c.model); err != nil {
			t.Errorf("SetStep(%q, %q) = %v, want it stored", c.step, c.model, err)
		}
	}
	got, err := uc.StepModels(ctx)
	if err != nil || got["research"] != "deepseek-ai/deepseek-v4.1-flash" || got["code"] != "ollama" || got["illustrations"] != "zai-org/glm-5.3-flash" {
		t.Fatalf("StepModels = %v, %v", got, err)
	}
}

func TestStepModelsFollowTheFirstCatalogModelUntilSet(t *testing.T) {
	got, err := NewDefaultModelUseCase(&memoryDefault{}).StepModels(context.Background())
	if err != nil || len(got) != len(domain.ConfigurableModelSteps) {
		t.Fatalf("StepModels = %v, %v; want one entry per configurable step", got, err)
	}
	for step, model := range got {
		if model != "ollama" {
			t.Errorf("step %q = %q, want ollama, the first model of the catalog", step, model)
		}
	}
}

func TestStepDefaultModelRefusesUnknownStepEmptyOrUnknownModel(t *testing.T) {
	uc := NewDefaultModelUseCase(&memoryDefault{})
	ctx := context.Background()
	if err := uc.SetStep(ctx, "render", "ollama"); !errors.Is(err, domain.ErrInvalidWizardInput) {
		t.Errorf("an unknown step must be refused, got %v", err)
	}
	if err := uc.SetStep(ctx, "brief", "gpt-x"); !errors.Is(err, domain.ErrInvalidWizardInput) {
		t.Errorf("an unknown model must be refused, got %v", err)
	}
	if err := uc.SetStep(ctx, "brief", ""); !errors.Is(err, domain.ErrInvalidWizardInput) {
		t.Errorf("an empty model must be refused, got %v", err)
	}
}

func TestStepDefaultModelReadsNothingWhenNoSourceOrUnreadable(t *testing.T) {
	ctx := context.Background()
	if got := stepDefaultModel(ctx, nil, "illustrations"); got != "" {
		t.Errorf("with no source the call runs on llm-service's default, got %q", got)
	}
	src := &memoryDefault{steps: map[string]string{"illustrations": "zai-org/glm-5.3-flash"}}
	if got := stepDefaultModel(ctx, src, "illustrations"); got != "zai-org/glm-5.3-flash" {
		t.Errorf("saved step default = %q, want the saved model", got)
	}
	src.err = errors.New("db down")
	if got := stepDefaultModel(ctx, src, "illustrations"); got != "" {
		t.Errorf("an unreadable setting must read as unset, got %q", got)
	}
}
