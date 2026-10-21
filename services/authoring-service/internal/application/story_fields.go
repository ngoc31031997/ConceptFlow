package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"authoring/internal/domain"
)

// ErrStoryIncomplete is returned when a story with empty fields is approved.
var ErrStoryIncomplete = errors.New("câu chuyện còn ô trống")

// currentFrameText shows the stored fields to the model, one labelled block each.
func currentFrameText(stored map[string]domain.ReviewUnit) (string, error) {
	var sb strings.Builder
	for _, key := range domain.StoryFieldKeys {
		u, ok := stored[key]
		if !ok {
			continue
		}
		f, err := readStoryField(u)
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(f.Text) == "" {
			continue
		}
		sb.WriteString(key + " (" + domain.StoryFieldLabel(key) + "):\n" + f.Text + "\n")
	}
	if sb.Len() == 0 {
		return "(chưa có)", nil
	}
	return strings.TrimRight(sb.String(), "\n"), nil
}

// keptStoryFieldsText lists the fields the Creator wrote, which stay as they are.
func keptStoryFieldsText(stored map[string]domain.ReviewUnit, want []string) (string, error) {
	writing := toSet(want)
	var lines []string
	for _, key := range domain.StoryFieldKeys {
		u, ok := stored[key]
		if !ok || writing[key] || !u.HasCreatorField(domain.TextFieldName) {
			continue
		}
		f, err := readStoryField(u)
		if err != nil {
			return "", err
		}
		lines = append(lines, "- "+key+": "+f.Text)
	}
	if len(lines) == 0 {
		return "(không có)", nil
	}
	return strings.Join(lines, "\n"), nil
}

// writeFields has the AI write the fields in want from the brief, the facts and
// the chosen approach, with note as the Creator's instruction for this run. A
// field the Creator wrote is kept as it is unless it is asked for by name
// (overwriteMine). When the research step was skipped and there is no fact, the
// facts the AI lists are stored too. A reply that cannot be read stores nothing.
func (uc *StoryDeskUseCase) writeFields(ctx context.Context, projectID string, want []string, note string, overwriteMine bool) error {
	if err := uc.requireAI(); err != nil {
		return err
	}
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return err
	}
	sc, err := uc.context(ctx, projectID)
	if err != nil {
		return err
	}
	listMode := len(sc.facts) == 0
	if listMode && !sc.skipped {
		return domain.ErrResearchEmpty
	}
	stored, err := uc.fieldUnits(ctx, projectID)
	if err != nil {
		return err
	}
	if !overwriteMine {
		var open []string
		for _, key := range want {
			if u, ok := stored[key]; !ok || !u.HasCreatorField(domain.TextFieldName) {
				open = append(open, key)
			}
		}
		if want = open; len(want) == 0 {
			return ErrStoryNothingToWrite
		}
	}
	unit, gate, err := uc.gate(ctx, projectID)
	if err != nil {
		return err
	}
	prompt, err := uc.prompts.GetActive(ctx, domain.RoleStoryFrame)
	if err != nil {
		return fmt.Errorf("load story frame prompt: %w", err)
	}
	approach, err := uc.approachText(ctx, projectID, gate.ApproachKey)
	if err != nil {
		return err
	}
	current, err := currentFrameText(stored)
	if err != nil {
		return err
	}
	kept, err := keptStoryFieldsText(stored, want)
	if err != nil {
		return err
	}
	system := domain.RenderPromptTemplate(prompt.TemplateText, map[string]string{
		"topic":           sc.topic,
		"brief_language":  briefLanguageName(sc.language),
		"brief":           sc.brief,
		"facts":           factsText(sc.facts),
		"facts_rule":      domain.StoryFactsRule(listMode),
		"facts_example":   domain.StoryFactsExample(listMode),
		"approach":        approach,
		"current_frame":   current,
		"kept_fields":     kept,
		"fields_to_write": strings.Join(want, ", "),
		"redo_note":       noteText(note),
	})
	content, err := uc.chat(ctx, domain.RoleStoryFrame, domain.UnitKindStoryField, projectID, sc.model, system)
	if err != nil {
		return err
	}
	reply, err := domain.ParseStoryReply(content, want, factKnownTo(sc.facts))
	if err != nil {
		return err
	}
	if listMode {
		if err := uc.storeListedFacts(ctx, projectID, reply.Facts); err != nil {
			return err
		}
	}

	sha, err := uc.units.UnitSHA(ctx, projectID, domain.UnitKindBrief, domain.BriefUnitKey)
	if err != nil {
		return err
	}
	for _, key := range want {
		raw, err := json.Marshal(reply.Fields[key])
		if err != nil {
			return err
		}
		fieldUnit, existed := stored[key]
		if !existed {
			fieldUnit = domain.ReviewUnit{ProjectID: projectID, Kind: domain.UnitKindStoryField, Key: key}
		} else {
			fieldUnit.RedoCount++
			fieldUnit.CreatorFields = withoutField(fieldUnit.CreatorFields, domain.TextFieldName)
		}
		fieldUnit.Status, fieldUnit.Content = domain.UnitPending, raw
		fieldUnit.InputRef, fieldUnit.InputSHA = storyBriefRef, sha
		if _, err := uc.units.SaveUnit(ctx, fieldUnit); err != nil {
			return err
		}
	}
	gate.UsedFacts = mergeFactCodes(gate.UsedFacts, reply.UsedFacts)
	unit.Status = domain.UnitPending
	_, err = uc.saveGate(ctx, unit, gate)
	return err
}

