package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"authoring/internal/domain"
)

// lockModelStep is the key of the model the Creator chose for this step: like
// the script step it keeps the "story" slot.
const lockModelStep = "story"

// requireAI fails unless a language model is configured.
func (uc *ScriptLockDeskUseCase) requireAI() error {
	if uc.provider == nil {
		return ErrLLMNotConfigured
	}
	if r, ok := uc.provider.(interface{ Ready() bool }); ok && !r.Ready() {
		return ErrLLMNotConfigured
	}
	return nil
}

// chat sends one system prompt and records the usage under role.
func (uc *ScriptLockDeskUseCase) chat(ctx context.Context, role domain.PromptRole, unitKind, projectID, model, system string) (string, error) {
	started := uc.now()
	result, err := uc.provider.Chat(ctx, ChatRequest{
		System: system, User: userTurnNudge, MaxTokens: uc.maxOutputTokens, Temperature: 0.4,
		Model: model, OnProgress: ProgressFrom(ctx),
	})
	if uc.recorder != nil {
		uc.recorder.Record(ctx, RecordFor(uc.provider.Name(), string(role), unitKind, projectID, result.Usage, started, err))
	}
	if err != nil {
		return "", err
	}
	return result.Content, nil
}

// lockContext is what both AI calls of the step read.
type lockContext struct {
	beatContext
	beats   []storedBeat
	scripts map[string]storedScript
}

// context gathers the inputs of an AI call and refuses when the script is not
// fully approved yet.
func (uc *ScriptLockDeskUseCase) context(ctx context.Context, projectID string) (lockContext, error) {
	var lc lockContext
	var err error
	if lc.beats, _, _, err = uc.beats.beats(ctx, projectID); err != nil {
		return lc, err
	}
	if lc.beatContext, err = uc.beats.context(ctx, projectID); err != nil {
		return lc, err
	}
	if lc.scripts, err = uc.script.scripts(ctx, projectID); err != nil {
		return lc, err
	}
	for _, b := range lc.beats {
		s, ok := lc.scripts[b.unit.Key]
		if !ok || len(s.Lines) == 0 || s.unit.Status != domain.UnitApproved {
			return lc, fmt.Errorf("%w: %w", domain.ErrInvalidStatus, ErrLockNeedsScript)
		}
	}
	models, err := uc.units.GetAuthoringModels(ctx, projectID)
	if err != nil {
		return lc, fmt.Errorf("load authoring models: %w", err)
	}
	lc.model = models.ModelFor(lockModelStep)
	return lc, nil
}

// scriptText shows the whole script to the model, beat by beat, each line with its id and facts.
func (lc lockContext) scriptText() string {
	var sb strings.Builder
	for i, b := range lc.beats {
		fmt.Fprintf(&sb, "%s (B%d) · %s · %s · mục tiêu: %s\n", b.unit.Key, i+1, b.Fn, b.Title, b.Goal)
		for _, l := range lc.scripts[b.unit.Key].Lines {
			facts := ""
			if len(l.Facts) > 0 {
				facts = " [" + strings.Join(l.Facts, ", ") + "]"
			}
			fmt.Fprintf(&sb, "  %s: %s%s\n", l.ID, l.Text, facts)
		}
	}
	return strings.TrimRight(sb.String(), "\n")
}

func (lc lockContext) lineKnown() func(string) bool {
	set := map[string]bool{}
	for _, s := range lc.scripts {
		for _, l := range s.Lines {
			set[l.ID] = true
		}
	}
	return func(id string) bool { return set[id] }
}

func (lc lockContext) beatKnown() func(string) bool {
	set := map[string]bool{}
	for _, b := range lc.beats {
		set[b.unit.Key] = true
	}
	return func(key string) bool { return set[key] }
}

func (lc lockContext) lineSHAParts() [][]domain.ScriptLine {
	out := make([][]domain.ScriptLine, 0, len(lc.beats))
	for _, b := range lc.beats {
		out = append(out, lc.scripts[b.unit.Key].Lines)
	}
	return out
}

