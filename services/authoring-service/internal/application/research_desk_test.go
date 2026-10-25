package application

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"testing"

	"authoring/internal/domain"
)

// researchStore is an in-memory ResearchStorePort on top of briefUnits.
type researchStore struct {
	*briefUnits
	topic    string
	duration domain.TargetDuration
	models   domain.AuthoringStepModels
	pastes   []PasteLog
	facts    map[string]domain.StoredFact
}

func (s *researchStore) GetAuthoringTopic(context.Context, string) (string, error) {
	return s.topic, nil
}
func (s *researchStore) GetAuthoringModels(context.Context, string) (domain.AuthoringStepModels, error) {
	return s.models, nil
}
func (s *researchStore) GetTargetDuration(context.Context, string) (domain.TargetDuration, error) {
	return s.duration, nil
}
func (s *researchStore) AddPaste(_ context.Context, _ string, p PasteLog) (PasteLog, error) {
	p.N = len(s.pastes) + 1
	s.pastes = append(s.pastes, p)
	return p, nil
}
func (s *researchStore) ListPastes(context.Context, string) ([]PasteLog, error) {
	return append([]PasteLog{}, s.pastes...), nil
}

func (s *researchStore) factKey(projectID, id string) string { return projectID + "|" + id }

func (s *researchStore) ListResearchFacts(_ context.Context, projectID string) ([]domain.StoredFact, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []domain.StoredFact{}
	for k, f := range s.facts {
		if strings.HasPrefix(k, projectID+"|") {
			out = append(out, f)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return len(out[i].ID) < len(out[j].ID) || len(out[i].ID) == len(out[j].ID) && out[i].ID < out[j].ID
	})
	return out, nil
}

func (s *researchStore) GetResearchFact(_ context.Context, projectID, id string) (domain.StoredFact, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, ok := s.facts[s.factKey(projectID, id)]
	if !ok {
		return domain.StoredFact{}, domain.ErrUnitNotFound
	}
	return f, nil
}

func (s *researchStore) SaveResearchFact(_ context.Context, projectID string, f domain.StoredFact) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.facts == nil {
		s.facts = map[string]domain.StoredFact{}
	}
	s.facts[s.factKey(projectID, f.ID)] = f
	return nil
}

// SaveFactLink stores a check result while the fact still has the url and quote
// the check started from, as the real store does.
func (s *researchStore) SaveFactLink(_ context.Context, projectID, key, url, quote string, link domain.FactLink) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, ok := s.facts[s.factKey(projectID, key)]
	if !ok || f.Fact.URL != url || f.Fact.Quote != quote {
		return false, nil
	}
	f.Fact.Link = link
	s.facts[s.factKey(projectID, key)] = f
	return true, nil
}

// approvedSteps is the approved idea and brief of the first two steps.
type approvedSteps struct {
	idea    domain.ProjectIdea
	ideaOK  bool
	brief   domain.ProjectBrief
	briefOK bool
}

func (a *approvedSteps) ApprovedIdea(context.Context, string) (domain.ProjectIdea, bool, error) {
	return a.idea, a.ideaOK, nil
}

func (a *approvedSteps) ApprovedBrief(context.Context, string) (domain.ProjectBrief, bool, error) {
	return a.brief, a.briefOK, nil
}

type researchFixture struct {
	units    *briefUnits
	store    *researchStore
	approved *approvedSteps
	project  *briefProject
	llm      *briefLLM
	uc       *ResearchDeskUseCase
}

const questionsReplyJSON = `{"questions":["Quả sung có phải là hoa không?","Ong nào thụ phấn cho sung?","Ong mẹ có chết trong quả không?"]}`

func newResearchFixture() *researchFixture {
	units := newBriefUnits()
	f := &researchFixture{
		units:    units,
		store:    &researchStore{briefUnits: units, topic: "Vì sao trong quả sung có xác ong?", duration: domain.Duration5to7},
		approved: &approvedSteps{},
		project: &briefProject{
			project: domain.Project{ContentLanguage: domain.LanguageVietnamese, VideoOutputMode: domain.ModeLongOnly},
			status:  domain.StatusDraft,
		},
		llm: &briefLLM{reply: "```json\n" + questionsReplyJSON + "\n```"},
	}
	f.uc = NewResearchDeskUseCase(f.store, f.approved, f.approved, f.project, f.project, briefPrompts{}, f.llm, 4000)
	return f
}

