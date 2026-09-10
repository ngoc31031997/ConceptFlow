package application

import (
	"context"
	"strings"
	"testing"

	"orchestrator/internal/domain"
)

type fakeResumer struct {
	calls []string
}

func (f *fakeResumer) ResumeAfterReview(_ context.Context, project *domain.Project) error {
	f.calls = append(f.calls, project.ProjectID)
	return nil
}

func newReviewUseCase() (*ReviewOutlineUseCase, *fakeRepo, *fakeResumer, *fakeProgress, *fakePublisher) {
	repo := newFakeRepo()
	resume := &fakeResumer{}
	progress := &fakeProgress{}
	commands := &fakePublisher{}
	return NewReviewOutlineUseCase(repo, resume, progress, commands, nil), repo, resume, progress, commands
}

func awaitingProject() *domain.Project {
	return &domain.Project{
		ProjectID: "proj-1", SagaID: "saga-1",
		Status: domain.StatusAwaitingReview, TTSEnabled: true, ReviewEnabled: true,
	}
}

func TestApprove_ResumesTheSaga(t *testing.T) {
	uc, repo, resume, _, _ := newReviewUseCase()
	repo.projects["proj-1"] = awaitingProject()

	if err := uc.Approve(context.Background(), "proj-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resume.calls) != 1 {
		t.Fatalf("muốn resume đúng 1 lần, có %d", len(resume.calls))
	}
}

func TestApprove_IsIdempotent(t *testing.T) {
	// FR69.4: bấm duyệt lần thứ hai không được sinh ra một lượt TTS thứ hai.
	// Guard chính là trạng thái — không dựng thêm cơ chế thứ hai chồng lên
	// lớp bảo vệ ở cấp SagaStep vốn đã có.
	uc, repo, resume, _, _ := newReviewUseCase()
	project := awaitingProject()
	repo.projects["proj-1"] = project

	if err := uc.Approve(context.Background(), "proj-1"); err != nil {
		t.Fatalf("lần đầu phải thành công: %v", err)
	}
	project.Status = domain.StatusSynthesizingSpeech // resume thật sẽ đổi trạng thái

	if err := uc.Approve(context.Background(), "proj-1"); err != ErrNotAwaitingReview {
		t.Fatalf("lần hai phải bị từ chối, có %v", err)
	}
	if len(resume.calls) != 1 {
		t.Fatalf("chỉ được resume 1 lần, có %d", len(resume.calls))
	}
}

func TestApprove_RefusesAProjectThatNeverReachedTheGate(t *testing.T) {
	uc, repo, resume, _, _ := newReviewUseCase()
	repo.projects["proj-1"] = &domain.Project{ProjectID: "proj-1", Status: domain.StatusRendering}

	if err := uc.Approve(context.Background(), "proj-1"); err != ErrNotAwaitingReview {
		t.Fatalf("muốn ErrNotAwaitingReview, có %v", err)
	}
	if len(resume.calls) != 0 {
		t.Fatal("không được resume")
	}
}

func TestReject_EndsTheSagaCleanly(t *testing.T) {
	// FR69.3: Saga kết thúc chứ không nằm treo — script sắp thay đổi, nên một
	// saga chờ trên một script không còn tồn tại chỉ là một dòng không ai đóng.
	uc, repo, resume, progress, _ := newReviewUseCase()
	repo.projects["proj-1"] = awaitingProject()

	if err := uc.Reject(context.Background(), "proj-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	project, _ := repo.Get(context.Background(), "proj-1")
	if project.Status != domain.StatusDraft {
		t.Fatalf("muốn draft, có %s", project.Status)
	}
	if len(resume.calls) != 0 {
		t.Fatal("từ chối thì không được chạy tiếp")
	}
	if last := progress.last(); last == nil || last.Status != "rejected" {
		t.Fatalf("phải báo rejected lên GUI: %+v", last)
	}
}

func TestEditNarration_RewritesTheScriptAndRevalidates(t *testing.T) {
	// FR70.2/70.4: sửa phải đi vào SCRIPT (thứ được render), rồi chạy lại
	// validate — một câu sửa xong vẫn có thể vi phạm ngân sách beat hoặc chứa
	// ký hiệu TTS đọc sai.
	uc, repo, _, _, commands := newReviewUseCase()
	project := awaitingProject()
	project.ScriptContent = "class A(ConceptFlowScene):\n    def construct(self):\n        self.narrate(\"câu gốc\")\n"
	project.Scenes = []domain.Scene{{SceneIndex: 0, NarrationText: "câu gốc"}}
	repo.projects["proj-1"] = project

	if err := uc.EditNarration(context.Background(), "proj-1", 0, "câu đã sửa"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, _ := repo.Get(context.Background(), "proj-1")
	if !strings.Contains(updated.ScriptContent, `self.narrate("câu đã sửa")`) {
		t.Fatalf("script chưa đổi: %s", updated.ScriptContent)
	}
	if updated.Status != domain.StatusValidatingScript {
		t.Fatalf("phải chạy lại validate, trạng thái = %s", updated.Status)
	}
	if last := commands.last(); last == nil || last.routingKey != "rendering" {
		t.Fatalf("phải gửi lại lệnh validate_script: %+v", last)
	}
}

func TestEditNarration_RefusesOutsideTheGate(t *testing.T) {
	uc, repo, _, _, commands := newReviewUseCase()
	repo.projects["proj-1"] = &domain.Project{ProjectID: "proj-1", Status: domain.StatusRendering}

	if err := uc.EditNarration(context.Background(), "proj-1", 0, "x"); err != ErrNotAwaitingReview {
		t.Fatalf("muốn ErrNotAwaitingReview, có %v", err)
	}
	if commands.last() != nil {
		t.Fatal("không được gửi lệnh nào")
	}
}