func previousFindingsText(r domain.ScriptReview) string {
	var rows []string
	for _, f := range r.Findings {
		if r.StateOf(f) == domain.FindingOpen || r.StateOf(f) == domain.FindingFixed {
			rows = append(rows, fmt.Sprintf("%s [%s] %s — %s", f.ID, f.Group, f.Title, f.Detail))
		}
	}
	if len(rows) == 0 {
		return "(chưa có lần rà soát trước)"
	}
	return strings.Join(rows, "\n")
}

// Review has the AI read the whole script in one call and report what the
// Creator should deal with. On a second run it compares the script as it
// stands with the findings it made before. A reply that cannot be read stores
// nothing.
func (uc *ScriptLockDeskUseCase) Review(ctx context.Context, projectID string) error {
	if err := uc.requireAI(); err != nil {
		return err
	}
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return err
	}
	lc, err := uc.context(ctx, projectID)
	if err != nil {
		return err
	}
	prompt, err := uc.prompts.GetActive(ctx, domain.RoleScriptReview)
	if err != nil {
		return fmt.Errorf("load script review prompt: %w", err)
	}
	unit, prev, err := uc.review(ctx, projectID)
	if err != nil {
		return err
	}
	system := domain.RenderPromptTemplate(prompt.TemplateText, map[string]string{
		"topic": lc.topic, "brief_language": briefLanguageName(lc.language), "target": lc.targetLabel,
		"brief": lc.brief, "story": lc.story, "facts": factsText(lc.facts),
		"script": lc.scriptText(), "previous_findings": previousFindingsText(prev),
	})
	earlier := map[string]bool{}
	for _, f := range prev.Findings {
		earlier[f.ID] = true
	}
	var parsed domain.ReviewReply
	err = askUntilUsable(ctx, "script review", projectID, domain.ErrReviewReplyUnusable, func() error {
		reply, err := uc.chat(ctx, domain.RoleScriptReview, domain.UnitKindScriptReview, projectID, lc.model, system)
		if err != nil {
			return err
		}
		parsed, err = domain.ParseReviewReply(reply, lc.lineKnown(), lc.beatKnown(), func(id string) bool { return earlier[id] })
		if err != nil {
			logUnusableReply("script review", projectID, reply, err)
		}
		return err
	})
	if err != nil {
		return err
	}
	next := domain.ApplyReview(prev, parsed, domain.ScriptSHA(lc.lineSHAParts()))
	return uc.saveReview(ctx, projectID, unit, next)
}

func noteSectionsText(n domain.HandoffNote) string {
	var sb strings.Builder
	for _, s := range domain.HandoffSections {
		items := n.Section(s.Key)
		if len(items) == 0 {
			continue
		}
		sb.WriteString(s.Key + " (" + s.Label + "):\n")
		for _, it := range items {
			sb.WriteString("- " + it + "\n")
		}
	}
	if sb.Len() == 0 {
		return "(chưa có)"
	}
	return strings.TrimRight(sb.String(), "\n")
}

func glossaryText(g domain.Glossary) string {
	if len(g.Terms) == 0 {
		return "(chưa có)"
	}
	rows := make([]string, 0, len(g.Terms))
	for _, t := range g.Terms {
		rows = append(rows, fmt.Sprintf("%s → nói là %q, vẽ là %q, tránh %q", t.Term, t.Say, t.Draw, t.Avoid))
	}
	return strings.Join(rows, "\n")
}

