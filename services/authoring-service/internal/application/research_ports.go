package application

import (
	"context"
	"time"

	"authoring/internal/domain"
)

// ResearchFactStorePort is where the facts of the research step live (the
// project_research table), which the later steps read.
type ResearchFactStorePort interface {
	// ListResearchFacts returns the project's facts in code order.
	ListResearchFacts(ctx context.Context, projectID string) ([]domain.StoredFact, error)
	// GetResearchFact returns one fact, or domain.ErrUnitNotFound.
	GetResearchFact(ctx context.Context, projectID, factID string) (domain.StoredFact, error)
	// SaveResearchFact inserts a fact or replaces the one with the same code.
	SaveResearchFact(ctx context.Context, projectID string, fact domain.StoredFact) error
}

// ApprovedBriefPort reads the brief the Creator approved on the second step.
type ApprovedBriefPort interface {
	// ApprovedBrief returns the approved brief; ok is false when there is none.
	ApprovedBrief(ctx context.Context, projectID string) (brief domain.ProjectBrief, ok bool, err error)
}

// ResearchStorePort is what the research step reads and writes: its question
// and gate review units, its facts, the project's topic and model choice, and
// the log of pasted answers.
type ResearchStorePort interface {
	ReviewUnitPort
	ResearchFactStorePort
	GetAuthoringTopic(ctx context.Context, projectID string) (string, error)
	GetAuthoringModels(ctx context.Context, projectID string) (domain.AuthoringStepModels, error)
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
