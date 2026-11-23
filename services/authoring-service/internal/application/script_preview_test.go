package application

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"authoring/internal/domain"
)

// externalScriptDesk is the fixture's script step reading its replies through the external-AI provider.
func externalScriptDesk(f *scriptFixture) *ScriptDeskUseCase {
	project := &briefProject{project: domain.Project{ContentLanguage: domain.LanguageVietnamese, VideoOutputMode: domain.ModeLongOnly, VoiceID: "vi-HoaiMy"}, status: domain.StatusDraft}
	provider := NewExternalAIProvider(f.llm, &pastedReplier{})
	return NewScriptDeskUseCase(f.beat.store, project, project, briefPrompts{}, provider, 4000, f.beat.uc).WithVoices(f.voices, f.dispatcher)
}

func storedScriptUnits(t *testing.T, f *scriptFixture) int {
	t.Helper()
	units, err := f.beat.units.ListUnits(context.Background(), "p1", domain.UnitKindScriptBeat)
	if err != nil {
		t.Fatal(err)
	}
	return len(units)
}

func TestWritingABeatInPreviewModeStopsWithTheLinesAndStoresNothing(t *testing.T) {
	f := newScriptFixture(t)
	uc := externalScriptDesk(f)
	run := NewExternalAIRun(ExternalAIPreview, scriptReply("Câu một.", "Câu hai."))
	_, err := uc.ScriptUnitKind().Redo(WithExternalAI(context.Background(), run), "p1", "b1", "")
	if !errors.Is(err, ErrExternalAIPreviewReady) {
		t.Fatalf("err = %v, want ErrExternalAIPreviewReady", err)
	}
	raw, ok := run.Preview()
	if !ok {
		t.Fatal("no preview was kept")
	}
	var got domain.ScriptBeat
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Lines) != 2 || got.Lines[0].ID != "b1.1" || got.Lines[1].Text != "Câu hai." || got.NextSeq != 3 {
		t.Errorf("preview = %s, want lines b1.1 and b1.2", raw)
	}
	if n := storedScriptUnits(t, f); n != 0 {
		t.Errorf("%d dialogues stored in preview mode, want none", n)
	}
}

func TestRedoingABeatInPreviewModeKeepsTheStoredDialogue(t *testing.T) {
	f := newScriptFixture(t)
	f.llm.replies[scriptWrite] = scriptReply("Câu một.", "Câu hai.")
	if err := f.uc.Generate(context.Background(), "p1", "b1", ""); err != nil {
		t.Fatal(err)
	}
	uc := externalScriptDesk(f)
	run := NewExternalAIRun(ExternalAIPreview, scriptReply("Câu mới."))
	if _, err := uc.ScriptUnitKind().Redo(WithExternalAI(context.Background(), run), "p1", "b1", "ngắn hơn"); !errors.Is(err, ErrExternalAIPreviewReady) {
		t.Fatalf("err = %v, want ErrExternalAIPreviewReady", err)
	}
	if raw, _ := run.Preview(); !json.Valid(raw) || len(raw) == 0 {
		t.Errorf("preview = %s", raw)
	}
	if s := f.script(t, "b1"); len(s.Lines) != 2 || s.Lines[0].Text != "Câu một." {
		t.Errorf("stored dialogue = %+v, want it unchanged", s)
	}
}

func TestWritingABeatInPasteModeStores(t *testing.T) {
	f := newScriptFixture(t)
	uc := externalScriptDesk(f)
	run := NewExternalAIRun(ExternalAIPaste, scriptReply("Câu một."))
	if _, err := uc.ScriptUnitKind().Redo(WithExternalAI(context.Background(), run), "p1", "b1", ""); err != nil {
		t.Fatalf("Redo: %v", err)
	}
	if s := f.script(t, "b1"); len(s.Lines) != 1 || s.Lines[0].Text != "Câu một." {
		t.Errorf("stored = %+v, want the pasted line", s)
	}
}

func flaggedLineFixture(t *testing.T) *scriptFixture {
	t.Helper()
	f := newScriptFixture(t)
	if _, err := f.uc.ScriptUnitKind().Edit(context.Background(), "p1", "b1", map[string]string{ScriptFieldAddStart: "Chất CO2 rất nhiều."}); err != nil {
		t.Fatal(err)
	}
	return f
}

func TestSuggestingInPreviewModeStopsWithTheTextAndStoresNothingNew(t *testing.T) {
	f := flaggedLineFixture(t)
	uc := externalScriptDesk(f)
	run := NewExternalAIRun(ExternalAIPreview, `{"text":"Chất các-bon đi-ô-xít rất nhiều."}`)
	_, err := uc.Suggest(WithExternalAI(context.Background(), run), "p1", "line", "b1.1")
	if !errors.Is(err, ErrExternalAIPreviewReady) {
		t.Fatalf("err = %v, want ErrExternalAIPreviewReady", err)
	}
	raw, ok := run.Preview()
	if !ok || string(raw) != `{"text":"Chất các-bon đi-ô-xít rất nhiều."}` {
		t.Errorf("preview = %s", raw)
	}
}

func TestSuggestingInPasteModeReturnsTheText(t *testing.T) {
	f := flaggedLineFixture(t)
	uc := externalScriptDesk(f)
	run := NewExternalAIRun(ExternalAIPaste, `{"text":"Chất các-bon đi-ô-xít rất nhiều."}`)
	text, err := uc.Suggest(WithExternalAI(context.Background(), run), "p1", "line", "b1.1")
	if err != nil || text != "Chất các-bon đi-ô-xít rất nhiều." {
		t.Errorf("text=%q err=%v", text, err)
	}
}
