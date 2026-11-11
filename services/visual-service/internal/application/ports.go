package application

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"visual/internal/domain"
)

// Errors of the AI path.
var (
	// ErrLLMNotConfigured: llm-service has no model to run the step on.
	ErrLLMNotConfigured = errors.New("chưa cấu hình AI ở llm-service")
	// ErrPromptNotFound: llm-service has no prompt for the role.
	ErrPromptNotFound = errors.New("llm-service chưa có prompt cho vai này")
	// ErrAuthoringFailed: authoring-service could not give the locked script.
	ErrAuthoringFailed = errors.New("authoring-service không trả được kịch bản chốt")
	// ErrRenderingFailed: rendering could not give its list of fonts.
	ErrRenderingFailed = errors.New("rendering không trả được danh sách font")
	// ErrLLMFailed: llm-service or the model behind it failed the call.
	ErrLLMFailed = errors.New("llm-service báo lỗi")
	// ErrUnknownModel: the model is not in llm-service's catalog.
	ErrUnknownModel = errors.New("model không có trong danh mục của llm-service")
)

// AuthoringInputPort reads what step 8 needs from authoring-service.
type AuthoringInputPort interface {
	// LatestScriptVersion returns the latest locked script version, or
	// domain.ErrScriptNotLocked when there is none.
	LatestScriptVersion(ctx context.Context, projectID string) (domain.ScriptVersion, error)
}

// ChatRequest is one model turn, as llm-service's /v1/chat takes it.
type ChatRequest struct {
	System     string
	User       string
	Model      string
	MaxTokens  int
	ProjectID  string
	Role       string
	Step       string
	ExpectJSON bool
}

// ChatResult is the answer of one turn; JSON is set when ExpectJSON was.
type ChatResult struct {
	Content string
	JSON    json.RawMessage
}

// LLMPort sends one turn to llm-service.
type LLMPort interface {
	Chat(ctx context.Context, req ChatRequest) (ChatResult, error)
}

// PromptLibraryPort reads the template a role runs on from llm-service.
type PromptLibraryPort interface {
	ActiveTemplate(ctx context.Context, role string) (string, error)
}

// ModelOption is one model of llm-service's catalog.
type ModelOption struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Local bool   `json:"local"`
}

// ModelCatalogPort reads the models and the step default from llm-service.
type ModelCatalogPort interface {
	Models(ctx context.Context) ([]ModelOption, error)
	StepDefaultModel(ctx context.Context, step string) (string, error)
}

// FontCatalogPort lists the fonts rendering can draw text in; it fails with
// ErrRenderingFailed when rendering does not answer.
type FontCatalogPort interface {
	Fonts(ctx context.Context) ([]string, error)
}

// VersionRecord is one locked version of a video's visual concept.
type VersionRecord struct {
	ProjectID     string          `json:"-"`
	Version       int             `json:"version"`
	ScriptVersion int             `json:"script_version"`
	Document      json.RawMessage `json:"document,omitempty"`
	Note          string          `json:"note"`
	LockedAt      time.Time       `json:"locked_at"`
}

// ReferenceReader reads the shared reference library and the descriptions a
// video picked. Step 8 only reads the library; adding and editing descriptions
// is the library's own.
type ReferenceReader interface {
	// PickedReferences returns the descriptions the video picked, in library order.
	PickedReferences(ctx context.Context, projectID string) ([]domain.ReferenceStyle, error)
	// ReferenceLibrary returns every description of the library.
	ReferenceLibrary(ctx context.Context) ([]domain.ReferenceStyle, error)
}

// ConceptStorePort keeps step 8's data in visual-db.
type ConceptStorePort interface {
	ListUnits(ctx context.Context, projectID, kind string) ([]domain.ReviewUnit, error)
	SaveUnit(ctx context.Context, u domain.ReviewUnit) error
	ConceptModel(ctx context.Context, projectID string) (string, error)
	SetConceptModel(ctx context.Context, projectID, model string) error
	// SetPicks replaces the video's picks; an id not in the library is
	// domain.ErrUnknownReference and nothing is written.
	SetPicks(ctx context.Context, projectID string, refIDs []string) error
	// Versions lists the locked versions without their documents, oldest first.
	Versions(ctx context.Context, projectID string) ([]VersionRecord, error)
	// Version returns one locked version, or domain.ErrNoVersion.
	Version(ctx context.Context, projectID string, version int) (VersionRecord, error)
	// InsertVersion adds a version; a taken number is domain.ErrVersionExists.
	InsertVersion(ctx context.Context, v VersionRecord) error
}
