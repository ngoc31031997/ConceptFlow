package application

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"visual/internal/domain"
)

const astro = "AST-ASTRO-001"

// drawableAsset puts an asset with its detail in an approved list, so it can be drawn.
func (f *sbFixture) drawableAsset() {
	f.t.Helper()
	f.must(f.desk.AddAsset(f.ctx, project, jfields(f.t, map[string]any{"concept_id": "astro", "name": "Astro", "portrayal": "phi hành gia"})))
	f.must(f.desk.ApproveAssets(f.ctx, project))
}

// drawCalls counts the turns of asset drawings that reached the model.
func (f *sbFixture) drawCalls() int {
	n := 0
	for _, c := range f.llm.asked {
		if c.Role == AssetDrawRole {
			n++
		}
	}
	return n
}

func assetReq(action string, body any) AIRunRequest {
	b, _ := json.Marshal(body)
	return AIRunRequest{Method: http.MethodPost, Path: "/v1/projects/" + project + "/storyboard/assets/" + astro + "/" + action, Body: string(b)}
}

func detailOf(t *testing.T, run AIRun) AssetDrawError {
	t.Helper()
	var d AssetDrawError
	if err := json.Unmarshal(run.ErrorDetail, &d); err != nil {
		t.Fatalf("the run carries no structured error: %s (%v)", run.ErrorDetail, err)
	}
	return d
}

func TestAssetDraw_OnTheAPIChannelBrokenCodeStopsTheRunWithItsErrorAndTheModelIsAskedOnce(t *testing.T) {
	f := newSbFixture(t)
	f.drawableAsset()
	f.llm.always[AssetDrawRole] = "```tsx\n" + brokenCode + "\n```"
	runs, store := newRuns(f)
	run, err := runs.Start(f.ctx, project, "Vẽ Astro", assetReq("draw", map[string]any{}), AIRunAPI)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := store.AIRun(f.ctx, project, run.ID)
	if got.Status != AIRunUnreadable || got.Error != "Thiếu dấu đóng thẻ <g>." || len(got.Preview) != 0 {
		t.Fatalf("broken code leaves an unreadable run with rendering's message: %+v", got)
	}
	d := detailOf(t, got)
	if d.Kind != AssetDrawSyntax || d.Line == nil || *d.Line != 2 || d.Snippet == nil || !strings.Contains(*d.Snippet, "> BROKEN") {
		t.Errorf("the structured error is kept whole: %+v", d)
	}
	if n := f.drawCalls(); n != 1 {
		t.Errorf("the model must be asked exactly once, no retry; got %d calls", n)
	}
	if len(f.rendering.drawChecks) != 1 || len(f.rendering.drawApplies) != 0 || f.rendering.assets[0].Version != 0 {
		t.Errorf("one check, nothing stored: checks %d applies %d version %d", len(f.rendering.drawChecks), len(f.rendering.drawApplies), f.rendering.assets[0].Version)
	}
}

