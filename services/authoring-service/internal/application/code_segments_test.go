package application_test

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"sync"
	"testing"

	"authoring/internal/application"
	"authoring/internal/domain"
)

// fakeSegments is the code segment store in memory, with the same rules as
// the Postgres one (ApplySegmentPlan resets a segment whose fingerprint changed;
// a repaired segment keeps its source; a failed one keeps its content by the
// rule of domain.FailedContent).
type fakeSegments struct {
	mu          sync.Mutex
	rows        map[string]domain.CodeSegment // key -> segment (one project, step code)
	chunkShots  int
	diagnostics []domain.CheckDiagnosticRecord
	deleted     int
}

func newFakeSegments() *fakeSegments {
	return &fakeSegments{rows: map[string]domain.CodeSegment{}, chunkShots: domain.DefaultChunkShots}
}

func (f *fakeSegments) ListSegments(context.Context, string, string) ([]domain.CodeSegment, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []domain.CodeSegment{}
	for _, s := range f.rows {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Position < out[j].Position })
	return out, nil
}

func (f *fakeSegments) ApplySegmentPlan(_ context.Context, _, _ string, plan []domain.CodeSegment) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	keep := map[string]bool{}
	for _, p := range plan {
		keep[p.Key] = true
		have, ok := f.rows[p.Key]
		if ok && have.Fingerprint == p.Fingerprint {
			have.Position, have.Kind, have.Shots = p.Position, p.Kind, p.Shots
			f.rows[p.Key] = have
			continue
		}
		f.rows[p.Key] = domain.CodeSegment{Key: p.Key, Kind: p.Kind, Position: p.Position, Shots: p.Shots,
			Fingerprint: p.Fingerprint, Status: domain.SegmentPending}
	}
	for k := range f.rows {
		if !keep[k] {
			delete(f.rows, k)
		}
	}
	return nil
}

func (f *fakeSegments) MarkSegmentRunning(_ context.Context, _, _, key string) error {
	return f.update(key, func(s *domain.CodeSegment) {
		s.Status, s.ErrorKind, s.ErrorMessage, s.FailedShots = domain.SegmentRunning, "", "", nil
	})
}

func (f *fakeSegments) SaveSegmentDone(_ context.Context, _, _ string, s domain.CodeSegment, repaired bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	have, ok := f.rows[s.Key]
	s.Status, s.ErrorKind, s.ErrorMessage, s.FailedShots = domain.SegmentDone, "", "", nil
	if ok && repaired {
		s.Source, s.DurationMS = have.Source, have.DurationMS
	}
	f.rows[s.Key] = s
	return nil
}

func (f *fakeSegments) SaveSegmentFailed(_ context.Context, _, _, key string, failure domain.SegmentFailure) error {
	return f.update(key, func(s *domain.CodeSegment) {
		s.Status, s.ErrorKind, s.ErrorMessage, s.DurationMS = domain.SegmentFailed, failure.Kind, failure.Message, failure.DurationMS
		s.FailedShots = failure.FailedShots
		s.Fingerprint, s.Content = domain.FailedContent(s.Fingerprint, s.Content, s.Shots, failure)
	})
}

func (f *fakeSegments) update(key string, change func(*domain.CodeSegment)) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.rows[key]
	if !ok {
		return errors.New("segment not found")
	}
	change(&s)
	f.rows[key] = s
	return nil
}

func (f *fakeSegments) FailRunningSegments(_ context.Context, _, _, kind, message string) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var n int64
	for k, s := range f.rows {
		if s.Status == domain.SegmentRunning {
			s.Status, s.ErrorKind, s.ErrorMessage = domain.SegmentFailed, kind, message
			f.rows[k] = s
			n++
		}
	}
	return n, nil
}

func (f *fakeSegments) FailAllRunningSegments(ctx context.Context, kind, message string) (int64, error) {
	return f.FailRunningSegments(ctx, "", "", kind, message)
}

func (f *fakeSegments) DeleteSegments(context.Context, string, string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rows = map[string]domain.CodeSegment{}
	f.deleted++
	return nil
}

func (f *fakeSegments) GetCodeChunkShots(context.Context, string) (int, error) {
	return f.chunkShots, nil
}

func (f *fakeSegments) SaveCodeChunkShots(_ context.Context, _ string, n int) error {
	f.chunkShots = n
	return nil
}

func (f *fakeSegments) InsertCheckDiagnostics(_ context.Context, list []domain.CheckDiagnosticRecord) error {
	f.diagnostics = append(f.diagnostics, list...)
	return nil
}

