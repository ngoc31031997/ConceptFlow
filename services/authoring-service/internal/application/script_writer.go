package application

import (
	"context"
	"fmt"
	"strings"

	"authoring/internal/domain"
)

// scriptContext is what every AI call of the script step reads.
type scriptContext struct {
	beatContext
	beats   []storedBeat
	scripts map[string]storedScript
}

// scriptModelStep is the key of the model the Creator chose for this step: the
// script step keeps the "story" slot the earlier script step used.
const scriptModelStep = "story"

// context gathers the inputs of an AI call and refuses when the step is not
// open yet: every beat must be approved.
func (uc *ScriptDeskUseCase) context(ctx context.Context, projectID string) (scriptContext, error) {
	var sc scriptContext
	var err error
	if sc.beats, _, _, err = uc.beats.beats(ctx, projectID); err != nil {
		return sc, err
	}
	if !beatsApproved(sc.beats) {
		return sc, fmt.Errorf("%w: %w", domain.ErrInvalidStatus, ErrScriptNeedsBeats)
	}
	if sc.beatContext, err = uc.beats.context(ctx, projectID); err != nil {
		return sc, err
	}
	models, err := uc.units.GetAuthoringModels(ctx, projectID)
	if err != nil {
		return sc, fmt.Errorf("load authoring models: %w", err)
	}
	sc.model = models.ModelFor(scriptModelStep)
	if sc.scripts, err = uc.scripts(ctx, projectID); err != nil {
		return sc, err
	}
	return sc, nil
}

// requireAI fails unless a language model is configured.
func (uc *ScriptDeskUseCase) requireAI() error {
	if uc.provider == nil {
		return ErrLLMNotConfigured
	}
	if r, ok := uc.provider.(interface{ Ready() bool }); ok && !r.Ready() {
		return ErrLLMNotConfigured
	}
	return nil
}

// chat sends one system prompt and records the usage.
func (uc *ScriptDeskUseCase) chat(ctx context.Context, projectID, model, system string) (string, error) {
	started := uc.now()
	result, err := uc.provider.Chat(ctx, ChatRequest{
		System: system, User: userTurnNudge, MaxTokens: uc.maxOutputTokens, Temperature: 0.7,
		Model: model, OnProgress: ProgressFrom(ctx),
	})
	if uc.recorder != nil {
		uc.recorder.Record(ctx, RecordFor(uc.provider.Name(), string(domain.RoleScriptWriter), domain.UnitKindScriptBeat, projectID, result.Usage, started, err))
	}
	if err != nil {
		return "", err
	}
	return result.Content, nil
}

// beatText describes one beat to the model.
func beatText(b storedBeat, number int, facts []storyFact) string {
	claims := map[string]string{}
	for _, f := range facts {
		claims[f.ID] = f.ClaimVI
	}
	used := make([]string, 0, len(b.Facts))
	for _, id := range b.Facts {
		used = append(used, id+": "+claims[id])
	}
	if len(used) == 0 {
		used = []string{"(không có)"}
	}
	return fmt.Sprintf("%s (B%d) · %s · %s · %d giây\nmục tiêu: %s\nnội dung: %s\ndữ kiện của beat:\n- %s",
		b.unit.Key, number, b.Fn, b.Title, b.Dur, b.Goal, b.Content, strings.Join(used, "\n- "))
}

// neighbourText shows the dialogue of the beat next to the one being written.
func neighbourText(sc scriptContext, at int) string {
	if at < 0 || at >= len(sc.beats) {
		return "(không có beat ở phía này)"
	}
	s, ok := sc.scripts[sc.beats[at].unit.Key]
	if !ok || len(s.Lines) == 0 {
		return "(beat này chưa có lời thoại)"
	}
	return linesText(s.Lines)
}

// run renders the script prompt for one task about the beat at index at and returns the model's reply.
func (uc *ScriptDeskUseCase) run(ctx context.Context, projectID string, sc scriptContext, at int, task, note string) (string, error) {
	prompt, err := uc.prompts.GetActive(ctx, domain.RoleScriptWriter)
	if err != nil {
		return "", fmt.Errorf("load script writer prompt: %w", err)
	}
	key := sc.beats[at].unit.Key
	current := "(chưa có)"
	if s, ok := sc.scripts[key]; ok {
		current = linesText(s.Lines)
	}
	system := domain.RenderPromptTemplate(prompt.TemplateText, map[string]string{
		"topic":          sc.topic,
		"brief_language": briefLanguageName(sc.language),
		"target":         sc.targetLabel,
		"brief":          sc.brief,
		"story":          sc.story,
		"facts":          factsText(sc.facts),
		"beat":           beatText(sc.beats[at], at+1, sc.facts),
		"previous":       neighbourText(sc, at-1),
		"following":      neighbourText(sc, at+1),
		"current_lines":  current,
		"redo_note":      noteText(note),
		"task":           task,
	})
	return uc.chat(ctx, projectID, sc.model, system)
}