// writeHandoff has the AI write the handoff note of the beat at index at and
// stores it as a pending unit; terms it proposes join the glossary. old is the
// note being replaced, if any.
func (uc *ScriptLockDeskUseCase) writeHandoff(ctx context.Context, projectID string, lc lockContext, at int, note string, old *storedHandoff) error {
	prompt, err := uc.prompts.GetActive(ctx, domain.RoleScriptHandoff)
	if err != nil {
		return fmt.Errorf("load script handoff prompt: %w", err)
	}
	glossary, err := uc.glossary(ctx, projectID)
	if err != nil {
		return err
	}
	b := lc.beats[at]
	current := "(chưa có)"
	if old != nil {
		current = noteSectionsText(old.HandoffNote)
	}
	system := domain.RenderPromptTemplate(prompt.TemplateText, map[string]string{
		"topic": lc.topic, "brief_language": briefLanguageName(lc.language), "brief": lc.brief,
		"facts": factsText(lc.facts), "beat": beatText(b, at+1, lc.facts),
		"lines": linesText(lc.scripts[b.unit.Key].Lines), "current_note": current,
		"redo_note": noteText(note), "glossary": glossaryText(glossary),
	})
	var parsed domain.HandoffReply
	err = askUntilUsable(ctx, "script handoff", projectID, domain.ErrHandoffReplyUnusable, func() error {
		reply, err := uc.chat(ctx, domain.RoleScriptHandoff, domain.UnitKindHandoffBeat, projectID, lc.model, system)
		if err != nil {
			return err
		}
		parsed, err = domain.ParseHandoffReply(reply)
		if err != nil {
			logUnusableReply("script handoff", projectID, reply, err)
		}
		return err
	})
	if err != nil {
		return err
	}
	unit := domain.ReviewUnit{Status: domain.UnitPending, CreatorFields: []string{}}
	if old != nil {
		unit.RedoCount = old.unit.RedoCount + 1
	}
	if _, err := uc.saveHandoff(ctx, projectID, b.unit.Key, parsed.Note, unit); err != nil {
		return err
	}
	if len(parsed.Terms) > 0 {
		return uc.saveGlossary(ctx, projectID, glossary.Merge(parsed.Terms))
	}
	return nil
}

// GenerateHandoff has the AI write the handoff note of every beat that has
// none, one beat after another. With beatKey set it writes only that beat,
// which must have no note yet. A beat whose reply cannot be read stores
// nothing and stops the run; notes already written stay.
func (uc *ScriptLockDeskUseCase) GenerateHandoff(ctx context.Context, projectID, beatKey string) error {
	if err := uc.requireAI(); err != nil {
		return err
	}
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return err
	}
	lc, err := uc.context(ctx, projectID)
	if err != nil {
		return err
	}
	notes, err := uc.handoffs(ctx, projectID)
	if err != nil {
		return err
	}
	var todo []int
	for i, b := range lc.beats {
		_, has := notes[b.unit.Key]
		switch {
		case beatKey == "" && !has:
			todo = append(todo, i)
		case beatKey == b.unit.Key && has:
			return ErrHandoffExists
		case beatKey == b.unit.Key:
			todo = append(todo, i)
		}
	}
	if beatKey != "" && len(todo) == 0 {
		return domain.ErrUnitNotFound
	}
	for _, at := range todo {
		if err := uc.writeHandoff(ctx, projectID, lc, at, "", nil); err != nil {
			return fmt.Errorf("beat %s: %w", lc.beats[at].unit.Key, err)
		}
	}
	return nil
}

// logUnusableReply records why a model reply was refused and the ends of the
// reply, so a failure the Creator reports can be told from a truncated reply.
func logUnusableReply(what, projectID, reply string, err error) {
	r := []rune(reply)
	head, tail := string(r[:min(len(r), 300)]), ""
	if len(r) > 300 {
		tail = string(r[max(300, len(r)-300):])
	}
	slog.Warn("model reply refused", "what", what, "project_id", projectID, "reason", err.Error(), "reply_runes", len(r), "head", head, "tail", tail)
}

// maxReplyAttempts is how many times a model is asked for a reply it can be
// read from before the run fails: a small model now and then breaks the JSON.
const maxReplyAttempts = 3

// askUntilUsable runs ask until it succeeds, asking again only when the reply
// could not be read (unusable); any other error, or the last attempt's, is returned.
func askUntilUsable(ctx context.Context, what, projectID string, unusable error, ask func() error) error {
	var err error
	for attempt := 1; attempt <= maxReplyAttempts; attempt++ {
		if err = ask(); err == nil || !errors.Is(err, unusable) || ctx.Err() != nil {
			return err
		}
		slog.Warn("asking the model again", "what", what, "project_id", projectID, "attempt", attempt)
	}
	return err
}
