package application

import (
	"context"
	"errors"

	"visual/internal/domain"
)

// ErrReferenceStyleNotFound is returned when no description has the given id.
var ErrReferenceStyleNotFound = errors.New("không có mô tả tham chiếu này")

// ErrNoReferencePicked is returned when a video has picked no description.
var ErrNoReferencePicked = errors.New("dự án chưa chọn phong cách tham chiếu")

// ErrReferenceKitMissing is returned when the style has no kit to draw with.
var ErrReferenceKitMissing = errors.New("phong cách tham chiếu chưa có bộ thông số vẽ")

// ReferenceStylePort stores the shared reference library. Create assigns the
// next id from a sequence that never reuses a number; Update and Delete return
// ErrReferenceStyleNotFound for an unknown id.
type ReferenceStylePort interface {
	ListReferenceStyles(ctx context.Context) ([]domain.ReferenceStyle, error)
	CreateReferenceStyle(ctx context.Context, in domain.ReferenceStyleInput) (domain.ReferenceStyle, error)
	UpdateReferenceStyle(ctx context.Context, id string, in domain.ReferenceStyleInput) (domain.ReferenceStyle, error)
	DeleteReferenceStyle(ctx context.Context, id string) error
	// ReferenceStyleUsers returns the projects that picked the description,
	// or ErrReferenceStyleNotFound when it does not exist.
	ReferenceStyleUsers(ctx context.Context, id string) ([]string, error)
	// ReferenceStyleKit returns the kit of id, nil when it has none, or
	// ErrReferenceStyleNotFound.
	ReferenceStyleKit(ctx context.Context, id string) (*domain.ReferenceKit, error)
	// ProjectKit returns the id and kit (nil when none) of the style the
	// video picked, or ErrNoReferencePicked.
	ProjectKit(ctx context.Context, projectID string) (string, *domain.ReferenceKit, error)
}

// ReferenceKitResult is a style's kit with the style it belongs to.
type ReferenceKitResult struct {
	RefID string              `json:"ref_id"`
	Kit   domain.ReferenceKit `json:"kit"`
}

// ReferenceLibraryUseCase manages the reference library shared by every video.
// Editing or deleting a description changes what the picking videos read; the
// Creator is warned about them through UsedBy before confirming.
type ReferenceLibraryUseCase struct {
	store ReferenceStylePort
}

// NewReferenceLibraryUseCase returns the use case over store.
func NewReferenceLibraryUseCase(store ReferenceStylePort) *ReferenceLibraryUseCase {
	return &ReferenceLibraryUseCase{store: store}
}

// List returns every description in the order they were added.
func (uc *ReferenceLibraryUseCase) List(ctx context.Context) ([]domain.ReferenceStyle, error) {
	return uc.store.ListReferenceStyles(ctx)
}

// Create checks and stores a new description.
func (uc *ReferenceLibraryUseCase) Create(ctx context.Context, in domain.ReferenceStyleInput) (domain.ReferenceStyle, error) {
	clean, err := in.Normalize()
	if err != nil {
		return domain.ReferenceStyle{}, err
	}
	return uc.store.CreateReferenceStyle(ctx, clean)
}

// Update checks and replaces every field of id.
func (uc *ReferenceLibraryUseCase) Update(ctx context.Context, id string, in domain.ReferenceStyleInput) (domain.ReferenceStyle, error) {
	clean, err := in.Normalize()
	if err != nil {
		return domain.ReferenceStyle{}, err
	}
	return uc.store.UpdateReferenceStyle(ctx, id, clean)
}

// Delete removes id; the videos' picks of it go with it.
func (uc *ReferenceLibraryUseCase) Delete(ctx context.Context, id string) error {
	return uc.store.DeleteReferenceStyle(ctx, id)
}

// UsedBy returns the ids of the projects that picked id.
func (uc *ReferenceLibraryUseCase) UsedBy(ctx context.Context, id string) ([]string, error) {
	return uc.store.ReferenceStyleUsers(ctx, id)
}

// Validate checks a pasted description and returns it cleaned, without storing
// it, so the Creator can review it before saving.
func (uc *ReferenceLibraryUseCase) Validate(in domain.ReferenceStyleInput) (domain.ReferenceStyleInput, error) {
	return in.Normalize()
}

// KitOf returns the kit of style id; ErrReferenceKitMissing when it has none.
func (uc *ReferenceLibraryUseCase) KitOf(ctx context.Context, id string) (ReferenceKitResult, error) {
	kit, err := uc.store.ReferenceStyleKit(ctx, id)
	if err != nil {
		return ReferenceKitResult{}, err
	}
	if kit == nil {
		return ReferenceKitResult{}, ErrReferenceKitMissing
	}
	return ReferenceKitResult{RefID: id, Kit: *kit}, nil
}

// KitOfProject returns the kit of the style the video picked:
// ErrNoReferencePicked when it picked none, ErrReferenceKitMissing when that
// style has no kit.
func (uc *ReferenceLibraryUseCase) KitOfProject(ctx context.Context, projectID string) (ReferenceKitResult, error) {
	id, kit, err := uc.store.ProjectKit(ctx, projectID)
	if err != nil {
		return ReferenceKitResult{}, err
	}
	if kit == nil {
		return ReferenceKitResult{}, ErrReferenceKitMissing
	}
	return ReferenceKitResult{RefID: id, Kit: *kit}, nil
}
