package application

import (
	"context"
	"fmt"
	"strings"

	"authoring/internal/domain"
)

// beatContext is what every AI call of the step reads.
type beatContext struct {
	topic       string
	brief       string
	language    string
	story       string
	facts       []storyFact
	targetLabel string
	// model is the model the project chose for this step; "" is the server default.
	model string
}

// context gathers the inputs of an AI call and refuses when the step is not
// open yet: the story must be approved.
func (uc *BeatDeskUseCase) context(ctx context.Context, projectID string) (beatContext, error) {
	var bc beatContext
	approved, err := uc.storyApproved(ctx, projectID)
	if err != nil {
		return bc, err
	}
	if !approved {
		return bc, fmt.Errorf("%w: %w", domain.ErrInvalidStatus, ErrBeatsNeedStory)
	}
	brief, _, err := loadApprovedBrief(ctx, uc.units, projectID)
	if err != nil {
		return bc, err
	}
	project, err := uc.projects.Get(ctx, projectID)
	if err != nil {
		return bc, fmt.Errorf("load project: %w", err)
	}
	if bc.topic, err = uc.units.GetAuthoringTopic(ctx, projectID); err != nil {
		return bc, fmt.Errorf("load topic: %w", err)
	}
	duration, err := uc.units.GetTargetDuration(ctx, projectID)
	if err != nil {
		return bc, err
	}
	bc.language = briefLanguage(project.ContentLanguage)
	bc.targetLabel = domain.TargetDurationLabel(project.VideoOutputMode, duration, bc.language)
	bc.brief = domain.BriefSection(brief, bc.targetLabel)
	if bc.facts, err = loadKeptFacts(ctx, uc.units, projectID); err != nil {
		return bc, err
	}
	if bc.story, err = uc.storyText(ctx, projectID); err != nil {
		return bc, err
	}
	models, err := uc.units.GetAuthoringModels(ctx, projectID)
	if err != nil {
		return bc, fmt.Errorf("load authoring models: %w", err)
	}
	bc.model = models.ModelFor("beat_sheet")
	return bc, nil
}

// storyText shows the approved story frame to the model, one labelled block per field.
func (uc *BeatDeskUseCase) storyText(ctx context.Context, projectID string) (string, error) {
	units, err := uc.units.ListUnits(ctx, projectID, domain.UnitKindStoryField)
	if err != nil {
		return "", err
	}
	byKey := make(map[string]domain.ReviewUnit, len(units))
	for _, u := range units {
		byKey[u.Key] = u
	}
	var sb strings.Builder
	for _, key := range domain.StoryFieldKeys {
		u, ok := byKey[key]
		if !ok {
			continue
		}
		f, err := readStoryField(u)
		if err != nil {
			return "", err
		}
		sb.WriteString(key + " (" + domain.StoryFieldLabel(key) + "):\n" + f.Text + "\n")
	}
	return strings.TrimRight(sb.String(), "\n"), nil
}

// currentBeatsText lists the stored beats for a prompt, in order, with their keys.
func currentBeatsText(beats []storedBeat) string {
	if len(beats) == 0 {
		return "(chưa có beat nào)"
	}
	lines := make([]string, 0, len(beats))
	for i, b := range beats {
		lines = append(lines, fmt.Sprintf("%s (B%d) · %s · %s · %d s · dữ kiện %s\n  mục tiêu: %s\n  nội dung: %s\n  nối sang beat sau: %s",
			b.unit.Key, i+1, b.Fn, b.Title, b.Dur, strings.Join(b.Facts, ", "), b.Goal, b.Content, b.Next))
	}
	return strings.Join(lines, "\n")
}

// keptBeatsText lists the parts of the beats the Creator wrote.
func keptBeatsText(beats []storedBeat) string {
	var lines []string
	for _, b := range beats {
		if len(b.unit.CreatorFields) > 0 {
			lines = append(lines, fmt.Sprintf("- %s: %s", b.unit.Key, strings.Join(b.unit.CreatorFields, ", ")))
		}
	}
	if len(lines) == 0 {
		return "(không có)"
	}
	return strings.Join(lines, "\n")
}

// run renders the beat prompt for one task and returns the beats the AI wrote.
func (uc *BeatDeskUseCase) run(ctx context.Context, projectID string, bc beatContext, current []storedBeat, task, note string) ([]domain.BeatDraft, error) {
	prompt, err := uc.prompts.GetActive(ctx, domain.RoleBeatSheet)
	if err != nil {
		return nil, fmt.Errorf("load beat sheet prompt: %w", err)
	}
	system := domain.RenderPromptTemplate(prompt.TemplateText, map[string]string{
		"topic":          bc.topic,
		"brief_language": briefLanguageName(bc.language),
		"target":         bc.targetLabel,
		"brief":          bc.brief,
		"story":          bc.story,
		"facts":          factsText(bc.facts),
		"current_beats":  currentBeatsText(current),
		"kept_beats":     keptBeatsText(current),
		"redo_note":      noteText(note),
		"task":           task,
	})
	content, err := uc.chat(ctx, projectID, bc.model, system)
	if err != nil {
		return nil, err
	}
	return domain.ParseBeatsReply(content, factKnownTo(bc.facts))
}

// creatorFieldsOf copies the fields of src named in fields onto dst.
func creatorFieldsOf(dst *domain.Beat, src domain.Beat, fields []string) {
	for _, f := range fields {
		switch f {
		case domain.BeatFieldFn:
			dst.Fn = src.Fn
		case domain.BeatFieldTitle:
			dst.Title = src.Title
		case domain.BeatFieldGoal:
			dst.Goal = src.Goal
		case domain.BeatFieldContent:
			dst.Content = src.Content
		case domain.BeatFieldFacts:
			dst.Facts = src.Facts
		case domain.BeatFieldDur:
			dst.Dur = src.Dur
		case domain.BeatFieldNext:
			dst.Next = src.Next
		}
	}
}