func (f *fakeSegments) get(key string) domain.CodeSegment {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.rows[key]
}

// fourShots is a storyboard of 4 shots in two scenes.
const fourShots = `{"scenes":[{"id":"a","shots":[{"id":"1.1"},{"id":"1.2"}]},{"id":"b","shots":[{"id":"2.1"},{"id":"2.2"}]}]}`

func segmentFixture(t *testing.T, engine domain.RenderEngine, storyboard string) (
	*application.GenerateAuthoringUseCase, *fakeSegments, *usagePort, *contentSaver,
) {
	t.Helper()
	renderCtx := &fakeRenderContext{
		project: &domain.Project{ProjectID: "p1", ContentLanguage: domain.LanguageVietnamese, RenderEngine: engine},
		topic:   "t", storyboard: storyboard,
	}
	code, usage, store := &contentSaver{}, &usagePort{}, newFakeSegments()
	uc := application.NewGenerateAuthoringUseCase(
		newRenderer("SYSTEM {{topic}}", renderCtx), &stubProvider{content: "x"},
		application.NewLLMUsageRecorder(usage, nil), renderCtx, nil,
		&recordingSaver{}, &contentSaver{}, code, 0, 16000,
	).WithSegments(store)
	return uc, store, usage, code
}

func planEvent(keys ...string) application.CodeEvent {
	ev := application.CodeEvent{Type: "plan"}
	for i, k := range keys {
		kind := domain.SegmentKindShots
		if k == domain.FrameSegmentKey {
			kind = domain.SegmentKindFrame
		}
		ev.Plan = append(ev.Plan, domain.CodeSegment{Key: k, Kind: kind, Position: i, Fingerprint: "fp-" + k, Shots: []string{}})
	}
	return ev
}

func doneEvent(key string) application.CodeEvent {
	return application.CodeEvent{Type: "segment_done", Key: key, Fingerprint: "fp-" + key,
		Content: json.RawMessage(`{"code":"` + key + `"}`), Source: domain.SegmentSourceAI, DurationMS: 10}
}

// Every segment is stored as it finishes; a failed one does
// not stop the step from keeping the rest, and the step says what is left.
func TestCodeStepStoresSegmentsAndEndsIncompleteWhenOneFailed(t *testing.T) {
	uc, store, usage, code := segmentFixture(t, domain.RenderEngineManim, fourShots)
	gen := &stubCodegen{
		events: []application.CodeEvent{
			planEvent("frame", "1.1-1.2", "2.1-2.2"),
			{Type: "segment_start", Key: "frame"}, doneEvent("frame"),
			{Type: "segment_start", Key: "1.1-1.2"}, {Type: "segment_start", Key: "2.1-2.2"},
			callEvent(application.CodeCall{Phase: "chunk", OK: false, ErrorKind: application.ErrKindTimeout, Segment: "1.1-1.2"}),
			{Type: "segment_failed", Key: "1.1-1.2", ErrorKind: "timeout", ErrorText: "slow"},
			doneEvent("2.1-2.2"),
		},
		result: application.CodeGenResult{Status: application.CodeGenIncomplete, Failed: []string{"1.1-1.2"}},
	}
	uc.WithPipeline(&stubFinalizer{}, gen)

	_, err := uc.Execute(context.Background(), "p1", "code")
	var inc *application.ErrSegmentsIncomplete
	if !errors.As(err, &inc) || len(inc.Failed) != 1 || inc.Total != 3 {
		t.Fatalf("err = %v, want ErrSegmentsIncomplete with one failed segment of 3", err)
	}
	if !strings.Contains(err.Error(), "1/3 đoạn lỗi (1.1-1.2: timeout)") || !strings.Contains(err.Error(), "đã được lưu") {
		t.Errorf("message = %q", err.Error())
	}
	if s := store.get("2.1-2.2"); s.Status != domain.SegmentDone || string(s.Content) != `{"code":"2.1-2.2"}` || s.Source != "ai" {
		t.Errorf("done segment = %+v", s)
	}
	if s := store.get("1.1-1.2"); s.Status != domain.SegmentFailed || s.ErrorMessage != "slow" {
		t.Errorf("failed segment = %+v", s)
	}
	if code.calls != 0 {
		t.Error("an incomplete step must not overwrite the saved code")
	}
	if len(usage.rows) != 1 || usage.rows[0].ErrorKind != application.ErrKindTimeout {
		t.Errorf("usage rows = %+v", usage.rows)
	}
}

