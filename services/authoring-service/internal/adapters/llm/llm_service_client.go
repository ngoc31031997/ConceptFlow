// Package llm is the orchestrator's only path to a language model: an HTTP
// client for llm-service, which owns the Hive and Ollama connections
// (through the OpenAI SDK), retries, error classification and the chunked code
// pipeline. Nothing in the orchestrator talks to a provider directly any more.
package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"authoring/internal/application"
	"authoring/internal/domain"
)

// Client implements application.LLMProviderPort, MetadataSuggesterPort,
// ShortScriptSuggesterPort, StoryboardFinalizerPort and CodePipelinePort.
type Client struct {
	baseURL string
	// timeout bounds a whole call; 0 waits as long as llm-service does (Hive
	// can take minutes on a reasoning model, and HIVE_TIMEOUT_SECONDS=0 means
	// "no timeout" there too).
	timeout time.Duration
	http    *http.Client

	mu          sync.Mutex
	readyAt     time.Time
	readyResult bool
}

func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{baseURL: strings.TrimSuffix(baseURL, "/"), timeout: timeout, http: &http.Client{}}
}

// Name is the provider recorded in llm_usage rows and shown in the GUI status
// panel: the pipeline steps always run on Hive.
func (c *Client) Name() string { return "hive" }

// Ready reports whether the AI path can be offered: llm-service is reachable
// and holds a Hive key. Cached briefly — the GUI polls this.
func (c *Client) Ready() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if time.Since(c.readyAt) < 10*time.Second {
		return c.readyResult
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	ok := false
	if req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil); err == nil {
		if resp, err := c.http.Do(req); err == nil {
			var h struct {
				HiveConfigured bool `json:"hive_configured"`
			}
			if resp.StatusCode == http.StatusOK && json.NewDecoder(resp.Body).Decode(&h) == nil {
				ok = h.HiveConfigured
			}
			resp.Body.Close()
		}
	}
	c.readyAt, c.readyResult = time.Now(), ok
	return ok
}

// --- wire types ---------------------------------------------------------------

type wireUsage struct {
	Model            string `json:"model"`
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	ReasoningTokens  int    `json:"reasoning_tokens"`
	CachedTokens     int    `json:"cached_tokens"`
	ReasoningChars   int    `json:"reasoning_chars"`
	// Absent means reported.
	UsageReported *bool `json:"usage_reported"`
}

func (u wireUsage) usage() application.TokenUsage {
	return application.TokenUsage{
		Model: u.Model, PromptTokens: u.PromptTokens, CompletionTokens: u.CompletionTokens,
		ReasoningTokens: u.ReasoningTokens, CachedTokens: u.CachedTokens,
		ReasoningChars: u.ReasoningChars, UsageMissing: u.UsageReported != nil && !*u.UsageReported,
	}
}

type wireError struct {
	Kind     string    `json:"kind"`
	Provider string    `json:"provider"`
	Message  string    `json:"message"`
	Usage    wireUsage `json:"usage"`
	Partial  string    `json:"partial"`
	Diag     string    `json:"diag"`
}

type wireCall struct {
	Phase        string    `json:"phase"`
	Label        string    `json:"label"`
	Segment      string    `json:"segment"`
	OK           bool      `json:"ok"`
	Usage        wireUsage `json:"usage"`
	DurationMs   int64     `json:"duration_ms"`
	ErrorKind    string    `json:"error_kind"`
	ErrorMessage string    `json:"error_message"`
}

func (c wireCall) call() application.CodeCall {
	return application.CodeCall{
		Phase: c.Phase, Label: c.Label, Segment: c.Segment, OK: c.OK, Usage: c.Usage.usage(),
		Duration:  time.Duration(c.DurationMs) * time.Millisecond,
		ErrorKind: application.LLMErrorKind(c.ErrorKind), ErrorMessage: c.ErrorMessage,
	}
}

type wireDiagnostic struct {
	Message string `json:"message"`
	Line    *int   `json:"line"`
	Kind    string `json:"kind"`
	Rule    string `json:"rule"`
	Shot    string `json:"shot"`
	Segment string `json:"segment"`
}

