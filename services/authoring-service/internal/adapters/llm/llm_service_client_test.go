package llm

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"authoring/internal/application"
	"authoring/internal/domain"
)

func ndjson(w http.ResponseWriter, lines ...string) {
	w.Header().Set("Content-Type", "application/x-ndjson")
	for _, l := range lines {
		_, _ = w.Write([]byte(l + "\n"))
	}
}

func serve(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return NewClient(srv.URL, 5*time.Second)
}

func TestChatStreamsProgressThenReturnsTheResult(t *testing.T) {
	var gotBody map[string]any
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		ndjson(w,
			`{"type":"progress","reasoning_chars":12,"content_chars":0}`,
			`{"type":"progress","reasoning_chars":12,"content_chars":40}`,
			`{"type":"result","content":"xin chào","provider":"hive","usage":{"model":"deepseek","prompt_tokens":10,"completion_tokens":5,"reasoning_tokens":3,"cached_tokens":2}}`)
	})

	var seen []application.ChatProgress
	res, err := c.Chat(context.Background(), application.ChatRequest{
		System: "SYS", User: "USR", Model: "m", MaxTokens: 99, Temperature: 0.5,
		OnProgress: func(p application.ChatProgress) { seen = append(seen, p) },
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if res.Content != "xin chào" || res.Usage.ReasoningTokens != 3 || res.Usage.CachedTokens != 2 || res.Usage.Model != "deepseek" {
		t.Errorf("result = %+v", res)
	}
	if len(seen) != 2 || seen[1].ContentChars != 40 {
		t.Errorf("progress = %+v", seen)
	}
	if gotBody["system"] != "SYS" || gotBody["user"] != "USR" || gotBody["model"] != "m" || gotBody["max_tokens"] != float64(99) {
		t.Errorf("request body = %v", gotBody)
	}
}

func TestChatErrorKeepsKindBilledUsagePartialAndDiag(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, _ *http.Request) {
		ndjson(w, `{"type":"error","error":{"kind":"truncated","provider":"hive","message":"cut off","partial":"half","diag":"finish_reason=length","usage":{"model":"m","prompt_tokens":8,"completion_tokens":50}},"calls":[]}`)
	})
	_, err := c.Chat(context.Background(), application.ChatRequest{User: "u"})
	var llmErr *application.LLMError
	if !asLLMError(err, &llmErr) {
		t.Fatalf("err = %v, want an LLMError", err)
	}
	if llmErr.Kind != application.ErrKindTruncated || llmErr.Partial != "half" || llmErr.Usage.CompletionTokens != 50 || llmErr.Diag != "finish_reason=length" {
		t.Errorf("llm error = %+v", llmErr)
	}
}

func TestAStreamThatEndsWithoutAResultIsAnErrorNotAnEmptyAnswer(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, _ *http.Request) {
		ndjson(w, `{"type":"progress","reasoning_chars":1,"content_chars":0}`)
	})
	_, err := c.Chat(context.Background(), application.ChatRequest{User: "u"})
	if application.LLMErrorKindOf(err) != application.ErrKindServer || !strings.Contains(err.Error(), "without a result") {
		t.Fatalf("err = %v", err)
	}
}

func TestAnUnreachableServiceIsAServerErrorAndATimeoutIsATimeout(t *testing.T) {
	dead := NewClient("http://127.0.0.1:1", time.Second)
	if _, err := dead.Chat(context.Background(), application.ChatRequest{User: "u"}); application.LLMErrorKindOf(err) != application.ErrKindServer {
		t.Fatalf("unreachable: err = %v", err)
	}

	// The server only notices the client hanging up once the request body has
	// been read, so drain it before waiting.
	slow := serve(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		<-r.Context().Done()
	})
	slow.timeout = 50 * time.Millisecond
	if _, err := slow.Chat(context.Background(), application.ChatRequest{User: "u"}); application.LLMErrorKindOf(err) != application.ErrKindTimeout {
		t.Fatalf("slow: err = %v, want timeout", err)
	}
}