func TestAssetDraw_OnTheAPIChannelAGoodDrawingIsPreviewedThenStoredOnlyOnApply(t *testing.T) {
	f := newSbFixture(t)
	f.drawableAsset()
	runs, store := newRuns(f)
	run, err := runs.Start(f.ctx, project, "Vẽ Astro", assetReq("draw", map[string]any{}), AIRunAPI)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := store.AIRun(f.ctx, project, run.ID)
	if got.Status != AIRunReady || got.ErrorDetail != nil {
		t.Fatalf("a drawing rendering accepts is ready: %+v", got)
	}
	var preview struct {
		Code      string          `json:"code"`
		PNGBase64 string          `json:"png_base64"`
		Warnings  json.RawMessage `json:"warnings"`
	}
	if err := json.Unmarshal(got.Preview, &preview); err != nil || !strings.Contains(preview.Code, "export function Astro") || preview.PNGBase64 != "UE5H" || string(preview.Warnings) != "[]" {
		t.Fatalf("the preview is {code, png_base64, warnings}: %s %v", got.Preview, err)
	}
	if len(f.rendering.drawApplies) != 0 || f.rendering.assets[0].Status != domain.AssetPlanned {
		t.Fatal("nothing is stored before apply")
	}
	applied, state, err := runs.Apply(f.ctx, project, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if applied.Status != AIRunApplied || !strings.Contains(string(state), `"status":"drawn"`) {
		t.Fatalf("applying stores the drawing and answers the screen: %+v", applied)
	}
	if len(f.rendering.drawApplies) != 1 || f.rendering.drawApplies[0].Code != preview.Code || f.rendering.drawApplies[0].RefID != "R12" {
		t.Errorf("draw-apply gets the checked code with the style of step 8: %+v", f.rendering.drawApplies)
	}
	if n := f.drawCalls(); n != 1 {
		t.Errorf("applying reads the kept reply and asks no model; got %d calls", n)
	}
}

func TestAssetDraw_TheRunPromptIsRenderingsPromptAndAsksNoModel(t *testing.T) {
	f := newSbFixture(t)
	f.drawableAsset()
	runs, _ := newRuns(f)
	run, err := runs.Start(f.ctx, project, "", assetReq("draw", map[string]any{}), AIRunManual)
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != AIRunWaiting || run.Prompt != "SYSTEM vẽ "+astro+"\n\nVẽ hình Astro" {
		t.Fatalf("the prompt is rendering's system and user turn: %+v", run)
	}
	if len(f.llm.calls) != 0 || len(f.rendering.drawChecks) != 0 {
		t.Error("building the prompt asks no model and checks nothing")
	}
}

func TestAssetDraw_APastedReplyIsCheckedAndABrokenOneKeepsItsErrorUntilAGoodOne(t *testing.T) {
	f := newSbFixture(t)
	f.drawableAsset()
	runs, _ := newRuns(f)
	run, err := runs.Start(f.ctx, project, "", assetReq("draw", map[string]any{}), AIRunExtension)
	if err != nil {
		t.Fatal(err)
	}
	run, err = runs.Reply(f.ctx, project, run.ID, "```tsx\n"+brokenCode+"\n```")
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != AIRunUnreadable || detailOf(t, run).Kind != AssetDrawSyntax {
		t.Fatalf("a broken pasted drawing is unreadable with its error: %+v", run)
	}
	run, err = runs.Reply(f.ctx, project, run.ID, goodDrawing)
	if err != nil || run.Status != AIRunReady || run.ErrorDetail != nil {
		t.Fatalf("a good reply clears the error: %+v %v", run, err)
	}
	if len(f.rendering.drawApplies) != 0 {
		t.Fatal("nothing is stored before apply")
	}
}

func TestAssetDraw_ADrawingRefusedOnApplyLeavesTheRunUnreadableWithItsError(t *testing.T) {
	f := newSbFixture(t)
	f.drawableAsset()
	runs, _ := newRuns(f)
	run, _ := runs.Start(f.ctx, project, "", assetReq("draw", map[string]any{}), AIRunManual)
	run, _ = runs.Reply(f.ctx, project, run.ID, goodDrawing)
	f.rendering.refuseApply = true
	got, _, err := runs.Apply(f.ctx, project, run.ID)
	var refused *AIRunHTTPError
	if !errors.As(err, &refused) || refused.Status != http.StatusUnprocessableEntity {
		t.Fatalf("the step's 422 is answered: %v", err)
	}
	if got.Status != AIRunUnreadable || detailOf(t, got).Kind != AssetDrawSyntax {
		t.Errorf("the run is unreadable with the structured error: %+v", got)
	}
}

func TestAssetDraw_FixSendsTheBrokenCodeAndItsErrorInANewTurn(t *testing.T) {
	f := newSbFixture(t)
	f.drawableAsset()
	runs, _ := newRuns(f)
	line := 2
	fix := map[string]any{"code": "const a = 1;\n" + brokenCode, "error": AssetDrawError{Kind: AssetDrawSyntax, Message: "Thiếu dấu đóng thẻ <g>.", Line: &line}}
	run, err := runs.Start(f.ctx, project, "", assetReq("fix", fix), AIRunManual)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(run.Prompt, brokenCode) || !strings.Contains(run.Prompt, "Thiếu dấu đóng thẻ <g>.") {
		t.Fatalf("the prompt carries the broken code and its error: %q", run.Prompt)
	}
	p := f.rendering.drawPrompts[len(f.rendering.drawPrompts)-1]
	if p.FixError == nil || p.FixError.Error.Line == nil || *p.FixError.Error.Line != 2 || p.Redraw {
		t.Errorf("draw-prompt gets fix_error on a first drawing: %+v", p)
	}
	if _, err := f.desk.FixAsset(f.ctx, project, astro, AssetDrawFix{Code: "x", Error: AssetDrawError{Kind: "typo", Message: "m"}}, false); !errors.Is(err, domain.ErrInvalidEdit) {
		t.Errorf("an error of an unknown kind: want ErrInvalidEdit, got %v", err)
	}
	if _, err := f.desk.FixAsset(f.ctx, project, astro, AssetDrawFix{Error: AssetDrawError{Kind: AssetDrawNoCode, Message: " "}}, false); !errors.Is(err, domain.ErrInvalidEdit) {
		t.Errorf("an error with no message: want ErrInvalidEdit, got %v", err)
	}
}

func TestAssetDraw_ARedrawCarriesTheNoteAndNeedsADrawing(t *testing.T) {
	f := newSbFixture(t)
	f.drawableAsset()
	var re *RenderingError
	if _, err := f.desk.RedrawAsset(f.ctx, project, astro, "to hơn"); !errors.As(err, &re) || re.Status != http.StatusConflict {
		t.Fatalf("redrawing an asset never drawn: want rendering's 409, got %v", err)
	}
	if f.drawCalls() != 0 {
		t.Error("no model is asked when rendering refuses the prompt")
	}
	f.must(f.desk.DrawAsset(f.ctx, project, astro))
	f.must(f.desk.RedrawAsset(f.ctx, project, astro, " to hơn "))
	p := f.rendering.drawPrompts[len(f.rendering.drawPrompts)-1]
	if !p.Redraw || p.Note != "to hơn" || string(p.Kit) == "" || p.RefID != "R12" {
		t.Errorf("the redraw prompt request: %+v", p)
	}
	if f.rendering.assets[0].Version != 2 {
		t.Errorf("each applied drawing is a version: %d", f.rendering.assets[0].Version)
	}
}

func TestAssetDraw_BrokenCodeOutsideARunIsRefusedOnceAndStoresNothing(t *testing.T) {
	f := newSbFixture(t)
	f.drawableAsset()
	f.llm.always[AssetDrawRole] = brokenCode
	var drawn *AssetDrawCheckError
	if _, err := f.desk.DrawAsset(f.ctx, project, astro); !errors.As(err, &drawn) || drawn.Detail.Kind != AssetDrawSyntax {
		t.Fatalf("want the structured error, got %v", err)
	}
	if f.drawCalls() != 1 || len(f.rendering.drawApplies) != 0 {
		t.Errorf("one model call, nothing stored: calls %d applies %d", f.drawCalls(), len(f.rendering.drawApplies))
	}
}

func TestAssetDraw_FixKeepsThePreconditionsOfADrawing(t *testing.T) {
	f := newSbFixture(t)
	f.must(f.desk.AddAsset(f.ctx, project, jfields(t, map[string]any{"concept_id": "astro", "name": "Astro"})))
	fix := AssetDrawFix{Error: AssetDrawError{Kind: AssetDrawNoCode, Message: "Câu trả lời không có khối code tsx."}}
	if _, err := f.desk.FixAsset(f.ctx, project, astro, fix, false); !errors.Is(err, domain.ErrAssetNoDetail) {
		t.Errorf("a first drawing without detail: want ErrAssetNoDetail, got %v", err)
	}
	if _, err := f.desk.FixAsset(f.ctx, project, astro, fix, true); !errors.Is(err, domain.ErrAssetListNotApproved) {
		t.Errorf("a list not approved: want ErrAssetListNotApproved, got %v", err)
	}
	if _, err := f.desk.FixAsset(f.ctx, project, "AST-GHOST-001", fix, false); !errors.Is(err, domain.ErrUnknownAsset) {
		t.Errorf("an unknown asset: want ErrUnknownAsset, got %v", err)
	}
	if f.drawCalls() != 0 || len(f.rendering.drawPrompts) != 0 {
		t.Error("a refused drawing asks neither rendering nor the model")
	}
}

func TestAssetDraw_AnEmptyModelReplyFailsTheRunWithoutAStructuredError(t *testing.T) {
	f := newSbFixture(t)
	f.drawableAsset()
	f.llm.always[AssetDrawRole] = "  "
	runs, store := newRuns(f)
	run, err := runs.Start(f.ctx, project, "", assetReq("draw", map[string]any{}), AIRunAPI)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := store.AIRun(f.ctx, project, run.ID)
	if got.Status != AIRunFailed || got.ErrorDetail != nil || len(f.rendering.drawChecks) != 0 {
		t.Errorf("an empty reply is a failed call, never checked: %+v", got)
	}
}
