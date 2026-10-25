package application

import (
	"context"
	"errors"
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
	// loSeconds and hiSeconds bound the total length of the sheet.
	loSeconds, hiSeconds int
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
	project, err := uc.projects.Get(ctx, projectID)
	if err != nil {
		return bc, fmt.Errorf("load project: %w", err)
	}
	if bc.topic, err = uc.units.GetAuthoringTopic(ctx, projectID); err != nil {
		return bc, fmt.Errorf("load topic: %w", err)
	}
	bc.language = briefLanguage(project.ContentLanguage)
	brief, briefOK, err := uc.briefs.ApprovedBrief(ctx, projectID)
	if err != nil {
		return bc, err
	}
	if !briefOK {
		return bc, fmt.Errorf("%w: %w", domain.ErrInvalidStatus, ErrBeatsNeedBrief)
	}
	target := domain.NewTargetLength(brief.DurationSeconds)
	bc.targetLabel = target.Label(bc.language)
	bc.loSeconds, bc.hiSeconds = target.Lo, target.Hi
	bc.brief = brief.Section()
	if bc.facts, err = loadKeptFacts(ctx, uc.units, projectID); err != nil {
		return bc, err
	}
	fields, err := uc.storyFields(ctx, projectID)
	if err != nil {
		return bc, err
	}
	bc.story = storyFieldsText(fields)
	models, err := uc.units.GetAuthoringModels(ctx, projectID)
	if err != nil {
		return bc, fmt.Errorf("load authoring models: %w", err)
	}
	bc.model = models.ModelFor("beat_sheet")
	return bc, nil
}

// input reads what the AI is given besides the facts, as the prompt gets it.
func (uc *BeatDeskUseCase) input(ctx context.Context, projectID string) (BeatInput, error) {
	var in BeatInput
	var err error
	if in.Topic, err = uc.units.GetAuthoringTopic(ctx, projectID); err != nil {
		return in, fmt.Errorf("load topic: %w", err)
	}
	brief, ok, err := uc.briefs.ApprovedBrief(ctx, projectID)
	if err != nil {
		return in, err
	}
	if ok {
		in.Brief, in.BriefApproved = brief.Section(), true
	}
	fields, err := uc.storyFields(ctx, projectID)
	if err != nil {
		return in, err
	}
	in.Story = storyFieldsText(fields)
	return in, nil
}

// storyFields reads the approved story in the fields the beat sheet is built from.
func (uc *BeatDeskUseCase) storyFields(ctx context.Context, projectID string) (map[string]domain.StoryField, error) {
	story, _, ok, err := uc.stories.ApprovedStory(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("%w: %w", domain.ErrInvalidStatus, ErrBeatsNeedStory)
	}
	return story.BeatFields(), nil
}

