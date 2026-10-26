package application

import (
	"context"
	"fmt"
	"strings"

	"authoring/internal/domain"
)

// Where a find-and-replace writes: the wording the voice reads, or the text on screen.
const (
	ScriptReplaceSpoken = "spoken"
	ScriptReplaceText   = "text"
)

// maxReplaceRunes caps the word being found and the word it becomes.
const maxReplaceRunes = 300

// ScriptReplaceRequest is one find-and-replace over the dialogue of the beats.
type ScriptReplaceRequest struct {
	Find    string
	Replace string
	// Field is ScriptReplaceSpoken or ScriptReplaceText.
	Field     string
	MatchCase bool
	// Apply writes the change; without it the request only previews it.
	Apply bool
}

// ScriptReplaceMatch is one line the replacement changes, as it reads before and after.
type ScriptReplaceMatch struct {
	Beat   string `json:"beat"`
	Line   string `json:"line"`
	Before string `json:"before"`
	After  string `json:"after"`
}

// ScriptReplaceSkipped counts the matching lines of an approved beat, which is left as it is.
type ScriptReplaceSkipped struct {
	Beat  string `json:"beat"`
	Lines int    `json:"lines"`
}

// ScriptReplaceResult is what a find-and-replace changes, or would change.
type ScriptReplaceResult struct {
	Matches         []ScriptReplaceMatch   `json:"matches"`
	SkippedApproved []ScriptReplaceSkipped `json:"skipped_approved"`
	Applied         bool                   `json:"applied"`
}

// Replace finds a word in the dialogue of every beat and replaces it. A beat
// that is approved is never changed; the result counts its matching lines. A
// replaced beat goes back to review like any hand edit. Nothing is stored
// unless every changed beat is still valid after the change, so a refusal
// leaves all beats as they were.
func (uc *ScriptDeskUseCase) Replace(ctx context.Context, projectID string, req ScriptReplaceRequest) (ScriptReplaceResult, error) {
	find, replacement := strings.TrimSpace(req.Find), strings.TrimSpace(req.Replace)
	if err := checkReplaceRequest(find, replacement, req.Field); err != nil {
		return ScriptReplaceResult{}, err
	}
	if req.Apply {
		if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
			return ScriptReplaceResult{}, err
		}
	}
	beats, _, err := uc.beats.beats(ctx, projectID)
	if err != nil {
		return ScriptReplaceResult{}, err
	}
	scripts, err := uc.scripts(ctx, projectID)
	if err != nil {
		return ScriptReplaceResult{}, err
	}
	facts, err := loadKeptFacts(ctx, uc.units, projectID)
	if err != nil {
		return ScriptReplaceResult{}, err
	}

	result := ScriptReplaceResult{Matches: []ScriptReplaceMatch{}, SkippedApproved: []ScriptReplaceSkipped{}}
	changed := map[string]storedScript{}
	var order []string
	for _, b := range beats {
		key := b.unit.Key
		stored, ok := scripts[key]
		if !ok {
			continue
		}
		next := stored
		next.Lines = append([]domain.ScriptLine(nil), stored.Lines...)
		var matches []ScriptReplaceMatch
		for i, line := range next.Lines {
			before := replaceBase(line, req.Field)
			after, n := domain.ReplaceWord(before, find, replacement, req.MatchCase)
			if n == 0 {
				continue
			}
			if strings.TrimSpace(after) == "" {
				return ScriptReplaceResult{}, fmt.Errorf("%w: câu %s sẽ trống sau khi thay", domain.ErrInvalidUnitEdit, line.ID)
			}
			setReplaced(&next.Lines[i], req.Field, after)
			matches = append(matches, ScriptReplaceMatch{Beat: key, Line: line.ID, Before: before, After: after})
		}
		switch {
		case len(matches) == 0:
		case stored.unit.Status == domain.UnitApproved:
			result.SkippedApproved = append(result.SkippedApproved, ScriptReplaceSkipped{Beat: key, Lines: len(matches)})
		default:
			if err := next.ScriptBeat.Validate(factKnownTo(facts)); err != nil {
				return ScriptReplaceResult{}, fmt.Errorf("%w: beat %s: %v", domain.ErrInvalidUnitEdit, key, err)
			}
			result.Matches = append(result.Matches, matches...)
			changed[key] = next
			order = append(order, key)
		}
	}
	if !req.Apply {
		return result, nil
	}
	for _, key := range order {
		next := changed[key]
		unit := next.unit
		if !unit.HasCreatorField(scriptCreatorField) {
			unit.CreatorFields = append(unit.CreatorFields, scriptCreatorField)
		}
		unit.Status = domain.UnitPending
		if _, err := uc.saveScript(ctx, projectID, key, next.ScriptBeat, unit); err != nil {
			return ScriptReplaceResult{}, fmt.Errorf("save beat %s: %w", key, err)
		}
	}
	result.Applied = true
	return result, nil
}

func checkReplaceRequest(find, replacement, field string) error {
	switch {
	case find == "":
		return fmt.Errorf("%w: nhập từ cần tìm", domain.ErrInvalidUnitEdit)
	case len([]rune(find)) > maxReplaceRunes || len([]rune(replacement)) > maxReplaceRunes:
		return fmt.Errorf("%w: từ dài quá %d ký tự", domain.ErrInvalidUnitEdit, maxReplaceRunes)
	case field != ScriptReplaceSpoken && field != ScriptReplaceText:
		return fmt.Errorf("%w: đổi vào lời đọc hoặc chữ hiển thị", domain.ErrInvalidUnitEdit)
	}
	return nil
}

// replaceBase is the text a replacement reads in line: the wording the voice
// reads for a spoken replacement (the written text while the line has none of
// its own), the text on screen otherwise.
func replaceBase(line domain.ScriptLine, field string) string {
	if field == ScriptReplaceSpoken {
		return line.SpokenText()
	}
	return line.Text
}

// setReplaced stores the replaced wording in the field the replacement targets.
func setReplaced(line *domain.ScriptLine, field, value string) {
	if field == ScriptReplaceSpoken {
		line.Spoken = value
		return
	}
	line.Text = value
}
