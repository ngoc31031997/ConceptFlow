package application_test

import (
	"context"
	"strings"
	"testing"

	"orchestrator/internal/application"
	"orchestrator/internal/domain"
)

type fakeRenderContext struct {
	project    *domain.Project
	topic      string
	story      string
	storyboard string
	code       string
}

func (f *fakeRenderContext) Get(_ context.Context, _ string) (*domain.Project, error) {
	return f.project, nil
}
func (f *fakeRenderContext) GetAuthoringTopic(_ context.Context, _ string) (string, error) {
	return f.topic, nil
}
func (f *fakeRenderContext) GetAuthoringStory(_ context.Context, _ string) (string, error) {
	return f.story, nil
}
func (f *fakeRenderContext) GetAuthoringStoryboard(_ context.Context, _ string) (string, error) {
	return f.storyboard, nil
}
func (f *fakeRenderContext) GetAuthoringCode(_ context.Context, _ string) (string, error) {
	return f.code, nil
}

type fakeFormats struct{ format domain.VideoFormat }

func (f *fakeFormats) GetVideoFormat(_ context.Context, _ string, _ int) (domain.VideoFormat, error) {
	return f.format, nil
}

type fakeCalibration struct{ c domain.VoiceCalibration }

func (f *fakeCalibration) GetVoiceCalibration(_ context.Context, _ string) (domain.VoiceCalibration, error) {
	return f.c, nil
}

// templateStore serves one template for every role, so a test can assert on
// which variables were substituted rather than on prompt prose.
type templateStore struct{ text string }

func (t *templateStore) GetActive(_ context.Context, role domain.PromptRole) (domain.Prompt, error) {
	return domain.Prompt{ID: "p", Role: role, Name: "test", TemplateText: t.text, IsActive: true}, nil
}

func newRenderer(tmpl string, ctxData *fakeRenderContext) *application.RenderPromptUseCase {
	return application.NewRenderPromptUseCase(
		&templateStore{text: tmpl},
		ctxData,
		&fakeFormats{format: domain.VideoFormat{
			Name: "Giải thích khái niệm",
			Beats: []domain.FormatBeat{
				{ID: "hook", Required: true, MinSeconds: 10, MaxSeconds: 20, MaxRepeat: 1},
			},
		}},
		&fakeCalibration{},
	)
}

func aProject() *domain.Project {
	return &domain.Project{
		ContentLanguage: domain.LanguageVietnamese,
		RenderEngine:    domain.RenderEngineManim,
		VideoFormatID:   "concept-explainer",
	}
}

// TestRoleFor_StoryIsSharedByBothEngines — step 1 decides the story, not the
// pixels, so both engines run the same role.
func TestRoleFor_StoryIsSharedByBothEngines(t *testing.T) {
	for _, engine := range []string{"manim", "remotion"} {
		if got, _ := application.RoleFor("story", engine); got != domain.RoleStoryArchitect {
			t.Fatalf("%s: want story_architect, got %q", engine, got)
		}
	}
}

// CR-030 — bước duyệt đã bị bỏ khỏi sản phẩm, nên "review" phải bị từ chối
// như bất cứ tên bước lạ nào, chứ không lặng lẽ render một prompt không ai gọi.
func TestRoleFor_ReviewIsNoLongerAStep(t *testing.T) {
	for _, engine := range []string{"manim", "remotion"} {
		if _, err := application.RoleFor("review", engine); err == nil {
			t.Fatalf("%s: want an error for the removed review step", engine)
		}
	}
}

// TestRoleFor_StoryboardIsSharedAcrossEngines: there is one Visual Director.
// It writes an engine-agnostic shooting script, and only the code step forks
// by render engine.
func TestRoleFor_StoryboardIsSharedAcrossEngines(t *testing.T) {
	for _, engine := range []string{"manim", "remotion"} {
		if got, _ := application.RoleFor("storyboard", engine); got != domain.RoleVisualDirector {
			t.Fatalf("%s: want visual_director, got %q", engine, got)
		}
	}
}

func TestRoleFor_CodeBranchesByEngine(t *testing.T) {
	if got, _ := application.RoleFor("code", "manim"); got != domain.RoleManimEngineer {
		t.Fatalf("manim: want manim_engineer, got %q", got)
	}
	if got, _ := application.RoleFor("code", "remotion"); got != domain.RoleRemotionEngineer {
		t.Fatalf("remotion: want remotion_engineer, got %q", got)
	}
}

func TestRoleFor_RejectsAnUnknownStep(t *testing.T) {
	if _, err := application.RoleFor("polish", "manim"); err == nil {
		t.Fatal("expected an error for an unknown step")
	}
}

func TestRender_SubstitutesEveryVariable(t *testing.T) {
	uc := newRenderer(
		"CHỦ ĐỀ: {{topic}}\n{{channel_identity}}\n{{format_beats}}\n{{narration_language_rule}}",
		&fakeRenderContext{project: aProject(), topic: "Vì sao bầu trời có màu xanh"},
	)

	out, err := uc.Execute(context.Background(), "p1", domain.RoleStoryArchitect)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(out.Prompt, "{{") {
		t.Fatalf("a placeholder survived rendering:\n%s", out.Prompt)
	}
	if !strings.Contains(out.Prompt, "Vì sao bầu trời có màu xanh") {
		t.Fatal("the topic did not reach the prompt")
	}
	if !strings.Contains(out.Prompt, "BẢN SẮC KÊNH") {
		t.Fatal("the channel identity block did not reach the prompt")
	}
	if !strings.Contains(out.Prompt, "CẤU TRÚC BẮT BUỘC") {
		t.Fatal("the beat sheet did not reach the prompt")
	}
}

