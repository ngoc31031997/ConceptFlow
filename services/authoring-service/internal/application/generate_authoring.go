package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"authoring/internal/domain"
)

// GenerateAuthoringUseCase runs one authoring step with the configured LLM
// provider instead of the copy-out-to-an-AI round trip.
//
// It adds a second way to do a step; it does not replace the first. The Copy
// button keeps working, renders through the SAME RenderPromptUseCase, and is
// the documented fallback whenever this path is unavailable — no key, no
// credit, provider down. That is why every failure here is
// classified rather than collapsed into "AI failed": the Creator needs to
// know whether to top up an account or just paste the prompt elsewhere.
//
// What it deliberately does NOT do: advance the wizard, submit
// anything to the render saga, or touch the other steps' saved output. It
// renders, calls, saves that one step, and hands the text back for the
// Creator to edit.
type GenerateAuthoringUseCase struct {
	renderer authoringPromptRenderer
	// illustrations is the drawing stage; nil = off (Manim, or not wired).
	illustrations IllustrationStagePort
	provider      LLMProviderPort
	recorder      *LLMUsageRecorder
	projects      PromptRenderContextPort
	models        AuthoringModelsReaderPort
	story         authoringStorySaver
	storyboard    authoringContentSaver
	code          authoringContentSaver
	// errorLog receives a row for every failed run; nil disables it.
	errorLog ProjectErrorLogPort
	// events receives one journal line per run start/end; nil disables it.
	events domain.ProjectEventPort
	// finalizer validates the storyboard JSON; codegen runs the chunked code
	// pipeline. Both are llm-service; a run of the step that needs one
	// fails loudly when it is missing rather than quietly doing something else.
	finalizer StoryboardFinalizerPort
	codegen   CodePipelinePort
	// segments stores the code step segment by segment (see ADR-0030).
	segments CodeSegmentPort
	// formats/calibration feed the post-1b narration length check;
	// see WithStoryboardChecks. Nil formats = no length check.
	formats     FormatLookupPort
	calibration VoiceCalibrationPort

	// maxInputChars is HIVE_MAX_INPUT_CHARS: not a context limit (Hive's
	// window is 1M tokens) but a blast radius, so one broken project cannot
	// bill for an unbounded prompt.
	maxInputChars   int
	maxOutputTokens int

	// running holds the (project, step) pairs a call is in flight for, so a
	// double click costs one billed call instead of two. In-memory
	// on purpose — the lock only has to outlive a single HTTP request, and a
	// Postgres advisory lock would buy cross-replica correctness for a
	// single-replica orchestrator.
	mu       sync.Mutex
	running  map[string]bool
	progress map[string]*AuthoringProgress
}

// authoringPromptRenderer is RenderPromptUseCase. Shared with the Copy path
// by construction: one renderer, so the two paths cannot send different text
// to the same model.
type authoringPromptRenderer interface {
	Execute(ctx context.Context, projectID string, role domain.PromptRole) (RenderedPrompt, error)
}

// authoringStorySaver is SaveAuthoringStoryUseCase — step 1 alone carries the
// topic alongside its content.
type authoringStorySaver interface {
	Execute(ctx context.Context, projectID, content, topic string) error
}

// authoringContentSaver is the shape steps 2–3 share.
type authoringContentSaver interface {
	Execute(ctx context.Context, projectID, content string) error
}

// AuthoringModelsReaderPort is the read side of the model-per-step picker —
// separate from AuthoringModelsPort (prompt_templates.go's write side) since
// this use case only ever reads it, once per call, to know which model to
// pass the provider.
type AuthoringModelsReaderPort interface {
	GetAuthoringModels(ctx context.Context, projectID string) (domain.AuthoringStepModels, error)
}

// ErrGenerateBusy is the second of two concurrent clicks — 409, not
// an error the Creator did anything about.
var ErrGenerateBusy = errors.New("một lượt chạy AI cho bước này đang diễn ra")