func TestSuggestMetadataAndShortScript(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		switch r.URL.Path {
		case "/v1/suggest-metadata":
			if body["language"] != "vi" || body["script_content"] != "S" || body["category_hint"] != "C" {
				t.Errorf("metadata body = %v", body)
			}
			_, _ = w.Write([]byte(`{"title":"T","description":"D","tags":null,"usage":{}}`))
		case "/v1/suggest-short-script":
			_, _ = w.Write([]byte(`{"script":"from conceptflow import *","usage":{}}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	})
	title, desc, tags, err := c.Suggest(context.Background(), "S", "C", domain.LanguageVietnamese)
	if err != nil || title != "T" || desc != "D" {
		t.Fatalf("Suggest: %q %q %v", title, desc, err)
	}
	if tags == nil || len(tags) != 0 {
		t.Errorf("tags = %#v, want an empty non-nil slice (the API contract says array, never null)", tags)
	}
	script, err := c.SuggestShortScript(context.Background(), "topic", "", domain.LanguageEnglish)
	if err != nil || script != "from conceptflow import *" {
		t.Fatalf("SuggestShortScript: %q %v", script, err)
	}
}

func TestANon200AnswerCarriesTheServicesErrorKind(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":{"kind":"balance","provider":"ollama","message":"no credit","usage":{}}}`))
	})
	_, _, _, err := c.Suggest(context.Background(), "S", "C", domain.LanguageVietnamese)
	if application.LLMErrorKindOf(err) != application.ErrKindBalance {
		t.Fatalf("err = %v, want balance", err)
	}

	bad := serve(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`<html>boom</html>`))
	})
	if _, err := bad.SuggestShortScript(context.Background(), "t", "", domain.LanguageEnglish); application.LLMErrorKindOf(err) != application.ErrKindServer {
		t.Fatalf("err = %v, want server", err)
	}
}

