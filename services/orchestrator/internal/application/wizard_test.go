package application_test

import (
	"context"
	"errors"
	"testing"

	"orchestrator/internal/application"
	"orchestrator/internal/domain"
)

type fakeWizardRepo struct {
	status  domain.ProjectStatus
	patch   *domain.WizardSettingsPatch
	project domain.Project
}

func (f *fakeWizardRepo) GetStatus(context.Context, string) (domain.ProjectStatus, error) {
	return f.status, nil
}
func (f *fakeWizardRepo) Get(context.Context, string) (*domain.Project, error) {
	p := f.project
	return &p, nil
}
func (f *fakeWizardRepo) GetVideoFormat(_ context.Context, id string, _ int) (domain.VideoFormat, error) {
	for _, format := range domain.BuiltinFormats() {
		if format.ID == id {
			return format, nil
		}
	}
	return domain.VideoFormat{}, errors.New("no such format")
}
func (f *fakeWizardRepo) PatchWizardSettings(_ context.Context, _ string, p domain.WizardSettingsPatch) error {
	f.patch = &p
	return nil
}

func TestPatchWizardSettings_PassesOnlyGivenFields(t *testing.T) {
	repo := &fakeWizardRepo{status: domain.StatusDraft}
	uc := application.NewPatchWizardSettingsUseCase(repo)
	q := domain.RenderQuality("720p30")

	if err := uc.Execute(context.Background(), "p1", domain.WizardSettingsPatch{RenderQuality: &q}); err != nil {
		t.Fatal(err)
	}
	if repo.patch.RenderQuality == nil || *repo.patch.RenderQuality != q {
		t.Errorf("quality not passed: %+v", repo.patch)
	}
	if repo.patch.ContentLanguage != nil || repo.patch.RenderEngine != nil || repo.patch.VideoFormatID != nil || repo.patch.Confirm {
		t.Errorf("absent fields must stay nil: %+v", repo.patch)
	}
}

func TestPatchWizardSettings_RejectsBadInputAndStartedProjects(t *testing.T) {
	uc := application.NewPatchWizardSettingsUseCase(&fakeWizardRepo{status: domain.StatusDraft})
	fr := domain.ContentLanguage("fr")
	err := uc.Execute(context.Background(), "p1", domain.WizardSettingsPatch{ContentLanguage: &fr})
	if !errors.Is(err, domain.ErrInvalidWizardInput) {
		t.Errorf("bad language: err = %v", err)
	}
	bad := domain.RenderQuality("8k")
	err = uc.Execute(context.Background(), "p1", domain.WizardSettingsPatch{RenderQuality: &bad})
	if !errors.Is(err, domain.ErrInvalidWizardInput) {
		t.Errorf("bad quality: err = %v", err)
	}
	vol := 1.5
	err = uc.Execute(context.Background(), "p1", domain.WizardSettingsPatch{BackgroundMusicVolume: &vol})
	if !errors.Is(err, domain.ErrInvalidWizardInput) {
		t.Errorf("bad volume: err = %v", err)
	}

	started := application.NewPatchWizardSettingsUseCase(&fakeWizardRepo{status: domain.StatusRendering})
	vi := domain.LanguageVietnamese
	err = started.Execute(context.Background(), "p1", domain.WizardSettingsPatch{ContentLanguage: &vi})
	if !errors.Is(err, domain.ErrInvalidStatus) {
		t.Errorf("started project: err = %v, want ErrInvalidStatus", err)
	}
}

func TestEffectiveWizardStep(t *testing.T) {
	cases := []struct {
		status domain.ProjectStatus
		stored int
		want   int
	}{
		{domain.StatusDraft, 0, 1},
		{domain.StatusDraft, 3, 3},
		{domain.StatusAwaitingReview, 3, 4},
		{domain.StatusRendering, 3, 5},
		{domain.StatusFailedRenderScenes, 3, 5},
		{domain.StatusReadyToPublish, 3, 6},
		{domain.StatusPublished, 3, 7},
	}
	for _, c := range cases {
		if got := domain.EffectiveWizardStep(&domain.Project{Status: c.status, WizardStep: c.stored}); got != c.want {
			t.Errorf("%s stored=%d: got %d, want %d", c.status, c.stored, got, c.want)
		}
	}
}

// Render settings stay editable until render_scenes has run, merge settings
// until assemble_video has run — so a failure there can be fixed and retried —
// while authoring/TTS settings keep their old lock (draft or any failure).
func TestPatchWizardSettings_LocksEachSettingAtItsReadingStep(t *testing.T) {
	q := domain.RenderQuality("1080p60")
	font := "Montserrat"
	mode := domain.SubtitleMode("track")
	music := ""
	vol := 0.3
	voice := "vi-VN-HoaiMyNeural"

	render := domain.WizardSettingsPatch{RenderQuality: &q, VideoFont: &font}
	merge := domain.WizardSettingsPatch{SubtitleMode: &mode, BackgroundMusicPath: &music, BackgroundMusicVolume: &vol}
	tts := domain.WizardSettingsPatch{VoiceID: &voice}

	cases := []struct {
		status             domain.ProjectStatus
		render, merge, tts bool
	}{
		{domain.StatusDraft, true, true, true},
		{domain.StatusAwaitingReview, true, true, false},
		{domain.StatusFailedValidateScript, true, true, true},
		{domain.StatusFailedSynthesizeSpeech, true, true, true},
		{domain.StatusFailedRenderScenes, true, true, true},
		{domain.StatusFailedAssembleVideo, false, true, true},
		{domain.StatusFailedQCVideo, false, false, true},
		{domain.StatusSynthesizingSpeech, false, false, false},
		{domain.StatusRendering, false, false, false},
		{domain.StatusReadyToPublish, false, false, false},
	}
	for _, c := range cases {
		for _, g := range []struct {
			name  string
			patch domain.WizardSettingsPatch
			want  bool
		}{{"render", render, c.render}, {"merge", merge, c.merge}, {"tts", tts, c.tts}} {
			repo := &fakeWizardRepo{status: c.status}
			err := application.NewPatchWizardSettingsUseCase(repo).Execute(context.Background(), "p1", g.patch)
			if g.want && err != nil {
				t.Errorf("%s at %s: err = %v, want allowed", g.name, c.status, err)
			}
			if !g.want && !errors.Is(err, domain.ErrInvalidStatus) {
				t.Errorf("%s at %s: err = %v, want ErrInvalidStatus", g.name, c.status, err)
			}
			if !g.want && repo.patch != nil {
				t.Errorf("%s at %s: refused patch was still stored", g.name, c.status)
			}
		}
	}
}

