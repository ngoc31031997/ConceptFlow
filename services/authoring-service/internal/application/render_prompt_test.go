package application_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"authoring/internal/application"
	"authoring/internal/domain"
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

// --- CR-039: the AI flow's own roles ----------------------------------------

func TestAIRoleFor_UsesTheAIPromptsAndLeavesTheManualMappingAlone(t *testing.T) {
	cases := []struct {
		step, engine string
		want         domain.PromptRole
	}{
		{"story", "manim", domain.RoleStoryArchitect},
		{"story", "remotion", domain.RoleStoryArchitect},
		{"storyboard", "manim", domain.RoleVisualDirectorAI},
		{"storyboard", "remotion", domain.RoleVisualDirectorAI},
		{"code", "manim", domain.RoleManimEngineerAI},
		{"code", "remotion", domain.RoleRemotionEngineerAI},
	}
	for _, c := range cases {
		got, err := application.AIRoleFor(c.step, c.engine)
		if err != nil || got != c.want {
			t.Errorf("AIRoleFor(%s,%s) = %q, %v; want %q", c.step, c.engine, got, err, c.want)
		}
	}
	if _, err := application.AIRoleFor("review", "manim"); err == nil {
		t.Error("an unknown step must be an error")
	}
	// The manual flow's mapping is unchanged.
	if got, _ := application.RoleFor("storyboard", "manim"); got != domain.RoleVisualDirector {
		t.Errorf("RoleFor(storyboard) = %q, want the manual visual_director", got)
	}
}

func TestRender_AIEngineerPromptCarriesTheStoryButNotTheStoryboard(t *testing.T) {
	ctx := &fakeRenderContext{project: aProject(), story: "STORY-TEXT", storyboard: `{"scenes":"STORYBOARD-JSON"}`}
	r := newRenderer("PREV={{previous_output}}", ctx)

	for _, role := range []domain.PromptRole{domain.RoleRemotionEngineerAI, domain.RoleManimEngineerAI} {
		out, err := r.Execute(context.Background(), "p1", role)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.Prompt, "STORY-TEXT") || strings.Contains(out.Prompt, "STORYBOARD-JSON") {
			t.Errorf("%s prompt = %q — llm-service hands each call its own slice of the storyboard", role, out.Prompt)
		}
	}
	// …while the manual engineer prompt keeps both, exactly as before.
	out, _ := r.Execute(context.Background(), "p1", domain.RoleRemotionEngineer)
	if !strings.Contains(out.Prompt, "STORY-TEXT") || !strings.Contains(out.Prompt, "STORYBOARD-JSON") {
		t.Errorf("manual engineer prompt = %q, want story and storyboard", out.Prompt)
	}
	out, _ = r.Execute(context.Background(), "p1", domain.RoleVisualDirectorAI)
	if !strings.Contains(out.Prompt, "STORY-TEXT") || strings.Contains(out.Prompt, "STORYBOARD-JSON") {
		t.Errorf("visual_director_ai prompt = %q, want the story only", out.Prompt)
	}
}

// --- CR-048 T3: the code step carries the story's core, not the whole outline

