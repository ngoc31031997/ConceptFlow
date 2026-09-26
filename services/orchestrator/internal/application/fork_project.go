package application

import (
	"context"
	"errors"
	"fmt"

	"orchestrator/internal/domain"
)

// ErrForkStepInvalid is returned when from_step is not one of the steps a fork
// can start at (2..5).
var ErrForkStepInvalid = errors.New("from_step must be between 2 and 5")

// ForkPort is what forking needs: reading the source and building the copy.
type ForkPort interface {
	Get(ctx context.Context, projectID string) (*domain.Project, error)
	Save(ctx context.Context, project *domain.Project) error
	SaveWizardSettings(ctx context.Context, projectID string, s domain.WizardSettings) error
	SetForkedFrom(ctx context.Context, projectID, from string) error
}

// ForkAuthoringPort reads and writes the authoring artefacts directly (no
// draft lock, no edit history: the copy is brand new).
type ForkAuthoringPort interface {
	GetAuthoringTopic(ctx context.Context, projectID string) (string, error)
	GetAuthoringMode(ctx context.Context, projectID string) (string, error)
	GetAuthoringStory(ctx context.Context, projectID string) (string, error)
	GetAuthoringStoryboard(ctx context.Context, projectID string) (string, error)
	GetAuthoringModels(ctx context.Context, projectID string) (domain.AuthoringStepModels, error)
	SaveAuthoringTopic(ctx context.Context, projectID, topic string, language domain.ContentLanguage) error
	SaveAuthoringStory(ctx context.Context, projectID, content, topic string) error
	SaveAuthoringStoryboard(ctx context.Context, projectID, content string) error
	SaveAuthoringMode(ctx context.Context, projectID, mode string) error
	SaveAuthoringModels(ctx context.Context, projectID string, models domain.AuthoringStepModels) error
}

// ForkProjectUseCase makes a NEW project from an existing one, to change a
// video without touching the original. The Creator picks the step to start
// again from (2..5); everything before it is copied, everything from it on is
// left empty to be redone:
//
//	from 2 Cấu hình → topic
//	from 3 Kịch bản → topic + settings
//	from 4 Visual   → topic + settings + story
//	from 5 Code     → topic + settings + story + storyboard
//
// Nothing produced by the render (scenes, audio, video, clips, YouTube data) is
// copied, and the original is never modified.
type ForkProjectUseCase struct {
	repo      ForkPort
	authoring ForkAuthoringPort
}

func NewForkProjectUseCase(repo ForkPort, authoring ForkAuthoringPort) *ForkProjectUseCase {
	return &ForkProjectUseCase{repo: repo, authoring: authoring}
}

// ForkProjectOutput is the new project.
type ForkProjectOutput struct {
	ProjectID string
	FromStep  int
	// NeedsMusicReselect: the source had background music, which is NOT carried
	// over. The file lives in the source's own folder on the shared volume and
	// is deleted with it, so the fork would lose it later without warning; the
	// Creator picks it again on the config step.
	NeedsMusicReselect bool
}