// Generate has the AI write the whole sheet from the approved story; note is
// the Creator's instruction for this run. A beat the Creator wrote any part of
// keeps its key and that part, and the reply must keep every such beat. The
// other beats are replaced. A reply that cannot be read stores nothing. Every
// beat goes to review.
func (uc *BeatDeskUseCase) Generate(ctx context.Context, projectID, note string) error {
	if err := uc.requireAI(); err != nil {
		return err
	}
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return err
	}
	bc, err := uc.context(ctx, projectID)
	if err != nil {
		return err
	}
	current, gate, sheetUnit, err := uc.beats(ctx, projectID)
	if err != nil {
		return err
	}
	drafts, err := uc.run(ctx, projectID, bc, current, domain.BeatTaskWriteAll(), note)
	if err != nil {
		return err
	}
	storedByKey := make(map[string]storedBeat, len(current))
	for _, b := range current {
		storedByKey[b.unit.Key] = b
	}
	kept := map[string]bool{}
	for _, d := range drafts {
		if d.Key == "" {
			continue
		}
		if _, ok := storedByKey[d.Key]; !ok {
			return fmt.Errorf("%w: khoá %s không phải beat hiện có", domain.ErrBeatReplyUnusable, d.Key)
		}
		kept[d.Key] = true
	}
	for _, b := range current {
		if len(b.unit.CreatorFields) > 0 && !kept[b.unit.Key] {
			return fmt.Errorf("%w: bỏ mất beat %s mà bạn đã viết", domain.ErrBeatReplyUnusable, b.unit.Key)
		}
	}

	sha, err := uc.units.UnitSHA(ctx, projectID, domain.UnitKindStory, domain.StoryUnitKey)
	if err != nil {
		return err
	}
	order := make([]string, 0, len(drafts))
	for _, d := range drafts {
		key, beat, unit := d.Key, d.Beat, domain.ReviewUnit{Status: domain.UnitPending, InputRef: beatStoryRef, InputSHA: sha}
		if old, ok := storedByKey[key]; ok {
			unit.CreatorFields, unit.RedoCount = old.unit.CreatorFields, old.unit.RedoCount
			creatorFieldsOf(&beat, old.Beat, old.unit.CreatorFields)
			if err := beat.Validate(factKnownTo(bc.facts)); err != nil {
				return fmt.Errorf("%w: beat %s: %v", domain.ErrBeatReplyUnusable, key, err)
			}
		} else {
			key = domain.BeatKey(gate.NextSeq)
			gate.NextSeq++
		}
		if _, err := uc.saveBeat(ctx, projectID, key, beat, unit); err != nil {
			return err
		}
		order = append(order, key)
	}
	for _, old := range current {
		if !kept[old.unit.Key] {
			if err := uc.units.DeleteUnit(ctx, projectID, domain.UnitKindBeat, old.unit.Key); err != nil {
				return err
			}
		}
	}
	gate.Order = order
	return uc.saveSheet(ctx, sheetUnit, gate)
}

// Add puts a new beat after the beat with key after (the first place of an
// empty sheet when after is ""). With blank false the AI writes it between its
// neighbours, following note; with blank true an empty beat is added for the
// Creator to fill in, which works without a language model.
func (uc *BeatDeskUseCase) Add(ctx context.Context, projectID, after string, blank bool, note string) (string, error) {
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return "", err
	}
	current, gate, sheetUnit, err := uc.beats(ctx, projectID)
	if err != nil {
		return "", err
	}
	if len(current) >= domain.MaxBeats {
		return "", ErrBeatSheetFull
	}
	at := 0
	if after != "" {
		if at = indexOf(gate.Order, after) + 1; at == 0 {
			return "", domain.ErrUnitNotFound
		}
	} else if len(current) > 0 {
		return "", fmt.Errorf("%w: nêu beat mà beat mới đứng sau", domain.ErrInvalidUnitEdit)
	}
	key := domain.BeatKey(gate.NextSeq)
	unit := domain.ReviewUnit{Status: domain.UnitPending}
	var beat domain.Beat
	if blank {
		beat = domain.Beat{Fn: domain.BeatFns[2], Title: "Beat mới", Facts: []string{}, Dur: 15}
	} else {
		if err := uc.requireAI(); err != nil {
			return "", err
		}
		bc, err := uc.context(ctx, projectID)
		if err != nil {
			return "", err
		}
		before := ""
		if at < len(current) {
			before = current[at].unit.Key
		}
		drafts, err := uc.run(ctx, projectID, bc, current, domain.BeatTaskAdd(after, before), note)
		if err != nil {
			return "", err
		}
		if len(drafts) != 1 || drafts[0].Key != "" {
			return "", fmt.Errorf("%w: cần đúng một beat mới", domain.ErrBeatReplyUnusable)
		}
		beat = drafts[0].Beat
		sha, err := uc.units.UnitSHA(ctx, projectID, domain.UnitKindStory, domain.StoryUnitKey)
		if err != nil {
			return "", err
		}
		unit.InputRef, unit.InputSHA = beatStoryRef, sha
	}
	if _, err := uc.saveBeat(ctx, projectID, key, beat, unit); err != nil {
		return "", err
	}
	gate.NextSeq++
	gate.Order = append(append(append([]string{}, gate.Order[:at]...), key), gate.Order[at:]...)
	return key, uc.saveSheet(ctx, sheetUnit, gate)
}
