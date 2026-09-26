package application_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"authoring/internal/application"
	"authoring/internal/domain"
)

// fakePromptLibrary mirrors the repository's contract: one active row per
// role, a system row per role.
type fakePromptLibrary struct {
	rows []domain.Prompt
	next int
}

func newFakePromptLibrary() *fakePromptLibrary {
	f := &fakePromptLibrary{}
	for _, sys := range domain.SystemPrompts() {
		sys.IsActive = true
		f.rows = append(f.rows, sys)
	}
	return f
}

func (f *fakePromptLibrary) find(id string) (int, bool) {
	for i, r := range f.rows {
		if r.ID == id {
			return i, true
		}
	}
	return 0, false
}

func (f *fakePromptLibrary) GetActive(_ context.Context, role domain.PromptRole) (domain.Prompt, error) {
	for _, r := range f.rows {
		if r.Role == role && r.IsActive {
			return r, nil
		}
	}
	return domain.Prompt{}, application.ErrPromptNotFound
}
func (f *fakePromptLibrary) GetSystem(_ context.Context, role domain.PromptRole) (domain.Prompt, error) {
	return f.Get(context.Background(), domain.SystemPromptID(role))
}
func (f *fakePromptLibrary) List(_ context.Context, role domain.PromptRole) ([]domain.Prompt, error) {
	var out []domain.Prompt
	for _, r := range f.rows {
		if role == "" || r.Role == role {
			out = append(out, r)
		}
	}
	return out, nil
}
func (f *fakePromptLibrary) Get(_ context.Context, id string) (domain.Prompt, error) {
	i, ok := f.find(id)
	if !ok {
		return domain.Prompt{}, application.ErrPromptNotFound
	}
	return f.rows[i], nil
}
func (f *fakePromptLibrary) Create(_ context.Context, role domain.PromptRole, name, text string) (domain.Prompt, error) {
	f.next++
	p := domain.Prompt{ID: fmt.Sprintf("u%d", f.next), Role: role, Name: name, TemplateText: text}
	f.rows = append(f.rows, p)
	return p, nil
}
func (f *fakePromptLibrary) Update(_ context.Context, id, name, text string) (domain.Prompt, error) {
	i, _ := f.find(id)
	f.rows[i].Name, f.rows[i].TemplateText = name, text
	return f.rows[i], nil
}
func (f *fakePromptLibrary) Delete(_ context.Context, id string) error {
	i, _ := f.find(id)
	f.rows = append(f.rows[:i], f.rows[i+1:]...)
	return nil
}
func (f *fakePromptLibrary) Activate(_ context.Context, id string) (domain.Prompt, error) {
	i, ok := f.find(id)
	if !ok {
		return domain.Prompt{}, application.ErrPromptNotFound
	}
	for j := range f.rows {
		if f.rows[j].Role == f.rows[i].Role {
			f.rows[j].IsActive = false
		}
	}
	f.rows[i].IsActive = true
	return f.rows[i], nil
}

func TestSystemPrompts_OnePerRoleAndVietnameseOnly(t *testing.T) {
	seen := map[domain.PromptRole]int{}
	for _, p := range domain.SystemPrompts() {
		seen[p.Role]++
		if !p.IsSystem || p.TemplateText == "" {
			t.Errorf("%s: not a usable system row: %+v", p.Role, p)
		}
	}
	for _, role := range []domain.PromptRole{domain.RoleStoryArchitect, domain.RoleVisualDirector, domain.RoleManimEngineer, domain.RoleRemotionEngineer} {
		if seen[role] != 1 {
			t.Errorf("role %s has %d system rows, want 1", role, seen[role])
		}
	}
}

func TestSystemPromptCannotBeEditedOrDeleted(t *testing.T) {
	uc := application.NewPromptsUseCase(newFakePromptLibrary())
	id := domain.SystemPromptID(domain.RoleStoryArchitect)

	if _, err := uc.Update(context.Background(), id, "x", "y"); !errors.Is(err, application.ErrPromptReadOnly) {
		t.Errorf("Update err = %v, want ErrPromptReadOnly", err)
	}
	if err := uc.Delete(context.Background(), id); !errors.Is(err, application.ErrPromptReadOnly) {
		t.Errorf("Delete err = %v, want ErrPromptReadOnly", err)
	}
}

func TestCopyOfSystemPromptIsEditableInactiveAndUserOwned(t *testing.T) {
	store := newFakePromptLibrary()
	uc := application.NewPromptsUseCase(store)

	copied, err := uc.Copy(context.Background(), domain.SystemPromptID(domain.RoleVisualDirector))
	if err != nil {
		t.Fatalf("Copy: %v", err)
	}
	if copied.IsSystem || copied.IsActive {
		t.Errorf("copy must be user-owned and inactive, got %+v", copied)
	}
	if copied.Role != domain.RoleVisualDirector || copied.TemplateText == "" {
		t.Errorf("copy lost role/text: %+v", copied)
	}
	if _, err := uc.Update(context.Background(), copied.ID, "tên mới", "chữ mới"); err != nil {
		t.Errorf("a copy must be editable: %v", err)
	}
	active, _ := uc.Active(context.Background(), domain.RoleVisualDirector)
	if !active.IsSystem {
		t.Error("copying must not change which prompt is active")
	}
}

func TestActivateLeavesExactlyOneActivePerRole(t *testing.T) {
	uc := application.NewPromptsUseCase(newFakePromptLibrary())
	ctx := context.Background()

	mine, _ := uc.Create(ctx, domain.RoleStoryArchitect, "của tôi", "nội dung")
	if _, err := uc.Activate(ctx, mine.ID); err != nil {
		t.Fatalf("Activate: %v", err)
	}
	rows, _ := uc.List(ctx, domain.RoleStoryArchitect)
	active := 0
	for _, r := range rows {
		if r.IsActive {
			active++
		}
	}
	if active != 1 {
		t.Fatalf("%d active rows for the role, want exactly 1", active)
	}
	got, _ := uc.Active(ctx, domain.RoleStoryArchitect)
	if got.ID != mine.ID {
		t.Errorf("active = %s, want %s", got.ID, mine.ID)
	}
}

func TestDeletingTheActivePromptHandsTheRoleBackToTheSystemOne(t *testing.T) {
	uc := application.NewPromptsUseCase(newFakePromptLibrary())
	ctx := context.Background()

	mine, _ := uc.Create(ctx, domain.RoleManimEngineer, "của tôi", "nội dung")
	_, _ = uc.Activate(ctx, mine.ID)
	if err := uc.Delete(ctx, mine.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	got, err := uc.Active(ctx, domain.RoleManimEngineer)
	if err != nil || !got.IsSystem {
		t.Errorf("active after delete = %+v, %v; want the system prompt", got, err)
	}
}

func TestCreateValidatesRoleNameAndText(t *testing.T) {
	uc := application.NewPromptsUseCase(newFakePromptLibrary())
	ctx := context.Background()
	if _, err := uc.Create(ctx, "nonsense", "a", "b"); err == nil {
		t.Error("expected an error for an unknown role")
	}
	if _, err := uc.Create(ctx, domain.RoleStoryArchitect, "  ", "b"); err == nil {
		t.Error("expected an error for a blank name")
	}
	if _, err := uc.Create(ctx, domain.RoleStoryArchitect, "a", " "); err == nil {
		t.Error("expected an error for blank text")
	}
}
