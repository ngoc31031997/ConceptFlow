package application

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"authoring/internal/domain"
)

// The three replies a skeleton of a hook, one chain step and a payoff gets, one per beat.
const (
	beatReplyHook   = `{"beats":[{"key":"","fn":"Hook","title":"Quả sung","goal":"Gây tò mò","content":"Mở từ đĩa sung","facts":[],"dur":18,"next":"Zoom vào"}]}`
	beatReplyMiddle = `{"beats":[{"key":"","fn":"Giải thích cơ chế","title":"Cửa hẹp","goal":"Ong mất cánh","content":"Ong chui qua cửa","facts":["F02"],"dur":200,"next":"Vào trong"}]}`
	beatReplyPayoff = `{"beats":[{"key":"","fn":"Payoff","title":"Ai nuôi ai","goal":"Trả lời","content":"Hai loài dựa nhau","facts":["F03"],"dur":50,"next":""}]}`
)

// beatLLM answers each call with the next reply, then repeats the last, and keeps the prompts it was sent.
type beatLLM struct {
	replies []string
	systems []string
	err     error
}

func (l *beatLLM) Name() string { return "fake" }
func (l *beatLLM) Chat(_ context.Context, req ChatRequest) (ChatResult, error) {
	l.systems = append(l.systems, req.System)
	i := min(len(l.systems)-1, len(l.replies)-1)
	return ChatResult{Content: l.replies[i]}, l.err
}

// say makes every later call answer with the given replies in turn.
func (l *beatLLM) say(replies ...string) {
	l.replies, l.systems = replies, nil
}

// fakeStories is the approved story the beat sheet reads; ok false is a story not approved.
type fakeStories struct {
	story domain.StoryStructure
	ok    bool
}

func (s *fakeStories) ApprovedStory(context.Context, string) (domain.StoryStructure, string, bool, error) {
	return s.story, s.story.Hash(), s.ok, nil
}

type beatFixture struct {
	units   *briefUnits
	store   *researchStore
	stories *fakeStories
	briefs  *approvedSteps
	llm     *beatLLM
	uc      *BeatDeskUseCase
}

func newBeatFixture() *beatFixture {
	units := newBriefUnits()
	project := &briefProject{project: domain.Project{ContentLanguage: domain.LanguageVietnamese, VideoOutputMode: domain.ModeLongOnly}, status: domain.StatusDraft}
	f := &beatFixture{
		units:   units,
		store:   &researchStore{briefUnits: units, topic: "Vì sao trong quả sung có xác ong?", duration: domain.Duration4to5},
		stories: &fakeStories{},
		briefs:  &approvedSteps{},
		llm:     &beatLLM{replies: []string{beatReplyHook, beatReplyMiddle, beatReplyPayoff}},
	}
	f.uc = NewBeatDeskUseCase(f.store, f.stories, f.briefs, project, project, briefPrompts{}, f.llm, nil, 4000)
	return f
}

func (f *beatFixture) save(t *testing.T, kind, key string, status domain.UnitStatus, content any, creatorFields ...string) {
	t.Helper()
	raw, err := json.Marshal(content)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.units.SaveUnit(context.Background(), domain.ReviewUnit{
		ProjectID: "p1", Kind: kind, Key: key, Status: status, Content: raw, CreatorFields: creatorFields,
	}); err != nil {
		t.Fatal(err)
	}
}

// open approves the brief and the story and seeds three facts, F01 a must. The
// chain has one step citing F02 and the payoff cites F03, so the skeleton is
// three beats.
func (f *beatFixture) open(t *testing.T) {
	t.Helper()
	f.briefs.brief, f.briefs.briefOK = domain.ProjectBrief{Audience: "Người tò mò", DurationSeconds: 270, Tone: "Gần gũi", LearningObjectives: []string{"Hiểu vì sao"}, Avoid: []string{"Thuật ngữ"}}, true
	f.stories.story, f.stories.ok = domain.StoryStructure{
		CentralQuestion: "Vì sao ong chết trong quả sung?",
		CoreAnswer:      "Hai loài dựa nhau. [F03]",
		NarrativeArc:    []domain.ArcBeat{{Beat: "Cơ chế", Text: "Ong chui qua cửa hẹp.", Facts: []string{"F02"}}},
	}, true
	f.saveFact(t, "F01", domain.Fact{ClaimVI: "Sung là chùm hoa kín.", Type: domain.FactTypeFact, Keep: true, Must: true})
	f.saveFact(t, "F02", domain.Fact{ClaimVI: "Ong mẹ chết trong quả.", Type: domain.FactTypeFact, Keep: true})
	f.saveFact(t, "F03", domain.Fact{ClaimVI: "Ficin phân giải xác ong.", Type: domain.FactTypeHypothesis, Keep: true})
	f.saveFact(t, "F04", domain.Fact{ClaimVI: "Đã bỏ.", Type: domain.FactTypeFact, Keep: false})
}