// writeLines turns the lines the AI wrote into stored lines numbered from next.
func writeLines(beatKey string, drafts []domain.ScriptLineDraft, next int) domain.ScriptBeat {
	out := domain.ScriptBeat{Lines: make([]domain.ScriptLine, 0, len(drafts)), NextSeq: next}
	for _, d := range drafts {
		out.Lines = append(out.Lines, domain.ScriptLine{ID: domain.ScriptLineID(beatKey, out.NextSeq), Text: d.Text, Spoken: d.Spoken, Facts: d.Facts})
		out.NextSeq++
	}
	return out
}

// writeBeat has the AI write the dialogue of the beat at index at and stores
// it as a pending unit. old is the stored dialogue being replaced, if any: its
// line numbers are not reused.
func (uc *ScriptDeskUseCase) writeBeat(ctx context.Context, projectID string, sc scriptContext, at int, task, note string, old *storedScript) (domain.ReviewUnit, error) {
	reply, err := uc.run(ctx, projectID, sc, at, task, note)
	if err != nil {
		return domain.ReviewUnit{}, err
	}
	drafts, err := domain.ParseScriptReply(reply, factKnownTo(sc.facts))
	if err != nil {
		return domain.ReviewUnit{}, err
	}
	key := sc.beats[at].unit.Key
	next := 1
	unit := domain.ReviewUnit{Status: domain.UnitPending, CreatorFields: []string{}}
	if old != nil {
		next, unit.RedoCount = old.NextSeq, old.unit.RedoCount+1
	}
	script := writeLines(key, drafts, next)
	if err := script.Validate(factKnownTo(sc.facts)); err != nil {
		return domain.ReviewUnit{}, fmt.Errorf("%w: %v", domain.ErrScriptReplyUnusable, err)
	}
	sha, err := uc.units.UnitSHA(ctx, projectID, domain.UnitKindBeat, key)
	if err != nil {
		return domain.ReviewUnit{}, err
	}
	unit.InputRef, unit.InputSHA = domain.UnitKindBeat+":"+key, sha
	return uc.saveScript(ctx, projectID, key, script, unit)
}

// Generate has the AI write the dialogue of the beats that have none, one beat
// after another, each reading the beats around it. With beatKey set it writes
// only that beat, which must have no dialogue yet. A beat whose reply cannot
// be read stores nothing and stops the run; beats already written stay.
func (uc *ScriptDeskUseCase) Generate(ctx context.Context, projectID, beatKey, note string) error {
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
	var todo []int
	for i, b := range sc.beats {
		has := len(sc.scripts[b.unit.Key].Lines) > 0
		switch {
		case beatKey == "" && !has:
			todo = append(todo, i)
		case beatKey == b.unit.Key && has:
			return ErrScriptBeatHasLines
		case beatKey == b.unit.Key:
			todo = append(todo, i)
		}
	}
	if beatKey != "" && len(todo) == 0 {
		return domain.ErrUnitNotFound
	}
	for _, at := range todo {
		unit, err := uc.writeBeat(ctx, projectID, sc, at, domain.ScriptTaskWrite(), note, nil)
		if err != nil {
			return fmt.Errorf("beat %s: %w", sc.beats[at].unit.Key, err)
		}
		s, _ := readScriptBeat(unit)
		sc.scripts[unit.Key] = storedScript{unit: unit, ScriptBeat: s}
	}
	return nil
}

