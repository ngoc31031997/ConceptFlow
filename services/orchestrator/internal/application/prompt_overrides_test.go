package application_test

import (
	"context"
	"testing"

	"orchestrator/internal/application"
	"orchestrator/internal/domain"
)

// fakeOverrideStore models the two layers the way the SQL does: a shipped
// map that nothing here edits, and an override map with an on/off flag.
type fakeOverrideStore struct {
	seed      map[string]string
	overrides map[string]domain.PromptOverride
	deleted   []string
}

func newFakeOverrideStore() *fakeOverrideStore {
	return &fakeOverrideStore{
		seed: map[string]string{
			"story_architect|vi": "SHIPPED story vi",
			"manim_engineer|vi":  "SHIPPED manim vi",
		},
		overrides: map[string]domain.PromptOverride{},
	}
}

func key(role domain.PromptRole, language string) string {
	return string(role) + "|" + language
}

func (f *fakeOverrideStore) GetEffective(_ context.Context, role domain.PromptRole, language string) (domain.EffectivePromptTemplate, error) {
	k := key(role, language)
	out := domain.EffectivePromptTemplate{Role: role, Language: language, TemplateText: f.seed[k], SeedVersion: 2}
	if o, ok := f.overrides[k]; ok && o.IsActive {
		out.TemplateText = o.TemplateText
		out.FromOverride = true
	}
	return out, nil
}

func (f *fakeOverrideStore) GetOverride(_ context.Context, role domain.PromptRole, language string) (domain.PromptOverride, error) {
	return f.overrides[key(role, language)], nil
}

func (f *fakeOverrideStore) ListOverrides(_ context.Context) ([]domain.PromptOverride, error) {
	out := make([]domain.PromptOverride, 0, len(f.overrides))
	for _, o := range f.overrides {
		out = append(out, o)
	}
	return out, nil
}

func (f *fakeOverrideStore) SaveOverride(_ context.Context, role domain.PromptRole, language, text string) (domain.PromptOverride, error) {
	k := key(role, language)
	existing, had := f.overrides[k]
	o := domain.PromptOverride{Role: role, Language: language, TemplateText: text, IsActive: true}
	if had {
		// Mirrors the SQL: editing the text leaves is_active alone.
		o.IsActive = existing.IsActive
	}
	f.overrides[k] = o
	return o, nil
}

func (f *fakeOverrideStore) SetOverrideActive(_ context.Context, role domain.PromptRole, language string, active bool) (domain.PromptOverride, error) {
	k := key(role, language)
	o := f.overrides[k]
	o.Role, o.Language, o.IsActive = role, language, active
	f.overrides[k] = o
	return o, nil
}

func (f *fakeOverrideStore) DeleteOverride(_ context.Context, role domain.PromptRole, language string) error {
	k := key(role, language)
	delete(f.overrides, k)
	f.deleted = append(f.deleted, k)
	return nil
}

// TestEffective_ShippedWordingIsTheDefault — FR84.4. No override is the
// normal, clean state, not a missing configuration.
func TestEffective_ShippedWordingIsTheDefault(t *testing.T) {
	uc := application.NewPromptOverridesUseCase(newFakeOverrideStore())

	got, err := uc.Effective(context.Background(), domain.RoleStoryArchitect, "vi")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TemplateText != "SHIPPED story vi" || got.FromOverride {
		t.Fatalf("expected the shipped wording, got %+v", got)
	}
}

