package application

import (
	"context"
	"errors"
	"strings"
	"testing"

	"authoring/internal/domain"
)

const projectIdeaReply = "```json\n" + `{"central_question":"Người rơi thấy gì, người ở xa thấy gì?","curiosity_points":["Chân trời sự kiện là gì?","Cơ thể có bị kéo dài không?"],"scope":"Trực quan, không đi sâu vào toán."}` + "\n```"

const ideaReviewReply = `{"clarity":{"rating":"good","comment":"Câu hỏi rõ."},"researchability":{"rating":"fair","comment":"Cần nguồn chuyên môn."},"scope_fit":{"rating":"good","comment":"Vừa một video."},"unclear":["Khán giả nào?"],"suggestions":["Nêu rõ hố đen sao hay siêu nặng."]}`

type ideaStore struct {
	idea    *domain.StoredIdea
	review  *domain.IdeaReview
	saveErr error
}

func (s *ideaStore) GetProjectIdea(context.Context, string) (domain.StoredIdea, error) {
	if s.idea == nil {
		return domain.StoredIdea{}, domain.ErrIdeaNotFound
	}
	return *s.idea, nil
}
func (s *ideaStore) SaveProjectIdea(_ context.Context, i domain.StoredIdea) error {
	if s.saveErr != nil {
		return s.saveErr
	}
	s.idea = &i
	return nil
}
func (s *ideaStore) GetIdeaReview(context.Context, string) (domain.IdeaReview, error) {
	if s.review == nil {
		return domain.IdeaReview{}, domain.ErrIdeaNotFound
	}
	return *s.review, nil
}
func (s *ideaStore) SaveIdeaReview(_ context.Context, _ string, r domain.IdeaReview) error {
	s.review = &r
	return nil
}

type ideaInputFake struct{ in IdeaInput }

func (f *ideaInputFake) IdeaInput(context.Context, string) (IdeaInput, error) { return f.in, nil }

type projectIdeaFixture struct {
	store   *ideaStore
	input   *ideaInputFake
	models  *briefStore
	project *briefProject
	llm     *briefLLM
	uc      *ProjectIdeaUseCase
}

func newProjectIdeaFixture() *projectIdeaFixture {
	f := &projectIdeaFixture{
		store:   &ideaStore{},
		input:   &ideaInputFake{in: IdeaInput{Subject: "Chuyện gì xảy ra nếu ta rơi vào hố đen?", Language: domain.LanguageVietnamese}},
		models:  &briefStore{},
		project: &briefProject{status: domain.StatusDraft},
		llm:     &briefLLM{reply: projectIdeaReply},
	}
	f.uc = NewProjectIdeaUseCase(f.store, f.input, f.models, f.project, briefPrompts{}, f.llm, 4000)
	return f
}

func TestProjectIdeaGenerateWritesTheIdeaForApproval(t *testing.T) {
	f := newProjectIdeaFixture()
	got, err := f.uc.Generate(context.Background(), "p1", "")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.IdeaPending || got.RedoCount != 0 {
		t.Errorf("a first idea waits for approval with no redo, got %q redo %d", got.Status, got.RedoCount)
	}
	if got.Idea.CentralQuestion != "Người rơi thấy gì, người ở xa thấy gì?" || len(got.Idea.CuriosityPoints) != 2 {
		t.Errorf("idea stored wrong: %+v", got.Idea)
	}
	for _, want := range []string{"Chuyện gì xảy ra nếu ta rơi vào hố đen?", "tiếng Việt"} {
		if !strings.Contains(f.llm.systems[0], want) {
			t.Errorf("the prompt lacks %q", want)
		}
	}
	if got.Language != "vi" || got.InputHash != domain.HashInput(f.input.in.Subject) {
		t.Errorf("the idea does not remember what it was written from: %+v", got)
	}
}

func TestProjectIdeaGenerateWritesInTheChosenLanguage(t *testing.T) {
	f := newProjectIdeaFixture()
	f.input.in.Language = domain.LanguageEnglish
	if _, err := f.uc.Generate(context.Background(), "p1", ""); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(f.llm.systems[0], "English") {
		t.Error("the prompt does not ask for English")
	}
}

func TestProjectIdeaGenerateRefusesWithoutText(t *testing.T) {
	f := newProjectIdeaFixture()
	f.input.in.Subject = "  "
	_, err := f.uc.Generate(context.Background(), "p1", "")
	if !errors.Is(err, domain.ErrInvalidWizardInput) {
		t.Fatalf("err = %v, want ErrInvalidWizardInput", err)
	}
	if len(f.llm.systems) != 0 {
		t.Error("the AI was called with nothing to write from")
	}
}

