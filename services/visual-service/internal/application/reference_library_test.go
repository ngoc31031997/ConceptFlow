package application

import (
	"context"
	"errors"
	"testing"

	"visual/internal/domain"
)

type fakeReferenceStore struct {
	created domain.ReferenceStyleInput
	updated domain.ReferenceStyleInput
	calls   int
}

func (f *fakeReferenceStore) ListReferenceStyles(context.Context) ([]domain.ReferenceStyle, error) {
	return nil, nil
}

func (f *fakeReferenceStore) CreateReferenceStyle(_ context.Context, in domain.ReferenceStyleInput) (domain.ReferenceStyle, error) {
	f.calls++
	f.created = in
	return domain.ReferenceStyle{ID: "R01", ReferenceStyleInput: domain.ReferenceStyleInput{StyleName: in.StyleName}}, nil
}

func (f *fakeReferenceStore) UpdateReferenceStyle(_ context.Context, id string, in domain.ReferenceStyleInput) (domain.ReferenceStyle, error) {
	f.calls++
	f.updated = in
	return domain.ReferenceStyle{ID: id, ReferenceStyleInput: domain.ReferenceStyleInput{StyleName: in.StyleName}}, nil
}

func (f *fakeReferenceStore) DeleteReferenceStyle(context.Context, string) error { return nil }

func (f *fakeReferenceStore) ReferenceStyleUsers(context.Context, string) ([]string, error) {
	return []string{"p1"}, nil
}

func TestReferenceLibrary_StoresTheNormalizedInput(t *testing.T) {
	store := &fakeReferenceStore{}
	uc := NewReferenceLibraryUseCase(store)
	if _, err := uc.Create(context.Background(), domain.ReferenceStyleInput{StyleName: " a ", Intent: " d ", Tags: []string{"x", "x"}}); err != nil {
		t.Fatal(err)
	}
	if store.created.StyleName != "a" || store.created.Intent != "d" || len(store.created.Tags) != 1 {
		t.Errorf("create stored %+v, want the trimmed input with one tag", store.created)
	}
	if _, err := uc.Update(context.Background(), "R01", domain.ReferenceStyleInput{StyleName: " b ", Intent: "d"}); err != nil {
		t.Fatal(err)
	}
	if store.updated.StyleName != "b" {
		t.Errorf("update stored %+v, want the trimmed name", store.updated)
	}
}

func TestReferenceLibrary_InvalidInputNeverReachesTheStore(t *testing.T) {
	store := &fakeReferenceStore{}
	uc := NewReferenceLibraryUseCase(store)
	if _, err := uc.Create(context.Background(), domain.ReferenceStyleInput{StyleName: "", Intent: "d"}); !errors.Is(err, domain.ErrInvalidReferenceStyle) {
		t.Errorf("create: err %v", err)
	}
	if _, err := uc.Update(context.Background(), "R01", domain.ReferenceStyleInput{StyleName: "n"}); !errors.Is(err, domain.ErrInvalidReferenceStyle) {
		t.Errorf("update: err %v", err)
	}
	if store.calls != 0 {
		t.Errorf("store was called %d times", store.calls)
	}
}
