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
// the other beats as context. The Creator asked for it by name, so the beat
// stops being the Creator's.
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
	current, _, _, err := uc.beats(ctx, projectID)
	if err != nil {
		return domain.ReviewUnit{}, err
	}
	var old *storedBeat
	for i := range current {
		if current[i].unit.Key == key {
			old = &current[i]
		}
	}
	if old == nil {
		return domain.ReviewUnit{}, domain.ErrUnitNotFound
	}
	drafts, err := uc.run(ctx, projectID, bc, current, domain.BeatTaskRedo(key), note)
	if err != nil {
		return domain.ReviewUnit{}, err
	}
	if len(drafts) != 1 || (drafts[0].Key != "" && drafts[0].Key != key) {
		return domain.ReviewUnit{}, fmt.Errorf("%w: cần đúng một beat %s", domain.ErrBeatReplyUnusable, key)
	}
	sha, err := uc.units.UnitSHA(ctx, projectID, domain.UnitKindStory, domain.StoryUnitKey)
	if err != nil {
		return domain.ReviewUnit{}, err
	}
	unit := old.unit
	unit.Status, unit.CreatorFields, unit.RedoCount = domain.UnitPending, []string{}, unit.RedoCount+1
	unit.InputRef, unit.InputSHA = beatStoryRef, sha
	return uc.saveBeat(ctx, projectID, key, drafts[0].Beat, unit)
}

// Impact is empty: nothing is built on the beats yet.
func (beatKind) Impact(context.Context, string, string) ([]string, error) { return []string{}, nil }

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