// mergeFactCodes returns the codes of a followed by those of b that a lacks.
func mergeFactCodes(a, b []string) []string {
	seen := toSet(a)
	out := append([]string{}, a...)
	for _, id := range b {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

// storeListedFacts stores the facts the AI listed from its own knowledge, each
// without a source. A code that already belongs to a fact (kept or dropped) is
// an error: the listed facts are never merged over the Creator's.
func (uc *StoryDeskUseCase) storeListedFacts(ctx context.Context, projectID string, listed []domain.NewStoryFact) error {
	if len(listed) == 0 {
		return nil
	}
	taken, err := uc.allFactIDs(ctx, projectID)
	if err != nil {
		return err
	}
	for _, f := range listed {
		if taken[f.ID] {
			return fmt.Errorf("%w: mã dữ kiện %s do AI liệt kê đã có trong bảng dữ kiện, thử viết lại", domain.ErrStoryReplyUnusable, f.ID)
		}
	}
	for _, f := range listed {
		raw, err := json.Marshal(domain.Fact{
			ClaimVI: f.ClaimVI, Type: f.Type, Note: f.Note, Keep: true, Link: domain.FactLink{State: domain.LinkNone},
		})
		if err != nil {
			return err
		}
		if _, err := uc.units.SaveUnit(ctx, domain.ReviewUnit{
			ProjectID: projectID, Kind: domain.UnitKindFact, Key: f.ID, Status: domain.UnitPending, Content: raw,
		}); err != nil {
			return err
		}
	}
	return nil
}

// GenerateFrame has the AI write the fields of the story the Creator has not
// written, following the chosen approach; note is the Creator's instruction for
// this run. The story goes to review.
func (uc *StoryDeskUseCase) GenerateFrame(ctx context.Context, projectID, note string) error {
	return uc.writeFields(ctx, projectID, domain.StoryFieldKeys, note, false)
}

// fieldKind makes each story field editable and redoable through the generic
// unit API (UnitKind for domain.UnitKindStoryField).
type fieldKind struct{ desk *StoryDeskUseCase }

// FieldUnitKind returns the UnitKind of the story fields.
func (uc *StoryDeskUseCase) FieldUnitKind() UnitKind { return fieldKind{desk: uc} }

func requireFieldKey(key string) error {
	if !domain.IsStoryFieldKey(key) {
		return domain.ErrUnitNotFound
	}
	return nil
}

// Edit stores the Creator's hand edit of a field's text. A field whose text
// changes becomes the Creator's; a field emptied goes back to the AI. Editing
// sends an approved story back to review. A chain or payoff may end a line with
// the facts it rests on as "[F01, F02]"; a code not in the fact table is refused.
func (k fieldKind) Edit(ctx context.Context, projectID, key string, fields map[string]string) (domain.ReviewUnit, error) {
	if err := requireFieldKey(key); err != nil {
		return domain.ReviewUnit{}, err
	}
	text, ok := fields[domain.TextFieldName]
	if len(fields) != 1 || !ok {
		return domain.ReviewUnit{}, fmt.Errorf("%w: ô câu chuyện chỉ có nội dung %q để sửa", domain.ErrInvalidUnitEdit, domain.TextFieldName)
	}
	facts, err := k.desk.keptFacts(ctx, projectID)
	if err != nil {
		return domain.ReviewUnit{}, err
	}
	field, err := domain.NewStoryField(key, text, factKnownTo(facts))
	if err != nil {
		return domain.ReviewUnit{}, fmt.Errorf("%w: %v", domain.ErrInvalidUnitEdit, err)
	}
	unit, err := k.desk.units.GetUnit(ctx, projectID, domain.UnitKindStoryField, key)
	switch {
	case errors.Is(err, domain.ErrUnitNotFound):
		unit = domain.ReviewUnit{ProjectID: projectID, Kind: domain.UnitKindStoryField, Key: key, Status: domain.UnitPending}
	case err != nil:
		return domain.ReviewUnit{}, err
	}
	previous := ""
	if len(unit.Content) > 0 {
		old, err := readStoryField(unit)
		if err != nil {
			return domain.ReviewUnit{}, err
		}
		previous = old.Text
	}
	switch {
	case field.Text == "":
		unit.CreatorFields = withoutField(unit.CreatorFields, domain.TextFieldName)
	case field.Text != previous && !unit.HasCreatorField(domain.TextFieldName):
		unit.CreatorFields = append(unit.CreatorFields, domain.TextFieldName)
	}
	if unit.Content, err = json.Marshal(field); err != nil {
		return domain.ReviewUnit{}, err
	}
	saved, err := k.desk.units.SaveUnit(ctx, unit)
	if err != nil {
		return domain.ReviewUnit{}, err
	}
	if err := k.desk.reopen(ctx, projectID); err != nil {
		return domain.ReviewUnit{}, err
	}
	return saved, nil
}

// Redo has the AI write this one field again with the Creator's note, reading
// the other fields as context. The Creator asked for it by name, so the field
// stops being the Creator's.
func (k fieldKind) Redo(ctx context.Context, projectID, key, note string) (domain.ReviewUnit, error) {
	if err := requireFieldKey(key); err != nil {
		return domain.ReviewUnit{}, err
	}
	if err := k.desk.writeFields(ctx, projectID, []string{key}, note, true); err != nil {
		return domain.ReviewUnit{}, err
	}
	return k.desk.units.GetUnit(ctx, projectID, domain.UnitKindStoryField, key)
}

// Impact is empty: nothing is built on the story yet.
func (fieldKind) Impact(context.Context, string, string) ([]string, error) { return []string{}, nil }

// CheckApprovable accepts any field; the story as a whole is what gets approved.
func (fieldKind) CheckApprovable(context.Context, domain.ReviewUnit) error { return nil }

// storyGateKind makes the story approvable through the generic unit API
// (UnitKind for domain.UnitKindStory). Approving it approves the six fields together.
type storyGateKind struct{ desk *StoryDeskUseCase }

// GateUnitKind returns the gate's UnitKind.
func (uc *StoryDeskUseCase) GateUnitKind() UnitKind { return storyGateKind{desk: uc} }

// Initial gives the gate of a project that has none stored yet, so that a story
// the Creator wrote by hand can be approved before anything else was saved.
func (storyGateKind) Initial(_ context.Context, projectID, key string) (domain.ReviewUnit, bool) {
	if key != domain.StoryUnitKey {
		return domain.ReviewUnit{}, false
	}
	return domain.ReviewUnit{
		ProjectID: projectID, Kind: domain.UnitKindStory, Key: key, Status: domain.UnitPending,
		Content: json.RawMessage(`{"approach_key":"","used_facts":[]}`),
	}, true
}

// Edit is refused: the gate holds no text.
func (storyGateKind) Edit(context.Context, string, string, map[string]string) (domain.ReviewUnit, error) {
	return domain.ReviewUnit{}, fmt.Errorf("%w: câu chuyện được duyệt theo từng ô, không sửa ở đây", domain.ErrInvalidUnitEdit)
}

// Redo is refused: the AI writes the fields, not the gate.
func (storyGateKind) Redo(context.Context, string, string, string) (domain.ReviewUnit, error) {
	return domain.ReviewUnit{}, fmt.Errorf("%w: làm lại từng ô của câu chuyện, không làm lại cổng duyệt", domain.ErrInvalidUnitEdit)
}

// Impact is empty: nothing is built on the story yet.
func (storyGateKind) Impact(context.Context, string, string) ([]string, error) {
	return []string{}, nil
}

// CheckApprovable refuses a story with an empty field.
func (k storyGateKind) CheckApprovable(ctx context.Context, unit domain.ReviewUnit) error {
	stored, err := k.desk.fieldUnits(ctx, unit.ProjectID)
	if err != nil {
		return err
	}
	var missing []string
	for _, key := range domain.StoryFieldKeys {
		u, ok := stored[key]
		if !ok {
			missing = append(missing, domain.StoryFieldLabel(key))
			continue
		}
		f, err := readStoryField(u)
		if err != nil {
			return err
		}
		if strings.TrimSpace(f.Text) == "" {
			missing = append(missing, domain.StoryFieldLabel(key))
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("%w: %s", ErrStoryIncomplete, strings.Join(missing, "; "))
	}
	return nil
}