// ErrLLMNotConfigured means no API key is set, so this path does not exist
// for this deployment. The Copy button still does.
var ErrLLMNotConfigured = errors.New("chưa cấu hình API key cho nhà cung cấp AI")

func NewGenerateAuthoringUseCase(
	renderer authoringPromptRenderer,
	provider LLMProviderPort,
	recorder *LLMUsageRecorder,
	projects PromptRenderContextPort,
	models AuthoringModelsReaderPort,
	story authoringStorySaver,
	storyboard authoringContentSaver,
	code authoringContentSaver,
	maxInputChars, maxOutputTokens int,
) *GenerateAuthoringUseCase {
	return &GenerateAuthoringUseCase{
		renderer: renderer, provider: provider, recorder: recorder,
		projects: projects, models: models, story: story, storyboard: storyboard,
		code:          code,
		maxInputChars: maxInputChars, maxOutputTokens: maxOutputTokens,
		running: map[string]bool{}, progress: map[string]*AuthoringProgress{},
	}
}

// WithErrorLog makes every failed run leave a row in the project's
// project_errors column.
// WithEvents attaches the project journey log: each finished authoring run
// (1a/1b/1c) leaves one line with its duration, size and token use.
func (uc *GenerateAuthoringUseCase) WithEvents(events domain.ProjectEventPort) *GenerateAuthoringUseCase {
	uc.events = events
	return uc
}

func (uc *GenerateAuthoringUseCase) WithErrorLog(log ProjectErrorLogPort) *GenerateAuthoringUseCase {
	uc.errorLog = log
	return uc
}

// WithPipeline wires the two llm-service capabilities the AI flow's storyboard
// and code steps need.
func (uc *GenerateAuthoringUseCase) WithPipeline(f StoryboardFinalizerPort, c CodePipelinePort) *GenerateAuthoringUseCase {
	uc.finalizer, uc.codegen = f, c
	return uc
}

// GeneratedStep is what one run produced, plus what it cost. The content is
// returned as well as saved so the GUI can drop it straight into the editor
// without a second round trip.
type GeneratedStep struct {
	Step     string     `json:"step"`
	Role     string     `json:"role"`
	Content  string     `json:"content"`
	Provider string     `json:"provider"`
	Usage    TokenUsage `json:"usage"`
	// SaveError is set when the call succeeded but persisting its output did
	// not — a locked project, say. The content still comes back: the
	// tokens are already paid for, and the Creator can keep the text in the
	// editor rather than buy it a second time.
	SaveError string `json:"save_error,omitempty"`
	// Set by the code step. CheckFailed means the script was saved but
	// still fails the check after the last repair round; the Creator can
	// read Diagnostics and fix it by hand. CompileIssues and LayoutIssues
	// count Diagnostics by kind: a script with layout issues only still
	// renders, some drawings may just reach past the safe area.
	CheckFailed   bool     `json:"check_failed,omitempty"`
	Diagnostics   []string `json:"diagnostics,omitempty"`
	CompileIssues int      `json:"compile_issues,omitempty"`
	LayoutIssues  int      `json:"layout_issues,omitempty"`
	RepairRounds  int      `json:"repair_rounds,omitempty"`
	Warnings      []string `json:"warnings,omitempty"`
	ModelCalls    int      `json:"model_calls,omitempty"`
	// Set by the illustrations step when it finished drawing but some
	// drawings still wait for the Creator; Message says which.
	AwaitingReview bool   `json:"awaiting_review,omitempty"`
	Message        string `json:"message,omitempty"`
}

// Available reports whether the AI path can be offered at all. The
// GUI asks so it can explain a missing button instead of showing one that
// fails when pressed.
func (uc *GenerateAuthoringUseCase) Available() bool {
	if uc == nil || uc.provider == nil {
		return false
	}
	// The provider knows whether it can actually serve (llm-service reachable,
	// Hive key set); one that does not say is taken at its word.
	if r, ok := uc.provider.(interface{ Ready() bool }); ok {
		return r.Ready()
	}
	return true
}

