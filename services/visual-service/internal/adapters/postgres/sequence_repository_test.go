package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"visual/internal/application"
	"visual/internal/domain"
)

func TestSequenceRepository(t *testing.T) {
	data := openTestPool(t)
	repo := NewSequenceRepository(data.pool)
	ctx := context.Background()
	const p = "test-sequence-repo"
	t.Cleanup(func() { _ = data.DeleteProjectData(ctx, p) })

	if _, err := repo.LatestPlanVersion(ctx, p); !errors.Is(err, domain.ErrNoVersion) {
		t.Fatalf("a video with no locked plan: want ErrNoVersion, got %v", err)
	}
	for v := 1; v <= 2; v++ {
		rec := application.PlanVersionRecord{ProjectID: p, Version: v, ScriptVersion: 3, ConceptVersion: 1, Document: []byte(`{"units":[]}`), LockedAt: time.Now()}
		if err := repo.InsertPlanVersion(ctx, rec); err != nil {
			t.Fatal(err)
		}
	}
	plan, err := repo.LatestPlanVersion(ctx, p)
	if err != nil || plan.Version != 2 || plan.ConceptVersion != 1 || len(plan.Document) == 0 {
		t.Fatalf("the newest plan version with its document: %+v %v", plan, err)
	}

	if v, _ := repo.SequenceVersions(ctx, p); len(v) != 0 {
		t.Fatalf("no sequence version yet: %v", v)
	}
	rec := application.SequenceVersionRecord{ProjectID: p, Version: 1, PlanVersion: 2, ConceptVersion: 1, ScriptVersion: 3,
		Document: []byte(`{"sequence":[]}`), Note: "n", LockedAt: time.Now()}
	if err := repo.InsertSequenceVersion(ctx, rec); err != nil {
		t.Fatal(err)
	}
	if err := repo.InsertSequenceVersion(ctx, rec); !errors.Is(err, domain.ErrVersionExists) {
		t.Errorf("a taken version number: want ErrVersionExists, got %v", err)
	}
	list, err := repo.SequenceVersions(ctx, p)
	if err != nil || len(list) != 1 || list[0].PlanVersion != 2 || list[0].ScriptVersion != 3 || list[0].Document != nil {
		t.Fatalf("listed versions carry their sources and no document: %+v %v", list, err)
	}
	got, err := repo.SequenceVersion(ctx, p, 1)
	if err != nil || got.Note != "n" || string(got.Document) != `{"sequence": []}` {
		t.Fatalf("one version with its document: %+v %v", got, err)
	}
	if _, err := repo.SequenceVersion(ctx, p, 9); !errors.Is(err, domain.ErrNoVersion) {
		t.Errorf("a version not locked: want ErrNoVersion, got %v", err)
	}

	if m, err := repo.SequenceModel(ctx, p); err != nil || m != "" {
		t.Fatalf("a video with no model of its own: %q %v", m, err)
	}
	if err := repo.SetPlanModel(ctx, p, "ollama"); err != nil {
		t.Fatal(err)
	}
	if err := repo.SetSequenceModel(ctx, p, "deepseek-chat"); err != nil {
		t.Fatal(err)
	}
	if m, _ := repo.SequenceModel(ctx, p); m != "deepseek-chat" {
		t.Errorf("the sequence model is its own, got %q", m)
	}
	if m, _ := repo.PlanModel(ctx, p); m != "ollama" {
		t.Errorf("saving the sequence model must not touch the plan model, got %q", m)
	}

	if err := data.DeleteProjectData(ctx, p); err != nil {
		t.Fatal(err)
	}
	if v, _ := repo.SequenceVersions(ctx, p); len(v) != 0 {
		t.Errorf("deleting the video removes its sequence versions, left %v", v)
	}
	if m, _ := repo.SequenceModel(ctx, p); m != "" {
		t.Errorf("deleting the video removes its sequence model: %q", m)
	}
}