// seedBrief gives the project the idea and brief of the first two steps, both
// approved when status is UnitApproved and both still pending otherwise.
func (f *researchFixture) seedBrief(t *testing.T, status domain.UnitStatus) {
	t.Helper()
	ok := status == domain.UnitApproved
	f.approved.idea = domain.ProjectIdea{
		CentralQuestion: "Vì sao ong chết trong quả sung?", CuriosityPoints: []string{"Ong vào quả bằng đường nào?"}, Scope: "Sung và ong thụ phấn",
	}
	f.approved.brief = domain.ProjectBrief{
		Audience: "Người xem phổ thông", DurationSeconds: 270, Tone: "Tò mò",
		LearningObjectives: []string{"Hiểu vì sao ong chết trong quả"}, Avoid: []string{"Nói ong bị ăn thịt"},
	}
	f.approved.ideaOK, f.approved.briefOK = ok, ok
}

func (f *researchFixture) seedFact(t *testing.T, id string, fact domain.Fact) {
	t.Helper()
	if err := f.store.SaveResearchFact(context.Background(), "p1", domain.StoredFact{ID: id, Fact: fact, CreatorFields: []string{}}); err != nil {
		t.Fatal(err)
	}
}

func questionTexts(qs []QuestionView) []string {
	out := make([]string, len(qs))
	for i, q := range qs {
		out[i] = q.ID + "=" + q.Text
	}
	return out
}

func TestGenerateQuestionsWritesFromTheBriefOnTheResearchModel(t *testing.T) {
	f := newResearchFixture()
	f.seedBrief(t, domain.UnitApproved)
	f.store.models = domain.AuthoringStepModels{Research: "ollama"}

	got, err := f.uc.GenerateQuestions(context.Background(), "p1", false, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0].ID != "Q1" || got[2].ID != "Q3" || got[1].Text != "Ong nào thụ phấn cho sung?" {
		t.Fatalf("questions = %v, want Q1..Q3 from the reply", questionTexts(got))
	}
	for _, q := range got {
		if q.Mine || q.Stale {
			t.Errorf("an AI question is not the Creator's and not stale yet: %+v", q)
		}
	}
	if len(f.llm.models) != 1 || f.llm.models[0] != "ollama" {
		t.Errorf("model used = %v, want the research model", f.llm.models)
	}
	for _, want := range []string{"Vì sao ong chết trong quả sung?", "Ong vào quả bằng đường nào?", "Người xem phổ thông", "Hiểu vì sao ong chết trong quả"} {
		if !strings.Contains(f.llm.systems[0], want) {
			t.Errorf("the prompt must carry the approved idea and brief, lacks %q", want)
		}
	}
	unit, _ := f.units.GetUnit(context.Background(), "p1", domain.UnitKindResearchQuestion, "Q1")
	var stored domain.ResearchQuestion
	if err := json.Unmarshal(unit.Content, &stored); err != nil {
		t.Fatal(err)
	}
	if stored.BriefHash != f.approved.brief.Hash() || unit.Status != domain.UnitPending {
		t.Errorf("a question must record the brief it was built from: %+v", unit)
	}
}

func TestGenerateQuestionsKeepsTheCreatorsAndNeverReusesACode(t *testing.T) {
	f := newResearchFixture()
	f.seedBrief(t, domain.UnitApproved)
	ctx := context.Background()
	if _, err := f.uc.GenerateQuestions(ctx, "p1", false, ""); err != nil {
		t.Fatal(err)
	}
	// The Creator rewrites Q2 and adds a question of their own.
	if _, err := f.uc.QuestionUnitKind().Edit(ctx, "p1", "Q2", map[string]string{"text": "Câu Q2 do tôi viết?"}); err != nil {
		t.Fatal(err)
	}
	added, err := f.uc.AddQuestion(ctx, "p1", "Câu thêm tay?")
	if err != nil || added.ID != "Q4" || !added.Mine {
		t.Fatalf("added = %+v, %v, want Q4 marked mine", added, err)
	}

	f.llm.reply = `{"questions":["Mới một?","Mới hai?"]}`
	got, err := f.uc.GenerateQuestions(ctx, "p1", false, "tập trung vào cơ chế")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"Q2=Câu Q2 do tôi viết?", "Q4=Câu thêm tay?", "Q5=Mới một?", "Q6=Mới hai?"}
	if strings.Join(questionTexts(got), "|") != strings.Join(want, "|") {
		t.Fatalf("questions = %v, want %v", questionTexts(got), want)
	}
	last := f.llm.systems[len(f.llm.systems)-1]
	if !strings.Contains(last, "Câu Q2 do tôi viết?") || !strings.Contains(last, "tập trung vào cơ chế") {
		t.Errorf("the prompt must carry the Creator's questions and note, got %q", last)
	}

	f.llm.reply = `{"questions":["Chỉ một?"]}`
	got, err = f.uc.GenerateQuestions(ctx, "p1", true, "")
	if err != nil || len(got) != 1 || got[0].ID != "Q7" || got[0].Mine {
		t.Fatalf("replaceEdited must replace every question with a new code: %v, %v", questionTexts(got), err)
	}
}

