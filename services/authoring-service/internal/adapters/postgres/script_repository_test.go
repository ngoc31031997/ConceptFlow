package postgres

import (
	"encoding/json"
	"testing"

	"authoring/internal/domain"
)

func scriptUnit(pid, key string, lines ...string) domain.ReviewUnit {
	s := domain.ScriptBeat{NextSeq: len(lines) + 1}
	for i, l := range lines {
		s.Lines = append(s.Lines, domain.ScriptLine{ID: domain.ScriptLineID(key, i+1), Text: l, Facts: []string{"F01"}})
	}
	raw, _ := json.Marshal(s)
	return domain.ReviewUnit{ProjectID: pid, Kind: domain.UnitKindScriptBeat, Key: key, Status: domain.UnitPending,
		Content: raw, CreatorFields: []string{"lines"}, RedoCount: 1}
}

// The dialogue of a beat is a row of project_script: it reads back as the unit
// that was saved, lists in beat order, and goes when deleted.
func TestScriptUnitRoundTripInPostgres(t *testing.T) {
	const pid = "script-test"
	r, ctx := researchRepo(t, pid)
	for _, u := range []domain.ReviewUnit{scriptUnit(pid, "b10", "Mười."), scriptUnit(pid, "b2", "Một.", "Hai.")} {
		if _, err := r.SaveUnit(ctx, u); err != nil {
			t.Fatal(err)
		}
	}
	got, err := r.GetUnit(ctx, pid, domain.UnitKindScriptBeat, "b2")
	if err != nil {
		t.Fatal(err)
	}
	var s domain.ScriptBeat
	if err := json.Unmarshal(got.Content, &s); err != nil {
		t.Fatal(err)
	}
	if len(s.Lines) != 2 || s.Lines[1].ID != "b2.2" || s.NextSeq != 3 || s.Lines[0].Facts[0] != "F01" {
		t.Errorf("content = %s", got.Content)
	}
	if got.Kind != domain.UnitKindScriptBeat || got.Status != domain.UnitPending || got.RedoCount != 1 || !got.HasCreatorField("lines") {
		t.Errorf("unit = %+v", got)
	}
	listed, err := r.ListUnits(ctx, pid, domain.UnitKindScriptBeat)
	if err != nil || len(listed) != 2 || listed[0].Key != "b2" || listed[1].Key != "b10" {
		t.Fatalf("listed = %+v, %v, want b2 before b10", listed, err)
	}
	if sha, _ := r.UnitSHA(ctx, pid, domain.UnitKindScriptBeat, "b2"); sha == "" {
		t.Error("UnitSHA of a written dialogue must not be empty")
	}
	if err := r.DeleteUnit(ctx, pid, domain.UnitKindScriptBeat, "b2"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.GetUnit(ctx, pid, domain.UnitKindScriptBeat, "b2"); err != domain.ErrUnitNotFound {
		t.Errorf("err = %v, want the deleted dialogue not found", err)
	}
}

// A unit built from a beat's dialogue (its handoff note) reads as stale once
// that dialogue changes.
func TestScriptContentHashDrivesTheStaleFlagOfAHandoff(t *testing.T) {
	const pid = "script-handoff-test"
	r, ctx := researchRepo(t, pid)
	if _, err := r.SaveUnit(ctx, scriptUnit(pid, "b1", "Xin chào.")); err != nil {
		t.Fatal(err)
	}
	sha, _ := r.UnitSHA(ctx, pid, domain.UnitKindScriptBeat, "b1")
	if _, err := r.SaveUnit(ctx, domain.ReviewUnit{
		ProjectID: pid, Kind: domain.UnitKindHandoffBeat, Key: "b1", Status: domain.UnitPending, Content: []byte(`{}`),
		InputRef: domain.UnitKindScriptBeat + ":b1", InputSHA: sha,
	}); err != nil {
		t.Fatal(err)
	}
	if u, _ := r.GetUnit(ctx, pid, domain.UnitKindHandoffBeat, "b1"); u.Stale {
		t.Fatal("a handoff built from the dialogue as it is must not read as stale")
	}
	if _, err := r.SaveUnit(ctx, scriptUnit(pid, "b1", "Xin chào.", "Câu mới.")); err != nil {
		t.Fatal(err)
	}
	if u, _ := r.GetUnit(ctx, pid, domain.UnitKindHandoffBeat, "b1"); !u.Stale {
		t.Error("a handoff built from dialogue that changed must read as stale")
	}
}
