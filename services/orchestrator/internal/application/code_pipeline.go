package application

import (
	"context"
	"time"
)

// CR-039 — the AI flow's storyboard and code steps are no longer one model
// call each. These ports are how the orchestrator hands them to llm-service,
// which owns every conversation with a model.

// StoryboardFinalizerPort validates the Visual Director's JSON (and gets one
// model repair turn if it is invalid) and returns the canonical text to save.
type StoryboardFinalizerPort interface {
	FinalizeStoryboard(ctx context.Context, content, model string, maxTokens int) (FinalizedStoryboard, error)
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
// parallel → merge → compile check → repair.
type CodePipelinePort interface {
	GenerateCode(ctx context.Context, req CodeGenRequest, onEvent func(CodeEvent)) (CodeGenResult, error)
}

type CodeGenRequest struct {
	Engine     string // "remotion" | "manim"
	Topic      string
	Storyboard string // JSON
	// System is the rendered *_engineer_ai prompt.
	System    string
	Model     string
	MaxTokens int
}

// CodeEvent is one progress event of a run. Type is "phase", "chunk_start" or
// "chunk_done"; the other fields are filled as they apply.
type CodeEvent struct {
	Type    string
	Phase   string // layout | cast | chunks | merge | check | repair
	Index   int
	Total   int
	Done    int
	Round   int
	Targets []string
}

// CodeCall is one model call inside a run, kept so each is billed as its own
// llm_usage row.
type CodeCall struct {
	Phase        string // layout | cast | chunk | repair
	Label        string
	OK           bool
	Cached       bool
	Usage        TokenUsage
	Duration     time.Duration
	ErrorKind    LLMErrorKind
	ErrorMessage string
}

type CodeDiagnostic struct {
	Message string
	Line    int // 0 = none
}

type CodeGenResult struct {
	Code           string
	CheckOK        bool
	Diagnostics    []CodeDiagnostic
	RepairRounds   int
	Calls          []CodeCall
	Warnings       []string
	SceneClassName string
}