// Provider names the provider the GUI would be calling, for the same status
// panel.
func (uc *GenerateAuthoringUseCase) Provider() string {
	if !uc.Available() {
		return ""
	}
	return uc.provider.Name()
}

// Execute runs one step end to end: render the prompt, call the provider,
// record the cost, save the output.
//
// web-gui chạy ba bước bằng cách gọi hàm này ba lần theo thứ tự, chứ
// không có một endpoint "chạy cả chuỗi": mỗi lượt đã tự lưu kết quả rồi, nên
// bước sau render prompt từ đúng dữ liệu bước trước vừa lưu, và một bước hỏng
// giữa chừng không xoá mất những bước đã xong.
func (uc *GenerateAuthoringUseCase) Execute(
	ctx context.Context, projectID, step string,
) (GeneratedStep, error) {
	return uc.execute(ctx, projectID, step, CodeRunOptions{})
}

// ExecuteCode runs the code step with options: one segment only, or
// every segment dropped and written again. Execute("code") runs the missing
// segments, which is what the chain does.
func (uc *GenerateAuthoringUseCase) ExecuteCode(
	ctx context.Context, projectID string, opts CodeRunOptions,
) (GeneratedStep, error) {
	return uc.execute(ctx, projectID, "code", opts)
}

func (uc *GenerateAuthoringUseCase) execute(
	ctx context.Context, projectID, step string, opts CodeRunOptions,
) (GeneratedStep, error) {
	started := time.Now()
	// Flow trace: one start + one end line per run, keyed by project_id, so a
	// project's journey (which step, how long, how big, how it ended) can be
	// reconstructed from the logs alone.
	slog.Info("authoring step start", "project_id", projectID, "step", step)
	out, info, err := uc.run(ctx, projectID, step, opts)
	uc.traceEnd(projectID, step, out, info, err, started)
	if errors.Is(err, ErrGenerateBusy) || errors.Is(err, ErrLLMNotConfigured) {
		// Not a run: nothing started, so the journal gets no end line either.
	} else if err != nil {
		uc.recordEvent(ctx, projectID, step, domain.RunFailed, started, out, info, err.Error())
	} else if out.SaveError != "" {
		uc.recordEvent(ctx, projectID, step, domain.RunFailed, started, out, info, out.SaveError)
	} else if out.CheckFailed {
		uc.recordEvent(ctx, projectID, step, domain.RunFailed, started, out, info,
			out.checkFailedHead()+": "+strings.Join(out.Diagnostics, " · "))
	} else {
		uc.recordEvent(ctx, projectID, step, domain.RunDone, started, out, info, "")
	}
	if err != nil {
		uc.logError(ctx, projectID, step, err, info, started)
	} else if out.SaveError != "" {
		uc.logError(ctx, projectID, step, errors.New(out.SaveError), info, started)
	} else if out.CheckFailed {
		uc.logError(ctx, projectID, step, &LLMError{
			Kind: ErrKindCheckFailed, Provider: "llm-service",
			Diag: strings.Join(out.Diagnostics, "\n"),
			Err:  fmt.Errorf("the script was saved but still fails the %s check after %d repair round(s)", out.failedCheck(), out.RepairRounds),
		}, info, started)
	}
	return out, err
}

// LayoutOnly reports whether the issues left after a failed check are all
// layout issues: the script compiles and renders.
func (g GeneratedStep) LayoutOnly() bool {
	return g.CheckFailed && g.CompileIssues == 0 && g.LayoutIssues > 0
}

// failedCheck names the check a saved script still fails, for the logs.
func (g GeneratedStep) failedCheck() string {
	if g.LayoutOnly() {
		return "layout"
	}
	return "compile"
}