// storyFieldsText shows the story to the model, one labelled block per field.
func storyFieldsText(fields map[string]domain.StoryField) string {
	labels := []struct{ key, name string }{
		{domain.BeatFieldQuestion, "Câu hỏi trung tâm"},
		{domain.BeatFieldChain, "Mạch kể, một nhịp một dòng"},
		{domain.BeatFieldPayoff, "Câu trả lời cốt lõi"},
	}
	var sb strings.Builder
	for _, l := range labels {
		f, ok := fields[l.key]
		if !ok {
			continue
		}
		sb.WriteString(l.key + " (" + l.name + "):\n" + f.Text + "\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}

// currentBeatsText lists the stored beats for a prompt, in order, with their keys.
func currentBeatsText(beats []storedBeat) string {
	if len(beats) == 0 {
		return "(chưa có beat nào)"
	}
	lines := make([]string, 0, len(beats))
	for i, b := range beats {
		if !b.Written() {
			lines = append(lines, fmt.Sprintf("%s (B%d) · %s · dữ kiện %s · CHƯA VIẾT\n  bước câu chuyện: %s", b.unit.Key, i+1, b.Fn, strings.Join(b.Facts, ", "), b.Step))
			continue
		}
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

// ErrBeatSkeletonNeeded is returned when the content is asked for before the
// Creator approved the skeleton.
var ErrBeatSkeletonNeeded = errors.New("duyệt bộ khung của beat sheet trước khi AI viết nội dung")

// ErrBeatSkeletonLocked is returned when the skeleton is built again after it
// was approved.
var ErrBeatSkeletonLocked = errors.New("bộ khung đã duyệt, không dựng lại được")

// beatWriteAttempts is how many times one beat is asked for before the run
// gives up.
const beatWriteAttempts = 2

// BuildSkeleton builds the skeleton of the sheet from the approved story, with
// no model: a Hook, one beat per step of the chain and a Payoff, each with the
// facts its line cites. It replaces the beats now on the sheet, so it is the
// Creator's call to rebuild; once the skeleton is approved and the sheet has
// beats it is refused. The skeleton it builds is not approved.
func (uc *BeatDeskUseCase) BuildSkeleton(ctx context.Context, projectID string) error {
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return err
	}
	approved, err := uc.storyApproved(ctx, projectID)
	if err != nil {
		return err
	}
	if !approved {
		return fmt.Errorf("%w: %w", domain.ErrInvalidStatus, ErrBeatsNeedStory)
	}
	current, gate, err := uc.beats(ctx, projectID)
	if err != nil {
		return err
	}
	if gate.SkeletonApproved && len(current) > 0 {
		return fmt.Errorf("%w: %w", domain.ErrInvalidStatus, ErrBeatSkeletonLocked)
	}
	fields, err := uc.storyFields(ctx, projectID)
	if err != nil {
		return err
	}
	target, briefOK, err := approvedTarget(ctx, uc.briefs, projectID)
	if err != nil {
		return err
	}
	if !briefOK {
		return fmt.Errorf("%w: %w", domain.ErrInvalidStatus, ErrBeatsNeedBrief)
	}
	outline, err := domain.NewBeatOutline(fields[domain.BeatFieldChain].Text, target.Lo, target.Hi)
	if err != nil {
		return fmt.Errorf("read story chain: %w", err)
	}
	payoffText, payoffFacts, err := domain.ParseRefLine(fields[domain.BeatFieldPayoff].Text, nil)
	if err != nil {
		return fmt.Errorf("read story payoff: %w", err)
	}
	beats, err := outline.Skeleton(domain.ChainStep{Text: strings.TrimSpace(fields[domain.BeatFieldQuestion].Text)}, domain.ChainStep{Text: payoffText, Facts: payoffFacts})
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrInvalidUnitEdit, err)
	}
	facts, err := loadKeptFacts(ctx, uc.units, projectID)
	if err != nil {
		return err
	}
	known := factKnownTo(facts)
	sha, err := uc.storyHash(ctx, projectID)
	if err != nil {
		return err
	}
	built := make([]storedBeat, 0, len(beats))
	for _, b := range beats {
		kept := b.Facts[:0:0]
		for _, id := range b.Facts {
			if known(id) {
				kept = append(kept, id)
			}
		}
		b.Facts = kept
		key := domain.BeatKey(gate.NextSeq)
		gate.NextSeq++
		built = append(built, storedBeat{unit: newBeatSlot(key, sha), Beat: b})
	}
	gate.Rebalanced, gate.SkeletonApproved = 0, false
	return uc.saveSheet(ctx, projectID, built, gate)
}

// storyHash is the hash of the approved story, which a beat records as the
// story it was written from.
func (uc *BeatDeskUseCase) storyHash(ctx context.Context, projectID string) (string, error) {
	_, hash, ok, err := uc.stories.ApprovedStory(ctx, projectID)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("%w: %w", domain.ErrInvalidStatus, ErrBeatsNeedStory)
	}
	return hash, nil
}

// ApproveSkeleton records that the Creator accepted the functions and facts of
// the beats; the AI may write their content from then on.
func (uc *BeatDeskUseCase) ApproveSkeleton(ctx context.Context, projectID string) error {
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return err
	}
	current, gate, err := uc.beats(ctx, projectID)
	if err != nil {
		return err
	}
	if len(current) == 0 {
		return fmt.Errorf("%w: chưa có beat nào để duyệt bộ khung", domain.ErrInvalidUnitEdit)
	}
	gate.SkeletonApproved = true
	return uc.saveSheet(ctx, projectID, current, gate)
}

