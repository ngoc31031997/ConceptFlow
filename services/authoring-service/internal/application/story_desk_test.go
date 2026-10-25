package application

import (
	"context"
	"errors"
	"strings"
	"testing"

	"authoring/internal/domain"
)

// storyStore is the research store plus the project's story row.
type storyStore struct {
	*researchStore
	story *domain.StoredStory
}

func (s *storyStore) GetProjectStory(context.Context, string) (domain.StoredStory, error) {
	if s.story == nil {
		return domain.StoredStory{}, domain.ErrStoryNotFound
	}
	return *s.story, nil
}

func (s *storyStore) SaveProjectStory(_ context.Context, story domain.StoredStory) error {
	s.story = &story
	return nil
}

// standing is where the research step stands.
type standing struct{ resolved, skipped bool }

func (s *standing) ResearchStanding(context.Context, string) (bool, bool, error) {
	return s.resolved, s.skipped, nil
}

type storyFixture struct {
	store    *storyStore
	approved *approvedSteps
	standing *standing
	project  *briefProject
	llm      *briefLLM
	uc       *StoryDeskUseCase
}

const storyReplyFull = `{"central_question":"Vì sao ong chết trong quả sung?","core_answer":"Ong là cái giá của một quan hệ.","narrative_arc":[
{"beat":"Hook","text":"Mở từ đĩa sung có xác ong.","facts":[]},
{"beat":"Cơ chế","text":"Sung là chùm hoa kín, ong chui vào và mất cánh.","facts":["F01","F02"]},
{"beat":"Payoff","text":"Ficin phân giải xác ong.","facts":["F03"]}]}`

func newStoryFixture() *storyFixture {
	units := newBriefUnits()
	f := &storyFixture{
		store:    &storyStore{researchStore: &researchStore{briefUnits: units, topic: "Vì sao trong quả sung có xác ong?", duration: domain.Duration5to7}},
		approved: &approvedSteps{},
		standing: &standing{},
		project:  &briefProject{project: domain.Project{ContentLanguage: domain.LanguageVietnamese, VideoOutputMode: domain.ModeLongOnly}, status: domain.StatusDraft},
		llm:      &briefLLM{reply: storyReplyFull},
	}
	f.uc = NewStoryDeskUseCase(f.store, f.approved, f.standing, f.project, f.project, briefPrompts{}, f.llm, 4000)
	return f
}

// open approves the brief and the research step and seeds three kept facts and a dropped one.
func (f *storyFixture) open(t *testing.T) {
	t.Helper()
	f.approved.brief = domain.ProjectBrief{
		Audience: "Học sinh cấp 3", DurationSeconds: 360, Tone: "Ấm, tò mò",
		LearningObjectives: []string{"Hiểu quan hệ sung và ong"}, Avoid: []string{"Thuật ngữ nặng"},
	}
	f.approved.briefOK = true
	f.standing.resolved = true
	for _, fact := range []domain.StoredFact{
		{ID: "F01", Fact: domain.Fact{ClaimVI: "Sung là chùm hoa kín.", Type: domain.FactTypeFact, Keep: true, Must: true, Link: domain.FactLink{State: domain.LinkOK}}},
		{ID: "F02", Fact: domain.Fact{ClaimVI: "Ong mẹ chết trong quả.", Type: domain.FactTypeFact, Keep: true, Link: domain.FactLink{State: domain.LinkOK}}},
		{ID: "F03", Fact: domain.Fact{ClaimVI: "Ficin phân giải xác ong.", Type: domain.FactTypeHypothesis, Keep: true, Link: domain.FactLink{State: domain.LinkOK}}},
		{ID: "F04", Fact: domain.Fact{ClaimVI: "Đã bỏ.", Type: domain.FactTypeFact, Keep: false}},
	} {
		if err := f.store.SaveResearchFact(context.Background(), "p1", fact); err != nil {
			t.Fatal(err)
		}
	}
}

func (f *storyFixture) generate(t *testing.T, note string) domain.StoredStory {
	t.Helper()
	s, err := f.uc.Generate(context.Background(), "p1", note)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	return s
}

