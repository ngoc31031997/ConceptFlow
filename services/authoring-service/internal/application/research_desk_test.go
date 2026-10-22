package application

import (
	"context"
	"encoding/json"
	"errors"
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

// SaveFactLink stores a check result while the fact still has the url and quote
// the check started from, as the real store does.
func (s *researchStore) SaveFactLink(_ context.Context, projectID, key, url, quote string, link domain.FactLink) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := projectID + "|" + domain.UnitKindFact + "|" + key
	unit, ok := s.units[k]
	if !ok {
		return false, nil
	}
	var fact domain.Fact
	if err := json.Unmarshal(unit.Content, &fact); err != nil {
		return false, err
	}
	if fact.URL != url || fact.Quote != quote {
		return false, nil
	}
	fact.Link = link
	unit.Content, _ = json.Marshal(fact)
	s.units[k] = unit
	return true, nil
}

type researchFixture struct {
	units   *briefUnits
	store   *researchStore
	project *briefProject
	llm     *briefLLM
	uc      *ResearchDeskUseCase
}

const questionsReplyJSON = `{"questions":["Quả sung có phải là hoa không?","Ong nào thụ phấn cho sung?","Ong mẹ có chết trong quả không?"]}`

func newResearchFixture() *researchFixture {
	units := newBriefUnits()
	f := &researchFixture{
		units: units,
		store: &researchStore{briefUnits: units, topic: "Vì sao trong quả sung có xác ong?", duration: domain.Duration5to7},
		project: &briefProject{
			project: domain.Project{ContentLanguage: domain.LanguageVietnamese, VideoOutputMode: domain.ModeLongOnly},
			status:  domain.StatusDraft,
		},
		llm: &briefLLM{reply: "```json\n" + questionsReplyJSON + "\n```"},
	}
	f.uc = NewResearchDeskUseCase(f.store, f.project, f.project, briefPrompts{}, f.llm, nil, 4000)
	return f
}

func (f *researchFixture) seedBrief(t *testing.T, status domain.UnitStatus) {
	t.Helper()
	ideaRaw, _ := json.Marshal(domain.Idea{Question: "Vì sao ong chết trong quả sung?", Why: "w", Known: "k", Unknown: "u"})
	if _, err := f.units.SaveUnit(context.Background(), domain.ReviewUnit{
		ProjectID: "p1", Kind: domain.UnitKindIdea, Key: domain.IdeaUnitKey, Status: status, Content: ideaRaw,
	}); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(domain.CreativeBrief{
		Audience: "Người xem phổ thông", Know: "k", Understand: "u", Feel: "f", Tone: "t", OutOfScope: "o",
	})
	if _, err := f.units.SaveUnit(context.Background(), domain.ReviewUnit{
		ProjectID: "p1", Kind: domain.UnitKindBrief, Key: domain.BriefUnitKey, Status: status, Content: raw,
	}); err != nil {
		t.Fatal(err)
	}
}

func (f *researchFixture) seedFact(t *testing.T, id string, fact domain.Fact) {
	t.Helper()
	raw, _ := json.Marshal(fact)
	if _, err := f.units.SaveUnit(context.Background(), domain.ReviewUnit{
		ProjectID: "p1", Kind: domain.UnitKindFact, Key: id, Status: domain.UnitPending, Content: raw,
	}); err != nil {
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
	if !strings.Contains(f.llm.systems[0], "Vì sao ong chết trong quả sung?") || !strings.Contains(f.llm.systems[0], "Vì sao trong quả sung có xác ong?") {
		t.Errorf("the prompt must carry the brief and the topic, got %q", f.llm.systems[0])
	}
	unit, _ := f.units.GetUnit(context.Background(), "p1", domain.UnitKindResearchQuestion, "Q1")
	if unit.InputRef != "brief:main" || unit.InputSHA == "" || unit.Status != domain.UnitPending {
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
	f.llm.reply = "xin lỗi, tôi không biết"
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

	changed, _ := json.Marshal(domain.CreativeBrief{Audience: "Khán giả mới"})
	brief, _ := f.units.GetUnit(ctx, "p1", domain.UnitKindBrief, domain.BriefUnitKey)
	brief.Content = changed
	if _, err := f.units.SaveUnit(ctx, brief); err != nil {
		t.Fatal(err)
	}
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
		"Vì sao trong quả sung có xác ong?", "Vì sao ong chết trong quả sung?", "Người xem phổ thông", "5–7 phút",
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
	brief := NewCreativeBriefUseCase(f.units, &briefStore{}, f.project, f.project, briefPrompts{}, nil, nil, 4000)
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