// The next run is sent what is done — and "re-run this segment" leaves
// that one out and runs only it; "write everything again" drops them all first.
func TestCodeRunOptionsDecideWhatIsSentBack(t *testing.T) {
	uc, store, _, _ := segmentFixture(t, domain.RenderEngineManim, fourShots)
	gen := &stubCodegen{events: []application.CodeEvent{
		planEvent("frame", "1.1-1.2", "2.1-2.2"), doneEvent("frame"), doneEvent("1.1-1.2"),
		{Type: "segment_failed", Key: "2.1-2.2", ErrorKind: "server"},
	}, result: application.CodeGenResult{Status: application.CodeGenIncomplete}}
	uc.WithPipeline(&stubFinalizer{}, gen)
	_, _ = uc.Execute(context.Background(), "p1", "code")

	gen.events, gen.result = nil, application.CodeGenResult{Code: "x", CheckOK: true}
	if _, err := uc.Execute(context.Background(), "p1", "code"); err != nil {
		t.Fatal(err)
	}
	if keys := doneKeys(gen.req); keys != "1.1-1.2,frame" || gen.req.Only != nil {
		t.Errorf("missing run sent %q only %v, want the done segments and no filter", keys, gen.req.Only)
	}

	if _, err := uc.ExecuteCode(context.Background(), "p1", application.CodeRunOptions{Segment: "1.1-1.2"}); err != nil {
		t.Fatal(err)
	}
	if keys := doneKeys(gen.req); keys != "frame" || len(gen.req.Only) != 1 || gen.req.Only[0] != "1.1-1.2" {
		t.Errorf("segment run sent %q only %v", keys, gen.req.Only)
	}

	if _, err := uc.ExecuteCode(context.Background(), "p1", application.CodeRunOptions{Fresh: true}); err != nil {
		t.Fatal(err)
	}
	if store.deleted != 1 || len(gen.req.Done) != 0 {
		t.Errorf("fresh run: deleted %d, sent %d done segments", store.deleted, len(gen.req.Done))
	}
}

// A chunk that failed part-way keeps the shots it wrote; the next run sends
// them back so only the missing shot is written, and the step names the shot.
func TestAPartlyWrittenChunkIsStoredAndSentBack(t *testing.T) {
	uc, store, _, _ := segmentFixture(t, domain.RenderEngineRemotion, fourShots)
	plan := application.CodeEvent{Type: "plan", Plan: []domain.CodeSegment{
		{Key: "frame", Kind: domain.SegmentKindFrame, Position: 0, Fingerprint: "fp-frame", Shots: []string{}},
		{Key: "1.1-2.1", Kind: domain.SegmentKindShots, Position: 1, Fingerprint: "fp-a", Shots: []string{"1.1", "1.2", "2.1"}},
		{Key: "2.2-2.2", Kind: domain.SegmentKindShots, Position: 2, Fingerprint: "fp-b", Shots: []string{"2.2"}},
	}}
	written := json.RawMessage(`{"shots":{"1.1":"A","1.2":"B"}}`)
	gen := &stubCodegen{events: []application.CodeEvent{
		plan, doneEvent("frame"), doneEvent("2.2-2.2"),
		{Type: "segment_start", Key: "1.1-2.1"},
		{Type: "segment_failed", Key: "1.1-2.1", ErrorKind: "budget", Fingerprint: "fp-a", Content: written,
			FailedShots: []string{"2.1"}, ErrorText: "Shot 2.1: model suy nghĩ quá 60000 ký tự. Đã lưu shot 1.1, 1.2."},
	}, result: application.CodeGenResult{Status: application.CodeGenIncomplete, Failed: []string{"1.1-2.1"}}}
	uc.WithPipeline(&stubFinalizer{}, gen)

	_, err := uc.Execute(context.Background(), "p1", "code")
	if err == nil || !strings.Contains(err.Error(), "1/3 đoạn lỗi (shot 2.1: budget)") {
		t.Fatalf("the step error must name the failed shot, got %v", err)
	}
	s := store.get("1.1-2.1")
	if s.Status != domain.SegmentFailed || string(s.Content) != string(written) || s.Fingerprint != "fp-a" ||
		strings.Join(s.FailedShots, ",") != "2.1" {
		t.Errorf("failed chunk = %+v content=%s", s, s.Content)
	}

	gen.events, gen.result = nil, application.CodeGenResult{Code: "x", CheckOK: true}
	if _, err := uc.ExecuteCode(context.Background(), "p1", application.CodeRunOptions{Segment: "1.1-2.1"}); err != nil {
		t.Fatal(err)
	}
	if keys := doneKeys(gen.req); keys != "1.1-2.1,2.2-2.2,frame" {
		t.Errorf("re-running the chunk sent %q; its written shots must go back with it", keys)
	}
}

