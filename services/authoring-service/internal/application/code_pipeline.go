package application

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"authoring/internal/domain"
)

// The AI flow's storyboard and code steps are not one model call each.
// These ports are how this service hands them to llm-service,
// which owns every conversation with a model.

// StoryboardFinalizerPort validates the Visual Director's JSON (and gets one
// model repair turn if it is invalid) and returns the canonical text to save.
type StoryboardFinalizerPort interface {
	FinalizeStoryboard(ctx context.Context, content, model string, maxTokens int, frame domain.Frame) (FinalizedStoryboard, error)
}

type FinalizedStoryboard struct {
	Storyboard string
	Shots      int
	// Repaired is true when the first answer was invalid and a second model
	// call fixed it; Usage then includes that call.
	Repaired bool
	Usage    TokenUsage
}

// CodePipelinePort runs the chunked code pipeline: layout/cast → chunks in
// parallel → merge → compile check → repair. The run (see ADR-0030)
// is cut into segments this service stores: each run is sent the segments
// already done, writes only the missing ones (or Only), and streams every
// segment result, billed call and failed check as it happens.
type CodePipelinePort interface {
	GenerateCode(ctx context.Context, req CodeGenRequest, onEvent func(CodeEvent)) (CodeGenResult, error)
	// PlanSegments is how llm-service cuts req.Storyboard into segments
	// (Engine and ChunkShots matter; no model call). This service keeps no copy
	// of that rule.
	PlanSegments(ctx context.Context, req CodeGenRequest) ([]domain.CodeSegment, error)
	// SegmentPrompt is the exact (system, user) turn one segment would be
	// asked, for an outside AI.
	SegmentPrompt(ctx context.Context, req CodeGenRequest, key string) (system, user string, err error)
	// ParseSegment checks a reply written outside the pipeline for one
	// segment and returns its fingerprint and content.
	ParseSegment(ctx context.Context, req CodeGenRequest, key, reply string) (fingerprint string, content json.RawMessage, err error)
}

type CodeGenRequest struct {
	Engine     string // "remotion" | "manim"
	Topic      string
	Storyboard string // JSON
	// System is the rendered *_engineer_ai prompt.
	System    string
	Model     string
	MaxTokens int
	// Illustrations are the approved library drawings the Remotion Engineer
	// may use; llm-service lists them in the prompt and pastes the used ones
	// into the script.
	Illustrations []LibraryDrawing
	// SubtitleBand is the strip burned-in subtitles cover (nil = nothing is
	// burned into the frame) and VideoFont the font text is drawn in ("" =
	// the default): the rendering layout check holds every drawn shot to
	// them. Remotion only.
	SubtitleBand *domain.SubtitleBand
	VideoFont    string
	// Frame is the canvas the video is built on; the merged script registers a
	// composition of its size and the storyboard's layout is checked against it.
	Frame domain.Frame
	// ChunkShots is the Creator's shots per segment.
	ChunkShots int
	// Done are the stored segments sent back so they are not written again.
	Done []DoneSegment
	// Only restricts the run to these segment keys; nil = every
	// missing one.
	Only []string
}

// DoneSegment is a stored segment as llm-service reads it back.
type DoneSegment struct {
	Key         string
	Fingerprint string
	Content     json.RawMessage
}

// CodeEvent is one event of a run. Progress types: "phase", "chunk_start",
// "chunk_done", "chunk_repair", "chunk_split". Segment types:
// "plan" (Plan), "segment_start", "segment_done" (Key, Fingerprint, Content,
// Source, Repaired, DurationMS), "segment_failed" (Key, Error, FailedShots,
// Fingerprint, Content), "call" (Call)
// and "check" (Check). Types a consumer does not use are ignored.
type CodeEvent struct {
	Type    string
	Phase   string // layout | cast | chunks | merge | check | repair
	Index   int
	Total   int
	Done    int
	Round   int
	Targets []string

	Plan        []domain.CodeSegment
	Key         string
	Fingerprint string
	Content     json.RawMessage
	Source      string
	Repaired    bool
	DurationMS  int
	ErrorKind   string
	ErrorText   string
	// FailedShots are the shots a "segment_failed" chunk could not write;
	// its Fingerprint and Content then carry the shots it did write.
	FailedShots []string
	Call        *CodeCall
	Check       *CodeCheck
}

// CodeCall is one model call inside a run, billed as its own llm_usage row.
type CodeCall struct {
	Phase        string // layout | cast | chunk | repair
	Label        string
	Segment      string
	OK           bool
	Usage        TokenUsage
	Duration     time.Duration
	ErrorKind    LLMErrorKind
	ErrorMessage string
}

// CheckKindLayout is the CodeDiagnostic.Kind of a layout issue found by
// drawing the shots; every other kind ("compile", "") fails the build itself.
const CheckKindLayout = "layout"

type CodeDiagnostic struct {
	Message string
	Line    int    // 0 = none
	Kind    string // "compile" | CheckKindLayout
	Rule    string
	// Shot and Segment the failing line belongs to ("" = none / the frame).
	Shot    string
	Segment string
}

// CodeCheck is one failed check of a run: the early check of one chunk
// (Phase "chunk") or the full-file check (Phase "final").
type CodeCheck struct {
	Phase       string
	Round       int
	Segment     string
	Diagnostics []CodeDiagnostic
}

// CodeGenStatus says whether a run reached the merge.
const (
	CodeGenDone       = "done"
	CodeGenIncomplete = "incomplete"
)

type CodeGenResult struct {
	Status         string
	Code           string
	CheckOK        bool
	Diagnostics    []CodeDiagnostic
	RepairRounds   int
	Warnings       []string
	SceneClassName string
	// Failed are the segments that failed in this run, Missing the ones
	// still without content (not run, or waiting for a failed frame).
	Failed  []string
	Missing []string
}

// ErrSegmentNotReady: a chunk's prompt needs a frame that is not written yet
// (or the frame comes from the storyboard and has nothing to write).
type ErrSegmentNotReady struct{ Message string }

func (e *ErrSegmentNotReady) Error() string { return e.Message }

// ErrSegmentReply: a pasted reply does not have the segment's shape.
type ErrSegmentReply struct{ Message string }

func (e *ErrSegmentReply) Error() string {
	return "kết quả dán vào không dùng được: " + e.Message
}

// ErrSegmentsUnsupported: the llm-service running has no /v2/code routes, so
// the code step cannot run.
var ErrSegmentsUnsupported = errors.New("llm-service đang chạy bản cũ, chưa có bước Code theo đoạn (/v2) — rebuild llm-service rồi chạy lại")

// ErrSegmentUnknown: no segment has this key in the current storyboard.
var ErrSegmentUnknown = errors.New("không có đoạn này trong storyboard hiện tại — tải lại danh sách đoạn")
