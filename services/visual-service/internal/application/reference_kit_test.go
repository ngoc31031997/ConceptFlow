package application

import (
	"context"
	"errors"
	"testing"

	"visual/internal/domain"
)

func (f *fakeReferenceStore) ReferenceStyleKit(_ context.Context, id string) (*domain.ReferenceKit, error) {
	switch id {
	case "R99":
		return nil, ErrReferenceStyleNotFound
	case "R02":
		return nil, nil
	}
	return &domain.ReferenceKit{Ink: "#112233"}, nil
}

func (f *fakeReferenceStore) ProjectKit(_ context.Context, projectID string) (string, *domain.ReferenceKit, error) {
	switch projectID {
	case "none":
		return "", nil, ErrNoReferencePicked
	case "nokit":
		return "R02", nil, nil
	}
	return "R01", &domain.ReferenceKit{Ink: "#112233"}, nil
}

func TestReferenceLibrary_KitOf(t *testing.T) {
	uc := NewReferenceLibraryUseCase(&fakeReferenceStore{})
	ctx := context.Background()
	if res, err := uc.KitOf(ctx, "R01"); err != nil || res.RefID != "R01" || res.Kit.Ink != "#112233" {
		t.Errorf("kit of R01: %+v, %v", res, err)
	}
	if _, err := uc.KitOf(ctx, "R02"); !errors.Is(err, ErrReferenceKitMissing) {
		t.Errorf("style without kit: err %v, want ErrReferenceKitMissing", err)
	}
	if _, err := uc.KitOf(ctx, "R99"); !errors.Is(err, ErrReferenceStyleNotFound) {
		t.Errorf("unknown style: err %v", err)
	}
}

func TestReferenceLibrary_KitOfProject(t *testing.T) {
	uc := NewReferenceLibraryUseCase(&fakeReferenceStore{})
	ctx := context.Background()
	if res, err := uc.KitOfProject(ctx, "p"); err != nil || res.RefID != "R01" {
		t.Errorf("project kit: %+v, %v", res, err)
	}
	if _, err := uc.KitOfProject(ctx, "none"); !errors.Is(err, ErrNoReferencePicked) {
		t.Errorf("no pick: err %v", err)
	}
	if _, err := uc.KitOfProject(ctx, "nokit"); !errors.Is(err, ErrReferenceKitMissing) {
		t.Errorf("no kit: err %v", err)
	}
}
