// Package application holds visual-service's use cases.
package application

import (
	"context"
	"errors"
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
// removed them (ADR-0048).
type DeleteProjectUseCase struct {
	store ProjectDataPort
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
	return uc.store.DeleteProjectData(ctx, projectID)
}
