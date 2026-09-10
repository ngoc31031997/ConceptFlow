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
	projects    map[string]*domain.Project
	steps       map[string]*domain.SagaStep // key: sagaID+"/"+stepName
	calibration map[string]domain.VoiceCalibration
	formats     map[string]domain.VideoFormat
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		projects:    map[string]*domain.Project{},
		steps:       map[string]*domain.SagaStep{},
		calibration: map[string]domain.VoiceCalibration{},
		formats:     map[string]domain.VideoFormat{},
	}
}

func (r *fakeRepo) RecordVoiceSamples(_ context.Context, voiceID string, words int, seconds float64) error {
	if voiceID == "" || words <= 0 || seconds <= 0 {
		return nil
	}
	c := r.calibration[voiceID]
	c.VoiceID = voiceID
	c.SampleCount++
	c.TotalWords += words
	c.TotalSecond += seconds
	r.calibration[voiceID] = c
	return nil
}

func (r *fakeRepo) GetVoiceCalibration(_ context.Context, voiceID string) (domain.VoiceCalibration, error) {
	if c, ok := r.calibration[voiceID]; ok {
		return c, nil
	}
	return domain.VoiceCalibration{VoiceID: voiceID}, nil
}

func (r *fakeRepo) SeedVideoFormats(_ context.Context) error { return nil }

func (r *fakeRepo) GetVideoFormat(_ context.Context, formatID string, _ int) (domain.VideoFormat, error) {
	if f, ok := r.formats[formatID]; ok {
		return f, nil
	}
	return domain.FormatVisualFirst7Min, nil
}

func (r *fakeRepo) ListVideoFormats(_ context.Context) ([]domain.VideoFormat, error) {
	out := make([]domain.VideoFormat, 0, len(r.formats))
	for _, f := range r.formats {
		out = append(out, f)
	}
	return out, nil
}

func (r *fakeRepo) SaveVideoFormat(_ context.Context, format domain.VideoFormat) (domain.VideoFormat, error) {
	format.Version++
	r.formats[format.ID] = format
	return format, nil
}

func (r *fakeRepo) ListVoiceCalibrations(_ context.Context) ([]domain.VoiceCalibration, error) {
	out := make([]domain.VoiceCalibration, 0, len(r.calibration))
	for _, c := range r.calibration {
		out = append(out, c)
	}
	return out, nil
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
