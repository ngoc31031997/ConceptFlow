package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"authoring/internal/domain"
)

type memVersions struct{ all []domain.ScriptVersion }

func (m *memVersions) SaveScriptVersion(_ context.Context, v domain.ScriptVersion) error {
	for _, e := range m.all {
		if e.Version == v.Version {
			return errors.New("duplicate version")
		}
	}
	m.all = append(m.all, v)
	return nil
}
func (m *memVersions) ListScriptVersions(context.Context, string) ([]domain.ScriptVersion, error) {
	return append([]domain.ScriptVersion{}, m.all...), nil
}
func (m *memVersions) LatestScriptVersion(context.Context, string) (domain.ScriptVersion, error) {
	if len(m.all) == 0 {
		return domain.ScriptVersion{}, domain.ErrNoLockedVersion
	}
	return m.all[len(m.all)-1], nil
}

const (
	reviewMarker  = "BIÊN TẬP VIÊN rà soát"
	handoffMarker = "ĐẠO DIỄN bàn giao"
)

type lockFixture struct {
	*scriptFixture
	versions *memVersions
	uc2      *ScriptLockDeskUseCase
}

// newLockFixture opens the step: three beats, each with approved dialogue of two lines.
func newLockFixture(t *testing.T) *lockFixture {
	t.Helper()
	sf := newScriptFixture(t)
	// Enough 20-word lines to land in the middle of the 4-5 minute target; each
	// opens with a word of its own so no join is flagged.
	perBeat := int(270*domain.ProfileFor(domain.LanguageVietnamese).WordsPerMinute/60/20/3) + 1
	n := 0
	for _, key := range []string{"b1", "b2", "b3"} {
		s := domain.ScriptBeat{Lines: []domain.ScriptLine{}, NextSeq: perBeat + 1}
		for i := 1; i <= perBeat; i++ {
			n++
			opener := string(rune('a'+n%26)) + string(rune('a'+(n/26)%26)) + "mo"
			text := opener + strings.Repeat(" từ", 19) + "."
			facts := []string{}
			if i == 1 {
				facts = []string{"F01"}
			}
			s.Lines = append(s.Lines, domain.ScriptLine{ID: fmt.Sprintf("%s.%d", key, i), Text: text, Facts: facts})
		}
		sf.beat.save(t, domain.UnitKindScriptBeat, key, domain.UnitApproved, s)
	}
	project := &briefProject{project: domain.Project{ContentLanguage: domain.LanguageVietnamese, VideoOutputMode: domain.ModeLongOnly, VoiceID: "v"}, status: domain.StatusDraft}
	v := &memVersions{}
	uc := NewScriptLockDeskUseCase(sf.beat.store, project, project, briefPrompts{}, sf.llm, nil, 4000, sf.beat.uc, sf.uc, v)
	return &lockFixture{scriptFixture: sf, versions: v, uc2: uc}
}

const handoffReply = "```json\n{\"must\":[\"Sung là chùm hoa (F01)\"],\"terms\":[],\"exact\":[\"Hoa mọc ở mặt trong\"],\"free\":[],\"unsure\":[],\"glossary\":[{\"term\":\"ficin\",\"say\":\"phi-xin\",\"draw\":\"quầng sáng\",\"avoid\":\"cảnh tiêu hoá\"}]}\n```"

func (f *lockFixture) reviewReply(findings string) string {
	return "```json\n{\"findings\":[" + findings + "],\"notes\":[\"Nhịp ổn.\"]}\n```"
}

func (f *lockFixture) approveAllHandoff(t *testing.T) {
	t.Helper()
	f.llm.replies[handoffMarker] = handoffReply
	if err := f.uc2.GenerateHandoff(context.Background(), "p1", ""); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"b1", "b2", "b3"} {
		if err := f.uc2.ConfirmHandoff(context.Background(), "p1", k); err != nil {
			t.Fatal(err)
		}
	}
}

func openFindingIDs(s ScriptLockState) []string {
	var ids []string
	for _, f := range s.Findings {
		if f.State == domain.FindingOpen {
			ids = append(ids, f.ID)
		}
	}
	return ids
}