func (d wireDiagnostic) diagnostic() application.CodeDiagnostic {
	out := application.CodeDiagnostic{Message: d.Message, Kind: d.Kind, Rule: d.Rule, Shot: d.Shot, Segment: d.Segment}
	if d.Line != nil {
		out.Line = *d.Line
	}
	return out
}

func (e wireError) llmError() *application.LLMError {
	provider := e.Provider
	if provider == "" {
		provider = "llm-service"
	}
	kind := application.LLMErrorKind(e.Kind)
	if kind == "" {
		kind = application.ErrKindServer
	}
	return &application.LLMError{
		Kind: kind, Provider: provider, Usage: e.Usage.usage(), Partial: e.Partial, Diag: e.Diag,
		Err: errors.New(e.Message),
	}
}

// errRouteMissing: llm-service answered 404 for the route itself.
var errRouteMissing = errors.New("llm-service has no such route")

func unreachable(err error) *application.LLMError {
	kind := application.ErrKindServer
	if errors.Is(err, context.DeadlineExceeded) {
		kind = application.ErrKindTimeout
	}
	return &application.LLMError{
		Kind: kind, Provider: "llm-service",
		Err: fmt.Errorf("call llm-service: %w", err),
	}
}

func (c *Client) ctx(parent context.Context) (context.Context, context.CancelFunc) {
	if c.timeout > 0 {
		return context.WithTimeout(parent, c.timeout)
	}
	return parent, func() {}
}

// --- streaming ----------------------------------------------------------------

// stream POSTs body to path and reads newline-delimited JSON events. onEvent
// sees every event that is not the final `result`/`error`; the final one is
// returned raw.
func (c *Client) stream(
	ctx context.Context, path string, body any, onEvent func(kind string, raw json.RawMessage),
) (json.RawMessage, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, &application.LLMError{Kind: application.ErrKindMalformed, Provider: "llm-service", Err: fmt.Errorf("marshal request: %w", err)}
	}
	ctx, cancel := c.ctx(ctx)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return nil, &application.LLMError{Kind: application.ErrKindMalformed, Provider: "llm-service", Err: fmt.Errorf("build request: %w", err)}
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, unreachable(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, statusError(resp)
	}

	rd := bufio.NewReaderSize(resp.Body, 64*1024)
	for {
		line, rerr := rd.ReadBytes('\n')
		if trimmed := bytes.TrimSpace(line); len(trimmed) > 0 {
			var head struct {
				Type string `json:"type"`
			}
			if err := json.Unmarshal(trimmed, &head); err != nil {
				return nil, &application.LLMError{Kind: application.ErrKindMalformed, Provider: "llm-service",
					Err: fmt.Errorf("unreadable event from llm-service: %w", err)}
			}
			switch head.Type {
			case "result":
				return json.RawMessage(trimmed), nil
			case "error":
				// A code run streamed its billed calls one by one before this
				// and sends none here (ADR-0030); an error that does list calls
				// hands each to onEvent, so each is recorded exactly once.
				var ev struct {
					Error wireError         `json:"error"`
					Calls []json.RawMessage `json:"calls"`
				}
				if err := json.Unmarshal(trimmed, &ev); err != nil {
					return nil, &application.LLMError{Kind: application.ErrKindMalformed, Provider: "llm-service", Err: err}
				}
				if onEvent != nil {
					for _, c := range ev.Calls {
						onEvent("call", c)
					}
				}
				return nil, ev.Error.llmError()
			default:
				if onEvent != nil {
					onEvent(head.Type, json.RawMessage(trimmed))
				}
			}
		}
		if rerr != nil {
			if errors.Is(rerr, io.EOF) {
				return nil, &application.LLMError{Kind: application.ErrKindServer, Provider: "llm-service",
					Err: errors.New("llm-service closed the stream without a result")}
			}
			if errors.Is(rerr, context.DeadlineExceeded) || errors.Is(rerr, context.Canceled) {
				return nil, &application.LLMError{Kind: application.ErrKindTimeout, Provider: "llm-service", Err: rerr}
			}
			return nil, &application.LLMError{Kind: application.ErrKindServer, Provider: "llm-service", Err: fmt.Errorf("read stream: %w", rerr)}
		}
	}
}

