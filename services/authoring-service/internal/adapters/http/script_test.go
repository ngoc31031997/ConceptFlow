package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"authoring/internal/application"
	"authoring/internal/domain"
)

type fakeScript struct {
	err     error
	beat    string
	note    string
	kind    string
	lineID  string
	lineIDs []string
	runs    int
	replace application.ScriptReplaceRequest
}

func (f *fakeScript) State(context.Context, string) (application.ScriptState, error) {
	return application.ScriptState{Status: domain.UnitPending, Beats: []application.ScriptBeatView{{Key: "b1"}}}, f.err
}
func (f *fakeScript) Generate(_ context.Context, _, beat, note string) error {
	f.beat, f.note = beat, note
	f.runs++
	return f.err
}
func (f *fakeScript) Suggest(_ context.Context, _, kind, lineID string) (string, error) {
	f.kind, f.lineID = kind, lineID
	return "Câu thay.", f.err
}
func (f *fakeScript) Speak(_ context.Context, _ string, ids []string) error {
	f.lineIDs = ids
	return f.err
}

func (f *fakeScript) Replace(_ context.Context, _ string, req application.ScriptReplaceRequest) (application.ScriptReplaceResult, error) {
	f.replace = req
	return application.ScriptReplaceResult{Matches: []application.ScriptReplaceMatch{{Beat: "b1", Line: "b1.1", Before: "TRNG", After: "X"}}, Applied: req.Apply}, f.err
}

type fakeVoiceLines struct {
	lines   []domain.VoiceLine
	results []domain.VoiceLineResult
	reason  string
	err     error
}

func (f *fakeVoiceLines) ClaimPending(context.Context, string) ([]domain.VoiceLine, error) {
	return f.lines, f.err
}
func (f *fakeVoiceLines) SaveResults(_ context.Context, _ string, r []domain.VoiceLineResult) error {
	f.results = r
	return f.err
}
func (f *fakeVoiceLines) FailAll(_ context.Context, _, reason string) error {
	f.reason = reason
	return f.err
}

func serveScript(f *fakeScript, v *fakeVoiceLines, method, path, body string) *httptest.ResponseRecorder {
	rt := NewRouter(&fakeSuggestMetadata{}, nil).WithScript(f, v).WithOperations(application.NewOperations())
	rec := httptest.NewRecorder()
	rt.Handler().ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)))
	return rec
}