// checkFailedHead is the journal's one-line verdict on a saved script that
// still fails the check, naming the kind of issue that is left.
func (g GeneratedStep) checkFailedHead() string {
	if g.LayoutOnly() {
		return fmt.Sprintf("code đã lưu, còn %d lỗi bố cục (không chặn render)", g.LayoutIssues)
	}
	if g.LayoutIssues > 0 {
		return fmt.Sprintf("code đã lưu nhưng vẫn lỗi biên dịch (kèm %d lỗi bố cục)", g.LayoutIssues)
	}
	return "code đã lưu nhưng vẫn lỗi biên dịch"
}

// recordEvent journals a run's start or end. Best-effort, like logError: a lost
// journal line must not fail a run the Creator waited minutes for.
func (uc *GenerateAuthoringUseCase) recordEvent(
	ctx context.Context, projectID, step string, state domain.RunState, started time.Time,
	out GeneratedStep, info runInfo, detail string,
) {
	if uc.events == nil || projectID == "" {
		return
	}
	fs, source := domain.FlowStepForAuthoring(step), "authoring"
	if step == StepIllustrations {
		// Illustrations has its own numbered flow step.
		fs = domain.FlowIllustrations
	}
	if fs == 0 {
		return
	}
	if detail == "" && out.AwaitingReview {
		detail = out.Message
	}
	e := domain.ProjectEvent{
		ProjectID: projectID, FlowStep: fs, RunState: state, Source: source, Detail: detail,
	}
	if state != domain.RunRunning {
		e.DurationMS = time.Since(started).Milliseconds()
		e.ContentChars = len(out.Content)
		e.PromptTokens, e.CompletionTokens = info.usage.PromptTokens, info.usage.CompletionTokens
	}
	_ = uc.events.AppendProjectEvent(context.WithoutCancel(ctx), e)
}

// traceEnd logs how a run ended. Busy/not-configured are not failures but are
// still worth a line: they explain a step that "did nothing".
func (uc *GenerateAuthoringUseCase) traceEnd(
	projectID, step string, out GeneratedStep, info runInfo, err error, started time.Time,
) {
	attrs := []any{
		"project_id", projectID, "step", step,
		"elapsed_ms", time.Since(started).Milliseconds(),
		"content_chars", len(out.Content),
		"prompt_tokens", info.usage.PromptTokens, "completion_tokens", info.usage.CompletionTokens,
	}
	switch {
	case err != nil:
		slog.Warn("authoring step failed", append(attrs, "partial_chars", info.partialChars, "error", err.Error())...)
	case out.SaveError != "":
		slog.Warn("authoring step generated but not saved", append(attrs, "error", out.SaveError)...)
	case out.CheckFailed:
		slog.Warn("authoring step saved with compile errors", append(attrs, "repair_rounds", out.RepairRounds)...)
	default:
		slog.Info("authoring step done", attrs...)
	}
}

// runInfo is what run learned that the error log wants but the return value
// does not carry.
type runInfo struct {
	partialChars int
	usage        TokenUsage
	// codeOpts are the code step's run options.
	codeOpts CodeRunOptions
}

// logError never fails the run: it is a trace, and losing a trace must not
// turn one problem into two. WithoutCancel because the usual reason for a
// failed run is that the caller went away, which cancels ctx too.
func (uc *GenerateAuthoringUseCase) logError(
	ctx context.Context, projectID, step string, err error, info runInfo, started time.Time,
) {
	if uc.errorLog == nil || projectID == "" {
		return
	}
	// Not failures: a double click, and a deployment with no key.
	if errors.Is(err, ErrGenerateBusy) || errors.Is(err, ErrLLMNotConfigured) {
		return
	}
	e := ProjectError{
		At: time.Now().UTC(), Source: "authoring", Step: step,
		Kind: string(LLMErrorKindOf(err)), Message: err.Error(), Detail: errorDetail(err),
		PartialChars:   info.partialChars,
		ElapsedSeconds: int(time.Since(started).Seconds()),
	}
	var llmErr *LLMError
	if errors.As(err, &llmErr) {
		e.Provider = llmErr.Provider
	}
	if info.usage.PromptTokens+info.usage.CompletionTokens > 0 {
		e.Usage = &TokenUsageJSON{
			Model: info.usage.Model, PromptTokens: info.usage.PromptTokens,
			CompletionTokens: info.usage.CompletionTokens, ReasoningTokens: info.usage.ReasoningTokens,
		}
	}
	_ = uc.errorLog.AppendProjectError(context.WithoutCancel(ctx), projectID, e)
}

