package application

import (
	"context"
	"errors"
	"strings"
	"testing"

	"authoring/internal/domain"
)

const projectBriefReply = "```json\n" + `{"audience":"Khán giả phổ thông từ cấp 2 trở lên.","duration_seconds":270,"tone":"Tò mò, dễ hiểu, chính xác.","learning_objectives":["Giải thích chân trời sự kiện ở mức khái niệm.","Hiểu lực thủy triều phụ thuộc vào điều kiện vật lý."],"avoid":["Mô tả chân trời sự kiện như một lớp vỏ cứng."]}` + "\n```"

const briefReviewReply = `{"audience_fit":{"rating":"good","comment":"Rõ."},"duration_fit":{"rating":"fair","comment":"Hơi dài."},"objectives":{"rating":"good","comment":"Cụ thể."},"avoid_list":{"rating":"good","comment":"Đúng chỗ."},"unclear":[],"suggestions":["Thêm một dòng về kích thước hố đen."]}`

type briefStoreFake struct {
	brief   *domain.StoredBrief
	review  *domain.BriefReview
	saveErr error
}

func (s *briefStoreFake) GetProjectBrief(context.Context, string) (domain.StoredBrief, error) {
	if s.brief == nil {
		return domain.StoredBrief{}, domain.ErrBriefNotFound
	}
	return *s.brief, nil
}
func (s *briefStoreFake) SaveProjectBrief(_ context.Context, b domain.StoredBrief) error {
	if s.saveErr != nil {
		return s.saveErr
	}
	s.brief = &b
	return nil
}
func (s *briefStoreFake) GetBriefReview(context.Context, string) (domain.BriefReview, error) {
	if s.review == nil {
		return domain.BriefReview{}, domain.ErrBriefNotFound
	}
	return *s.review, nil
}
func (s *briefStoreFake) SaveBriefReview(_ context.Context, _ string, r domain.BriefReview) error {
	s.review = &r
	return nil
}

type approvedIdeaFake struct {
	idea domain.ProjectIdea
	ok   bool
}

func (f *approvedIdeaFake) ApprovedIdea(context.Context, string) (domain.ProjectIdea, bool, error) {
	return f.idea, f.ok, nil
}

type projectBriefFixture struct {
	store   *briefStoreFake
	ideas   *approvedIdeaFake
	input   *ideaInputFake
	models  *briefStore
	project *briefProject
	llm     *briefLLM
	uc      *ProjectBriefUseCase
}

func newProjectBriefFixture() *projectBriefFixture {
	f := &projectBriefFixture{
		store: &briefStoreFake{},
		ideas: &approvedIdeaFake{ok: true, idea: domain.ProjectIdea{
			CentralQuestion: "Người rơi thấy gì, người ở xa thấy gì?",
			CuriosityPoints: []string{"Chân trời sự kiện là gì?", "Cơ thể có bị kéo dài không?"},
			Scope:           "Trực quan, không đi sâu vào toán.",
		}},
		input:   &ideaInputFake{in: IdeaInput{Subject: "Chuyện gì xảy ra nếu ta rơi vào hố đen?", Language: domain.LanguageVietnamese}},
		models:  &briefStore{},
		project: &briefProject{status: domain.StatusDraft, project: domain.Project{VideoOutputMode: domain.ModeLongOnly}},
		llm:     &briefLLM{reply: projectBriefReply},
	}
	f.uc = NewProjectBriefUseCase(f.store, f.ideas, f.input, f.project, f.models, f.project, briefPrompts{}, f.llm, nil, 4000)
	return f
}

func TestProjectBriefGenerateWritesTheBriefForApproval(t *testing.T) {
	f := newProjectBriefFixture()
	got, err := f.uc.Generate(context.Background(), "p1", "")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.BriefPending || got.RedoCount != 0 {
		t.Errorf("a first brief waits for approval with no redo, got %q redo %d", got.Status, got.RedoCount)
	}
	if got.Brief.DurationSeconds != 270 || len(got.Brief.LearningObjectives) != 2 || len(got.Brief.Avoid) != 1 {
		t.Errorf("brief stored wrong: %+v", got.Brief)
	}
	for _, want := range []string{"Người rơi thấy gì, người ở xa thấy gì?", "Chân trời sự kiện là gì?", "Trực quan, không đi sâu vào toán.", "121 đến 1800 giây", "tiếng Việt"} {
		if !strings.Contains(f.llm.systems[0], want) {
			t.Errorf("the prompt lacks %q", want)
		}
	}
	if got.IdeaHash != f.ideas.idea.Hash() || got.Language != "vi" {
		t.Errorf("the brief does not remember what it was written from: %+v", got)
	}
}

