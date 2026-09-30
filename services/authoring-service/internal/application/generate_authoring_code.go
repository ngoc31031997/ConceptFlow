package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"authoring/internal/domain"
)

// runCode is the AI flow's code step (CR-039): instead of one model call that
// writes the whole file, llm-service writes a shared layout/cast, then the
// shots in chunks, merges them deterministically, compile-checks the result and
// repairs only the shots that fail.
//
// CR-050 (ADR-0030): the run is cut into segments stored here as they finish.
// A run sends the segments already done and llm-service writes only the
// missing ones (or the one segment the Creator re-runs). A failed segment does
// not stop the others: the step ends "incomplete" and keeps everything else.
// Every model call is recorded as its own llm_usage row the moment it ends,
// and every failed check is logged (FR-21, FR-22).
func (uc *GenerateAuthoringUseCase) runCode(
	ctx context.Context, project *domain.Project, rendered RenderedPrompt, role domain.PromptRole,
	model string, info *runInfo, started time.Time,
) (GeneratedStep, error) {
	const step = "code"
	projectID := project.ProjectID
	if uc.codegen == nil {
		return GeneratedStep{}, errors.New("the code pipeline is not wired")
	}
	if uc.segments == nil {
		return GeneratedStep{}, errors.New("the code segment store is not wired")
	}
	req, err := uc.codeRequest(ctx, project, rendered.Prompt, model)
	if err != nil {
		return GeneratedStep{}, err
	}
	opts := info.codeOpts
	if opts.Fresh {
		if err := uc.segments.DeleteSegments(ctx, projectID, step); err != nil {
			return GeneratedStep{}, fmt.Errorf("drop the code segments: %w", err)
		}
	}
	if err := uc.sendStoredSegments(ctx, projectID, &req, opts.Segment); err != nil {
		return GeneratedStep{}, err
	}
	if opts.Segment != "" {
		req.Only = []string{opts.Segment}
	}

	run := &codeRun{uc: uc, projectID: projectID, role: string(role), model: model, engine: req.Engine,
		// Stored even when the caller has gone: a segment the model finished is paid for.
		ctx: context.WithoutCancel(ctx), plan: map[string]domain.CodeSegment{}}
	result, genErr := uc.codegen.GenerateCode(ctx, req, run.onEvent)
	info.usage = run.total

	if genErr != nil {
		// NFR-3: a segment still running when the run stopped is failed with
		// the reason, so the Creator sees it and can run it again.
		kind, reason := domain.SegmentErrInterrupted, "bị ngắt: "+genErr.Error()
		if ctx.Err() != nil {
			kind, reason = domain.SegmentErrCancelled, "đã huỷ"
		}
		if _, err := uc.segments.FailRunningSegments(run.ctx, projectID, step, kind, reason); err != nil {
			run.fail(fmt.Errorf("mark the running segments failed: %w", err))
		}
		var llmErr *LLMError
		if errors.As(genErr, &llmErr) {
			info.partialChars = len(llmErr.Partial)
		}
		return GeneratedStep{}, errors.Join(genErr, run.persistErr)
	}
	if run.persistErr != nil {
		return GeneratedStep{}, fmt.Errorf("không lưu được kết quả các đoạn code: %w", run.persistErr)
	}
	if result.Status != CodeGenDone {
		return GeneratedStep{}, uc.incomplete(ctx, projectID)
	}
	if strings.TrimSpace(result.Code) == "" {
		return GeneratedStep{}, &LLMError{
			Kind: ErrKindEmpty, Provider: uc.provider.Name(), Usage: run.total,
			Err: errors.New("the code pipeline returned no code"),
		}
	}

	out := GeneratedStep{
		Step: step, Role: string(role), Content: result.Code, Provider: uc.provider.Name(),
		Usage: run.total, RepairRounds: result.RepairRounds, Warnings: result.Warnings, ModelCalls: run.calls,
	}
	if !result.CheckOK {
		// Saved anyway: the tokens are spent, and the Creator can read the
		// diagnostics and fix the script by hand. Flagged, never passed off as clean.
		out.CheckFailed = true
		for _, d := range result.Diagnostics {
			if d.Line > 0 {
				out.Diagnostics = append(out.Diagnostics, fmt.Sprintf("dòng %d: %s", d.Line, d.Message))
			} else {
				out.Diagnostics = append(out.Diagnostics, d.Message)
			}
		}
	}
	if err := uc.save(ctx, projectID, step, result.Code); err != nil {
		out.SaveError = fmt.Sprintf("Đã sinh được nội dung nhưng chưa lưu được: %v", err)
	}
	return out, nil
}

