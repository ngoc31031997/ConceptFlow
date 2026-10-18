package application

import (
	"context"
	"time"

	"authoring/internal/domain"
)

// ResearchStorePort is what the research step reads and writes: its review
// units, the project's topic, length and model choice, and the log of pasted
// answers.
type ResearchStorePort interface {
	ReviewUnitPort
	GetAuthoringTopic(ctx context.Context, projectID string) (string, error)
	GetAuthoringModels(ctx context.Context, projectID string) (domain.AuthoringStepModels, error)
	GetTargetDuration(ctx context.Context, projectID string) (domain.TargetDuration, error)
	AddPaste(ctx context.Context, projectID string, p PasteLog) (PasteLog, error)
	ListPastes(ctx context.Context, projectID string) ([]PasteLog, error)
}

// PasteLog is one answer the Creator pasted into the research step: when, and
// what was read from it. The pasted text itself is not kept.
type PasteLog struct {
	// N numbers the pastes of a project from 1.
	N        int                    `json:"n"`
	At       time.Time              `json:"at"`
	Added    []string               `json:"added"`
	Updated  []string               `json:"updated"`
	Rejected []domain.PasteRejected `json:"rejected"`
}