func TestProjectBriefGenerateAsksForTheShortRangeOnAShort(t *testing.T) {
	f := newProjectBriefFixture()
	f.project.project.VideoOutputMode = domain.ModeShortOnly
	f.llm.reply = strings.Replace(projectBriefReply, "270", "90", 1)
	got, err := f.uc.Generate(context.Background(), "p1", "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(f.llm.systems[0], "60 đến 120 giây") || got.Brief.DurationSeconds != 90 {
		t.Errorf("short range not used: prompt range or duration wrong (%d)", got.Brief.DurationSeconds)
	}
}

func TestProjectBriefGenerateRefusesBeforeTheIdeaIsApproved(t *testing.T) {
	f := newProjectBriefFixture()
	f.ideas.ok = false
	_, err := f.uc.Generate(context.Background(), "p1", "")
	if !errors.Is(err, domain.ErrBriefNeedsApprovedIdea) || !errors.Is(err, domain.ErrInvalidStatus) {
		t.Fatalf("err = %v, want ErrBriefNeedsApprovedIdea", err)
	}
	if len(f.llm.systems) != 0 {
		t.Error("the AI was called without an approved idea")
	}
}

func TestProjectBriefGenerateStoresNothingFromAnUnusableReply(t *testing.T) {
	for name, reply := range map[string]string{
		"missing field":      `{"audience":"a"}`,
		"duration too short": strings.Replace(projectBriefReply, "270", "30", 1),
	} {
		f := newProjectBriefFixture()
		f.llm.reply = reply
		if _, err := f.uc.Generate(context.Background(), "p1", ""); !errors.Is(err, domain.ErrProjectBriefReplyUnusable) {
			t.Fatalf("%s: err = %v", name, err)
		}
		if f.store.brief != nil {
			t.Errorf("%s: a half-written brief was stored", name)
		}
	}
}

func TestProjectBriefNeedsAModel(t *testing.T) {
	f := newProjectBriefFixture()
	f.uc.provider = nil
	if _, err := f.uc.Generate(context.Background(), "p1", ""); !errors.Is(err, ErrLLMNotConfigured) {
		t.Fatalf("err = %v", err)
	}
}

func TestProjectBriefRunsOnTheBriefModelSlot(t *testing.T) {
	f := newProjectBriefFixture()
	f.models.models = domain.AuthoringStepModels{Brief: "ollama"}
	if _, err := f.uc.Generate(context.Background(), "p1", ""); err != nil {
		t.Fatal(err)
	}
	if len(f.llm.models) != 1 || f.llm.models[0] != "ollama" {
		t.Errorf("models used = %v, want [ollama]", f.llm.models)
	}
}

func TestProjectBriefRedoKeepsWhatTheCreatorWrote(t *testing.T) {
	f := newProjectBriefFixture()
	ctx := context.Background()
	if _, err := f.uc.Generate(ctx, "p1", ""); err != nil {
		t.Fatal(err)
	}
	tone := "Giọng của Creator"
	if _, err := f.uc.Edit(ctx, "p1", BriefPatch{Tone: &tone}); err != nil {
		t.Fatal(err)
	}
	got, err := f.uc.Redo(ctx, "p1", "ngắn hơn")
	if err != nil {
		t.Fatal(err)
	}
	if got.Brief.Tone != tone {
		t.Errorf("tone = %q, the AI overwrote the Creator's text", got.Brief.Tone)
	}
	if got.RedoCount != 1 || !strings.Contains(f.llm.systems[1], "ngắn hơn") || !strings.Contains(f.llm.systems[1], tone) {
		t.Errorf("redo count %d or prompt lacks the note and the kept field", got.RedoCount)
	}
}

func TestProjectBriefEmptyingAFieldGivesItBackToTheAI(t *testing.T) {
	f := newProjectBriefFixture()
	ctx := context.Background()
	if _, err := f.uc.Generate(ctx, "p1", ""); err != nil {
		t.Fatal(err)
	}
	tone := "Của Creator"
	if _, err := f.uc.Edit(ctx, "p1", BriefPatch{Tone: &tone}); err != nil {
		t.Fatal(err)
	}
	empty := ""
	got, err := f.uc.Edit(ctx, "p1", BriefPatch{Tone: &empty})
	if err != nil {
		t.Fatal(err)
	}
	if got.HasCreatorField(domain.BriefFieldTone) {
		t.Error("an emptied field is still the Creator's")
	}
}

