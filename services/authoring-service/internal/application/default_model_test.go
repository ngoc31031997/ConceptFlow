package application

import (
	"context"
	"errors"
	"testing"

	"authoring/internal/domain"
)

type memoryDefault struct{ model string }

func (m *memoryDefault) GetDefaultModel(context.Context) (string, error) { return m.model, nil }
func (m *memoryDefault) SaveDefaultModel(_ context.Context, model string) error {
	m.model = model
	return nil
}

func TestDefaultModelAcceptsAKnownModelAndClears(t *testing.T) {
	store := &memoryDefault{}
	uc := NewDefaultModelUseCase(store)
	if err := uc.Set(context.Background(), "zai-org/glm-5.3-flash"); err != nil {
		t.Fatal(err)
	}
	if got, _ := uc.Get(context.Background()); got != "zai-org/glm-5.3-flash" {
		t.Errorf("default not stored, got %q", got)
	}
	if err := uc.Set(context.Background(), ""); err != nil || store.model != "" {
		t.Errorf("an empty model must clear the default, got %q, %v", store.model, err)
	}
}

func TestDefaultModelRefusesAnUnknownModelAndOneThatCannotWriteCode(t *testing.T) {
	uc := NewDefaultModelUseCase(&memoryDefault{})
	if err := uc.Set(context.Background(), "gpt-x"); !errors.Is(err, domain.ErrInvalidWizardInput) {
		t.Errorf("an unknown model must be refused, got %v", err)
	}
	if err := uc.Set(context.Background(), "ollama"); err == nil {
		t.Error("the default also serves code and drawings, so a model that cannot write code must be refused")
	}
}