// codeRequest is everything llm-service needs for this project's code step,
// shared by a run and the one-segment prompt/paste paths so they fingerprint
// the same input the same way.
func (uc *GenerateAuthoringUseCase) codeRequest(
	ctx context.Context, project *domain.Project, system, model string,
) (CodeGenRequest, error) {
	projectID := project.ProjectID
	storyboard, err := uc.projects.GetAuthoringStoryboard(ctx, projectID)
	if err != nil {
		return CodeGenRequest{}, fmt.Errorf("load storyboard: %w", err)
	}
	if strings.TrimSpace(storyboard) == "" {
		return CodeGenRequest{}, errors.New("chưa có storyboard — hãy chạy Bước 4 — Visual trước")
	}
	topic, err := uc.projects.GetAuthoringTopic(ctx, projectID)
	if err != nil {
		return CodeGenRequest{}, fmt.Errorf("load topic: %w", err)
	}

	// CR-044/045: a Remotion video's drawings are planned and drawn by the
	// illustrations step before this one; the code step only checks that the
	// list exists and that the Creator approved or skipped every drawing.
	var drawings []LibraryDrawing
	if project.RenderEngine == domain.RenderEngineRemotion && uc.illustrations != nil {
		pending, planned, err := uc.illustrations.Gate(ctx, projectID)
		if err != nil {
			return CodeGenRequest{}, fmt.Errorf("check the video's drawings: %w", err)
		}
		if !planned {
			return CodeGenRequest{}, ErrIllustrationsNotPlanned
		}
		stale, err := uc.illustrations.Stale(ctx, projectID)
		if err != nil {
			return CodeGenRequest{}, fmt.Errorf("check the video's drawings: %w", err)
		}
		if stale {
			return CodeGenRequest{}, ErrIllustrationsStale
		}
		if len(pending) > 0 {
			return CodeGenRequest{}, &ErrIllustrationsPending{Rows: pending}
		}
		if drawings, err = uc.illustrations.ForCode(ctx, projectID); err != nil {
			return CodeGenRequest{}, fmt.Errorf("load library drawings: %w", err)
		}
	}

	req := CodeGenRequest{
		Engine: string(project.RenderEngine), Topic: topic, Storyboard: storyboard,
		System: system, Model: model, MaxTokens: uc.maxOutputTokens, Illustrations: drawings,
	}
	if project.RenderEngine == domain.RenderEngineRemotion {
		// CR-048 T6b: the same subtitle strip {{subtitle_zone}} told the model
		// to keep clear, and the video's font, for the layout check.
		if band, burned := domain.ProjectSubtitleBand(project); burned {
			req.SubtitleBand = &band
		}
		req.VideoFont = project.VideoFont
	}
	if uc.segments != nil {
		if req.ChunkShots, err = uc.segments.GetCodeChunkShots(ctx, projectID); err != nil {
			return CodeGenRequest{}, fmt.Errorf("load shots per segment: %w", err)
		}
	}
	return req, nil
}

// sendStoredSegments adds the finished segments to req, except `skip` (the
// segment the Creator asked to write again).
func (uc *GenerateAuthoringUseCase) sendStoredSegments(ctx context.Context, projectID string, req *CodeGenRequest, skip string) error {
	stored, err := uc.segments.ListSegments(ctx, projectID, "code")
	if err != nil {
		return fmt.Errorf("load the code segments: %w", err)
	}
	for _, s := range stored {
		if s.Status == domain.SegmentDone && len(s.Content) > 0 && s.Key != skip {
			req.Done = append(req.Done, DoneSegment{Key: s.Key, Fingerprint: s.Fingerprint, Content: s.Content})
		}
	}
	return nil
}

// incomplete reads the segments back to say what is left (FR-2).
func (uc *GenerateAuthoringUseCase) incomplete(ctx context.Context, projectID string) error {
	segs, err := uc.segments.ListSegments(ctx, projectID, "code")
	if err != nil {
		return fmt.Errorf("the code step is not finished, and its segments could not be read: %w", err)
	}
	out := &ErrSegmentsIncomplete{Total: len(segs)}
	for _, s := range segs {
		switch s.Status {
		case domain.SegmentDone:
		case domain.SegmentFailed:
			out.Failed = append(out.Failed, s)
		default:
			out.Missing++
		}
	}
	return out
}

