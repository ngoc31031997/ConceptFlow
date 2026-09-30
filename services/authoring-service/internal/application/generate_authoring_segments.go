package application

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"authoring/internal/domain"
)

// CR-050 FR-4/5/7/9 — what the Creator does with the code step's segments
// outside a run: see them, copy one segment's prompt for an outside AI, paste
// or hand-edit one segment's result, and choose the shots per segment.

// CodeSegmentsView is the segment list the code step's panel shows.
type CodeSegmentsView struct {
	ChunkShots int                  `json:"chunk_shots"`
	Running    bool                 `json:"running"`
	Segments   []domain.CodeSegment `json:"segments"`
}

// ErrStoryboardNotSegmentable: the saved storyboard is not the AI flow's JSON
// (written by hand for the manual flow), so it has no shots to cut.
type ErrStoryboardNotSegmentable struct{ Cause error }

func (e *ErrStoryboardNotSegmentable) Error() string {
	return "storyboard hiện tại không chia được thành đoạn (" + e.Cause.Error() + ") — chạy lại Bước 4 — Visual bằng AI"
}

// CodeSegments lists the segments of the current storyboard, cut by
// llm-service with the Creator's shots per segment, each with what is stored
// for it. Stored segments the current cut no longer has are not shown (the
// next run deletes them). Before any run every segment reads as pending.
func (uc *GenerateAuthoringUseCase) CodeSegments(ctx context.Context, projectID string) (CodeSegmentsView, error) {
	if uc.segments == nil {
		return CodeSegmentsView{}, errors.New("the code segment store is not wired")
	}
	project, err := uc.projects.Get(ctx, projectID)
	if err != nil {
		return CodeSegmentsView{}, fmt.Errorf("load project: %w", err)
	}
	n, err := uc.segments.GetCodeChunkShots(ctx, projectID)
	if err != nil {
		return CodeSegmentsView{}, fmt.Errorf("load shots per segment: %w", err)
	}
	view := CodeSegmentsView{ChunkShots: n, Running: uc.Progress(projectID, "code").Running, Segments: []domain.CodeSegment{}}
	storyboard, err := uc.projects.GetAuthoringStoryboard(ctx, projectID)
	if err != nil {
		return CodeSegmentsView{}, fmt.Errorf("load storyboard: %w", err)
	}
	if strings.TrimSpace(storyboard) == "" {
		return view, nil
	}
	cut, err := uc.planSegments(ctx, project, storyboard, n)
	if err != nil {
		return CodeSegmentsView{}, err
	}
	stored, err := uc.segments.ListSegments(ctx, projectID, "code")
	if err != nil {
		return CodeSegmentsView{}, fmt.Errorf("load the code segments: %w", err)
	}
	byKey := make(map[string]domain.CodeSegment, len(stored))
	for _, s := range stored {
		byKey[s.Key] = s
	}
	for _, s := range cut {
		if have, ok := byKey[s.Key]; ok {
			have.Position = s.Position
			s = have
		} else if s.Source == domain.SegmentSourceStoryboard {
			// No model call: the frame is the storyboard's own layout.
			s.Status = domain.SegmentDone
		} else {
			s.Source = ""
		}
		s.Fingerprint = ""
		view.Segments = append(view.Segments, s)
	}
	return view, nil
}

// planSegments asks llm-service for the cut (the one place its rule lives).
func (uc *GenerateAuthoringUseCase) planSegments(ctx context.Context, project *domain.Project, storyboard string, n int) ([]domain.CodeSegment, error) {
	if uc.codegen == nil {
		return nil, errors.New("the code pipeline is not wired")
	}
	if n < 1 || n > domain.MaxChunkShots {
		return nil, domain.ErrInvalidChunkShots
	}
	cut, err := uc.codegen.PlanSegments(ctx, CodeGenRequest{
		Engine: string(project.RenderEngine), Storyboard: storyboard, ChunkShots: n,
	})
	var llmErr *LLMError
	if errors.As(err, &llmErr) && llmErr.Kind == ErrKindMalformed {
		return nil, &ErrStoryboardNotSegmentable{Cause: llmErr.Err}
	}
	return cut, err
}