// statusError turns a non-200 answer into an LLMError, reading llm-service's
// {"error": {...}} body when there is one.
func statusError(resp *http.Response) error {
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var body struct {
		Error wireError `json:"error"`
	}
	if json.Unmarshal(raw, &body) == nil && body.Error.Message != "" {
		return body.Error.llmError()
	}
	// A 404 without llm-service's own error body is a route this llm-service
	// does not have (an older build): not a failed call.
	if resp.StatusCode == http.StatusNotFound {
		return errRouteMissing
	}
	kind := application.ErrKindServer
	if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusUnprocessableEntity {
		kind = application.ErrKindMalformed
	}
	msg := strings.TrimSpace(string(raw))
	if len(msg) > 300 {
		msg = msg[:300] + "…"
	}
	return &application.LLMError{Kind: kind, Provider: "llm-service",
		Err: fmt.Errorf("llm-service returned %d: %s", resp.StatusCode, msg)}
}

// post is the non-streaming call: one JSON request, one JSON answer.
func (c *Client) post(ctx context.Context, path string, body, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return &application.LLMError{Kind: application.ErrKindMalformed, Provider: "llm-service", Err: err}
	}
	ctx, cancel := c.ctx(ctx)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return &application.LLMError{Kind: application.ErrKindMalformed, Provider: "llm-service", Err: err}
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return unreachable(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return statusError(resp)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return &application.LLMError{Kind: application.ErrKindMalformed, Provider: "llm-service", Err: fmt.Errorf("decode response: %w", err)}
	}
	return nil
}

// --- application.LLMProviderPort ---------------------------------------------

