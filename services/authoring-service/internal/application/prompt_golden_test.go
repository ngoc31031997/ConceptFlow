package application_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"authoring/internal/application"
	"authoring/internal/domain"
)

// seededStore serves each role's shipped template, the row a fresh database
// holds — so the golden comparison covers the seed text, not a test double.
type seededStore struct{}

func (seededStore) GetActive(_ context.Context, role domain.PromptRole) (domain.Prompt, error) {
	for _, p := range domain.SystemPrompts() {
		if p.Role == role {
			return p, nil
		}
	}
	return domain.Prompt{}, application.ErrPromptNotFound
}

// CR-040 FR113.2: testdata/prompt_golden.json holds what web-gui's TypeScript
// builders produced for each language × input, generated before that code was
// deleted. The server must reproduce it byte for byte.
func TestRender_MatchesWhatTheBrowserUsedToBuild(t *testing.T) {
	raw, err := os.ReadFile("../domain/testdata/prompt_golden.json")
	if err != nil {
		t.Fatal(err)
	}
	var golden struct {
		Prompts []struct {
			Role  string `json:"role"`
			Input struct {
				Language string `json:"language"`
				Script   string `json:"script"`
				Topic    string `json:"topic"`
			} `json:"input"`
			Expected string `json:"expected"`
		} `json:"prompts"`
	}
	if err := json.Unmarshal(raw, &golden); err != nil {
		t.Fatal(err)
	}
	if len(golden.Prompts) == 0 {
		t.Fatal("golden file has no cases")
	}

	uc := application.NewRenderPromptUseCase(seededStore{}, &fakeRenderContext{}, &fakeFormats{}, &fakeCalibration{})
	for _, c := range golden.Prompts {
		got, err := uc.Render(context.Background(), application.RenderInput{
			Role: domain.PromptRole(c.Role), Language: c.Input.Language, Script: c.Input.Script, Topic: c.Input.Topic,
		})
		if err != nil {
			t.Fatalf("%s/%s: %v", c.Role, c.Input.Language, err)
		}
		if got.Prompt != c.Expected {
			t.Errorf("%s lang=%s script=%q topic=%q differs from the TypeScript output (got %d bytes, want %d)",
				c.Role, c.Input.Language, c.Input.Script, c.Input.Topic, len(got.Prompt), len(c.Expected))
		}
	}
}

func TestRender_PastedTextIsNeverExpanded(t *testing.T) {
	uc := application.NewRenderPromptUseCase(&templateStore{text: "A={{script}} B={{topic}}"}, &fakeRenderContext{}, &fakeFormats{}, &fakeCalibration{})
	got, err := uc.Render(context.Background(), application.RenderInput{
		Role: domain.RoleManimAdjust, Script: "x {{topic}} y $& z", Topic: "T",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Prompt != "A=x {{topic}} y $& z B=T" {
		t.Fatalf("got %q", got.Prompt)
	}
}

func TestRender_RemotionRolesUseTheRemotionNarrationRule(t *testing.T) {
	uc := application.NewRenderPromptUseCase(&templateStore{text: "{{narration_language_rule}}"}, &fakeRenderContext{}, &fakeFormats{}, &fakeCalibration{})
	manim, _ := uc.Render(context.Background(), application.RenderInput{Role: domain.RoleManimEngineer, Language: "vi"})
	remotion, _ := uc.Render(context.Background(), application.RenderInput{Role: domain.RoleRemotionEngineer, Language: "vi"})
	if manim.Prompt == remotion.Prompt {
		t.Fatal("Remotion has no self.narrate(...): its rule must differ from Manim's")
	}
}