// codeSegmentRequest is the request a run of this project's code step would
// send, with the segments stored so far, for the one-segment calls.
func (uc *GenerateAuthoringUseCase) codeSegmentRequest(ctx context.Context, projectID string) (CodeGenRequest, error) {
	if uc.codegen == nil || uc.segments == nil {
		return CodeGenRequest{}, errors.New("the code pipeline is not wired")
	}
	project, err := uc.projects.Get(ctx, projectID)
	if err != nil {
		return CodeGenRequest{}, fmt.Errorf("load project: %w", err)
	}
	role, err := AIRoleFor("code", string(project.RenderEngine))
	if err != nil {
		return CodeGenRequest{}, err
	}
	rendered, err := uc.renderer.Execute(ctx, projectID, role)
	if err != nil {
		return CodeGenRequest{}, err
	}
	req, err := uc.codeRequest(ctx, project, rendered.Prompt, "")
	if err != nil {
		return CodeGenRequest{}, err
	}
	if err := uc.sendStoredSegments(ctx, projectID, &req, ""); err != nil {
		return CodeGenRequest{}, err
	}
	return req, nil
}

// CodeSegmentPrompt is the exact turn one segment would be asked, for the
// Creator to run in an outside AI (FR-5).
func (uc *GenerateAuthoringUseCase) CodeSegmentPrompt(ctx context.Context, projectID, key string) (system, user string, err error) {
	req, err := uc.codeSegmentRequest(ctx, projectID)
	if err != nil {
		return "", "", err
	}
	return uc.codegen.SegmentPrompt(ctx, req, key)
}

// PasteCodeSegment stores a reply written outside the pipeline — pasted from
// an outside AI or edited by hand — as the segment's result, once llm-service
// has checked it has the segment's shape (FR-5). Not while the step runs.
func (uc *GenerateAuthoringUseCase) PasteCodeSegment(
	ctx context.Context, projectID, key, reply, source string,
) (domain.CodeSegment, error) {
	if source != domain.SegmentSourceExternal && source != domain.SegmentSourceManual {
		return domain.CodeSegment{}, fmt.Errorf("source must be %q or %q", domain.SegmentSourceExternal, domain.SegmentSourceManual)
	}
	if strings.TrimSpace(reply) == "" {
		return domain.CodeSegment{}, &ErrSegmentReply{Message: "chưa dán gì"}
	}
	release, err := uc.acquire(projectID, "code")
	if err != nil {
		return domain.CodeSegment{}, err
	}
	defer release()
	req, err := uc.codeSegmentRequest(ctx, projectID)
	if err != nil {
		return domain.CodeSegment{}, err
	}
	project, err := uc.projects.Get(ctx, projectID)
	if err != nil {
		return domain.CodeSegment{}, fmt.Errorf("load project: %w", err)
	}
	cut, err := uc.planSegments(ctx, project, req.Storyboard, req.ChunkShots)
	if err != nil {
		return domain.CodeSegment{}, err
	}
	var seg *domain.CodeSegment
	for i := range cut {
		if cut[i].Key == key {
			seg = &cut[i]
		}
	}
	if seg == nil {
		return domain.CodeSegment{}, ErrSegmentUnknown
	}
	fingerprint, content, err := uc.codegen.ParseSegment(ctx, req, key, reply)
	if err != nil {
		return domain.CodeSegment{}, err
	}
	seg.Fingerprint, seg.Content, seg.Source, seg.Status = fingerprint, content, source, domain.SegmentDone
	if err := uc.segments.SaveSegmentDone(ctx, projectID, "code", *seg, false); err != nil {
		return domain.CodeSegment{}, fmt.Errorf("store segment %s: %w", key, err)
	}
	return *seg, nil
}

// SetCodeChunkShots stores the Creator's shots per segment (FR-7). Not while
// the step runs: the run in flight was cut with the old size.
func (uc *GenerateAuthoringUseCase) SetCodeChunkShots(ctx context.Context, projectID string, n int) error {
	if uc.segments == nil {
		return errors.New("the code segment store is not wired")
	}
	if n < 1 || n > domain.MaxChunkShots {
		return domain.ErrInvalidChunkShots
	}
	release, err := uc.acquire(projectID, "code")
	if err != nil {
		return err
	}
	defer release()
	return uc.segments.SaveCodeChunkShots(ctx, projectID, n)
}

// FailInterruptedSegments is the startup sweep (NFR-3): a segment still
// marked running was cut off when this service stopped.
func (uc *GenerateAuthoringUseCase) FailInterruptedSegments(ctx context.Context) (int64, error) {
	if uc.segments == nil {
		return 0, nil
	}
	return uc.segments.FailAllRunningSegments(ctx, domain.SegmentErrInterrupted, "bị ngắt khi authoring-service khởi động lại")
}
