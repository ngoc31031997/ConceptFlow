package postgres

import (
	"errors"
	"testing"
	"time"

	"authoring/internal/domain"
)

func TestScriptVersionsAreAppendOnlyAndListedOldestFirst(t *testing.T) {
	r, ctx := researchRepo(t, "sv-test")
	if _, err := r.LatestScriptVersion(ctx, "sv-test"); !errors.Is(err, domain.ErrNoLockedVersion) {
		t.Fatalf("err = %v, want ErrNoLockedVersion", err)
	}
	part, err := domain.NewVersionPart(domain.PartOrder, []string{"b1"})
	if err != nil {
		t.Fatal(err)
	}
	for v := 1; v <= 2; v++ {
		if err := r.SaveScriptVersion(ctx, domain.ScriptVersion{ProjectID: "sv-test", Version: v, Note: "n", LockedAt: time.Now().UTC(), Parts: []domain.VersionPart{part}}); err != nil {
			t.Fatal(err)
		}
	}
	if err := r.SaveScriptVersion(ctx, domain.ScriptVersion{ProjectID: "sv-test", Version: 2, LockedAt: time.Now().UTC(), Parts: []domain.VersionPart{part}}); !errors.Is(err, domain.ErrInvalidStatus) {
		t.Errorf("saving v2 twice: err = %v, want ErrInvalidStatus", err)
	}
	all, err := r.ListScriptVersions(ctx, "sv-test")
	if err != nil || len(all) != 2 || all[0].Version != 1 || all[1].Parts[0].SHA != part.SHA {
		t.Fatalf("list = %+v err %v", all, err)
	}
	if latest, _ := r.LatestScriptVersion(ctx, "sv-test"); latest.Version != 2 {
		t.Errorf("latest = %d, want 2", latest.Version)
	}
	if err := r.DeleteAuthoring(ctx, "sv-test"); err != nil {
		t.Fatal(err)
	}
	if left, _ := r.ListScriptVersions(ctx, "sv-test"); len(left) != 0 {
		t.Errorf("%d versions left after the project was deleted", len(left))
	}
}