// writeOne has the AI write the content of the beat at index at of work, whose
// earlier beats are already written. A reply that cannot be read is asked for
// once more with what was wrong in the note. The function and facts stay as
// the skeleton fixed them and the parts the Creator wrote keep their value.
func (uc *BeatDeskUseCase) writeOne(ctx context.Context, projectID string, bc beatContext, work []storedBeat, written []bool, at int, note string) (domain.Beat, error) {
	old := work[at]
	used, remaining := 0, 0
	for i, b := range work {
		if written[i] {
			used += b.Dur
		} else {
			remaining++
		}
	}
	task := domain.BeatTaskWrite(old.unit.Key, at+1, len(work), old.Beat, bc.loSeconds, bc.hiSeconds, used, remaining)
	runNote := note
	for attempt := 1; ; attempt++ {
		drafts, err := uc.run(ctx, projectID, bc, work, task, runNote)
		if err == nil && (len(drafts) != 1 || (drafts[0].Key != "" && drafts[0].Key != old.unit.Key)) {
			err = fmt.Errorf("%w: cần đúng một beat %s", domain.ErrBeatReplyUnusable, old.unit.Key)
		}
		if err == nil {
			beat := drafts[0].Beat
			beat.Fn, beat.Facts, beat.Step = old.Fn, old.Facts, old.Step
			creatorFieldsOf(&beat, old.Beat, old.unit.CreatorFields)
			if verr := beat.Validate(factKnownTo(bc.facts)); verr != nil {
				err = fmt.Errorf("%w: %v", domain.ErrBeatReplyUnusable, verr)
			} else {
				return beat, nil
			}
		}
		if !errors.Is(err, domain.ErrBeatReplyUnusable) {
			return domain.Beat{}, err
		}
		if attempt == beatWriteAttempts {
			return domain.Beat{}, fmt.Errorf("beat %s (B%d) viết %d lần vẫn sai: %w", old.unit.Key, at+1, attempt, err)
		}
		runNote = strings.TrimSpace(note + "\nLẦN VIẾT TRƯỚC SAI: " + err.Error() + ". Viết lại beat này và sửa điểm trên.")
	}
}

// saveContent stores the beats a content run wrote and, when the run leaves no
// beat unwritten, brings the total length inside the target first: the lengths
// the Creator set and the beats already approved stay as they are and every
// other beat is resized together. rewritten marks the beats the run wrote; the
// others change only when their length was resized. It fails, and stores
// nothing, when no lengths fit.
func (uc *BeatDeskUseCase) saveContent(ctx context.Context, projectID string, bc beatContext, work []storedBeat, rewritten []bool, gate domain.BeatSheetGate) error {
	durs := make([]int, len(work))
	unwritten := false
	for i, b := range work {
		durs[i] = b.Dur
		unwritten = unwritten || !b.Written()
	}
	if !unwritten {
		beats := make([]domain.Beat, len(work))
		fixed := make([]bool, len(work))
		for i, b := range work {
			beats[i] = b.Beat
			fixed[i] = b.unit.HasCreatorField(domain.BeatFieldDur) || (!rewritten[i] && b.unit.Status == domain.UnitApproved)
		}
		var err error
		if durs, err = domain.BalanceSeconds(beats, fixed, bc.loSeconds, bc.hiSeconds); err != nil {
			return fmt.Errorf("%w: %v", domain.ErrBeatReplyUnusable, err)
		}
	}
	sha, err := uc.storyHash(ctx, projectID)
	if err != nil {
		return err
	}
	gate.Rebalanced = 0
	saved := make([]storedBeat, len(work))
	for i, b := range work {
		resized := durs[i] != b.Dur
		if resized {
			gate.Rebalanced++
		}
		if rewritten[i] || resized {
			b.Dur = durs[i]
		}
		if rewritten[i] {
			b.unit.Status, b.unit.StoryHash = domain.UnitPending, sha
		}
		saved[i] = b
	}
	return uc.saveSheet(ctx, projectID, saved, gate)
}