func (c *Client) Chat(ctx context.Context, req application.ChatRequest) (application.ChatResult, error) {
	body := map[string]any{
		"provider": "hive", "system": req.System, "user": req.User, "model": req.Model,
		"max_tokens": req.MaxTokens, "temperature": req.Temperature,
	}
	raw, err := c.stream(ctx, "/v1/chat", body, func(kind string, ev json.RawMessage) {
		if kind != "progress" || req.OnProgress == nil {
			return
		}
		var p struct {
			ReasoningChars int `json:"reasoning_chars"`
			ContentChars   int `json:"content_chars"`
		}
		if json.Unmarshal(ev, &p) == nil {
			req.OnProgress(application.ChatProgress{ReasoningChars: p.ReasoningChars, ContentChars: p.ContentChars})
		}
	})
	if err != nil {
		return application.ChatResult{}, err
	}
	var res struct {
		Content string    `json:"content"`
		Usage   wireUsage `json:"usage"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		return application.ChatResult{}, &application.LLMError{Kind: application.ErrKindMalformed, Provider: "llm-service", Err: err}
	}
	return application.ChatResult{Content: res.Content, Usage: res.Usage.usage()}, nil
}

// --- MetadataSuggesterPort / ShortScriptSuggesterPort ------------------------

// suggestCall posts a suggestion request. When ctx carries a progress callback
// it asks llm-service to stream and forwards the reply's size as
// it grows; otherwise it is the plain one-shot call.
func (c *Client) suggestCall(ctx context.Context, path string, body map[string]any, out any) error {
	report := application.ProgressFrom(ctx)
	if report == nil {
		return c.post(ctx, path, body, out)
	}
	body["stream"] = true
	raw, err := c.stream(ctx, path, body, func(kind string, ev json.RawMessage) {
		if kind != "progress" {
			return
		}
		var p struct {
			ReasoningChars int `json:"reasoning_chars"`
			ContentChars   int `json:"content_chars"`
		}
		if json.Unmarshal(ev, &p) == nil {
			report(application.ChatProgress{ReasoningChars: p.ReasoningChars, ContentChars: p.ContentChars})
		}
	})
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return &application.LLMError{Kind: application.ErrKindMalformed, Provider: "llm-service", Err: fmt.Errorf("decode response: %w", err)}
	}
	return nil
}

func (c *Client) Suggest(
	ctx context.Context, scriptContent, categoryHint string, language domain.ContentLanguage,
) (string, string, []string, error) {
	var out struct {
		Title       string   `json:"title"`
		Description string   `json:"description"`
		Tags        []string `json:"tags"`
	}
	err := c.suggestCall(ctx, "/v1/suggest-metadata", map[string]any{
		"script_content": scriptContent, "category_hint": categoryHint, "language": string(language),
	}, &out)
	if err != nil {
		return "", "", nil, err
	}
	if out.Tags == nil {
		out.Tags = []string{} // the API contract says tags is an array, never null
	}
	return out.Title, out.Description, out.Tags, nil
}

func (c *Client) SuggestShortScript(
	ctx context.Context, topic, sourceScriptContent string, language domain.ContentLanguage,
) (string, error) {
	var out struct {
		Script string `json:"script"`
	}
	err := c.suggestCall(ctx, "/v1/suggest-short-script", map[string]any{
		"topic": topic, "source_script_content": sourceScriptContent, "language": string(language),
	}, &out)
	if err != nil {
		return "", err
	}
	return out.Script, nil
}

// --- StoryboardFinalizerPort ---------------------------------------------------

func (c *Client) FinalizeStoryboard(
	ctx context.Context, content, model string, maxTokens int, frame domain.Frame,
) (application.FinalizedStoryboard, error) {
	var out struct {
		Storyboard string    `json:"storyboard"`
		Shots      int       `json:"shots"`
		Repaired   bool      `json:"repaired"`
		Usage      wireUsage `json:"usage"`
	}
	if err := c.post(ctx, "/v1/storyboard/finalize", map[string]any{
		"content": content, "model": model, "max_tokens": maxTokens, "frame": frameBody(frame),
	}, &out); err != nil {
		return application.FinalizedStoryboard{}, err
	}
	return application.FinalizedStoryboard{
		Storyboard: out.Storyboard, Shots: out.Shots, Repaired: out.Repaired, Usage: out.Usage.usage(),
	}, nil
}

// --- CodePipelinePort ----------------------------------------------------------

// frameBody is the `frame` of a request: the canvas size. A zero Frame is the
// landscape one.
func frameBody(f domain.Frame) map[string]int {
	if f.Width == 0 {
		f = domain.LandscapeFrame
	}
	return map[string]int{"width": f.Width, "height": f.Height}
}

// codeBody is the /v2/code/* request (docs/contracts/authoring-llm-code-v2.md).
func codeBody(req application.CodeGenRequest) map[string]any {
	body := map[string]any{
		"engine": req.Engine, "topic": req.Topic, "storyboard": req.Storyboard, "system": req.System,
		"model": req.Model, "max_tokens": req.MaxTokens, "chunk_shots": req.ChunkShots,
		"frame": frameBody(req.Frame),
	}
	if len(req.Illustrations) > 0 {
		ills := make([]map[string]any, 0, len(req.Illustrations))
		for _, d := range req.Illustrations {
			ill := map[string]any{"name": d.Name, "usage": d.Usage, "description": d.Description, "code": d.Code}
			if d.Backdrop {
				ill["kind"], ill["shots"] = "backdrop", d.Shots
			}
			ills = append(ills, ill)
		}
		body["illustrations"] = ills
	}
	// What the rendering layout check needs beyond the code.
	if req.SubtitleBand != nil {
		body["subtitle_band"] = map[string]any{"edge": req.SubtitleBand.Edge, "px": req.SubtitleBand.Px}
	}
	if req.VideoFont != "" {
		body["video_font"] = req.VideoFont
	}
	segs := make([]map[string]any, 0, len(req.Done))
	for _, d := range req.Done {
		segs = append(segs, map[string]any{"key": d.Key, "fingerprint": d.Fingerprint, "content": d.Content})
	}
	body["segments"] = segs
	if req.Only != nil {
		body["only"] = req.Only
	}
	return body
}

// wirePlanned is one segment of llm-service's plan. Source is "storyboard"
// for a frame that is the storyboard's own layout (no model call).
type wirePlanned struct {
	Key         string   `json:"key"`
	Kind        string   `json:"kind"`
	Shots       []string `json:"shots"`
	Fingerprint string   `json:"fingerprint"`
	Source      string   `json:"source"`
}

func planFrom(in []wirePlanned) []domain.CodeSegment {
	out := make([]domain.CodeSegment, 0, len(in))
	for i, s := range in {
		shots := s.Shots
		if shots == nil {
			shots = []string{}
		}
		out = append(out, domain.CodeSegment{
			Key: s.Key, Kind: s.Kind, Position: i, Shots: shots, Fingerprint: s.Fingerprint,
			Source: s.Source, Status: domain.SegmentPending,
		})
	}
	return out
}

// codeEvent decodes one streamed event of a code run.
func codeEvent(kind string, raw json.RawMessage) (application.CodeEvent, error) {
	var e struct {
		Phase       string           `json:"phase"`
		Index       int              `json:"index"`
		Total       int              `json:"total"`
		Done        int              `json:"done"`
		Round       int              `json:"round"`
		Targets     []string         `json:"targets"`
		Key         string           `json:"key"`
		Fingerprint string           `json:"fingerprint"`
		Content     json.RawMessage  `json:"content"`
		Source      string           `json:"source"`
		Repaired    bool             `json:"repaired"`
		DurationMs  int              `json:"duration_ms"`
		Error       wireError        `json:"error"`
		FailedShots []string         `json:"failed_shots"`
		Segments    []wirePlanned    `json:"segments"`
		Segment     string           `json:"segment"`
		Diagnostics []wireDiagnostic `json:"diagnostics"`
	}
	if err := json.Unmarshal(raw, &e); err != nil {
		return application.CodeEvent{}, err
	}
	out := application.CodeEvent{
		Type: kind, Phase: e.Phase, Index: e.Index, Total: e.Total, Done: e.Done, Round: e.Round, Targets: e.Targets,
		Key: e.Key, Fingerprint: e.Fingerprint, Content: e.Content, Source: e.Source, Repaired: e.Repaired,
		DurationMS: e.DurationMs, ErrorKind: e.Error.Kind, ErrorText: e.Error.Message, FailedShots: e.FailedShots,
	}
	if string(out.Content) == "null" {
		out.Content = nil
	}
	switch kind {
	case "plan":
		out.Plan = planFrom(e.Segments)
	case "call":
		var c wireCall
		if err := json.Unmarshal(raw, &c); err != nil {
			return application.CodeEvent{}, err
		}
		call := c.call()
		out.Call = &call
	case "check":
		chk := &application.CodeCheck{Phase: e.Phase, Round: e.Round, Segment: e.Segment}
		for _, d := range e.Diagnostics {
			chk.Diagnostics = append(chk.Diagnostics, d.diagnostic())
		}
		out.Check = chk
	}
	return out, nil
}

// GenerateCode runs POST /v2/code/generate (see ADR-0030). Every event is
// handed to onEvent as it arrives; an event this client cannot read fails the
// run rather than being skipped, because a skipped segment_done or call would
// lose paid work or its cost.
//
// An llm-service without /v2 is ErrSegmentsUnsupported; there is no
// fallback, so llm-service must be deployed first.
func (c *Client) GenerateCode(
	ctx context.Context, req application.CodeGenRequest, onEvent func(application.CodeEvent),
) (application.CodeGenResult, error) {
	var badEvent error
	raw, err := c.stream(ctx, "/v2/code/generate", codeBody(req), func(kind string, ev json.RawMessage) {
		e, err := codeEvent(kind, ev)
		if err != nil {
			if badEvent == nil {
				badEvent = fmt.Errorf("unreadable %q event from llm-service: %w", kind, err)
			}
			return
		}
		if onEvent != nil {
			onEvent(e)
		}
	})
	if errors.Is(err, errRouteMissing) {
		return application.CodeGenResult{}, application.ErrSegmentsUnsupported
	}
	if err != nil {
		return application.CodeGenResult{}, err
	}
	if badEvent != nil {
		return application.CodeGenResult{}, &application.LLMError{Kind: application.ErrKindMalformed, Provider: "llm-service", Err: badEvent}
	}
	var res struct {
		Status         string           `json:"status"`
		Code           string           `json:"code"`
		CheckOK        bool             `json:"check_ok"`
		Diagnostics    []wireDiagnostic `json:"diagnostics"`
		RepairRounds   int              `json:"repair_rounds"`
		Warnings       []string         `json:"warnings"`
		SceneClassName string           `json:"scene_class_name"`
		Failed         []string         `json:"failed"`
		Missing        []string         `json:"missing"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		return application.CodeGenResult{}, &application.LLMError{Kind: application.ErrKindMalformed, Provider: "llm-service", Err: err}
	}
	if res.Status != application.CodeGenDone && res.Status != application.CodeGenIncomplete {
		return application.CodeGenResult{}, &application.LLMError{Kind: application.ErrKindMalformed, Provider: "llm-service",
			Err: fmt.Errorf("llm-service answered an unknown run status %q", res.Status)}
	}
	out := application.CodeGenResult{
		Status: res.Status, Code: res.Code, CheckOK: res.CheckOK, RepairRounds: res.RepairRounds,
		Warnings: res.Warnings, SceneClassName: res.SceneClassName, Failed: res.Failed, Missing: res.Missing,
	}
	for _, d := range res.Diagnostics {
		out.Diagnostics = append(out.Diagnostics, d.diagnostic())
	}
	return out, nil
}