func TestGenerateQuestionsFailsWithoutChangingTheList(t *testing.T) {
	f := newResearchFixture()
	f.seedBrief(t, domain.UnitApproved)
	ctx := context.Background()
	if _, err := f.uc.GenerateQuestions(ctx, "p1", false, ""); err != nil {
		t.Fatal(err)
	}
	f.llm.reply = `{"questions":[]}`
	if _, err := f.uc.GenerateQuestions(ctx, "p1", false, ""); !errors.Is(err, domain.ErrResearchReplyUnusable) {
		t.Fatalf("err = %v, want ErrResearchReplyUnusable", err)
	}
	f.llm.reply, f.llm.err = "", errors.New("model down")
	if _, err := f.uc.GenerateQuestions(ctx, "p1", false, ""); err == nil {
		t.Fatal("a model failure must be an error")
	}
	got, _ := f.uc.questions(ctx, "p1")
	if len(got) != 3 || got[0].ID != "Q1" {
		t.Fatalf("a failed run must leave the questions as they were, got %v", questionTexts(got))
	}
}

func TestGenerateQuestionsPreconditions(t *testing.T) {
	ctx := context.Background()
	f := newResearchFixture()
	if _, err := f.uc.GenerateQuestions(ctx, "p1", false, ""); !errors.Is(err, domain.ErrInvalidStatus) {
		t.Errorf("no brief: err = %v, want ErrInvalidStatus", err)
	}
	f.seedBrief(t, domain.UnitApproved)
	f.project.status = domain.StatusAwaitingReview
	if _, err := f.uc.GenerateQuestions(ctx, "p1", false, ""); !errors.Is(err, domain.ErrInvalidStatus) {
		t.Errorf("locked project: err = %v, want ErrInvalidStatus", err)
	}
	f.project.status = domain.StatusDraft
	f.uc.provider = nil
	if _, err := f.uc.GenerateQuestions(ctx, "p1", false, ""); !errors.Is(err, ErrLLMNotConfigured) {
		t.Errorf("no model: err = %v, want ErrLLMNotConfigured", err)
	}
}