// Generate has the AI write the content of the beats of the approved
// skeleton, one beat at a time in order; each call sees the beats written so
// far and the time budget left. With onlyUnwritten the beats that already have
// content are left alone; otherwise every beat is written again. Function and
// facts are not changed and the parts the Creator wrote keep their value. When
// the run leaves no beat unwritten their lengths are brought inside the
// target. Nothing is stored unless all of it succeeds. Every beat written goes
// to review.
func (uc *BeatDeskUseCase) Generate(ctx context.Context, projectID, note string, onlyUnwritten bool) error {
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
	current, gate, err := uc.beats(ctx, projectID)
	if err != nil {
		return err
	}
	if !gate.SkeletonApproved || len(current) == 0 {
		return fmt.Errorf("%w: %w", domain.ErrInvalidStatus, ErrBeatSkeletonNeeded)
	}
	// work shows the model the sheet as the run builds it: a beat not yet
	// written in this run appears as it stands in the skeleton.
	work := make([]storedBeat, len(current))
	written := make([]bool, len(current))
	rewritten := make([]bool, len(current))
	for i, b := range current {
		work[i] = b
		if onlyUnwritten && b.Written() {
			written[i] = true
			continue
		}
		rewritten[i] = true
		work[i].Title, work[i].Goal, work[i].Content, work[i].Next = "", "", "", ""
		creatorFieldsOf(&work[i].Beat, b.Beat, b.unit.CreatorFields)
	}
	for i := range current {
		if written[i] {
			continue
		}
		beat, err := uc.writeOne(ctx, projectID, bc, work, written, i, note)
		if err != nil {
			return err
		}
		work[i].Beat, written[i] = beat, true
	}
	return uc.saveContent(ctx, projectID, bc, work, rewritten, gate)
}

// Add puts a new beat after the beat with key after (the first place of an
// empty sheet when after is ""). With blank false the AI writes it between its
// neighbours, following note; with blank true an empty beat is added for the
// Creator to fill in, which works without a language model.
func (uc *BeatDeskUseCase) Add(ctx context.Context, projectID, after string, blank bool, note string) (string, error) {
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return "", err
	}
	current, gate, err := uc.beats(ctx, projectID)
	if err != nil {
		return "", err
	}
	if len(current) >= domain.MaxBeats {
		return "", ErrBeatSheetFull
	}
	at := 0
	if after != "" {
		if at = indexOfBeat(current, after) + 1; at == 0 {
			return "", domain.ErrUnitNotFound
		}
	} else if len(current) > 0 {
		return "", fmt.Errorf("%w: nêu beat mà beat mới đứng sau", domain.ErrInvalidUnitEdit)
	}
	key := domain.BeatKey(gate.NextSeq)
	slot := newBeatSlot(key, "")
	var beat domain.Beat
	if blank {
		beat = domain.Beat{Fn: domain.BeatFns[2], Title: "Beat mới", Facts: []string{}, Dur: 15}
	} else {
		if err := uc.requireAI(); err != nil {
			return "", err
		}
		if !gate.SkeletonApproved {
			return "", fmt.Errorf("%w: %w", domain.ErrInvalidStatus, ErrBeatSkeletonNeeded)
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
		if slot.StoryHash, err = uc.storyHash(ctx, projectID); err != nil {
			return "", err
		}
	}
	gate.NextSeq++
	added := append(append(append([]storedBeat{}, current[:at]...), storedBeat{unit: slot, Beat: beat}), current[at:]...)
	return key, uc.saveSheet(ctx, projectID, added, gate)
}