// saveFact puts a fact in the research table the beat sheet reads.
func (f *beatFixture) saveFact(t *testing.T, id string, fact domain.Fact) {
	t.Helper()
	if err := f.store.SaveResearchFact(context.Background(), "p1", domain.StoredFact{ID: id, Fact: fact}); err != nil {
		t.Fatal(err)
	}
}

func (f *beatFixture) beat(t *testing.T, key string) (domain.ReviewUnit, domain.Beat) {
	t.Helper()
	u, err := f.units.GetUnit(context.Background(), "p1", domain.UnitKindBeat, key)
	if err != nil {
		t.Fatalf("beat %s: %v", key, err)
	}
	b, err := readBeat(u)
	if err != nil {
		t.Fatal(err)
	}
	return u, b
}

// skeleton opens the step, builds the skeleton and gives the hook the must fact F01.
func (f *beatFixture) skeleton(t *testing.T) {
	t.Helper()
	f.open(t)
	if err := f.uc.BuildSkeleton(context.Background(), "p1"); err != nil {
		t.Fatalf("BuildSkeleton: %v", err)
	}
	if _, err := f.uc.BeatUnitKind().Edit(context.Background(), "p1", "b1", map[string]string{"facts": "F01"}); err != nil {
		t.Fatal(err)
	}
}

// generate takes the step through both phases: skeleton built and approved, content written.
func (f *beatFixture) generate(t *testing.T) BeatState {
	t.Helper()
	f.skeleton(t)
	if err := f.uc.ApproveSkeleton(context.Background(), "p1"); err != nil {
		t.Fatalf("ApproveSkeleton: %v", err)
	}
	if err := f.uc.Generate(context.Background(), "p1", "", false); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	state, err := f.uc.State(context.Background(), "p1")
	if err != nil {
		t.Fatal(err)
	}
	return state
}

func TestBuildSkeletonHasAHookAStepBeatAndAPayoffWithTheirFacts(t *testing.T) {
	f := newBeatFixture()
	f.open(t)
	if err := f.uc.BuildSkeleton(context.Background(), "p1"); err != nil {
		t.Fatalf("BuildSkeleton: %v", err)
	}
	state, _ := f.uc.State(context.Background(), "p1")
	if state.Phase != BeatPhaseSkeleton || len(state.Beats) != 3 || state.Unwritten != 3 {
		t.Fatalf("phase=%s beats=%d unwritten=%d, want a 3 beat skeleton with nothing written", state.Phase, len(state.Beats), state.Unwritten)
	}
	fns := []string{state.Beats[0].Fn, state.Beats[1].Fn, state.Beats[2].Fn}
	if strings.Join(fns, "|") != "Hook|Giải thích cơ chế|Payoff" {
		t.Errorf("functions = %v", fns)
	}
	if strings.Join(state.Beats[1].Facts, ",") != "F02" || strings.Join(state.Beats[2].Facts, ",") != "F03" || state.Beats[1].Step != "Ong chui qua cửa hẹp." {
		t.Errorf("beats = %+v, want the chain step's and the payoff's facts and text", state.Beats)
	}
	if state.Beats[0].NoFact || state.Beats[1].NoFact {
		t.Error("the hook needs no fact and the step cites one: neither is flagged")
	}
	if len(f.llm.systems) != 0 {
		t.Error("the skeleton must be built without calling the model")
	}
	if len(state.Checks) != 3 || state.Checks[2].OK {
		t.Errorf("checks = %+v, want the three skeleton checks with the must fact F01 missing", state.Checks)
	}
}

