package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"authoring/internal/domain"
)

// questionTextField is the one editable field of a research question.
const questionTextField = "text"

// briefRef names the brief as the unit AI-written questions are built from.
const briefRef = domain.UnitKindBrief + ":" + domain.BriefUnitKey

// questions loads every research question in code order.
func (uc *ResearchDeskUseCase) questions(ctx context.Context, projectID string) ([]QuestionView, error) {
	units, err := uc.units.ListUnits(ctx, projectID, domain.UnitKindResearchQuestion)
	if err != nil {
		return nil, err
	}
	out := make([]QuestionView, 0, len(units))
	for _, u := range units {
		var q domain.ResearchQuestion
		if err := json.Unmarshal(u.Content, &q); err != nil {
			return nil, fmt.Errorf("read stored question %s: %w", u.Key, err)
		}
		out = append(out, QuestionView{ID: u.Key, Text: q.Text, Mine: u.HasCreatorField(questionTextField), Stale: u.Stale})
	}
	return out, nil
}

// GenerateQuestions has the AI write the research questions from the approved
// brief. Questions the Creator wrote or edited stay unless replaceEdited; every
// other question is replaced by the new ones, which take new codes (a code is
// never reused, so a fact keeps pointing at the question it answered). note is
// the Creator's instruction for this run. A reply that cannot be read stores
// nothing and leaves the questions as they were.
func (uc *ResearchDeskUseCase) GenerateQuestions(ctx context.Context, projectID string, replaceEdited bool, note string) ([]QuestionView, error) {
	if uc.provider == nil {
		return nil, ErrLLMNotConfigured
	}
	if r, ok := uc.provider.(interface{ Ready() bool }); ok && !r.Ready() {
		return nil, ErrLLMNotConfigured
	}
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return nil, err
	}
	_, brief, err := uc.brief(ctx, projectID)
	if err != nil {
		return nil, err
	}
	project, err := uc.projects.Get(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("load project: %w", err)
	}
	topic, err := uc.units.GetAuthoringTopic(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("load topic: %w", err)
	}
	label, err := uc.durationLabel(ctx, projectID, project)
	if err != nil {
		return nil, err
	}
	models, err := uc.units.GetAuthoringModels(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("load authoring models: %w", err)
	}
	prompt, err := uc.prompts.GetActive(ctx, domain.RoleResearchQuestions)
	if err != nil {
		return nil, fmt.Errorf("load research questions prompt: %w", err)
	}
	existing, err := uc.questions(ctx, projectID)
	if err != nil {
		return nil, err
	}

	language := briefLanguage(project.ContentLanguage)
	system := domain.RenderPromptTemplate(prompt.TemplateText, map[string]string{
		"channel_identity": domain.ChannelIdentity(language),
		"topic":            topic,
		"brief":            domain.BriefSection(brief, label),
		"brief_language":   briefLanguageName(language),
		"kept_questions":   keptQuestionsText(existing, replaceEdited),
		"redo_note":        noteText(note),
	})
	started := uc.now()
	result, chatErr := uc.provider.Chat(ctx, ChatRequest{
		System:      system,
		User:        userTurnNudge,
		MaxTokens:   uc.maxOutputTokens,
		Temperature: 0.7,
		Model:       models.ModelFor("research"),
		OnProgress:  ProgressFrom(ctx),
	})
	if uc.recorder != nil {
		uc.recorder.Record(ctx, RecordFor(uc.provider.Name(), string(domain.RoleResearchQuestions), domain.UnitKindResearchQuestion, projectID, result.Usage, started, chatErr))
	}
	if chatErr != nil {
		return nil, chatErr
	}
	written, err := domain.ParseResearchQuestionsReply(result.Content)
	if err != nil {
		return nil, err
	}

	// The new questions are stored before the old ones are removed, so a failure
	// part way leaves too many questions rather than none.
	gateUnit, gate, err := uc.gate(ctx, projectID)
	if err != nil {
		return nil, err
	}
	next := nextQuestionNumber(gate, existing)
	sha, err := uc.units.UnitSHA(ctx, projectID, domain.UnitKindBrief, domain.BriefUnitKey)
	if err != nil {
		return nil, err
	}
	for _, text := range written {
		raw, err := json.Marshal(domain.ResearchQuestion{Text: text})
		if err != nil {
			return nil, err
		}
		if _, err := uc.units.SaveUnit(ctx, domain.ReviewUnit{
			ProjectID: projectID, Kind: domain.UnitKindResearchQuestion, Key: domain.QuestionID(next),
			Status: domain.UnitPending, Content: raw, InputRef: briefRef, InputSHA: sha,
		}); err != nil {
			return nil, err
		}
		next++
	}
	gate.NextQuestion = next
	if _, err := uc.saveGate(ctx, gateUnit, gate); err != nil {
		return nil, err
	}
	for _, q := range existing {
		if q.Mine && !replaceEdited {
			continue
		}
		if err := uc.units.DeleteUnit(ctx, projectID, domain.UnitKindResearchQuestion, q.ID); err != nil {
			return nil, err
		}
	}
	return uc.questions(ctx, projectID)
}