func TestGenerateWritesTheStoryFromTheBriefAndTheKeptFacts(t *testing.T) {
	f := newStoryFixture()
	f.open(t)
	s := f.generate(t, "")
	if s.Story.CentralQuestion != "Vì sao ong chết trong quả sung?" || len(s.Story.NarrativeArc) != 3 {
		t.Fatalf("story = %+v, want the question and the three beats of the reply", s.Story)
	}
	if s.Status != domain.StoryPending || s.BriefHash != f.approved.brief.Hash() || s.ResearchHash == "" {
		t.Errorf("status=%s briefHash=%q researchHash=%q, want a pending story stamped with its inputs", s.Status, s.BriefHash, s.ResearchHash)
	}
	if f.store.story == nil || f.store.story.Story.CoreAnswer != s.Story.CoreAnswer {
		t.Error("the story was not stored")
	}
	prompt := f.llm.systems[0]
	for _, want := range []string{"Học sinh cấp 3", "Hiểu quan hệ sung và ong", "F01 [", "Ficin phân giải xác ong.", "tiếng Việt"} {
		if !strings.Contains(prompt, want) {
			t.Errorf("the prompt must carry %q", want)
		}
	}
	if strings.Contains(prompt, "Đã bỏ.") {
		t.Error("the prompt must not carry a dropped fact")
	}
	if strings.Contains(prompt, "{{") {
		t.Errorf("the prompt has an unfilled placeholder: %s", prompt)
	}
}

func TestGenerateKeepsTheFieldsTheCreatorWrote(t *testing.T) {
	f := newStoryFixture()
	f.open(t)
	if _, err := f.uc.Edit(context.Background(), "p1", StoryPatch{CentralQuestion: ptr("Câu hỏi của tôi?")}); err != nil {
		t.Fatalf("Edit: %v", err)
	}
	s := f.generate(t, "kể ngắn hơn")
	if s.Story.CentralQuestion != "Câu hỏi của tôi?" {
		t.Errorf("question = %q, want the Creator's kept", s.Story.CentralQuestion)
	}
	if s.Story.CoreAnswer != "Ong là cái giá của một quan hệ." {
		t.Errorf("answer = %q, want the AI's", s.Story.CoreAnswer)
	}
	prompt := f.llm.systems[0]
	if !strings.Contains(prompt, "Câu hỏi của tôi?") || !strings.Contains(prompt, "kể ngắn hơn") {
		t.Error("the prompt must carry the kept field and the note")
	}
}

