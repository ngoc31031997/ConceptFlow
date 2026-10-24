package application

import (
	"context"
	"strings"
	"testing"

	"authoring/internal/domain"
)

func TestProjectIdeaGenerateGivesTheAIOnlyCleanedText(t *testing.T) {
	f := newProjectIdeaFixture()
	// Text kept before the cleaning existed still holds straight quotes.
	f.input.in.Subject = `Vì sao người ta nói "ăn sung là ăn xác ong"? C:\tmp`
	got, err := f.uc.Generate(context.Background(), "p1", `nhấn mạnh "ngộ nhận"`)
	if err != nil {
		t.Fatal(err)
	}
	prompt := f.llm.systems[0]
	if !strings.Contains(prompt, "Vì sao người ta nói “ăn sung là ăn xác ong”? C:/tmp") {
		t.Errorf("the cleaned text is not in the prompt:\n%s", prompt)
	}
	if !strings.Contains(prompt, "nhấn mạnh “ngộ nhận”") {
		t.Error("the redo note was not cleaned")
	}
	if strings.Contains(prompt, `"ăn sung`) || strings.Contains(prompt, `"ngộ nhận"`) {
		t.Error("a straight quote around the Creator's words reached the prompt")
	}
	if got.InputHash != domain.HashInput(f.input.in.Subject) {
		t.Error("the idea does not remember the text it was written from")
	}
}

func TestProjectIdeaStateIgnoresCleaningWhenComparingTheText(t *testing.T) {
	f := newProjectIdeaFixture()
	f.input.in.Subject = `nói "a" đi`
	ctx := context.Background()
	if _, err := f.uc.Generate(ctx, "p1", ""); err != nil {
		t.Fatal(err)
	}
	// The orchestrator now keeps the cleaned form of the same text.
	f.input.in.Subject = "nói “a” đi"
	state, err := f.uc.State(ctx, "p1")
	if err != nil {
		t.Fatal(err)
	}
	if state.InputChanged {
		t.Error("cleaning the stored text made the idea look written from other text")
	}
}

func TestProjectIdeaEditCleansWhatTheCreatorTyped(t *testing.T) {
	f := newProjectIdeaFixture()
	ctx := context.Background()
	q, scope := `Có thật là "kéo dài như sợi mì" không?`, `Không đi vào C:\toán`
	points := []string{`Điểm "một"`, ` `, `Điểm hai`}
	got, err := f.uc.Edit(ctx, "p1", IdeaPatch{CentralQuestion: &q, Scope: &scope, CuriosityPoints: &points})
	if err != nil {
		t.Fatal(err)
	}
	if got.Idea.CentralQuestion != "Có thật là “kéo dài như sợi mì” không?" || got.Idea.Scope != "Không đi vào C:/toán" {
		t.Errorf("text not cleaned: %+v", got.Idea)
	}
	if len(got.Idea.CuriosityPoints) != 2 || got.Idea.CuriosityPoints[0] != "Điểm “một”" {
		t.Errorf("points not cleaned: %v", got.Idea.CuriosityPoints)
	}

	// A redo hands the cleaned fields to the AI as the ones to keep.
	if _, err := f.uc.Generate(ctx, "p1", ""); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(f.llm.systems[0], `"kéo dài`) {
		t.Error("a straight quote came back through the kept fields")
	}
}