func (uc *GenerateAuthoringUseCase) run(
	ctx context.Context, projectID, step string, opts CodeRunOptions,
) (GeneratedStep, runInfo, error) {
	info := runInfo{codeOpts: opts}
	out, err := uc.runInner(ctx, projectID, step, &info)
	return out, info, err
}

func (uc *GenerateAuthoringUseCase) runInner(
	ctx context.Context, projectID, step string, info *runInfo,
) (GeneratedStep, error) {
	if projectID == "" {
		return GeneratedStep{}, fmt.Errorf("project_id is required")
	}
	if !uc.Available() {
		return GeneratedStep{}, ErrLLMNotConfigured
	}

	project, err := uc.projects.Get(ctx, projectID)
	if err != nil {
		return GeneratedStep{}, fmt.Errorf("load project: %w", err)
	}
	if step == StepIllustrations {
		return uc.runIllustrations(ctx, project)
	}
	// Step → role is the server's decision, read off the project's
	// engine, not the GUI's.
	role, err := AIRoleFor(step, string(project.RenderEngine))
	if err != nil {
		return GeneratedStep{}, err
	}

	release, err := uc.acquire(projectID, step)
	if err != nil {
		return GeneratedStep{}, err
	}
	defer release()
	uc.recordEvent(ctx, projectID, step, domain.RunRunning, time.Now(), GeneratedStep{}, runInfo{}, "")

	rendered, err := uc.renderer.Execute(ctx, projectID, role)
	if err != nil {
		return GeneratedStep{}, err
	}
	if uc.maxInputChars > 0 && len(rendered.Prompt) > uc.maxInputChars {
		// Refuse rather than truncate: half a prompt produces a plausible
		// answer to a question nobody asked, and the Creator has no way to
		// see which half went missing.
		return GeneratedStep{}, fmt.Errorf(
			"prompt dài %d ký tự, vượt trần HIVE_MAX_INPUT_CHARS=%d — hãy rút gọn kết quả các bước trước",
			len(rendered.Prompt), uc.maxInputChars)
	}

	// Model-per-step picker: "" (unset, or this use case wired without a
	// models port — e.g. an older test) falls through to the provider's own
	// configured default, same as before this existed.
	var model string
	if uc.models != nil {
		stepModels, err := uc.models.GetAuthoringModels(ctx, projectID)
		if err != nil {
			return GeneratedStep{}, fmt.Errorf("load authoring models: %w", err)
		}
		model = stepModels.ModelFor(step)
	}
	if err := checkCodeModel(step, model); err != nil {
		return GeneratedStep{}, err
	}

	started := time.Now()
	uc.beginProgress(projectID, step, started)
	defer uc.endProgress(projectID, step)

	if step == "code" {
		return uc.runCode(ctx, project, rendered, role, model, info, started)
	}

	result, chatErr := uc.provider.Chat(ctx, ChatRequest{
		OnProgress: func(p ChatProgress) { uc.updateProgress(projectID, step, p) },
		// The rendered template is the whole instruction. It goes in the
		// system slot, with a minimal user turn, because the prompts are
		// written as standing instructions ("bạn là Story Architect..."),
		// not as a question.
		System:      rendered.Prompt,
		User:        userTurnNudge,
		MaxTokens:   uc.maxOutputTokens,
		Temperature: 0.7,
		Model:       model,
	})
	uc.recorder.Record(ctx, RecordFor(
		uc.provider.Name(), string(role), step, projectID, result.Usage, started, chatErr,
	))
	if chatErr != nil {
		info.usage = billedUsage(chatErr)
		info.partialChars = len(partialOf(chatErr))
		return GeneratedStep{}, chatErr
	}
	info.usage = result.Usage

	content := strings.TrimSpace(result.Content)
	if content == "" {
		return GeneratedStep{}, &LLMError{
			Kind: ErrKindEmpty, Provider: uc.provider.Name(),
			Usage: result.Usage, Err: errors.New("provider returned no content"),
		}
	}

	usage := result.Usage
	var warnings []string
	if step == "storyboard" {
		// The storyboard is JSON the code step splits by shot, so it is
		// validated here — with one model repair turn if it is not — and saved in
		// canonical form. An unusable storyboard is an error, never saved as-is.
		if uc.finalizer == nil {
			return GeneratedStep{}, errors.New("the storyboard finalizer is not wired")
		}
		frame := domain.FrameFor(project.VideoOutputMode)
		fin, finErr := uc.finalizer.FinalizeStoryboard(ctx, content, model, uc.maxOutputTokens, frame)
		if finErr != nil {
			fixed := billedUsage(finErr)
			uc.recordPhase(ctx, string(role), step, "storyboard_fix", projectID, fixed, started, finErr)
			info.usage = usageSum(result.Usage, fixed)
			return GeneratedStep{}, finErr
		}
		if fin.Repaired {
			uc.recordPhase(ctx, string(role), step, "storyboard_fix", projectID, fin.Usage, started, nil)
		}
		content = fin.Storyboard
		usage = usageSum(result.Usage, fin.Usage)
		info.usage = usage
		// Cheap checks the Creator sees before the code step
		// spends money on this storyboard. Warnings only; never block.
		warnings = uc.storyboardWarnings(ctx, project, content)
	}

	// Saving overwrites this step and only this step; the existing
	// save use cases already carry the draft lock and the
	// history write, so an AI run is audited exactly like a paste.
	out := GeneratedStep{
		Step: step, Role: string(role), Content: content,
		Provider: uc.provider.Name(), Usage: usage, Warnings: warnings,
	}
	if err := uc.save(ctx, projectID, step, content); err != nil {
		out.SaveError = fmt.Sprintf("Đã sinh được nội dung nhưng chưa lưu được: %v", err)
	}
	return out, nil
}

