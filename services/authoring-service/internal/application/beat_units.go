package application

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"authoring/internal/domain"
)

// ErrBeatNoGoal is returned when a beat without a goal is approved.
var ErrBeatNoGoal = errors.New("beat chưa có mục tiêu")

// beatKind makes each beat editable, redoable and approvable through the
// generic unit API (UnitKind for domain.UnitKindBeat).
type beatKind struct{ desk *BeatDeskUseCase }

// BeatUnitKind returns the UnitKind of the beats.
func (uc *BeatDeskUseCase) BeatUnitKind() UnitKind { return beatKind{desk: uc} }

func requireBeatKey(key string) error {
	if !domain.IsBeatKey(key) {
		return domain.ErrUnitNotFound
	}
	return nil
}

// parseFactList reads "F01, F02" into fact codes, each once.
func parseFactList(text string) ([]string, error) {
	ids := []string{}
	for _, part := range strings.Split(text, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, ok := domain.NormalizeFactID(part)
		if !ok {
			return nil, fmt.Errorf("%q không phải mã dữ kiện", part)
		}
		ids = mergeFactCodes(ids, []string{id})
	}
	return ids, nil
}

// Edit stores the Creator's hand edit of a beat: any of title, goal, content,
// next (text), fn (a function of domain.BeatFns), dur (seconds) and facts
// ("F01, F02"). A field whose value changes becomes the Creator's; a text
// field emptied goes back to the AI. Editing sends the beat back to review.
func (k beatKind) Edit(ctx context.Context, projectID, key string, fields map[string]string) (domain.ReviewUnit, error) {
	if err := requireBeatKey(key); err != nil {
		return domain.ReviewUnit{}, err
	}
	if len(fields) == 0 {
		return domain.ReviewUnit{}, fmt.Errorf("%w: không có gì để sửa", domain.ErrInvalidUnitEdit)
	}
	unit, err := k.desk.units.GetUnit(ctx, projectID, domain.UnitKindBeat, key)
	if err != nil {
		return domain.ReviewUnit{}, err
	}
	beat, err := readBeat(unit)
	if err != nil {
		return domain.ReviewUnit{}, err
	}
	facts, err := loadKeptFacts(ctx, k.desk.units, projectID)
	if err != nil {
		return domain.ReviewUnit{}, err
	}
	mark := func(field string, changed bool) {
		if changed && !unit.HasCreatorField(field) {
			unit.CreatorFields = append(unit.CreatorFields, field)
		}
	}
	for field, raw := range fields {
		value := strings.TrimSpace(raw)
		switch field {
		case domain.BeatFieldFn:
			mark(field, value != beat.Fn)
			beat.Fn = value
		case domain.BeatFieldTitle:
			mark(field, value != beat.Title)
			beat.Title = value
		case domain.BeatFieldGoal, domain.BeatFieldContent, domain.BeatFieldNext:
			target := map[string]*string{domain.BeatFieldGoal: &beat.Goal, domain.BeatFieldContent: &beat.Content, domain.BeatFieldNext: &beat.Next}[field]
			if value == "" {
				unit.CreatorFields = withoutField(unit.CreatorFields, field)
			} else {
				mark(field, value != *target)
			}
			*target = value
		case domain.BeatFieldDur:
			secs, err := strconv.Atoi(value)
			if err != nil {
				return domain.ReviewUnit{}, fmt.Errorf("%w: thời lượng %q không phải số giây", domain.ErrInvalidUnitEdit, raw)
			}
			mark(field, secs != beat.Dur)
			beat.Dur = secs
		case domain.BeatFieldFacts:
			ids, err := parseFactList(value)
			if err != nil {
				return domain.ReviewUnit{}, fmt.Errorf("%w: %v", domain.ErrInvalidUnitEdit, err)
			}
			mark(field, strings.Join(ids, ",") != strings.Join(beat.Facts, ","))
			beat.Facts = ids
		default:
			return domain.ReviewUnit{}, fmt.Errorf("%w: beat không có trường %q để sửa", domain.ErrInvalidUnitEdit, field)
		}
	}
	if err := beat.Validate(factKnownTo(facts)); err != nil {
		return domain.ReviewUnit{}, fmt.Errorf("%w: %v", domain.ErrInvalidUnitEdit, err)
	}
	unit.Status = domain.UnitPending
	return k.desk.saveBeat(ctx, projectID, key, beat, unit)
}

