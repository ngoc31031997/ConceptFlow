package application_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"orchestrator/internal/application"
	"orchestrator/internal/domain"
)

type stubFinalizer struct {
	out      string
	repaired bool
	usage    application.TokenUsage
	err      error
	gotIn    string
}

func (f *stubFinalizer) FinalizeStoryboard(_ context.Context, content, _ string, _ int) (application.FinalizedStoryboard, error) {
	f.gotIn = content
	if f.err != nil {
		return application.FinalizedStoryboard{}, f.err
	}
	return application.FinalizedStoryboard{Storyboard: f.out, Shots: 3, Repaired: f.repaired, Usage: f.usage}, nil
}

type stubCodegen struct {
	req    application.CodeGenRequest
	result application.CodeGenResult
	err    error
	events []application.CodeEvent
	calls  int
}

func (c *stubCodegen) GenerateCode(_ context.Context, req application.CodeGenRequest, on func(application.CodeEvent)) (application.CodeGenResult, error) {
	c.calls++
	c.req = req
	for _, e := range c.events {
		on(e)
	}
	return c.result, c.err
}

type usagePort struct {
	mu   sync.Mutex
	rows []application.LLMUsageRecord
}

func (u *usagePort) RecordLLMUsage(_ context.Context, r application.LLMUsageRecord) error {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.rows = append(u.rows, r)
	return nil
}

func codeFixture(t *testing.T, engine domain.RenderEngine, storyboard string) (
	*application.GenerateAuthoringUseCase, *contentSaver, *contentSaver, *usagePort,
) {
	t.Helper()
	renderCtx := &fakeRenderContext{
		project: &domain.Project{ProjectID: "p1", ContentLanguage: domain.LanguageVietnamese, RenderEngine: engine},
		topic:   "TCP và UDP", storyboard: storyboard,
	}
	sb, code := &contentSaver{}, &contentSaver{}
	usage := &usagePort{}
	uc := application.NewGenerateAuthoringUseCase(
		newRenderer("SYSTEM {{topic}}", renderCtx), &stubProvider{content: "x"},
		application.NewLLMUsageRecorder(usage, nil), renderCtx, nil,
		&recordingSaver{}, sb, code, 0, 16000,
	)
	return uc, sb, code, usage
}

func TestStoryboardStepSavesTheCanonicalJSONAndBillsTheRepairTurn(t *testing.T) {
	provider := &stubProvider{content: "  {raw json}  "}
	uc, sb, _, usage := codeFixture(t, domain.RenderEngineRemotion, "")
	uc = application.NewGenerateAuthoringUseCase(
		newRenderer("SYSTEM {{topic}}", &fakeRenderContext{project: &domain.Project{
			ProjectID: "p1", RenderEngine: domain.RenderEngineRemotion}}),
		provider, application.NewLLMUsageRecorder(usage, nil), &fakeRenderContext{project: &domain.Project{
			ProjectID: "p1", RenderEngine: domain.RenderEngineRemotion}}, nil,
		&recordingSaver{}, sb, &contentSaver{}, 0, 16000,
	).WithPipeline(&stubFinalizer{out: "{canonical}", repaired: true,
		usage: application.TokenUsage{Model: "m", PromptTokens: 7, CompletionTokens: 3}}, &stubCodegen{})

	got, err := uc.Execute(context.Background(), "p1", "storyboard")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if got.Content != "{canonical}" || sb.content != "{canonical}" {
		t.Errorf("content=%q saved=%q, want the finalizer's canonical JSON", got.Content, sb.content)
	}
	if got.Usage.PromptTokens != 17 || got.Usage.CompletionTokens != 8 {
		t.Errorf("usage = %+v, want the chat (10/5) plus the repair turn (7/3)", got.Usage)
	}
	var phases []string
	for _, r := range usage.rows {
		phases = append(phases, r.Phase)
	}
	if len(phases) != 2 || phases[0] != "" || phases[1] != "storyboard_fix" {
		t.Errorf("usage rows phases = %v, want the chat then storyboard_fix", phases)
	}
}

func TestStoryboardStepNeverSavesAnInvalidStoryboard(t *testing.T) {
	uc, sb, _, _ := codeFixture(t, domain.RenderEngineManim, "")
	uc.WithPipeline(&stubFinalizer{err: &application.LLMError{
		Kind: application.ErrKindMalformed, Provider: "llm-service", Err: errors.New("still invalid"),
		Usage: application.TokenUsage{PromptTokens: 4},
	}}, &stubCodegen{})

	_, err := uc.Execute(context.Background(), "p1", "storyboard")
	if application.LLMErrorKindOf(err) != application.ErrKindMalformed {
		t.Fatalf("err = %v, want malformed", err)
	}
	if sb.calls != 0 {
		t.Errorf("saver called %d times, want 0 — an invalid storyboard must not be stored", sb.calls)
	}
}

