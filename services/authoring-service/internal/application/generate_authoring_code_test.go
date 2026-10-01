package application_test

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"sync"
	"testing"

	"authoring/internal/application"
	"authoring/internal/domain"
)

type stubFinalizer struct {
	out      string
	repaired bool
	usage    application.TokenUsage
	err      error
	gotIn    string
}

func (f *stubFinalizer) FinalizeStoryboard(_ context.Context, content, _ string, _ int, _ domain.Frame) (application.FinalizedStoryboard, error) {
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
	res := c.result
	if res.Status == "" && c.err == nil {
		res.Status = application.CodeGenDone // the tests that only care about the merged code
	}
	return res, c.err
}

// PlanSegments stands in for llm-service's cut: the frame, then every
// ChunkShots shots; a storyboard with a layout gives the Remotion frame.
func (c *stubCodegen) PlanSegments(_ context.Context, req application.CodeGenRequest) ([]domain.CodeSegment, error) {
	scenes, err := domain.ParseStoryboardScenes(req.Storyboard)
	if err != nil {
		return nil, &application.LLMError{Kind: application.ErrKindMalformed, Provider: "llm-service", Err: err}
	}
	var ids []string
	for _, sc := range scenes {
		for _, sh := range sc.Shots {
			ids = append(ids, sh.ID)
		}
	}
	frame := domain.CodeSegment{Key: "frame", Kind: domain.SegmentKindFrame, Shots: []string{}, Status: domain.SegmentPending}
	if req.Engine == "remotion" && strings.Contains(req.Storyboard, `"layout"`) {
		frame.Source = domain.SegmentSourceStoryboard
	}
	out := []domain.CodeSegment{frame}
	for i := 0; i < len(ids); i += req.ChunkShots {
		chunk := ids[i:min(i+req.ChunkShots, len(ids))]
		out = append(out, domain.CodeSegment{Key: chunk[0] + "-" + chunk[len(chunk)-1], Kind: domain.SegmentKindShots,
			Position: len(out), Shots: chunk, Fingerprint: "fp", Status: domain.SegmentPending})
	}
	return out, c.err
}

func (c *stubCodegen) SegmentPrompt(_ context.Context, req application.CodeGenRequest, key string) (string, string, error) {
	c.req = req
	return req.System, "USER " + key, c.err
}

func (c *stubCodegen) ParseSegment(_ context.Context, req application.CodeGenRequest, key, reply string) (string, json.RawMessage, error) {
	c.req = req
	if c.err != nil {
		return "", nil, c.err
	}
	return "fp-" + key, json.RawMessage(`{"reply":` + strconv.Quote(reply) + `}`), nil
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
	).WithSegments(newFakeSegments())
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

func callEvent(c application.CodeCall) application.CodeEvent {
	return application.CodeEvent{Type: "call", Call: &c}
}

