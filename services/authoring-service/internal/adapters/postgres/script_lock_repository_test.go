package postgres

import (
	"encoding/json"
	"testing"

	"authoring/internal/domain"
)

// A handoff note is a row of project_handoff: its sections and review state
// read back as saved, notes list in beat order, and a deleted note is gone.
func TestHandoffUnitRoundTripInPostgres(t *testing.T) {
	const pid = "handoff-test"
	r, ctx := researchRepo(t, pid)
	for _, key := range []string{"b10", "b2"} {
		raw, _ := json.Marshal(domain.HandoffNote{Must: []string{"F01 " + key}, Unsure: []string{"chưa chắc"}})
		if _, err := r.SaveUnit(ctx, domain.ReviewUnit{ProjectID: pid, Kind: domain.UnitKindHandoffBeat, Key: key,
			Status: domain.UnitApproved, Content: raw, CreatorFields: []string{"must"}, RedoCount: 2}); err != nil {
			t.Fatal(err)
		}
	}
	got, err := r.GetUnit(ctx, pid, domain.UnitKindHandoffBeat, "b2")
	if err != nil {
		t.Fatal(err)
	}
	var n domain.HandoffNote
	if err := json.Unmarshal(got.Content, &n); err != nil {
		t.Fatal(err)
	}
	if len(n.Must) != 1 || n.Must[0] != "F01 b2" || len(n.Unsure) != 1 || n.Terms == nil {
		t.Errorf("content = %s", got.Content)
	}
	if got.Status != domain.UnitApproved || got.RedoCount != 2 || !got.HasCreatorField("must") {
		t.Errorf("unit = %+v", got)
	}
	listed, err := r.ListUnits(ctx, pid, domain.UnitKindHandoffBeat)
	if err != nil || len(listed) != 2 || listed[0].Key != "b2" || listed[1].Key != "b10" {
		t.Fatalf("listed = %+v, %v, want b2 before b10", listed, err)
	}
	if err := r.DeleteUnit(ctx, pid, domain.UnitKindHandoffBeat, "b2"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.GetUnit(ctx, pid, domain.UnitKindHandoffBeat, "b2"); err != domain.ErrUnitNotFound {
		t.Errorf("err = %v, want the deleted note not found", err)
	}
}

// The script review is one row of project_script_review that reads back whole.
func TestScriptReviewUnitRoundTripInPostgres(t *testing.T) {
	const pid = "script-review-test"
	r, ctx := researchRepo(t, pid)
	if _, err := r.GetUnit(ctx, pid, domain.UnitKindScriptReview, domain.ScriptReviewUnitKey); err != domain.ErrUnitNotFound {
		t.Fatalf("err = %v, want no review before the first save", err)
	}
	rev := domain.ScriptReview{
		Findings:  []domain.ReviewFinding{{ID: "r1", Source: "ai", Group: "logic", Title: "T", Detail: "D", Line: "b1.1", Fix: "Câu mới."}},
		NextSeq:   2,
		Decisions: map[string]domain.FindingDecision{"r1": {State: "accepted", Why: "giữ", Fingerprint: "ab"}},
		AINotes:   []string{"ổn"}, MyNotes: []string{"của tôi"}, ReviewedSHA: "sha", Rounds: 1,
	}
	raw, _ := json.Marshal(rev)
	if _, err := r.SaveUnit(ctx, domain.ReviewUnit{ProjectID: pid, Kind: domain.UnitKindScriptReview,
		Key: domain.ScriptReviewUnitKey, Status: domain.UnitPending, Content: raw}); err != nil {
		t.Fatal(err)
	}
	got, err := r.GetUnit(ctx, pid, domain.UnitKindScriptReview, domain.ScriptReviewUnitKey)
	if err != nil {
		t.Fatal(err)
	}
	var back domain.ScriptReview
	if err := json.Unmarshal(got.Content, &back); err != nil {
		t.Fatal(err)
	}
	if len(back.Findings) != 1 || back.Findings[0].Fix != "Câu mới." || back.NextSeq != 2 ||
		back.Decisions["r1"].Why != "giữ" || back.AINotes[0] != "ổn" || back.MyNotes[0] != "của tôi" ||
		back.ReviewedSHA != "sha" || back.Rounds != 1 {
		t.Errorf("review = %s", got.Content)
	}
}

// The glossary is kept one row per term, in order; saving replaces the terms.
func TestGlossaryUnitRoundTripInPostgres(t *testing.T) {
	const pid = "glossary-test"
	r, ctx := researchRepo(t, pid)
	save := func(terms ...domain.GlossaryTerm) {
		t.Helper()
		raw, _ := json.Marshal(domain.Glossary{Terms: terms})
		if _, err := r.SaveUnit(ctx, domain.ReviewUnit{ProjectID: pid, Kind: domain.UnitKindGlossary,
			Key: domain.GlossaryUnitKey, Status: domain.UnitPending, Content: raw}); err != nil {
			t.Fatal(err)
		}
	}
	read := func() domain.Glossary {
		t.Helper()
		u, err := r.GetUnit(ctx, pid, domain.UnitKindGlossary, domain.GlossaryUnitKey)
		if err == domain.ErrUnitNotFound {
			return domain.Glossary{}
		}
		if err != nil {
			t.Fatal(err)
		}
		var g domain.Glossary
		if err := json.Unmarshal(u.Content, &g); err != nil {
			t.Fatal(err)
		}
		return g
	}
	save(domain.GlossaryTerm{Term: "chân trời", Say: "ranh giới", Mine: true}, domain.GlossaryTerm{Term: "lỗ đen", Draw: "vòng tối"})
	g := read()
	if len(g.Terms) != 2 || g.Terms[0].Term != "chân trời" || !g.Terms[0].Mine || g.Terms[1].Draw != "vòng tối" {
		t.Errorf("glossary = %+v", g)
	}
	save(domain.GlossaryTerm{Term: "lỗ đen"})
	if g := read(); len(g.Terms) != 1 || g.Terms[0].Term != "lỗ đen" {
		t.Errorf("glossary after replace = %+v", g)
	}
	save()
	if g := read(); len(g.Terms) != 0 {
		t.Errorf("glossary after clearing = %+v", g)
	}
}

// The research gate is one row of project_research_gate that reads back whole.
func TestResearchGateUnitRoundTripInPostgres(t *testing.T) {
	const pid = "research-gate-test"
	r, ctx := researchRepo(t, pid)
	raw, _ := json.Marshal(domain.ResearchGate{Skipped: true, NextQuestion: 4})
	if _, err := r.SaveUnit(ctx, domain.ReviewUnit{ProjectID: pid, Kind: domain.UnitKindResearch,
		Key: domain.ResearchUnitKey, Status: domain.UnitApproved, Content: raw}); err != nil {
		t.Fatal(err)
	}
	got, err := r.GetUnit(ctx, pid, domain.UnitKindResearch, domain.ResearchUnitKey)
	if err != nil {
		t.Fatal(err)
	}
	var g domain.ResearchGate
	if err := json.Unmarshal(got.Content, &g); err != nil {
		t.Fatal(err)
	}
	if !g.Skipped || g.NextQuestion != 4 || got.Status != domain.UnitApproved {
		t.Errorf("gate = %s, unit = %+v", got.Content, got)
	}
}
