package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"authoring/internal/domain"
)

// ErrIdeaIncomplete is returned when an idea with empty fields is approved.
var ErrIdeaIncomplete = errors.New("ý tưởng còn ô trống")

// ErrBriefNeedsIdea is returned when a brief is written before the idea is
// approved: the brief is built from the approved idea.
var ErrBriefNeedsIdea = errors.New("cần duyệt ý tưởng trước khi viết brief")

// IdeaState is everything the idea screen shows.
type IdeaState struct {
	// Unit is nil until an idea was written or typed.
	Unit *domain.ReviewUnit
}

// IdeaUseCase backs the idea step: the AI writing the idea from the topic and
// the idea as a review unit (UnitKind for domain.UnitKindIdea).
type IdeaUseCase struct {
	units           ReviewUnitPort
	store           BriefStorePort
	projects        ProjectReaderPort
	locks           AuthoringLockPort
	prompts         PromptActivePort
	provider        LLMProviderPort
	recorder        *LLMUsageRecorder
	maxOutputTokens int
	now             func() time.Time
}

// NewIdeaUseCase wires the use case. provider may be nil when no language
// model is configured: the Creator then writes the idea by hand.
func NewIdeaUseCase(
	units ReviewUnitPort, store BriefStorePort, projects ProjectReaderPort, locks AuthoringLockPort,
	prompts PromptActivePort, provider LLMProviderPort, recorder *LLMUsageRecorder, maxOutputTokens int,
) *IdeaUseCase {
	return &IdeaUseCase{
		units: units, store: store, projects: projects, locks: locks, prompts: prompts,
		provider: provider, recorder: recorder, maxOutputTokens: maxOutputTokens, now: time.Now,
	}
}

// State returns the idea screen's data.
func (uc *IdeaUseCase) State(ctx context.Context, projectID string) (IdeaState, error) {
	unit, err := uc.units.GetUnit(ctx, projectID, domain.UnitKindIdea, domain.IdeaUnitKey)
	switch {
	case err == nil:
		return IdeaState{Unit: &unit}, nil
	case errors.Is(err, domain.ErrUnitNotFound):
		return IdeaState{}, nil
	}
	return IdeaState{}, err
}

// Generate has the AI write the idea from the topic, keeping every field the
// Creator wrote; note is the Creator's instruction for this run ("" for the
// first run). The idea goes to review. A reply that cannot be read, or that
// misses a field, is an error and nothing is stored.
func (uc *IdeaUseCase) Generate(ctx context.Context, projectID, note string) (domain.ReviewUnit, error) {
	if uc.provider == nil {
		return domain.ReviewUnit{}, ErrLLMNotConfigured
	}
	if r, ok := uc.provider.(interface{ Ready() bool }); ok && !r.Ready() {
		return domain.ReviewUnit{}, ErrLLMNotConfigured
	}
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return domain.ReviewUnit{}, err
	}
	project, err := uc.projects.Get(ctx, projectID)
	if err != nil {
		return domain.ReviewUnit{}, fmt.Errorf("load project: %w", err)
	}
	topic, err := uc.store.GetAuthoringTopic(ctx, projectID)
	if err != nil {
		return domain.ReviewUnit{}, fmt.Errorf("load topic: %w", err)
	}
	if strings.TrimSpace(topic) == "" {
		return domain.ReviewUnit{}, fmt.Errorf("%w: project chưa có đề tài", domain.ErrInvalidWizardInput)
	}
	models, err := uc.store.GetAuthoringModels(ctx, projectID)
	if err != nil {
		return domain.ReviewUnit{}, fmt.Errorf("load authoring models: %w", err)
	}

	unit, current, exists, err := uc.current(ctx, projectID)
	if err != nil {
		return domain.ReviewUnit{}, err
	}
	prompt, err := uc.prompts.GetActive(ctx, domain.RoleIdea)
	if err != nil {
		return domain.ReviewUnit{}, fmt.Errorf("load idea prompt: %w", err)
	}
	language := briefLanguage(project.ContentLanguage)
	system := domain.RenderPromptTemplate(prompt.TemplateText, map[string]string{
		"topic":         topic,
		"idea_language": briefLanguageName(language),
		"current_idea":  currentJSONText(current, exists),
		"kept_fields":   keptIdeaFieldsText(current, unit.CreatorFields),
		"redo_note":     noteText(note),
	})

	started := uc.now()
	result, chatErr := uc.provider.Chat(ctx, ChatRequest{
		System:      system,
		User:        userTurnNudge,
		MaxTokens:   uc.maxOutputTokens,
		Temperature: 0.7,
		Model:       models.ModelFor("brief"),
		OnProgress:  ProgressFrom(ctx),
	})
	if uc.recorder != nil {
		uc.recorder.Record(ctx, RecordFor(uc.provider.Name(), string(domain.RoleIdea), domain.UnitKindIdea, projectID, result.Usage, started, chatErr))
	}
	if chatErr != nil {
		return domain.ReviewUnit{}, chatErr
	}
	written, err := domain.ParseIdeaReply(result.Content)
	if err != nil {
		return domain.ReviewUnit{}, err
	}
	written = domain.KeepIdeaCreatorFields(written, current, unit.CreatorFields)

	if exists {
		unit.RedoCount++
	}
	unit.Status = domain.UnitPending
	if unit.Content, err = json.Marshal(written); err != nil {
		return domain.ReviewUnit{}, err
	}
	return uc.units.SaveUnit(ctx, unit)
}

