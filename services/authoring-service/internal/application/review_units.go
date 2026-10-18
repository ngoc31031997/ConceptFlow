package application

import (
	"context"
	"errors"
	"fmt"

	"authoring/internal/domain"
)

// ReviewUnitPort stores review units. GetUnit returns domain.ErrUnitNotFound
// when the project has no unit of that kind and key.
type ReviewUnitPort interface {
	GetUnit(ctx context.Context, projectID, kind, key string) (domain.ReviewUnit, error)
	SaveUnit(ctx context.Context, unit domain.ReviewUnit) (domain.ReviewUnit, error)
	// ListUnits returns every unit of a kind, in key order (Q2 before Q10).
	ListUnits(ctx context.Context, projectID, kind string) ([]domain.ReviewUnit, error)
	// DeleteUnit removes a unit; a unit that is not there is not an error.
	DeleteUnit(ctx context.Context, projectID, kind, key string) error
	// UnitSHA is the hash of a unit's content, the form a built unit records
	// as the input it was built from; "" when the unit does not exist.
	UnitSHA(ctx context.Context, projectID, kind, key string) (string, error)
}

// UnitKind is what one step registers for its kind of review unit: how a
// manual edit is applied, how the AI redoes a unit with the Creator's note,
// what a change would send back in later steps, and when a unit is complete
// enough to approve. See ADR-0035.
type UnitKind interface {
	Edit(ctx context.Context, projectID, key string, fields map[string]string) (domain.ReviewUnit, error)
	Redo(ctx context.Context, projectID, key, note string) (domain.ReviewUnit, error)
	Impact(ctx context.Context, projectID, key string) ([]string, error)
	CheckApprovable(ctx context.Context, unit domain.ReviewUnit) error
}

// UnitInitializer is implemented by a kind whose unit exists from the start of
// its step even before anything was saved, such as the gate of a step. It
// returns the unit a project has before its first save; false when the key is
// not one it knows.
type UnitInitializer interface {
	Initial(ctx context.Context, projectID, key string) (domain.ReviewUnit, bool)
}

// ReviewUnitsUseCase is the one entry point for reading, editing, redoing and
// approving any review unit, whatever step registered its kind. Every change
// requires the project to still be editable, like every other authoring save.
type ReviewUnitsUseCase struct {
	store ReviewUnitPort
	locks AuthoringLockPort
	kinds map[string]UnitKind
}

// NewReviewUnitsUseCase builds the use case with no kinds registered.
func NewReviewUnitsUseCase(store ReviewUnitPort, locks AuthoringLockPort) *ReviewUnitsUseCase {
	return &ReviewUnitsUseCase{store: store, locks: locks, kinds: map[string]UnitKind{}}
}

// Register makes a kind of unit editable through this use case.
func (uc *ReviewUnitsUseCase) Register(kind string, k UnitKind) *ReviewUnitsUseCase {
	uc.kinds[kind] = k
	return uc
}

func (uc *ReviewUnitsUseCase) kind(kind string) (UnitKind, error) {
	k, ok := uc.kinds[kind]
	if !ok {
		return nil, fmt.Errorf("%w: %q", domain.ErrUnknownUnitKind, kind)
	}
	return k, nil
}

// Get returns one unit.
func (uc *ReviewUnitsUseCase) Get(ctx context.Context, projectID, kind, key string) (domain.ReviewUnit, error) {
	if _, err := uc.kind(kind); err != nil {
		return domain.ReviewUnit{}, err
	}
	return uc.store.GetUnit(ctx, projectID, kind, key)
}

// Edit applies the Creator's hand edit; the edited fields become the
// Creator's and an AI redo keeps them.
func (uc *ReviewUnitsUseCase) Edit(ctx context.Context, projectID, kind, key string, fields map[string]string) (domain.ReviewUnit, error) {
	k, err := uc.kind(kind)
	if err != nil {
		return domain.ReviewUnit{}, err
	}
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return domain.ReviewUnit{}, err
	}
	return k.Edit(ctx, projectID, key, fields)
}

// Redo has the AI write the unit again with the Creator's note.
func (uc *ReviewUnitsUseCase) Redo(ctx context.Context, projectID, kind, key, note string) (domain.ReviewUnit, error) {
	k, err := uc.kind(kind)
	if err != nil {
		return domain.ReviewUnit{}, err
	}
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return domain.ReviewUnit{}, err
	}
	return k.Redo(ctx, projectID, key, note)
}

// Impact lists, in the Creator's words, what changing the unit sends back in
// later steps; empty when nothing depends on it yet.
func (uc *ReviewUnitsUseCase) Impact(ctx context.Context, projectID, kind, key string) ([]string, error) {
	k, err := uc.kind(kind)
	if err != nil {
		return nil, err
	}
	return k.Impact(ctx, projectID, key)
}

// Approve marks the unit approved once its kind finds it complete.
func (uc *ReviewUnitsUseCase) Approve(ctx context.Context, projectID, kind, key string) (domain.ReviewUnit, error) {
	return uc.setStatus(ctx, projectID, kind, key, domain.UnitApproved)
}

// Unapprove sends an approved unit back to review.
func (uc *ReviewUnitsUseCase) Unapprove(ctx context.Context, projectID, kind, key string) (domain.ReviewUnit, error) {
	return uc.setStatus(ctx, projectID, kind, key, domain.UnitPending)
}

func (uc *ReviewUnitsUseCase) setStatus(ctx context.Context, projectID, kind, key string, status domain.UnitStatus) (domain.ReviewUnit, error) {
	k, err := uc.kind(kind)
	if err != nil {
		return domain.ReviewUnit{}, err
	}
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return domain.ReviewUnit{}, err
	}
	unit, err := uc.store.GetUnit(ctx, projectID, kind, key)
	if errors.Is(err, domain.ErrUnitNotFound) {
		if init, ok := k.(UnitInitializer); ok {
			if initial, found := init.Initial(ctx, projectID, key); found {
				unit, err = initial, nil
			}
		}
	}
	if err != nil {
		return domain.ReviewUnit{}, err
	}
	if status == domain.UnitApproved {
		if err := k.CheckApprovable(ctx, unit); err != nil {
			return domain.ReviewUnit{}, err
		}
	}
	unit.Status = status
	return uc.store.SaveUnit(ctx, unit)
}