func TestEffective_AnActiveOverrideWins(t *testing.T) {
	store := newFakeOverrideStore()
	uc := application.NewPromptOverridesUseCase(store)

	if _, err := uc.Save(context.Background(), domain.RoleStoryArchitect, "vi", "MY OWN story"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := uc.Effective(context.Background(), domain.RoleStoryArchitect, "vi")
	if got.TemplateText != "MY OWN story" || !got.FromOverride {
		t.Fatalf("expected the override to win, got %+v", got)
	}
}

// TestSetActive_OffFallsBackWithoutDestroying is CR-027 FR84.5, the
// non-destructive replacement for the old reset endpoint. Reset overwrote
// the Creator's text for good; switching off must keep it.
func TestSetActive_OffFallsBackWithoutDestroying(t *testing.T) {
	store := newFakeOverrideStore()
	uc := application.NewPromptOverridesUseCase(store)
	ctx := context.Background()

	_, _ = uc.Save(ctx, domain.RoleStoryArchitect, "vi", "MY OWN story")
	if _, err := uc.SetActive(ctx, domain.RoleStoryArchitect, "vi", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := uc.Effective(ctx, domain.RoleStoryArchitect, "vi")
	if got.TemplateText != "SHIPPED story vi" || got.FromOverride {
		t.Fatalf("switching off must fall back to the shipped wording, got %+v", got)
	}

	// The whole point: it is still there and comes back unchanged.
	if _, err := uc.SetActive(ctx, domain.RoleStoryArchitect, "vi", true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, _ = uc.Effective(ctx, domain.RoleStoryArchitect, "vi")
	if got.TemplateText != "MY OWN story" {
		t.Fatalf("the Creator's wording must survive being switched off, got %q", got.TemplateText)
	}
}

// TestSave_EditingAnInactiveOverrideDoesNotSwitchItOn — editing text and
// putting it into service are two separate decisions.
func TestSave_EditingAnInactiveOverrideDoesNotSwitchItOn(t *testing.T) {
	store := newFakeOverrideStore()
	uc := application.NewPromptOverridesUseCase(store)
	ctx := context.Background()

	_, _ = uc.Save(ctx, domain.RoleStoryArchitect, "vi", "draft one")
	_, _ = uc.SetActive(ctx, domain.RoleStoryArchitect, "vi", false)
	_, _ = uc.Save(ctx, domain.RoleStoryArchitect, "vi", "draft two")

	got, _ := uc.Effective(ctx, domain.RoleStoryArchitect, "vi")
	if got.FromOverride {
		t.Fatalf("an edit must not silently put an off override back in service, got %+v", got)
	}
}

func TestSave_RejectsAnEmptyOverride(t *testing.T) {
	uc := application.NewPromptOverridesUseCase(newFakeOverrideStore())

	if _, err := uc.Save(context.Background(), domain.RoleStoryArchitect, "vi", ""); err == nil {
		t.Fatal("an empty override is a confusing way to spell 'use the shipped wording'")
	}
}

func TestOverrides_RejectUnknownRoleAndLanguage(t *testing.T) {
	uc := application.NewPromptOverridesUseCase(newFakeOverrideStore())
	ctx := context.Background()

	if _, err := uc.Save(ctx, domain.PromptRole("not_a_role"), "vi", "x"); err == nil {
		t.Fatal("expected an error for an unknown role")
	}
	if _, err := uc.Save(ctx, domain.RoleStoryArchitect, "fr", "x"); err == nil {
		t.Fatal("expected an error for an unsupported language")
	}
	if _, err := uc.Effective(ctx, domain.PromptRole("not_a_role"), "vi"); err == nil {
		t.Fatal("expected an error for an unknown role")
	}
}

func TestDelete_LeavesTheShippedWordingInCharge(t *testing.T) {
	store := newFakeOverrideStore()
	uc := application.NewPromptOverridesUseCase(store)
	ctx := context.Background()

	_, _ = uc.Save(ctx, domain.RoleStoryArchitect, "vi", "MY OWN story")
	if err := uc.Delete(ctx, domain.RoleStoryArchitect, "vi"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := uc.Effective(ctx, domain.RoleStoryArchitect, "vi")
	if got.TemplateText != "SHIPPED story vi" || got.FromOverride {
		t.Fatalf("expected the shipped wording after delete, got %+v", got)
	}
}
