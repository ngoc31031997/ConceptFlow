package postgres

import (
	"errors"
	"testing"

	"authoring/internal/domain"
)

func sampleSheet(pid string) domain.StoredBeatSheet {
	return domain.StoredBeatSheet{
		ProjectID:     pid,
		BeatSheetGate: domain.BeatSheetGate{NextSeq: 3, SkeletonApproved: true, Rebalanced: 1},
		Beats: []domain.StoredBeat{
			{
				BeatSlot: domain.BeatSlot{Key: "b2", Status: domain.UnitApproved, CreatorFields: []string{"goal"}, RedoCount: 2, StoryHash: "h1"},
				Beat:     domain.Beat{Fn: "Hook", Title: "Mở", Goal: "G", Content: "C", Facts: []string{"F01"}, Dur: 18, Next: "N", Step: "S"},
			},
			{
				BeatSlot: domain.BeatSlot{Key: "b1", Status: domain.UnitPending},
				Beat:     domain.Beat{Fn: "Payoff", Title: "Kết", Dur: 30},
			},
		},
	}
}

// A sheet survives a round trip through project_beat_sheet with the order of
// its beats and its gate; saving again replaces the whole sheet.
func TestBeatSheetRoundTripInPostgres(t *testing.T) {
	const pid = "beat-sheet-test"
	r, ctx := researchRepo(t, pid)
	if _, err := r.GetBeatSheet(ctx, pid); !errors.Is(err, domain.ErrBeatSheetNotFound) {
		t.Fatalf("err = %v, want a project with no sheet not found", err)
	}
	if err := r.SaveBeatSheet(ctx, sampleSheet(pid)); err != nil {
		t.Fatal(err)
	}
	got, err := r.GetBeatSheet(ctx, pid)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Beats) != 2 || got.Beats[0].Key != "b2" || got.Beats[1].Key != "b1" {
		t.Fatalf("beats = %+v, want the order kept", got.Beats)
	}
	first := got.Beats[0]
	if first.Status != domain.UnitApproved || first.RedoCount != 2 || first.StoryHash != "h1" || !first.HasCreatorField("goal") ||
		first.Beat.Content != "C" || first.Beat.Facts[0] != "F01" || first.Beat.Dur != 18 {
		t.Errorf("first beat = %+v", first)
	}
	if got.Beats[1].CreatorFields == nil || got.Beats[1].Beat.Facts == nil {
		t.Error("empty lists must read back as empty, not nil")
	}
	if got.NextSeq != 3 || !got.SkeletonApproved || got.Rebalanced != 1 {
		t.Errorf("gate = %+v", got.BeatSheetGate)
	}
	next := sampleSheet(pid)
	next.Beats = next.Beats[1:]
	if err := r.SaveBeatSheet(ctx, next); err != nil {
		t.Fatal(err)
	}
	if got, _ = r.GetBeatSheet(ctx, pid); len(got.Beats) != 1 || got.Beats[0].Key != "b1" {
		t.Errorf("beats = %+v, want the saved sheet to replace the old one", got.Beats)
	}
}

// Dialogue written from a beat reads as stale once the beat's content changes,
// but not when only its review state changes; deleting the project removes the sheet.
func TestBeatContentHashDrivesTheStaleFlagOfDialogue(t *testing.T) {
	const pid = "beat-sheet-stale-test"
	r, ctx := researchRepo(t, pid)
	sheet := sampleSheet(pid)
	if err := r.SaveBeatSheet(ctx, sheet); err != nil {
		t.Fatal(err)
	}
	sha, err := r.UnitSHA(ctx, pid, domain.UnitKindBeat, "b2")
	if err != nil || sha == "" {
		t.Fatalf("UnitSHA = %q, %v, want the hash of beat b2", sha, err)
	}
	if none, _ := r.UnitSHA(ctx, pid, domain.UnitKindBeat, "b9"); none != "" {
		t.Errorf("UnitSHA of a missing beat = %q, want empty", none)
	}
	if _, err := r.SaveUnit(ctx, domain.ReviewUnit{
		ProjectID: pid, Kind: domain.UnitKindScriptBeat, Key: "b2", Status: domain.UnitPending, Content: []byte(`{}`),
		InputRef: domain.UnitKindBeat + ":b2", InputSHA: sha,
	}); err != nil {
		t.Fatal(err)
	}
	if u, _ := r.GetUnit(ctx, pid, domain.UnitKindScriptBeat, "b2"); u.Stale {
		t.Fatal("dialogue written from the beat as it is must not read as stale")
	}
	sheet.Beats[0].Status = domain.UnitPending
	sheet.Beats[0].RedoCount = 9
	if err := r.SaveBeatSheet(ctx, sheet); err != nil {
		t.Fatal(err)
	}
	if u, _ := r.GetUnit(ctx, pid, domain.UnitKindScriptBeat, "b2"); u.Stale {
		t.Error("a change of review state alone must not make the dialogue stale")
	}
	sheet.Beats[0].Beat.Content = "Nội dung khác"
	if err := r.SaveBeatSheet(ctx, sheet); err != nil {
		t.Fatal(err)
	}
	if u, _ := r.GetUnit(ctx, pid, domain.UnitKindScriptBeat, "b2"); !u.Stale {
		t.Error("dialogue written from a beat whose content changed must read as stale")
	}
	if listed, _ := r.ListUnits(ctx, pid, domain.UnitKindScriptBeat); len(listed) != 1 || !listed[0].Stale {
		t.Errorf("listed = %+v, want the stale flag in lists too", listed)
	}
	if err := r.DeleteAuthoring(ctx, pid); err != nil {
		t.Fatal(err)
	}
	if _, err := r.GetBeatSheet(ctx, pid); !errors.Is(err, domain.ErrBeatSheetNotFound) {
		t.Errorf("err = %v, want the sheet removed with the project", err)
	}
}