func TestLockStepIsClosedUntilEveryBeatsDialogueIsApproved(t *testing.T) {
	f := newLockFixture(t)
	s, _ := f.beat.units.GetUnit(context.Background(), "p1", domain.UnitKindScriptBeat, "b2")
	s.Status = domain.UnitPending
	if _, err := f.beat.units.SaveUnit(context.Background(), s); err != nil {
		t.Fatal(err)
	}
	st, err := f.uc2.State(context.Background(), "p1")
	if err != nil || st.ScriptReady || st.CanLock || len(st.Findings) != 0 {
		t.Fatalf("state = ready %v canlock %v findings %d err %v; want a closed step", st.ScriptReady, st.CanLock, len(st.Findings), err)
	}
	if err := f.uc2.Review(context.Background(), "p1"); !errors.Is(err, ErrLockNeedsScript) {
		t.Errorf("Review err = %v, want ErrLockNeedsScript", err)
	}
	if len(f.llm.systems) != 0 {
		t.Error("the model must not be called while the script is not approved")
	}
}

func TestFreshStepAsksForAReviewAndCannotLock(t *testing.T) {
	f := newLockFixture(t)
	st, err := f.uc2.State(context.Background(), "p1")
	if err != nil {
		t.Fatal(err)
	}
	var review *FindingView
	for i := range st.Findings {
		if st.Findings[i].ID == "m-review" {
			review = &st.Findings[i]
		}
	}
	if review == nil || review.State != domain.FindingOpen || st.CanLock {
		t.Fatalf("review finding %+v canLock %v; a script nobody reviewed must hold the lock back", review, st.CanLock)
	}
}

func TestReviewStoresFindingsAndWholeScriptGoesInOneCall(t *testing.T) {
	f := newLockFixture(t)
	f.llm.replies[reviewMarker] = f.reviewReply(`{"group":"logic","title":"Chuyển đột ngột","detail":"d","line":"b2.1","fix":"Câu thay thế."}`)
	if err := f.uc2.Review(context.Background(), "p1"); err != nil {
		t.Fatal(err)
	}
	if len(f.llm.systems) != 1 || !strings.Contains(f.llm.systems[0], "b3.2: ") {
		t.Fatalf("%d calls; the whole script must be in the one prompt", len(f.llm.systems))
	}
	st, _ := f.uc2.State(context.Background(), "p1")
	if len(st.AINotes) != 1 {
		t.Errorf("ai notes = %v", st.AINotes)
	}
	ids := openFindingIDs(st)
	if len(ids) != 1 || ids[0] != "r1" {
		t.Errorf("open = %v, want only r1 (the machine passes and the script was reviewed)", ids)
	}
}

func TestUnusableReviewReplyStoresNothing(t *testing.T) {
	f := newLockFixture(t)
	f.llm.replies[reviewMarker] = f.reviewReply(`{"group":"logic","title":"x","line":"b9.9"}`)
	if err := f.uc2.Review(context.Background(), "p1"); !errors.Is(err, domain.ErrReviewReplyUnusable) {
		t.Fatalf("err = %v, want ErrReviewReplyUnusable", err)
	}
	st, _ := f.uc2.State(context.Background(), "p1")
	for _, fv := range st.Findings {
		if fv.Source == domain.FindingByAI {
			t.Errorf("an AI finding %s was stored from an unusable reply", fv.ID)
		}
	}
}