func TestCodeStepRunsThePipelineAndBillsEveryCall(t *testing.T) {
	uc, _, code, usage := codeFixture(t, domain.RenderEngineRemotion, `{"scenes":[]}`)
	gen := &stubCodegen{
		result: application.CodeGenResult{Code: "export const narrations = []", CheckOK: true, RepairRounds: 1},
		events: []application.CodeEvent{
			{Type: "phase", Phase: "chunks", Total: 3},
			callEvent(application.CodeCall{Phase: "layout", Label: "LAYOUT", OK: true, Usage: application.TokenUsage{Model: "m", PromptTokens: 100, CompletionTokens: 10}}),
			callEvent(application.CodeCall{Phase: "chunk", Label: "1.1-1.10", OK: true, Usage: application.TokenUsage{Model: "m", PromptTokens: 200, CompletionTokens: 50}}),
			{Type: "chunk_done", Done: 2, Total: 3},
			callEvent(application.CodeCall{Phase: "repair", Label: "1.3", OK: false, ErrorKind: application.ErrKindMalformed, Usage: application.TokenUsage{Model: "m", PromptTokens: 30}}),
		},
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
	if gen.req.ChunkShots != domain.DefaultChunkShots || gen.req.Only != nil {
		t.Errorf("chunk shots %d only %v, want the default and every missing segment", gen.req.ChunkShots, gen.req.Only)
	}
	if got.Role != string(domain.RoleRemotionEngineerAI) {
		t.Errorf("role = %q, want remotion_engineer_ai", got.Role)
	}
	if code.content != "export const narrations = []" || got.CheckFailed {
		t.Errorf("saved=%q checkFailed=%v", code.content, got.CheckFailed)
	}
	if got.Usage.PromptTokens != 330 || got.ModelCalls != 3 || got.RepairRounds != 1 {
		t.Errorf("usage=%+v calls=%d rounds=%d", got.Usage, got.ModelCalls, got.RepairRounds)
	}
	// One row per billed call, the failed repair recorded as failed.
	if len(usage.rows) != 3 {
		t.Fatalf("usage rows = %d, want 3 (layout, chunk, failed repair)", len(usage.rows))
	}
	if usage.rows[0].Phase != "layout" || usage.rows[2].Phase != "repair" || usage.rows[2].OK || usage.rows[2].ErrorKind != application.ErrKindMalformed {
		t.Errorf("rows = %+v", usage.rows)
	}
}

// The layout check gets the same subtitle strip {{subtitle_zone}}
// describes, and the video's font; Manim gets neither.
func TestCodeStepHandsTheSubtitleBandAndFontToTheLayoutCheck(t *testing.T) {
	run := func(p *domain.Project) application.CodeGenRequest {
		t.Helper()
		renderCtx := &fakeRenderContext{project: p, topic: "t", storyboard: `{"scenes":[]}`}
		gen := &stubCodegen{result: application.CodeGenResult{Code: "x", CheckOK: true}}
		uc := application.NewGenerateAuthoringUseCase(
			newRenderer("SYSTEM {{topic}}", renderCtx), &stubProvider{content: "x"},
			application.NewLLMUsageRecorder(&usagePort{}, nil), renderCtx, nil,
			&recordingSaver{}, &contentSaver{}, &contentSaver{}, 0, 16000,
		).WithPipeline(&stubFinalizer{}, gen).WithSegments(newFakeSegments())
		if _, err := uc.Execute(context.Background(), p.ProjectID, "code"); err != nil {
			t.Fatalf("Execute: %v", err)
		}
		return gen.req
	}

	req := run(&domain.Project{ProjectID: "p1", RenderEngine: domain.RenderEngineRemotion, VideoFont: "Montserrat",
		SubtitleMode: domain.SubtitleModeBurnIn, SubtitleStyle: &domain.SubtitleStyle{FontSize: "large", Position: "top"}})
	if req.SubtitleBand == nil || *req.SubtitleBand != (domain.SubtitleBand{Edge: "top", Px: 280}) || req.VideoFont != "Montserrat" {
		t.Errorf("burn-in remotion request = band %+v font %q", req.SubtitleBand, req.VideoFont)
	}
	req = run(&domain.Project{ProjectID: "p2", RenderEngine: domain.RenderEngineRemotion, SubtitleMode: domain.SubtitleModeTrack})
	if req.SubtitleBand != nil || req.VideoFont != "" {
		t.Errorf("a caption track paints nothing on the frame: band %+v font %q", req.SubtitleBand, req.VideoFont)
	}
	req = run(&domain.Project{ProjectID: "p3", RenderEngine: domain.RenderEngineManim, SubtitleMode: domain.SubtitleModeBurnIn})
	if req.SubtitleBand != nil {
		t.Errorf("manim has no layout check: band %+v", req.SubtitleBand)
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
	uc.WithPipeline(&stubFinalizer{}, &stubCodegen{
		events: []application.CodeEvent{
			callEvent(application.CodeCall{Phase: "layout", OK: true, Usage: application.TokenUsage{Model: "m", PromptTokens: 100}}),
			callEvent(application.CodeCall{Phase: "chunk", OK: false, ErrorKind: application.ErrKindBalance}),
		},
		err: &application.LLMError{Kind: application.ErrKindBalance, Provider: "hive", Err: errors.New("no credit")},
	})

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
	if _, err := uc.Execute(context.Background(), "p1", "code"); err == nil || !strings.Contains(err.Error(), "Bước 4 — Visual") {
		t.Fatalf("err = %v, want a 'run Bước 4 — Visual first' error", err)
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
		// A chunk written again as two halves is still one chunk of
		// the progress; the event changes nothing here.
		{Type: "chunk_split", Index: 4, Total: 4},
		{Type: "phase", Phase: "repair", Round: 2, Total: 3},
	}
	uc.WithPipeline(&stubFinalizer{}, &progressSpy{inner: gen, uc: uc, out: &seen})

	if _, err := uc.Execute(context.Background(), "p1", "code"); err != nil {
		t.Fatal(err)
	}
	if seen[2] != seen[1] {
		t.Errorf("chunk_split changed the progress: %+v -> %+v", seen[1], seen[2])
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

func (p *progressSpy) PlanSegments(ctx context.Context, req application.CodeGenRequest) ([]domain.CodeSegment, error) {
	return p.inner.PlanSegments(ctx, req)
}

func (p *progressSpy) SegmentPrompt(ctx context.Context, req application.CodeGenRequest, key string) (string, string, error) {
	return p.inner.SegmentPrompt(ctx, req, key)
}

func (p *progressSpy) ParseSegment(ctx context.Context, req application.CodeGenRequest, key, reply string) (string, json.RawMessage, error) {
	return p.inner.ParseSegment(ctx, req, key, reply)
}

func (p *progressSpy) GenerateCode(ctx context.Context, req application.CodeGenRequest, on func(application.CodeEvent)) (application.CodeGenResult, error) {
	wrapped := func(e application.CodeEvent) {
		on(e)
		*p.out = append(*p.out, p.uc.Progress("p1", "code"))
	}
	return p.inner.GenerateCode(ctx, req, wrapped)
}

type stubStage struct {
	pending  []domain.ProjectIllustration
	drawings []application.LibraryDrawing
	planned  bool
	prepared int
	gated    int
	reports  []application.StageReport
	model    string
	stale    bool
}

func (s *stubStage) Stale(context.Context, string) (bool, error) { return s.stale, nil }

func (s *stubStage) Prepare(_ context.Context, _ string, model string, report func(application.StageReport)) ([]domain.ProjectIllustration, error) {
	s.prepared++
	s.model = model
	for _, r := range s.reports {
		report(r)
	}
	return s.pending, nil
}
func (s *stubStage) Gate(context.Context, string) ([]domain.ProjectIllustration, bool, error) {
	s.gated++
	return s.pending, s.planned, nil
}
func (s *stubStage) ForCode(context.Context, string) ([]application.LibraryDrawing, error) {
	return s.drawings, nil
}

// A Remotion video waits for its drawings; approved ones reach the pipeline.
func TestCodeStepWaitsForTheVideosDrawingsThenHandsThemOver(t *testing.T) {
	uc, _, _, _ := codeFixture(t, domain.RenderEngineRemotion, `{"scenes":[]}`)
	gen := &stubCodegen{result: application.CodeGenResult{Code: "x", CheckOK: true}}
	st := &stubStage{planned: true, pending: []domain.ProjectIllustration{{Name: "Motorbike", State: domain.PIDrawn}}}
	uc.WithPipeline(&stubFinalizer{}, gen).WithIllustrations(st)

	_, err := uc.Execute(context.Background(), "p1", "code")
	var pending *application.ErrIllustrationsPending
	if !errors.As(err, &pending) || gen.calls != 0 || !strings.Contains(err.Error(), "Motorbike") {
		t.Fatalf("want the step held before any code call, got %v (codegen calls %d)", err, gen.calls)
	}

	st.pending = nil
	st.drawings = []application.LibraryDrawing{{Name: "Motorbike", Code: "export function Motorbike() {}"}}
	if _, err := uc.Execute(context.Background(), "p1", "code"); err != nil {
		t.Fatal(err)
	}
	if len(gen.req.Illustrations) != 1 || gen.req.Illustrations[0].Name != "Motorbike" {
		t.Fatalf("drawings not handed to the pipeline: %+v", gen.req.Illustrations)
	}
}

func TestManimCodeStepSkipsTheDrawingStage(t *testing.T) {
	uc, _, _, _ := codeFixture(t, domain.RenderEngineManim, `{"scenes":[]}`)
	st := &stubStage{pending: []domain.ProjectIllustration{{Name: "X"}}}
	uc.WithPipeline(&stubFinalizer{}, &stubCodegen{result: application.CodeGenResult{Code: "x", CheckOK: true}}).WithIllustrations(st)
	if _, err := uc.Execute(context.Background(), "p1", "code"); err != nil || st.gated != 0 {
		t.Fatalf("manim must not wait for drawings: %v, gated %d", err, st.gated)
	}
}

// The code step does not draw; a list never planned sends the Creator
// to the illustrations step instead.
func TestCodeStepRefusesAVideoWhoseDrawingsWereNeverPlanned(t *testing.T) {
	uc, _, _, _ := codeFixture(t, domain.RenderEngineRemotion, `{"scenes":[]}`)
	gen := &stubCodegen{result: application.CodeGenResult{Code: "x", CheckOK: true}}
	st := &stubStage{}
	uc.WithPipeline(&stubFinalizer{}, gen).WithIllustrations(st)
	_, err := uc.Execute(context.Background(), "p1", "code")
	if !errors.Is(err, application.ErrIllustrationsNotPlanned) || gen.calls != 0 || st.prepared != 0 {
		t.Fatalf("err=%v codegen=%d prepared=%d", err, gen.calls, st.prepared)
	}
}

// A list planned from an older storyboard stops the code step
// before any model call, even when every drawing on it is approved.
func TestCodeStepRefusesADrawingListPlannedFromAnOlderStoryboard(t *testing.T) {
	uc, _, _, _ := codeFixture(t, domain.RenderEngineRemotion, `{"scenes":[]}`)
	gen := &stubCodegen{result: application.CodeGenResult{Code: "x", CheckOK: true}}
	st := &stubStage{planned: true, stale: true}
	uc.WithPipeline(&stubFinalizer{}, gen).WithIllustrations(st)
	_, err := uc.Execute(context.Background(), "p1", "code")
	if !errors.Is(err, application.ErrIllustrationsStale) || gen.calls != 0 {
		t.Fatalf("err=%v codegen=%d, want ErrIllustrationsStale before any call", err, gen.calls)
	}
}

type eventLog struct {
	mu     sync.Mutex
	events []domain.ProjectEvent
}

func (e *eventLog) AppendProjectEvent(_ context.Context, ev domain.ProjectEvent) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.events = append(e.events, ev)
	return nil
}

func TestIllustrationsStepDrawsReportsProgressAndWaitsForReview(t *testing.T) {
	uc, _, _, _ := codeFixture(t, domain.RenderEngineRemotion, `{"scenes":[]}`)
	events := &eventLog{}
	st := &stubStage{
		pending: []domain.ProjectIllustration{{Name: "Motorbike", State: domain.PIDrawn}},
		reports: []application.StageReport{{Phase: "plan"}, {Phase: "draw", Total: 3, Done: 2, Failed: 1, Reused: 4, Planned: 7}},
	}
	var seen application.AuthoringProgress
	st.reports = append(st.reports, application.StageReport{Phase: "draw", Total: 3, Done: 3, Failed: 1, Reused: 4, Planned: 7})
	uc.WithPipeline(&stubFinalizer{}, &stubCodegen{}).WithIllustrations(&progressPeek{stubStage: st, uc: uc, seen: &seen}).WithEvents(events)

	out, err := uc.Execute(context.Background(), "p1", application.StepIllustrations)
	if err != nil {
		t.Fatal(err)
	}
	if !out.AwaitingReview || !strings.Contains(out.Message, "Motorbike") {
		t.Fatalf("want an awaiting-review result naming the drawing, got %+v", out)
	}
	if seen.Phase != "draw" || seen.DrawingsTotal != 3 || seen.DrawingsDone != 3 || seen.DrawingsFailed != 1 ||
		seen.DrawingsReused != 4 || seen.DrawingsPlanned != 7 || !seen.Running {
		t.Errorf("progress = %+v", seen)
	}
	if p := uc.Progress("p1", application.StepIllustrations); p.Running {
		t.Error("progress still running after the step")
	}
	if len(events.events) != 2 || events.events[1].Source != "authoring" || events.events[1].FlowStep != domain.FlowIllustrations ||
		events.events[1].RunState != domain.RunDone || !strings.Contains(events.events[1].Detail, "Motorbike") {
		t.Errorf("journal = %+v", events.events)
	}

	st.pending = nil
	if out, err = uc.Execute(context.Background(), "p1", application.StepIllustrations); err != nil || out.AwaitingReview {
		t.Fatalf("all reviewed: want a plain finish, got %+v, %v", out, err)
	}
}

// progressPeek reads the step's live progress as the stage reports it.
type progressPeek struct {
	*stubStage
	uc   *application.GenerateAuthoringUseCase
	seen *application.AuthoringProgress
}

func (p *progressPeek) Prepare(ctx context.Context, pid, model string, report func(application.StageReport)) ([]domain.ProjectIllustration, error) {
	return p.stubStage.Prepare(ctx, pid, model, func(r application.StageReport) {
		report(r)
		*p.seen = p.uc.Progress(pid, application.StepIllustrations)
	})
}

func TestIllustrationsStepIsForRemotionOnly(t *testing.T) {
	uc, _, _, _ := codeFixture(t, domain.RenderEngineManim, `{"scenes":[]}`)
	st := &stubStage{}
	uc.WithPipeline(&stubFinalizer{}, &stubCodegen{}).WithIllustrations(st)
	if _, err := uc.Execute(context.Background(), "p1", application.StepIllustrations); err == nil || st.prepared != 0 {
		t.Fatalf("a Manim video must not run the illustrations step: %v", err)
	}
}

type fixedModels domain.AuthoringStepModels

func (m fixedModels) GetAuthoringModels(context.Context, string) (domain.AuthoringStepModels, error) {
	return domain.AuthoringStepModels(m), nil
}

// A project that saved Ollama for code before the rule existed
// is stopped before any model call, with the reason.
func TestCodeStepRefusesASavedOllamaChoiceBeforeAnyCall(t *testing.T) {
	renderCtx := &fakeRenderContext{
		project: &domain.Project{ProjectID: "p1", ContentLanguage: domain.LanguageVietnamese, RenderEngine: domain.RenderEngineManim},
		topic:   "t", storyboard: `{"scenes":[]}`,
	}
	gen := &stubCodegen{result: application.CodeGenResult{Code: "x", CheckOK: true}}
	uc := application.NewGenerateAuthoringUseCase(
		newRenderer("SYSTEM {{topic}}", renderCtx), &stubProvider{content: "x"},
		application.NewLLMUsageRecorder(&usagePort{}, nil), renderCtx, fixedModels{Code: "ollama"},
		&recordingSaver{}, &contentSaver{}, &contentSaver{}, 0, 16000,
	).WithPipeline(&stubFinalizer{}, gen)
	_, err := uc.Execute(context.Background(), "p1", "code")
	var notForCode *application.ErrModelNotForCode
	if !errors.As(err, &notForCode) || gen.calls != 0 {
		t.Fatalf("err=%v codegen=%d, want ErrModelNotForCode before any call", err, gen.calls)
	}
}
