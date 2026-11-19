package application

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"visual/internal/domain"
)

// fakeSequenceStore keeps step 10's data in memory.
type fakeSequenceStore struct {
	units    map[string]domain.ReviewUnit
	concepts []VersionRecord
	plans    []PlanVersionRecord
	versions []SequenceVersionRecord
	model    string
}

func (s *fakeSequenceStore) SequenceModel(context.Context, string) (string, error) {
	return s.model, nil
}
func (s *fakeSequenceStore) SetSequenceModel(_ context.Context, _ string, m string) error {
	s.model = m
	return nil
}

func (s *fakeSequenceStore) ListUnits(_ context.Context, p, kind string) ([]domain.ReviewUnit, error) {
	var out []domain.ReviewUnit
	for _, u := range s.units {
		if u.ProjectID == p && u.Kind == kind {
			out = append(out, u)
		}
	}
	return out, nil
}
func (s *fakeSequenceStore) SaveUnit(_ context.Context, u domain.ReviewUnit) error {
	s.units[u.Kind+"/"+u.Key] = u
	return nil
}
func (s *fakeSequenceStore) LatestConceptVersion(context.Context, string) (VersionRecord, error) {
	if len(s.concepts) == 0 {
		return VersionRecord{}, domain.ErrNoVersion
	}
	return s.concepts[len(s.concepts)-1], nil
}
func (s *fakeSequenceStore) LatestPlanVersion(context.Context, string) (PlanVersionRecord, error) {
	if len(s.plans) == 0 {
		return PlanVersionRecord{}, domain.ErrNoVersion
	}
	return s.plans[len(s.plans)-1], nil
}
func (s *fakeSequenceStore) SequenceVersions(context.Context, string) ([]SequenceVersionRecord, error) {
	var out []SequenceVersionRecord
	for _, v := range s.versions {
		v.Document = nil
		out = append(out, v)
	}
	return out, nil
}
func (s *fakeSequenceStore) SequenceVersion(_ context.Context, _ string, n int) (SequenceVersionRecord, error) {
	for _, v := range s.versions {
		if v.Version == n {
			return v, nil
		}
	}
	return SequenceVersionRecord{}, domain.ErrNoVersion
}
func (s *fakeSequenceStore) InsertSequenceVersion(_ context.Context, v SequenceVersionRecord) error {
	for _, have := range s.versions {
		if have.Version == v.Version {
			return domain.ErrVersionExists
		}
	}
	s.versions = append(s.versions, v)
	return nil
}