func (uc *ForkProjectUseCase) Execute(ctx context.Context, sourceID string, fromStep int) (*ForkProjectOutput, error) {
	if fromStep < domain.FlowConfig || fromStep > domain.FlowCode {
		return nil, ErrForkStepInvalid
	}
	src, err := uc.repo.Get(ctx, sourceID)
	if err != nil {
		return nil, err
	}
	topic, err := uc.authoring.GetAuthoringTopic(ctx, sourceID)
	if err != nil {
		return nil, fmt.Errorf("read topic: %w", err)
	}
	mode, _ := uc.authoring.GetAuthoringMode(ctx, sourceID)
	models, _ := uc.authoring.GetAuthoringModels(ctx, sourceID)

	dst := blankFork(src)
	dst.ProjectID = newUUID()
	dst.SagaID = newUUID()
	if err := uc.repo.Save(ctx, dst); err != nil {
		return nil, fmt.Errorf("create fork: %w", err)
	}
	if err := uc.repo.SetForkedFrom(ctx, dst.ProjectID, sourceID); err != nil {
		return nil, fmt.Errorf("link fork: %w", err)
	}
	if topic != "" {
		if err := uc.authoring.SaveAuthoringTopic(ctx, dst.ProjectID, topic, dst.ContentLanguage); err != nil {
			return nil, err
		}
	}
	if mode != "" {
		if err := uc.authoring.SaveAuthoringMode(ctx, dst.ProjectID, mode); err != nil {
			return nil, err
		}
	}
	if err := uc.authoring.SaveAuthoringModels(ctx, dst.ProjectID, models); err != nil {
		return nil, err
	}

	if fromStep >= domain.FlowStory {
		// Step 2 (settings) is done from step 3 on: this also lifts wizard_step
		// so the copy opens at the script steps, not at config.
		if err := uc.repo.SaveWizardSettings(ctx, dst.ProjectID, settingsOf(src)); err != nil {
			return nil, err
		}
	}
	if fromStep >= domain.FlowVisual {
		story, err := uc.authoring.GetAuthoringStory(ctx, sourceID)
		if err != nil {
			return nil, fmt.Errorf("read story: %w", err)
		}
		if err := uc.authoring.SaveAuthoringStory(ctx, dst.ProjectID, story, ""); err != nil {
			return nil, err
		}
	}
	if fromStep >= domain.FlowCode {
		board, err := uc.authoring.GetAuthoringStoryboard(ctx, sourceID)
		if err != nil {
			return nil, fmt.Errorf("read storyboard: %w", err)
		}
		if err := uc.authoring.SaveAuthoringStoryboard(ctx, dst.ProjectID, board); err != nil {
			return nil, err
		}
	}
	return &ForkProjectOutput{
		ProjectID: dst.ProjectID, FromStep: fromStep, NeedsMusicReselect: src.BackgroundMusicPath != nil,
	}, nil
}

// blankFork copies a project's settings and drops everything a run produced.
func blankFork(src *domain.Project) *domain.Project {
	dst := *src
	dst.Status = domain.StatusDraft
	dst.Scenes = nil
	dst.ScriptContent = ""
	dst.ManimSceneClassName = ""
	dst.Beats = nil
	dst.ValidationWarnings = nil
	dst.VideoPath = nil
	dst.RenderedVideoPath = nil
	dst.CaptionPath = nil
	dst.CaptionStatus = nil
	dst.ClipMarks = nil
	dst.ClipRequests = nil
	dst.Clips = nil
	dst.LayoutMarks = nil
	dst.WaitOffsets = nil
	dst.Chapters = nil
	dst.RenderedVideoSeconds = 0
	dst.IntroDurationSeconds = 0
	dst.CompanionProjectID = nil
	dst.YoutubeTitle, dst.YoutubeDescription, dst.YoutubeVisibility = nil, nil, nil
	dst.YoutubeTags, dst.YoutubePublishAt, dst.YoutubeThumbnailPath = nil, nil, nil
	dst.YoutubeChannelID, dst.YoutubeVideoURL = nil, nil
	dst.ErrorMessage = nil
	dst.BackgroundMusicPath = nil
	dst.WizardStep = 0
	dst.WizardRoute = ""
	return &dst
}

func settingsOf(p *domain.Project) domain.WizardSettings {
	return domain.WizardSettings{
		ContentLanguage: p.ContentLanguage, RenderEngine: p.RenderEngine, TTSEnabled: p.TTSEnabled,
		VoiceID: p.VoiceID, SubtitleMode: p.SubtitleMode, SubtitleStyle: p.SubtitleStyle,
		RenderQuality: p.RenderQuality, VideoFormatID: p.VideoFormatID, VideoOutputMode: p.VideoOutputMode,
		BackgroundMusicVolume: p.BackgroundMusicVolume,
		VideoFont:             p.VideoFont,
	}
}