// A re-run of a complete chunk that fails part-way keeps the complete
// result, and that result is not sent back as if the chunk were half done.
func TestAFailedRerunKeepsACompleteChunk(t *testing.T) {
	uc, store, _, _ := segmentFixture(t, domain.RenderEngineRemotion, fourShots)
	shots := []string{"1.1", "1.2"}
	complete := json.RawMessage(`{"shots":{"1.1":"A","1.2":"B"}}`)
	plan := application.CodeEvent{Type: "plan", Plan: []domain.CodeSegment{
		{Key: "frame", Kind: domain.SegmentKindFrame, Position: 0, Fingerprint: "fp-frame", Shots: []string{}},
		{Key: "1.1-1.2", Kind: domain.SegmentKindShots, Position: 1, Fingerprint: "fp-a", Shots: shots},
	}}
	gen := &stubCodegen{events: []application.CodeEvent{plan, doneEvent("frame"),
		{Type: "segment_done", Key: "1.1-1.2", Fingerprint: "fp-a", Content: complete, Source: "ai"}},
		result: application.CodeGenResult{Code: "x", CheckOK: true}}
	uc.WithPipeline(&stubFinalizer{}, gen)
	if _, err := uc.Execute(context.Background(), "p1", "code"); err != nil {
		t.Fatal(err)
	}

	gen.events = []application.CodeEvent{plan, {Type: "segment_start", Key: "1.1-1.2"},
		{Type: "segment_failed", Key: "1.1-1.2", ErrorKind: "budget", Fingerprint: "fp-a",
			Content: json.RawMessage(`{"shots":{"1.1":"NEW"}}`), FailedShots: []string{"1.2"}}}
	gen.result = application.CodeGenResult{Status: application.CodeGenIncomplete, Failed: []string{"1.1-1.2"}}
	_, _ = uc.ExecuteCode(context.Background(), "p1", application.CodeRunOptions{Segment: "1.1-1.2"})
	if s := store.get("1.1-1.2"); string(s.Content) != string(complete) {
		t.Errorf("a failed re-run replaced the complete chunk with %s", s.Content)
	}

	gen.events, gen.result = nil, application.CodeGenResult{Code: "x", CheckOK: true}
	_, _ = uc.Execute(context.Background(), "p1", "code")
	if keys := doneKeys(gen.req); keys != "frame" {
		t.Errorf("a failed complete chunk sent back as %q; it is written again like any failed chunk", keys)
	}
}

func doneKeys(req application.CodeGenRequest) string {
	var keys []string
	for _, d := range req.Done {
		keys = append(keys, d.Key)
	}
	sort.Strings(keys)
	return strings.Join(keys, ",")
}

// NFR-3: a run cut off (llm-service restarted, the Creator cancelled) leaves
// its running segments failed with the reason; what finished stays.
func TestAnInterruptedRunFailsItsRunningSegments(t *testing.T) {
	uc, store, _, _ := segmentFixture(t, domain.RenderEngineManim, fourShots)
	uc.WithPipeline(&stubFinalizer{}, &stubCodegen{
		events: []application.CodeEvent{planEvent("frame", "1.1-1.2", "2.1-2.2"), doneEvent("frame"),
			{Type: "segment_start", Key: "1.1-1.2"}},
		err: &application.LLMError{Kind: application.ErrKindServer, Provider: "llm-service", Err: errors.New("stream closed")},
	})
	if _, err := uc.Execute(context.Background(), "p1", "code"); application.LLMErrorKindOf(err) != application.ErrKindServer {
		t.Fatalf("err = %v", err)
	}
	if s := store.get("1.1-1.2"); s.Status != domain.SegmentFailed || s.ErrorKind != domain.SegmentErrInterrupted ||
		!strings.Contains(s.ErrorMessage, "stream closed") {
		t.Errorf("running segment = %+v", s)
	}
	if store.get("frame").Status != domain.SegmentDone {
		t.Error("the finished frame was lost")
	}

	// the startup sweep does the same for a run the service itself lost
	_ = store.MarkSegmentRunning(context.Background(), "p1", "code", "2.1-2.2")
	if n, err := uc.FailInterruptedSegments(context.Background()); err != nil || n != 1 {
		t.Fatalf("sweep: %d %v", n, err)
	}
	if s := store.get("2.1-2.2"); s.Status != domain.SegmentFailed || s.ErrorKind != domain.SegmentErrInterrupted {
		t.Errorf("swept segment = %+v", s)
	}
}