// CodeRunOptions are the Creator's choices for one code run (CR-050 FR-4).
type CodeRunOptions struct {
	// Segment re-runs this one segment only ("Chạy lại đoạn này").
	Segment string
	// Fresh drops every stored segment first ("Sinh lại toàn bộ").
	Fresh bool
}

// CodeSegmentPort stores the code step's segments (CR-050, ADR-0030).
type CodeSegmentPort interface {
	ListSegments(ctx context.Context, projectID, step string) ([]domain.CodeSegment, error)
	ApplySegmentPlan(ctx context.Context, projectID, step string, plan []domain.CodeSegment) error
	MarkSegmentRunning(ctx context.Context, projectID, step, key string) error
	SaveSegmentDone(ctx context.Context, projectID, step string, s domain.CodeSegment, repaired bool) error
	SaveSegmentFailed(ctx context.Context, projectID, step, key, kind, message string, durationMS int) error
	FailRunningSegments(ctx context.Context, projectID, step, kind, message string) (int64, error)
	FailAllRunningSegments(ctx context.Context, kind, message string) (int64, error)
	DeleteSegments(ctx context.Context, projectID, step string) error
	GetCodeChunkShots(ctx context.Context, projectID string) (int, error)
	SaveCodeChunkShots(ctx context.Context, projectID string, n int) error
	InsertCheckDiagnostics(ctx context.Context, list []domain.CheckDiagnosticRecord) error
}

// WithSegments turns on the CR-050 segment store the code step needs.
func (uc *GenerateAuthoringUseCase) WithSegments(store CodeSegmentPort) *GenerateAuthoringUseCase {
	uc.segments = store
	return uc
}

// ErrSegmentsIncomplete: the code step ended with segments failed or not run.
// Everything that finished is stored; the Creator re-runs or pastes the rest.
type ErrSegmentsIncomplete struct {
	Total   int
	Failed  []domain.CodeSegment
	Missing int
}

func (e *ErrSegmentsIncomplete) Error() string {
	var parts []string
	for i, s := range e.Failed {
		if i == 3 {
			parts = append(parts, fmt.Sprintf("và %d đoạn khác", len(e.Failed)-3))
			break
		}
		reason := s.ErrorKind
		if reason == "" {
			reason = "lỗi"
		}
		parts = append(parts, fmt.Sprintf("%s: %s", s.Key, reason))
	}
	msg := fmt.Sprintf("Bước Code chưa xong: %d/%d đoạn lỗi", len(e.Failed), e.Total)
	if len(parts) > 0 {
		msg += " (" + strings.Join(parts, "; ") + ")"
	}
	if e.Missing > 0 {
		msg += fmt.Sprintf(", %d đoạn chưa chạy", e.Missing)
	}
	return msg + ". Các đoạn đã xong đã được lưu — chạy lại đoạn lỗi hoặc dán kết quả AI ngoài."
}

// codeRun turns a run's events into stored segments, usage rows and the
// failed-check log, as they arrive.
type codeRun struct {
	uc        *GenerateAuthoringUseCase
	ctx       context.Context
	projectID string
	role      string
	model     string
	engine    string
	plan      map[string]domain.CodeSegment
	total     TokenUsage
	calls     int
	// persistErr is the first store that failed: the run goes on (the model
	// work is not wasted for the other segments) but the step reports it.
	persistErr error
}

func (r *codeRun) fail(err error) {
	if r.persistErr == nil {
		r.persistErr = err
	}
}

