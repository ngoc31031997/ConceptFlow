package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"authoring/internal/application"
	"authoring/internal/domain"
)

// fakeCodeSegments is the generate use case with the CR-050 segment side.
type fakeCodeSegments struct {
	fakeGenerateAuthoring
	opts     application.CodeRunOptions
	execCode int
	view     application.CodeSegmentsView
	err      error
	pasted   [3]string
	shots    int
}

func (f *fakeCodeSegments) ExecuteCode(_ context.Context, _ string, opts application.CodeRunOptions) (application.GeneratedStep, error) {
	f.execCode++
	f.opts = opts
	return f.out, f.fakeGenerateAuthoring.err
}
func (f *fakeCodeSegments) CodeSegments(context.Context, string) (application.CodeSegmentsView, error) {
	return f.view, f.err
}
func (f *fakeCodeSegments) CodeSegmentPrompt(_ context.Context, _, key string) (string, string, error) {
	return "SYS", "USER " + key, f.err
}
func (f *fakeCodeSegments) PasteCodeSegment(_ context.Context, _, key, reply, source string) (domain.CodeSegment, error) {
	f.pasted = [3]string{key, reply, source}
	return domain.CodeSegment{Key: key, Status: domain.SegmentDone, Source: source}, f.err
}
func (f *fakeCodeSegments) SetCodeChunkShots(_ context.Context, _ string, n int) error {
	f.shots = n
	return f.err
}

func serveSegments(f *fakeCodeSegments, method, path, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, bytes.NewReader([]byte(body)))
	newGenerateRouter(f).Handler().ServeHTTP(rec, req)
	return rec
}

func TestGenerateCodeTakesRunOptions(t *testing.T) {
	f := &fakeCodeSegments{fakeGenerateAuthoring: fakeGenerateAuthoring{available: true}}
	if rec := serveSegments(f, "POST", "/v1/projects/p1/authoring/code/generate", ""); rec.Code != 200 || f.execCode != 0 || f.gotStep != "code" {
		t.Fatalf("no body must run the missing segments through Execute: %d exec=%d", rec.Code, f.execCode)
	}
	if rec := serveSegments(f, "POST", "/v1/projects/p1/authoring/code/generate", `{"segment":"1.1-1.3"}`); rec.Code != 200 || f.opts.Segment != "1.1-1.3" {
		t.Fatalf("segment: %d %+v", rec.Code, f.opts)
	}
	if rec := serveSegments(f, "POST", "/v1/projects/p1/authoring/code/generate", `{"fresh":true}`); rec.Code != 200 || !f.opts.Fresh {
		t.Fatalf("fresh: %d %+v", rec.Code, f.opts)
	}
	if rec := serveSegments(f, "POST", "/v1/projects/p1/authoring/code/generate", `{"fresh":true,"segment":"x"}`); rec.Code != 400 {
		t.Errorf("both options: %d", rec.Code)
	}
	if rec := serveSegments(f, "POST", "/v1/projects/p1/authoring/story/generate", `{"fresh":true}`); rec.Code != 400 {
		t.Errorf("options on another step: %d", rec.Code)
	}
	if rec := serveSegments(f, "POST", "/v1/projects/p1/authoring/code/generate", `{`); rec.Code != 400 {
		t.Errorf("broken body: %d", rec.Code)
	}
}

