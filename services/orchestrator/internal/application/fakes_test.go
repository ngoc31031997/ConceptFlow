package application

import (
	"context"
	"fmt"

	"orchestrator/internal/domain"
)

// fakeRepo is a hand-written in-memory implementation of
// domain.ProjectRepositoryPort for unit-testing use cases without a real
// Postgres instance (dependency-injection.md "Testability").
type fakeRepo struct {
	projects map[string]*domain.Project
	steps    map[string]*domain.SagaStep // key: sagaID+"/"+stepName
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{projects: map[string]*domain.Project{}, steps: map[string]*domain.SagaStep{}}
}

func stepKey(sagaID string, stepName domain.StepName) string {
	return sagaID + "/" + string(stepName)
}

func (f *fakeRepo) Get(_ context.Context, projectID string) (*domain.Project, error) {
	p, ok := f.projects[projectID]
	if !ok {
		return nil, domain.ErrProjectNotFound
	}
	cp := *p
	return &cp, nil
}

func (f *fakeRepo) Save(_ context.Context, project *domain.Project) error {
	cp := *project
	f.projects[project.ProjectID] = &cp
	return nil
}

func (f *fakeRepo) List(_ context.Context) ([]domain.ProjectSummary, error) {
	summaries := make([]domain.ProjectSummary, 0, len(f.projects))
	for _, p := range f.projects {
		summaries = append(summaries, domain.ProjectSummary{
			ProjectID:    p.ProjectID,
			Status:       p.Status,
			VideoPath:    p.VideoPath,
			ErrorMessage: p.ErrorMessage,
		})
	}
	return summaries, nil
}

func (f *fakeRepo) Delete(_ context.Context, projectID string) error {
	if _, ok := f.projects[projectID]; !ok {
		return domain.ErrProjectNotFound
	}
	delete(f.projects, projectID)
	return nil
}

func (f *fakeRepo) UpdateStatus(_ context.Context, projectID string, status domain.ProjectStatus) error {
	p, ok := f.projects[projectID]
	if !ok {
		return domain.ErrProjectNotFound
	}
	p.Status = status
	return nil
}

func (f *fakeRepo) GetStep(_ context.Context, sagaID string, stepName domain.StepName) (*domain.SagaStep, error) {
	s, ok := f.steps[stepKey(sagaID, stepName)]
	if !ok {
		return nil, domain.ErrSagaStepNotFound
	}
	cp := *s
	return &cp, nil
}

func (f *fakeRepo) UpdateStep(_ context.Context, step *domain.SagaStep) error {
	cp := *step
	f.steps[stepKey(step.SagaID, step.StepName)] = &cp
	return nil
}

// fakePublisher is an in-memory domain.CommandPublisherPort recording every
// published command for assertions.
type fakePublisher struct {
	published []publishedCommand
	failNext  bool
}

type publishedCommand struct {
	routingKey string
	envelope   domain.Envelope
}

func (f *fakePublisher) PublishCommand(_ context.Context, routingKey string, envelope domain.Envelope) error {
	if f.failNext {
		f.failNext = false
		return fmt.Errorf("simulated publish failure")
	}
	f.published = append(f.published, publishedCommand{routingKey: routingKey, envelope: envelope})
	return nil
}

func (f *fakePublisher) last() *publishedCommand {
	if len(f.published) == 0 {
		return nil
	}
	return &f.published[len(f.published)-1]
}

// fakeProgress is an in-memory domain.ProgressPublisherPort recording every
// published ProgressMessage.
type fakeProgress struct {
	messages []domain.ProgressMessage
}

func (f *fakeProgress) PublishProgress(_ context.Context, msg domain.ProgressMessage) error {
	f.messages = append(f.messages, msg)
	return nil
}

func (f *fakeProgress) last() *domain.ProgressMessage {
	if len(f.messages) == 0 {
		return nil
	}
	return &f.messages[len(f.messages)-1]
}
