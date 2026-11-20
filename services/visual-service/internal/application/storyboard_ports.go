package application

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"visual/internal/domain"
)

// StoryboardStep is the step of the prompt library the prompts of step 11 belong to.
const StoryboardStep = "visual_storyboard"

// StoryboardStorePort keeps step 11's review units in visual-db and reads the
// locked visual concept, plan and sequence it is written from.
type StoryboardStorePort interface {
	ListUnits(ctx context.Context, projectID, kind string) ([]domain.ReviewUnit, error)
	SaveUnit(ctx context.Context, u domain.ReviewUnit) error
	// LatestConceptVersion returns the latest locked visual concept, or domain.ErrNoVersion.
	LatestConceptVersion(ctx context.Context, projectID string) (VersionRecord, error)
	// LatestPlanVersion returns the latest locked visual plan, or domain.ErrNoVersion.
	LatestPlanVersion(ctx context.Context, projectID string) (PlanVersionRecord, error)
	// SequenceVersions lists the locked sequence versions without documents, oldest first.
	SequenceVersions(ctx context.Context, projectID string) ([]SequenceVersionRecord, error)
	// SequenceVersion returns one locked sequence version with its document.
	SequenceVersion(ctx context.Context, projectID string, version int) (SequenceVersionRecord, error)
	// StoryboardModel returns the video's model of step 11; "" when none is saved.
	StoryboardModel(ctx context.Context, projectID string) (string, error)
	// SetStoryboardModel saves the video's model of step 11.
	SetStoryboardModel(ctx context.Context, projectID, model string) error
}

// ProjectKitPort reads the drawing kit of the reference style a video picked at
// step 8; ReferenceLibraryUseCase is one. A video with no style or a style with
// no kit is ErrNoReferencePicked or ErrReferenceKitMissing.
type ProjectKitPort interface {
	KitOfProject(ctx context.Context, projectID string) (ReferenceKitResult, error)
}

// AssetWarning is one finding of rendering's builder on an asset's code.
type AssetWarning struct {
	Message string `json:"message"`
	Line    *int   `json:"line"`
}

// ProjectAsset is a project asset as rendering holds it. The embedded asset is
// what visual-service reads to check shots; the rest only reaches the screen.
type ProjectAsset struct {
	domain.StoryboardAsset
	ToLibrary             bool           `json:"to_library"`
	LibraryFolderID       string         `json:"library_folder_id"`
	LibraryIllustrationID string         `json:"library_illustration_id"`
	Warnings              []AssetWarning `json:"warnings"`
}

// AssetInput is the body that creates or replaces an asset at rendering.
type AssetInput struct {
	ConceptID          string   `json:"concept_id"`
	Name               string   `json:"name"`
	Kind               string   `json:"kind"`
	Tier               string   `json:"tier"`
	Portrayal          string   `json:"portrayal"`
	MustNotImply       []string `json:"must_not_imply"`
	PaletteTokens      []string `json:"palette_tokens"`
	Expressions        []string `json:"expressions"`
	BaseIllustrationID string   `json:"base_illustration_id"`
	Units              []string `json:"units"`
	ToLibrary          bool     `json:"to_library"`
	LibraryFolderID    string   `json:"library_folder_id"`
}

// AssetInputOf is the input that leaves a as it is.
func AssetInputOf(a ProjectAsset) AssetInput {
	return AssetInput{ConceptID: a.ConceptID, Name: a.Name, Kind: a.Kind, Tier: a.Tier, Portrayal: a.Portrayal,
		MustNotImply: nonNilList(a.MustNotImply), PaletteTokens: nonNilList(a.PaletteTokens), Expressions: nonNilList(a.Expressions),
		BaseIllustrationID: a.BaseIllustrationID, Units: nonNilList(a.Units), ToLibrary: a.ToLibrary, LibraryFolderID: a.LibraryFolderID}
}

func nonNilList(l []string) []string {
	if l == nil {
		return []string{}
	}
	return l
}

// AssetDrawRequest is what rendering draws or redraws an asset from: the video's
// kit and the reference style it belongs to. Note only counts on a redraw.
type AssetDrawRequest struct {
	Kit   json.RawMessage `json:"kit"`
	RefID string          `json:"ref_id"`
	Note  string          `json:"note,omitempty"`
}