func TestGenerateRefusesWhenEveryFieldIsTheCreators(t *testing.T) {
	f := newStoryFixture()
	f.open(t)
	arc := []domain.ArcBeat{{Beat: "A", Text: "a"}, {Beat: "B", Text: "b"}, {Beat: "C", Text: "c"}}
	if _, err := f.uc.Edit(context.Background(), "p1", StoryPatch{CentralQuestion: ptr("Q"), CoreAnswer: ptr("A"), NarrativeArc: &arc}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.uc.Generate(context.Background(), "p1", ""); !errors.Is(err, ErrStoryNothingToWrite) {
		t.Errorf("err = %v, want nothing left for the AI to write", err)
	}
	if len(f.llm.systems) != 0 {
		t.Error("the model must not be called")
	}
}

func TestGenerateWaitsForTheBriefAndTheResearchStep(t *testing.T) {
	f := newStoryFixture()
	f.open(t)
	f.approved.briefOK = false
	if _, err := f.uc.Generate(context.Background(), "p1", ""); !errors.Is(err, ErrStoryNeedsBrief) || !errors.Is(err, domain.ErrInvalidStatus) {
		t.Errorf("err = %v, want the brief to be required", err)
	}
	f.approved.briefOK, f.standing.resolved = true, false
	if _, err := f.uc.Generate(context.Background(), "p1", ""); !errors.Is(err, ErrStoryNeedsResearch) {
		t.Errorf("err = %v, want the research step to be required", err)
	}
	if len(f.llm.systems) != 0 {
		t.Error("the model must not be called while the step is closed")
	}
}

func TestGenerateWithSkippedResearchWritesWithoutFacts(t *testing.T) {
	f := newStoryFixture()
	f.open(t)
	f.store.researchStore.facts = nil
	f.standing.skipped = true
	f.llm.reply = `{"central_question":"Q","core_answer":"A","narrative_arc":[{"beat":"H","text":"1","facts":[]},{"beat":"M","text":"2","facts":[]},{"beat":"P","text":"3","facts":[]}]}`
	s := f.generate(t, "")
	if len(s.Story.CitedFacts()) != 0 {
		t.Errorf("cited = %v, want none", s.Story.CitedFacts())
	}
	if !strings.Contains(f.llm.systems[0], "bước Nghiên cứu đã bỏ qua") {
		t.Error("the prompt must say the research step was skipped")
	}
	f.llm.reply = `{"central_question":"Q","core_answer":"A","narrative_arc":[{"beat":"H","text":"1","facts":[]},{"beat":"M","text":"2","facts":[]},{"beat":"P","text":"3","facts":["F01"]}]}`
	if _, err := f.uc.Generate(context.Background(), "p1", ""); !errors.Is(err, domain.ErrStoryReplyUnusable) {
		t.Errorf("err = %v, want a citation with no fact table refused", err)
	}
}

func TestGenerateStoresNothingFromAnUnusableReply(t *testing.T) {
	f := newStoryFixture()
	f.open(t)
	for name, reply := range map[string]string{
		"no fields":    `{}`,
		"unknown fact": `{"central_question":"Q","core_answer":"A","narrative_arc":[{"beat":"H","text":"1","facts":["F09"]},{"beat":"M","text":"2","facts":[]},{"beat":"P","text":"3","facts":[]}]}`,
		"dropped fact": `{"central_question":"Q","core_answer":"A","narrative_arc":[{"beat":"H","text":"1","facts":["F04"]},{"beat":"M","text":"2","facts":[]},{"beat":"P","text":"3","facts":[]}]}`,
		"two beats":    `{"central_question":"Q","core_answer":"A","narrative_arc":[{"beat":"H","text":"1","facts":[]},{"beat":"P","text":"3","facts":[]}]}`,
	} {
		f.llm.reply = reply
		if _, err := f.uc.Generate(context.Background(), "p1", ""); !errors.Is(err, domain.ErrStoryReplyUnusable) {
			t.Errorf("%s: err = %v, want ErrStoryReplyUnusable", name, err)
		}
		if f.store.story != nil {
			t.Errorf("%s: a story was stored from an unusable reply", name)
		}
	}
}

func TestEditBuildsAStoryByHandWithoutTheAI(t *testing.T) {
	f := newStoryFixture()
	f.open(t)
	arc := []domain.ArcBeat{{Beat: " Hook ", Text: "Mở đầu", Facts: []string{"f1"}}, {Beat: "Giữa", Text: "Giữa"}, {Beat: "Payoff", Text: "Kết"}}
	s, err := f.uc.Edit(context.Background(), "p1", StoryPatch{CentralQuestion: ptr("Q"), CoreAnswer: ptr("A"), NarrativeArc: &arc})
	if err != nil {
		t.Fatalf("Edit: %v", err)
	}
	if s.Story.NarrativeArc[0].Beat != "Hook" || strings.Join(s.Story.NarrativeArc[0].Facts, ",") != "F01" {
		t.Errorf("beat 1 = %+v, want it cleaned with the code in canonical form", s.Story.NarrativeArc[0])
	}
	if len(s.CreatorFields) != 3 || s.Status != domain.StoryPending || s.BriefHash == "" {
		t.Errorf("creator=%v status=%s briefHash=%q, want three owned fields on a pending story stamped with the brief", s.CreatorFields, s.Status, s.BriefHash)
	}
	if len(f.llm.systems) != 0 {
		t.Error("an edit must not call the model")
	}
}

func TestEditRefusesAFactNotInTheTable(t *testing.T) {
	f := newStoryFixture()
	f.open(t)
	for _, code := range []string{"F09", "F04", "xyz"} {
		arc := []domain.ArcBeat{{Beat: "A", Text: "a", Facts: []string{code}}}
		if _, err := f.uc.Edit(context.Background(), "p1", StoryPatch{NarrativeArc: &arc}); !errors.Is(err, domain.ErrInvalidUnitEdit) {
			t.Errorf("%s: err = %v, want the edit refused", code, err)
		}
	}
	if f.store.story != nil {
		t.Error("a refused edit stored a story")
	}
}

func TestEditRefusesTooManyBeats(t *testing.T) {
	f := newStoryFixture()
	f.open(t)
	var arc []domain.ArcBeat
	for i := 0; i <= domain.MaxArcBeats; i++ {
		arc = append(arc, domain.ArcBeat{Beat: "N", Text: "t"})
	}
	if _, err := f.uc.Edit(context.Background(), "p1", StoryPatch{NarrativeArc: &arc}); !errors.Is(err, domain.ErrInvalidUnitEdit) {
		t.Errorf("err = %v, want the arc over the limit refused", err)
	}
}

func TestEditSendsAnApprovedStoryBackAndEmptyingReturnsTheFieldToTheAI(t *testing.T) {
	f := newStoryFixture()
	f.open(t)
	f.generate(t, "")
	if _, err := f.uc.Approve(context.Background(), "p1"); err != nil {
		t.Fatalf("Approve: %v", err)
	}
	s, err := f.uc.Edit(context.Background(), "p1", StoryPatch{CoreAnswer: ptr("Đáp án khác")})
	if err != nil || s.Status != domain.StoryPending || !s.HasCreatorField(domain.StoryFieldCoreAnswer) {
		t.Fatalf("status=%s creator=%v err=%v, want a pending story with the answer owned", s.Status, s.CreatorFields, err)
	}
	s, err = f.uc.Edit(context.Background(), "p1", StoryPatch{CoreAnswer: ptr("")})
	if err != nil || s.HasCreatorField(domain.StoryFieldCoreAnswer) {
		t.Errorf("creator=%v err=%v, want the emptied answer back with the AI", s.CreatorFields, err)
	}
}

func TestEditWithNothingToSaveIsRefused(t *testing.T) {
	f := newStoryFixture()
	if _, err := f.uc.Edit(context.Background(), "p1", StoryPatch{}); !errors.Is(err, domain.ErrInvalidUnitEdit) {
		t.Errorf("err = %v, want an empty edit refused", err)
	}
}

func TestApproveNeedsAValidStoryAndApprovedStoryIsServedToLaterSteps(t *testing.T) {
	f := newStoryFixture()
	f.open(t)
	if _, err := f.uc.Approve(context.Background(), "p1"); !errors.Is(err, domain.ErrStoryNotFound) {
		t.Errorf("err = %v, want no story to approve", err)
	}
	if _, err := f.uc.Edit(context.Background(), "p1", StoryPatch{CentralQuestion: ptr("Q")}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.uc.Approve(context.Background(), "p1"); !errors.Is(err, domain.ErrStoryInvalid) {
		t.Errorf("err = %v, want a story with empty fields refused", err)
	}
	f.generate(t, "")
	if _, _, ok, _ := f.uc.ApprovedStory(context.Background(), "p1"); ok {
		t.Error("a pending story is served as approved")
	}
	if _, err := f.uc.Approve(context.Background(), "p1"); err != nil {
		t.Fatalf("Approve: %v", err)
	}
	story, hash, ok, err := f.uc.ApprovedStory(context.Background(), "p1")
	if err != nil || !ok || hash == "" || len(story.NarrativeArc) != 3 {
		t.Errorf("ok=%v hash=%q err=%v story=%+v, want the approved story", ok, hash, err, story)
	}
	if _, err := f.uc.Unapprove(context.Background(), "p1"); err != nil {
		t.Fatal(err)
	}
	if _, _, ok, _ := f.uc.ApprovedStory(context.Background(), "p1"); ok {
		t.Error("an unapproved story is served as approved")
	}
}

func TestStateShowsCitedFactsAndWhetherTheInputsChanged(t *testing.T) {
	f := newStoryFixture()
	f.open(t)
	state, err := f.uc.State(context.Background(), "p1")
	if err != nil || state.Story != nil || len(state.Facts) != 3 || state.Brief == nil || !state.ResearchResolved {
		t.Fatalf("state = %+v err=%v, want no story, three facts and the brief", state, err)
	}
	f.generate(t, "")
	state, _ = f.uc.State(context.Background(), "p1")
	cited := map[string]bool{}
	for _, v := range state.Facts {
		cited[v.ID] = v.Cited
	}
	if !cited["F01"] || !cited["F02"] || !cited["F03"] {
		t.Errorf("cited = %v, want the facts the arc rests on", cited)
	}
	if state.BriefStale || state.ResearchStale {
		t.Errorf("stale = %v/%v, want a fresh story", state.BriefStale, state.ResearchStale)
	}
	f.approved.brief.Tone = "Giọng khác"
	fact, _ := f.store.GetResearchFact(context.Background(), "p1", "F02")
	fact.Fact.ClaimVI = "Ong mẹ sống sót."
	_ = f.store.SaveResearchFact(context.Background(), "p1", fact)
	state, _ = f.uc.State(context.Background(), "p1")
	if !state.BriefStale || !state.ResearchStale {
		t.Errorf("stale = %v/%v, want both inputs reported changed", state.BriefStale, state.ResearchStale)
	}
}

func TestFactDependentsNameTheBeatsThatRestOnAFact(t *testing.T) {
	f := newStoryFixture()
	f.open(t)
	find := f.uc.FactDependents()
	if items, err := find(context.Background(), "p1", "F02"); err != nil || len(items) != 0 {
		t.Fatalf("items=%v err=%v, want none before a story exists", items, err)
	}
	f.generate(t, "")
	items, err := find(context.Background(), "p1", "F02")
	if err != nil || len(items) != 1 || !strings.Contains(items[0], "Cơ chế") {
		t.Errorf("items=%v err=%v, want the beat «Cơ chế» named", items, err)
	}
	if items, _ := find(context.Background(), "p1", "F04"); len(items) != 0 {
		t.Errorf("items=%v, want none for a fact no beat cites", items)
	}
}

func ptr[T any](v T) *T { return &v }
