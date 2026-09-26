package application

import (
	"context"
	"errors"
	"testing"

	"orchestrator/internal/domain"
)

type fakeControl struct {
	sent []domain.CancelRequest
	err  error
}

func (f *fakeControl) PublishCancel(_ context.Context, r domain.CancelRequest) error {
	if f.err != nil {
		return f.err
	}
	f.sent = append(f.sent, r)
	return nil
}

type cancelRepo struct {
	domain.ProjectRepositoryPort
	project *domain.Project
	steps   []domain.SagaStep
	status  domain.ProjectStatus
}

func (r *cancelRepo) Get(context.Context, string) (*domain.Project, error) { return r.project, nil }
func (r *cancelRepo) UpdateStep(_ context.Context, s *domain.SagaStep) error {
	r.steps = append(r.steps, *s)
	return nil
}
func (r *cancelRepo) UpdateStatus(_ context.Context, _ string, s domain.ProjectStatus) error {
	r.status = s
	return nil
}

func TestCancelStepStopsWhereItIs(t *testing.T) {
	repo := &cancelRepo{project: &domain.Project{ProjectID: "p", SagaID: "s", Status: domain.StatusRendering}}
	ctl := &fakeControl{}
	out, err := NewCancelStepUseCase(repo, ctl).Execute(context.Background(), "p")
	if err != nil {
		t.Fatal(err)
	}
	if out.Step != domain.StepRenderScenes || repo.status != domain.StatusFailedRenderScenes {
		t.Errorf("out %+v status %s", out, repo.status)
	}
	if len(ctl.sent) != 1 || ctl.sent[0].ProjectID != "p" || ctl.sent[0].Step != "render_scenes" {
		t.Errorf("control %+v", ctl.sent)
	}
	if len(repo.steps) != 1 || repo.steps[0].Status != domain.SagaStepFailed || *repo.steps[0].ErrorMessage != domain.CancelledErrorMessage {
		t.Errorf("steps %+v", repo.steps)
	}
	if got := domain.RunStateOf(repo.status, repo.steps[0].ErrorMessage); got != domain.RunCancelled {
		t.Errorf("run state %s, want cancelled", got)
	}
}

func TestCancelStepRejectsAProjectNothingIsRunningFor(t *testing.T) {
	for _, st := range []domain.ProjectStatus{domain.StatusDraft, domain.StatusAwaitingReview, domain.StatusReadyToPublish, domain.StatusFailedRenderScenes} {
		repo := &cancelRepo{project: &domain.Project{Status: st}}
		if _, err := NewCancelStepUseCase(repo, &fakeControl{}).Execute(context.Background(), "p"); !errors.Is(err, domain.ErrInvalidStatus) {
			t.Errorf("%s: got %v, want ErrInvalidStatus", st, err)
		}
	}
}

func TestCancelStepLeavesProjectUntouchedWhenBroadcastFails(t *testing.T) {
	repo := &cancelRepo{project: &domain.Project{Status: domain.StatusRendering}}
	if _, err := NewCancelStepUseCase(repo, &fakeControl{err: errors.New("broker down")}).Execute(context.Background(), "p"); err == nil {
		t.Fatal("want the broker error")
	}
	if repo.status != "" || len(repo.steps) != 0 {
		t.Errorf("project changed despite failed broadcast: status %q steps %v", repo.status, repo.steps)
	}
}