func TestBuildSkeletonFlagsAStepWithoutAFact(t *testing.T) {
	f := newBeatFixture()
	f.open(t)
	f.stories.story.NarrativeArc = []domain.ArcBeat{
		{Beat: "Cơ chế", Text: "Ong chui qua cửa hẹp.", Facts: []string{"F02"}},
		{Beat: "Rẽ ngang", Text: "Mắt người nhạy với màu xanh."},
	}
	if err := f.uc.BuildSkeleton(context.Background(), "p1"); err != nil {
		t.Fatalf("BuildSkeleton: %v", err)
	}
	state, _ := f.uc.State(context.Background(), "p1")
	var flagged []string
	for _, b := range state.Beats {
		if b.NoFact {
			flagged = append(flagged, b.Step)
		}
	}
	if len(state.Beats) != 4 || len(flagged) != 1 || flagged[0] != "Mắt người nhạy với màu xanh." {
		t.Errorf("beats=%d flagged=%v, want only the step that cites no fact", len(state.Beats), flagged)
	}
}

func TestBuildSkeletonRebuildsUntilItIsApproved(t *testing.T) {
	f := newBeatFixture()
	f.skeleton(t)
	if err := f.uc.BuildSkeleton(context.Background(), "p1"); err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	state, _ := f.uc.State(context.Background(), "p1")
	if len(state.Beats) != 3 || state.Beats[0].Key != "b4" {
		t.Errorf("beats = %+v, want three new beats keyed from b4 (keys are never reused)", state.Beats)
	}
	if err := f.uc.ApproveSkeleton(context.Background(), "p1"); err != nil {
		t.Fatal(err)
	}
	if err := f.uc.BuildSkeleton(context.Background(), "p1"); !errors.Is(err, ErrBeatSkeletonLocked) {
		t.Errorf("err = %v, want the approved skeleton to stay", err)
	}
}

func TestBuildSkeletonRefusesWhenTheChainCannotFitTheTarget(t *testing.T) {
	f := newBeatFixture()
	f.open(t)
	f.stories.story.NarrativeArc = nil
	for i := 0; i < 19; i++ {
		f.stories.story.NarrativeArc = append(f.stories.story.NarrativeArc, domain.ArcBeat{Beat: "Nhịp", Text: "Một bước.", Facts: []string{"F02"}})
	}
	if err := f.uc.BuildSkeleton(context.Background(), "p1"); !errors.Is(err, domain.ErrInvalidUnitEdit) {
		t.Errorf("err = %v, want a chain of 19 steps refused", err)
	}
}

func TestDeletingEveryBeatOpensTheSkeletonAgain(t *testing.T) {
	f := newBeatFixture()
	f.generate(t)
	for _, key := range []string{"b1", "b2", "b3"} {
		if err := f.uc.Delete(context.Background(), "p1", key); err != nil {
			t.Fatalf("Delete %s: %v", key, err)
		}
	}
	if state, _ := f.uc.State(context.Background(), "p1"); state.Phase != BeatPhaseEmpty {
		t.Fatalf("phase = %s, want an empty sheet", state.Phase)
	}
	if err := f.uc.BuildSkeleton(context.Background(), "p1"); err != nil {
		t.Fatalf("BuildSkeleton after deleting every beat: %v", err)
	}
	state, _ := f.uc.State(context.Background(), "p1")
	if state.Phase != BeatPhaseSkeleton || len(state.Beats) != 3 {
		t.Errorf("phase=%s beats=%d, want a new unapproved skeleton of three beats", state.Phase, len(state.Beats))
	}
	if err := f.uc.Generate(context.Background(), "p1", "", false); !errors.Is(err, ErrBeatSkeletonNeeded) {
		t.Errorf("err = %v, want the new skeleton to need its own approval", err)
	}
}

func TestBuildSkeletonIgnoresAnApprovalLeftOnAnEmptySheet(t *testing.T) {
	f := newBeatFixture()
	f.open(t)
	sheet, gate, _ := f.uc.sheet(context.Background(), "p1")
	gate.SkeletonApproved = true
	if err := f.uc.saveSheet(context.Background(), sheet, gate); err != nil {
		t.Fatal(err)
	}
	if err := f.uc.BuildSkeleton(context.Background(), "p1"); err != nil {
		t.Errorf("BuildSkeleton: %v, want an approval with no beat behind it ignored", err)
	}
}

