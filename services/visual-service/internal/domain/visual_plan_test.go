package domain

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

// threeLineBeat is a script whose beat b1 has three lines of three words, so a
// unit over one line lasts a third of the beat's narration.
func threeLineBeat(m map[string]any) {
	m["script:b1"] = map[string]any{"lines": []Line{
		{ID: "b1.1", Text: "A là thật", Facts: []string{"F01"}},
		{ID: "b1.2", Text: "Có thể B", Facts: []string{"F02"}},
		{ID: "b1.3", Text: "Rồi tới D", Facts: []string{"F01"}},
	}}
}

func conceptDocJSON(t *testing.T) []byte {
	t.Helper()
	doc, err := BuildDocument("p", 1, 2, time.Unix(0, 0).UTC(), decodedParts(t, validParts()), nil, ReviewState{})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func planInput(t *testing.T, edit func(map[string]any)) PlanInput {
	t.Helper()
	in, err := NewPlanInput(scriptVersion(t, edit), 1, conceptDocJSON(t))
	if err != nil {
		t.Fatal(err)
	}
	return in
}

func goodUnit(id string, refs ...string) PlanUnit {
	return PlanUnit{
		UnitID: id, ScriptRefs: refs, NarrationSummary: "tóm tắt", CommunicationGoal: "hiểu A",
		EssentialVisualInfo:   []EssentialInfo{{Item: "ong", Priority: PriorityMust}},
		VisualApproach:        VisualApproach{Method: "diagram", Description: "sơ đồ", ConceptRuleRefs: []string{"color_system.palette.ink"}},
		NarrationRelationship: "direct_illustration", Status: UnitPending,
	}
}

func hasProblem(list []string, part string) bool {
	for _, s := range list {
		if strings.Contains(s, part) {
			return true
		}
	}
	return false
}

func TestNewPlanInput_PaceFollowsTheBriefAndTheWholeScript(t *testing.T) {
	in := planInput(t, nil)
	// 120 s over 9 words.
	if got := NarrationMin(3, in.SecondsPerWord); got != 40 {
		t.Errorf("a 3-word unit should last at least 40s, got %v", got)
	}
	if in.ConceptVersion != 1 || in.Concept.DocumentID != "visual-concept-v1" {
		t.Errorf("the locked concept was not read: %+v", in.ConceptVersion)
	}
	noDuration := planInput(t, func(m map[string]any) {
		b := m["brief"].(Brief)
		b.DurationSeconds = 0
		m["brief"] = b
	})
	if noDuration.SecondsPerWord != 0 {
		t.Errorf("pace without a brief duration must be unknown, got %v", noDuration.SecondsPerWord)
	}
}

func TestPlanInput_StaleSourcesAreOnlyTheOnesThatChanged(t *testing.T) {
	base := planInput(t, nil)
	stored := base.HashesFor("b1")
	if got := base.StaleSources("b1", stored); len(got) != 0 {
		t.Fatalf("nothing changed, yet stale: %v", got)
	}
	changedB2 := planInput(t, func(m map[string]any) {
		m["script:b2"] = map[string]any{"lines": []Line{{ID: "b2.1", Text: "Khác hẳn rồi", Facts: []string{"F03"}}}}
	})
	if got := changedB2.StaleSources("b1", stored); len(got) != 0 {
		t.Errorf("editing beat b2 must leave beat b1 fresh, got %v", got)
	}
	if got := changedB2.StaleSources("b2", base.HashesFor("b2")); len(got) != 1 || got[0] != PlanSourceBeat {
		t.Errorf("editing beat b2 must make b2 stale on its own lines, got %v", got)
	}
	changedGlossary := planInput(t, func(m map[string]any) {
		m["glossary"] = map[string]any{"terms": []GlossaryTerm{{Term: "Enzyme", Avoid: "enzim"}}}
	})
	if got := changedGlossary.StaleSources("b1", stored); len(got) != 1 || got[0] != PlanSourceGlossary {
		t.Errorf("a glossary change must stale every beat on the glossary, got %v", got)
	}
	if got := base.StaleSources("b1", nil); len(got) != len(planSourceOrder) {
		t.Errorf("a beat stored without hashes is stale on every source, got %v", got)
	}
}

func TestRuleRefs_ListEveryRuleAUnitCanPointAt(t *testing.T) {
	refs := planInput(t, nil).RuleRefs
	for _, want := range []string{
		"color_system.palette.ink", "color_system.palette.accent", "visual_language.shape_rules[0]",
		"representation_rules.characters.items.wasp", "representation_rules.objects.gate",
		"representation_rules.scientific_concepts.enz", "motion_language.preferred_transitions[0]",
		"creative_direction.principles[0]",
	} {
		if _, ok := refs[want]; !ok {
			t.Errorf("rule %s missing from %d refs", want, len(refs))
		}
	}
}

func TestCheckPlanUnit_ValidUnitPasses(t *testing.T) {
	in := planInput(t, threeLineBeat)
	pc := in.PlanContextFor("b1")
	if p := CheckPlanUnit(goodUnit("VP-001", "b1.1", "b1.2"), pc); len(p) != 0 {
		t.Fatalf("a valid unit failed: %v", p)
	}
}

func TestCheckPlanUnit_EachRuleFails(t *testing.T) {
	in := planInput(t, threeLineBeat)
	pc := in.PlanContextFor("b1")
	cases := []struct {
		name string
		edit func(*PlanUnit)
		want string
	}{
		{"bad id", func(u *PlanUnit) { u.UnitID = "X1" }, "mã unit"},
		{"no lines", func(u *PlanUnit) { u.ScriptRefs = nil }, "chưa có câu"},
		{"line of another beat", func(u *PlanUnit) { u.ScriptRefs = []string{"b2.1"} }, "không còn trong lời thoại"},
		{"gap in lines", func(u *PlanUnit) { u.ScriptRefs = []string{"b1.1", "b1.3"} }, "liền nhau"},
		{"no goal", func(u *PlanUnit) { u.CommunicationGoal = " " }, "mục tiêu truyền đạt"},
		{"no essential info", func(u *PlanUnit) { u.EssentialVisualInfo = nil }, "thông tin hình"},
		{"bad priority", func(u *PlanUnit) { u.EssentialVisualInfo[0].Priority = "high" }, "mức ưu tiên"},
		{"unknown method", func(u *PlanUnit) { u.VisualApproach.Method = "magic" }, "cách hiển thị"},
		{"unknown relationship", func(u *PlanUnit) { u.NarrationRelationship = "echo" }, "quan hệ với lời thoại"},
		{"unknown concept rule", func(u *PlanUnit) { u.VisualApproach.ConceptRuleRefs = []string{"color_system.palette.nope"} }, "không có trong ngôn ngữ hình ảnh"},
		{"risk note for a fact not shown", func(u *PlanUnit) { u.RiskNotes = []RiskNote{{FactID: "F03", Note: "x"}} }, "không có trong lời thoại"},
		{"duplicate question id", func(u *PlanUnit) {
			u.OpenQuestions = []OpenQuestion{{ID: "Q1", Question: "a"}, {ID: "Q1", Question: "b"}}
		}, "trùng mã"},
		{"shorter than the narration", func(u *PlanUnit) { u.Duration.PlannedS = 30 }, "ngắn hơn lời thoại"},
		{"longer without a reason", func(u *PlanUnit) { u.Duration.PlannedS = 90 }, "cần ghi lý do"},
	}
	for _, c := range cases {
		u := goodUnit("VP-001", "b1.1", "b1.2")
		c.edit(&u)
		if p := CheckPlanUnit(u, pc); !hasProblem(p, c.want) {
			t.Errorf("%s: want a problem containing %q, got %v", c.name, c.want, p)
		}
	}
}

func TestCheckPlanUnit_DurationRules(t *testing.T) {
	in := planInput(t, threeLineBeat)
	pc := in.PlanContextFor("b1")
	u := goodUnit("VP-001", "b1.1", "b1.2") // 6 of 12 words in a 120s brief: 60s at least
	for _, c := range []struct {
		planned float64
		reason  string
		ok      bool
	}{
		{0, "", true},  // empty means the narration minimum
		{60, "", true}, // exactly the minimum
		{120, "để so sánh", true},
		{59.9, "", false},
	} {
		u.Duration = PlannedDuration{PlannedS: c.planned, ExtraReason: c.reason}
		if got := len(CheckPlanUnit(u, pc)) == 0; got != c.ok {
			t.Errorf("planned %v reason %q: ok=%v, want %v (%v)", c.planned, c.reason, got, c.ok, CheckPlanUnit(u, pc))
		}
	}
	noPace := planInput(t, func(m map[string]any) {
		threeLineBeat(m)
		b := m["brief"].(Brief)
		b.DurationSeconds = 0
		m["brief"] = b
	})
	if p := CheckPlanUnit(goodUnit("VP-001", "b1.1"), noPace.PlanContextFor("b1")); !hasProblem(p, "không tính được mức tối thiểu") {
		t.Errorf("a video with no pace must not pass the duration check, got %v", p)
	}
}

func TestApprovalBlockers_ListUnansweredQuestions(t *testing.T) {
	u := goodUnit("VP-001", "b1.1")
	u.OpenQuestions = []OpenQuestion{{ID: "Q1", Question: "a", Answer: "đã rõ"}, {ID: "Q2", Question: "b"}}
	got := ApprovalBlockers(u)
	if len(got) != 1 || !strings.Contains(got[0], "Q2") {
		t.Errorf("only the unanswered question blocks: %v", got)
	}
}

func TestComputeCoverage_CountsLinesInExactlyOneUnit(t *testing.T) {
	in := planInput(t, threeLineBeat)
	plans := map[string]BeatPlan{
		"b1": {Units: []PlanUnit{goodUnit("VP-001", "b1.1", "b1.2"), goodUnit("VP-002", "b1.2", "b1.3")}},
	}
	cov := ComputeCoverage(in, plans)
	if cov.LinesTotal != 4 || cov.LinesCovered != 2 {
		t.Errorf("total/covered = %d/%d, want 4/2", cov.LinesTotal, cov.LinesCovered)
	}
	if strings.Join(cov.Uncovered, ",") != "b2.1" || strings.Join(cov.Duplicated, ",") != "b1.2" {
		t.Errorf("uncovered %v duplicated %v", cov.Uncovered, cov.Duplicated)
	}
	if cov.Complete() {
		t.Error("a plan with a gap and an overlap is not complete")
	}
	plans = map[string]BeatPlan{
		"b1": {Units: []PlanUnit{goodUnit("VP-001", "b1.1", "b1.2", "b1.3")}},
		"b2": {Units: []PlanUnit{goodUnit("VP-002", "b2.1")}},
	}
	if cov := ComputeCoverage(in, plans); !cov.Complete() || cov.LinesCovered != 4 {
		t.Errorf("full coverage not seen: %+v", cov)
	}
}

func TestPlanChecks_RiskNoteAndAvoidedWording(t *testing.T) {
	in := planInput(t, func(m map[string]any) {
		threeLineBeat(m)
		m["glossary"] = map[string]any{"terms": []GlossaryTerm{{Term: "Enzyme", Avoid: "enzim; chất xúc tác"}}}
	})
	u := goodUnit("VP-001", "b1.1", "b1.2") // b1.2 shows F02, a hypothesis
	u.OnScreenText = []string{"Chất xúc tác"}
	checks := PlanChecks(in, map[string]BeatPlan{"b1": {Units: []PlanUnit{u}}})
	byID := map[string]MachineCheck{}
	for _, c := range checks {
		byID[c.ID] = c
	}
	if c := byID[CheckPlanRisk]; c.OK || !strings.Contains(c.Detail, "F02") {
		t.Errorf("a hypothesis with no risk note must fail: %+v", c)
	}
	if c := byID[CheckPlanAvoid]; c.OK || !strings.Contains(c.Detail, "chất xúc tác") {
		t.Errorf("an avoided wording on screen must fail: %+v", c)
	}
	u.RiskNotes = []RiskNote{{FactID: "F02", Note: "mới là giả thuyết"}}
	u.OnScreenText = []string{"Enzyme"}
	for _, c := range PlanChecks(in, map[string]BeatPlan{"b1": {Units: []PlanUnit{u}}}) {
		if !c.OK {
			t.Errorf("check %s should pass now: %s", c.ID, c.Detail)
		}
	}
}

func TestSplitUnit_CutsBeforeTheLineAndKeepsRiskNotesWithTheirFacts(t *testing.T) {
	in := planInput(t, threeLineBeat)
	lines := in.Script.Lines["b1"]
	u := goodUnit("VP-001", "b1.1", "b1.2", "b1.3")
	u.RiskNotes = []RiskNote{{FactID: "F02", Note: "giả thuyết"}}
	u.OpenQuestions = []OpenQuestion{{ID: "Q1", Question: "?"}}
	u.Duration = PlannedDuration{PlannedS: 200, ExtraReason: "chậm"}
	u.Status = UnitApproved
	a, b, err := SplitUnit(u, "b1.2", "VP-007", lines)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(a.ScriptRefs, ",") != "b1.1" || strings.Join(b.ScriptRefs, ",") != "b1.2,b1.3" || b.UnitID != "VP-007" {
		t.Errorf("refs/id wrong: %v | %v %s", a.ScriptRefs, b.ScriptRefs, b.UnitID)
	}
	if len(a.RiskNotes) != 0 || len(b.RiskNotes) != 1 {
		t.Errorf("the F02 note belongs with b1.2: a=%v b=%v", a.RiskNotes, b.RiskNotes)
	}
	if len(a.OpenQuestions) != 1 || len(b.OpenQuestions) != 0 {
		t.Errorf("questions stay with the first unit: a=%v b=%v", a.OpenQuestions, b.OpenQuestions)
	}
	for _, x := range []PlanUnit{a, b} {
		if x.Status != UnitPending || x.Duration != (PlannedDuration{}) {
			t.Errorf("a cut unit goes back to review with a fresh duration: %+v", x)
		}
	}
	for _, at := range []string{"b1.1", "b2.1", "zz"} {
		if _, _, err := SplitUnit(u, at, "VP-008", lines); !errors.Is(err, ErrNotSplittable) {
			t.Errorf("cutting at %q must be refused, got %v", at, err)
		}
	}
}

func TestMergeUnits_JoinsNeighboursAndKeepsTheFirstId(t *testing.T) {
	a := goodUnit("VP-001", "b1.1")
	a.OnScreenText = []string{"ong"}
	a.OpenQuestions = []OpenQuestion{{ID: "Q1", Question: "x"}}
	a.Status = UnitApproved
	b := goodUnit("VP-002", "b1.2")
	b.OnScreenText = []string{"ong", "hoa"}
	b.OpenQuestions = []OpenQuestion{{ID: "Q1", Question: "y"}}
	b.RiskNotes = []RiskNote{{FactID: "F02", Note: "n"}}
	got, err := MergeUnits(a, b)
	if err != nil {
		t.Fatal(err)
	}
	if got.UnitID != "VP-001" || strings.Join(got.ScriptRefs, ",") != "b1.1,b1.2" {
		t.Errorf("id/refs wrong: %s %v", got.UnitID, got.ScriptRefs)
	}
	if strings.Join(got.OnScreenText, ",") != "ong,hoa" || len(got.EssentialVisualInfo) != 1 {
		t.Errorf("lists should merge once each: %v %v", got.OnScreenText, got.EssentialVisualInfo)
	}
	if len(got.OpenQuestions) != 2 || got.OpenQuestions[0].ID == got.OpenQuestions[1].ID || len(got.RiskNotes) != 1 {
		t.Errorf("questions need distinct ids and risk notes carry over: %+v %+v", got.OpenQuestions, got.RiskNotes)
	}
	if got.Status != UnitPending {
		t.Errorf("a merged unit goes back to review, got %s", got.Status)
	}
}

func TestComputeTiming_ShowsBothNumbersAndWarnsOnlyOverTheBrief(t *testing.T) {
	in := planInput(t, nil) // b1 lasts 20s in the beat sheet, 2 lines = 80s narration; brief 120s
	u1 := goodUnit("VP-001", "b1.1", "b1.2")
	u2 := goodUnit("VP-002", "b2.1")
	timing := ComputeTiming(in, map[string]BeatPlan{"b1": {Units: []PlanUnit{u1}}, "b2": {Units: []PlanUnit{u2}}})
	b1 := timing.Beats["b1"]
	if b1.NarrationMinS != 80 || b1.PlannedS != 80 || b1.BeatSheetS != 20 || b1.DeltaS != 60 {
		t.Errorf("b1 timing wrong: %+v", b1)
	}
	if timing.PlannedS != 120 || timing.OverBrief {
		t.Errorf("120s planned against a 120s brief must not warn: %+v", timing)
	}
	u2.Duration = PlannedDuration{PlannedS: 41, ExtraReason: "r"}
	timing = ComputeTiming(in, map[string]BeatPlan{"b1": {Units: []PlanUnit{u1}}, "b2": {Units: []PlanUnit{u2}}})
	if !timing.OverBrief {
		t.Errorf("121s against a 120s brief must warn: %+v", timing)
	}
}

func TestBuildPlanDocument_ShapeOrderAndFacts(t *testing.T) {
	in := planInput(t, nil)
	u1 := goodUnit("VP-003", "b1.1", "b1.2")
	u1.Status = UnitApproved
	u2 := goodUnit("VP-001", "b2.1")
	u2.Duration = PlannedDuration{PlannedS: 50, ExtraReason: "kịp nhìn"}
	u2.Status = UnitApproved
	plans := map[string]BeatPlan{"b1": {Units: []PlanUnit{u1}}, "b2": {Units: []PlanUnit{u2}}}
	review := ReviewState{Checks: map[string]CheckDecision{}, Findings: []Finding{{ID: "f1", Group: "flow", Text: "mạch", State: FindingAccepted, Why: "ok"}}}
	doc, err := BuildPlanDocument("p", 2, time.Unix(0, 0).UTC(), in, plans, review)
	if err != nil {
		t.Fatal(err)
	}
	if doc.DocumentID != "visual-plan-v2" || doc.VisualConceptID != "visual-concept-v1" || doc.Status != "locked" {
		t.Errorf("ids wrong: %+v", doc)
	}
	if doc.SourceVersions.Script != 2 || doc.SourceVersions.VisualConcept != 1 {
		t.Errorf("source versions wrong: %+v", doc.SourceVersions)
	}
	if len(doc.Units) != 2 || doc.Units[0].UnitID != "VP-003" || doc.Units[0].Order != 1 || doc.Units[1].Order != 2 {
		t.Errorf("units go in story order, numbered from 1: %+v", doc.Units)
	}
	if strings.Join(doc.Units[0].FactRefs, ",") != "F01,F02" || doc.Units[1].Duration.PlannedS != 50 || doc.Units[1].Duration.NarrationMinS != 40 {
		t.Errorf("fact refs / duration wrong: %+v %+v", doc.Units[0].FactRefs, doc.Units[1].Duration)
	}
	if len(doc.Review.Comments) != 1 || doc.Review.Comments[0].Decision != FindingAccepted {
		t.Errorf("review comments wrong: %+v", doc.Review.Comments)
	}
	if _, err := BuildPlanDocument("p", 1, time.Now(), in, map[string]BeatPlan{"b1": {Units: []PlanUnit{u1}}}, review); !errors.Is(err, ErrLockNotReady) {
		t.Errorf("a plan that leaves a line out must not build, got %v", err)
	}
}

func TestDecodeBeatPlan_RefusesUnknownFieldsAndFillsLists(t *testing.T) {
	if _, err := DecodeBeatPlan([]byte(`{"units":[{"unit_id":"VP-001","colour":"red"}]}`)); !errors.Is(err, ErrInvalidEdit) {
		t.Errorf("an unknown field must be refused, got %v", err)
	}
	p, err := DecodeBeatPlan([]byte(`{"units":[{"unit_id":"VP-001","on_screen_text":null}]}`))
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := p.Encode()
	if !strings.Contains(string(raw), `"on_screen_text":[]`) || !strings.Contains(string(raw), `"status":"pending"`) {
		t.Errorf("lists must encode as arrays and status default to pending: %s", raw)
	}
	if empty, err := DecodeBeatPlan(nil); err != nil || len(empty.Units) != 0 {
		t.Errorf("empty content is a beat with no unit: %v %v", empty, err)
	}
}