func TestFinalizeStoryboard(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["content"] != "RAW" || body["model"] != "m" {
			t.Errorf("body = %v", body)
		}
		_, _ = w.Write([]byte(`{"storyboard":"{}","shots":12,"repaired":true,"usage":{"prompt_tokens":7,"completion_tokens":9}}`))
	})
	got, err := c.FinalizeStoryboard(context.Background(), "RAW", "m", 1000)
	if err != nil || got.Storyboard != "{}" || got.Shots != 12 || !got.Repaired || got.Usage.PromptTokens != 7 {
		t.Fatalf("got %+v err %v", got, err)
	}

	bad := serve(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"error":{"kind":"malformed","message":"still invalid","usage":{"prompt_tokens":4}}}`))
	})
	_, err = bad.FinalizeStoryboard(context.Background(), "RAW", "", 0)
	var llmErr *application.LLMError
	if !asLLMError(err, &llmErr) || llmErr.Kind != application.ErrKindMalformed || llmErr.Usage.PromptTokens != 4 {
		t.Fatalf("err = %v", err)
	}
}

func TestGenerateCodeStreamsEventsAndReturnsTheResult(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/code/generate" {
			t.Errorf("path = %s", r.URL.Path)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["engine"] != "remotion" || body["storyboard"] != "SB" || body["system"] != "SYS" || body["chunk_shots"] != float64(2) {
			t.Errorf("body = %v", body)
		}
		segs, _ := body["segments"].([]any)
		if len(segs) != 1 || segs[0].(map[string]any)["key"] != "frame" || body["only"].([]any)[0] != "1.3-1.4" {
			t.Errorf("segments/only = %v / %v", body["segments"], body["only"])
		}
		ndjson(w,
			`{"type":"plan","segments":[{"key":"frame","kind":"frame","shots":[],"fingerprint":"f0"},{"key":"1.3-1.4","kind":"shots","shots":["1.3","1.4"],"fingerprint":"f2"}]}`,
			`{"type":"phase","phase":"chunks","total":1}`,
			`{"type":"segment_start","key":"1.3-1.4"}`,
			`{"type":"call","phase":"chunk","label":"1.3-1.4","segment":"1.3-1.4","ok":true,"duration_ms":1500,"usage":{"model":"m","prompt_tokens":5,"completion_tokens":6}}`,
			`{"type":"check","phase":"chunk","round":0,"segment":"1.3-1.4","diagnostics":[{"message":"boom","line":9,"kind":"layout","rule":"safe_area","shot":"1.3","segment":"1.3-1.4"}]}`,
			`{"type":"segment_done","key":"1.3-1.4","fingerprint":"f2","content":{"shots":{"1.3":"a","1.4":"b"}},"source":"ai","repaired":false,"duration_ms":2000}`,
			`{"type":"segment_failed","key":"1.5-1.5","error":{"kind":"timeout","message":"slow"}}`,
			`{"type":"chunk_done","index":1,"total":1,"done":1}`,
			`{"type":"segment_failed","key":"1.6-1.8","fingerprint":"f3","content":{"shots":{"1.6":"a"}},"failed_shots":["1.7","1.8"],"error":{"kind":"budget","message":"Shot 1.7–1.8"}}`,
			`{"type":"segment_failed","key":"1.9-1.9","fingerprint":"f4","content":null,"failed_shots":["1.9"],"error":{"kind":"server","message":"Shot 1.9"}}`,
			`{"type":"result","status":"done","code":"CODE","check_ok":false,"repair_rounds":3,"scene_class_name":"XScene","warnings":["w"],`+
				`"diagnostics":[{"message":"boom","line":9,"rule":"TS1"},{"message":"no line","line":null}],"failed":[],"missing":[]}`)
	})
	var events []application.CodeEvent
	res, err := c.GenerateCode(context.Background(),
		application.CodeGenRequest{Engine: "remotion", Storyboard: "SB", System: "SYS", ChunkShots: 2, Only: []string{"1.3-1.4"},
			Done: []application.DoneSegment{{Key: "frame", Fingerprint: "f0", Content: json.RawMessage(`{"code":"L"}`)}}},
		func(e application.CodeEvent) { events = append(events, e) })
	if err != nil {
		t.Fatalf("GenerateCode: %v", err)
	}
	if len(events) != 10 {
		t.Fatalf("events = %+v", events)
	}
	if p := events[0].Plan; len(p) != 2 || p[1].Key != "1.3-1.4" || p[1].Position != 1 || p[1].Fingerprint != "f2" || len(p[1].Shots) != 2 {
		t.Errorf("plan = %+v", p)
	}
	if call := events[3].Call; call == nil || call.Segment != "1.3-1.4" || call.Duration != 1500*time.Millisecond || call.Usage.PromptTokens != 5 {
		t.Errorf("call = %+v", events[3])
	}
	if chk := events[4].Check; chk == nil || chk.Segment != "1.3-1.4" || len(chk.Diagnostics) != 1 ||
		chk.Diagnostics[0].Rule != "safe_area" || chk.Diagnostics[0].Shot != "1.3" || chk.Diagnostics[0].Line != 9 {
		t.Errorf("check = %+v", events[4])
	}
	if d := events[5]; d.Key != "1.3-1.4" || d.Fingerprint != "f2" || d.Source != "ai" || d.DurationMS != 2000 ||
		string(d.Content) != `{"shots":{"1.3":"a","1.4":"b"}}` {
		t.Errorf("segment_done = %+v", d)
	}
	if f := events[6]; f.Key != "1.5-1.5" || f.ErrorKind != "timeout" || f.ErrorText != "slow" || len(f.FailedShots) != 0 {
		t.Errorf("segment_failed = %+v", f)
	}
	if f := events[8]; f.Key != "1.6-1.8" || f.Fingerprint != "f3" || string(f.Content) != `{"shots":{"1.6":"a"}}` ||
		strings.Join(f.FailedShots, ",") != "1.7,1.8" || f.ErrorKind != "budget" {
		t.Errorf("partly written segment_failed = %+v content=%s", f, f.Content)
	}
	if f := events[9]; f.Content != nil || strings.Join(f.FailedShots, ",") != "1.9" {
		t.Errorf("segment_failed without written shots must carry no content: %+v content=%s", f, f.Content)
	}
	if res.Status != application.CodeGenDone || res.Code != "CODE" || res.CheckOK || res.RepairRounds != 3 || res.SceneClassName != "XScene" {
		t.Errorf("result = %+v", res)
	}
	if len(res.Diagnostics) != 2 || res.Diagnostics[0].Line != 9 || res.Diagnostics[0].Rule != "TS1" || res.Diagnostics[1].Line != 0 {
		t.Errorf("diagnostics = %+v", res.Diagnostics)
	}
}

func TestGenerateCodeRefusesAnUnknownStatusOrAnUnreadableEvent(t *testing.T) {
	bad := serve(t, func(w http.ResponseWriter, _ *http.Request) {
		ndjson(w, `{"type":"result","status":"weird"}`)
	})
	if _, err := bad.GenerateCode(context.Background(), application.CodeGenRequest{Engine: "remotion"}, nil); application.LLMErrorKindOf(err) != application.ErrKindMalformed {
		t.Errorf("unknown status: err = %v", err)
	}
	broken := serve(t, func(w http.ResponseWriter, _ *http.Request) {
		ndjson(w, `{"type":"call","usage":"not an object"}`, `{"type":"result","status":"done","code":"C"}`)
	})
	if _, err := broken.GenerateCode(context.Background(), application.CodeGenRequest{Engine: "remotion"}, nil); application.LLMErrorKindOf(err) != application.ErrKindMalformed {
		t.Errorf("an unreadable call event must fail the run, not lose its cost: err = %v", err)
	}
}

func TestSegmentCallsMapLLMServiceAnswers(t *testing.T) {
	status := http.StatusOK
	var path string
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["key"] != "1.1-1.2" {
			t.Errorf("body = %v", body)
		}
		w.WriteHeader(status)
		switch {
		case status != http.StatusOK:
			_, _ = w.Write([]byte(`{"error":{"kind":"malformed","message":"missing shot function(s): 1.2"}}`))
		case r.URL.Path == "/v2/code/segment-prompt":
			_, _ = w.Write([]byte(`{"system":"S","user":"U"}`))
		default:
			_, _ = w.Write([]byte(`{"fingerprint":"fp","content":{"shots":{"1.1":"a"}}}`))
		}
	})
	req := application.CodeGenRequest{Engine: "remotion"}
	if sys, user, err := c.SegmentPrompt(context.Background(), req, "1.1-1.2"); err != nil || sys != "S" || user != "U" || path != "/v2/code/segment-prompt" {
		t.Errorf("prompt: %q %q %v %s", sys, user, err, path)
	}
	if fp, content, err := c.ParseSegment(context.Background(), req, "1.1-1.2", "reply"); err != nil || fp != "fp" || string(content) != `{"shots":{"1.1":"a"}}` {
		t.Errorf("parse: %q %s %v", fp, content, err)
	}
	status = http.StatusUnprocessableEntity
	var reply *application.ErrSegmentReply
	if _, _, err := c.ParseSegment(context.Background(), req, "1.1-1.2", "reply"); !errors.As(err, &reply) || !strings.Contains(reply.Message, "1.2") {
		t.Errorf("422: %v", err)
	}
	status = http.StatusConflict
	var notReady *application.ErrSegmentNotReady
	if _, _, err := c.SegmentPrompt(context.Background(), req, "1.1-1.2"); !errors.As(err, &notReady) {
		t.Errorf("409: %v", err)
	}
	status = http.StatusNotFound
	if _, _, err := c.SegmentPrompt(context.Background(), req, "1.1-1.2"); !errors.Is(err, application.ErrSegmentUnknown) {
		t.Errorf("404: %v", err)
	}
}

func TestGenerateCodeSendsTheLayoutContextOnlyWhenThereIsOne(t *testing.T) {
	// The rendering layout check needs the subtitle strip and the font.
	var bodies []map[string]any
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		bodies = append(bodies, body)
		ndjson(w, `{"type":"result","status":"done","code":"CODE","check_ok":true}`)
	})
	band := domain.SubtitleBand{Edge: "top", Px: 280}
	if _, err := c.GenerateCode(context.Background(), application.CodeGenRequest{
		Engine: "remotion", SubtitleBand: &band, VideoFont: "Montserrat"}, nil); err != nil {
		t.Fatalf("GenerateCode: %v", err)
	}
	if _, err := c.GenerateCode(context.Background(), application.CodeGenRequest{Engine: "remotion"}, nil); err != nil {
		t.Fatalf("GenerateCode: %v", err)
	}
	sb, ok := bodies[0]["subtitle_band"].(map[string]any)
	if !ok || sb["edge"] != "top" || sb["px"] != float64(280) || bodies[0]["video_font"] != "Montserrat" {
		t.Errorf("first body = %v", bodies[0])
	}
	if _, has := bodies[1]["subtitle_band"]; has {
		t.Errorf("no band must mean no subtitle_band field, got %v", bodies[1])
	}
	if _, has := bodies[1]["video_font"]; has {
		t.Errorf("no font must mean no video_font field, got %v", bodies[1])
	}
}

// The /v1 fallback is gone — an llm-service without /v2 fails the
// step with ErrSegmentsUnsupported instead of running it another way.
func TestGenerateCodeOnAnOlderLLMServiceIsUnsupported(t *testing.T) {
	var paths []string
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"detail":"Not Found"}`))
	})
	_, err := c.GenerateCode(context.Background(), application.CodeGenRequest{Engine: "remotion"}, nil)
	if !errors.Is(err, application.ErrSegmentsUnsupported) {
		t.Fatalf("err = %v", err)
	}
	if strings.Join(paths, ",") != "/v2/code/generate" {
		t.Errorf("paths = %v", paths)
	}
}