func TestProjectIdeaGenerateStoresNothingFromAnUnusableReply(t *testing.T) {
	f := newProjectIdeaFixture()
	f.llm.reply = `{"central_question":"a?"}`
	_, err := f.uc.Generate(context.Background(), "p1", "")
	if !errors.Is(err, domain.ErrProjectIdeaReplyUnusable) {
		t.Fatalf("err = %v", err)
	}
	if f.store.idea != nil {
		t.Error("a half-written idea was stored")
	}
}

func TestProjectIdeaNeedsAModel(t *testing.T) {
	f := newProjectIdeaFixture()
	f.uc.provider = nil
	if _, err := f.uc.Generate(context.Background(), "p1", ""); !errors.Is(err, ErrLLMNotConfigured) {
		t.Fatalf("err = %v", err)
	}
}

func TestProjectIdeaRunsOnTheBriefModelSlot(t *testing.T) {
	f := newProjectIdeaFixture()
	f.models.models = domain.AuthoringStepModels{Brief: "ollama"}
	if _, err := f.uc.Generate(context.Background(), "p1", ""); err != nil {
		t.Fatal(err)
	}
	if f.llm.models[0] != "ollama" {
		t.Errorf("model = %q", f.llm.models[0])
	}
}

func TestProjectIdeaRedoKeepsTheCreatorsFields(t *testing.T) {
	f := newProjectIdeaFixture()
	ctx := context.Background()
	if _, err := f.uc.Generate(ctx, "p1", ""); err != nil {
		t.Fatal(err)
	}
	mine := "Phạm vi của tôi"
	if _, err := f.uc.Edit(ctx, "p1", IdeaPatch{Scope: &mine}); err != nil {
		t.Fatal(err)
	}
	got, err := f.uc.Redo(ctx, "p1", "ngắn hơn")
	if err != nil {
		t.Fatal(err)
	}
	if got.Idea.Scope != mine {
		t.Errorf("the Creator's scope was overwritten: %q", got.Idea.Scope)
	}
	if got.RedoCount != 1 {
		t.Errorf("redo count = %d, want 1", got.RedoCount)
	}
	last := f.llm.systems[len(f.llm.systems)-1]
	if !strings.Contains(last, "ngắn hơn") || !strings.Contains(last, mine) {
		t.Error("the redo prompt lacks the note or the kept field")
	}
}

func TestProjectIdeaEditSendsAnApprovedIdeaBackToApproval(t *testing.T) {
	f := newProjectIdeaFixture()
	ctx := context.Background()
	if _, err := f.uc.Generate(ctx, "p1", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := f.uc.Approve(ctx, "p1"); err != nil {
		t.Fatal(err)
	}
	same := f.store.idea.Idea.Scope
	if got, _ := f.uc.Edit(ctx, "p1", IdeaPatch{Scope: &same}); got.Status != domain.IdeaApproved {
		t.Error("saving an unchanged field withdrew the approval")
	}
	changed := "Phạm vi khác"
	got, err := f.uc.Edit(ctx, "p1", IdeaPatch{Scope: &changed})
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.IdeaPending || got.ApprovedAt != nil {
		t.Errorf("an edited idea must wait for approval again, got %q", got.Status)
	}
	if _, ok, _ := f.uc.ApprovedIdea(ctx, "p1"); ok {
		t.Error("later steps can still read an idea that is no longer approved")
	}
}

func TestProjectIdeaEditWithoutAModelCreatesTheIdea(t *testing.T) {
	f := newProjectIdeaFixture()
	q, scope := "Rơi vào thì sao?", "Không toán."
	points := []string{"Chân trời sự kiện?"}
	got, err := f.uc.Edit(context.Background(), "p1", IdeaPatch{CentralQuestion: &q, CuriosityPoints: &points, Scope: &scope})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.CreatorFields) != 3 || got.Status != domain.IdeaPending {
		t.Errorf("hand-written idea = %+v", got)
	}
	if _, err := f.uc.Edit(context.Background(), "p1", IdeaPatch{}); !errors.Is(err, domain.ErrInvalidUnitEdit) {
		t.Errorf("an empty edit: err = %v", err)
	}
}