func TestApproveSkeletonNeedsABeat(t *testing.T) {
	f := newBeatFixture()
	f.open(t)
	if err := f.uc.ApproveSkeleton(context.Background(), "p1"); !errors.Is(err, domain.ErrInvalidUnitEdit) {
		t.Errorf("err = %v, want an empty sheet refused", err)
	}
}

func TestGenerateNeedsTheApprovedSkeleton(t *testing.T) {
	f := newBeatFixture()
	f.skeleton(t)
	if err := f.uc.Generate(context.Background(), "p1", "", false); !errors.Is(err, ErrBeatSkeletonNeeded) {
		t.Errorf("err = %v, want the skeleton to be approved first", err)
	}
	if len(f.llm.systems) != 0 {
		t.Error("the model must not be called before the skeleton is approved")
	}
}

func TestGenerateWritesOneBeatPerCallInOrderAndKeepsTheKeys(t *testing.T) {
	f := newBeatFixture()
	state := f.generate(t)
	if len(f.llm.systems) != 3 {
		t.Fatalf("%d model calls, want one per beat", len(f.llm.systems))
	}
	if len(state.Beats) != 3 || state.Beats[0].Key != "b1" || state.Beats[2].Key != "b3" || state.Beats[1].Title != "Cửa hẹp" || state.Unwritten != 0 {
		t.Fatalf("beats = %+v, want three written beats keyed b1..b3", state.Beats)
	}
	if state.Phase != BeatPhaseContent || state.Status != domain.UnitPending || state.TotalSeconds != 268 || state.Rebalanced != 0 {
		t.Errorf("phase=%s status=%s total=%d rebalanced=%d", state.Phase, state.Status, state.TotalSeconds, state.Rebalanced)
	}
	if !strings.Contains(f.llm.systems[0], "F01 [fact · bắt buộc nói]") || !strings.Contains(f.llm.systems[0], "question (Câu hỏi trung tâm)") {
		t.Errorf("the prompt must carry the facts and the story, got %q", f.llm.systems[0])
	}
	if !strings.Contains(f.llm.systems[1], "beat thứ 2 trong 3 beat") || !strings.Contains(f.llm.systems[1], "Ong chui qua cửa hẹp.") || !strings.Contains(f.llm.systems[1], "Quả sung") {
		t.Errorf("the second call must name its place, its step and show the beat written before it, got %q", f.llm.systems[1])
	}
	if !strings.Contains(f.llm.systems[0], "CHƯA VIẾT") {
		t.Error("the first call must show the beats still to write as not written")
	}
}

func TestGenerateKeepsFunctionAndFactsOfTheSkeleton(t *testing.T) {
	f := newBeatFixture()
	f.skeleton(t)
	if err := f.uc.ApproveSkeleton(context.Background(), "p1"); err != nil {
		t.Fatal(err)
	}
	f.llm.say(beatReplyHook, strings.Replace(strings.Replace(beatReplyMiddle, `"Giải thích cơ chế"`, `"Cao trào"`, 1), `["F02"]`, `["F03"]`, 1), beatReplyPayoff)
	if err := f.uc.Generate(context.Background(), "p1", "", false); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if _, b := f.beat(t, "b2"); b.Fn != "Giải thích cơ chế" || strings.Join(b.Facts, ",") != "F02" || b.Title != "Cửa hẹp" {
		t.Errorf("beat = %+v, want the model's wording with the skeleton's function and facts", b)
	}
}

func TestGenerateAsksAgainOnceForABeatItCannotRead(t *testing.T) {
	f := newBeatFixture()
	f.skeleton(t)
	if err := f.uc.ApproveSkeleton(context.Background(), "p1"); err != nil {
		t.Fatal(err)
	}
	f.llm.say("không phải JSON", beatReplyHook, beatReplyMiddle, beatReplyPayoff)
	if err := f.uc.Generate(context.Background(), "p1", "", false); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(f.llm.systems) != 4 || !strings.Contains(f.llm.systems[1], "LẦN VIẾT TRƯỚC SAI") {
		t.Fatalf("%d calls, want the first beat asked twice with what went wrong in the note", len(f.llm.systems))
	}
}