func TestStoryboardStepWithoutAFinalizerFailsLoudly(t *testing.T) {
	uc, sb, _, _ := codeFixture(t, domain.RenderEngineManim, "")
	if _, err := uc.Execute(context.Background(), "p1", "storyboard"); err == nil || !strings.Contains(err.Error(), "not wired") {
		t.Fatalf("err = %v, want a not-wired error rather than saving unvalidated output", err)
	}
	if sb.calls != 0 {
		t.Errorf("saver called %d times, want 0", sb.calls)
	}
}

func TestCodeStepRunsThePipelineAndBillsEveryCall(t *testing.T) {
	uc, _, code, usage := codeFixture(t, domain.RenderEngineRemotion, `{"scenes":[]}`)
	gen := &stubCodegen{
		result: application.CodeGenResult{
			Code: "export const narrations = []", CheckOK: true, RepairRounds: 1,
			Calls: []application.CodeCall{
				{Phase: "layout", Label: "LAYOUT", OK: true, Usage: application.TokenUsage{Model: "m", PromptTokens: 100, CompletionTokens: 10}},
				{Phase: "chunk", Label: "1.1-1.10", OK: true, Usage: application.TokenUsage{Model: "m", PromptTokens: 200, CompletionTokens: 50}},
				{Phase: "chunk", Label: "cached", OK: true, Cached: true},
				{Phase: "repair", Label: "1.3", OK: false, ErrorKind: application.ErrKindMalformed, Usage: application.TokenUsage{Model: "m", PromptTokens: 30}},
			},
		},
		events: []application.CodeEvent{{Type: "phase", Phase: "chunks", Total: 3}, {Type: "chunk_done", Done: 2, Total: 3}},
	}
	uc.WithPipeline(&stubFinalizer{}, gen)

	got, err := uc.Execute(context.Background(), "p1", "code")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if gen.req.Engine != "remotion" || gen.req.Storyboard != `{"scenes":[]}` || gen.req.Topic != "TCP và UDP" {
		t.Errorf("pipeline request = %+v", gen.req)
	}
	if !strings.HasPrefix(gen.req.System, "SYSTEM") {
		t.Errorf("system prompt = %q, want the rendered *_engineer_ai prompt", gen.req.System)
	}
	if got.Role != string(domain.RoleRemotionEngineerAI) {
		t.Errorf("role = %q, want remotion_engineer_ai", got.Role)
	}
	if code.content != "export const narrations = []" || got.CheckFailed {
		t.Errorf("saved=%q checkFailed=%v", code.content, got.CheckFailed)
	}
	if got.Usage.PromptTokens != 330 || got.ModelCalls != 4 || got.RepairRounds != 1 {
		t.Errorf("usage=%+v calls=%d rounds=%d", got.Usage, got.ModelCalls, got.RepairRounds)
	}
	// One row per BILLED call: the cached chunk cost nothing and has no row; the
	// failed repair is billed and recorded as failed.
	if len(usage.rows) != 3 {
		t.Fatalf("usage rows = %d, want 3 (layout, chunk, failed repair)", len(usage.rows))
	}
	if usage.rows[0].Phase != "layout" || usage.rows[2].Phase != "repair" || usage.rows[2].OK || usage.rows[2].ErrorKind != application.ErrKindMalformed {
		t.Errorf("rows = %+v", usage.rows)
	}
}

func TestCodeStepUsesTheManimEngineerForAManimProject(t *testing.T) {
	uc, _, _, _ := codeFixture(t, domain.RenderEngineManim, `{"scenes":[]}`)
	uc.WithPipeline(&stubFinalizer{}, &stubCodegen{result: application.CodeGenResult{Code: "from conceptflow import *", CheckOK: true}})
	got, err := uc.Execute(context.Background(), "p1", "code")
	if err != nil || got.Role != string(domain.RoleManimEngineerAI) {
		t.Fatalf("err=%v role=%q, want manim_engineer_ai", err, got.Role)
	}
}

