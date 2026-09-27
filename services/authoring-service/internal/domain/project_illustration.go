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