func TestAnIncompleteCodeStepIs422WithWhatIsLeft(t *testing.T) {
	f := &fakeCodeSegments{fakeGenerateAuthoring: fakeGenerateAuthoring{available: true,
		err: &application.ErrSegmentsIncomplete{Total: 5, Failed: []domain.CodeSegment{{Key: "1.4-1.6", ErrorKind: "timeout"}}}}}
	rec := serveSegments(f, "POST", "/v1/projects/p1/authoring/code/generate", "")
	if rec.Code != 422 || !strings.Contains(rec.Body.String(), "1/5 đoạn lỗi (1.4-1.6: timeout)") {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
}

func TestCodeSegmentRoutes(t *testing.T) {
	f := &fakeCodeSegments{view: application.CodeSegmentsView{ChunkShots: 3, Segments: []domain.CodeSegment{{Key: "frame", Status: domain.SegmentDone}}}}
	rec := serveSegments(f, "GET", "/v1/projects/p1/authoring/code/segments", "")
	var view application.CodeSegmentsView
	if rec.Code != 200 || json.Unmarshal(rec.Body.Bytes(), &view) != nil || view.ChunkShots != 3 || view.Segments[0].Key != "frame" {
		t.Fatalf("list: %d %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "fingerprint") {
		t.Error("the fingerprint is llm-service's business, not the GUI's")
	}

	rec = serveSegments(f, "GET", "/v1/projects/p1/authoring/code/segments/1.1-1.3/prompt", "")
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"user":"USER 1.1-1.3"`) {
		t.Fatalf("prompt: %d %s", rec.Code, rec.Body.String())
	}

	rec = serveSegments(f, "PUT", "/v1/projects/p1/authoring/code/segments/1.1-1.3", `{"reply":"R","source":"external"}`)
	if rec.Code != 200 || f.pasted != [3]string{"1.1-1.3", "R", "external"} {
		t.Fatalf("paste: %d %v", rec.Code, f.pasted)
	}
	if rec = serveSegments(f, "PUT", "/v1/projects/p1/authoring/code/segments/1.1-1.3", `{"reply":"R","source":"ai"}`); rec.Code != 400 {
		t.Errorf("paste as ai: %d", rec.Code)
	}

	if rec = serveSegments(f, "PUT", "/v1/projects/p1/authoring/code/chunk-shots", `{"chunk_shots":4}`); rec.Code != 204 || f.shots != 4 {
		t.Errorf("chunk shots: %d %d", rec.Code, f.shots)
	}

	for _, c := range []struct {
		err  error
		code int
	}{
		{application.ErrSegmentUnknown, 404},
		{&application.ErrSegmentNotReady{Message: "no frame"}, 409},
		{&application.ErrSegmentReply{Message: "missing shot"}, 422},
		{&application.ErrStoryboardNotSegmentable{Cause: errors.New("prose")}, 409},
		{domain.ErrInvalidChunkShots, 400},
		{application.ErrGenerateBusy, 409},
		{application.ErrSegmentsUnsupported, 501},
	} {
		f.err = c.err
		if rec = serveSegments(f, "PUT", "/v1/projects/p1/authoring/code/segments/k", `{"reply":"R","source":"manual"}`); rec.Code != c.code {
			t.Errorf("%v: %d, want %d", c.err, rec.Code, c.code)
		}
	}
}

func TestCodeSegmentRoutesNeedTheSegmentUseCase(t *testing.T) {
	rec := httptest.NewRecorder()
	newGenerateRouter(&fakeGenerateAuthoring{available: true}).Handler().ServeHTTP(rec,
		httptest.NewRequest("GET", "/v1/projects/p1/authoring/code/segments", nil))
	if rec.Code != 404 {
		t.Errorf("%d", rec.Code)
	}
}

type fakeChain struct {
	steps []string
	opts  application.CodeRunOptions
	err   error
}

func (f *fakeChain) StartWith(_ string, steps []string, opts application.CodeRunOptions) error {
	f.steps, f.opts = steps, opts
	return f.err
}
func (f *fakeChain) State(string) (application.ChainState, bool) {
	return application.ChainState{}, false
}
func (f *fakeChain) Cancel(string) bool { return false }

// CR-050: the panel's segment re-run and fresh run start the server chain.
func TestChainStartCarriesCodeRunOptions(t *testing.T) {
	ch := &fakeChain{}
	rt := newGenerateRouter(nil).WithAuthoringChain(ch)
	rec := httptest.NewRecorder()
	rt.Handler().ServeHTTP(rec, httptest.NewRequest("POST", "/v1/projects/p1/authoring/chain",
		strings.NewReader(`{"steps":["code"],"segment":"1.1-1.3"}`)))
	if rec.Code != 202 || ch.opts.Segment != "1.1-1.3" || len(ch.steps) != 1 {
		t.Fatalf("%d %+v %v", rec.Code, ch.opts, ch.steps)
	}
	ch.err = application.ErrChainInvalid
	rec = httptest.NewRecorder()
	rt.Handler().ServeHTTP(rec, httptest.NewRequest("POST", "/v1/projects/p1/authoring/chain",
		strings.NewReader(`{"steps":["story"],"fresh":true}`)))
	if rec.Code != 400 || !ch.opts.Fresh {
		t.Errorf("%d %+v", rec.Code, ch.opts)
	}
}