// AssetApproveRequest names the reference style an approved asset is added to
// the library under.
type AssetApproveRequest struct {
	RefID       string `json:"ref_id,omitempty"`
	ProjectName string `json:"project_name,omitempty"`
}

// LibraryDrawing is one approved drawing of the illustration library, what the
// asset writer may take as the base of an asset.
type LibraryDrawing struct {
	ID          string   `json:"id"`
	Kind        string   `json:"kind"`
	Name        string   `json:"name"`
	Title       string   `json:"title"`
	Tags        []string `json:"tags"`
	Description string   `json:"description"`
	DetailLevel string   `json:"detail_level"`
}

// FrameRequest is what rendering draws one shot's frame from.
type FrameRequest struct {
	domain.FrameSpec
	UnitID   string `json:"unit_id"`
	BeatID   string `json:"beat_id"`
	SpecHash string `json:"spec_hash"`
}

// FrameResult is rendering's record of a drawn frame.
type FrameResult struct {
	FramePath string `json:"frame_path"`
	FrameHash string `json:"frame_hash"`
	SpecHash  string `json:"spec_hash"`
	Error     string `json:"error"`
	CacheHit  bool   `json:"cache_hit"`
}

// StoryboardVersionRecord is one locked storyboard as rendering keeps it.
type StoryboardVersionRecord struct {
	ProjectID       string          `json:"project_id"`
	Version         int             `json:"version"`
	SequenceVersion int             `json:"sequence_version"`
	Note            string          `json:"note"`
	LockedAt        time.Time       `json:"locked_at"`
	Document        json.RawMessage `json:"document"`
}

// RenderingError is a refusal of rendering that carries its HTTP status, so the
// caller can pass the reason on unchanged. Extra holds the body's other keys
// (the assets that are not ready, the frames that are missing).
type RenderingError struct {
	Status  int
	Message string
	Extra   map[string]json.RawMessage
}

func (e *RenderingError) Error() string {
	return fmt.Sprintf("rendering trả %d: %s", e.Status, e.Message)
}

// StoryboardRenderingPort is everything step 11 asks of rendering. A refusal is
// a *RenderingError; a call that did not get an answer wraps ErrRenderingFailed.
type StoryboardRenderingPort interface {
	ListAssets(ctx context.Context, projectID string) ([]ProjectAsset, error)
	CreateAsset(ctx context.Context, projectID string, in AssetInput) (ProjectAsset, error)
	UpdateAsset(ctx context.Context, projectID, assetID string, in AssetInput) (ProjectAsset, error)
	DeleteAsset(ctx context.Context, projectID, assetID string) error
	DrawAsset(ctx context.Context, projectID, assetID string, req AssetDrawRequest) (ProjectAsset, error)
	RedrawAsset(ctx context.Context, projectID, assetID string, req AssetDrawRequest) (ProjectAsset, error)
	ApproveAsset(ctx context.Context, projectID, assetID string, req AssetApproveRequest) (ProjectAsset, error)
	UnapproveAsset(ctx context.Context, projectID, assetID string) (ProjectAsset, error)
	// AssetPreview returns the PNG of an asset's current drawing.
	AssetPreview(ctx context.Context, projectID, assetID string) ([]byte, error)
	// ApprovedLibrary lists the approved drawings of the illustration library.
	ApprovedLibrary(ctx context.Context) ([]LibraryDrawing, error)
	DrawFrame(ctx context.Context, projectID string, req FrameRequest) (FrameResult, error)
	// FrameImage returns the PNG of a shot's last drawn frame.
	FrameImage(ctx context.Context, projectID, shotID string) ([]byte, error)
	// LockStoryboard stores the locked document and returns its version.
	LockStoryboard(ctx context.Context, projectID string, document json.RawMessage, sequenceVersion int, note string) (StoryboardVersionRecord, error)
	// LatestStoryboard returns the latest locked storyboard, or domain.ErrNoVersion.
	LatestStoryboard(ctx context.Context, projectID string) (StoryboardVersionRecord, error)
	// StoryboardVersion returns one locked storyboard, or domain.ErrNoVersion.
	StoryboardVersion(ctx context.Context, projectID string, version int) (StoryboardVersionRecord, error)
	// PurgeStoryboard removes the video's assets, frames and versions at rendering.
	PurgeStoryboard(ctx context.Context, projectID string) error
}
