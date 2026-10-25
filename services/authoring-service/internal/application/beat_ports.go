package application

import (
	"context"

	"authoring/internal/domain"
)

// BeatSheetStorePort is where the beat sheet step keeps its beats (the
// project_beat_sheet and project_beat tables), which the script step reads.
type BeatSheetStorePort interface {
	// GetBeatSheet returns the project's sheet, or domain.ErrBeatSheetNotFound.
	GetBeatSheet(ctx context.Context, projectID string) (domain.StoredBeatSheet, error)
	// SaveBeatSheet inserts the sheet or replaces the project's, all of it at once.
	SaveBeatSheet(ctx context.Context, sheet domain.StoredBeatSheet) error
}