// Redo has the AI write this one beat again with the Creator's note, reading
// the other beats as context and the time budget they leave. It serves a beat
// of the skeleton that has no content yet as well as one already written.
// Function and facts stay as they are; the Creator asked for this beat by
// name, so the wording that was the Creator's is released while a length the
// Creator set is kept. When this beat was the last one without content the
// lengths of the sheet are brought inside the target.
func (k beatKind) Redo(ctx context.Context, projectID, key, note string) (domain.ReviewUnit, error) {
	if err := requireBeatKey(key); err != nil {
		return domain.ReviewUnit{}, err
	}
	uc := k.desk
	if err := uc.requireAI(); err != nil {
		return domain.ReviewUnit{}, err
	}
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return domain.ReviewUnit{}, err
	}
	bc, err := uc.context(ctx, projectID)
	if err != nil {
		return domain.ReviewUnit{}, err
	}
	work, gate, sheetUnit, err := uc.beats(ctx, projectID)
	if err != nil {
		return domain.ReviewUnit{}, err
	}
	if !gate.SkeletonApproved {
		return domain.ReviewUnit{}, fmt.Errorf("%w: %w", domain.ErrInvalidStatus, ErrBeatSkeletonNeeded)
	}
	at := -1
	written := make([]bool, len(work))
	for i, b := range work {
		written[i] = b.Written()
		if b.unit.Key == key {
			at = i
		}
	}
	if at < 0 {
		return domain.ReviewUnit{}, domain.ErrUnitNotFound
	}
	written[at] = false
	kept := []string{}
	for _, field := range work[at].unit.CreatorFields {
		if field == domain.BeatFieldFn || field == domain.BeatFieldFacts || field == domain.BeatFieldDur {
			kept = append(kept, field)
		}
	}
	work[at].unit.CreatorFields = kept
	work[at].unit.RedoCount++
	beat, err := uc.writeOne(ctx, projectID, bc, work, written, at, note)
	if err != nil {
		return domain.ReviewUnit{}, err
	}
	work[at].Beat = beat
	rewritten := make([]bool, len(work))
	rewritten[at] = true
	if err := uc.saveContent(ctx, projectID, bc, work, rewritten, gate, sheetUnit); err != nil {
		return domain.ReviewUnit{}, err
	}
	return uc.units.GetUnit(ctx, projectID, domain.UnitKindBeat, key)
}

// Impact lists the dialogue written for the beat: editing the beat marks it
// as out of date for the Creator to review again.
func (k beatKind) Impact(ctx context.Context, projectID, key string) ([]string, error) {
	items := []string{}
	if err := requireBeatKey(key); err != nil {
		return items, err
	}
	unit, err := k.desk.units.GetUnit(ctx, projectID, domain.UnitKindScriptBeat, key)
	if errors.Is(err, domain.ErrUnitNotFound) {
		return items, nil
	}
	if err != nil {
		return nil, err
	}
	script, err := readScriptBeat(unit)
	if err != nil {
		return nil, err
	}
	if len(script.Lines) > 0 {
		items = append(items, fmt.Sprintf("Viết kịch bản: lời thoại của beat này (%d câu) cần xem lại", len(script.Lines)))
	}
	return items, nil
}

// CheckApprovable refuses a beat without a goal.
func (beatKind) CheckApprovable(_ context.Context, unit domain.ReviewUnit) error {
	beat, err := readBeat(unit)
	if err != nil {
		return err
	}
	if strings.TrimSpace(beat.Goal) == "" {
		return fmt.Errorf("%w: viết mục tiêu của beat trước khi duyệt", ErrBeatNoGoal)
	}
	return nil
}