// nextQuestionNumber is the number the next question takes: past the gate's
// counter and past every question that exists.
func nextQuestionNumber(gate domain.ResearchGate, existing []QuestionView) int {
	next := gate.NextQuestion
	if next < 1 {
		next = 1
	}
	for _, q := range existing {
		if n, ok := domain.QuestionNumber(q.ID); ok && n >= next {
			next = n + 1
		}
	}
	return next
}

// keptQuestionsText lists the questions the AI must not repeat: the Creator's
// own when they stay, or none when they are being replaced too.
func keptQuestionsText(existing []QuestionView, replaceEdited bool) string {
	if replaceEdited {
		return "(không có)"
	}
	var lines []string
	for _, q := range existing {
		if q.Mine {
			lines = append(lines, "- "+q.ID+": "+q.Text)
		}
	}
	if len(lines) == 0 {
		return "(không có)"
	}
	return strings.Join(lines, "\n")
}

// AddQuestion adds a question the Creator wrote.
func (uc *ResearchDeskUseCase) AddQuestion(ctx context.Context, projectID, text string) (QuestionView, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return QuestionView{}, fmt.Errorf("%w: câu hỏi không được trống", domain.ErrInvalidUnitEdit)
	}
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return QuestionView{}, err
	}
	existing, err := uc.questions(ctx, projectID)
	if err != nil {
		return QuestionView{}, err
	}
	gateUnit, gate, err := uc.gate(ctx, projectID)
	if err != nil {
		return QuestionView{}, err
	}
	next := nextQuestionNumber(gate, existing)
	raw, err := json.Marshal(domain.ResearchQuestion{Text: text})
	if err != nil {
		return QuestionView{}, err
	}
	unit, err := uc.units.SaveUnit(ctx, domain.ReviewUnit{
		ProjectID: projectID, Kind: domain.UnitKindResearchQuestion, Key: domain.QuestionID(next),
		Status: domain.UnitPending, Content: raw, CreatorFields: []string{questionTextField},
	})
	if err != nil {
		return QuestionView{}, err
	}
	gate.NextQuestion = next + 1
	if _, err := uc.saveGate(ctx, gateUnit, gate); err != nil {
		return QuestionView{}, err
	}
	return QuestionView{ID: unit.Key, Text: text, Mine: true}, nil
}

// DeleteQuestion removes a question. The facts that answered it stay.
func (uc *ResearchDeskUseCase) DeleteQuestion(ctx context.Context, projectID, id string) error {
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return err
	}
	if _, err := uc.units.GetUnit(ctx, projectID, domain.UnitKindResearchQuestion, id); err != nil {
		return err
	}
	return uc.units.DeleteUnit(ctx, projectID, domain.UnitKindResearchQuestion, id)
}

// questionKind makes a research question editable through the generic unit
// API (UnitKind for domain.UnitKindResearchQuestion).
type questionKind struct{ desk *ResearchDeskUseCase }

// Edit stores the Creator's wording of a question. The question becomes the
// Creator's: a regeneration keeps it, and it no longer counts as built from the
// brief.
func (k questionKind) Edit(ctx context.Context, projectID, key string, fields map[string]string) (domain.ReviewUnit, error) {
	for name := range fields {
		if name != questionTextField {
			return domain.ReviewUnit{}, fmt.Errorf("%w: câu hỏi không có ô %q", domain.ErrInvalidUnitEdit, name)
		}
	}
	text, ok := fields[questionTextField]
	text = strings.TrimSpace(text)
	if !ok || text == "" {
		return domain.ReviewUnit{}, fmt.Errorf("%w: câu hỏi không được trống", domain.ErrInvalidUnitEdit)
	}
	unit, err := k.desk.units.GetUnit(ctx, projectID, domain.UnitKindResearchQuestion, key)
	if err != nil {
		return domain.ReviewUnit{}, err
	}
	var current domain.ResearchQuestion
	if err := json.Unmarshal(unit.Content, &current); err != nil {
		return domain.ReviewUnit{}, fmt.Errorf("read stored question: %w", err)
	}
	if current.Text == text && unit.HasCreatorField(questionTextField) {
		return unit, nil
	}
	if unit.Content, err = json.Marshal(domain.ResearchQuestion{Text: text}); err != nil {
		return domain.ReviewUnit{}, err
	}
	unit.CreatorFields = []string{questionTextField}
	unit.InputRef, unit.InputSHA = "", ""
	return k.desk.units.SaveUnit(ctx, unit)
}

// Redo is not offered for a single question: the AI writes the whole list.
func (questionKind) Redo(context.Context, string, string, string) (domain.ReviewUnit, error) {
	return domain.ReviewUnit{}, fmt.Errorf("%w: câu hỏi không làm lại riêng, dùng Sinh lại từ brief", domain.ErrInvalidUnitEdit)
}

// Impact is empty: nothing is built from a question.
func (questionKind) Impact(context.Context, string, string) ([]string, error) {
	return []string{}, nil
}

// CheckApprovable accepts any question; questions are not approved.
func (questionKind) CheckApprovable(context.Context, domain.ReviewUnit) error { return nil }

// QuestionUnitKind returns the research question's UnitKind.
func (uc *ResearchDeskUseCase) QuestionUnitKind() UnitKind { return questionKind{desk: uc} }
