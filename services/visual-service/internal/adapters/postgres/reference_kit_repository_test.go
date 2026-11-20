package postgres

import (
	"context"
	"errors"
	"testing"

	"visual/internal/application"
	"visual/internal/domain"
)

func TestReferenceStyles_KitRoundTripAndProjectKit(t *testing.T) {
	repo := openReferenceRepository(t)
	ctx := context.Background()
	project := referenceTestPrefix + "-kit-project"

	if _, _, err := repo.ProjectKit(ctx, project); !errors.Is(err, application.ErrNoReferencePicked) {
		t.Fatalf("no pick: err %v", err)
	}
	plain, err := repo.CreateReferenceStyle(ctx, domain.ReferenceStyleInput{StyleName: referenceTestPrefix + "-nokit", Intent: "d", Tags: []string{}})
	if err != nil {
		t.Fatal(err)
	}
	if kit, err := repo.ReferenceStyleKit(ctx, plain.ID); err != nil || kit != nil {
		t.Fatalf("style without kit: %+v, %v", kit, err)
	}
	withKit, err := repo.CreateReferenceStyle(ctx, domain.ReferenceStyleInput{
		StyleName: referenceTestPrefix + "-kit", Intent: "d", Tags: []string{}, Kit: &domain.ReferenceKit{Ink: "#112233", Grain: 0.25},
	})
	if err != nil {
		t.Fatal(err)
	}
	if withKit.Kit == nil || withKit.Kit.Grain != 0.25 {
		t.Fatalf("kit not stored: %+v", withKit.Kit)
	}
	if _, err := repo.ReferenceStyleKit(ctx, "R0000000"); !errors.Is(err, application.ErrReferenceStyleNotFound) {
		t.Fatalf("unknown style: err %v", err)
	}
	concepts := NewConceptRepository(repo.pool)
	if err := concepts.SetPicks(ctx, project, []string{withKit.ID}); err != nil {
		t.Fatal(err)
	}
	if id, kit, err := repo.ProjectKit(ctx, project); err != nil || id != withKit.ID || kit == nil {
		t.Fatalf("picked style: %s %+v, %v", id, kit, err)
	}
	if err := concepts.SetPicks(ctx, project, []string{plain.ID}); err != nil {
		t.Fatal(err)
	}
	id, kit, err := repo.ProjectKit(ctx, project)
	if err != nil {
		t.Fatal(err)
	}
	if id != plain.ID || kit != nil {
		t.Errorf("the new pick replaces the old: got %s %+v, want %s without kit", id, kit, plain.ID)
	}
}
