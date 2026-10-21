package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"authoring/internal/domain"
)

// Review units and the idea step's columns against a real database
// (TEST_DATABASE_URL): a unit round-trips, the target length is stored, a
// story outline is stamped with the brief approved when it was saved, and a
// deleted project takes its units with it.
func TestReviewUnitsAgainstPostgres(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := NewPool(ctx, url, 4)
	if err != nil {
		t.Fatal(err)
	}
	r := NewPromptTemplateRepository(pool)
	const pid = "review-units-test"
	t.Cleanup(func() { _ = r.DeleteAuthoring(ctx, pid) })
	_ = r.DeleteAuthoring(ctx, pid)

	if _, err := r.GetUnit(ctx, pid, domain.UnitKindBrief, domain.BriefUnitKey); !errors.Is(err, domain.ErrUnitNotFound) {
		t.Fatalf("no unit yet must be ErrUnitNotFound, got %v", err)
	}
	content, _ := json.Marshal(domain.CreativeBrief{Question: "Q", Tone: "T"})
	saved, err := r.SaveUnit(ctx, domain.ReviewUnit{
		ProjectID: pid, Kind: domain.UnitKindBrief, Key: domain.BriefUnitKey,
		Status: domain.UnitPending, Content: content, RedoCount: 2,
	})
	if err != nil {
		t.Fatalf("save unit: %v", err)
	}
	if saved.Status != domain.UnitPending || saved.RedoCount != 2 || len(saved.CreatorFields) != 0 {
		t.Errorf("unit stored wrong: %+v", saved)
	}
	saved.CreatorFields = []string{"tone"}
	saved.Status = domain.UnitApproved
	if _, err := r.SaveUnit(ctx, saved); err != nil {
		t.Fatal(err)
	}
	got, err := r.GetUnit(ctx, pid, domain.UnitKindBrief, domain.BriefUnitKey)
	if err != nil || got.Status != domain.UnitApproved || !got.HasCreatorField("tone") {
		t.Fatalf("updated unit read back wrong: %+v, %v", got, err)
	}

	models := domain.AuthoringStepModels{Brief: "ollama", Publish: "zai-org/glm-5.3-flash", Research: "ollama", Story: "zai-org/glm-5.3-flash"}
	if err := r.SaveAuthoringModels(ctx, pid, models); err != nil {
		t.Fatal(err)
	}
	if got, err := r.GetChosenAuthoringModels(ctx, pid); err != nil || got != models {
		t.Fatalf("the brief's model must round-trip with the others, got %+v, %v", got, err)
	}

	if err := r.SaveTargetDuration(ctx, pid, domain.Duration5to7); err != nil {
		t.Fatal(err)
	}
	if d, err := r.GetTargetDuration(ctx, pid); err != nil || d != domain.Duration5to7 {
		t.Fatalf("target duration read back %q, %v", d, err)
	}

	if link, err := r.StoryBriefLink(ctx, pid); err != nil || link.HasStory {
		t.Fatalf("no story yet, got %+v, %v", link, err)
	}
	if err := r.SaveAuthoringStory(ctx, pid, "dàn ý", ""); err != nil {
		t.Fatal(err)
	}
	if link, err := r.StoryBriefLink(ctx, pid); err != nil || !link.HasStory || !link.FromCurrentBrief {
		t.Fatalf("a story saved under the approved brief is current, got %+v, %v", link, err)
	}
	changed, _ := json.Marshal(domain.CreativeBrief{Question: "Q2", Tone: "T"})
	got.Content = changed
	if _, err := r.SaveUnit(ctx, got); err != nil {
		t.Fatal(err)
	}
	if link, err := r.StoryBriefLink(ctx, pid); err != nil || link.FromCurrentBrief {
		t.Fatalf("changing the approved brief makes the story stale, got %+v, %v", link, err)
	}

	if err := r.DeleteAuthoring(ctx, pid); err != nil {
		t.Fatal(err)
	}
	if _, err := r.GetUnit(ctx, pid, domain.UnitKindBrief, domain.BriefUnitKey); !errors.Is(err, domain.ErrUnitNotFound) {
		t.Fatalf("deleting the project must delete its units, got %v", err)
	}
}
