package application

import (
	"context"
	"fmt"

	"orchestrator/internal/domain"
)

// PromptOverridePort is the persistence capability the CR-027 FR84
// two-layer prompt store needs.
type PromptOverridePort interface {
	GetEffective(ctx context.Context, role domain.PromptRole, language string) (domain.EffectivePromptTemplate, error)
	GetOverride(ctx context.Context, role domain.PromptRole, language string) (domain.PromptOverride, error)
	ListOverrides(ctx context.Context) ([]domain.PromptOverride, error)
	SaveOverride(ctx context.Context, role domain.PromptRole, language, templateText string) (domain.PromptOverride, error)
	SetOverrideActive(ctx context.Context, role domain.PromptRole, language string, active bool) (domain.PromptOverride, error)
	DeleteOverride(ctx context.Context, role domain.PromptRole, language string) error
}

// PromptOverridesUseCase backs the Creator-owned layer of the prompt store
// (CR-027 FR84).
//
// The shipped layer has no use case of its own any more: nothing edits it.
// It is written by seeding at startup and read through GetEffective.
type PromptOverridesUseCase struct {
	overrides PromptOverridePort
}

func NewPromptOverridesUseCase(overrides PromptOverridePort) *PromptOverridesUseCase {
	return &PromptOverridesUseCase{overrides: overrides}
}

func validRoleLanguage(role domain.PromptRole, language string) error {
	if !domain.ValidPromptRole(string(role)) {
		return fmt.Errorf("invalid role %q", role)
	}
	if language != "vi" && language != "en" {
		return fmt.Errorf("language must be 'vi' or 'en'")
	}
	return nil
}

// Effective returns the wording the pipeline will actually use.
func (uc *PromptOverridesUseCase) Effective(ctx context.Context, role domain.PromptRole, language string) (domain.EffectivePromptTemplate, error) {
	if err := validRoleLanguage(role, language); err != nil {
		return domain.EffectivePromptTemplate{}, err
	}
	return uc.overrides.GetEffective(ctx, role, language)
}

// List returns every saved override for the admin screen.
func (uc *PromptOverridesUseCase) List(ctx context.Context) ([]domain.PromptOverride, error) {
	return uc.overrides.ListOverrides(ctx)
}

// Save stores the Creator's wording for one role/language.
func (uc *PromptOverridesUseCase) Save(ctx context.Context, role domain.PromptRole, language, templateText string) (domain.PromptOverride, error) {
	if err := validRoleLanguage(role, language); err != nil {
		return domain.PromptOverride{}, err
	}
	if templateText == "" {
		// Saving an empty override would be a confusing way to spell "go
		// back to the shipped wording" — that is what switching off, or
		// deleting, is for.
		return domain.PromptOverride{}, fmt.Errorf("template_text is required")
	}
	return uc.overrides.SaveOverride(ctx, role, language, templateText)
}

// SetActive switches an override on or off without destroying it (FR84.5).
func (uc *PromptOverridesUseCase) SetActive(ctx context.Context, role domain.PromptRole, language string, active bool) (domain.PromptOverride, error) {
	if err := validRoleLanguage(role, language); err != nil {
		return domain.PromptOverride{}, err
	}
	return uc.overrides.SetOverrideActive(ctx, role, language, active)
}

// Delete removes the Creator's wording entirely.
func (uc *PromptOverridesUseCase) Delete(ctx context.Context, role domain.PromptRole, language string) error {
	if err := validRoleLanguage(role, language); err != nil {
		return err
	}
	return uc.overrides.DeleteOverride(ctx, role, language)
}