func TestApplyFixEditsTheLineSendsBeatBackAndReopensTheFinalCheck(t *testing.T) {
	f := newLockFixture(t)
	ctx := context.Background()
	f.llm.replies[reviewMarker] = f.reviewReply(`{"group":"logic","title":"T","detail":"d","line":"b2.1","fix":"Câu thay thế."}`)
	if err := f.uc2.Review(ctx, "p1"); err != nil {
		t.Fatal(err)
	}
	if err := f.uc2.ApplyFix(ctx, "p1", "r1"); err != nil {
		t.Fatal(err)
	}
	if got := f.script(t, "b2").Lines[0].Text; got != "Câu thay thế." {
		t.Errorf("b2.1 = %q, want the fix", got)
	}
	u, _ := f.beat.units.GetUnit(ctx, "p1", domain.UnitKindScriptBeat, "b2")
	if u.Status != domain.UnitPending {
		t.Error("a beat edited by a fix must go back to review")
	}
	st, _ := f.uc2.State(ctx, "p1")
	if st.ScriptReady || st.CanLock {
		t.Errorf("ready %v canLock %v; b2 is pending again", st.ScriptReady, st.CanLock)
	}
	// Approve b2 again: the final check must say the dialogue changed since the review.
	u.Status = domain.UnitApproved
	if _, err := f.beat.units.SaveUnit(ctx, u); err != nil {
		t.Fatal(err)
	}
	st, _ = f.uc2.State(ctx, "p1")
	var fin FindingView
	for _, fv := range st.Findings {
		if fv.ID == "m-review" {
			fin = fv
		}
	}
	if fin.State != domain.FindingOpen || !strings.Contains(fin.Title, "đã đổi") {
		t.Errorf("final check = %+v, want open: dialogue changed after the review", fin)
	}
}