// A repair overwrites its segment but keeps who wrote it; every
// failed check is logged with its rule, shot and segment.
func TestRepairsKeepTheSourceAndChecksAreLogged(t *testing.T) {
	uc, store, _, _ := segmentFixture(t, domain.RenderEngineRemotion, fourShots)
	paste := doneEvent("1.1-1.2")
	paste.Source = domain.SegmentSourceExternal
	repaired := doneEvent("1.1-1.2")
	repaired.Content, repaired.Source, repaired.Repaired = json.RawMessage(`{"code":"fixed"}`), "", true
	uc.WithPipeline(&stubFinalizer{}, &stubCodegen{events: []application.CodeEvent{
		planEvent("frame", "1.1-1.2", "2.1-2.2"), paste,
		{Type: "check", Check: &application.CodeCheck{Phase: "final", Round: 0, Diagnostics: []application.CodeDiagnostic{
			{Message: "ra ngoài vùng an toàn", Line: 40, Kind: "layout", Rule: "safe_area", Shot: "1.2", Segment: "1.1-1.2"}}}},
		repaired,
	}, result: application.CodeGenResult{Code: "x", CheckOK: true, RepairRounds: 1}})
	if _, err := uc.Execute(context.Background(), "p1", "code"); err != nil {
		t.Fatal(err)
	}
	if s := store.get("1.1-1.2"); string(s.Content) != `{"code":"fixed"}` || s.Source != domain.SegmentSourceExternal {
		t.Errorf("repaired segment = %+v", s)
	}
	if len(store.diagnostics) != 1 {
		t.Fatalf("diagnostics = %+v", store.diagnostics)
	}
	d := store.diagnostics[0]
	if d.ProjectID != "p1" || d.Engine != "remotion" || d.Phase != "final" || d.Rule != "safe_area" || d.ShotID != "1.2" || d.SegmentKey != "1.1-1.2" || d.Line != 40 {
		t.Errorf("diagnostic = %+v", d)
	}
}

// Before any run the panel already lists every segment of the storyboard,
// cut with the Creator's size; a storyboard-given LAYOUT needs no run.
func TestCodeSegmentsListsTheCutWithWhatIsStored(t *testing.T) {
	uc, store, _, _ := segmentFixture(t, domain.RenderEngineManim, fourShots)
	uc.WithPipeline(&stubFinalizer{}, &stubCodegen{})
	store.chunkShots = 3
	view, err := uc.CodeSegments(context.Background(), "p1")
	if err != nil {
		t.Fatal(err)
	}
	var keys []string
	for _, s := range view.Segments {
		keys = append(keys, s.Key+":"+string(s.Status))
	}
	if strings.Join(keys, " ") != "frame:pending 1.1-2.1:pending 2.2-2.2:pending" || view.ChunkShots != 3 || view.Running {
		t.Errorf("view = %v %+v", keys, view)
	}
	_ = store.SaveSegmentDone(context.Background(), "p1", "code", domain.CodeSegment{Key: "2.2-2.2", Position: 9, Source: "manual",
		Content: json.RawMessage(`{"shots":{}}`)}, false)
	_ = store.SaveSegmentDone(context.Background(), "p1", "code", domain.CodeSegment{Key: "1.1-1.2", Position: 1}, false) // an old cut
	view, _ = uc.CodeSegments(context.Background(), "p1")
	if len(view.Segments) != 3 || view.Segments[2].Status != domain.SegmentDone || view.Segments[2].Source != "manual" || view.Segments[2].Position != 2 {
		t.Errorf("stored segment not merged into the cut: %+v", view.Segments)
	}

	remotion, _, _, _ := segmentFixture(t, domain.RenderEngineRemotion,
		`{"layout":{"hero":{"x":1}},"scenes":[{"id":"a","shots":[{"id":"1.1"}]}]}`)
	remotion.WithPipeline(&stubFinalizer{}, &stubCodegen{})
	view, _ = remotion.CodeSegments(context.Background(), "p1")
	if f := view.Segments[0]; f.Status != domain.SegmentDone || f.Source != domain.SegmentSourceStoryboard {
		t.Errorf("storyboard layout frame = %+v", f)
	}

	prose, _, _, _ := segmentFixture(t, domain.RenderEngineManim, "CẢNH 1 — văn xuôi")
	prose.WithPipeline(&stubFinalizer{}, &stubCodegen{})
	var notCut *application.ErrStoryboardNotSegmentable
	if _, err := prose.CodeSegments(context.Background(), "p1"); !errors.As(err, &notCut) {
		t.Errorf("prose storyboard: %v", err)
	}

	// An llm-service without /v2 has no plan: said so, not a crash.
	old, _, _, _ := segmentFixture(t, domain.RenderEngineManim, fourShots)
	old.WithPipeline(&stubFinalizer{}, &stubCodegen{err: application.ErrSegmentsUnsupported})
	if _, err := old.CodeSegments(context.Background(), "p1"); !errors.Is(err, application.ErrSegmentsUnsupported) {
		t.Errorf("old llm-service: %v", err)
	}
}