// userTurnNudge is the one-line nudge that follows the system prompt. Some
// providers answer an empty user turn with a question back.
const userTurnNudge = "Hãy thực hiện nhiệm vụ trên và chỉ trả về kết quả theo đúng định dạng đã mô tả."

func (uc *GenerateAuthoringUseCase) save(ctx context.Context, projectID, step, content string) error {
	switch step {
	case "story":
		if uc.story == nil {
			return fmt.Errorf("save-story use case is not wired")
		}
		// topic "" — the project already has its topic (POST /v1/projects
		// wrote it when the Creator typed it); passing "" leaves it untouched.
		return uc.story.Execute(ctx, projectID, content, "")
	case "storyboard":
		return saveWith(ctx, uc.storyboard, projectID, content, step)
	case "code":
		return saveWith(ctx, uc.code, projectID, content, step)
	default:
		return fmt.Errorf("unknown step %q", step)
	}
}

func saveWith(ctx context.Context, saver authoringContentSaver, projectID, content, step string) error {
	if saver == nil {
		return fmt.Errorf("save-%s use case is not wired", step)
	}
	return saver.Execute(ctx, projectID, content)
}

// acquire takes the (project, step) lock and returns its release. A second
// caller gets ErrGenerateBusy instead of queueing — waiting would hide the
// double click behind a long spinner and still bill twice.
func (uc *GenerateAuthoringUseCase) acquire(projectID, step string) (func(), error) {
	key := projectID + "\x00" + step
	uc.mu.Lock()
	defer uc.mu.Unlock()
	if uc.running[key] {
		return nil, ErrGenerateBusy
	}
	uc.running[key] = true
	return func() {
		uc.mu.Lock()
		delete(uc.running, key)
		uc.mu.Unlock()
	}, nil
}

