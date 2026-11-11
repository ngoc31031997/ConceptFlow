package application

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"testing"
	"time"

	"visual/internal/domain"
)

// fakePlanStore keeps step 9's data in memory.
type fakePlanStore struct {
	units    map[string]domain.ReviewUnit
	seq      int
	concepts []VersionRecord
	versions []PlanVersionRecord
	model    string
}

func newPlanStore(t *testing.T) *fakePlanStore {
	t.Helper()
	doc, _ := json.Marshal(domain.Document{DocumentID: "visual-concept-v1",
		ColorSystem: domain.ColorSystem{Palette: []domain.Swatch{{Token: "ink", Hex: "#ffffff", Role: "chữ", Name: "trắng"}}}})
	return &fakePlanStore{units: map[string]domain.ReviewUnit{}, seq: 1, concepts: []VersionRecord{{Version: 1, ScriptVersion: 2, Document: doc}}}
}

func (s *fakePlanStore) ListUnits(_ context.Context, p, kind string) ([]domain.ReviewUnit, error) {
	var out []domain.ReviewUnit
	for _, u := range s.units {
		if u.ProjectID == p && u.Kind == kind {
			out = append(out, u)
		}
	}
	return out, nil
}
func (s *fakePlanStore) SaveUnit(_ context.Context, u domain.ReviewUnit) error {
	s.units[u.Kind+"/"+u.Key] = u
	return nil
}
func (s *fakePlanStore) ReserveUnitNumbers(_ context.Context, _ string, n int) (int, error) {
	first := s.seq
	s.seq += n
	return first, nil
}
func (s *fakePlanStore) PlanModel(context.Context, string) (string, error) { return s.model, nil }
func (s *fakePlanStore) SetPlanModel(_ context.Context, _ string, m string) error {
	s.model = m
	return nil
}
func (s *fakePlanStore) LatestConceptVersion(context.Context, string) (VersionRecord, error) {
	if len(s.concepts) == 0 {
		return VersionRecord{}, domain.ErrNoVersion
	}
	return s.concepts[len(s.concepts)-1], nil
}
func (s *fakePlanStore) PlanVersions(context.Context, string) ([]PlanVersionRecord, error) {
	var out []PlanVersionRecord
	for _, v := range s.versions {
		v.Document = nil
		out = append(out, v)
	}
	return out, nil
}
func (s *fakePlanStore) PlanVersion(_ context.Context, _ string, n int) (PlanVersionRecord, error) {
	for _, v := range s.versions {
		if v.Version == n {
			return v, nil
		}
	}
	return PlanVersionRecord{}, domain.ErrNoVersion
}
func (s *fakePlanStore) InsertPlanVersion(_ context.Context, v PlanVersionRecord) error {
	for _, have := range s.versions {
		if have.Version == v.Version {
			return domain.ErrVersionExists
		}
	}
	s.versions = append(s.versions, v)
	return nil
}