func TestProjectIdeaApproveNeedsAValidIdea(t *testing.T) {
	f := newProjectIdeaFixture()
	ctx := context.Background()
	if _, err := f.uc.Approve(ctx, "p1"); !errors.Is(err, domain.ErrIdeaNotFound) {
		t.Fatalf("approving nothing: err = %v", err)
	}
	q := "Rơi vào thì sao?"
	if _, err := f.uc.Edit(ctx, "p1", IdeaPatch{CentralQuestion: &q}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.uc.Approve(ctx, "p1"); !errors.Is(err, domain.ErrProjectIdeaInvalid) {
		t.Fatalf("approving a half-filled idea: err = %v", err)
	}
}

func TestProjectIdeaApproveAndUnapprove(t *testing.T) {
	f := newProjectIdeaFixture()
	ctx := context.Background()
	if _, err := f.uc.Generate(ctx, "p1", ""); err != nil {
		t.Fatal(err)
	}
	got, err := f.uc.Approve(ctx, "p1")
	if err != nil || got.Status != domain.IdeaApproved || got.ApprovedAt == nil {
		t.Fatalf("approve: %+v %v", got, err)
	}
	idea, ok, _ := f.uc.ApprovedIdea(ctx, "p1")
	if !ok || idea.CentralQuestion == "" {
		t.Error("the approved idea cannot be read by later steps")
	}
	if got, err := f.uc.Unapprove(ctx, "p1"); err != nil || got.Status != domain.IdeaPending {
		t.Fatalf("unapprove: %+v %v", got, err)
	}
}

func TestProjectIdeaIsLockedOnceRenderStarted(t *testing.T) {
	f := newProjectIdeaFixture()
	f.project.status = domain.StatusRendering
	q := "x?"
	if _, err := f.uc.Edit(context.Background(), "p1", IdeaPatch{CentralQuestion: &q}); !errors.Is(err, domain.ErrInvalidStatus) {
		t.Errorf("edit: err = %v", err)
	}
	if _, err := f.uc.Generate(context.Background(), "p1", ""); !errors.Is(err, domain.ErrInvalidStatus) {
		t.Errorf("generate: err = %v", err)
	}
}

func TestProjectIdeaStateFlagsChangedInputAndStaleReview(t *testing.T) {
	f := newProjectIdeaFixture()
	ctx := context.Background()
	if _, err := f.uc.Generate(ctx, "p1", ""); err != nil {
		t.Fatal(err)
	}
	f.llm.reply = ideaReviewReply
	if _, err := f.uc.Review(ctx, "p1"); err != nil {
		t.Fatal(err)
	}
	state, _ := f.uc.State(ctx, "p1")
	if state.InputChanged || state.ReviewStale {
		t.Fatalf("fresh idea flagged: %+v", state)
	}
	f.input.in.Language = domain.LanguageEnglish
	if state, _ = f.uc.State(ctx, "p1"); !state.InputChanged {
		t.Error("a changed language is not flagged")
	}
	f.input.in.Language = domain.LanguageVietnamese
	f.input.in.Subject = "Một ý tưởng khác"
	if state, _ = f.uc.State(ctx, "p1"); !state.InputChanged {
		t.Error("changed text is not flagged")
	}
	scope := "Phạm vi mới"
	if _, err := f.uc.Edit(ctx, "p1", IdeaPatch{Scope: &scope}); err != nil {
		t.Fatal(err)
	}
	if state, _ = f.uc.State(ctx, "p1"); !state.ReviewStale {
		t.Error("a review of an older idea is not flagged stale")
	}
}

func TestProjectIdeaReviewIsAdviceOnly(t *testing.T) {
	f := newProjectIdeaFixture()
	ctx := context.Background()
	if _, err := f.uc.Generate(ctx, "p1", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := f.uc.Approve(ctx, "p1"); err != nil {
		t.Fatal(err)
	}
	f.llm.reply = ideaReviewReply
	review, err := f.uc.Review(ctx, "p1")
	if err != nil {
		t.Fatal(err)
	}
	if review.ScopeFit.Rating != domain.RatingGood || review.IdeaHash != f.store.idea.Idea.Hash() {
		t.Errorf("review = %+v", review)
	}
	if f.store.idea.Status != domain.IdeaApproved {
		t.Error("the review changed the approval")
	}
	f.llm.reply = `{"clarity":{"rating":"weak","comment":"x"}}`
	if _, err := f.uc.Review(ctx, "p1"); !errors.Is(err, domain.ErrIdeaReviewReplyUnusable) {
		t.Errorf("unusable review: err = %v", err)
	}
	if f.store.review.Clarity.Rating != domain.RatingGood {
		t.Error("an unusable review replaced the stored one")
	}
}