func TestScriptStateAndRunsPassTheirArgumentsOn(t *testing.T) {
	f := &fakeScript{}
	if rec := serveScript(f, nil, http.MethodGet, "/v1/projects/p1/script", ""); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"key":"b1"`) {
		t.Errorf("state: status %d body %s", rec.Code, rec.Body)
	}
	rec := serveScript(f, nil, http.MethodPost, "/v1/projects/p1/script/generate", `{"beat":"b2","note":"ngắn"}`)
	if rec.Code != http.StatusOK || f.beat != "b2" || f.note != "ngắn" || f.runs != 1 {
		t.Errorf("generate: status %d beat %q note %q", rec.Code, f.beat, f.note)
	}
	rec = serveScript(f, nil, http.MethodPost, "/v1/projects/p1/script/suggest", `{"kind":"join","line_id":"b2.1"}`)
	if rec.Code != http.StatusOK || f.kind != "join" || f.lineID != "b2.1" || !strings.Contains(rec.Body.String(), `"text":"Câu thay."`) {
		t.Errorf("suggest: status %d body %s", rec.Code, rec.Body)
	}
	rec = serveScript(f, nil, http.MethodPost, "/v1/projects/p1/script/speak", `{"line_ids":["b1.1","b1.2"]}`)
	if rec.Code != http.StatusOK || len(f.lineIDs) != 2 {
		t.Errorf("speak: status %d ids %v", rec.Code, f.lineIDs)
	}
}

func TestScriptErrorsKeepTheirMeaning(t *testing.T) {
	for name, c := range map[string]struct {
		err  error
		path string
		want int
	}{
		"beats not approved": {domain.ErrInvalidStatus, "/script/generate", http.StatusConflict},
		"beat has lines":     {application.ErrScriptBeatHasLines, "/script/generate", http.StatusBadRequest},
		"reply unusable":     {domain.ErrScriptReplyUnusable, "/script/generate", http.StatusBadGateway},
		"no model":           {application.ErrLLMNotConfigured, "/script/generate", http.StatusServiceUnavailable},
		"not flagged":        {application.ErrScriptLineNotFlagged, "/script/suggest", http.StatusBadRequest},
		"no voice":           {application.ErrVoiceNotConfigured, "/script/speak", http.StatusServiceUnavailable},
	} {
		rec := serveScript(&fakeScript{err: c.err}, nil, http.MethodPost, "/v1/projects/p1"+c.path, `{"kind":"line","line_id":"b1.1","line_ids":["b1.1"]}`)
		if rec.Code != c.want {
			t.Errorf("%s: status %d, want %d (%s)", name, rec.Code, c.want, rec.Body)
		}
	}
	if rec := serveScript(&fakeScript{err: errors.New("x")}, nil, http.MethodGet, "/v1/projects/p1/script", ""); rec.Code != http.StatusInternalServerError {
		t.Errorf("an unexpected error is a 500, got %d", rec.Code)
	}
}

func TestScriptRoutesAreOffUntilWired(t *testing.T) {
	rt := NewRouter(&fakeSuggestMetadata{}, nil)
	rec := httptest.NewRecorder()
	rt.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/projects/p1/script", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("status %d, want 404", rec.Code)
	}
}

func TestVoiceLinesAreClaimedAndAnsweredThroughTheInternalRoutes(t *testing.T) {
	v := &fakeVoiceLines{lines: []domain.VoiceLine{{LineID: "b1.1", Text: "Câu.", Language: "vi", VoiceID: "v", TextSHA: "abc"}}}
	rec := serveScript(&fakeScript{}, v, http.MethodPost, "/internal/v1/authoring/p1/voice-lines/claim", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"ref":"abc"`) || !strings.Contains(rec.Body.String(), `"line_id":"b1.1"`) {
		t.Errorf("claim: status %d body %s", rec.Code, rec.Body)
	}
	rec = serveScript(&fakeScript{}, v, http.MethodPost, "/internal/v1/authoring/p1/voice-lines/results",
		`{"lines":[{"line_id":"b1.1","ref":"abc","audio_path":"/shared/p1/audio/b1-1_k.wav","duration_seconds":3.2,"cache_key":"k","cache_hit":true}]}`)
	if rec.Code != http.StatusNoContent || len(v.results) != 1 || !v.results[0].CacheHit || v.results[0].DurationSeconds != 3.2 {
		t.Errorf("results: status %d %+v", rec.Code, v.results)
	}
	rec = serveScript(&fakeScript{}, v, http.MethodPost, "/internal/v1/authoring/p1/voice-lines/fail", `{"reason":"tts chết"}`)
	if rec.Code != http.StatusNoContent || v.reason != "tts chết" {
		t.Errorf("fail: status %d reason %q", rec.Code, v.reason)
	}
	if rec = serveScript(&fakeScript{}, v, http.MethodPost, "/internal/v1/authoring/p1/voice-lines/fail", `{}`); rec.Code != http.StatusBadRequest {
		t.Errorf("a failure without a reason is refused, got %d", rec.Code)
	}
}

func TestReplaceRouteHandsTheRequestToTheStepAndReturnsTheChanges(t *testing.T) {
	f := &fakeScript{}
	rt := NewRouter(&fakeSuggestMetadata{}, nil).WithScript(f, nil)
	req := httptest.NewRequest(http.MethodPost, "/v1/projects/p1/script/replace",
		strings.NewReader(`{"find":"TRNG","replace":"TI A EN GI","field":"spoken","match_case":true,"apply":true}`))
	rec := httptest.NewRecorder()
	rt.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"applied":true`) || !strings.Contains(rec.Body.String(), `"line":"b1.1"`) {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
	want := application.ScriptReplaceRequest{Find: "TRNG", Replace: "TI A EN GI", Field: "spoken", MatchCase: true, Apply: true}
	if f.replace != want {
		t.Errorf("request = %+v, want %+v", f.replace, want)
	}
}

func TestReplaceRouteRefusesABadRequestWith400(t *testing.T) {
	f := &fakeScript{err: domain.ErrInvalidUnitEdit}
	rt := NewRouter(&fakeSuggestMetadata{}, nil).WithScript(f, nil)
	req := httptest.NewRequest(http.MethodPost, "/v1/projects/p1/script/replace", strings.NewReader(`{"find":""}`))
	rec := httptest.NewRecorder()
	rt.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}