func TestGenerateStoresNothingWhenABeatStaysUnreadable(t *testing.T) {
	f := newBeatFixture()
	f.skeleton(t)
	if err := f.uc.ApproveSkeleton(context.Background(), "p1"); err != nil {
		t.Fatal(err)
	}
	f.llm.say(beatReplyHook, "không phải JSON")
	err := f.uc.Generate(context.Background(), "p1", "", false)
	if !errors.Is(err, domain.ErrBeatReplyUnusable) || !strings.Contains(err.Error(), "b2") {
		t.Fatalf("err = %v, want the unreadable beat b2 named", err)
	}
	if _, b := f.beat(t, "b1"); b.Written() {
		t.Error("beat b1 was written in a run that failed: nothing may be stored")
	}
}

func TestGenerateBringsTheTotalInsideTheTarget(t *testing.T) {
	f := newBeatFixture()
	f.skeleton(t)
	if err := f.uc.ApproveSkeleton(context.Background(), "p1"); err != nil {
		t.Fatal(err)
	}
	slow := func(reply, from, to string) string { return strings.Replace(reply, from, to, 1) }
	f.llm.say(beatReplyHook, slow(beatReplyMiddle, `"dur":200`, `"dur":500`), slow(beatReplyPayoff, `"dur":50`, `"dur":200`))
	if err := f.uc.Generate(context.Background(), "p1", "", false); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	state, _ := f.uc.State(context.Background(), "p1")
	if state.TotalSeconds < 240 || state.TotalSeconds > 300 || state.Rebalanced == 0 {
		t.Errorf("total=%d rebalanced=%d, want a total inside 240–300 and the change reported", state.TotalSeconds, state.Rebalanced)
	}
	if state.Beats[0].Dur > domain.MaxHookSeconds {
		t.Errorf("hook = %d s, want at most %d", state.Beats[0].Dur, domain.MaxHookSeconds)
	}
}

func TestGenerateOnlyUnwrittenLeavesWrittenBeatsAlone(t *testing.T) {
	f := newBeatFixture()
	f.generate(t)
	if _, err := f.uc.Add(context.Background(), "p1", "b1", true, ""); err != nil {
		t.Fatal(err)
	}
	f.llm.say(strings.Replace(beatReplyMiddle, `"title":"Cửa hẹp"`, `"title":"Beat mới viết"`, 1))
	if err := f.uc.Generate(context.Background(), "p1", "", true); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(f.llm.systems) != 1 {
		t.Fatalf("%d model calls, want one for the one beat without content", len(f.llm.systems))
	}
	state, _ := f.uc.State(context.Background(), "p1")
	if state.Unwritten != 0 || state.Beats[1].Title != "Beat mới viết" || state.Beats[0].Title != "Quả sung" || state.Beats[2].Title != "Cửa hẹp" {
		t.Errorf("beats = %+v, want only the new beat written", state.Beats)
	}
	if state.TotalSeconds < 240 || state.TotalSeconds > 300 {
		t.Errorf("total = %d, want the sheet brought inside the target once nothing is left unwritten", state.TotalSeconds)
	}
}

func TestGenerateOnlyUnwrittenKeepsApprovedLengths(t *testing.T) {
	f := newBeatFixture()
	f.generate(t)
	u, b := f.beat(t, "b2")
	u.Status = domain.UnitApproved
	if _, err := f.uc.saveBeat(context.Background(), "p1", "b2", b, u); err != nil {
		t.Fatal(err)
	}
	if _, err := f.uc.Add(context.Background(), "p1", "b1", true, ""); err != nil {
		t.Fatal(err)
	}
	f.llm.say(strings.Replace(beatReplyMiddle, `"dur":200`, `"dur":400`, 1))
	if err := f.uc.Generate(context.Background(), "p1", "", true); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if got, _ := f.beat(t, "b2"); got.Status != domain.UnitApproved {
		t.Error("an approved beat must stay approved when others are written")
	}
	if _, b2 := f.beat(t, "b2"); b2.Dur != 200 {
		t.Errorf("approved beat = %d s, want 200 unchanged", b2.Dur)
	}
}