// sampleStory is a full Story Architect outline in the format storyArchitectVI
// asks for, sized like a real one.
const sampleStory = `KIỂU VIDEO: co-che — vì chủ đề chỉ có một cơ chế: vi khuẩn ăn đường rồi thải axit làm mòn men răng.

CHẨN ĐOÁN CHỦ ĐỀ:
  Nút thắt: người xem nghĩ đường tự nó làm hỏng răng, không biết có một bên thứ ba sống trong miệng.
  Kiến thức nền: men răng là lớp cứng nhất cơ thể nhưng tan được trong axit.
  Công cụ giải thích: phóng to một chiếc răng như một thành phố nhỏ — vì cho thấy được sinh vật sống trên bề mặt.
  Trình tự tự nhiên: ăn kẹo → vi khuẩn ăn phần đường còn lại → thải axit → men răng mất khoáng → lỗ sâu.
  Điểm mở: một viên kẹo tan hết trong miệng nhưng tác hại mới bắt đầu.

TÌNH HUỐNG ỨNG VIÊN:
1. Một đứa trẻ ăn kẹo trước khi ngủ và không đánh răng.
2. Hai người ăn cùng lượng đường, một người ăn một lần, một người nhấm nháp cả ngày.
3. Nha sĩ soi một chiếc răng dưới kính hiển vi.
CHỌN: 2 — vì cho thấy thời gian quan trọng hơn lượng đường / loại 1 vì quá quen / loại 3 vì thiếu câu hỏi.

KHUNG BÀI:
  Thế giới chính: một chiếc răng hàm phóng to với mảng bám vi khuẩn trên bề mặt
  Cách làm hiển nhiên: bớt ăn đường là đủ
  Đáp án phản trực giác: số lần ăn quan trọng hơn tổng lượng đường
  Lời giải: mỗi lần ăn mở ra khoảng hai mươi phút axit; ít lần ăn thì men răng kịp hồi khoáng
  Tên khái niệm: mất khoáng và tái khoáng

CÂU HỎI CỐT LÕI: Vì sao nhấm nháp một gói kẹo cả ngày hại răng hơn ăn hết nó trong một lần?
INSIGHT CỐT LÕI: Răng không bị đường ăn mòn, mà bị axit do vi khuẩn thải ra — và mỗi lần ăn là một đợt tấn công mới.
SAI LẦM TRỰC GIÁC: Tổng lượng đường mới là thứ quyết định răng có sâu hay không.
ẨN DỤ CHỦ ĐẠO: men răng như bức tường bị mưa axit, được vá lại giữa các cơn mưa.
ẨN DỤ GÃY Ở ĐÂU: bức tường không tự vá, còn men răng thì tự hồi khoáng nhờ nước bọt.
AHA MOMENT: Mỗi lần ăn là một cơn mưa axit, không phải mỗi gam đường.
  Tôi từng nghĩ: ăn ít đường là răng an toàn.
  Nhưng bây giờ tôi nhận ra: ăn ít lần mới là điều giữ cho răng kịp tự vá.

DANH SÁCH VÍ DỤ:
  1. Viên kẹo buổi sáng — một đợt axit kéo dài hai mươi phút — mở đầu vì quen thuộc nhất.
  2. Nhấm nháp cả ngày — axit không bao giờ dừng — đặt sau để đối lập.

BEAT hook — Mở đầu:
- Cảnh: hai người, cùng một gói kẹo, hai cách ăn.
- Ý chính: cùng lượng đường, kết quả khác nhau.
- Vai trò nhận thức: gây tò mò.
- Người xem cần nhận ra trên màn hình: hai chiếc răng khác nhau sau một tuần.
- Kiểm chứng: cơ chế đã được nha khoa công nhận, không nêu số liệu cụ thể.
- Lời thoại nháp: "Hai người ăn cùng một gói kẹo. Một tuần sau, chỉ một người phải đi nha sĩ. Vì sao?"
- Số từ: 22

BEAT concrete — Cơ chế:
- Cảnh: phóng vào bề mặt răng, mảng bám hiện ra như một thành phố.
- Ý chính: vi khuẩn ăn đường và thải axit.
- Vai trò nhận thức: đưa bên thứ ba vào câu chuyện.
- Người xem cần nhận ra trên màn hình: axit thoát ra từ vi khuẩn và chạm vào men răng.
- Kiểm chứng: định tính, không nêu tên loài vi khuẩn.
- Lời thoại nháp: "Đường không tự làm hỏng răng. Nó là bữa ăn cho hàng triệu vi khuẩn đang sống trên bề mặt răng, và thứ chúng thải ra là axit."
- Số từ: 31

BEAT payoff — Kết:
- Cảnh: đồng hồ hai mươi phút lặp lại suốt ngày của người nhấm nháp.
- Ý chính: số lần ăn quyết định thời gian răng bị tấn công.
- Vai trò nhận thức: chốt insight.
- Người xem cần nhận ra trên màn hình: vạch axit không bao giờ hạ xuống.
- Kiểm chứng: định tính.
- Lời thoại nháp: "Mỗi lần ăn là một cơn mưa axit. Ăn ít lần, răng có thời gian tự vá lại."
- Số từ: 18

TỔNG SỐ TỪ: 71
TỰ KIỂM: đã soi 10 mục, không phải sửa gì`

// captureWarnings routes slog to a buffer for the duration of the test.
func captureWarnings(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return &buf
}

func TestRender_AICodeStepKeepsOnlyTheStoryCore(t *testing.T) {
	logs := captureWarnings(t)
	r := newRenderer("{{previous_output}}", &fakeRenderContext{project: aProject(), story: sampleStory, storyboard: `{"scenes":[]}`})

	wantLines := []string{
		"KIỂU VIDEO: co-che — vì chủ đề chỉ có một cơ chế: vi khuẩn ăn đường rồi thải axit làm mòn men răng.",
		"Thế giới chính: một chiếc răng hàm phóng to với mảng bám vi khuẩn trên bề mặt",
		"CÂU HỎI CỐT LÕI: Vì sao nhấm nháp một gói kẹo cả ngày hại răng hơn ăn hết nó trong một lần?",
		"INSIGHT CỐT LÕI: Răng không bị đường ăn mòn, mà bị axit do vi khuẩn thải ra — và mỗi lần ăn là một đợt tấn công mới.",
		"SAI LẦM TRỰC GIÁC: Tổng lượng đường mới là thứ quyết định răng có sâu hay không.",
		"AHA MOMENT: Mỗi lần ăn là một cơn mưa axit, không phải mỗi gam đường.",
		"  Tôi từng nghĩ: ăn ít đường là răng an toàn.",
		"  Nhưng bây giờ tôi nhận ra: ăn ít lần mới là điều giữ cho răng kịp tự vá.",
	}
	want := strings.Join(wantLines, "\n")

	for _, role := range []domain.PromptRole{domain.RoleRemotionEngineerAI, domain.RoleManimEngineerAI} {
		out, err := r.Execute(context.Background(), "p1", role)
		if err != nil {
			t.Fatal(err)
		}
		if out.Prompt != want {
			t.Errorf("%s previous_output =\n%s\n\nwant\n%s", role, out.Prompt, want)
		}
		if len(out.Prompt) >= len(sampleStory) {
			t.Errorf("%s: the core (%d bytes) must be shorter than the outline (%d)", role, len(out.Prompt), len(sampleStory))
		}
		t.Logf("%s {{previous_output}}: %d chars (whole outline: %d chars)",
			role, len([]rune(out.Prompt)), len([]rune(sampleStory)))
	}
	if logs.Len() != 0 {
		t.Errorf("a complete outline must not warn, got:\n%s", logs.String())
	}

	// The manual flow still gets the whole outline and the storyboard.
	for _, role := range []domain.PromptRole{domain.RoleRemotionEngineer, domain.RoleManimEngineer} {
		out, _ := r.Execute(context.Background(), "p1", role)
		if out.Prompt != sampleStory+"\n\n---\n\n"+`{"scenes":[]}` {
			t.Errorf("%s previous_output changed:\n%s", role, out.Prompt)
		}
	}
}