// Suggest asks the AI for one replacement sentence for a flagged line (kind
// "line") or for the first line of a beat that joins badly with the beat
// before it (kind "join"). It stores nothing: the Creator applies the text
// with an ordinary edit of the line.
func (uc *ScriptDeskUseCase) Suggest(ctx context.Context, projectID, kind, lineID string) (string, error) {
	if err := uc.requireAI(); err != nil {
		return "", err
	}
	beatKey, ok := domain.ScriptLineBeat(lineID)
	if !ok {
		return "", fmt.Errorf("%w: %q không phải mã câu", domain.ErrInvalidUnitEdit, lineID)
	}
	sc, err := uc.context(ctx, projectID)
	if err != nil {
		return "", err
	}
	at := beatIndex(sc.beats, beatKey)
	script, has := sc.scripts[beatKey]
	if at < 0 || !has {
		return "", domain.ErrUnitNotFound
	}
	li := lineIndex(script.Lines, lineID)
	if li < 0 {
		return "", domain.ErrUnitNotFound
	}
	var task string
	switch kind {
	case "line":
		flag := domain.FlagLine(script.Lines[li].SpokenText())
		if flag == nil {
			return "", ErrScriptLineNotFlagged
		}
		task = domain.ScriptTaskLine(lineID, flag.Kind+": "+flag.Msg)
	case "join":
		if li != 0 || at == 0 {
			return "", fmt.Errorf("%w: chỉ câu đầu của beat sau mới có chỗ nối", domain.ErrInvalidUnitEdit)
		}
		prev := sc.scripts[sc.beats[at-1].unit.Key]
		if len(prev.Lines) == 0 {
			return "", ErrScriptLineNotFlagged
		}
		j := domain.FlagJoin(prev.Lines[len(prev.Lines)-1].SpokenText(), script.Lines[0].SpokenText())
		if j == nil {
			return "", ErrScriptLineNotFlagged
		}
		task = domain.ScriptTaskJoin(lineID, j.Msg)
	default:
		return "", fmt.Errorf("%w: kind phải là line hoặc join", domain.ErrInvalidUnitEdit)
	}
	reply, err := uc.run(ctx, projectID, sc, at, task, "")
	if err != nil {
		return "", err
	}
	return domain.ParseLineSuggestion(reply)
}

// Speak sends the lines to be read in the project's voice. A line that
// already has a file for its current wording and voice is left alone, so a
// repeat request costs nothing; the others wait for the voice service and
// show up as ready in State once it answers.
func (uc *ScriptDeskUseCase) Speak(ctx context.Context, projectID string, lineIDs []string) error {
	if uc.voices == nil || uc.dispatcher == nil {
		return ErrVoiceNotConfigured
	}
	if len(lineIDs) == 0 {
		return fmt.Errorf("%w: chưa nêu câu nào để đọc", domain.ErrInvalidUnitEdit)
	}
	scripts, err := uc.scripts(ctx, projectID)
	if err != nil {
		return err
	}
	text := map[string]string{}
	for _, s := range scripts {
		for _, l := range s.Lines {
			text[l.ID] = l.SpokenText()
		}
	}
	voiceID, lang, err := uc.voiceOf(ctx, projectID)
	if err != nil {
		return err
	}
	rows, err := uc.voices.ListVoiceLines(ctx, projectID)
	if err != nil {
		return err
	}
	byLine := map[string]domain.VoiceLine{}
	for _, r := range rows {
		byLine[r.LineID] = r
	}
	var asked []string
	for _, id := range lineIDs {
		t, ok := text[id]
		if !ok {
			return fmt.Errorf("%w: không có câu %s", domain.ErrInvalidUnitEdit, id)
		}
		if row, ok := byLine[id]; ok && audioIsCurrent(row, t, voiceID) {
			if row.Status == domain.VoiceLineReady {
				continue
			}
			if row.Status == domain.VoiceLineRunning && uc.now().Sub(row.RequestedAt) < voiceRetryAfter {
				continue
			}
		}
		line := domain.VoiceLine{
			ProjectID: projectID, LineID: id, Text: strings.TrimSpace(t), TextSHA: domain.LineTextSHA(t),
			VoiceID: voiceID, Language: string(lang), Status: domain.VoiceLinePending,
		}
		if err := uc.voices.RequestVoiceLine(ctx, line); err != nil {
			return err
		}
		asked = append(asked, id)
	}
	if len(asked) == 0 {
		return nil
	}
	if err := uc.dispatcher.DispatchDraftVoice(ctx, projectID); err != nil {
		_ = uc.voices.FailVoiceLines(ctx, projectID, asked, "không gửi được lệnh đọc giọng: "+err.Error())
		return fmt.Errorf("gửi lệnh đọc giọng: %w", err)
	}
	return nil
}
