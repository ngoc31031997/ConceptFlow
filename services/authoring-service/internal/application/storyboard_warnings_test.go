package application_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"authoring/internal/application"
	"authoring/internal/domain"
)

// A canonical storyboard: the hook's narration is 6 + 25 = 31 words (~13 s at
// 140 wpm), and shot 1.2 talks about germs the picture never shows.
const warnedStoryboard = `{"hero":"chiếc răng","world":null,"palette":[{"role":"nen","hex":"#112233","meaning":"nền"}],
"scenes":[{"id":"hook","title":"Mở","invariant":"i","transition_in":null,"mood":"m","end_frame":"e","shots":[
 {"id":"1.1","camera":"toàn","visual":"Chiếc răng trắng mỉm cười giữa nền xanh.","narration":"Chiếc răng của bạn đang cười."},
 {"id":"1.2","camera":"cận","visual":"Chiếc răng nhăn mặt, một đốm nâu hiện ra.","narration":"Nhưng vi khuẩn đã tới rồi, và chúng đang ăn dần lớp men răng của bạn mỗi ngày mỗi đêm không ngừng nghỉ chút nào."}]}],
"layout":{}}`

type errFormats struct{}

func (errFormats) GetVideoFormat(context.Context, string, int) (domain.VideoFormat, error) {
	return domain.VideoFormat{}, errors.New("not found")
}

func storyboardWarningsFixture(t *testing.T, engine domain.RenderEngine, formats application.FormatLookupPort) *application.GenerateAuthoringUseCase {
	t.Helper()
	renderCtx := &fakeRenderContext{
		project: &domain.Project{
			ProjectID: "p1", ContentLanguage: domain.LanguageVietnamese, RenderEngine: engine,
			VideoFormatID: "fmt", VideoFormatVersion: 1, VoiceID: "v1",
		},
		topic: "Sâu răng",
	}
	uc := application.NewGenerateAuthoringUseCase(
		newRenderer("Chủ đề: {{topic}}", renderCtx),
		&stubProvider{content: "raw"}, nil, renderCtx, nil,
		&recordingSaver{}, &contentSaver{}, &contentSaver{}, 0, 16000,
	)
	uc.WithPipeline(&stubFinalizer{out: warnedStoryboard}, &stubCodegen{})
	if formats != nil {
		// An uncalibrated voice: the 140 wpm Vietnamese default applies.
		uc.WithStoryboardChecks(formats, &fakeCalibration{})
	}
	return uc
}

// hook is 6–8 s; 13.3 s is 66% over the 8 s maximum.
var hookFormat = &fakeFormats{format: domain.VideoFormat{
	Name:  "Thử",
	Beats: []domain.FormatBeat{{ID: "hook", Required: true, MinSeconds: 6, MaxSeconds: 8, MaxRepeat: 1}},
}}

func TestStoryboardStepReturnsLengthAndIllustrationWarnings(t *testing.T) {
	uc := storyboardWarningsFixture(t, domain.RenderEngineRemotion, hookFormat)
	got, err := uc.Execute(context.Background(), "p1", "storyboard")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"Cảnh hook: ~13 giây, ngân sách 6–8 giây (+66%)",
		"Shot 1.2: lời thoại nhắc 'vi khuẩn' nhưng HÌNH không có",
	}
	if strings.Join(got.Warnings, "\n") != strings.Join(want, "\n") {
		t.Fatalf("warnings = %q, want %q", got.Warnings, want)
	}
	if got.Content != warnedStoryboard || got.SaveError != "" {
		t.Error("warnings must not change or block what is saved")
	}
}

func TestStoryboardIllustrationCheckIsRemotionOnly(t *testing.T) {
	uc := storyboardWarningsFixture(t, domain.RenderEngineManim, hookFormat)
	got, err := uc.Execute(context.Background(), "p1", "storyboard")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Warnings) != 1 || !strings.HasPrefix(got.Warnings[0], "Cảnh hook:") {
		t.Fatalf("warnings = %q, want only the length warning", got.Warnings)
	}
}

func TestStoryboardLengthCheckSaysWhenItCannotRun(t *testing.T) {
	uc := storyboardWarningsFixture(t, domain.RenderEngineManim, errFormats{})
	got, err := uc.Execute(context.Background(), "p1", "storyboard")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Warnings) != 1 || !strings.Contains(got.Warnings[0], `không tải được format "fmt" (v1)`) {
		t.Fatalf("warnings = %q", got.Warnings)
	}
}

func TestStoryboardWithoutChecksWiredHasNoWarnings(t *testing.T) {
	uc := storyboardWarningsFixture(t, domain.RenderEngineManim, nil)
	got, err := uc.Execute(context.Background(), "p1", "storyboard")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Warnings) != 0 {
		t.Fatalf("warnings = %q", got.Warnings)
	}
}
