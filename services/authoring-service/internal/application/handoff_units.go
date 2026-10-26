package application

import (
	"context"
	"errors"
	"fmt"

	"authoring/internal/domain"
)

// handoffKind makes each beat's handoff note editable, redoable and
// approvable through the generic unit API (UnitKind for domain.UnitKindHandoffBeat).
type handoffKind struct{ desk *ScriptLockDeskUseCase }

// HandoffUnitKind returns the UnitKind of the beats' handoff notes.
func (uc *ScriptLockDeskUseCase) HandoffUnitKind() UnitKind { return handoffKind{desk: uc} }

// Edit stores the Creator's hand edit of sections of a note. A field is a
// section key (must, terms, exact, free, unsure) with one item per line; an
// empty value empties the section. The edited sections become the Creator's.
// A note that does not exist yet may be written by hand.
func (k handoffKind) Edit(ctx context.Context, projectID, key string, fields map[string]string) (domain.ReviewUnit, error) {
	if !domain.IsBeatKey(key) {
		return domain.ReviewUnit{}, domain.ErrUnitNotFound
	}
	if len(fields) == 0 {
		return domain.ReviewUnit{}, fmt.Errorf("%w: không có gì để sửa", domain.ErrInvalidUnitEdit)
	}
	uc := k.desk
	beats, _, err := uc.beats.beats(ctx, projectID)
	if err != nil {
		return domain.ReviewUnit{}, err
	}
	if beatIndex(beats, key) < 0 {
		return domain.ReviewUnit{}, domain.ErrUnitNotFound
	}
	note := domain.HandoffNote{}.Normalized()
	unit := domain.ReviewUnit{Status: domain.UnitPending, CreatorFields: []string{}}
	cur, err := uc.units.GetUnit(ctx, projectID, domain.UnitKindHandoffBeat, key)
	switch {
	case errors.Is(err, domain.ErrUnitNotFound):
	case err != nil:
		return domain.ReviewUnit{}, err
	default:
		unit = cur
		if note, err = readHandoff(cur); err != nil {
			return domain.ReviewUnit{}, err
		}
	}
	for _, field := range sortedKeys(fields) {
		if !domain.IsHandoffSection(field) {
			return domain.ReviewUnit{}, fmt.Errorf("%w: ghi chú bàn giao không có mục %q", domain.ErrInvalidUnitEdit, field)
		}
		note = note.With(field, domain.SplitItems(fields[field]))
		if !unit.HasCreatorField(field) {
			unit.CreatorFields = append(unit.CreatorFields, field)
		}
	}
	if err := note.Validate(); err != nil {
		return domain.ReviewUnit{}, fmt.Errorf("%w: %v", domain.ErrInvalidUnitEdit, err)
	}
	unit.Status = domain.UnitPending
	return uc.saveHandoff(ctx, projectID, key, note, unit)
}

// Redo has the AI write the whole note of this beat again with the Creator's
// note. The Creator asked for it by name, so the note stops being the Creator's.
func (k handoffKind) Redo(ctx context.Context, projectID, key, note string) (domain.ReviewUnit, error) {
	if !domain.IsBeatKey(key) {
		return domain.ReviewUnit{}, domain.ErrUnitNotFound
	}
	uc := k.desk
	if err := uc.requireAI(ctx); err != nil {
		return domain.ReviewUnit{}, err
	}
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return domain.ReviewUnit{}, err
	}
	lc, err := uc.context(ctx, projectID)
	if err != nil {
		return domain.ReviewUnit{}, err
	}
	at := beatIndex(lc.beats, key)
	if at < 0 {
		return domain.ReviewUnit{}, domain.ErrUnitNotFound
	}
	notes, err := uc.handoffs(ctx, projectID)
	if err != nil {
		return domain.ReviewUnit{}, err
	}
	var old *storedHandoff
	if n, ok := notes[key]; ok {
		old = &n
	}
	if err := uc.writeHandoff(ctx, projectID, lc, at, note, old); err != nil {
		return domain.ReviewUnit{}, err
	}
	return uc.units.GetUnit(ctx, projectID, domain.UnitKindHandoffBeat, key)
}

// Impact says what changing a note sends back: nothing downstream reads it
// until a version is locked.
func (k handoffKind) Impact(ctx context.Context, projectID, _ string) ([]string, error) {
	v, err := k.desk.versions.LatestScriptVersion(ctx, projectID)
	if errors.Is(err, domain.ErrNoLockedVersion) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}
	return []string{fmt.Sprintf("Tạo bản nháp v%d; các bước sau vẫn dùng v%d tới khi bạn chốt bản mới", v.Version+1, v.Version)}, nil
}

// CheckApprovable refuses a note with no item.
func (handoffKind) CheckApprovable(_ context.Context, unit domain.ReviewUnit) error {
	n, err := readHandoff(unit)
	if err != nil {
		return err
	}
	if n.Empty() {
		return domain.ErrHandoffEmpty
	}
	return nil
}
