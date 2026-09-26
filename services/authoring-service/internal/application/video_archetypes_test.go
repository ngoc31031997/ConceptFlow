package application_test

import (
	"context"
	"errors"
	"testing"

	"authoring/internal/application"
	"authoring/internal/domain"
)

type fakeArchetypes struct{ rows []domain.VideoArchetype }

func (f *fakeArchetypes) ListArchetypes(context.Context) ([]domain.VideoArchetype, error) {
	return f.rows, nil
}
func (f *fakeArchetypes) GetArchetype(_ context.Context, id string) (domain.VideoArchetype, error) {
	for _, r := range f.rows {
		if r.ID == id {
			return r, nil
		}
	}
	return domain.VideoArchetype{}, application.ErrArchetypeNotFound
}
func (f *fakeArchetypes) CreateArchetype(_ context.Context, a domain.VideoArchetype) (domain.VideoArchetype, error) {
	a.ID = "new-" + a.Code
	f.rows = append(f.rows, a)
	return a, nil
}
func (f *fakeArchetypes) UpdateArchetype(_ context.Context, a domain.VideoArchetype) (domain.VideoArchetype, error) {
	return a, nil
}
func (f *fakeArchetypes) DeleteArchetype(context.Context, string) error { return nil }

func newArchetypeUC() (*application.VideoArchetypesUseCase, *fakeArchetypes) {
	f := &fakeArchetypes{rows: domain.SystemVideoArchetypes()}
	return application.NewVideoArchetypesUseCase(f), f
}

func valid() domain.VideoArchetype {
	return domain.VideoArchetype{Name: "n", WhenToUse: "w", Playbook: "p"}
}

func TestArchetypeCreate_BlankCodeGetsNextFreeLetter(t *testing.T) {
	uc, _ := newArchetypeUC()
	got, err := uc.Create(context.Background(), valid())
	if err != nil || got.Code != "E" {
		t.Fatalf("want code E, got %q err %v", got.Code, err)
	}
}

func TestArchetypeCreate_ValidatesFields(t *testing.T) {
	uc, _ := newArchetypeUC()
	for name, mut := range map[string]func(*domain.VideoArchetype){
		"name":      func(a *domain.VideoArchetype) { a.Name = " " },
		"when":      func(a *domain.VideoArchetype) { a.WhenToUse = "" },
		"playbook":  func(a *domain.VideoArchetype) { a.Playbook = "" },
		"long code": func(a *domain.VideoArchetype) { a.Code = "ABCDEFGHI" },
		"bad code":  func(a *domain.VideoArchetype) { a.Code = "A B" },
	} {
		a := valid()
		mut(&a)
		if _, err := uc.Create(context.Background(), a); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestArchetype_SystemRowsAreReadOnlyButCopyable(t *testing.T) {
	uc, _ := newArchetypeUC()
	ctx := context.Background()
	if _, err := uc.Update(ctx, "system-A", valid()); !errors.Is(err, application.ErrArchetypeReadOnly) {
		t.Errorf("update: want read-only, got %v", err)
	}
	if err := uc.Delete(ctx, "system-A"); !errors.Is(err, application.ErrArchetypeReadOnly) {
		t.Errorf("delete: want read-only, got %v", err)
	}
	cp, err := uc.Copy(ctx, "system-B")
	if err != nil || cp.Code != "E" || cp.IsSystem {
		t.Errorf("copy: got %+v err %v", cp, err)
	}
}