// seqConceptDoc is concept v1 with a scientific concept and an object to anchor to.
func seqConceptDoc(t *testing.T, draw string) []byte {
	t.Helper()
	d := domain.Document{DocumentID: "visual-concept-v1"}
	d.RepresentationRules.ScientificConcepts = []domain.ScientificConcept{{ConceptID: "bee", Term: "Ong", Representation: "tròn vàng"}}
	d.RepresentationRules.Objects = []domain.RepresentedThing{{ConceptID: "clock", Term: "Đồng hồ", Draw: draw}}
	raw, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// seqPlanDoc is plan v1 of script v2 and concept v1: VP-001 and VP-002 in b1,
// VP-003 in b2.
func seqPlanDoc(t *testing.T, goal string) []byte {
	t.Helper()
	p := domain.PlanDocument{DocumentID: "visual-plan-v1", SourceVersions: domain.PlanSourceVersions{Script: 2, VisualConcept: 1}, Units: []domain.DocumentUnit{
		{UnitID: "VP-001", Order: 1, BeatID: "b1", ScriptRefs: []string{"b1.1"}, CommunicationGoal: goal},
		{UnitID: "VP-002", Order: 2, BeatID: "b1", ScriptRefs: []string{"b1.2", "b1.3"}, CommunicationGoal: "giữa"},
		{UnitID: "VP-003", Order: 3, BeatID: "b2", ScriptRefs: []string{"b2.1"}, CommunicationGoal: "đóng"},
	}}
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

type seqFixture struct {
	t     *testing.T
	ctx   context.Context
	desk  *SequenceDesk
	store *fakeSequenceStore
}

func newSeqFixture(t *testing.T) *seqFixture {
	t.Helper()
	store := &fakeSequenceStore{units: map[string]domain.ReviewUnit{},
		concepts: []VersionRecord{{Version: 1, ScriptVersion: 2, Document: seqConceptDoc(t, "mặt tròn")}},
		plans:    []PlanVersionRecord{{Version: 1, ScriptVersion: 2, ConceptVersion: 1, Document: seqPlanDoc(t, "mở")}}}
	desk := NewSequenceDesk(&fakeAuthoring{v: planScript(t, 2, "Không phải C")}, store)
	desk.now = func() time.Time { return time.Unix(1_700_000_000, 0) }
	return &seqFixture{t: t, ctx: context.Background(), desk: desk, store: store}
}

func (f *seqFixture) must(st *SequenceState, err error) *SequenceState {
	f.t.Helper()
	if err != nil {
		f.t.Fatal(err)
	}
	return st
}

func fields(t *testing.T, m map[string]any) map[string]json.RawMessage {
	t.Helper()
	out := map[string]json.RawMessage{}
	for k, v := range m {
		raw, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		out[k] = raw
	}
	return out
}

func (f *seqFixture) addMotif(name, anchor string) *SequenceState {
	f.t.Helper()
	return f.must(f.desk.AddMotif(f.ctx, project, fields(f.t, map[string]any{"name": name, "description": "lặp lại", "anchor_ref": anchor, "role": "neo"})))
}

// writeItem writes a whole valid item for unitID through the Creator's edit.
func (f *seqFixture) writeItem(unitID string, first bool, motifs ...string) *SequenceState {
	f.t.Helper()
	m := map[string]any{"purpose_in_flow": "nối mạch", "pacing": domain.Pacing{Name: "Đều", Tempo: domain.TempoMedium}, "motif_refs": motifs}
	if first {
		m["transition_from_previous"] = nil
	} else {
		m["transition_from_previous"] = domain.Transition{Type: "Cắt", Description: "cắt thẳng"}
	}
	return f.must(f.desk.Edit(f.ctx, project, unitID, fields(f.t, m)))
}

func item(st *SequenceState, id string) *SequenceUnitView {
	for _, b := range st.Beats {
		for i := range b.Units {
			if b.Units[i].UnitID == id {
				return &b.Units[i]
			}
		}
	}
	return nil
}

const anchorBee, anchorClock = "representation_rules.scientific_concepts.bee", "representation_rules.objects.clock"

func TestSequenceDesk_NeedsLockedConceptAndPlan(t *testing.T) {
	f := newSeqFixture(t)
	f.store.plans = nil
	if _, err := f.desk.State(f.ctx, project); !errors.Is(err, domain.ErrPlanNotLocked) {
		t.Errorf("no locked plan: want ErrPlanNotLocked, got %v", err)
	}
	f.store.concepts = nil
	if _, err := f.desk.State(f.ctx, project); !errors.Is(err, domain.ErrConceptNotLocked) {
		t.Errorf("no locked concept: want ErrConceptNotLocked, got %v", err)
	}
}

func TestSequenceDesk_StateListsEveryPlanUnitUnderItsBeat(t *testing.T) {
	f := newSeqFixture(t)
	st := f.must(f.desk.State(f.ctx, project))
	if len(st.Beats) != 2 || len(st.Beats[0].Units) != 2 || st.Beats[0].Fn != "Hook" || st.Beats[1].Units[0].PreviousUnit != "VP-002" {
		t.Fatalf("beats and units from the plan: %+v", st.Beats)
	}
	if st.Coverage.UnitsTotal != 3 || len(st.Coverage.Missing) != 3 || st.CanLock {
		t.Errorf("nothing written yet: %+v", st.Coverage)
	}
	if !strings.Contains(strings.Join(st.Blockers, " "), "Chưa có motif") {
		t.Errorf("blockers name the missing motifs: %v", st.Blockers)
	}
	if len(st.Motifs.Anchors) != 2 {
		t.Errorf("the anchors of step 8 are offered: %v", st.Motifs.Anchors)
	}
}

func TestSequenceDesk_Motifs(t *testing.T) {
	f := newSeqFixture(t)
	if _, err := f.desk.ApproveMotifs(f.ctx, project); !errors.Is(err, domain.ErrNoMotif) {
		t.Fatalf("no motif to approve: want ErrNoMotif, got %v", err)
	}
	f.addMotif("Ong", anchorBee)
	st := f.addMotif("Đồng hồ", "representation_rules.objects.rocket")
	if len(st.Motifs.Motifs) != 2 || st.Motifs.Motifs[1].MotifID != "MOTIF-02" || len(st.Motifs.Motifs[1].Problems) != 1 {
		t.Fatalf("two motifs, the second with an anchor step 8 lacks: %+v", st.Motifs.Motifs)
	}
	if st.Motifs.Motifs[0].AnchorText == "" || strings.Join(st.Motifs.Motifs[0].CreatorFields, ",") != "anchor_ref,description,name,role" {
		t.Errorf("a motif shows its anchor's text and the Creator's fields: %+v", st.Motifs.Motifs[0])
	}
	if _, err := f.desk.ApproveMotifs(f.ctx, project); !errors.Is(err, domain.ErrUnitInvalid) {
		t.Fatalf("a bad anchor blocks approval, got %v", err)
	}
	if _, err := f.desk.EditMotif(f.ctx, project, "MOTIF-02", fields(t, map[string]any{"motif_id": "X"})); !errors.Is(err, domain.ErrInvalidEdit) {
		t.Errorf("the id is not the Creator's to write, got %v", err)
	}
	f.must(f.desk.EditMotif(f.ctx, project, "MOTIF-02", fields(t, map[string]any{"anchor_ref": anchorClock})))
	st = f.must(f.desk.ApproveMotifs(f.ctx, project))
	if st.Motifs.Status != domain.UnitApproved || st.Motifs.Stale {
		t.Fatalf("motifs approved: %+v", st.Motifs)
	}
	st = f.addMotif("Thêm", anchorBee)
	if st.Motifs.Status != domain.UnitPending || st.Motifs.Motifs[2].MotifID != "MOTIF-03" {
		t.Errorf("adding a motif sends the motifs back to review: %+v", st.Motifs)
	}
	f.must(f.desk.DeleteMotif(f.ctx, project, "MOTIF-03"))
	st = f.addMotif("Sau xoá", anchorBee)
	if st.Motifs.Motifs[2].MotifID != "MOTIF-04" {
		t.Errorf("a deleted motif's number is not used again: %+v", st.Motifs.Motifs)
	}
	if _, err := f.desk.EditMotif(f.ctx, project, "MOTIF-09", fields(t, map[string]any{"name": "x"})); !errors.Is(err, domain.ErrUnknownMotif) {
		t.Errorf("an unknown motif: want ErrUnknownMotif, got %v", err)
	}
}

func TestSequenceDesk_MotifsGoStaleWhenTheirAnchorsChange(t *testing.T) {
	f := newSeqFixture(t)
	f.addMotif("Ong", anchorBee)
	f.must(f.desk.ApproveMotifs(f.ctx, project))
	f.store.concepts = append(f.store.concepts, VersionRecord{Version: 2, ScriptVersion: 2, Document: seqConceptDoc(t, "mặt vuông")})
	st := f.must(f.desk.State(f.ctx, project))
	if !st.Motifs.Stale || !strings.Contains(strings.Join(st.Blockers, " "), "đặt motif") {
		t.Fatalf("a changed representation rule makes the motifs stale: %+v %v", st.Motifs, st.Blockers)
	}
	st = f.must(f.desk.ApproveMotifs(f.ctx, project))
	if st.Motifs.Stale {
		t.Error("approving again confirms the motifs still fit")
	}
}

func TestSequenceDesk_EditApproveAndDeleteMotifFromItems(t *testing.T) {
	f := newSeqFixture(t)
	f.addMotif("Ong", anchorBee)
	st := f.writeItem("VP-001", true, "MOTIF-01")
	vp1 := item(st, "VP-001")
	if vp1.Item == nil || vp1.Item.Status != domain.UnitPending || len(vp1.Problems) != 0 {
		t.Fatalf("the Creator writes a whole item for a unit that had none: %+v", vp1)
	}
	if strings.Join(vp1.CreatorFields, ",") != "motif_refs,pacing,purpose_in_flow,transition_from_previous" {
		t.Errorf("every field written is the Creator's: %v", vp1.CreatorFields)
	}
	st = f.must(f.desk.Edit(f.ctx, project, "VP-002", fields(t, map[string]any{"purpose_in_flow": "giữa"})))
	if p := item(st, "VP-002").Problems; len(p) == 0 {
		t.Fatal("an item with only a purpose has problems")
	}
	if _, err := f.desk.Approve(f.ctx, project, "VP-002"); !errors.Is(err, domain.ErrUnitInvalid) {
		t.Errorf("an item with problems cannot be approved, got %v", err)
	}
	if _, err := f.desk.Approve(f.ctx, project, "VP-003"); !errors.Is(err, domain.ErrUnitNotWritten) {
		t.Errorf("a unit with no item cannot be approved, got %v", err)
	}
	if _, err := f.desk.Edit(f.ctx, project, "VP-001", fields(t, map[string]any{"status": "approved"})); !errors.Is(err, domain.ErrInvalidEdit) {
		t.Errorf("the status is not the Creator's to write, got %v", err)
	}
	if _, err := f.desk.Edit(f.ctx, project, "VP-404", fields(t, map[string]any{"purpose_in_flow": "x"})); !errors.Is(err, domain.ErrUnknownPlanUnit) {
		t.Errorf("a unit the plan lacks: want ErrUnknownPlanUnit, got %v", err)
	}
	st = f.must(f.desk.Approve(f.ctx, project, "VP-001"))
	if item(st, "VP-001").Item.Status != domain.UnitApproved {
		t.Fatal("VP-001 approved")
	}

	f.writeItem("VP-002", false, "MOTIF-01")
	vp2 := f.must(f.desk.Edit(f.ctx, project, "VP-002", fields(t, map[string]any{"transition_from_previous": domain.Transition{Type: "Match cut", ViaMotif: "MOTIF-01"}})))
	if !item(vp2, "VP-002").Item.HasMotif("MOTIF-01") {
		t.Fatal("VP-002 names MOTIF-01")
	}
	st = f.must(f.desk.DeleteMotif(f.ctx, project, "MOTIF-01"))
	for _, id := range []string{"VP-001", "VP-002"} {
		it := item(st, id).Item
		if it.HasMotif("MOTIF-01") || it.Status != domain.UnitPending {
			t.Errorf("%s loses the deleted motif and goes back to review: %+v", id, it)
		}
	}
	if item(st, "VP-002").Item.TransitionFromPrevious.ViaMotif != "" {
		t.Error("the transition no longer rides on the deleted motif")
	}
}

func TestSequenceDesk_StaleBeatBlocksApprovalUntilConfirmed(t *testing.T) {
	f := newSeqFixture(t)
	f.addMotif("Ong", anchorBee)
	f.writeItem("VP-001", true)
	f.writeItem("VP-002", false)
	f.store.plans = append(f.store.plans, PlanVersionRecord{Version: 2, ScriptVersion: 2, ConceptVersion: 1, Document: seqPlanDoc(t, "mở khác")})
	st := f.must(f.desk.State(f.ctx, project))
	if !st.Beats[0].Stale || st.Beats[1].Stale {
		t.Fatalf("only b1's unit changed: %+v / %+v", st.Beats[0].StaleSources, st.Beats[1].StaleSources)
	}
	if _, err := f.desk.Approve(f.ctx, project, "VP-001"); !errors.Is(err, domain.ErrBeatStale) {
		t.Fatalf("a stale beat: want ErrBeatStale, got %v", err)
	}
	st = f.must(f.desk.ApproveAll(f.ctx, project))
	if item(st, "VP-001").Item.Status == domain.UnitApproved {
		t.Error("approve all skips a stale beat")
	}
	st = f.must(f.desk.ConfirmBeat(f.ctx, project, "b1"))
	if st.Beats[0].Stale {
		t.Fatal("confirming takes the inputs' hashes of now")
	}
	if _, err := f.desk.ConfirmBeat(f.ctx, project, "b2"); !errors.Is(err, domain.ErrUnitNotWritten) {
		t.Errorf("a beat with no item has nothing to confirm, got %v", err)
	}
	f.must(f.desk.Approve(f.ctx, project, "VP-001"))
}

// readyToLock writes, approves and reviews a whole sequence over two motifs.
func (f *seqFixture) readyToLock() {
	f.t.Helper()
	f.addMotif("Ong", anchorBee)
	f.addMotif("Đồng hồ", anchorClock)
	f.must(f.desk.ApproveMotifs(f.ctx, project))
	f.writeItem("VP-001", true, "MOTIF-01", "MOTIF-02")
	f.writeItem("VP-002", false, "MOTIF-01")
	f.writeItem("VP-003", false, "MOTIF-02")
	f.must(f.desk.ApproveAll(f.ctx, project))
	raw, _ := json.Marshal(domain.ReviewState{Rounds: 1, NextSeq: 1, Notes: []string{"liền mạch"}, Checks: map[string]domain.CheckDecision{}})
	f.store.units[domain.UnitKindSequenceReview+"/main"] = domain.ReviewUnit{ProjectID: project, Kind: domain.UnitKindSequenceReview,
		Key: "main", Status: domain.UnitPending, Content: raw, CreatorFields: []string{}, InputHashes: map[string]string{}}
}

func TestSequenceDesk_LockThenNothingNewThenOutdated(t *testing.T) {
	f := newSeqFixture(t)
	f.readyToLock()
	st := f.must(f.desk.State(f.ctx, project))
	if !st.CanLock {
		t.Fatalf("ready to lock: %v", st.Blockers)
	}
	st = f.must(f.desk.Lock(f.ctx, project, ""))
	if st.LockedVersion != 1 || st.NextVersion != 2 || f.store.versions[0].Note != "Chốt lần đầu" || f.store.versions[0].PlanVersion != 1 {
		t.Fatalf("v1 locked: %+v %+v", st, f.store.versions)
	}
	if raw, _ := json.Marshal(st); !strings.Contains(string(raw), `"draft":[]`) {
		t.Errorf("a lock with nothing new sends an empty draft list, not null: %s", raw)
	}
	var doc domain.SequenceDocument
	if err := json.Unmarshal(f.store.versions[0].Document, &doc); err != nil || len(doc.Sequence) != 3 || len(doc.Motifs) != 2 {
		t.Fatalf("the locked document: %+v %v", doc, err)
	}
	if _, err := f.desk.Lock(f.ctx, project, ""); !errors.Is(err, domain.ErrNothingToLock) {
		t.Errorf("nothing changed: want ErrNothingToLock, got %v", err)
	}
	latest, err := f.desk.LatestDocument(f.ctx, project)
	if err != nil || !strings.Contains(string(latest), "visual-sequence-v1") {
		t.Errorf("a later step reads v1: %s %v", latest, err)
	}

	f.must(f.desk.Edit(f.ctx, project, "VP-003", fields(t, map[string]any{"purpose_in_flow": "đóng lại"})))
	st = f.must(f.desk.Approve(f.ctx, project, "VP-003"))
	if strings.Join(st.Draft, ",") != "VP-003 · đổi" {
		t.Fatalf("the draft names the changed item: %v", st.Draft)
	}
	f.store.concepts = append(f.store.concepts, VersionRecord{Version: 2, ScriptVersion: 2, Document: seqConceptDoc(t, "mặt tròn")})
	if _, err := f.desk.Lock(f.ctx, project, ""); !errors.Is(err, domain.ErrPlanOutdated) {
		t.Errorf("a plan of concept v1 against concept v2: want ErrPlanOutdated, got %v", err)
	}
}

func TestSequenceDesk_FindingsAndChecks(t *testing.T) {
	f := newSeqFixture(t)
	f.readyToLock()
	f.must(f.desk.Edit(f.ctx, project, "VP-002", fields(t, map[string]any{"transition_from_previous": domain.Transition{Type: "Match cut", ViaMotif: "MOTIF-01"}, "motif_refs": []string{"MOTIF-01"}})))
	f.must(f.desk.Edit(f.ctx, project, "VP-001", fields(t, map[string]any{"motif_refs": []string{"MOTIF-02"}})))
	st := f.must(f.desk.ApproveAll(f.ctx, project))
	if st.Checks[0].OK || st.CanLock {
		t.Fatalf("VP-002 rides on a motif VP-001 lost: %+v", st.Checks)
	}
	st = f.must(f.desk.AcceptFinding(f.ctx, project, domain.CheckSeqVia, "cố ý"))
	if !st.Checks[0].Accepted {
		t.Fatalf("the check accepted with a reason: %+v", st.Checks[0])
	}
	if _, err := f.desk.AcceptFinding(f.ctx, project, "m-unknown", ""); !errors.Is(err, domain.ErrUnknownFinding) {
		t.Errorf("an unknown check, got %v", err)
	}

	raw, _ := json.Marshal(domain.ReviewState{Rounds: 1, NextSeq: 3, Checks: map[string]domain.CheckDecision{}, Findings: []domain.Finding{
		{ID: "f1", Group: "clarity", UnitKey: "VP-003", Field: "purpose_in_flow", Text: "mơ hồ", Fix: "khép lại câu hỏi mở đầu", State: domain.FindingOpen},
		{ID: "f2", Group: "consistency", UnitKey: "MOTIF-02", Field: "description", Text: "lệch", Fix: "kim chậm dần", State: domain.FindingOpen},
	}})
	row := f.store.units[domain.UnitKindSequenceReview+"/main"]
	row.Content = raw
	f.store.units[domain.UnitKindSequenceReview+"/main"] = row
	st = f.must(f.desk.ApplyFinding(f.ctx, project, "f1"))
	vp3 := item(st, "VP-003")
	if vp3.Item.PurposeInFlow != "khép lại câu hỏi mở đầu" || vp3.Item.Status != domain.UnitPending || st.Review.Findings[0].State != domain.FindingApplied {
		t.Fatalf("the fix is written into the item and the item reopens: %+v %+v", vp3.Item, st.Review.Findings[0])
	}
	st = f.must(f.desk.ApplyFinding(f.ctx, project, "f2"))
	if st.Motifs.Motifs[1].Description != "kim chậm dần" || st.Motifs.Status != domain.UnitPending {
		t.Fatalf("the fix is written into the motif and the motifs reopen: %+v", st.Motifs)
	}
	if _, err := f.desk.ApplyFinding(f.ctx, project, "f1"); !errors.Is(err, domain.ErrFindingNotApplicable) {
		t.Errorf("an applied finding cannot apply again, got %v", err)
	}
}
