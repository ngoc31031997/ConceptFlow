package domain

// ProjectIllustrationState is where one planned drawing of a video stands (CR-044).
type ProjectIllustrationState string

const (
	PIPlanned ProjectIllustrationState = "planned" // needs drawing
	PIDrawing ProjectIllustrationState = "drawing"
	PIDrawn   ProjectIllustrationState = "drawn"  // drawn; its illustration's own status says if approved
	PIReused  ProjectIllustrationState = "reused" // an existing library drawing
	PIFailed  ProjectIllustrationState = "failed"
	PISkipped ProjectIllustrationState = "skipped" // the Creator chose to go on without it
)

// ProjectIllustration is one drawing a video needs: planned from its
// storyboard, then drawn or matched to the library, and reviewed before the
// code step may use it.
type ProjectIllustration struct {
	ID             string                   `json:"id"`
	ProjectID      string                   `json:"project_id"`
	Position       int                      `json:"position"`
	Name           string                   `json:"name"` // proposed or reused component name
	Description    string                   `json:"description"`
	FolderID       string                   `json:"folder_id"`
	Shots          []string                 `json:"shots"`
	State          ProjectIllustrationState `json:"state"`
	Error          string                   `json:"error,omitempty"`
	IllustrationID string                   `json:"illustration_id,omitempty"`
	Illustration   *Illustration            `json:"illustration,omitempty"`
	// Progress is the live state of a drawing in flight (CR-045); in memory
	// only, never stored — nil when the row is not being drawn right now.
	Progress *DrawProgress `json:"progress,omitempty"`
}

// DrawProgress is how far the AI drawer has got on one drawing (CR-045): which
// attempt of how many, and what it is doing in that attempt.
type DrawProgress struct {
	Attempt     int `json:"attempt"`
	MaxAttempts int `json:"max_attempts"`
	// Phase: "waiting" (call sent) | "reasoning" | "writing" | "checking"
	// (the renderer checks the code and draws the preview).
	Phase          string `json:"phase"`
	ReasoningChars int    `json:"reasoning_chars"`
	ContentChars   int    `json:"content_chars"`
	ElapsedSeconds int    `json:"elapsed_seconds"`
}

// Ready reports whether the code step may go ahead past this row.
func (p ProjectIllustration) Ready() bool {
	switch p.State {
	case PISkipped:
		return true
	case PIReused, PIDrawn:
		return p.Illustration != nil && (p.Illustration.Builtin || p.Illustration.Status == IllustrationApproved)
	}
	return false
}
