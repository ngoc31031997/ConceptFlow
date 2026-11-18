package domain

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

// seqConcept is a locked visual concept with one scientific concept and one
// object to anchor motifs to, and b1 set slow.
func seqConcept(t *testing.T, edit func(*Document)) []byte {
	t.Helper()
	d := Document{DocumentID: "visual-concept-v1"}
	d.RepresentationRules.ScientificConcepts = []ScientificConcept{{ConceptID: "black_hole", Term: "Hố đen", Representation: "tròn đen có vành sáng"}}
	d.RepresentationRules.Objects = []RepresentedThing{{ConceptID: "clock", Term: "Đồng hồ", Draw: "mặt tròn"}}
	d.MotionLanguage.Pace = []BeatPace{{BeatKeys: []string{"b1"}, Pace: "slow"}}
	if edit != nil {
		edit(&d)
	}
	raw, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// seqPlan is a locked plan of script v2 and concept v1: VP-001 in b1, VP-002
// and VP-003 in b2.
func seqPlan(t *testing.T, edit func(*PlanDocument)) []byte {
	t.Helper()
	p := PlanDocument{DocumentID: "visual-plan-v1", SourceVersions: PlanSourceVersions{Script: 2, VisualConcept: 1}, Units: []DocumentUnit{
		{UnitID: "VP-001", Order: 1, BeatID: "b1", ScriptRefs: []string{"b1.1", "b1.2"}, CommunicationGoal: "mở"},
		{UnitID: "VP-002", Order: 2, BeatID: "b2", ScriptRefs: []string{"b2.1"}, CommunicationGoal: "so sánh"},
		{UnitID: "VP-003", Order: 3, BeatID: "b2", ScriptRefs: []string{"b2.1"}, CommunicationGoal: "chốt"},
	}}
	if edit != nil {
		edit(&p)
	}
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func seqInput(t *testing.T, plan func(*PlanDocument), concept func(*Document)) SequenceInput {
	t.Helper()
	in, err := NewSequenceInput(scriptVersion(t, nil), 1, seqConcept(t, concept), 1, seqPlan(t, plan))
	if err != nil {
		t.Fatal(err)
	}
	return in
}

func okItem(id string, first bool, motifs ...string) SeqItem {
	it := SeqItem{UnitID: id, PurposeInFlow: "nối mạch", Pacing: Pacing{Name: "Đối chiếu", Tempo: TempoMedium}, MotifRefs: motifs, Status: UnitPending}
	if !first {
		it.TransitionFromPrevious = &Transition{Type: "Match cut", Description: "giữ hố đen"}
	}
	return it
}

func testMotifs() MotifSet {
	return MotifSet{NextSeq: 3, Motifs: []Motif{
		{MotifID: "MOTIF-01", Name: "Hố đen", Description: "neo", AnchorRef: "representation_rules.scientific_concepts.black_hole", Role: "neo"},
		{MotifID: "MOTIF-02", Name: "Đồng hồ", Description: "chậm dần", AnchorRef: "representation_rules.objects.clock", Role: "gieo và gặt"},
	}}
}

func TestNewSequenceInput_GroupsPlanUnitsUnderTheirBeats(t *testing.T) {
	in := seqInput(t, nil, nil)
	if len(in.Beats) != 2 || in.Beats[0].Key != "b1" || len(in.Beats[1].Units) != 2 {
		t.Fatalf("two beats with 1 and 2 units in plan order, got %+v", in.Beats)
	}
	if in.Beats[0].Fn != "Hook" || in.Beats[0].Pace != "slow" || in.Beats[1].Pace != "" {
		t.Errorf("beats carry the script's card and step 8's pace: %+v", in.Beats)
	}
	if len(in.Outdated) != 0 {
		t.Errorf("a plan of the latest script and concept is not outdated: %v", in.Outdated)
	}
	if _, ok := in.AnchorRefs["representation_rules.objects.clock"]; !ok || len(in.AnchorRefs) != 2 {
		t.Errorf("only representation rules are anchors: %v", in.AnchorRefs)
	}
	if beat, first, ok := in.LocateUnit("VP-001"); !ok || beat != "b1" || !first {
		t.Errorf("VP-001 opens the video in b1: %s %v %v", beat, first, ok)
	}
	if in.PreviousUnit("VP-002") != "VP-001" || in.PreviousUnit("VP-001") != "" {
		t.Error("the unit before VP-002 is VP-001, and none before VP-001")
	}
}

func TestNewSequenceInput_PlanOfAnOlderConceptIsOutdated(t *testing.T) {
	in, err := NewSequenceInput(scriptVersion(t, nil), 2, seqConcept(t, nil), 1, seqPlan(t, nil))
	if err != nil {
		t.Fatal(err)
	}
	if len(in.Outdated) != 1 || !strings.Contains(in.Outdated[0], "ngôn ngữ hình ảnh v1") {
		t.Fatalf("a plan of concept v1 against concept v2 is outdated: %v", in.Outdated)
	}
}

func TestSequenceInput_BeatStaleOnlyOnWhatChanged(t *testing.T) {
	in := seqInput(t, nil, nil)
	motifs := testMotifs()
	stored := map[string]map[string]string{"b1": in.BeatHashes("b1", motifs), "b2": in.BeatHashes("b2", motifs)}

	changedB1 := seqInput(t, func(p *PlanDocument) { p.Units[0].CommunicationGoal = "khác" }, nil)
	if got := changedB1.BeatStaleSources("b1", stored["b1"], motifs); len(got) != 1 || got[0] != SeqSourcePlan {
		t.Errorf("b1's own unit changed: want [plan], got %v", got)
	}
	if got := changedB1.BeatStaleSources("b2", stored["b2"], motifs); len(got) != 1 || got[0] != SeqSourcePrev {
		t.Errorf("b2's first transition comes from VP-001: want [prev], got %v", got)
	}
	reordered := seqInput(t, func(p *PlanDocument) {
		for i := range p.Units {
			p.Units[i].Order += 5
		}
	}, nil)
	if got := reordered.BeatStaleSources("b2", stored["b2"], motifs); len(got) != 0 {
		t.Errorf("units only moving in number do not make a beat stale, got %v", got)
	}
	motifs.Motifs[0].Description = "neo khác"
	if got := in.BeatStaleSources("b1", stored["b1"], motifs); len(got) != 1 || got[0] != SeqSourceMotifs {
		t.Errorf("a motif changed: want [motifs], got %v", got)
	}
	hinted := testMotifs()
	hinted.Motifs[0].PlannedUnits = []string{"VP-003"}
	if got := in.BeatStaleSources("b1", stored["b1"], hinted); len(got) != 0 {
		t.Errorf("the AI's hint of where a motif goes is not a source, got %v", got)
	}
	if got := in.BeatStaleSources("b1", nil, testMotifs()); len(got) != len(seqSourceOrder) {
		t.Errorf("a beat stored without hashes is stale on every source, got %v", got)
	}
}

func TestSequenceInput_MotifsStaleWhenRepresentationRulesChange(t *testing.T) {
	stored := seqInput(t, nil, nil).MotifHashes()
	in := seqInput(t, nil, func(d *Document) { d.RepresentationRules.Objects[0].Draw = "mặt vuông" })
	if got := in.MotifStaleSources(stored); len(got) != 1 || got[0] != MotifSourceRep {
		t.Fatalf("want [rep], got %v", got)
	}
}

func TestCheckSeqItem(t *testing.T) {
	motifs := testMotifs().IDs()
	if p := CheckSeqItem(okItem("VP-001", true, "MOTIF-01"), true, motifs); len(p) != 0 {
		t.Fatalf("a valid first item passes: %v", p)
	}
	cases := []struct {
		name  string
		first bool
		edit  func(*SeqItem)
		want  string
	}{
		{"no purpose", false, func(it *SeqItem) { it.PurposeInFlow = " " }, "vai trò"},
		{"no pacing name", false, func(it *SeqItem) { it.Pacing.Name = "" }, "tên nhịp"},
		{"tempo outside the list", false, func(it *SeqItem) { it.Pacing.Tempo = "very_fast" }, "tốc độ"},
		{"first unit with a transition", true, func(it *SeqItem) { it.TransitionFromPrevious = &Transition{Type: "cut"} }, "đầu video"},
		{"later unit without one", false, func(it *SeqItem) { it.TransitionFromPrevious = nil }, "Thiếu chuyển cảnh"},
		{"transition without a type", false, func(it *SeqItem) { it.TransitionFromPrevious.Type = "" }, "kiểu chuyển cảnh"},
		{"unknown motif", false, func(it *SeqItem) { it.MotifRefs = []string{"MOTIF-09"} }, "MOTIF-09"},
		{"motif twice", false, func(it *SeqItem) { it.MotifRefs = []string{"MOTIF-01", "MOTIF-01"} }, "hai lần"},
		{"via a motif the item lacks", false, func(it *SeqItem) { it.TransitionFromPrevious.ViaMotif = "MOTIF-02" }, "Motif nối"},
	}
	for _, c := range cases {
		it := okItem("VP-002", c.first, "MOTIF-01")
		if c.first {
			it.TransitionFromPrevious = nil
		}
		c.edit(&it)
		p := CheckSeqItem(it, c.first, motifs)
		if !strings.Contains(strings.Join(p, "|"), c.want) {
			t.Errorf("%s: want a problem about %q, got %v", c.name, c.want, p)
		}
	}
}

func TestCheckMotif(t *testing.T) {
	in := seqInput(t, nil, nil)
	m := testMotifs().Motifs[0]
	if p := CheckMotif(m, in.AnchorRefs); len(p) != 0 {
		t.Fatalf("a motif anchored to a locked rule passes: %v", p)
	}
	m.AnchorRef = "representation_rules.objects.rocket"
	if p := CheckMotif(m, in.AnchorRefs); len(p) != 1 || !strings.Contains(p[0], "rocket") {
		t.Errorf("an anchor not in the concept fails: %v", p)
	}
	m.AnchorRef, m.MotifID = "", "BH"
	if p := CheckMotif(m, in.AnchorRefs); len(p) != 2 {
		t.Errorf("no anchor and a bad id are two problems: %v", p)
	}
}

func TestSequenceCoverageAndChecks(t *testing.T) {
	in := seqInput(t, nil, nil)
	motifs := testMotifs()
	b2 := okItem("VP-002", false, "MOTIF-01")
	b2.TransitionFromPrevious.ViaMotif = "MOTIF-01"
	beats := map[string]BeatSequence{
		"b1": {Items: []SeqItem{okItem("VP-001", true, "MOTIF-02")}},
		"b2": {Items: []SeqItem{b2, okItem("VP-099", false)}},
	}
	cov := ComputeSequenceCoverage(in, beats)
	if cov.UnitsSequenced != 2 || len(cov.Missing) != 1 || cov.Missing[0] != "VP-003" || len(cov.Extra) != 1 || cov.Complete() {
		t.Fatalf("VP-003 missing and VP-099 extra: %+v", cov)
	}
	checks := SequenceChecks(in, beats, motifs)
	if checks[0].ID != CheckSeqVia || checks[0].OK || !strings.Contains(checks[0].Detail, "VP-002") {
		t.Errorf("VP-002 rides on MOTIF-01, which VP-001 lacks: %+v", checks[0])
	}
	if checks[1].ID != CheckMotifCount || checks[1].OK || !strings.Contains(checks[1].Detail, "MOTIF-01 xuất hiện ở 1") {
		t.Errorf("each motif in one unit only: %+v", checks[1])
	}
	seen := MotifAppearances(in, beats, motifs)
	if len(seen["MOTIF-01"]) != 1 || seen["MOTIF-01"][0] != "VP-002" {
		t.Errorf("an extra item does not count as an appearance: %v", seen)
	}
}

func TestBuildSequenceDocument(t *testing.T) {
	in := seqInput(t, nil, nil)
	motifs := testMotifs()
	beats := map[string]BeatSequence{
		"b1": {Items: []SeqItem{okItem("VP-001", true, "MOTIF-01")}},
		"b2": {Items: []SeqItem{okItem("VP-002", false, "MOTIF-01")}},
	}
	at := time.Unix(1_700_000_000, 0).UTC()
	if _, err := BuildSequenceDocument("p", 1, at, in, beats, motifs, ReviewState{}); !errors.Is(err, ErrLockNotReady) {
		t.Fatalf("a unit without an item cannot lock, got %v", err)
	}
	beats["b2"] = BeatSequence{Items: []SeqItem{okItem("VP-002", false, "MOTIF-01"), okItem("VP-003", false)}}
	doc, err := BuildSequenceDocument("p", 2, at, in, beats, motifs, ReviewState{Notes: []string{"liền mạch"}})
	if err != nil {
		t.Fatal(err)
	}
	if doc.DocumentID != "visual-sequence-v2" || doc.VisualPlanID != "visual-plan-v1" || doc.SourceVersions != (SequenceSourceVersions{Script: 2, VisualConcept: 1, VisualPlan: 1}) {
		t.Errorf("ids and sources: %+v", doc)
	}
	if len(doc.Sequence) != 3 || doc.Sequence[2].SequenceIndex != 3 || doc.Sequence[1].BeatID != "b2" || doc.Sequence[0].TransitionFromPrevious != nil {
		t.Errorf("items in plan order, numbered by place: %+v", doc.Sequence)
	}
	if len(doc.Motifs) != 2 || strings.Join(doc.Motifs[0].UnitRefs, ",") != "VP-001,VP-002" || len(doc.Motifs[1].UnitRefs) != 0 {
		t.Errorf("motifs carry where they appear: %+v", doc.Motifs)
	}
	if len(doc.SequenceNotes) != 1 {
		t.Errorf("the last review's notes go into the document: %v", doc.SequenceNotes)
	}
}

func TestSeqItemAndMotifCodec(t *testing.T) {
	if _, err := DecodeSeqItem([]byte(`{"unit_id":"VP-001","colour":"x"}`)); !errors.Is(err, ErrInvalidEdit) {
		t.Errorf("an unknown field is refused, got %v", err)
	}
	s, err := DecodeBeatSequence([]byte(`{}`))
	if err != nil || len(s.Items) != 0 {
		t.Fatalf("empty content is a beat with no item: %+v %v", s, err)
	}
	set, err := DecodeMotifSet([]byte(`{"motifs":[{"motif_id":"MOTIF-04","name":"a","description":"","anchor_ref":"","role":"","planned_units":null}],"next_seq":2}`))
	if err != nil || set.NextSeq != 5 || set.Motifs[0].PlannedUnits == nil {
		t.Fatalf("the next number is past every motif kept: %+v %v", set, err)
	}
	it := okItem("VP-002", false, "MOTIF-01", "MOTIF-02")
	it.TransitionFromPrevious.ViaMotif = "MOTIF-01"
	out, changed := it.WithoutMotif("MOTIF-01")
	if !changed || out.HasMotif("MOTIF-01") || out.TransitionFromPrevious.ViaMotif != "" || it.TransitionFromPrevious.ViaMotif != "MOTIF-01" {
		t.Errorf("removing a motif clears it from the item and its transition without touching the original: %+v / %+v", out, it)
	}
	if _, changed := out.WithoutMotif("MOTIF-07"); changed {
		t.Error("removing a motif the item lacks changes nothing")
	}
	if MotifID(3) != "MOTIF-03" || MotifSeq("MOTIF-12") != 12 || MotifSeq("M-1") != 0 {
		t.Error("motif ids are MOTIF- and two digits")
	}
}
