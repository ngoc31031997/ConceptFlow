// Package application — cổng duyệt dàn ý (CR-024).
package application

import (
	"context"
	"fmt"
	"log/slog"

	"orchestrator/internal/domain"
)

// resumer is the slice of HandleStepEventUseCase this package needs to restart
// a paused saga. Declared as an interface so the approve use case can be tested
// without standing up the whole event handler.
type resumer interface {
	ResumeAfterReview(ctx context.Context, project *domain.Project) error
}

// ReviewOutlineUseCase implements the two ways out of `awaiting_review`
// (CR-024 FR69.2/FR69.3).
//
// Until this existed the Render Saga ran start to finish with no place to stop,
// so the first time the Creator saw what a video says was when they watched the
// finished thing — after every cost had already been paid.
type ReviewOutlineUseCase struct {
	repo     domain.ProjectRepositoryPort
	resume   resumer
	progress domain.ProgressPublisherPort
	commands domain.CommandPublisherPort
	logger   *slog.Logger
}

func NewReviewOutlineUseCase(
	repo domain.ProjectRepositoryPort,
	resume resumer,
	progress domain.ProgressPublisherPort,
	commands domain.CommandPublisherPort,
	logger *slog.Logger,
) *ReviewOutlineUseCase {
	if logger == nil {
		logger = slog.Default()
	}
	return &ReviewOutlineUseCase{repo: repo, resume: resume, progress: progress, commands: commands, logger: logger}
}

// ErrNotAwaitingReview is returned when the project is not at the gate.
//
// It is deliberately the same answer for "already approved" as for "never got
// there": approving twice must not start a second run of TTS (FR69.4), and the
// status check is the guard that makes that true. No second mechanism is added
// on top of the step-level one that already exists.
var ErrNotAwaitingReview = fmt.Errorf("project is not awaiting review")

// Approve releases the saga: narration is synthesized, and the rest follows.
func (uc *ReviewOutlineUseCase) Approve(ctx context.Context, projectID string) error {
	project, err := uc.repo.Get(ctx, projectID)
	if err != nil {
		return err
	}
	if project == nil || project.Status != domain.StatusAwaitingReview {
		return ErrNotAwaitingReview
	}

	uc.logger.Info("outline approved", "project_id", projectID, "saga_id", project.SagaID)
	return uc.resume.ResumeAfterReview(ctx, project)
}

// EditNarration rewrites one narration line and re-runs validation (FR70).
//
// The edit goes back into the *script*, not into a copy of the outline: the
// script is what gets rendered, so an outline that drifts from it would be a
// review of something that never ships. Re-validating afterwards is not
// optional either (FR70.4) — a corrected line can still bust a beat budget or
// contain a symbol TTS will read wrong.
func (uc *ReviewOutlineUseCase) EditNarration(ctx context.Context, projectID string, sceneIndex int, newText string) error {
	project, err := uc.repo.Get(ctx, projectID)
	if err != nil {
		return err
	}
	if project == nil || project.Status != domain.StatusAwaitingReview {
		return ErrNotAwaitingReview
	}

	var current string
	for _, scene := range project.Scenes {
		if scene.SceneIndex == sceneIndex {
			current = scene.NarrationText
		}
	}
	if current == "" {
		return domain.ErrNarrationNotEditable
	}

	updated, err := domain.ReplaceNarrationLiteral(project.ScriptContent, current, newText)
	if err != nil {
		return err
	}

	project.ScriptContent = updated
	if err := uc.repo.Save(ctx, project); err != nil {
		return err
	}

	uc.logger.Info("narration edited at review", "project_id", projectID, "scene_index", sceneIndex)
	return uc.revalidate(ctx, project)
}

// revalidate sends the script back through the validation pass so the outline
// the Creator approves is the outline the edited script actually produces.
func (uc *ReviewOutlineUseCase) revalidate(ctx context.Context, project *domain.Project) error {
	if err := uc.repo.UpdateStep(ctx, &domain.SagaStep{
		SagaID: project.SagaID, StepName: domain.StepValidateScript, Status: domain.SagaStepInProgress,
	}); err != nil {
		return err
	}
	if err := uc.repo.UpdateStatus(ctx, project.ProjectID, domain.StatusValidatingScript); err != nil {
		return err
	}
	return uc.commands.PublishCommand(ctx, "rendering", domain.Envelope{
		MessageID: newUUID(),
		SagaID:    project.SagaID,
		ProjectID: project.ProjectID,
		EventType: string(domain.StepValidateScript),
		Payload: map[string]interface{}{
			"script_content":   project.ScriptContent,
			"scene_class_name": project.ManimSceneClassName,
			"render_quality":   string(project.RenderQuality),
		},
	})
}

// Reject sends the project back to the Creator to edit (FR69.3).
//
// The saga ends here rather than hanging: a rejected outline means the script
// is going to change, and a saga waiting on a script that no longer exists is
// just a row nobody will ever close.
func (uc *ReviewOutlineUseCase) Reject(ctx context.Context, projectID string) error {
	project, err := uc.repo.Get(ctx, projectID)
	if err != nil {
		return err
	}
	if project == nil || project.Status != domain.StatusAwaitingReview {
		return ErrNotAwaitingReview
	}

	if err := uc.repo.UpdateStatus(ctx, projectID, domain.StatusDraft); err != nil {
		return err
	}
	uc.logger.Info("outline rejected", "project_id", projectID, "saga_id", project.SagaID)

	return uc.progress.PublishProgress(ctx, domain.ProgressMessage{
		ProjectID: projectID,
		Step:      string(domain.StepValidateScript),
		Status:    "rejected",
	})
}
