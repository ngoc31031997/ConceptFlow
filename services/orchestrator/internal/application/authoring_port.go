package application

import (
	"context"
	"time"

	"orchestrator/internal/domain"
)

// AuthoringSummary is what authoring-service reports per project: the topic to
// name it by and which of the three authoring outputs exist (CR-040 FR111).
type AuthoringSummary struct {
	Topic      string `json:"topic"`
	Story      bool   `json:"story"`
	Storyboard bool   `json:"storyboard"`
	Code       bool   `json:"code"`
	// IllustrationsReady (CR-051): the drawing list is planned and every drawing
	// approved or skipped. nil means an authoring-service from before CR-051
	// that does not send it — read as ready, which is the old placement.
	IllustrationsReady *bool `json:"illustrations_ready"`
}

// Content is the summary as FlowStateFor reads it. Only a Remotion project has
// the illustrations step (5).
func (s AuthoringSummary) Content(engine domain.RenderEngine) domain.AuthoredContent {
	return domain.AuthoredContent{
		Story: s.Story, Storyboard: s.Storyboard, Code: s.Code,
		NeedsIllustrations: engine == domain.RenderEngineRemotion,
		Illustrations:      s.IllustrationsReady == nil || *s.IllustrationsReady,
	}
}

// SimilarProject is one match CR-028 FR85 surfaces back to the Creator when a
// topic collides (after normalization) with an existing project's saved topic,
// in the same content_language.
type SimilarProject struct {
	ProjectID string               `json:"project_id"`
	Topic     string               `json:"topic"`
	Status    domain.ProjectStatus `json:"status"`
	CreatedAt time.Time            `json:"created_at"`
}

// AuthoringLookupPort is the slice of authoring-service the orchestrator's own
// use cases (draft, fork, list, delete) call. The artefacts themselves live in
// authoring-service's database.
type AuthoringLookupPort interface {
	SaveAuthoringTopic(ctx context.Context, projectID, topic string, language domain.ContentLanguage) error
	FindSimilarTopics(ctx context.Context, language domain.ContentLanguage, normalizedTopic, excludeProjectID string) ([]SimilarProject, error)
	Summaries(ctx context.Context, projectIDs []string) (map[string]AuthoringSummary, error)
	DeleteAuthoring(ctx context.Context, projectID string) error
}
