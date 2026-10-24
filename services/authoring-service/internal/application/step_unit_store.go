package application

import (
	"context"

	"authoring/internal/domain"
)

// StepUnitStorePort is what a step built on review units reads and writes: its
// units, the project's topic and length, and the model choices.
type StepUnitStorePort interface {
	ReviewUnitPort
	GetAuthoringTopic(ctx context.Context, projectID string) (string, error)
	GetAuthoringModels(ctx context.Context, projectID string) (domain.AuthoringStepModels, error)
	GetTargetDuration(ctx context.Context, projectID string) (domain.TargetDuration, error)
}