func TestSecondReviewMarksVanishedFindingsFixedAndReopensRepeatedOnes(t *testing.T) {
	f := newLockFixture(t)
	ctx := context.Background()
	f.llm.replies[reviewMarker] = f.reviewReply(`{"group":"logic","title":"A","line":"b1.1"},{"group":"clarity","title":"B"}`)
	if err := f.uc2.Review(ctx, "p1"); err != nil {
		t.Fatal(err)
	}
	if err := f.uc2.Accept(ctx, "p1", "r2", "đã biết"); err != nil {
		t.Fatal(err)
	}
	f.llm.replies[reviewMarker] = f.reviewReply(`{"group":"logic","title":"A","line":"b1.1","same_as":"r1"}`)
	if err := f.uc2.Review(ctx, "p1"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(f.llm.systems[1], "r1 [logic] A") {
		t.Errorf("the second review must be shown the first one's findings: %q", f.llm.systems[1])
	}
	st, _ := f.uc2.State(ctx, "p1")
	by := map[string]FindingView{}
	for _, fv := range st.Findings {
		by[fv.ID] = fv
	}
	if by["r1"].State != domain.FindingOpen || by["r2"].State != domain.FindingAccepted || by["r2"].Why != "đã biết" {
		t.Errorf("r1 %s r2 %s/%q; want r1 still open and r2 still accepted with its reason", by["r1"].State, by["r2"].State, by["r2"].Why)
	}
}

func TestAddToHandoffPutsTheLineInTheBeatNoteAndSendsItBackToReview(t *testing.T) {
	f := newLockFixture(t)
	ctx := context.Background()
	f.llm.replies[reviewMarker] = f.reviewReply(`{"group":"clarity","title":"đáy quả","handoff":{"beat":"b2","section":"terms","text":"đáy quả là đầu đối diện cuống"}}`)
	if err := f.uc2.Review(ctx, "p1"); err != nil {
		t.Fatal(err)
	}
	if err := f.uc2.AddToHandoff(ctx, "p1", "r1"); err != nil {
		t.Fatal(err)
	}
	st, _ := f.uc2.State(ctx, "p1")
	var b LockBeatView
	for _, v := range st.Beats {
		if v.Key == "b2" {
			b = v
		}
	}
	if !b.Written || b.Status != domain.UnitPending || len(b.Note.Terms) != 1 {
		t.Errorf("b2 note = %+v, want terms with the line and pending", b)
	}
	if err := f.uc2.AddToHandoff(ctx, "p1", "r1"); !errors.Is(err, ErrFindingNotApplicable) {
		t.Errorf("a finding already dealt with must not apply twice, err = %v", err)
	}
}

func TestGenerateHandoffWritesNotesBeatByBeatAndMergesGlossaryKeepingTheCreators(t *testing.T) {
	f := newLockFixture(t)
	ctx := context.Background()
	if err := f.uc2.AddTerm(ctx, "p1", domain.GlossaryTerm{Term: "Ficin", Say: "của Creator"}); err != nil {
		t.Fatal(err)
	}
	f.llm.replies[handoffMarker] = handoffReply
	if err := f.uc2.GenerateHandoff(ctx, "p1", ""); err != nil {
		t.Fatal(err)
	}
	if len(f.llm.systems) != 3 {
		t.Fatalf("%d calls, want one per beat", len(f.llm.systems))
	}
	st, _ := f.uc2.State(ctx, "p1")
	if len(st.Glossary) != 1 || st.Glossary[0].Say != "của Creator" || !st.Glossary[0].Mine {
		t.Errorf("glossary = %+v; the Creator's row must survive an AI term of the same word", st.Glossary)
	}
	if err := f.uc2.GenerateHandoff(ctx, "p1", "b1"); !errors.Is(err, ErrHandoffExists) {
		t.Errorf("err = %v, want ErrHandoffExists", err)
	}
}

func TestHandoffEditMarksCreatorSectionsAndRedoStopsThemBeingTheirs(t *testing.T) {
	f := newLockFixture(t)
	ctx := context.Background()
	kind := f.uc2.HandoffUnitKind()
	u, err := kind.Edit(ctx, "p1", "b1", map[string]string{"must": "một\n\nhai"})
	if err != nil || !u.HasCreatorField("must") {
		t.Fatalf("edit = %+v err %v", u, err)
	}
	n, _ := readHandoff(u)
	if len(n.Must) != 2 {
		t.Errorf("must = %v, want blank lines dropped", n.Must)
	}
	if _, err := kind.Edit(ctx, "p1", "b1", map[string]string{"bogus": "x"}); !errors.Is(err, domain.ErrInvalidUnitEdit) {
		t.Errorf("unknown section err = %v", err)
	}
	f.llm.replies[handoffMarker] = handoffReply
	u, err = kind.Redo(ctx, "p1", "b1", "ngắn hơn")
	if err != nil || len(u.CreatorFields) != 0 || u.RedoCount != 1 {
		t.Errorf("redo = %+v err %v, want the note no longer the Creator's and RedoCount 1", u, err)
	}
	empty := domain.ReviewUnit{Content: []byte(`{"must":[],"terms":[],"exact":[],"free":[],"unsure":[]}`)}
	if err := kind.CheckApprovable(ctx, empty); !errors.Is(err, domain.ErrHandoffEmpty) {
		t.Errorf("an empty note must not be approvable, err = %v", err)
	}
}

func TestScriptEditMakesTheHandoffNoteStaleAndHoldsTheLock(t *testing.T) {
	f := newLockFixture(t)
	ctx := context.Background()
	f.approveAllHandoff(t)
	if _, err := f.uc.ScriptUnitKind().Edit(ctx, "p1", "b1", map[string]string{"line:b1.1": "Câu mới hẳn."}); err != nil {
		t.Fatal(err)
	}
	u, _ := f.beat.units.GetUnit(ctx, "p1", domain.UnitKindScriptBeat, "b1")
	u.Status = domain.UnitApproved
	f.beat.units.SaveUnit(ctx, u)
	st, _ := f.uc2.State(ctx, "p1")
	var b1 LockBeatView
	for _, v := range st.Beats {
		if v.Key == "b1" {
			b1 = v
		}
	}
	if !b1.Stale {
		t.Fatalf("b1 note = %+v; a note built from older dialogue must be stale", b1)
	}
	if st.CanLock || st.Conditions[2].OK {
		t.Error("a stale handoff note must hold the lock back")
	}
	if err := f.uc2.ConfirmHandoff(ctx, "p1", "b1"); err != nil {
		t.Fatal(err)
	}
	st, _ = f.uc2.State(ctx, "p1")
	if st.Conditions[2].OK == false {
		t.Errorf("confirming the note must clear the condition: %+v", st.Conditions)
	}
}

// reviewClean runs a review that finds nothing, so only the Creator's own work holds the lock back.
func (f *lockFixture) reviewClean(t *testing.T) {
	t.Helper()
	f.llm.replies[reviewMarker] = f.reviewReply("")
	if err := f.uc2.Review(context.Background(), "p1"); err != nil {
		t.Fatal(err)
	}
}

func TestLockRefusesUntilEveryConditionHoldsThenFreezesV1(t *testing.T) {
	f := newLockFixture(t)
	ctx := context.Background()
	if _, err := f.uc2.Lock(ctx, "p1", ""); !errors.Is(err, domain.ErrLockNotReady) {
		t.Fatalf("err = %v, want ErrLockNotReady before anything is done", err)
	}
	f.reviewClean(t)
	f.approveAllHandoff(t)
	st, _ := f.uc2.State(ctx, "p1")
	// Machine findings that may remain open: the figures/flags on the fixture's plain lines are none.
	if !st.CanLock {
		t.Fatalf("conditions = %+v open = %v; want ready", st.Conditions, openFindingIDs(st))
	}
	v, err := f.uc2.Lock(ctx, "p1", "")
	if err != nil || v.Version != 1 || v.Note != "Chốt lần đầu" {
		t.Fatalf("lock = %+v err %v", v, err)
	}
	if _, err := f.uc2.Lock(ctx, "p1", ""); !errors.Is(err, domain.ErrNothingToLock) {
		t.Errorf("second lock with no change: err = %v, want ErrNothingToLock", err)
	}
}

func TestEditAfterLockOpensADraftAndLockingV2KeepsV1InHistory(t *testing.T) {
	f := newLockFixture(t)
	ctx := context.Background()
	f.reviewClean(t)
	f.approveAllHandoff(t)
	if _, err := f.uc2.Lock(ctx, "p1", ""); err != nil {
		t.Fatal(err)
	}
	v1, _ := f.versions.LatestScriptVersion(ctx, "p1")

	// Change the dialogue of b2 after v1 was locked, approve it, review and confirm its note again.
	if _, err := f.uc.ScriptUnitKind().Edit(ctx, "p1", "b2", map[string]string{"line:b2.2": "Một câu hai mới."}); err != nil {
		t.Fatal(err)
	}
	u, _ := f.beat.units.GetUnit(ctx, "p1", domain.UnitKindScriptBeat, "b2")
	u.Status = domain.UnitApproved
	f.beat.units.SaveUnit(ctx, u)
	f.reviewClean(t)
	if err := f.uc2.ConfirmHandoff(ctx, "p1", "b2"); err != nil {
		t.Fatal(err)
	}

	st, _ := f.uc2.State(ctx, "p1")
	if st.LockedVersion != 1 || st.NextVersion != 2 {
		t.Fatalf("locked %d next %d", st.LockedVersion, st.NextVersion)
	}
	if len(st.Draft) != 1 || st.Draft[0] != "Lời thoại · B2" {
		t.Fatalf("draft = %v, want exactly 'Lời thoại · B2'", st.Draft)
	}
	if !st.CanLock {
		t.Fatalf("conditions = %+v open = %v", st.Conditions, openFindingIDs(st))
	}
	if cur, _ := f.versions.LatestScriptVersion(ctx, "p1"); cur.Version != 1 {
		t.Error("the steps after must still read v1 until v2 is locked")
	}
	v, err := f.uc2.Lock(ctx, "p1", "")
	if err != nil || v.Version != 2 || v.Note != "Chốt 1 thay đổi" {
		t.Fatalf("lock v2 = %+v err %v", v, err)
	}
	st, _ = f.uc2.State(ctx, "p1")
	if len(st.Versions) != 2 || st.Versions[0].Version != 1 || len(st.Draft) != 0 {
		t.Errorf("versions %+v draft %v; want v1 and v2 in history and no draft", st.Versions, st.Draft)
	}
	// v1 is untouched.
	again, _ := f.versions.ListScriptVersions(ctx, "p1")
	if again[0].Parts[0].SHA != v1.Parts[0].SHA || len(again[0].Parts) != len(v1.Parts) {
		t.Error("v1 changed after v2 was locked")
	}
	pkg, err := f.uc2.HandoffForRole(ctx, "p1", domain.HandoffRoleStoryboard)
	if err != nil || pkg.Version != 2 || len(pkg.Beats) != 3 {
		t.Fatalf("handoff = %+v err %v", pkg, err)
	}
}

func TestAcceptedFindingNeedsNoReasonTextAndIsFrozenWithTheVersion(t *testing.T) {
	f := newLockFixture(t)
	ctx := context.Background()
	f.reviewClean(t)
	f.approveAllHandoff(t)
	// Make the length leave the target by accepting nothing yet: force a machine finding with a long line.
	if _, err := f.uc.ScriptUnitKind().Edit(ctx, "p1", "b1", map[string]string{"line:b1.2": strings.Repeat("rất dài ", 30)}); err != nil {
		t.Fatal(err)
	}
	u, _ := f.beat.units.GetUnit(ctx, "p1", domain.UnitKindScriptBeat, "b1")
	u.Status = domain.UnitApproved
	f.beat.units.SaveUnit(ctx, u)
	f.reviewClean(t)
	if err := f.uc2.ConfirmHandoff(ctx, "p1", "b1"); err != nil {
		t.Fatal(err)
	}
	st, _ := f.uc2.State(ctx, "p1")
	ids := openFindingIDs(st)
	if len(ids) == 0 || st.CanLock {
		t.Fatalf("a long line must leave an open machine finding, open = %v", ids)
	}
	for _, id := range ids {
		if err := f.uc2.Accept(ctx, "p1", id, " "); err != nil {
			t.Fatal(err)
		}
	}
	st, _ = f.uc2.State(ctx, "p1")
	if !st.CanLock {
		t.Fatalf("after accepting, conditions = %+v", st.Conditions)
	}
	if _, err := f.uc2.Lock(ctx, "p1", ""); err != nil {
		t.Fatal(err)
	}
	v, _ := f.versions.LatestScriptVersion(ctx, "p1")
	p, ok := v.Part(domain.PartReview)
	if !ok || !strings.Contains(string(p.Content), "không ghi lý do") {
		t.Errorf("review part = %s; the accepted finding and its reason must be frozen", p.Content)
	}
	if err := f.uc2.Accept(ctx, "p1", "m-must", "x"); !errors.Is(err, ErrFindingNotApplicable) {
		t.Errorf("a passing check cannot be accepted, err = %v", err)
	}
}

// flakyLLM answers each call from replies in order, and keeps the last one for any call after.
type flakyLLM struct {
	replies []string
	calls   int
}

func (l *flakyLLM) Name() string { return "fake" }
func (l *flakyLLM) Chat(context.Context, ChatRequest) (ChatResult, error) {
	i := min(l.calls, len(l.replies)-1)
	l.calls++
	return ChatResult{Content: l.replies[i]}, nil
}

func TestReviewAsksAgainWhenTheJSONIsBrokenAndGivesUpAfterThreeTries(t *testing.T) {
	f := newLockFixture(t)
	broken := `{"findings":[],"notes":["a,"b"]}}`
	good := f.reviewReply(`{"group":"logic","title":"T","line":"b1.1"}`)
	llm := &flakyLLM{replies: []string{broken, good}}
	f.uc2.provider = llm
	if err := f.uc2.Review(context.Background(), "p1"); err != nil {
		t.Fatalf("Review: %v", err)
	}
	if llm.calls != 2 {
		t.Errorf("%d calls, want the broken reply asked again once", llm.calls)
	}
	st, _ := f.uc2.State(context.Background(), "p1")
	if ids := openFindingIDs(st); len(ids) != 1 || ids[0] != "r1" {
		t.Errorf("open = %v, want r1 from the second reply", ids)
	}

	always := &flakyLLM{replies: []string{broken}}
	f.uc2.provider = always
	if err := f.uc2.Review(context.Background(), "p1"); !errors.Is(err, domain.ErrReviewReplyUnusable) {
		t.Errorf("err = %v, want ErrReviewReplyUnusable", err)
	}
	if always.calls != maxReplyAttempts {
		t.Errorf("%d calls, want %d", always.calls, maxReplyAttempts)
	}
}