// planScript is a locked script of 120s with two beats: b1 has three lines of
// three words (F01, F02 a hypothesis, F01) and b2 one line (F03 a misconception).
// Twelve words in 120s is 10s a word, so a line lasts at least 30s.
func planScript(t *testing.T, version int, b2line string) domain.ScriptVersion {
	t.Helper()
	m := map[string]any{
		"brief": domain.Brief{Audience: "phổ thông", Tone: "tò mò", DurationSeconds: 120},
		"facts": []map[string]any{
			{"ID": "F01", "type": "fact", "claim_vi": "A"},
			{"ID": "F02", "type": "hypothesis", "claim_vi": "B"},
			{"ID": "F03", "type": "misconception", "claim_vi": "C"},
		},
		"story":      map[string]domain.StoryField{"question": {Text: "q"}},
		"order":      []string{"b1", "b2"},
		"beat:b1":    map[string]any{"fn": "Hook", "title": "Mở", "dur": 60},
		"beat:b2":    map[string]any{"fn": "Payoff", "title": "Đóng", "dur": 60},
		"script:b1":  map[string]any{"lines": []domain.Line{{ID: "b1.1", Text: "A là thật", Facts: []string{"F01"}}, {ID: "b1.2", Text: "Có thể B", Facts: []string{"F02"}}, {ID: "b1.3", Text: "Rồi tới D", Facts: []string{"F01"}}}},
		"script:b2":  map[string]any{"lines": []domain.Line{{ID: "b2.1", Text: b2line, Facts: []string{"F03"}}}},
		"handoff:b1": domain.Handoff{},
		"handoff:b2": domain.Handoff{},
		"glossary":   map[string]any{"terms": []domain.GlossaryTerm{{Term: "Enzyme", Avoid: "enzim"}}},
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var parts []domain.ScriptVersionPart
	for _, k := range keys {
		parts = append(parts, frozen(t, k, m[k]))
	}
	return domain.ScriptVersion{ProjectID: project, Version: version, Parts: parts}
}

func okUnit(id string, refs ...string) domain.PlanUnit {
	return domain.PlanUnit{
		UnitID: id, ScriptRefs: refs, NarrationSummary: "tóm tắt", CommunicationGoal: "hiểu A",
		EssentialVisualInfo:   []domain.EssentialInfo{{Item: "ong", Priority: domain.PriorityMust}},
		VisualApproach:        domain.VisualApproach{Method: "diagram", Description: "sơ đồ", ConceptRuleRefs: []string{"color_system.palette.ink"}},
		NarrationRelationship: "direct_illustration", Status: domain.UnitPending,
	}
}

type planFixture struct {
	t      *testing.T
	desk   *PlanDesk
	store  *fakePlanStore
	script *fakeAuthoring
}

// newPlanFixture seeds three units over the whole script, each with the hashes
// the beat has now, so nothing is stale.
func newPlanFixture(t *testing.T) *planFixture {
	t.Helper()
	store := newPlanStore(t)
	script := &fakeAuthoring{v: planScript(t, 2, "Không phải C")}
	f := &planFixture{t: t, store: store, script: script, desk: NewPlanDesk(script, store)}
	f.desk.now = func() time.Time { return time.Unix(1_700_000_000, 0) }
	u2 := okUnit("VP-002", "b1.2", "b1.3")
	u2.RiskNotes = []domain.RiskNote{{FactID: "F02", Note: "mới là giả thuyết"}}
	u3 := okUnit("VP-003", "b2.1")
	u3.RiskNotes = []domain.RiskNote{{FactID: "F03", Note: "đừng để người xem tin"}}
	f.seedBeat("b1", okUnit("VP-001", "b1.1"), u2)
	f.seedBeat("b2", u3)
	f.store.seq = 4
	return f
}

func (f *planFixture) input() domain.PlanInput {
	f.t.Helper()
	cv, _ := f.store.LatestConceptVersion(context.Background(), project)
	in, err := domain.NewPlanInput(f.script.v, cv.Version, cv.Document)
	if err != nil {
		f.t.Fatal(err)
	}
	return in
}

func (f *planFixture) seedBeat(key string, units ...domain.PlanUnit) {
	f.t.Helper()
	raw, err := domain.BeatPlan{Units: units}.Encode()
	if err != nil {
		f.t.Fatal(err)
	}
	f.store.units[domain.UnitKindPlan+"/"+key] = domain.ReviewUnit{ProjectID: project, Kind: domain.UnitKindPlan, Key: key,
		Status: domain.UnitPending, Content: raw, CreatorFields: []string{}, InputHashes: f.input().HashesFor(key)}
}

func (f *planFixture) seedReview(r domain.ReviewState) {
	f.t.Helper()
	raw, _ := json.Marshal(r)
	f.store.units[domain.UnitKindPlanReview+"/main"] = domain.ReviewUnit{ProjectID: project, Kind: domain.UnitKindPlanReview, Key: "main",
		Status: domain.UnitPending, Content: raw, CreatorFields: []string{}, InputHashes: map[string]string{}}
}

func (f *planFixture) must(st *PlanState, err error) *PlanState {
	f.t.Helper()
	if err != nil {
		f.t.Fatal(err)
	}
	return st
}

func (f *planFixture) unit(st *PlanState, id string) PlanUnitView {
	f.t.Helper()
	for _, b := range st.Beats {
		for _, u := range b.Units {
			if u.UnitID == id {
				return u
			}
		}
	}
	f.t.Fatalf("unit %s not on the screen", id)
	return PlanUnitView{}
}

func (f *planFixture) approveAll() *PlanState {
	f.t.Helper()
	return f.must(f.desk.ApproveAll(context.Background(), project))
}

func has(list []string, part string) bool {
	for _, s := range list {
		if strings.Contains(s, part) {
			return true
		}
	}
	return false
}

func TestPlanDesk_NeedsTheLockedScriptAndTheLockedConcept(t *testing.T) {
	f := newPlanFixture(t)
	f.script.err = domain.ErrScriptNotLocked
	if _, err := f.desk.State(context.Background(), project); !errors.Is(err, domain.ErrScriptNotLocked) {
		t.Errorf("no locked script must be ErrScriptNotLocked, got %v", err)
	}
	f.script.err = nil
	f.store.concepts = nil
	if _, err := f.desk.State(context.Background(), project); !errors.Is(err, domain.ErrConceptNotLocked) {
		t.Errorf("no locked concept must be ErrConceptNotLocked, got %v", err)
	}
	if _, err := f.desk.State(context.Background(), " "); !errors.Is(err, ErrNoProjectID) {
		t.Errorf("a blank project must be refused, got %v", err)
	}
}

func TestPlanDesk_StateShowsCoverageTimingAndBlockers(t *testing.T) {
	f := newPlanFixture(t)
	st := f.must(f.desk.State(context.Background(), project))
	if !st.Coverage.Complete() || st.Coverage.LinesTotal != 4 || st.UnitsTotal != 3 {
		t.Fatalf("coverage/units wrong: %+v total %d", st.Coverage, st.UnitsTotal)
	}
	u2 := f.unit(st, "VP-002")
	if u2.NarrationMinS != 60 || u2.EffectiveS != 60 || strings.Join(u2.FactRefs, ",") != "F02,F01" || len(u2.Lines) != 2 {
		t.Errorf("unit view wrong: %+v", u2)
	}
	if st.Timing.PlannedS != 120 || st.Timing.OverBrief {
		t.Errorf("timing wrong: %+v", st.Timing)
	}
	b1 := st.Beats[0]
	if b1.Timing == nil || b1.Timing.BeatSheetS != 60 || b1.Timing.DeltaS != 30 || b1.Stale {
		t.Errorf("beat timing wrong: %+v", b1.Timing)
	}
	if st.CanLock || !has(st.Blockers, "3 unit chưa duyệt") || !has(st.Blockers, "Chưa rà soát") {
		t.Errorf("a fresh plan cannot lock: %v", st.Blockers)
	}
}

func TestPlanDesk_BeatsWithNoUnitLeaveLinesUncovered(t *testing.T) {
	f := newPlanFixture(t)
	delete(f.store.units, domain.UnitKindPlan+"/b2")
	st := f.must(f.desk.State(context.Background(), project))
	if st.Coverage.Complete() || strings.Join(st.Coverage.Uncovered, ",") != "b2.1" || st.Beats[1].Written {
		t.Errorf("b2 has no unit, so its line is uncovered: %+v", st.Coverage)
	}
	if !has(st.Blockers, "1 câu lời thoại chưa thuộc unit nào") {
		t.Errorf("blockers: %v", st.Blockers)
	}
}

func TestPlanDesk_ApproveRefusesUnitsThatFailTheirChecks(t *testing.T) {
	f := newPlanFixture(t)
	bad := okUnit("VP-001", "b1.1")
	bad.Duration = domain.PlannedDuration{PlannedS: 10}
	bad.OpenQuestions = []domain.OpenQuestion{{ID: "Q1", Question: "chia đôi?"}}
	u2 := okUnit("VP-002", "b1.2", "b1.3")
	f.seedBeat("b1", bad, u2)
	if _, err := f.desk.Approve(context.Background(), project, "VP-001"); !errors.Is(err, domain.ErrUnitInvalid) ||
		!strings.Contains(err.Error(), "ngắn hơn lời thoại") || !strings.Contains(err.Error(), "Q1") {
		t.Errorf("a short unit with an open question must not be approved, got %v", err)
	}
	if _, err := f.desk.Approve(context.Background(), project, "VP-404"); !errors.Is(err, domain.ErrUnknownPlanUnit) {
		t.Errorf("an unknown unit is ErrUnknownPlanUnit, got %v", err)
	}
	st := f.approveAll()
	if f.unit(st, "VP-001").Status != domain.UnitPending || f.unit(st, "VP-002").Status != domain.UnitApproved {
		t.Errorf("approve-all takes only the units that pass: %v / %v", f.unit(st, "VP-001").Status, f.unit(st, "VP-002").Status)
	}
}

func TestPlanDesk_AnsweringAQuestionClearsTheBlockerAndSendsTheUnitBackToReview(t *testing.T) {
	f := newPlanFixture(t)
	u := okUnit("VP-001", "b1.1")
	u.OpenQuestions = []domain.OpenQuestion{{ID: "Q1", Question: "chia đôi?"}}
	u2 := okUnit("VP-002", "b1.2", "b1.3")
	f.seedBeat("b1", u, u2)
	if _, err := f.desk.SetAnswer(context.Background(), project, "VP-001", "Q9", "x"); !errors.Is(err, domain.ErrUnknownQuestion) {
		t.Errorf("unknown question, got %v", err)
	}
	if _, err := f.desk.SetAnswer(context.Background(), project, "VP-001", "Q1", "  "); !errors.Is(err, domain.ErrInvalidEdit) {
		t.Errorf("a blank answer must be refused, got %v", err)
	}
	st := f.must(f.desk.SetAnswer(context.Background(), project, "VP-001", "Q1", "cắt qua lại"))
	got := f.unit(st, "VP-001")
	if got.OpenQuestions[0].Answer != "cắt qua lại" || len(got.Problems) != 0 {
		t.Errorf("answer not kept or still blocked: %+v", got)
	}
	if !has(got.CreatorFields, "open_questions") {
		t.Errorf("the answer is the Creator's own: %v", got.CreatorFields)
	}
	f.must(f.desk.Approve(context.Background(), project, "VP-001"))
	st = f.must(f.desk.SetAnswer(context.Background(), project, "VP-001", "Q1", "chia đôi"))
	if f.unit(st, "VP-001").Status != domain.UnitPending {
		t.Error("changing an answer must send an approved unit back to review")
	}
}

func TestPlanDesk_EditKeepsTheCreatorsFieldsAndRefusesOthers(t *testing.T) {
	f := newPlanFixture(t)
	ctx := context.Background()
	f.must(f.desk.Approve(ctx, project, "VP-001"))
	st := f.must(f.desk.Edit(ctx, project, "VP-001", map[string]json.RawMessage{"communication_goal": json.RawMessage(`"hiểu B"`)}))
	u := f.unit(st, "VP-001")
	if u.CommunicationGoal != "hiểu B" || u.Status != domain.UnitPending || strings.Join(u.CreatorFields, ",") != "communication_goal" {
		t.Errorf("edit not applied as the Creator's: %+v", u)
	}
	for name, fields := range map[string]map[string]json.RawMessage{
		"id":      {"unit_id": json.RawMessage(`"VP-009"`)},
		"lines":   {"script_refs": json.RawMessage(`["b1.1","b1.2"]`)},
		"status":  {"status": json.RawMessage(`"approved"`)},
		"unknown": {"colour": json.RawMessage(`"red"`)},
		"empty":   {},
	} {
		if _, err := f.desk.Edit(ctx, project, "VP-001", fields); !errors.Is(err, domain.ErrInvalidEdit) {
			t.Errorf("editing %s must be refused, got %v", name, err)
		}
	}
	if _, err := f.desk.Edit(ctx, project, "VP-001", map[string]json.RawMessage{"duration": json.RawMessage(`"abc"`)}); !errors.Is(err, domain.ErrInvalidEdit) {
		t.Errorf("a value of the wrong type must be refused, got %v", err)
	}
	if _, err := f.desk.Edit(ctx, project, "VP-404", map[string]json.RawMessage{"communication_goal": json.RawMessage(`"x"`)}); !errors.Is(err, domain.ErrUnknownPlanUnit) {
		t.Errorf("unknown unit, got %v", err)
	}
}

func TestPlanDesk_SplitAndMergeKeepCoverageAndNeverReuseIds(t *testing.T) {
	f := newPlanFixture(t)
	ctx := context.Background()
	f.must(f.desk.Edit(ctx, project, "VP-002", map[string]json.RawMessage{"communication_goal": json.RawMessage(`"hiểu B"`)}))
	st := f.must(f.desk.Split(ctx, project, "VP-002", "b1.3"))
	if !st.Coverage.Complete() || len(st.Beats[0].Units) != 3 {
		t.Fatalf("split must keep every line covered: %+v", st.Coverage)
	}
	a, b := f.unit(st, "VP-002"), f.unit(st, "VP-004")
	if strings.Join(a.ScriptRefs, ",") != "b1.2" || strings.Join(b.ScriptRefs, ",") != "b1.3" || b.CommunicationGoal != "hiểu B" {
		t.Errorf("split wrong: %v | %v", a.ScriptRefs, b.ScriptRefs)
	}
	if !has(b.CreatorFields, "communication_goal") {
		t.Errorf("the copied text keeps being the Creator's: %v", b.CreatorFields)
	}
	if _, err := f.desk.Split(ctx, project, "VP-002", "b1.2"); !errors.Is(err, domain.ErrNotSplittable) {
		t.Errorf("cutting at the first line must be refused, got %v", err)
	}
	if _, err := f.desk.Merge(ctx, project, "VP-001", "VP-004"); !errors.Is(err, domain.ErrNotMergeable) {
		t.Errorf("units that are not neighbours must not merge, got %v", err)
	}
	st = f.must(f.desk.Merge(ctx, project, "VP-002", "VP-004"))
	if len(st.Beats[0].Units) != 2 || strings.Join(f.unit(st, "VP-002").ScriptRefs, ",") != "b1.2,b1.3" || !st.Coverage.Complete() {
		t.Errorf("merge wrong: %+v", st.Beats[0].Units)
	}
	st = f.must(f.desk.Split(ctx, project, "VP-002", "b1.3"))
	if f.unit(st, "VP-005").UnitID == "" {
		t.Error("an id given up by a merge must not be handed out again")
	}
}

func TestPlanDesk_ChangedScriptStalesOnlyItsBeatAndBlocksApproval(t *testing.T) {
	f := newPlanFixture(t)
	ctx := context.Background()
	f.approveAll()
	f.script.v = planScript(t, 3, "Khác hẳn ở đây")
	st := f.must(f.desk.State(ctx, project))
	if st.Beats[0].Stale || !st.Beats[1].Stale || len(st.Beats[1].StaleSources) != 1 {
		t.Fatalf("only b2 is stale: b1=%v b2=%v %v", st.Beats[0].Stale, st.Beats[1].Stale, st.Beats[1].StaleSources)
	}
	f.must(f.desk.Unapprove(ctx, project, "VP-003"))
	if _, err := f.desk.Approve(ctx, project, "VP-003"); !errors.Is(err, domain.ErrBeatStale) {
		t.Errorf("a stale beat must not be approved, got %v", err)
	}
	st = f.approveAll()
	if f.unit(st, "VP-003").Status != domain.UnitPending {
		t.Error("approve-all must skip a stale beat")
	}
	if !has(st.Blockers, "beat 2") {
		t.Errorf("the blocker should name the stale beat: %v", st.Blockers)
	}
}

func TestPlanDesk_LockNeedsEverythingAndVersionsAreImmutable(t *testing.T) {
	f := newPlanFixture(t)
	ctx := context.Background()
	f.approveAll()
	if _, err := f.desk.Lock(ctx, project, ""); !errors.Is(err, domain.ErrLockNotReady) || !strings.Contains(err.Error(), "Chưa rà soát") {
		t.Fatalf("an unreviewed plan must not lock, got %v", err)
	}
	f.seedReview(domain.ReviewState{Rounds: 1, NextSeq: 2, Findings: []domain.Finding{{ID: "f1", Group: "flow", UnitKey: "VP-001", Field: "communication_goal", Text: "mờ", Fix: "hiểu rõ A", State: domain.FindingOpen}}})
	if _, err := f.desk.Lock(ctx, project, ""); !errors.Is(err, domain.ErrLockNotReady) || !strings.Contains(err.Error(), "1 vấn đề rà soát") {
		t.Fatalf("an open finding must block the lock, got %v", err)
	}
	st := f.must(f.desk.ApplyFinding(ctx, project, "f1"))
	u := f.unit(st, "VP-001")
	if u.CommunicationGoal != "hiểu rõ A" || u.Status != domain.UnitPending || !has(u.CreatorFields, "communication_goal") || st.Review.OpenCount != 0 {
		t.Fatalf("apply must write the fix as the Creator's and send the unit back: %+v", u)
	}
	f.approveAll()

	st = f.must(f.desk.Lock(ctx, project, ""))
	if st.LockedVersion != 1 || st.NextVersion != 2 || st.CanLock || len(st.Draft) != 0 {
		t.Fatalf("lock v1 wrong: locked %d next %d can %v draft %v", st.LockedVersion, st.NextVersion, st.CanLock, st.Draft)
	}
	rec, err := f.desk.Version(ctx, project, 1)
	if err != nil {
		t.Fatal(err)
	}
	var doc domain.PlanDocument
	if err := json.Unmarshal(rec.Document, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.DocumentID != "visual-plan-v1" || len(doc.Units) != 3 || doc.Coverage.LinesCovered != 4 || rec.Note != "Chốt lần đầu" || doc.Review.Comments[0].Decision != domain.FindingApplied {
		t.Errorf("locked document wrong: %+v note %q", doc, rec.Note)
	}
	if _, err := f.desk.Lock(ctx, project, ""); !errors.Is(err, domain.ErrNothingToLock) {
		t.Errorf("locking with no change must be refused, got %v", err)
	}

	f.must(f.desk.Edit(ctx, project, "VP-003", map[string]json.RawMessage{"narration_summary": json.RawMessage(`"khác"`)}))
	st = f.must(f.desk.State(ctx, project))
	if len(st.Draft) != 1 || !strings.HasPrefix(st.Draft[0], "VP-003") {
		t.Errorf("draft should name the one unit that changed: %v", st.Draft)
	}
	f.approveAll()
	st = f.must(f.desk.Lock(ctx, project, "đổi tóm tắt"))
	if st.LockedVersion != 2 || len(st.Versions) != 2 {
		t.Errorf("lock v2 wrong: %+v", st.Versions)
	}
	latest, err := f.desk.LatestDocument(ctx, project)
	if err != nil || !strings.Contains(string(latest), "visual-plan-v2") {
		t.Errorf("latest document should be v2: %v %s", err, latest)
	}
	if _, err := f.desk.Version(ctx, project, 9); !errors.Is(err, domain.ErrNoVersion) {
		t.Errorf("unknown version, got %v", err)
	}
}

func TestPlanDesk_SoftChecksBlockTheLockUntilFixedOrAcceptedWithAReason(t *testing.T) {
	f := newPlanFixture(t)
	ctx := context.Background()
	u := okUnit("VP-003", "b2.1") // F03 is a misconception, and the unit has no risk note
	f.seedBeat("b2", u)
	f.seedReview(domain.ReviewState{Rounds: 1, NextSeq: 1})
	f.approveAll()
	st := f.must(f.desk.State(ctx, project))
	if st.CanLock || !has(st.Blockers, "ghi chú rủi ro") {
		t.Fatalf("a misconception with no risk note must block: %v", st.Blockers)
	}
	if _, err := f.desk.AcceptFinding(ctx, project, "m-plan-avoid", "x"); !errors.Is(err, domain.ErrFindingNotApplicable) {
		t.Errorf("a passing check cannot be accepted, got %v", err)
	}
	if _, err := f.desk.AcceptFinding(ctx, project, "m-nope", "x"); !errors.Is(err, domain.ErrUnknownFinding) {
		t.Errorf("unknown check, got %v", err)
	}
	st = f.must(f.desk.AcceptFinding(ctx, project, domain.CheckPlanRisk, "để người xem tự thấy"))
	if !st.CanLock {
		t.Fatalf("an accepted check must stop blocking: %v", st.Blockers)
	}
	st = f.must(f.desk.Lock(ctx, project, ""))
	rec, _ := f.desk.Version(ctx, project, st.LockedVersion)
	if !strings.Contains(string(rec.Document), "để người xem tự thấy") {
		t.Errorf("the accepted reason must be kept in the locked document: %s", rec.Document)
	}
	f.must(f.desk.Edit(ctx, project, "VP-003", map[string]json.RawMessage{"on_screen_text": json.RawMessage(`["Enzim"]`)}))
	f.approveAll()
	st = f.must(f.desk.State(ctx, project))
	var avoid CheckView
	for _, c := range st.Checks {
		if c.ID == domain.CheckPlanAvoid {
			avoid = c
		}
	}
	if avoid.OK || avoid.Accepted {
		t.Errorf("a new failure must open again, not inherit another check's acceptance: %+v", avoid)
	}
}

func TestPlanDesk_AcceptingAFindingKeepsItsReason(t *testing.T) {
	f := newPlanFixture(t)
	ctx := context.Background()
	f.seedReview(domain.ReviewState{Rounds: 1, NextSeq: 2, Findings: []domain.Finding{{ID: "f1", Group: "flow", UnitKey: "VP-001", Text: "mờ", State: domain.FindingOpen}}})
	if _, err := f.desk.ApplyFinding(ctx, project, "f1"); !errors.Is(err, domain.ErrFindingNotApplicable) {
		t.Errorf("a finding with no fix cannot be applied, got %v", err)
	}
	st := f.must(f.desk.AcceptFinding(ctx, project, "f1", ""))
	if st.Review.Findings[0].State != domain.FindingAccepted || st.Review.Findings[0].Why != "không ghi lý do" || st.Review.OpenCount != 0 {
		t.Errorf("accept wrong: %+v", st.Review.Findings[0])
	}
	if _, err := f.desk.AcceptFinding(ctx, project, "f1", "lại"); !errors.Is(err, domain.ErrFindingNotApplicable) {
		t.Errorf("an accepted finding cannot be accepted twice, got %v", err)
	}
	if _, err := f.desk.AcceptFinding(ctx, project, "f9", "x"); !errors.Is(err, domain.ErrUnknownFinding) {
		t.Errorf("unknown finding, got %v", err)
	}
}