func TestProjectBriefEditRefusesTooManyLines(t *testing.T) {
	f := newProjectBriefFixture()
	var lines []string
	for i := 0; i <= domain.MaxLearningObjectives; i++ {
		lines = append(lines, strings.Repeat("m", i+1))
	}
	_, err := f.uc.Edit(context.Background(), "p1", BriefPatch{LearningObjectives: &lines})
	if !errors.Is(err, domain.ErrInvalidUnitEdit) {
		t.Fatalf("err = %v, want ErrInvalidUnitEdit", err)
	}
}

func TestProjectBriefApproveNeedsAValidBriefAndEditUndoesIt(t *testing.T) {
	f := newProjectBriefFixture()
	ctx := context.Background()
	if _, err := f.uc.Generate(ctx, "p1", ""); err != nil {
		t.Fatal(err)
	}
	approved, err := f.uc.Approve(ctx, "p1")
	if err != nil || approved.Status != domain.BriefApproved || approved.ApprovedAt == nil {
		t.Fatalf("approve: %+v err %v", approved, err)
	}
	if _, ok, _ := f.uc.ApprovedBrief(ctx, "p1"); !ok {
		t.Error("an approved brief is not offered to later steps")
	}
	audience := "Khán giả khác"
	edited, err := f.uc.Edit(ctx, "p1", BriefPatch{Audience: &audience})
	if err != nil || edited.Status != domain.BriefPending {
		t.Fatalf("edit after approval: %+v err %v", edited, err)
	}
	if _, ok, _ := f.uc.ApprovedBrief(ctx, "p1"); ok {
		t.Error("an edited brief is still offered as approved")
	}
	tooShort := 10
	if _, err := f.uc.Edit(ctx, "p1", BriefPatch{DurationSeconds: &tooShort}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.uc.Approve(ctx, "p1"); !errors.Is(err, domain.ErrProjectBriefInvalid) {
		t.Errorf("approve with 10 s: err = %v, want ErrProjectBriefInvalid", err)
	}
}

func TestProjectBriefStateFlagsAnIdeaThatChanged(t *testing.T) {
	f := newProjectBriefFixture()
	ctx := context.Background()
	if _, err := f.uc.Generate(ctx, "p1", ""); err != nil {
		t.Fatal(err)
	}
	state, err := f.uc.State(ctx, "p1")
	if err != nil || state.IdeaChanged {
		t.Fatalf("fresh brief flagged: changed=%v err=%v", state.IdeaChanged, err)
	}
	f.ideas.idea.Scope = "Phạm vi đã đổi."
	if state, _ = f.uc.State(ctx, "p1"); !state.IdeaChanged {
		t.Error("an idea changed after the brief was written is not flagged")
	}
	if state.MinDurationSeconds != 121 || state.MaxDurationSeconds != 1800 {
		t.Errorf("duration range = %d–%d", state.MinDurationSeconds, state.MaxDurationSeconds)
	}
}

func TestProjectBriefReviewIsAdviceAndGoesStaleOnEdit(t *testing.T) {
	f := newProjectBriefFixture()
	ctx := context.Background()
	if _, err := f.uc.Generate(ctx, "p1", ""); err != nil {
		t.Fatal(err)
	}
	f.llm.reply = briefReviewReply
	review, err := f.uc.Review(ctx, "p1")
	if err != nil {
		t.Fatal(err)
	}
	if review.DurationFit.Rating != domain.RatingFair || review.BriefHash == "" {
		t.Errorf("review read wrong: %+v", review)
	}
	if f.store.brief.Status != domain.BriefPending {
		t.Error("the review changed the brief's status")
	}
	if state, _ := f.uc.State(ctx, "p1"); state.ReviewStale {
		t.Error("a review of the current brief is stale")
	}
	tone := "Đổi giọng"
	if _, err := f.uc.Edit(ctx, "p1", BriefPatch{Tone: &tone}); err != nil {
		t.Fatal(err)
	}
	if state, _ := f.uc.State(ctx, "p1"); !state.ReviewStale {
		t.Error("a brief edited after the review does not mark it stale")
	}
}