// A call whose stream was cut carries the reasoning it counted and
// says its usage was not reported; an older llm-service (no field) is reported.
func TestCodeCallUsageCarriesReasoningCharsAndTheReportedFlag(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		ndjson(w,
			`{"type":"call","phase":"chunk","label":"3.3-3.3","ok":false,"error_kind":"budget","segment":"3.3-3.5",`+
				`"usage":{"model":"m","reasoning_chars":61000,"usage_reported":false}}`,
			`{"type":"call","phase":"chunk","label":"1.1-1.3","ok":true,"segment":"1.1-1.3",`+
				`"usage":{"model":"m","prompt_tokens":5}}`,
			`{"type":"result","status":"incomplete","failed":["3.3-3.5"]}`)
	})
	var calls []application.CodeCall
	on := func(e application.CodeEvent) {
		if e.Call != nil {
			calls = append(calls, *e.Call)
		}
	}
	if _, err := c.GenerateCode(context.Background(), application.CodeGenRequest{Engine: "remotion"}, on); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 {
		t.Fatalf("calls = %+v", calls)
	}
	if u := calls[0].Usage; u.ReasoningChars != 61000 || !u.UsageMissing {
		t.Errorf("cut call usage = %+v", u)
	}
	if u := calls[1].Usage; u.UsageMissing || u.PromptTokens != 5 {
		t.Errorf("reported call usage = %+v", u)
	}
}