func TestQuestionsGoStaleWhenTheBriefChanges(t *testing.T) {
	f := newResearchFixture()
	f.seedBrief(t, domain.UnitApproved)
	ctx := context.Background()
	if _, err := f.uc.GenerateQuestions(ctx, "p1", false, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := f.uc.QuestionUnitKind().Edit(ctx, "p1", "Q3", map[string]string{"text": "Câu của tôi?"}); err != nil {
		t.Fatal(err)
	}
	if state, _ := f.uc.State(ctx, "p1"); state.QuestionsStale {
		t.Fatal("nothing changed yet")
	}

	f.approved.brief.Audience = "Khán giả mới"
	state, err := f.uc.State(ctx, "p1")
	if err != nil {
		t.Fatal(err)
	}
	if !state.QuestionsStale || !state.Questions[0].Stale || !state.Questions[1].Stale || state.Questions[2].Stale {
		t.Fatalf("AI questions go stale, the Creator's does not: %+v", state.Questions)
	}
}

func TestAddAndDeleteQuestions(t *testing.T) {
	f := newResearchFixture()
	ctx := context.Background()
	if _, err := f.uc.AddQuestion(ctx, "p1", "   "); !errors.Is(err, domain.ErrInvalidUnitEdit) {
		t.Errorf("blank question: err = %v, want ErrInvalidUnitEdit", err)
	}
	q, err := f.uc.AddQuestion(ctx, "p1", "  Câu đầu tiên?  ")
	if err != nil || q.ID != "Q1" || q.Text != "Câu đầu tiên?" {
		t.Fatalf("first question = %+v, %v", q, err)
	}
	if err := f.uc.DeleteQuestion(ctx, "p1", "Q1"); err != nil {
		t.Fatal(err)
	}
	if err := f.uc.DeleteQuestion(ctx, "p1", "Q1"); !errors.Is(err, domain.ErrUnitNotFound) {
		t.Errorf("deleting a missing question: err = %v, want ErrUnitNotFound", err)
	}
	next, err := f.uc.AddQuestion(ctx, "p1", "Câu sau?")
	if err != nil || next.ID != "Q2" {
		t.Fatalf("a deleted code must not be reused, got %+v, %v", next, err)
	}
}

func TestQuestionKindRules(t *testing.T) {
	f := newResearchFixture()
	ctx := context.Background()
	if _, err := f.uc.AddQuestion(ctx, "p1", "Một câu?"); err != nil {
		t.Fatal(err)
	}
	kind := f.uc.QuestionUnitKind()
	if _, err := kind.Edit(ctx, "p1", "Q1", map[string]string{"text": "  "}); !errors.Is(err, domain.ErrInvalidUnitEdit) {
		t.Errorf("blank edit: err = %v", err)
	}
	if _, err := kind.Edit(ctx, "p1", "Q1", map[string]string{"colour": "red"}); !errors.Is(err, domain.ErrInvalidUnitEdit) {
		t.Errorf("unknown field: err = %v", err)
	}
	if _, err := kind.Edit(ctx, "p1", "Q9", map[string]string{"text": "x"}); !errors.Is(err, domain.ErrUnitNotFound) {
		t.Errorf("missing question: err = %v", err)
	}
	if _, err := kind.Redo(ctx, "p1", "Q1", "ghi chú"); !errors.Is(err, domain.ErrInvalidUnitEdit) {
		t.Errorf("redo of one question: err = %v", err)
	}
	if items, err := kind.Impact(ctx, "p1", "Q1"); err != nil || len(items) != 0 {
		t.Errorf("impact = %v, %v, want empty", items, err)
	}
}

func TestResearchPromptCarriesBriefQuestionsAndTemplate(t *testing.T) {
	f := newResearchFixture()
	ctx := context.Background()
	if _, err := f.uc.ResearchPrompt(ctx, "p1"); !errors.Is(err, domain.ErrInvalidStatus) {
		t.Fatalf("no brief: err = %v, want ErrInvalidStatus", err)
	}
	f.seedBrief(t, domain.UnitApproved)
	if _, err := f.uc.ResearchPrompt(ctx, "p1"); !errors.Is(err, domain.ErrInvalidStatus) {
		t.Fatalf("no questions: err = %v, want ErrInvalidStatus", err)
	}
	if _, err := f.uc.GenerateQuestions(ctx, "p1", false, ""); err != nil {
		t.Fatal(err)
	}
	prompt, err := f.uc.ResearchPrompt(ctx, "p1")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"Vì sao trong quả sung có xác ong?", "Vì sao ong chết trong quả sung?", "Người xem phổ thông", "270 giây (4:30)",
		"CÂU HỎI CẦN TRA\nQ1. Quả sung có phải là hoa không?\nQ2. Ong nào thụ phấn cho sung?", domain.ResearchSchema,
	} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt lacks %q", want)
		}
	}
	if strings.Contains(prompt, "{{") || strings.Contains(prompt, "Đánh mã tiếp từ") {
		t.Errorf("the first prompt has no leftover placeholder and no continuation code:\n%s", prompt)
	}
}

