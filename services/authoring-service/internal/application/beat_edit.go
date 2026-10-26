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

// beatPosition finds the beat with key, or says it is not there.
func beatPosition(beats []storedBeat, key string) (int, error) {
	if err := requireBeatKey(key); err != nil {
		return 0, err
	}
	at := indexOfBeat(beats, key)
	if at < 0 {
		return 0, domain.ErrUnitNotFound
	}
	return at, nil
}

// Edit stores the Creator's hand edit of a beat: any of title, goal, content,
// next (text), fn (a function of domain.BeatFns), dur (seconds) and facts
// ("F01, F02"). A field whose value changes becomes the Creator's; a text
// field emptied goes back to the AI. Editing sends the beat back to review.
func (uc *BeatDeskUseCase) Edit(ctx context.Context, projectID, key string, fields map[string]string) error {
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return err
	}
	if err := requireBeatKey(key); err != nil {
		return err
	}
	if len(fields) == 0 {
		return fmt.Errorf("%w: không có gì để sửa", domain.ErrInvalidUnitEdit)
	}
	beats, gate, err := uc.beats(ctx, projectID)
	if err != nil {
		return err
	}
	at, err := beatPosition(beats, key)
	if err != nil {
		return err
	}
	slot, beat := beats[at].unit, beats[at].Beat
	facts, err := loadKeptFacts(ctx, uc.units, projectID)
	if err != nil {
		return err
	}
	mark := func(field string, changed bool) {
		if changed && !slot.HasCreatorField(field) {
			slot.CreatorFields = append(slot.CreatorFields, field)
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
				slot.CreatorFields = withoutField(slot.CreatorFields, field)
			} else {
				mark(field, value != *target)
			}
			*target = value
		case domain.BeatFieldDur:
			secs, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("%w: thời lượng %q không phải số giây", domain.ErrInvalidUnitEdit, raw)
			}
			mark(field, secs != beat.Dur)
			beat.Dur = secs
		case domain.BeatFieldFacts:
			ids, err := parseFactList(value)
			if err != nil {
				return fmt.Errorf("%w: %v", domain.ErrInvalidUnitEdit, err)
			}
			mark(field, strings.Join(ids, ",") != strings.Join(beat.Facts, ","))
			beat.Facts = ids
		default:
			return fmt.Errorf("%w: beat không có trường %q để sửa", domain.ErrInvalidUnitEdit, field)
		}
	}
	if err := beat.Validate(factKnownTo(facts)); err != nil {
		return fmt.Errorf("%w: %v", domain.ErrInvalidUnitEdit, err)
	}
	slot.Status = domain.UnitPending
	beats[at].unit, beats[at].Beat = slot, beat
	return uc.saveSheet(ctx, projectID, beats, gate)
}

// Redo has the AI write this one beat again with the Creator's note, reading
// the other beats as context and the time budget they leave. It serves a beat
// of the skeleton that has no content yet as well as one already written.
// Function and facts stay as they are; the Creator asked for this beat by
// name, so the wording that was the Creator's is released while a length the
// Creator set is kept. When this beat was the last one without content the
// lengths of the sheet are brought inside the target.
func (uc *BeatDeskUseCase) Redo(ctx context.Context, projectID, key, note string) error {
	if err := requireBeatKey(key); err != nil {
		return err
	}
	if err := uc.requireAI(ctx); err != nil {
		return err
	}
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return err
	}
	bc, err := uc.context(ctx, projectID)
	if err != nil {
		return err
	}
	work, gate, err := uc.beats(ctx, projectID)
	if err != nil {
		return err
	}
	if !gate.SkeletonApproved {
		return fmt.Errorf("%w: %w", domain.ErrInvalidStatus, ErrBeatSkeletonNeeded)
	}
	at, err := beatPosition(work, key)
	if err != nil {
		return err
	}
	written := make([]bool, len(work))
	for i, b := range work {
		written[i] = b.Written()
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
		return err
	}
	work[at].Beat = beat
	rewritten := make([]bool, len(work))
	rewritten[at] = true
	return uc.saveContent(ctx, projectID, bc, work, rewritten, gate)
}

// Approve marks a beat approved once it has a goal.
func (uc *BeatDeskUseCase) Approve(ctx context.Context, projectID, key string) error {
	return uc.setStatus(ctx, projectID, key, domain.UnitApproved)
}

// Unapprove sends an approved beat back to review.
func (uc *BeatDeskUseCase) Unapprove(ctx context.Context, projectID, key string) error {
	return uc.setStatus(ctx, projectID, key, domain.UnitPending)
}

func (uc *BeatDeskUseCase) setStatus(ctx context.Context, projectID, key string, status domain.UnitStatus) error {
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return err
	}
	beats, gate, err := uc.beats(ctx, projectID)
	if err != nil {
		return err
	}
	at, err := beatPosition(beats, key)
	if err != nil {
		return err
	}
	if status == domain.UnitApproved && strings.TrimSpace(beats[at].Goal) == "" {
		return fmt.Errorf("%w: viết mục tiêu của beat trước khi duyệt", ErrBeatNoGoal)
	}
	beats[at].unit.Status = status
	return uc.saveSheet(ctx, projectID, beats, gate)
}

// Impact lists the dialogue written for the beat: editing the beat marks it
// as out of date for the Creator to review again.
func (uc *BeatDeskUseCase) Impact(ctx context.Context, projectID, key string) ([]string, error) {
	items := []string{}
	if err := requireBeatKey(key); err != nil {
		return items, err
	}
	unit, err := uc.units.GetUnit(ctx, projectID, domain.UnitKindScriptBeat, key)
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