// current loads the idea unit, or a new empty one when none exists yet.
func (uc *IdeaUseCase) current(ctx context.Context, projectID string) (domain.ReviewUnit, domain.Idea, bool, error) {
	unit, idea, exists, err := loadIdea(ctx, uc.units, projectID)
	if err != nil {
		return domain.ReviewUnit{}, domain.Idea{}, false, err
	}
	if !exists {
		unit = domain.ReviewUnit{ProjectID: projectID, Kind: domain.UnitKindIdea, Key: domain.IdeaUnitKey, Status: domain.UnitPending}
	}
	return unit, idea, exists, nil
}

// loadIdea reads the idea unit and its content; exists is false when the
// project has none yet.
func loadIdea(ctx context.Context, units ReviewUnitPort, projectID string) (domain.ReviewUnit, domain.Idea, bool, error) {
	unit, err := units.GetUnit(ctx, projectID, domain.UnitKindIdea, domain.IdeaUnitKey)
	if errors.Is(err, domain.ErrUnitNotFound) {
		return domain.ReviewUnit{}, domain.Idea{}, false, nil
	}
	if err != nil {
		return domain.ReviewUnit{}, domain.Idea{}, false, err
	}
	var idea domain.Idea
	if err := json.Unmarshal(unit.Content, &idea); err != nil {
		return domain.ReviewUnit{}, domain.Idea{}, false, fmt.Errorf("read stored idea: %w", err)
	}
	return unit, idea, true, nil
}

// Edit stores the Creator's hand edit of idea fields. A field whose value
// changes becomes the Creator's; a field emptied goes back to the AI, which
// writes it on the next redo. An approved idea stays approved (the Creator
// wrote the change) unless the edit empties a field. Editing before any idea
// exists creates one, for a Creator working without a language model.
func (uc *IdeaUseCase) Edit(ctx context.Context, projectID, key string, fields map[string]string) (domain.ReviewUnit, error) {
	if key != domain.IdeaUnitKey {
		return domain.ReviewUnit{}, domain.ErrUnitNotFound
	}
	if len(fields) == 0 {
		return domain.ReviewUnit{}, fmt.Errorf("%w: không có ô nào để lưu", domain.ErrInvalidUnitEdit)
	}
	for name := range fields {
		if !domain.IsIdeaField(name) {
			return domain.ReviewUnit{}, fmt.Errorf("%w: ý tưởng không có ô %q", domain.ErrInvalidUnitEdit, name)
		}
	}
	unit, idea, _, err := uc.current(ctx, projectID)
	if err != nil {
		return domain.ReviewUnit{}, err
	}
	for _, name := range domain.IdeaFields {
		value, ok := fields[name]
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		if value == idea.Field(name) {
			continue
		}
		idea = idea.WithField(name, value)
		switch {
		case value == "":
			unit.CreatorFields = withoutField(unit.CreatorFields, name)
		case !unit.HasCreatorField(name):
			unit.CreatorFields = append(unit.CreatorFields, name)
		}
	}
	if unit.Status == domain.UnitApproved && len(idea.MissingFields()) > 0 {
		unit.Status = domain.UnitPending
	}
	if unit.Content, err = json.Marshal(idea); err != nil {
		return domain.ReviewUnit{}, err
	}
	return uc.units.SaveUnit(ctx, unit)
}

// Redo has the AI write the idea again with the Creator's note.
func (uc *IdeaUseCase) Redo(ctx context.Context, projectID, key, note string) (domain.ReviewUnit, error) {
	if key != domain.IdeaUnitKey {
		return domain.ReviewUnit{}, domain.ErrUnitNotFound
	}
	return uc.Generate(ctx, projectID, note)
}

// Impact lists what a change to the idea sends back: the brief written from
// it, and everything the brief sends back in turn.
func (uc *IdeaUseCase) Impact(ctx context.Context, projectID, key string) ([]string, error) {
	if key != domain.IdeaUnitKey {
		return nil, domain.ErrUnitNotFound
	}
	items := []string{}
	if _, err := uc.units.GetUnit(ctx, projectID, domain.UnitKindBrief, domain.BriefUnitKey); err == nil {
		items = append(items, "Creative brief: cần xem lại theo ý tưởng mới")
	} else if !errors.Is(err, domain.ErrUnitNotFound) {
		return nil, err
	}
	downstream, err := briefDownstream(ctx, uc.units, uc.store, projectID)
	if err != nil {
		return nil, err
	}
	return append(items, downstream...), nil
}

// CheckApprovable refuses an idea that still has empty fields.
func (uc *IdeaUseCase) CheckApprovable(_ context.Context, unit domain.ReviewUnit) error {
	var idea domain.Idea
	if err := json.Unmarshal(unit.Content, &idea); err != nil {
		return fmt.Errorf("read stored idea: %w", err)
	}
	if len(idea.MissingFields()) > 0 {
		return fmt.Errorf("%w: %s", ErrIdeaIncomplete, idea.MissingFieldLabels())
	}
	return nil
}

// currentJSONText renders a stored unit's content for a prompt, or a note that
// there is none yet.
func currentJSONText(content any, exists bool) string {
	if !exists {
		return "(chưa có)"
	}
	// Marshalling a struct of strings cannot fail.
	raw, _ := json.Marshal(content)
	return string(raw)
}

func keptIdeaFieldsText(idea domain.Idea, creatorFields []string) string {
	if len(creatorFields) == 0 {
		return "(không có)"
	}
	var lines []string
	for _, f := range domain.IdeaFields {
		for _, kept := range creatorFields {
			if kept == f {
				lines = append(lines, "- "+f+": "+idea.Field(f))
			}
		}
	}
	return strings.Join(lines, "\n")
}