func TestRedoWritesABeatOfTheSkeletonThatHasNoContent(t *testing.T) {
	f := newBeatFixture()
	f.skeleton(t)
	if err := f.uc.ApproveSkeleton(context.Background(), "p1"); err != nil {
		t.Fatal(err)
	}
	f.llm.say(strings.Replace(beatReplyMiddle, `"fn":"Giải thích cơ chế"`, `"fn":"Cao trào"`, 1))
	unit, err := f.uc.BeatUnitKind().Redo(context.Background(), "p1", "b2", "")
	if err != nil {
		t.Fatalf("Redo: %v", err)
	}
	b, _ := readBeat(unit)
	if !b.Written() || b.Fn != "Giải thích cơ chế" || strings.Join(b.Facts, ",") != "F02" || b.Step != "Ong chui qua cửa hẹp." {
		t.Errorf("beat = %+v, want the content written with the skeleton's function, facts and step", b)
	}
	if state, _ := f.uc.State(context.Background(), "p1"); state.Unwritten != 2 || state.Rebalanced != 0 {
		t.Errorf("unwritten=%d rebalanced=%d, want the other two still unwritten and no resizing yet", state.Unwritten, state.Rebalanced)
	}
}

func TestRedoBeforeTheSkeletonIsApprovedIsRefused(t *testing.T) {
	f := newBeatFixture()
	f.skeleton(t)
	if _, err := f.uc.BeatUnitKind().Redo(context.Background(), "p1", "b2", ""); !errors.Is(err, ErrBeatSkeletonNeeded) {
		t.Errorf("err = %v, want the skeleton to be approved first", err)
	}
}

func TestGenerateRefusesBeforeTheStoryIsApproved(t *testing.T) {
	f := newBeatFixture()
	f.open(t)
	f.stories.ok = false
	if err := f.uc.Generate(context.Background(), "p1", "", false); !errors.Is(err, ErrBeatsNeedStory) {
		t.Errorf("err = %v, want the story to be required first", err)
	}
	if len(f.llm.systems) != 0 {
		t.Error("the model must not be called while the step is not open")
	}
}

func TestGenerateKeepsWhatTheCreatorWrote(t *testing.T) {
	f := newBeatFixture()
	f.generate(t)
	if _, err := f.uc.BeatUnitKind().Edit(context.Background(), "p1", "b2", map[string]string{"goal": "Mục tiêu của tôi"}); err != nil {
		t.Fatal(err)
	}
	f.llm.say(beatReplyHook, strings.Replace(beatReplyMiddle, `"goal":"Ong mất cánh"`, `"goal":"AI viết lại"`, 1), beatReplyPayoff)
	if err := f.uc.Generate(context.Background(), "p1", "", false); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	u, b := f.beat(t, "b2")
	if b.Goal != "Mục tiêu của tôi" || !u.HasCreatorField(domain.BeatFieldGoal) {
		t.Errorf("goal = %q mine=%v, want the Creator's goal kept", b.Goal, u.CreatorFields)
	}
	if b.Title != "Cửa hẹp" || u.Status != domain.UnitPending {
		t.Errorf("beat = %+v status=%s, want the rest rewritten and back to review", b, u.Status)
	}
}

func TestEditMarksTheFieldAsTheCreatorsAndSendsTheBeatBackToReview(t *testing.T) {
	f := newBeatFixture()
	f.generate(t)
	kind := f.uc.BeatUnitKind()
	f.save(t, domain.UnitKindBeat, "b1", domain.UnitApproved, domain.Beat{Fn: "Hook", Title: "t", Goal: "g", Content: "c", Facts: []string{}, Dur: 18})
	unit, err := kind.Edit(context.Background(), "p1", "b1", map[string]string{"dur": "23", "facts": "f1, F02", "fn": "Đặt vấn đề"})
	if err != nil {
		t.Fatalf("Edit: %v", err)
	}
	b, _ := readBeat(unit)
	if b.Dur != 23 || strings.Join(b.Facts, ",") != "F01,F02" || b.Fn != "Đặt vấn đề" || unit.Status != domain.UnitPending {
		t.Errorf("beat = %+v status=%s", b, unit.Status)
	}
	if !unit.HasCreatorField("dur") || !unit.HasCreatorField("facts") || !unit.HasCreatorField("fn") || unit.HasCreatorField("goal") {
		t.Errorf("mine = %v, want exactly the fields edited", unit.CreatorFields)
	}
}