// A pasted segment is checked by llm-service and stored with its source.
func TestPasteCodeSegment(t *testing.T) {
	uc, store, _, _ := segmentFixture(t, domain.RenderEngineManim, fourShots)
	store.chunkShots = 2
	gen := &stubCodegen{}
	uc.WithPipeline(&stubFinalizer{}, gen)
	ctx := context.Background()

	seg, err := uc.PasteCodeSegment(ctx, "p1", "2.1-2.2", "def shot_2_1(self): ...", domain.SegmentSourceExternal)
	if err != nil {
		t.Fatal(err)
	}
	stored := store.get("2.1-2.2")
	if seg.Fingerprint != "fp-2.1-2.2" || stored.Status != domain.SegmentDone || stored.Source != "external" ||
		stored.Kind != domain.SegmentKindShots || strings.Join(stored.Shots, ",") != "2.1,2.2" || !strings.Contains(string(stored.Content), "shot_2_1") {
		t.Errorf("stored = %+v", stored)
	}
	if !strings.HasPrefix(gen.req.System, "SYSTEM") || gen.req.ChunkShots != 2 {
		t.Errorf("the paste must be checked against the run's own request: %+v", gen.req)
	}
	if _, err := uc.PasteCodeSegment(ctx, "p1", "9.9-9.9", "x", domain.SegmentSourceManual); !errors.Is(err, application.ErrSegmentUnknown) {
		t.Errorf("unknown key: %v", err)
	}
	if _, err := uc.PasteCodeSegment(ctx, "p1", "frame", "x", "ai"); err == nil {
		t.Error("source ai must be refused: only a pasted or hand-edited reply goes through here")
	}
	var reply *application.ErrSegmentReply
	if _, err := uc.PasteCodeSegment(ctx, "p1", "frame", "  ", domain.SegmentSourceManual); !errors.As(err, &reply) {
		t.Errorf("empty paste: %v", err)
	}
	gen.err = &application.ErrSegmentReply{Message: "missing shot function(s): 2.2"}
	if _, err := uc.PasteCodeSegment(ctx, "p1", "2.1-2.2", "x", domain.SegmentSourceManual); !errors.As(err, &reply) {
		t.Errorf("a reply llm-service refuses is not stored: %v", err)
	}
	if store.get("2.1-2.2").Source != "external" {
		t.Error("a refused paste overwrote the stored segment")
	}

	gen.err = nil
	system, user, err := uc.CodeSegmentPrompt(ctx, "p1", "1.1-1.2")
	if err != nil || !strings.HasPrefix(system, "SYSTEM") || user != "USER 1.1-1.2" {
		t.Errorf("prompt: %q %q %v", system, user, err)
	}
	if len(gen.req.Done) != 1 || gen.req.Done[0].Key != "2.1-2.2" {
		t.Errorf("the prompt request must carry the stored segments (the frame a chunk needs): %+v", gen.req.Done)
	}
}

func TestSetCodeChunkShotsIsBounded(t *testing.T) {
	uc, store, _, _ := segmentFixture(t, domain.RenderEngineManim, fourShots)
	for _, n := range []int{0, domain.MaxChunkShots + 1} {
		if err := uc.SetCodeChunkShots(context.Background(), "p1", n); !errors.Is(err, domain.ErrInvalidChunkShots) {
			t.Errorf("%d: %v", n, err)
		}
	}
	if err := uc.SetCodeChunkShots(context.Background(), "p1", 5); err != nil || store.chunkShots != 5 {
		t.Errorf("5: %v, stored %d", err, store.chunkShots)
	}
}