// segmentCall POSTs a one-segment request and maps llm-service's answer:
// 404 unknown segment, 409 not ready, 422 unusable reply.
func (c *Client) segmentCall(ctx context.Context, path string, body map[string]any, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return &application.LLMError{Kind: application.ErrKindMalformed, Provider: "llm-service", Err: err}
	}
	ctx, cancel := c.ctx(ctx)
	defer cancel()
	hreq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return &application.LLMError{Kind: application.ErrKindMalformed, Provider: "llm-service", Err: err}
	}
	hreq.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(hreq)
	if err != nil {
		return unreachable(err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return &application.LLMError{Kind: application.ErrKindMalformed, Provider: "llm-service", Err: fmt.Errorf("decode response: %w", err)}
		}
		return nil
	case http.StatusNotFound, http.StatusConflict, http.StatusUnprocessableEntity:
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		var eb struct {
			Error wireError `json:"error"`
		}
		msg := strings.TrimSpace(string(raw))
		own := json.Unmarshal(raw, &eb) == nil && eb.Error.Message != ""
		if own {
			msg = eb.Error.Message
		}
		switch resp.StatusCode {
		case http.StatusNotFound:
			if !own {
				// The route itself is missing: an llm-service without /v2.
				return application.ErrSegmentsUnsupported
			}
			return application.ErrSegmentUnknown
		case http.StatusConflict:
			return &application.ErrSegmentNotReady{Message: msg}
		default:
			return &application.ErrSegmentReply{Message: msg}
		}
	default:
		return statusError(resp)
	}
}

