package postgres

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"visual/internal/application"
	"visual/internal/domain"
)

func TestPlanRepository(t *testing.T) {
	data := openTestPool(t)
	repo := NewPlanRepository(data.pool)
	ctx := context.Background()
	const p = "test-plan-repo"
	t.Cleanup(func() { _ = data.DeleteProjectData(ctx, p) })

	if _, err := repo.LatestConceptVersion(ctx, p); !errors.Is(err, domain.ErrNoVersion) {
		t.Fatalf("a video with no locked concept: want ErrNoVersion, got %v", err)
	}
	for v := 1; v <= 2; v++ {
		rec := application.VersionRecord{ProjectID: p, Version: v, ScriptVersion: 3, Document: []byte(`{"document_id":"c"}`), LockedAt: time.Now()}
		if err := repo.InsertVersion(ctx, rec); err != nil {
			t.Fatal(err)
		}
	}
	latest, err := repo.LatestConceptVersion(ctx, p)
	if err != nil || latest.Version != 2 || string(latest.Document) != `{"document_id": "c"}` {
		t.Fatalf("the newest concept version with its document: %+v %v", latest, err)
	}

	first, err := repo.ReserveUnitNumbers(ctx, p, 1)
	if err != nil || first != 1 {
		t.Fatalf("first number: %d %v", first, err)
	}
	if first, _ = repo.ReserveUnitNumbers(ctx, p, 3); first != 2 {
		t.Errorf("three numbers after 1 start at 2, got %d", first)
	}
	if first, _ = repo.ReserveUnitNumbers(ctx, p, 1); first != 5 {
		t.Errorf("the next number is 5, got %d", first)
	}
	if _, err := repo.ReserveUnitNumbers(ctx, p, 0); err == nil {
		t.Error("reserving no number must fail")
	}
	var wg sync.WaitGroup
	got := make(chan int, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			n, err := repo.ReserveUnitNumbers(ctx, p, 1)
			if err != nil {
				t.Error(err)
			}
			got <- n
		}()
	}
	wg.Wait()
	close(got)
	seen := map[int]bool{}
	for n := range got {
		if seen[n] {
			t.Errorf("number %d was handed out twice", n)
		}
		seen[n] = true
	}

	if v, _ := repo.PlanVersions(ctx, p); len(v) != 0 {
		t.Fatalf("no plan version yet: %v", v)
	}
	rec := application.PlanVersionRecord{ProjectID: p, Version: 1, ScriptVersion: 3, ConceptVersion: 2, Document: []byte(`{"units":[]}`), Note: "n", LockedAt: time.Now()}
	if err := repo.InsertPlanVersion(ctx, rec); err != nil {
		t.Fatal(err)
	}
	if err := repo.InsertPlanVersion(ctx, rec); !errors.Is(err, domain.ErrVersionExists) {
		t.Errorf("a taken version number: want ErrVersionExists, got %v", err)
	}
	list, err := repo.PlanVersions(ctx, p)
	if err != nil || len(list) != 1 || list[0].ConceptVersion != 2 || list[0].Document != nil {
		t.Fatalf("list must hold versions without documents: %+v %v", list, err)
	}
	one, err := repo.PlanVersion(ctx, p, 1)
	if err != nil || string(one.Document) != `{"units": []}` || one.Note != "n" || one.ScriptVersion != 3 {
		t.Fatalf("one version with its document: %+v %v", one, err)
	}
	if _, err := repo.PlanVersion(ctx, p, 9); !errors.Is(err, domain.ErrNoVersion) {
		t.Errorf("unknown version: want ErrNoVersion, got %v", err)
	}

	plan := domain.ReviewUnit{ProjectID: p, Kind: domain.UnitKindPlan, Key: "b1", Status: domain.UnitPending,
		Content: []byte(`{"units":[]}`), CreatorFields: []string{"VP-001.communication_goal"}, InputHashes: map[string]string{"beat": "h"}}
	if err := repo.SaveUnit(ctx, plan); err != nil {
		t.Fatal(err)
	}
	units, err := repo.ListUnits(ctx, p, domain.UnitKindPlan)
	if err != nil || len(units) != 1 || units[0].CreatorFields[0] != "VP-001.communication_goal" || units[0].InputHashes["beat"] != "h" {
		t.Fatalf("a beat's units round trip: %+v %v", units, err)
	}

	if err := data.DeleteProjectData(ctx, p); err != nil {
		t.Fatal(err)
	}
	if v, _ := repo.PlanVersions(ctx, p); len(v) != 0 {
		t.Errorf("deleting the video must remove its plan versions: %v", v)
	}
	if u, _ := repo.ListUnits(ctx, p, domain.UnitKindPlan); len(u) != 0 {
		t.Errorf("deleting the video must remove its plan units: %v", u)
	}
	if first, _ = repo.ReserveUnitNumbers(ctx, p, 1); first != 1 {
		t.Errorf("deleting the video must reset its unit numbers, got %d", first)
	}
}
