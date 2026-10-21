package application

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"authoring/internal/domain"
)

// ErrScriptBeatEmpty is returned when a beat without dialogue is approved.
var ErrScriptBeatEmpty = errors.New("beat chưa có lời thoại")

// Fields of a script_beat unit a hand edit sets. A field is the prefix
// followed by a line id: "line:b1.2" is the text of line b1.2 (empty deletes
// the line), "facts:b1.2" its fact codes ("F01, F02"), and "add:b1.2" a new
// line after b1.2 ("add:start" puts one first).
const (
	scriptFieldLine  = "line:"
	scriptFieldFacts = "facts:"
	scriptFieldAdd   = "add:"
	// ScriptFieldAddStart adds a line before every other line of the beat.
	ScriptFieldAddStart = scriptFieldAdd + "start"
	// scriptCreatorField is the name under which a hand edit is remembered.
	scriptCreatorField = "lines"
)

// scriptKind makes each beat's dialogue editable, redoable and approvable
// through the generic unit API (UnitKind for domain.UnitKindScriptBeat).
type scriptKind struct{ desk *ScriptDeskUseCase }

// ScriptUnitKind returns the UnitKind of the beats' dialogue.
func (uc *ScriptDeskUseCase) ScriptUnitKind() UnitKind { return scriptKind{desk: uc} }

// Edit stores the Creator's hand edit of one beat's dialogue. Editing sends
// the beat back to review. A beat with no dialogue yet may be written by hand.
func (k scriptKind) Edit(ctx context.Context, projectID, key string, fields map[string]string) (domain.ReviewUnit, error) {
	if !domain.IsBeatKey(key) {
		return domain.ReviewUnit{}, domain.ErrUnitNotFound
	}
	if len(fields) == 0 {
		return domain.ReviewUnit{}, fmt.Errorf("%w: không có gì để sửa", domain.ErrInvalidUnitEdit)
	}
	uc := k.desk
	beats, _, _, err := uc.beats.beats(ctx, projectID)
	if err != nil {
		return domain.ReviewUnit{}, err
	}
	if beatIndex(beats, key) < 0 {
		return domain.ReviewUnit{}, domain.ErrUnitNotFound
	}
	unit, err := uc.units.GetUnit(ctx, projectID, domain.UnitKindScriptBeat, key)
	script := domain.ScriptBeat{Lines: []domain.ScriptLine{}, NextSeq: 1}
	switch {
	case errors.Is(err, domain.ErrUnitNotFound):
		sha, err := uc.units.UnitSHA(ctx, projectID, domain.UnitKindBeat, key)
		if err != nil {
			return domain.ReviewUnit{}, err
		}
		unit = domain.ReviewUnit{Status: domain.UnitPending, CreatorFields: []string{}, InputRef: domain.UnitKindBeat + ":" + key, InputSHA: sha}
	case err != nil:
		return domain.ReviewUnit{}, err
	default:
		if script, err = readScriptBeat(unit); err != nil {
			return domain.ReviewUnit{}, err
		}
	}
	facts, err := loadKeptFacts(ctx, uc.units, projectID)
	if err != nil {
		return domain.ReviewUnit{}, err
	}
	for _, field := range sortedKeys(fields) {
		value := strings.TrimSpace(fields[field])
		switch {
		case strings.HasPrefix(field, scriptFieldLine):
			at := lineIndex(script.Lines, strings.TrimPrefix(field, scriptFieldLine))
			if at < 0 {
				return domain.ReviewUnit{}, fmt.Errorf("%w: không có câu %s", domain.ErrInvalidUnitEdit, strings.TrimPrefix(field, scriptFieldLine))
			}
			if value == "" {
				script.Lines = append(script.Lines[:at:at], script.Lines[at+1:]...)
			} else {
				script.Lines[at].Text = value
			}
		case strings.HasPrefix(field, scriptFieldFacts):
			at := lineIndex(script.Lines, strings.TrimPrefix(field, scriptFieldFacts))
			if at < 0 {
				return domain.ReviewUnit{}, fmt.Errorf("%w: không có câu %s", domain.ErrInvalidUnitEdit, strings.TrimPrefix(field, scriptFieldFacts))
			}
			ids, err := parseFactList(value)
			if err != nil {
				return domain.ReviewUnit{}, fmt.Errorf("%w: %v", domain.ErrInvalidUnitEdit, err)
			}
			script.Lines[at].Facts = ids
		case strings.HasPrefix(field, scriptFieldAdd):
			after := strings.TrimPrefix(field, scriptFieldAdd)
			at := 0
			if after != "start" {
				if at = lineIndex(script.Lines, after) + 1; at == 0 {
					return domain.ReviewUnit{}, fmt.Errorf("%w: không có câu %s", domain.ErrInvalidUnitEdit, after)
				}
			}
			line := domain.ScriptLine{ID: domain.ScriptLineID(key, script.NextSeq), Text: value, Facts: []string{}}
			script.NextSeq++
			script.Lines = append(script.Lines[:at:at], append([]domain.ScriptLine{line}, script.Lines[at:]...)...)
		default:
			return domain.ReviewUnit{}, fmt.Errorf("%w: lời thoại không có trường %q để sửa", domain.ErrInvalidUnitEdit, field)
		}
	}
	if err := script.Validate(factKnownTo(facts)); err != nil {
		return domain.ReviewUnit{}, fmt.Errorf("%w: %v", domain.ErrInvalidUnitEdit, err)
	}
	if !unit.HasCreatorField(scriptCreatorField) {
		unit.CreatorFields = append(unit.CreatorFields, scriptCreatorField)
	}
	unit.Status = domain.UnitPending
	return uc.saveScript(ctx, projectID, key, script, unit)
}

// Redo has the AI write the whole dialogue of this beat again with the
// Creator's note. The Creator asked for it by name, so the dialogue stops
// being the Creator's.
func (k scriptKind) Redo(ctx context.Context, projectID, key, note string) (domain.ReviewUnit, error) {
	if !domain.IsBeatKey(key) {
		return domain.ReviewUnit{}, domain.ErrUnitNotFound
	}
	uc := k.desk
	if err := uc.requireAI(); err != nil {
		return domain.ReviewUnit{}, err
	}
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return domain.ReviewUnit{}, err
	}
	sc, err := uc.context(ctx, projectID)
	if err != nil {
		return domain.ReviewUnit{}, err
	}
	at := beatIndex(sc.beats, key)
	if at < 0 {
		return domain.ReviewUnit{}, domain.ErrUnitNotFound
	}
	var old *storedScript
	if s, ok := sc.scripts[key]; ok {
		old = &s
	}
	task := domain.ScriptTaskRedo()
	if old == nil {
		task = domain.ScriptTaskWrite()
	}
	return uc.writeBeat(ctx, projectID, sc, at, task, note, old)
}

// Impact is empty: nothing is built on the dialogue yet.
func (scriptKind) Impact(context.Context, string, string) ([]string, error) { return []string{}, nil }

// CheckApprovable refuses a beat with no dialogue.
func (scriptKind) CheckApprovable(_ context.Context, unit domain.ReviewUnit) error {
	s, err := readScriptBeat(unit)
	if err != nil {
		return err
	}
	if len(s.Lines) == 0 {
		return fmt.Errorf("%w: viết lời thoại của beat trước khi duyệt", ErrScriptBeatEmpty)
	}
	return nil
}
