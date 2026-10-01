package domain

import (
	"encoding/json"
	"fmt"
	"time"
)

// The code step is stored as segments (see ADR-0030): the shared
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
	SegmentSourceExternal   = "external" // pasted from an outside AI
	SegmentSourceManual     = "manual"   // edited by hand
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
// Creator changed it; MaxChunkShots bounds the setting.
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
	// FailedShots are the shots a failed chunk could not write; its Content
	// then holds the shots it did write. Empty for any other state.
	FailedShots []string  `json:"failed_shots"`
	DurationMS  int       `json:"duration_ms"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// SegmentFailure is what a run reports for a segment it could not finish.
type SegmentFailure struct {
	Kind       string
	Message    string
	DurationMS int
	// Fingerprint and Content are the shots the run did write, under the
	// segment's fingerprint; Content is nil when it wrote none.
	Fingerprint string
	Content     json.RawMessage
	FailedShots []string
}

// HasEveryShot reports whether a chunk's content ({"shots": {...}}) holds
// code for every one of its shots. Unreadable content holds none.
func HasEveryShot(content json.RawMessage, shots []string) bool {
	if len(content) == 0 || len(shots) == 0 {
		return false
	}
	var c struct {
		Shots map[string]string `json:"shots"`
	}
	if err := json.Unmarshal(content, &c); err != nil {
		return false
	}
	for _, id := range shots {
		if c.Shots[id] == "" {
			return false
		}
	}
	return true
}

// FailedContent is the fingerprint and content a segment keeps when a run
// fails it. Content that already holds every shot under the same
// fingerprint stays, since a failed re-run must not throw away a complete
// result; otherwise the shots the run did write replace what was there, and
// a run that wrote nothing leaves the stored content as it was.
func FailedContent(storedFingerprint string, stored json.RawMessage, shots []string, f SegmentFailure) (string, json.RawMessage) {
	if len(f.Content) == 0 || (storedFingerprint == f.Fingerprint && HasEveryShot(stored, shots)) {
		return storedFingerprint, stored
	}
	return f.Fingerprint, f.Content
}

// CheckDiagnosticRecord is one failed-check finding of a code run, kept for
// statistics.
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