func TestFollowUpPromptListsOnlyWeakQuestionsAndBadFacts(t *testing.T) {
	f := newResearchFixture()
	f.seedBrief(t, domain.UnitApproved)
	ctx := context.Background()
	if _, err := f.uc.GenerateQuestions(ctx, "p1", false, ""); err != nil {
		t.Fatal(err)
	}
	okLink := domain.FactLink{State: domain.LinkOK}
	f.seedFact(t, "F01", domain.Fact{Question: "Q1", ClaimVI: "Một", Type: domain.FactTypeFact, URL: "https://a.org/1", Keep: true, Link: okLink})
	f.seedFact(t, "F02", domain.Fact{Question: "Q1", ClaimVI: "Hai", Type: domain.FactTypeFact, URL: "https://b.org/1", Keep: true, Link: okLink})
	f.seedFact(t, "F03", domain.Fact{Question: "Q2", ClaimVI: "Ba bị sai đoạn trích", Type: domain.FactTypeFact, URL: "https://c.org/1", Keep: true, Link: domain.FactLink{State: domain.LinkBad}})

	prompt, err := f.uc.FollowUpPrompt(ctx, "p1")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(prompt, "CÂU HỎI CẦN TRA THÊM") || !strings.Contains(prompt, "Q2. Ong nào") || !strings.Contains(prompt, "Q3. Ong mẹ") {
		t.Errorf("weak questions Q2 and Q3 must be listed:\n%s", prompt)
	}
	if strings.Contains(prompt, "Q1. ") {
		t.Errorf("Q1 has two independent sources and must not be listed:\n%s", prompt)
	}
	if !strings.Contains(prompt, "DỮ KIỆN CẦN NGUỒN KHÁC\nF03. Ba bị sai đoạn trích") || !strings.Contains(prompt, "Đánh mã tiếp từ F04") {
		t.Errorf("the bad fact and the next code must be in the prompt:\n%s", prompt)
	}

	// Every question well sourced: nothing to follow up.
	for _, q := range []string{"Q2", "Q3"} {
		for i, host := range []string{"x", "y"} {
			f.seedFact(t, "F1"+q[1:]+string(rune('0'+i)), domain.Fact{
				Question: q, ClaimVI: "ok", Type: domain.FactTypeFact, URL: "https://" + host + q + ".org/", Keep: true, Link: okLink,
			})
		}
	}
	f.seedFact(t, "F03", domain.Fact{Question: "Q2", ClaimVI: "Ba đã sửa", Type: domain.FactTypeFact, URL: "https://c.org/1", Keep: true, Link: okLink})
	if _, err := f.uc.FollowUpPrompt(ctx, "p1"); !errors.Is(err, domain.ErrInvalidStatus) {
		t.Fatalf("all sourced: err = %v, want ErrInvalidStatus", err)
	}
}

func TestStateStartsEmptyAndPending(t *testing.T) {
	f := newResearchFixture()
	state, err := f.uc.State(context.Background(), "p1")
	if err != nil {
		t.Fatal(err)
	}
	if state.Gate.Status != domain.UnitPending || state.Gate.Skipped || state.Gate.Resolved || state.BriefApproved {
		t.Errorf("a new project's research is pending and unresolved: %+v", state)
	}
	if len(state.Questions) != 0 || len(state.Facts) != 0 || len(state.Pastes) != 0 || len(state.Coverage) != 0 {
		t.Errorf("nothing stored yet: %+v", state)
	}
}

func TestBriefImpactMentionsAIQuestions(t *testing.T) {
	f := newResearchFixture()
	f.seedBrief(t, domain.UnitApproved)
	brief := NewCreativeBriefUseCase(f.units, &briefStore{}, f.project, f.project, briefPrompts{}, nil, 4000)
	ctx := context.Background()
	items, err := brief.Impact(ctx, "p1", domain.BriefUnitKey)
	if err != nil || len(items) != 0 {
		t.Fatalf("no questions yet: items = %v, err = %v", items, err)
	}
	if _, err := f.uc.AddQuestion(ctx, "p1", "Câu của tôi?"); err != nil {
		t.Fatal(err)
	}
	if items, _ := brief.Impact(ctx, "p1", domain.BriefUnitKey); len(items) != 0 {
		t.Fatalf("the Creator's own question is not sent back by a brief change: %v", items)
	}
	if _, err := f.uc.GenerateQuestions(ctx, "p1", false, ""); err != nil {
		t.Fatal(err)
	}
	items, _ = brief.Impact(ctx, "p1", domain.BriefUnitKey)
	if len(items) != 1 || !strings.Contains(items[0], "Câu hỏi nghiên cứu") {
		t.Fatalf("items = %v, want the research questions listed", items)
	}
}

// The state carries the approved idea and brief the research is planned from, so
// the screen can show them before the AI is called; none while they are not approved.
func TestStateCarriesTheApprovedIdeaAndBrief(t *testing.T) {
	f := newResearchFixture()
	ctx := context.Background()
	f.seedBrief(t, domain.UnitPending)
	state, err := f.uc.State(ctx, "p1")
	if err != nil || state.Idea != nil || state.Brief != nil || state.BriefApproved {
		t.Fatalf("unapproved: idea %v, brief %v, approved %v, err %v, want none", state.Idea, state.Brief, state.BriefApproved, err)
	}
	f.seedBrief(t, domain.UnitApproved)
	state, err = f.uc.State(ctx, "p1")
	if err != nil || state.Idea == nil || state.Brief == nil || !state.BriefApproved {
		t.Fatalf("approved: %+v, %v, want the idea and brief", state, err)
	}
	if state.Idea.CentralQuestion != "Vì sao ong chết trong quả sung?" || state.Brief.DurationSeconds != 270 {
		t.Errorf("state carries %+v and %+v, want the seeded ones", state.Idea, state.Brief)
	}
}