func TestEditRefusesWhatABeatCannotHold(t *testing.T) {
	f := newBeatFixture()
	f.generate(t)
	kind := f.uc.BeatUnitKind()
	bad := []map[string]string{
		{"dur": "2"}, {"dur": "abc"}, {"fn": "Mở bài"}, {"facts": "F77"}, {"facts": "xyz"}, {"title": " "}, {"colour": "red"}, {},
	}
	for _, fields := range bad {
		if _, err := kind.Edit(context.Background(), "p1", "b1", fields); !errors.Is(err, domain.ErrInvalidUnitEdit) {
			t.Errorf("Edit(%v) err = %v, want ErrInvalidUnitEdit", fields, err)
		}
	}
	if _, b := f.beat(t, "b1"); b.Dur != 18 {
		t.Error("a refused edit must change nothing")
	}
}

func TestEmptyGoalCannotBeApproved(t *testing.T) {
	f := newBeatFixture()
	f.generate(t)
	kind := f.uc.BeatUnitKind()
	if _, err := kind.Edit(context.Background(), "p1", "b1", map[string]string{"goal": ""}); err != nil {
		t.Fatal(err)
	}
	u, _ := f.units.GetUnit(context.Background(), "p1", domain.UnitKindBeat, "b1")
	if err := kind.CheckApprovable(context.Background(), u); !errors.Is(err, ErrBeatNoGoal) {
		t.Errorf("err = %v, want ErrBeatNoGoal", err)
	}
}

func TestRedoWritesOnlyThatBeatAndReleasesItsCreatorFields(t *testing.T) {
	f := newBeatFixture()
	f.generate(t)
	kind := f.uc.BeatUnitKind()
	if _, err := kind.Edit(context.Background(), "p1", "b2", map[string]string{"title": "Của tôi"}); err != nil {
		t.Fatal(err)
	}
	f.llm.say(`{"beats":[{"key":"b2","fn":"Cao trào","title":"Viết lại","goal":"G","content":"C","facts":["F02"],"dur":40,"next":"N"}]}`)
	unit, err := kind.Redo(context.Background(), "p1", "b2", "ngắn hơn")
	if err != nil {
		t.Fatalf("Redo: %v", err)
	}
	b, _ := readBeat(unit)
	if b.Title != "Viết lại" || unit.RedoCount != 1 || len(unit.CreatorFields) != 0 || unit.Status != domain.UnitPending {
		t.Errorf("beat = %+v redo=%d mine=%v", b, unit.RedoCount, unit.CreatorFields)
	}
	if _, other := f.beat(t, "b1"); other.Title != "Quả sung" {
		t.Error("redoing one beat must not touch the others")
	}
	if !strings.Contains(f.llm.systems[len(f.llm.systems)-1], "ngắn hơn") {
		t.Error("the prompt must carry the Creator's note")
	}
}

func TestAddPutsTheNewBeatAfterTheGivenOneAndNeverReusesAKey(t *testing.T) {
	f := newBeatFixture()
	f.generate(t)
	f.llm.say(`{"beats":[{"key":"","fn":"Cao trào","title":"Mới","goal":"G","content":"C","facts":[],"dur":20,"next":"N"}]}`)
	key, err := f.uc.Add(context.Background(), "p1", "b1", false, "")
	if err != nil || key != "b4" {
		t.Fatalf("Add = %q, %v, want b4", key, err)
	}
	if err := f.uc.Delete(context.Background(), "p1", "b4"); err != nil {
		t.Fatal(err)
	}
	key, err = f.uc.Add(context.Background(), "p1", "b1", true, "")
	if err != nil || key != "b5" {
		t.Fatalf("Add blank = %q, %v, want b5 (b4 is not reused)", key, err)
	}
	state, _ := f.uc.State(context.Background(), "p1")
	order := []string{}
	for _, b := range state.Beats {
		order = append(order, b.Key)
	}
	if strings.Join(order, ",") != "b1,b5,b2,b3" {
		t.Errorf("order = %v, want b1,b5,b2,b3", order)
	}
}

func TestAddBlankWorksWithoutALanguageModel(t *testing.T) {
	f := newBeatFixture()
	f.open(t)
	f.uc.provider = nil
	if _, err := f.uc.Add(context.Background(), "p1", "", false, ""); !errors.Is(err, ErrLLMNotConfigured) {
		t.Errorf("err = %v, want ErrLLMNotConfigured for an AI beat", err)
	}
	key, err := f.uc.Add(context.Background(), "p1", "", true, "")
	if err != nil || key != "b1" {
		t.Fatalf("Add blank = %q, %v, want b1", key, err)
	}
}

