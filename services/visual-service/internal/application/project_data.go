// Package application holds visual-service's use cases.
package application

import (
	"context"
	"errors"
	"log/slog"
	"strings"
)

// ErrNoProjectID is returned when a use case is called without a project.
var ErrNoProjectID = errors.New("thiếu mã dự án")

// ProjectDataPort removes everything visual-service holds for one project.
// The reference library is shared by every video and is not touched.
type ProjectDataPort interface {
	DeleteProjectData(ctx context.Context, projectID string) error
}

// DeleteProjectUseCase is visual-service's part of deleting a video: the
// orchestrator calls it once every service holding the project's files has
// removed them (ADR-0049).
type DeleteProjectUseCase struct {
	store ProjectDataPort
	// purge and log are set by WithStoryboardPurge.
	purge StoryboardPurgePort
	log   *slog.Logger
}

// StoryboardPurgePort removes the assets, frames and versions of step 11 that
// rendering holds for a deleted video.
type StoryboardPurgePort interface {
	PurgeStoryboard(ctx context.Context, projectID string) error
}

// WithStoryboardPurge makes deleting a video also ask rendering to remove its
// step 11 data. A failure there is logged and does not fail the delete: the
// video's data here is already gone and the delete saga must be able to finish.
func (uc *DeleteProjectUseCase) WithStoryboardPurge(purge StoryboardPurgePort, log *slog.Logger) *DeleteProjectUseCase {
	uc.purge, uc.log = purge, log
	return uc
}

// NewDeleteProjectUseCase returns the use case over store.
func NewDeleteProjectUseCase(store ProjectDataPort) *DeleteProjectUseCase {
	return &DeleteProjectUseCase{store: store}
}

// Execute removes the project's data; a project with none is not an error.
func (uc *DeleteProjectUseCase) Execute(ctx context.Context, projectID string) error {
	if strings.TrimSpace(projectID) == "" {
		return ErrNoProjectID
	}
	if err := uc.store.DeleteProjectData(ctx, projectID); err != nil {
		return err
	}
	if uc.purge != nil {
		if err := uc.purge.PurgeStoryboard(ctx, projectID); err != nil {
			uc.log.Error("could not remove the storyboard data at rendering", "project_id", projectID, "error", err.Error())
		}
	}
	return nil
}
