package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"orchestrator/internal/domain"
)

// runCode is the AI flow's code step (CR-039): instead of one model call that
// writes the whole file, llm-service writes a shared layout/cast, then the
// shots in chunks, merges them deterministically, compile-checks the result and
// repairs only the shots that fail.
//
// Every model call of the run is recorded as its own llm_usage row, on success
// and on failure alike — a run that dies on its fifth chunk still paid for the
// first four.
func (uc *GenerateAuthoringUseCase) runCode(
	ctx context.Context, project *domain.Project, rendered RenderedPrompt, role domain.PromptRole,
	model string, info *runInfo, started time.Time,
) (GeneratedStep, error) {
	const step = "code"
	projectID := project.ProjectID
	if uc.codegen == nil {
		return GeneratedStep{}, errors.New("the code pipeline is not wired")
	}
	storyboard, err := uc.projects.GetAuthoringStoryboard(ctx, projectID)
	if err != nil {
		return GeneratedStep{}, fmt.Errorf("load storyboard: %w", err)
	}
	if strings.TrimSpace(storyboard) == "" {
		return GeneratedStep{}, errors.New("chưa có storyboard — hãy chạy bước 1b trước")
	}
	topic, err := uc.projects.GetAuthoringTopic(ctx, projectID)
	if err != nil {
		return GeneratedStep{}, fmt.Errorf("load topic: %w", err)
	}

	result, genErr := uc.codegen.GenerateCode(ctx, CodeGenRequest{
		Engine: string(project.RenderEngine), Topic: topic, Storyboard: storyboard,
		System: rendered.Prompt, Model: model, MaxTokens: uc.maxOutputTokens,
	}, func(ev CodeEvent) { uc.updateCodeProgress(projectID, step, ev) })

	var calls []CodeCall
	var llmErr *LLMError
	if genErr != nil {
		if errors.As(genErr, &llmErr) {
			calls = llmErr.Calls
		}
	} else {
		calls = result.Calls
	}
	var total TokenUsage
	for _, c := range calls {
		uc.recordCall(ctx, string(role), step, projectID, c, model)
		total = usageSum(total, c.Usage)
	}
	info.usage = total

	if genErr != nil {
		if llmErr != nil {
			info.partialChars = len(llmErr.Partial)
		}
		return GeneratedStep{}, genErr
	}
	if strings.TrimSpace(result.Code) == "" {
		return GeneratedStep{}, &LLMError{
			Kind: ErrKindEmpty, Provider: uc.provider.Name(), Usage: total,
			Err: errors.New("the code pipeline returned no code"),
		}
	}

	out := GeneratedStep{
		Step: step, Role: string(role), Content: result.Code, Provider: uc.provider.Name(),
		Usage: total, RepairRounds: result.RepairRounds, Warnings: result.Warnings, ModelCalls: len(calls),
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

// recordCall writes one llm_usage row for one model call of a run.
func (uc *GenerateAuthoringUseCase) recordCall(
	ctx context.Context, role, step, projectID string, c CodeCall, requestedModel string,
) {
	if c.Cached {
		return // nothing was billed
	}
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