// A patch mixing groups is refused whole when any group is locked: storing the
// allowed half would leave the Creator believing the rest was saved too.
func TestPatchWizardSettings_MixedPatchRefusedWhole(t *testing.T) {
	q := domain.RenderQuality("1080p60")
	mode := domain.SubtitleMode("track")
	repo := &fakeWizardRepo{status: domain.StatusFailedAssembleVideo}
	err := application.NewPatchWizardSettingsUseCase(repo).Execute(context.Background(), "p1",
		domain.WizardSettingsPatch{RenderQuality: &q, SubtitleMode: &mode})
	if !errors.Is(err, domain.ErrInvalidStatus) || repo.patch != nil {
		t.Errorf("err = %v, stored = %v; want refused and nothing stored", err, repo.patch != nil)
	}
}

func TestPatchWizardSettings_ChoosingTheShortBringsWhatAShortNeeds(t *testing.T) {
	repo := &fakeWizardRepo{status: domain.StatusDraft}
	short := domain.ModeShortOnly
	if err := application.NewPatchWizardSettingsUseCase(repo).Execute(
		context.Background(), "p1", domain.WizardSettingsPatch{VideoOutputMode: &short}); err != nil {
		t.Fatal(err)
	}
	p := repo.patch
	if *p.VideoFormatID != domain.FormatVerticalShort60s.ID || *p.RenderEngine != domain.RenderEngineRemotion || *p.SubtitleMode != domain.SubtitleModeBurnIn {
		t.Errorf("short defaults not applied: format %v engine %v subtitles %v", *p.VideoFormatID, *p.RenderEngine, *p.SubtitleMode)
	}
	if p.SubtitleStyle == nil || p.SubtitleStyle.Position != "top" {
		t.Errorf("a short's captions go at the top, got %+v", p.SubtitleStyle)
	}
}

func TestPatchWizardSettings_AShortKeepsTheSubtitlesTheCreatorPicked(t *testing.T) {
	repo := &fakeWizardRepo{status: domain.StatusDraft}
	short, off := domain.ModeShortOnly, domain.SubtitleModeOff
	style := domain.DefaultSubtitleStyle()
	if err := application.NewPatchWizardSettingsUseCase(repo).Execute(context.Background(), "p1",
		domain.WizardSettingsPatch{VideoOutputMode: &short, SubtitleMode: &off, SubtitleStyle: &style}); err != nil {
		t.Fatal(err)
	}
	if *repo.patch.SubtitleMode != domain.SubtitleModeOff || repo.patch.SubtitleStyle.Position != "bottom" {
		t.Errorf("the Creator's choice was overridden: %v %+v", *repo.patch.SubtitleMode, repo.patch.SubtitleStyle)
	}
}

func TestPatchWizardSettings_ConfirmRefusesAShortItCannotBuild(t *testing.T) {
	ctx := context.Background()
	manim := domain.RenderEngineManim
	cases := map[string]struct {
		project domain.Project
		patch   domain.WizardSettingsPatch
	}{
		"short on a long format":   {domain.Project{VideoOutputMode: domain.ModeShortOnly, VideoFormatID: "quick_explainer_3min", RenderEngine: domain.RenderEngineRemotion}, domain.WizardSettingsPatch{Confirm: true}},
		"long on the short format": {domain.Project{VideoOutputMode: domain.ModeLongOnly, VideoFormatID: domain.FormatVerticalShort60s.ID}, domain.WizardSettingsPatch{Confirm: true}},
		"short with Manim":         {domain.Project{VideoOutputMode: domain.ModeShortOnly, VideoFormatID: domain.FormatVerticalShort60s.ID}, domain.WizardSettingsPatch{RenderEngine: &manim, Confirm: true}},
	}
	for name, c := range cases {
		repo := &fakeWizardRepo{status: domain.StatusDraft, project: c.project}
		err := application.NewPatchWizardSettingsUseCase(repo).Execute(ctx, "p1", c.patch)
		if !errors.Is(err, domain.ErrInvalidWizardInput) || repo.patch != nil {
			t.Errorf("%s: err = %v, saved = %v", name, err, repo.patch != nil)
		}
	}
	ok := &fakeWizardRepo{status: domain.StatusDraft, project: domain.Project{
		VideoOutputMode: domain.ModeShortOnly, VideoFormatID: domain.FormatVerticalShort60s.ID, RenderEngine: domain.RenderEngineRemotion}}
	if err := application.NewPatchWizardSettingsUseCase(ok).Execute(ctx, "p1", domain.WizardSettingsPatch{Confirm: true}); err != nil {
		t.Errorf("a buildable short must confirm: %v", err)
	}
}