func (r *codeRun) onEvent(ev CodeEvent) {
	const step = "code"
	uc := r.uc
	switch ev.Type {
	case "plan":
		for _, s := range ev.Plan {
			r.plan[s.Key] = s
		}
		if err := uc.segments.ApplySegmentPlan(r.ctx, r.projectID, step, ev.Plan); err != nil {
			r.fail(fmt.Errorf("store the segment plan: %w", err))
		}
	case "segment_start":
		if err := uc.segments.MarkSegmentRunning(r.ctx, r.projectID, step, ev.Key); err != nil {
			r.fail(fmt.Errorf("mark segment %s running: %w", ev.Key, err))
		}
	case "segment_done":
		seg, ok := r.plan[ev.Key]
		if !ok {
			r.fail(fmt.Errorf("segment %s is not in the run's plan", ev.Key))
			return
		}
		seg.Fingerprint, seg.Content, seg.Source, seg.DurationMS = ev.Fingerprint, ev.Content, ev.Source, ev.DurationMS
		if err := uc.segments.SaveSegmentDone(r.ctx, r.projectID, step, seg, ev.Repaired); err != nil {
			r.fail(fmt.Errorf("store segment %s: %w", ev.Key, err))
		}
	case "segment_failed":
		if err := uc.segments.SaveSegmentFailed(r.ctx, r.projectID, step, ev.Key, ev.ErrorKind, ev.ErrorText, ev.DurationMS); err != nil {
			r.fail(fmt.Errorf("mark segment %s failed: %w", ev.Key, err))
		}
	case "call":
		if ev.Call == nil {
			return
		}
		r.calls++
		r.total = usageSum(r.total, ev.Call.Usage)
		uc.recordCall(r.ctx, r.role, step, r.projectID, *ev.Call, r.model)
	case "check":
		if ev.Check == nil {
			return
		}
		list := make([]domain.CheckDiagnosticRecord, 0, len(ev.Check.Diagnostics))
		for _, d := range ev.Check.Diagnostics {
			list = append(list, domain.CheckDiagnosticRecord{
				ProjectID: r.projectID, Engine: r.engine, Phase: ev.Check.Phase, Round: ev.Check.Round,
				SegmentKey: d.Segment, ShotID: d.Shot, Kind: d.Kind, Rule: d.Rule, Message: d.Message, Line: d.Line,
			})
		}
		// A lost statistics row must not fail a run the Creator waited for;
		// it is logged, like a lost usage row.
		if err := uc.segments.InsertCheckDiagnostics(r.ctx, list); err != nil {
			slog.Warn("could not log check diagnostics", "project_id", r.projectID, "error", err)
		}
	default:
		uc.updateCodeProgress(r.projectID, step, ev)
	}
}

// ErrModelNotForCode refuses a model the code step cannot use (CR-050 FR-19).
type ErrModelNotForCode struct{ Model string }

func (e *ErrModelNotForCode) Error() string {
	return fmt.Sprintf("model %q không dùng được cho bước Code và Hình minh hoạ (model local không viết nổi code cảnh; "+
		"mọi lượt đã đo đều hết giờ) — chọn DeepSeek hoặc GLM ở ô Model AI", e.Model)
}

// checkCodeModel applies domain.ModelAllowedForStep to a code-writing step.
func checkCodeModel(step, model string) error {
	if !domain.ModelAllowedForStep(step, model) {
		return &ErrModelNotForCode{Model: model}
	}
	return nil
}

// IllustrationStagePort is the per-video drawing list as the illustrations and
// code steps need it.
type IllustrationStagePort interface {
	Prepare(ctx context.Context, projectID, model string, report func(StageReport)) ([]domain.ProjectIllustration, error)
	Gate(ctx context.Context, projectID string) ([]domain.ProjectIllustration, bool, error)
	// Stale: the list was planned from an older storyboard (CR-050 FR-17).
	Stale(ctx context.Context, projectID string) (bool, error)
	ForCode(ctx context.Context, projectID string) ([]LibraryDrawing, error)
}

// WithIllustrations turns on the CR-044 drawing stage for Remotion projects.
func (uc *GenerateAuthoringUseCase) WithIllustrations(stage IllustrationStagePort) *GenerateAuthoringUseCase {
	uc.illustrations = stage
	return uc
}

// recordCall writes one llm_usage row for one model call of a run.
func (uc *GenerateAuthoringUseCase) recordCall(
	ctx context.Context, role, step, projectID string, c CodeCall, requestedModel string,
) {
	model := c.Usage.Model
	if model == "" {
		model = requestedModel
	}
	uc.recorder.Record(ctx, LLMUsageRecord{
		Provider: uc.provider.Name(), Model: model, Role: role, Step: step, Phase: c.Phase,
		ProjectID: projectID, PromptTokens: c.Usage.PromptTokens, CompletionTokens: c.Usage.CompletionTokens,
		ReasoningTokens: c.Usage.ReasoningTokens, CachedTokens: c.Usage.CachedTokens,
		Duration: c.Duration, OK: c.OK, ErrorKind: c.ErrorKind,
	})
}