// PlanSegments is how llm-service cuts this storyboard into segments; it is
// the only place that rule lives (no model call).
func (c *Client) PlanSegments(ctx context.Context, req application.CodeGenRequest) ([]domain.CodeSegment, error) {
	var out struct {
		Segments []wirePlanned `json:"segments"`
	}
	if err := c.segmentCall(ctx, "/v2/code/plan", codeBody(req), &out); err != nil {
		return nil, err
	}
	return planFrom(out.Segments), nil
}

func (c *Client) SegmentPrompt(ctx context.Context, req application.CodeGenRequest, key string) (string, string, error) {
	body := codeBody(req)
	body["key"] = key
	var out struct {
		System string `json:"system"`
		User   string `json:"user"`
	}
	if err := c.segmentCall(ctx, "/v2/code/segment-prompt", body, &out); err != nil {
		return "", "", err
	}
	return out.System, out.User, nil
}

func (c *Client) ParseSegment(ctx context.Context, req application.CodeGenRequest, key, reply string) (string, json.RawMessage, error) {
	body := codeBody(req)
	body["key"], body["reply"] = key, reply
	var out struct {
		Fingerprint string          `json:"fingerprint"`
		Content     json.RawMessage `json:"content"`
	}
	if err := c.segmentCall(ctx, "/v2/code/segment-parse", body, &out); err != nil {
		return "", nil, err
	}
	return out.Fingerprint, out.Content, nil
}