func TestAddNeedsAnAnchorOnceTheSheetHasBeats(t *testing.T) {
	f := newBeatFixture()
	f.generate(t)
	if _, err := f.uc.Add(context.Background(), "p1", "", true, ""); !errors.Is(err, domain.ErrInvalidUnitEdit) {
		t.Errorf("err = %v, want the anchor to be required", err)
	}
	if _, err := f.uc.Add(context.Background(), "p1", "b99", true, ""); !errors.Is(err, domain.ErrUnitNotFound) {
		t.Errorf("err = %v, want ErrUnitNotFound", err)
	}
}

func TestReorderChangesOnlyTheOrder(t *testing.T) {
	f := newBeatFixture()
	f.generate(t)
	f.save(t, domain.UnitKindBeat, "b1", domain.UnitApproved, domain.Beat{Fn: "Hook", Title: "t", Goal: "g", Content: "c", Facts: []string{}, Dur: 18})
	if err := f.uc.Reorder(context.Background(), "p1", []string{"b2", "b1", "b3"}); err != nil {
		t.Fatalf("Reorder: %v", err)
	}
	state, _ := f.uc.State(context.Background(), "p1")
	if state.Beats[0].Key != "b2" || state.Beats[1].Key != "b1" || state.Beats[1].Status != domain.UnitApproved {
		t.Errorf("beats = %+v, want the new order with b1 still approved", state.Beats)
	}
	for _, order := range [][]string{{"b1", "b2"}, {"b1", "b1", "b2"}, {"b1", "b2", "b9"}} {
		if err := f.uc.Reorder(context.Background(), "p1", order); !errors.Is(err, domain.ErrInvalidUnitEdit) {
			t.Errorf("Reorder(%v) err = %v, want ErrInvalidUnitEdit", order, err)
		}
	}
}

func TestDeleteNamesTheMustFactNoOtherBeatHolds(t *testing.T) {
	f := newBeatFixture()
	state := f.generate(t)
	if len(state.Beats[0].OnlyMust) != 1 || state.Beats[0].OnlyMust[0] != "F01" || len(state.Beats[1].OnlyMust) != 0 {
		t.Errorf("only_must = %v / %v, want F01 on b1 only", state.Beats[0].OnlyMust, state.Beats[1].OnlyMust)
	}
	if err := f.uc.Delete(context.Background(), "p1", "b1"); err != nil {
		t.Fatal(err)
	}
	state, _ = f.uc.State(context.Background(), "p1")
	if len(state.Beats) != 2 || state.Checks[2].OK {
		t.Errorf("beats=%d checks=%+v, want the must-fact check to fail after the delete", len(state.Beats), state.Checks)
	}
	if err := f.uc.Delete(context.Background(), "p1", "b1"); !errors.Is(err, domain.ErrUnitNotFound) {
		t.Errorf("err = %v, want ErrUnitNotFound for a beat already gone", err)
	}
}

func TestStateIsApprovedOnlyWhenEveryBeatIs(t *testing.T) {
	f := newBeatFixture()
	f.generate(t)
	for _, key := range []string{"b1", "b2"} {
		u, b := f.beat(t, key)
		u.Status = domain.UnitApproved
		if _, err := f.uc.saveBeat(context.Background(), "p1", key, b, u); err != nil {
			t.Fatal(err)
		}
	}
	if state, _ := f.uc.State(context.Background(), "p1"); state.Status != domain.UnitPending {
		t.Error("two of three approved must still wait for review")
	}
	u, b := f.beat(t, "b3")
	u.Status = domain.UnitApproved
	if _, err := f.uc.saveBeat(context.Background(), "p1", "b3", b, u); err != nil {
		t.Fatal(err)
	}
	if state, _ := f.uc.State(context.Background(), "p1"); state.Status != domain.UnitApproved {
		t.Error("every beat approved must approve the sheet")
	}
}

func TestFactDependentsNameTheBeatsThatUseAFact(t *testing.T) {
	f := newBeatFixture()
	f.generate(t)
	items, err := f.uc.FactDependents()(context.Background(), "p1", "F02")
	if err != nil || len(items) != 1 || !strings.Contains(items[0], "B2") {
		t.Errorf("items = %v, err = %v, want B2 named", items, err)
	}
	if items, _ := f.uc.FactDependents()(context.Background(), "p1", "F04"); len(items) != 0 {
		t.Errorf("items = %v, want none for an unused fact", items)
	}
}
