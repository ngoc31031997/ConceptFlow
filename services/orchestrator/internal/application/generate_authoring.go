package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"orchestrator/internal/domain"
)

// GenerateAuthoringUseCase runs one authoring step with the configured LLM
// provider instead of the copy-out-to-an-AI round trip (CR-027 FR78).
//
// It adds a second way to do a step; it does not replace the first. The Copy
// button keeps working, renders through the SAME RenderPromptUseCase, and is
// the documented fallback whenever this path is unavailable — no key, no
// credit, provider down (FR77.4/FR83.2). That is why every failure here is
// classified rather than collapsed into "AI failed": the Creator needs to
// know whether to top up an account or just paste the prompt elsewhere.
//
// What it deliberately does NOT do (FR78.2): advance the wizard, submit
// anything to the render saga, or touch the other steps' saved output. It
// renders, calls, saves that one step, and hands the text back for the
// Creator to edit.
type GenerateAuthoringUseCase struct {
	renderer   authoringPromptRenderer
	provider   LLMProviderPort
	recorder   *LLMUsageRecorder
	projects   PromptRenderContextPort
	story      authoringStorySaver
	storyboard authoringContentSaver
	code       authoringContentSaver
	review     authoringContentSaver

	// maxInputChars is HIVE_MAX_INPUT_CHARS: not a context limit (Hive's
	// window is 1M tokens) but a blast radius, so one broken project cannot
	// bill for an unbounded prompt (FR80.2).
	maxInputChars   int
	maxOutputTokens int

	// running holds the (project, step) pairs a call is in flight for, so a
	// double click costs one billed call instead of two (FR78.4). In-memory
	// on purpose — the lock only has to outlive a single HTTP request, and a
	// Postgres advisory lock would buy cross-replica correctness for a
	// single-replica orchestrator.
	mu      sync.Mutex
	running map[string]bool
}

// authoringPromptRenderer is RenderPromptUseCase. Shared with the Copy path
// by construction: one renderer, so the two paths cannot send different text
// to the same model (FR77.3).
type authoringPromptRenderer interface {
	Execute(ctx context.Context, projectID string, role domain.PromptRole, lintResults string) (RenderedPrompt, error)
}

// authoringStorySaver is SaveAuthoringStoryUseCase — step 1 alone carries the
// topic alongside its content.
type authoringStorySaver interface {
	Execute(ctx context.Context, projectID, content, topic string) error
}

// authoringContentSaver is the shape steps 2–4 share.
type authoringContentSaver interface {
	Execute(ctx context.Context, projectID, content string) error
}

// ErrGenerateBusy is the second of two concurrent clicks (FR78.4) — 409, not
// an error the Creator did anything about.
var ErrGenerateBusy = errors.New("một lượt chạy AI cho bước này đang diễn ra")

// ErrLLMNotConfigured means no API key is set, so this path does not exist
// for this deployment. The Copy button still does (FR83.2).
var ErrLLMNotConfigured = errors.New("chưa cấu hình API key cho nhà cung cấp AI")

func NewGenerateAuthoringUseCase(
	renderer authoringPromptRenderer,
	provider LLMProviderPort,
	recorder *LLMUsageRecorder,
	projects PromptRenderContextPort,
	story authoringStorySaver,
	storyboard authoringContentSaver,
	code authoringContentSaver,
	review authoringContentSaver,
	maxInputChars, maxOutputTokens int,
) *GenerateAuthoringUseCase {
	return &GenerateAuthoringUseCase{
		renderer: renderer, provider: provider, recorder: recorder,
		projects: projects, story: story, storyboard: storyboard,
		code: code, review: review,
		maxInputChars: maxInputChars, maxOutputTokens: maxOutputTokens,
		running: map[string]bool{},
	}
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
	// not — a locked project (FR84.2), say. The content still comes back: the
	// tokens are already paid for, and the Creator can keep the text in the
	// editor rather than buy it a second time.
	SaveError string `json:"save_error,omitempty"`
}

// Available reports whether the AI path can be offered at all (FR79.4). The
// GUI asks so it can explain a missing button instead of showing one that
// fails when pressed.
func (uc *GenerateAuthoringUseCase) Available() bool {
	return uc != nil && uc.provider != nil
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
// lintResults is passed through to the renderer for the review step, exactly
// as the Copy path passes it.
func (uc *GenerateAuthoringUseCase) Execute(
	ctx context.Context, projectID, step, lintResults string,
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
	// FR78.5 — step → role is the server's decision, read off the project's
	// engine. The GUI used to work this out, in two places.
	role, err := RoleFor(step, string(project.RenderEngine))
	if err != nil {
		return GeneratedStep{}, err
	}

	release, err := uc.acquire(projectID, step)
	if err != nil {
		return GeneratedStep{}, err
	}
	defer release()

	rendered, err := uc.renderer.Execute(ctx, projectID, role, lintResults)
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

	started := time.Now()
	result, chatErr := uc.provider.Chat(ctx, ChatRequest{
		// The rendered template is the whole instruction. It goes in the
		// system slot, with a minimal user turn, because the prompts are
		// written as standing instructions ("bạn là Story Architect..."),
		// not as a question.
		System:      rendered.Prompt,
		User:        userTurnFor(role),
		MaxTokens:   uc.maxOutputTokens,
		Temperature: 0.7,
	})
	uc.recorder.Record(ctx, RecordFor(
		uc.provider.Name(), string(role), step, projectID, result.Usage, started, chatErr,
	))
	if chatErr != nil {
		return GeneratedStep{}, chatErr
	}

	content := strings.TrimSpace(result.Content)
	if content == "" {
		return GeneratedStep{}, &LLMError{
			Kind: ErrKindEmpty, Provider: uc.provider.Name(),
			Usage: result.Usage, Err: errors.New("provider returned no content"),
		}
	}

	// FR78.3 — saving overwrites this step and only this step; the existing
	// save use cases already carry the FR84.2 draft lock and the FR84.3
	// history write, so an AI run is audited exactly like a paste.
	out := GeneratedStep{
		Step: step, Role: string(role), Content: content,
		Provider: uc.provider.Name(), Usage: result.Usage,
	}
	if err := uc.save(ctx, projectID, step, content); err != nil {
		out.SaveError = fmt.Sprintf("Đã sinh được nội dung nhưng chưa lưu được: %v", err)
	}
	return out, nil
}

// userTurnFor is the one-line nudge that follows the system prompt. Some
// providers answer an empty user turn with a question back.
func userTurnFor(role domain.PromptRole) string {
	if role == domain.RoleScriptReviewer {
		return "Hãy duyệt và trả kết quả theo đúng định dạng đã mô tả."
	}
	return "Hãy thực hiện nhiệm vụ trên và chỉ trả về kết quả theo đúng định dạng đã mô tả."
}

func (uc *GenerateAuthoringUseCase) save(ctx context.Context, projectID, step, content string) error {
	switch step {
	case "story":
		if uc.story == nil {
			return fmt.Errorf("save-story use case is not wired")
		}
		// topic "" — the project already has its topic (CR-028 FR83.1 wrote
		// it when the Creator typed it); passing "" leaves it untouched.
		return uc.story.Execute(ctx, projectID, content, "")
	case "storyboard":
		return saveWith(ctx, uc.storyboard, projectID, content, step)
	case "code":
		return saveWith(ctx, uc.code, projectID, content, step)
	case "review":
		return saveWith(ctx, uc.review, projectID, content, step)
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