func TestCodeStepSavesAScriptThatStillFailsTheCheckButFlagsIt(t *testing.T) {
	uc, _, code, _ := codeFixture(t, domain.RenderEngineRemotion, `{"scenes":[]}`)
	uc.WithPipeline(&stubFinalizer{}, &stubCodegen{result: application.CodeGenResult{
		Code: "broken", CheckOK: false, RepairRounds: 3,
		Diagnostics: []application.CodeDiagnostic{{Message: "TS2304: Cannot find name 'x'.", Line: 41}, {Message: "no line"}},
	}})

	got, err := uc.Execute(context.Background(), "p1", "code")
	if err != nil {
		t.Fatalf("Execute: %v — the tokens are spent, the script must still come back", err)
	}
	if code.content != "broken" || !got.CheckFailed || got.RepairRounds != 3 {
		t.Errorf("saved=%q checkFailed=%v rounds=%d", code.content, got.CheckFailed, got.RepairRounds)
	}
	if len(got.Diagnostics) != 2 || got.Diagnostics[0] != "dòng 41: TS2304: Cannot find name 'x'." || got.Diagnostics[1] != "no line" {
		t.Errorf("diagnostics = %v", got.Diagnostics)
	}
}

func TestCodeStepBillsTheCallsOfARunThatFailed(t *testing.T) {
	uc, _, code, usage := codeFixture(t, domain.RenderEngineRemotion, `{"scenes":[]}`)
	uc.WithPipeline(&stubFinalizer{}, &stubCodegen{err: &application.LLMError{
		Kind: application.ErrKindBalance, Provider: "hive", Err: errors.New("no credit"),
		Calls: []application.CodeCall{
			{Phase: "layout", OK: true, Usage: application.TokenUsage{Model: "m", PromptTokens: 100}},
			{Phase: "chunk", OK: false, ErrorKind: application.ErrKindBalance},
		},
	}})

	_, err := uc.Execute(context.Background(), "p1", "code")
	if application.LLMErrorKindOf(err) != application.ErrKindBalance {
		t.Fatalf("err = %v, want the balance error passed through", err)
	}
	if code.calls != 0 {
		t.Errorf("saver called %d times, want 0", code.calls)
	}
	if len(usage.rows) != 2 || usage.rows[0].PromptTokens != 100 {
		t.Errorf("rows = %+v, want both calls recorded even though the run failed", usage.rows)
	}
}

func TestCodeStepNeedsAStoryboardAndAPipeline(t *testing.T) {
	uc, _, _, _ := codeFixture(t, domain.RenderEngineRemotion, "   ")
	gen := &stubCodegen{}
	uc.WithPipeline(&stubFinalizer{}, gen)
	if _, err := uc.Execute(context.Background(), "p1", "code"); err == nil || !strings.Contains(err.Error(), "1b") {
		t.Fatalf("err = %v, want a 'run step 1b first' error", err)
	}
	if gen.calls != 0 {
		t.Errorf("pipeline called %d times with no storyboard", gen.calls)
	}

	uc2, _, _, _ := codeFixture(t, domain.RenderEngineRemotion, `{"scenes":[]}`)
	if _, err := uc2.Execute(context.Background(), "p1", "code"); err == nil || !strings.Contains(err.Error(), "not wired") {
		t.Fatalf("err = %v, want a not-wired error", err)
	}
}

func TestCodeStepProgressReflectsChunksAndRepairRounds(t *testing.T) {
	uc, _, _, _ := codeFixture(t, domain.RenderEngineRemotion, `{"scenes":[]}`)
	var seen []application.AuthoringProgress
	gen := &stubCodegen{result: application.CodeGenResult{Code: "x", CheckOK: true}}
	gen.events = []application.CodeEvent{
		{Type: "phase", Phase: "chunks", Total: 4},
		{Type: "chunk_done", Done: 3, Total: 4},
		{Type: "phase", Phase: "repair", Round: 2, Total: 3},
	}
	uc.WithPipeline(&stubFinalizer{}, &progressSpy{inner: gen, uc: uc, out: &seen})

	if _, err := uc.Execute(context.Background(), "p1", "code"); err != nil {
		t.Fatal(err)
	}
	last := seen[len(seen)-1]
	if last.Phase != "repair" || last.ChunksDone != 3 || last.ChunksTotal != 4 || last.RepairRound != 2 || last.RepairMax != 3 {
		t.Errorf("progress = %+v", last)
	}
}

// progressSpy samples the use case's progress after each event, the way the GUI's poll would.
type progressSpy struct {
	inner *stubCodegen
	uc    *application.GenerateAuthoringUseCase
	out   *[]application.AuthoringProgress
}

func (p *progressSpy) GenerateCode(ctx context.Context, req application.CodeGenRequest, on func(application.CodeEvent)) (application.CodeGenResult, error) {
	wrapped := func(e application.CodeEvent) {
		on(e)
		*p.out = append(*p.out, p.uc.Progress("p1", "code"))
	}
	return p.inner.GenerateCode(ctx, req, wrapped)
}
