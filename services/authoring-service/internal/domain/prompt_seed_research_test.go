package domain

import (
	"strings"
	"testing"
)

func seededText(t *testing.T, role PromptRole) string {
	t.Helper()
	tpl, ok := DefaultPromptTemplate(role, "vi")
	if !ok {
		t.Fatalf("no shipped prompt for role %q", role)
	}
	return tpl.TemplateText
}

func TestResearchRolesAreShippedAndNotGenericallyRendered(t *testing.T) {
	for _, role := range []PromptRole{RoleResearchQuestions, RoleResearchPrompt} {
		if !ValidPromptRole(string(role)) {
			t.Errorf("%q must be a known role", role)
		}
		if IsRenderablePromptRole(string(role)) {
			t.Errorf("%q is filled by its own use case, not the generic renderer", role)
		}
		if strings.TrimSpace(seededText(t, role)) == "" {
			t.Errorf("%q has an empty shipped prompt", role)
		}
	}
	if !IsRenderablePromptRole(string(RoleStoryArchitect)) {
		t.Error("the story architect prompt must stay renderable")
	}
}

func TestResearchQuestionsPromptTakesItsValues(t *testing.T) {
	out := RenderPromptTemplate(seededText(t, RoleResearchQuestions), map[string]string{
		"topic": "ĐỀ-TÀI", "brief": "BRIEF-NỘI-DUNG", "brief_language": "tiếng Việt",
		"kept_questions": "- Q1: câu của Creator", "redo_note": "GHI-CHÚ",
	})
	if strings.Contains(out, "{{") {
		t.Errorf("an unfilled placeholder is left in the questions prompt: %q", out[strings.Index(out, "{{"):])
	}
	for _, want := range []string{"ĐỀ-TÀI", "BRIEF-NỘI-DUNG", "câu của Creator", "GHI-CHÚ", `{"questions"`} {
		if !strings.Contains(out, want) {
			t.Errorf("questions prompt lacks %q", want)
		}
	}
}

func TestResearchPromptTakesItsValuesAndCarriesTheTemplate(t *testing.T) {
	out := RenderPromptTemplate(seededText(t, RoleResearchPrompt), map[string]string{
		"topic": "ĐỀ-TÀI", "central_question": "CÂU-HỎI-TRUNG-TÂM", "audience": "KHÁN-GIẢ", "target_duration": "4–5 phút",
		"questions_heading": "CÂU HỎI CẦN TRA", "questions": "Q1. Hỏi gì?", "extra": "", "answer_note": ". Đánh mã tiếp từ F13",
	})
	if strings.Contains(out, "{{") {
		t.Errorf("an unfilled placeholder is left in the research prompt: %q", out[strings.Index(out, "{{"):])
	}
	for _, want := range []string{"ĐỀ-TÀI", "CÂU-HỎI-TRUNG-TÂM", "Q1. Hỏi gì?", "Đánh mã tiếp từ F13", ResearchSchema, `"claim_vi"`, "Không bịa nguồn"} {
		if !strings.Contains(out, want) {
			t.Errorf("research prompt lacks %q", want)
		}
	}
	// The template the tool is asked to answer in must be one the paste reader accepts.
	start := strings.Index(out, `{"schema"`)
	if start < 0 {
		t.Fatal("research prompt has no JSON template")
	}
	parsed, err := ParseResearchPaste(out[start:], nil)
	if err != nil || len(parsed.Claims) != 1 {
		t.Fatalf("the prompt's own JSON template must be readable: claims=%d err=%v rejected=%+v", len(parsed.Claims), err, parsed.Rejected)
	}
}

func TestModelForResearch(t *testing.T) {
	m := AuthoringStepModels{Brief: "b", Research: "r", Story: "s"}
	if m.ModelFor("research") != "r" || m.ModelFor("brief") != "b" || m.ModelFor("nothing") != "" {
		t.Errorf("ModelFor wrong for %+v", m)
	}
}