func TestPlanSegmentsAndAnOlderLLMService(t *testing.T) {
	missing := false
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		if missing {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"detail":"Not Found"}`))
			return
		}
		if r.URL.Path != "/v2/code/plan" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"segments":[{"key":"frame","kind":"frame","shots":[],"fingerprint":"f","source":"storyboard"},` +
			`{"key":"1.1-1.3","kind":"shots","shots":["1.1","1.2","1.3"],"fingerprint":"g"}]}`))
	})
	plan, err := c.PlanSegments(context.Background(), application.CodeGenRequest{Engine: "remotion", ChunkShots: 3})
	if err != nil || len(plan) != 2 || plan[0].Source != "storyboard" || plan[1].Position != 1 || len(plan[1].Shots) != 3 {
		t.Fatalf("plan = %+v err = %v", plan, err)
	}
	missing = true
	if _, err := c.PlanSegments(context.Background(), application.CodeGenRequest{}); !errors.Is(err, application.ErrSegmentsUnsupported) {
		t.Errorf("no route: %v", err)
	}
	if _, _, err := c.SegmentPrompt(context.Background(), application.CodeGenRequest{}, "frame"); !errors.Is(err, application.ErrSegmentsUnsupported) {
		t.Errorf("no route must not read as an unknown segment: %v", err)
	}
}

func TestGenerateCodeFailureIsTheProviderError(t *testing.T) {
	// ADR-0030: the billed calls were streamed before the error; it carries none.
	c := serve(t, func(w http.ResponseWriter, _ *http.Request) {
		ndjson(w, `{"type":"call","phase":"layout","label":"LAYOUT","ok":true,"usage":{"model":"m","prompt_tokens":100}}`,
			`{"type":"error","error":{"kind":"balance","provider":"hive","message":"out of credit"},"calls":[]}`)
	})
	var calls []application.CodeCall
	_, err := c.GenerateCode(context.Background(), application.CodeGenRequest{Engine: "manim"}, func(e application.CodeEvent) {
		if e.Call != nil {
			calls = append(calls, *e.Call)
		}
	})
	var llmErr *application.LLMError
	if !asLLMError(err, &llmErr) || llmErr.Kind != application.ErrKindBalance {
		t.Fatalf("err = %v", err)
	}
	if len(calls) != 1 || calls[0].Usage.PromptTokens != 100 {
		t.Errorf("calls = %+v", calls)
	}
}

func TestReadyReflectsTheServicesHiveKeyAndIsCached(t *testing.T) {
	hits := 0
	configured := false
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		hits++
		_, _ = w.Write([]byte(`{"status":"ok","hive_configured":` + map[bool]string{true: "true", false: "false"}[configured] + `}`))
	})
	if c.Ready() {
		t.Fatal("Ready with no Hive key, want false")
	}
	configured = true
	if c.Ready() { // still inside the cache window
		t.Fatal("Ready flipped inside the cache window")
	}
	c.readyAt = time.Time{}
	if !c.Ready() {
		t.Fatal("Ready after the key appeared and the cache expired, want true")
	}
	if hits != 2 {
		t.Errorf("health hits = %d, want 2", hits)
	}
	if NewClient("http://127.0.0.1:1", time.Second).Ready() {
		t.Error("Ready with llm-service down, want false")
	}
}

func asLLMError(err error, target **application.LLMError) bool { return errors.As(err, target) }
