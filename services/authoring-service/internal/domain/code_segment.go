package domain

import (
	"encoding/json"
	"fmt"
	"time"
)

// CR-050 Unit 2 / ADR-0030 — the code step is stored as segments: the shared
// frame (LAYOUT for Remotion, cast for Manim) and one segment per chunk of
// shots. llm-service decides what each segment's fingerprint is; this service
// keeps the segments so a failed or interrupted run loses nothing already
// written, and the Creator can re-run or paste a single segment.

type SegmentStatus string

const (
	SegmentPending SegmentStatus = "pending"
	SegmentRunning SegmentStatus = "running"
	SegmentDone    SegmentStatus = "done"
	SegmentFailed  SegmentStatus = "failed"
)

// Where a segment's content came from.
const (
	SegmentSourceAI         = "ai"
	SegmentSourceExternal   = "external" // pasted from an outside AI (FR-5)
	SegmentSourceManual     = "manual"   // edited by hand (FR-5)
	SegmentSourceStoryboard = "storyboard"
)

// Segment kinds and the frame's key; a chunk's key is "<first shot>-<last shot>".
const (
	SegmentKindFrame = "frame"
	SegmentKindShots = "shots"
	FrameSegmentKey  = "frame"
)

// Why a segment that was running when its run ended is now failed (NFR-3).
const (
	SegmentErrInterrupted = "interrupted"
	SegmentErrCancelled   = "cancelled"
)

// DefaultChunkShots is how many shots one code segment holds unless the
// Creator changed it (FR-7, C3b); MaxChunkShots bounds the setting.
const (
	DefaultChunkShots = 3
	MaxChunkShots     = 10
)

var ErrInvalidChunkShots = fmt.Errorf("số shot mỗi đoạn phải từ 1 đến %d", MaxChunkShots)

// CodeSegment is one stored segment of a project's code step.
type CodeSegment struct {
	Key         string        `json:"key"`
	Kind        string        `json:"kind"`
	Position    int           `json:"position"`
	Shots       []string      `json:"shots"`
	Status      SegmentStatus `json:"status"`
	Source      string        `json:"source"`
	Fingerprint string        `json:"-"`
	// Content is {"code": "..."} for the frame, {"shots": {"1.1": "..."}} for a chunk.
	Content      json.RawMessage `json:"content,omitempty"`
	ErrorKind    string          `json:"error_kind"`
	ErrorMessage string          `json:"error_message"`
	DurationMS   int             `json:"duration_ms"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// CheckDiagnosticRecord is one failed-check finding of a code run, kept for
// statistics (CR-050 FR-22).
type CheckDiagnosticRecord struct {
	ProjectID  string
	Engine     string
	Phase      string // "chunk" (early per-chunk check) | "final"
	Round      int
	SegmentKey string
	ShotID     string
	Kind       string // compile | layout
	Rule       string
	Message    string
	Line       int // 0 = none
}