// AuthoringProgress is what the GUI polls while a run is in flight: which
// phase the model is in and how much it has produced so far.
type AuthoringProgress struct {
	Running bool `json:"running"`
	// "idle" | "waiting" | "reasoning" | "writing" | code step: "layout" | "cast" |
	// "chunks" | "merge" | "check" | "repair" | illustrations step: "plan" | "draw"
	Phase          string `json:"phase"`
	ReasoningChars int    `json:"reasoning_chars"`
	ContentChars   int    `json:"content_chars"`
	ElapsedSeconds int    `json:"elapsed_seconds"`
	// The code step's own progress: chunks finished of the total, and
	// the current repair round of the maximum.
	ChunksDone  int `json:"chunks_done,omitempty"`
	ChunksTotal int `json:"chunks_total,omitempty"`
	RepairRound int `json:"repair_round,omitempty"`
	RepairMax   int `json:"repair_max,omitempty"`
	// The illustrations step: drawings to draw in this run, finished
	// (drawn or failed), failed, served by the library, and the list's size.
	DrawingsTotal   int `json:"drawings_total,omitempty"`
	DrawingsDone    int `json:"drawings_done,omitempty"`
	DrawingsFailed  int `json:"drawings_failed,omitempty"`
	DrawingsReused  int `json:"drawings_reused,omitempty"`
	DrawingsPlanned int `json:"drawings_planned,omitempty"`

	started time.Time
}

func progressKey(projectID, step string) string { return projectID + "\x00" + step }

func (uc *GenerateAuthoringUseCase) beginProgress(projectID, step string, started time.Time) {
	uc.mu.Lock()
	defer uc.mu.Unlock()
	uc.progress[progressKey(projectID, step)] = &AuthoringProgress{Running: true, Phase: "waiting", started: started}
}

func (uc *GenerateAuthoringUseCase) updateProgress(projectID, step string, p ChatProgress) {
	uc.mu.Lock()
	defer uc.mu.Unlock()
	st := uc.progress[progressKey(projectID, step)]
	if st == nil {
		return
	}
	st.ReasoningChars, st.ContentChars = p.ReasoningChars, p.ContentChars
	if p.ContentChars > 0 {
		st.Phase = "writing"
	} else if p.ReasoningChars > 0 {
		st.Phase = "reasoning"
	}
}

func (uc *GenerateAuthoringUseCase) endProgress(projectID, step string) {
	uc.mu.Lock()
	defer uc.mu.Unlock()
	delete(uc.progress, progressKey(projectID, step))
}

// Progress returns the live state of a (project, step) run; Running is false
// when nothing is in flight.
func (uc *GenerateAuthoringUseCase) Progress(projectID, step string) AuthoringProgress {
	uc.mu.Lock()
	defer uc.mu.Unlock()
	st := uc.progress[progressKey(projectID, step)]
	if st == nil {
		return AuthoringProgress{Phase: "idle"}
	}
	out := *st
	out.ElapsedSeconds = int(time.Since(st.started).Seconds())
	return out
}

// errorDetail is the raw error chain plus, when the provider attached one, its
// diagnostics (HTTP status, request id, finish_reason, response body...) —
// what a developer needs to see exactly what the API said.
func errorDetail(err error) string {
	var llmErr *LLMError
	if errors.As(err, &llmErr) && llmErr.Diag != "" {
		return err.Error() + "\n" + llmErr.Diag
	}
	return err.Error()
}