// recordPhase writes a usage row for a secondary call inside a step (the
// storyboard repair turn).
func (uc *GenerateAuthoringUseCase) recordPhase(
	ctx context.Context, role, step, phase, projectID string, usage TokenUsage, started time.Time, err error,
) {
	rec := RecordFor(uc.provider.Name(), role, step, projectID, usage, started, err)
	rec.Phase = phase
	uc.recorder.Record(ctx, rec)
}

func usageSum(a, b TokenUsage) TokenUsage {
	model := a.Model
	if model == "" {
		model = b.Model
	}
	return TokenUsage{
		Model: model, PromptTokens: a.PromptTokens + b.PromptTokens,
		CompletionTokens: a.CompletionTokens + b.CompletionTokens,
		ReasoningTokens:  a.ReasoningTokens + b.ReasoningTokens, CachedTokens: a.CachedTokens + b.CachedTokens,
	}
}

func (uc *GenerateAuthoringUseCase) updateCodeProgress(projectID, step string, ev CodeEvent) {
	uc.mu.Lock()
	defer uc.mu.Unlock()
	st := uc.progress[progressKey(projectID, step)]
	if st == nil {
		return
	}
	switch ev.Type {
	case "phase":
		st.Phase = ev.Phase
		if ev.Phase == "chunks" {
			st.ChunksTotal = ev.Total
		}
		if ev.Phase == "repair" {
			st.RepairRound, st.RepairMax = ev.Round, ev.Total
		}
	case "chunk_done":
		st.ChunksDone, st.ChunksTotal = ev.Done, ev.Total
	}
}

// StepIllustrations is the CR-045 authoring step between Visual and Code: plan
// the video's drawings from the storyboard and draw the missing ones. Remotion only.
const StepIllustrations = "illustrations"

// runIllustrations is the illustrations step. It is not a prompt-and-save step:
// the planner and the drawer record their own model calls, and the result is
// the list itself. It ends "awaiting review" while any drawing still needs the
// Creator — not an error, the chain simply stops there.
func (uc *GenerateAuthoringUseCase) runIllustrations(ctx context.Context, project *domain.Project) (GeneratedStep, error) {
	const step = StepIllustrations
	projectID := project.ProjectID
	if project.RenderEngine != domain.RenderEngineRemotion {
		return GeneratedStep{}, errors.New("bước Hình minh hoạ chỉ dùng cho video Remotion")
	}
	if uc.illustrations == nil {
		return GeneratedStep{}, errors.New("the illustrations stage is not wired")
	}
	release, err := uc.acquire(projectID, step)
	if err != nil {
		return GeneratedStep{}, err
	}
	defer release()
	uc.recordEvent(ctx, projectID, step, domain.RunRunning, time.Now(), GeneratedStep{}, runInfo{}, "")

	// The drawings are code, so they use the code step's model.
	var model string
	if uc.models != nil {
		stepModels, err := uc.models.GetAuthoringModels(ctx, projectID)
		if err != nil {
			return GeneratedStep{}, fmt.Errorf("load authoring models: %w", err)
		}
		model = stepModels.ModelFor("code")
	}
	if err := checkCodeModel(step, model); err != nil {
		return GeneratedStep{}, err
	}

	uc.beginProgress(projectID, step, time.Now())
	defer uc.endProgress(projectID, step)
	pending, err := uc.illustrations.Prepare(ctx, projectID, model, func(r StageReport) {
		uc.updateIllustrationProgress(projectID, r)
	})
	if err != nil {
		return GeneratedStep{}, err
	}
	out := GeneratedStep{Step: step, Role: plannerRole, Provider: uc.provider.Name()}
	if len(pending) > 0 {
		out.AwaitingReview = true
		out.Message = AwaitingReviewMessage(pending)
	}
	return out, nil
}

func (uc *GenerateAuthoringUseCase) updateIllustrationProgress(projectID string, r StageReport) {
	uc.mu.Lock()
	defer uc.mu.Unlock()
	st := uc.progress[progressKey(projectID, StepIllustrations)]
	if st == nil {
		return
	}
	st.Phase = r.Phase
	st.DrawingsTotal, st.DrawingsDone, st.DrawingsFailed = r.Total, r.Done, r.Failed
	st.DrawingsReused, st.DrawingsPlanned = r.Reused, r.Planned
}
