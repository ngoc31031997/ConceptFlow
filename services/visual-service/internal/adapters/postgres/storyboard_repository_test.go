package postgres

import (
	"context"
	"testing"

	"visual/internal/domain"
)

func TestStoryboardRepository(t *testing.T) {
	data := openTestPool(t)
	repo := NewStoryboardRepository(data.pool)
	ctx := context.Background()
	const p = "test-storyboard-repo"
	t.Cleanup(func() { _ = data.DeleteProjectData(ctx, p) })

	if m, err := repo.StoryboardModel(ctx, p); err != nil || m != "" {
		t.Fatalf("a video with no model of its own: %q %v", m, err)
	}
	if err := repo.SetSequenceModel(ctx, p, "ollama"); err != nil {
		t.Fatal(err)
	}
	if err := repo.SetStoryboardModel(ctx, p, "deepseek-chat"); err != nil {
		t.Fatal(err)
	}
	if m, _ := repo.StoryboardModel(ctx, p); m != "deepseek-chat" {
		t.Errorf("the storyboard model is its own, got %q", m)
	}
	if m, _ := repo.SequenceModel(ctx, p); m != "ollama" {
		t.Errorf("saving the storyboard model must not touch the sequence model, got %q", m)
	}

	// The three kinds of review units round-trip and are listed apart.
	shots := domain.UnitShots{NextSeq: 2, Shots: []domain.Shot{{ShotID: "SH-001", ScriptRefs: []string{"b1.1"}, Status: domain.UnitPending}},
		Frames: map[string]domain.DrawnFrame{"SH-001": {FramePath: "storyboard/SH-001.png", SpecHash: "abc", FrameHash: "h"}}}
	raw, err := shots.Encode()
	if err != nil {
		t.Fatal(err)
	}
	for _, u := range []domain.ReviewUnit{
		{ProjectID: p, Kind: domain.UnitKindStoryboardUnit, Key: "VP-001", Status: domain.UnitPending, Content: raw, CreatorFields: []string{"SH-001.notes"}, InputHashes: map[string]string{"plan": "x"}},
		{ProjectID: p, Kind: domain.UnitKindStoryboardAssets, Key: domain.ReviewUnitKey, Status: domain.UnitPending, Content: []byte(`{"assets":[{"asset_id":"AST-A-001","status":"approved"}],"next_seq":1}`), CreatorFields: []string{}, InputHashes: map[string]string{}},
		{ProjectID: p, Kind: domain.UnitKindStoryboardReview, Key: domain.ReviewUnitKey, Status: domain.UnitPending, Content: []byte(`{"findings":[],"next_seq":1,"notes":[],"rounds":1,"checks":{}}`), CreatorFields: []string{}, InputHashes: map[string]string{}},
	} {
		if err := repo.SaveUnit(ctx, u); err != nil {
			t.Fatalf("%s: %v", u.Kind, err)
		}
	}
	rows, err := repo.ListUnits(ctx, p, domain.UnitKindStoryboardUnit)
	if err != nil || len(rows) != 1 || rows[0].Key != "VP-001" || rows[0].CreatorFields[0] != "SH-001.notes" || rows[0].InputHashes["plan"] != "x" {
		t.Fatalf("units %+v %v", rows, err)
	}
	got, err := domain.DecodeUnitShots(rows[0].Content)
	if err != nil || len(got.Shots) != 1 || got.Frames["SH-001"].SpecHash != "abc" || got.NextSeq != 2 {
		t.Fatalf("the stored shots keep the frame record: %+v %v", got, err)
	}
	for _, kind := range []string{domain.UnitKindStoryboardAssets, domain.UnitKindStoryboardReview} {
		if rows, err := repo.ListUnits(ctx, p, kind); err != nil || len(rows) != 1 {
			t.Errorf("%s: %+v %v", kind, rows, err)
		}
	}

	if err := data.DeleteProjectData(ctx, p); err != nil {
		t.Fatal(err)
	}
	if rows, _ := repo.ListUnits(ctx, p, domain.UnitKindStoryboardUnit); len(rows) != 0 {
		t.Errorf("deleting the video removes its storyboard units, left %v", rows)
	}
	if m, _ := repo.StoryboardModel(ctx, p); m != "" {
		t.Errorf("deleting the video removes its storyboard model: %q", m)
	}
}
