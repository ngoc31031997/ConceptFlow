package application_test

import (
	"context"
	"strings"
	"testing"

	"authoring/internal/application"
	"authoring/internal/domain"
)

func TestRender_PastedTextIsNeverExpanded(t *testing.T) {
	uc := application.NewRenderPromptUseCase(&templateStore{text: "A={{script}} B={{topic}}"}, &fakeRenderContext{}, &fakeFormats{}, &fakeCalibration{})
	got, err := uc.Render(context.Background(), application.RenderInput{
		Role: domain.RoleRemotionAdjust, Script: "x {{topic}} y $& z", Topic: "T",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Prompt != "A=x {{topic}} y $& z B=T" {
		t.Fatalf("got %q", got.Prompt)
	}
}

// The code roles' narration rule names the `narrations` array; the steps
// before code get the plain language rule.
func TestRender_CodeRolesNameTheNarrationsArray(t *testing.T) {
	uc := application.NewRenderPromptUseCase(&templateStore{text: "{{narration_language_rule}}"}, &fakeRenderContext{}, &fakeFormats{}, &fakeCalibration{})
	story, _ := uc.Render(context.Background(), application.RenderInput{Role: domain.RoleStoryArchitect, Language: "vi"})
	code, _ := uc.Render(context.Background(), application.RenderInput{Role: domain.RoleRemotionEngineer, Language: "vi"})
	if !strings.Contains(code.Prompt, "`narrations`") || strings.Contains(story.Prompt, "narrations") {
		t.Fatalf("story rule %q, code rule %q", story.Prompt, code.Prompt)
	}
	if strings.Contains(story.Prompt, "self.narrate") || strings.Contains(code.Prompt, "self.narrate") {
		t.Fatal("no narration rule may mention self.narrate")
	}
}