// TestRender_ProjectWithoutATopicGetsThePlaceholder — every project created
// before CR-027 D0 has no topic. Its prompt must read exactly as it did
// before, not show a blank where the subject should be.
func TestRender_ProjectWithoutATopicGetsThePlaceholder(t *testing.T) {
	uc := newRenderer("CHỦ ĐỀ: {{topic}}", &fakeRenderContext{project: aProject(), topic: ""})

	out, _ := uc.Execute(context.Background(), "p1", domain.RoleStoryArchitect)

	if !strings.Contains(out.Prompt, application.TopicPlaceholder) {
		t.Fatalf("want the placeholder, got %q", out.Prompt)
	}
}

// TestRender_PreviousOutputGrowsWithThePipeline — each step sees every
// earlier artefact and nothing later. Step 1 has nothing before it.
func TestRender_PreviousOutputGrowsWithThePipeline(t *testing.T) {
	data := &fakeRenderContext{
		project: aProject(), topic: "chủ đề",
		story: "DÀN Ý", storyboard: "STORYBOARD", code: "CODE",
	}

	cases := []struct {
		role     domain.PromptRole
		contains []string
		absent   []string
	}{
		{domain.RoleStoryArchitect, nil, []string{"DÀN Ý", "STORYBOARD", "CODE"}},
		{domain.RoleVisualDirector, []string{"DÀN Ý"}, []string{"STORYBOARD", "CODE"}},
		{domain.RoleManimEngineer, []string{"DÀN Ý", "STORYBOARD"}, []string{"CODE"}},
	}

	for _, tc := range cases {
		t.Run(string(tc.role), func(t *testing.T) {
			uc := newRenderer("TRƯỚC ĐÓ:\n{{previous_output}}", data)
			out, err := uc.Execute(context.Background(), "p1", tc.role)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			for _, want := range tc.contains {
				if !strings.Contains(out.Prompt, want) {
					t.Fatalf("%q missing from the prompt:\n%s", want, out.Prompt)
				}
			}
			for _, notWant := range tc.absent {
				if strings.Contains(out.Prompt, notWant) {
					t.Fatalf("%q must not reach this step:\n%s", notWant, out.Prompt)
				}
			}
		})
	}
}

// TestRender_ArtefactsAreJoinedTheWayTheGUIJoinedThem — the separator is
// part of the prompt the model has been reading all along.
func TestRender_ArtefactsAreJoinedTheWayTheGUIJoinedThem(t *testing.T) {
	uc := newRenderer("{{previous_output}}", &fakeRenderContext{
		project: aProject(), story: "A", storyboard: "B",
	})

	out, _ := uc.Execute(context.Background(), "p1", domain.RoleManimEngineer)

	if out.Prompt != "A\n\n---\n\nB" {
		t.Fatalf("unexpected join: %q", out.Prompt)
	}
}

func TestRender_EmptyPipelineSaysSoInsteadOfLeavingABlank(t *testing.T) {
	uc := newRenderer("{{previous_output}}", &fakeRenderContext{project: aProject()})

	out, _ := uc.Execute(context.Background(), "p1", domain.RoleVisualDirector)

	if strings.TrimSpace(out.Prompt) == "" {
		t.Fatal("an empty previous_output must explain itself, not render as nothing")
	}
}

// TestRender_BeatSheetOnlyForTheOutlineStep — the later steps have no
// {{format_beats}} placeholder, so fetching a format for them would be work
// whose result nothing reads.
func TestRender_BeatSheetOnlyForTheOutlineStep(t *testing.T) {
	uc := newRenderer("{{format_beats}}", &fakeRenderContext{project: aProject()})

	out, _ := uc.Execute(context.Background(), "p1", domain.RoleManimEngineer)

	if strings.Contains(out.Prompt, "CẤU TRÚC BẮT BUỘC") {
		t.Fatal("the beat sheet belongs to the outline step only")
	}
}

func TestRender_LanguageFollowsTheProject(t *testing.T) {
	project := aProject()
	project.ContentLanguage = domain.LanguageEnglish
	uc := newRenderer("{{narration_language_rule}}", &fakeRenderContext{project: project})

	out, _ := uc.Execute(context.Background(), "p1", domain.RoleStoryArchitect)

	if !strings.Contains(out.Prompt, "TIẾNG ANH") {
		t.Fatalf("an English project must get the English narration rule, got %q", out.Prompt)
	}
	if out.Language != "en" {
		t.Fatalf("want language en, got %q", out.Language)
	}
}

func TestRender_RequiresAProjectID(t *testing.T) {
	uc := newRenderer("x", &fakeRenderContext{project: aProject()})

	if _, err := uc.Execute(context.Background(), "", domain.RoleStoryArchitect); err == nil {
		t.Fatal("expected an error for an empty project_id")
	}
}