// TestRender_AICodeStepSystemPromptSize renders the shipped AI code templates
// with sampleStory and logs the system prompt size with the core versus with
// the whole outline — the number CR-048 T3 reports.
func TestRender_AICodeStepSystemPromptSize(t *testing.T) {
	core, missing := "", []string(nil)
	for _, tmpl := range domain.DefaultPromptTemplates() {
		if tmpl.Role != domain.RoleRemotionEngineerAI && tmpl.Role != domain.RoleManimEngineerAI {
			continue
		}
		project := aProject()
		if tmpl.Role == domain.RoleRemotionEngineerAI {
			project.RenderEngine = domain.RenderEngineRemotion
		}
		r := newRenderer(tmpl.TemplateText, &fakeRenderContext{project: project, story: sampleStory})
		out, err := r.Execute(context.Background(), "p1", tmpl.Role)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(out.Prompt, "BEAT hook") || strings.Contains(out.Prompt, "TÌNH HUỐNG ỨNG VIÊN:\n1.") {
			t.Errorf("%s: the outline's beats reached the code step's system prompt", tmpl.Role)
		}
		if core == "" {
			core, missing = application.ExtractStoryCoreForTest(sampleStory)
			if len(missing) != 0 {
				t.Fatalf("sample story missing labels %v", missing)
			}
		}
		uses := strings.Count(tmpl.TemplateText, "{{previous_output}}")
		before := len([]rune(out.Prompt)) + uses*(len([]rune(sampleStory))-len([]rune(core)))
		t.Logf("%s system prompt: %d chars with the whole outline -> %d chars with the core ({{previous_output}} used %d time(s))",
			tmpl.Role, before, len([]rune(out.Prompt)), uses)
	}
}

func TestRender_AICodeStepFallsBackToTheWholeStoryAndWarns(t *testing.T) {
	logs := captureWarnings(t)
	story := "Dàn ý Creator tự viết lại, không theo mẫu nào.\n\nCảnh 1: hai người ăn kẹo.\nCảnh 2: vi khuẩn."
	r := newRenderer("{{previous_output}}", &fakeRenderContext{project: aProject(), story: story})

	for _, role := range []domain.PromptRole{domain.RoleRemotionEngineerAI, domain.RoleManimEngineerAI} {
		logs.Reset()
		out, err := r.Execute(context.Background(), "proj-42", role)
		if err != nil {
			t.Fatal(err)
		}
		if out.Prompt != story {
			t.Errorf("%s: want the whole story, got %q", role, out.Prompt)
		}
		if !strings.Contains(logs.String(), "level=WARN") || !strings.Contains(logs.String(), "project_id=proj-42") {
			t.Errorf("%s: want a warning naming the project, got %q", role, logs.String())
		}
	}
}

func TestRender_AICodeStepToleratesMarkdownAndWarnsOnMissingLabels(t *testing.T) {
	logs := captureWarnings(t)
	story := "## **KIỂU VIDEO:** so-sanh — vì có hai cách làm\n" +
		"   **CÂU HỎI CỐT LÕI**: Nên chọn cái nào?\n" +
		"- insight cốt lõi :\n  Cái rẻ hơn không phải lúc nào cũng tiết kiệm hơn.\n\n" +
		"BEAT hook — Mở đầu:\n- Lời thoại nháp: \"...\""
	r := newRenderer("{{previous_output}}", &fakeRenderContext{project: aProject(), story: story})

	out, _ := r.Execute(context.Background(), "p7", domain.RoleManimEngineerAI)
	want := "KIỂU VIDEO: so-sanh — vì có hai cách làm\n" +
		"CÂU HỎI CỐT LÕI: Nên chọn cái nào?\n" +
		"insight cốt lõi : Cái rẻ hơn không phải lúc nào cũng tiết kiệm hơn."
	if out.Prompt != want {
		t.Errorf("got\n%s\nwant\n%s", out.Prompt, want)
	}
	if !strings.Contains(logs.String(), "project_id=p7") || !strings.Contains(logs.String(), "AHA MOMENT") {
		t.Errorf("want a warning listing the missing labels, got %q", logs.String())
	}
}
