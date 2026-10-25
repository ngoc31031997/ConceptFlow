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

// ProjectIdeaStorePort keeps a project's idea and the AI's review of it.
type ProjectIdeaStorePort interface {
	// GetProjectIdea returns the idea, or domain.ErrIdeaNotFound.
	GetProjectIdea(ctx context.Context, projectID string) (domain.StoredIdea, error)
	SaveProjectIdea(ctx context.Context, idea domain.StoredIdea) error
	// GetIdeaReview returns the latest review, or domain.ErrIdeaNotFound.
	GetIdeaReview(ctx context.Context, projectID string) (domain.IdeaReview, error)
	SaveIdeaReview(ctx context.Context, projectID string, review domain.IdeaReview) error
}

// IdeaInput is what the Creator typed on the first step and the language the
// video is written in; the orchestrator keeps both with the project.
type IdeaInput struct {
	Subject  string
	Language domain.ContentLanguage
}

// IdeaInputPort reads IdeaInput from the orchestrator.
type IdeaInputPort interface {
	IdeaInput(ctx context.Context, projectID string) (IdeaInput, error)
}

// ProjectIdeaState is everything the idea screen shows.
type ProjectIdeaState struct {
	// Idea is nil until the AI or the Creator wrote one.
	Idea *domain.StoredIdea
	// Subject and Language are the project's input.
	Subject  string
	Language domain.ContentLanguage
	// InputChanged is true when the text or the language changed after the idea
	// was written, so the idea no longer matches what the Creator typed.
	InputChanged bool
	Review       *domain.IdeaReview
	// ReviewStale is true when the idea changed after the review read it.
	ReviewStale bool
}

// ProjectIdeaUseCase backs the first step: the AI writing the idea from the
// Creator's text, the Creator's edits and approval, and the AI's review of the
// idea before approval.
type ProjectIdeaUseCase struct {
	store           ProjectIdeaStorePort
	input           IdeaInputPort
	models          AuthoringModelsReaderPort
	locks           AuthoringLockPort
	prompts         PromptActivePort
	provider        LLMProviderPort
	maxOutputTokens int
	now             func() time.Time
}

// NewProjectIdeaUseCase wires the use case. provider may be nil when no language
// model is configured: the Creator then writes the idea by hand.
func NewProjectIdeaUseCase(
	store ProjectIdeaStorePort, input IdeaInputPort, models AuthoringModelsReaderPort, locks AuthoringLockPort,
	prompts PromptActivePort, provider LLMProviderPort, maxOutputTokens int,
) *ProjectIdeaUseCase {
	return &ProjectIdeaUseCase{
		store: store, input: input, models: models, locks: locks, prompts: prompts,
		provider: provider, maxOutputTokens: maxOutputTokens, now: time.Now,
	}
}

// State returns the idea screen's data.
func (uc *ProjectIdeaUseCase) State(ctx context.Context, projectID string) (ProjectIdeaState, error) {
	in, err := uc.input.IdeaInput(ctx, projectID)
	if err != nil {
		return ProjectIdeaState{}, err
	}
	state := ProjectIdeaState{Subject: in.Subject, Language: in.Language}
	stored, err := uc.store.GetProjectIdea(ctx, projectID)
	switch {
	case err == nil:
		state.Idea = &stored
		state.InputChanged = stored.InputHash != domain.HashInput(in.Subject) || stored.Language != string(in.Language)
	case !errors.Is(err, domain.ErrIdeaNotFound):
		return ProjectIdeaState{}, err
	}
	review, err := uc.store.GetIdeaReview(ctx, projectID)
	switch {
	case err == nil:
		state.Review = &review
		state.ReviewStale = state.Idea == nil || review.IdeaHash != state.Idea.Idea.Hash()
	case !errors.Is(err, domain.ErrIdeaNotFound):
		return ProjectIdeaState{}, err
	}
	return state, nil
}

// ApprovedIdea returns the idea the Creator approved, for the steps after this
// one; ok is false when there is none.
func (uc *ProjectIdeaUseCase) ApprovedIdea(ctx context.Context, projectID string) (domain.ProjectIdea, bool, error) {
	stored, err := uc.store.GetProjectIdea(ctx, projectID)
	if errors.Is(err, domain.ErrIdeaNotFound) {
		return domain.ProjectIdea{}, false, nil
	}
	if err != nil {
		return domain.ProjectIdea{}, false, err
	}
	if stored.Status != domain.IdeaApproved {
		return domain.ProjectIdea{}, false, nil
	}
	return stored.Idea, true, nil
}

// requireAI refuses when no language model is ready.
func (uc *ProjectIdeaUseCase) requireAI() error {
	if uc.provider == nil {
		return ErrLLMNotConfigured
	}
	if r, ok := uc.provider.(interface{ Ready() bool }); ok && !r.Ready() {
		return ErrLLMNotConfigured
	}
	return nil
}

// Generate has the AI write the idea from what the Creator typed, keeping every
// field the Creator wrote; note is the Creator's instruction for this run ("" for
// the first run). The idea goes back to waiting for approval. A reply that
// cannot be read, or that misses a field, is an error and nothing is stored.
func (uc *ProjectIdeaUseCase) Generate(ctx context.Context, projectID, note string) (domain.StoredIdea, error) {
	if err := uc.requireAI(); err != nil {
		return domain.StoredIdea{}, err
	}
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return domain.StoredIdea{}, err
	}
	in, err := uc.input.IdeaInput(ctx, projectID)
	if err != nil {
		return domain.StoredIdea{}, fmt.Errorf("load project input: %w", err)
	}
	// The orchestrator already cleaned what it keeps; text saved before that rule
	// existed is cleaned here, so no straight quote reaches the prompt.
	in.Subject = domain.SanitizeIdeaText(in.Subject)
	note = domain.SanitizeIdeaText(note)
	if in.Subject == "" {
		return domain.StoredIdea{}, fmt.Errorf("%w: %w", domain.ErrInvalidWizardInput, domain.ErrIdeaNoInput)
	}
	models, err := uc.models.GetAuthoringModels(ctx, projectID)
	if err != nil {
		return domain.StoredIdea{}, fmt.Errorf("load authoring models: %w", err)
	}
	current, exists, err := uc.current(ctx, projectID)
	if err != nil {
		return domain.StoredIdea{}, err
	}
	prompt, err := uc.prompts.GetActive(ctx, domain.RoleProjectIdea)
	if err != nil {
		return domain.StoredIdea{}, fmt.Errorf("load idea prompt: %w", err)
	}
	language := briefLanguageName(briefLanguage(in.Language))
	system := domain.RenderPromptTemplate(prompt.TemplateText, map[string]string{
		"topic":         in.Subject,
		"idea_language": language,
		"current_idea":  currentIdeaText(current, exists),
		"kept_fields":   keptProjectIdeaFieldsText(current),
		"redo_note":     noteText(note),
	})

	result, chatErr := uc.provider.Chat(ctx, ChatRequest{
		System:      system,
		User:        userTurnNudge,
		MaxTokens:   uc.maxOutputTokens,
		Temperature: 0.7,
		Model:       models.ModelFor("brief"),
		OnProgress:  ProgressFrom(ctx),
		ProjectID:   projectID,
		Role:        string(domain.RoleProjectIdea),
		Step:        "idea",
		ExpectJSON:  true,
	})
	if chatErr != nil {
		return domain.StoredIdea{}, chatErr
	}
	written, err := domain.ParseProjectIdeaReply(string(result.JSON))
	if err != nil {
		return domain.StoredIdea{}, err
	}
	written = domain.KeepProjectIdeaCreatorFields(written, current.Idea, current.CreatorFields).Normalized()
	if err := written.Validate(); err != nil {
		return domain.StoredIdea{}, fmt.Errorf("%w: %v", domain.ErrProjectIdeaReplyUnusable, err)
	}

	if exists {
		current.RedoCount++
	}
	current.Idea = written
	current.Status = domain.IdeaPending
	current.ApprovedAt = nil
	current.Language = string(in.Language)
	current.InputHash = domain.HashInput(in.Subject)
	current.UpdatedAt = uc.now()
	if err := uc.store.SaveProjectIdea(ctx, current); err != nil {
		return domain.StoredIdea{}, err
	}
	return current, nil
}

// current loads the stored idea, or a new empty one when there is none yet.
func (uc *ProjectIdeaUseCase) current(ctx context.Context, projectID string) (domain.StoredIdea, bool, error) {
	stored, err := uc.store.GetProjectIdea(ctx, projectID)
	if errors.Is(err, domain.ErrIdeaNotFound) {
		return domain.StoredIdea{ProjectID: projectID, Status: domain.IdeaPending, CreatorFields: []string{}}, false, nil
	}
	if err != nil {
		return domain.StoredIdea{}, false, err
	}
	return stored, true, nil
}

// IdeaPatch is the Creator's hand edit; a nil field is left alone.
type IdeaPatch struct {
	CentralQuestion *string
	CuriosityPoints *[]string
	Scope           *string
}

// Edit stores the Creator's hand edit. A field whose value changes becomes the
// Creator's; a field emptied goes back to the AI, which writes it on the next
// redo. Any change sends an approved idea back to waiting for approval, since
// the steps after read only the approved idea. Editing before any idea exists
// creates one, for a Creator working without a language model.
func (uc *ProjectIdeaUseCase) Edit(ctx context.Context, projectID string, patch IdeaPatch) (domain.StoredIdea, error) {
	if patch.CentralQuestion == nil && patch.CuriosityPoints == nil && patch.Scope == nil {
		return domain.StoredIdea{}, fmt.Errorf("%w: không có ô nào để lưu", domain.ErrInvalidUnitEdit)
	}
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return domain.StoredIdea{}, err
	}
	stored, exists, err := uc.current(ctx, projectID)
	if err != nil {
		return domain.StoredIdea{}, err
	}
	if !exists {
		in, err := uc.input.IdeaInput(ctx, projectID)
		if err != nil {
			return domain.StoredIdea{}, fmt.Errorf("load project input: %w", err)
		}
		stored.Language = string(in.Language)
		stored.InputHash = domain.HashInput(in.Subject)
	}
	// What the Creator typed is cleaned the same way as the text the idea is
	// written from, so the next redo does not hand a straight quote to the AI.
	patch = cleanedPatch(patch)
	before := stored.Idea.Hash()
	idea := stored.Idea
	setText := func(name string, value *string, dst *string) {
		if value == nil {
			return
		}
		v := strings.TrimSpace(*value)
		if v == strings.TrimSpace(*dst) {
			return
		}
		*dst = v
		stored.CreatorFields = markCreatorField(stored.CreatorFields, name, v != "")
	}
	setText(domain.IdeaFieldCentralQuestion, patch.CentralQuestion, &idea.CentralQuestion)
	setText(domain.IdeaFieldScope, patch.Scope, &idea.Scope)
	if patch.CuriosityPoints != nil {
		points := domain.ProjectIdea{CuriosityPoints: *patch.CuriosityPoints}.Normalized().CuriosityPoints
		if len(points) > domain.MaxCuriosityPoints {
			return domain.StoredIdea{}, fmt.Errorf("%w: tối đa %d điểm gây tò mò", domain.ErrInvalidUnitEdit, domain.MaxCuriosityPoints)
		}
		if strings.Join(points, "\n") != strings.Join(domain.ProjectIdea{CuriosityPoints: idea.CuriosityPoints}.Normalized().CuriosityPoints, "\n") {
			idea.CuriosityPoints = points
			stored.CreatorFields = markCreatorField(stored.CreatorFields, domain.IdeaFieldCuriosityPoints, len(points) > 0)
		}
	}
	stored.Idea = idea
	if stored.Idea.CuriosityPoints == nil {
		stored.Idea.CuriosityPoints = []string{}
	}
	if stored.Idea.Hash() != before || !exists {
		stored.Status = domain.IdeaPending
		stored.ApprovedAt = nil
	}
	stored.UpdatedAt = uc.now()
	if err := uc.store.SaveProjectIdea(ctx, stored); err != nil {
		return domain.StoredIdea{}, err
	}
	return stored, nil
}

// cleanedPatch returns the patch with every text run through
// domain.SanitizeIdeaText; a nil field stays nil.
func cleanedPatch(p IdeaPatch) IdeaPatch {
	clean := func(s *string) *string {
		if s == nil {
			return nil
		}
		v := domain.SanitizeIdeaText(*s)
		return &v
	}
	out := IdeaPatch{CentralQuestion: clean(p.CentralQuestion), Scope: clean(p.Scope)}
	if p.CuriosityPoints != nil {
		points := make([]string, len(*p.CuriosityPoints))
		for i, pt := range *p.CuriosityPoints {
			points[i] = domain.SanitizeIdeaText(pt)
		}
		out.CuriosityPoints = &points
	}
	return out
}

// markCreatorField adds name to fields when owned, removes it when not.
func markCreatorField(fields []string, name string, owned bool) []string {
	out := []string{}
	for _, f := range fields {
		if f != name {
			out = append(out, f)
		}
	}
	if owned {
		out = append(out, name)
	}
	return out
}

// Redo has the AI write the idea again with the Creator's note.
func (uc *ProjectIdeaUseCase) Redo(ctx context.Context, projectID, note string) (domain.StoredIdea, error) {
	return uc.Generate(ctx, projectID, note)
}

// Approve records the Creator's approval. The idea must pass the machine check;
// the AI review informs the decision and does not gate it.
func (uc *ProjectIdeaUseCase) Approve(ctx context.Context, projectID string) (domain.StoredIdea, error) {
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return domain.StoredIdea{}, err
	}
	stored, err := uc.store.GetProjectIdea(ctx, projectID)
	if err != nil {
		return domain.StoredIdea{}, err
	}
	if err := stored.Idea.Validate(); err != nil {
		return domain.StoredIdea{}, err
	}
	now := uc.now()
	stored.Status = domain.IdeaApproved
	stored.ApprovedAt = &now
	stored.UpdatedAt = now
	if err := uc.store.SaveProjectIdea(ctx, stored); err != nil {
		return domain.StoredIdea{}, err
	}
	return stored, nil
}

// Unapprove takes the approval back.
func (uc *ProjectIdeaUseCase) Unapprove(ctx context.Context, projectID string) (domain.StoredIdea, error) {
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return domain.StoredIdea{}, err
	}
	stored, err := uc.store.GetProjectIdea(ctx, projectID)
	if err != nil {
		return domain.StoredIdea{}, err
	}
	stored.Status = domain.IdeaPending
	stored.ApprovedAt = nil
	stored.UpdatedAt = uc.now()
	if err := uc.store.SaveProjectIdea(ctx, stored); err != nil {
		return domain.StoredIdea{}, err
	}
	return stored, nil
}

// Review has the AI read the idea for clarity, researchability and scope, and
// stores what it found. The review is advice: it never changes the idea or its
// approval.
func (uc *ProjectIdeaUseCase) Review(ctx context.Context, projectID string) (domain.IdeaReview, error) {
	if err := uc.requireAI(); err != nil {
		return domain.IdeaReview{}, err
	}
	stored, err := uc.store.GetProjectIdea(ctx, projectID)
	if err != nil {
		return domain.IdeaReview{}, err
	}
	if err := stored.Idea.Validate(); err != nil {
		return domain.IdeaReview{}, err
	}
	in, err := uc.input.IdeaInput(ctx, projectID)
	if err != nil {
		return domain.IdeaReview{}, fmt.Errorf("load project input: %w", err)
	}
	models, err := uc.models.GetAuthoringModels(ctx, projectID)
	if err != nil {
		return domain.IdeaReview{}, fmt.Errorf("load authoring models: %w", err)
	}
	prompt, err := uc.prompts.GetActive(ctx, domain.RoleProjectIdeaReview)
	if err != nil {
		return domain.IdeaReview{}, fmt.Errorf("load idea review prompt: %w", err)
	}
	system := domain.RenderPromptTemplate(prompt.TemplateText, map[string]string{
		"idea":          stored.Idea.Section(),
		"idea_language": briefLanguageName(briefLanguage(in.Language)),
	})
	result, chatErr := uc.provider.Chat(ctx, ChatRequest{
		System:      system,
		User:        userTurnNudge,
		MaxTokens:   uc.maxOutputTokens,
		Temperature: 0.3,
		Model:       models.ModelFor("brief"),
		OnProgress:  ProgressFrom(ctx),
		ProjectID:   projectID,
		Role:        string(domain.RoleProjectIdeaReview),
		Step:        "idea_review",
		ExpectJSON:  true,
	})
	if chatErr != nil {
		return domain.IdeaReview{}, chatErr
	}
	review, err := domain.ParseIdeaReviewReply(string(result.JSON))
	if err != nil {
		return domain.IdeaReview{}, err
	}
	review.IdeaHash = stored.Idea.Hash()
	review.CreatedAt = uc.now()
	if err := uc.store.SaveIdeaReview(ctx, projectID, review); err != nil {
		return domain.IdeaReview{}, err
	}
	return review, nil
}

func currentIdeaText(s domain.StoredIdea, exists bool) string {
	if !exists {
		return "(chưa có)"
	}
	// Marshalling a struct of strings cannot fail.
	raw, _ := json.Marshal(s.Idea.Normalized())
	return string(raw)
}

func keptProjectIdeaFieldsText(s domain.StoredIdea) string {
	if len(s.CreatorFields) == 0 {
		return "(không có)"
	}
	var lines []string
	for _, f := range domain.ProjectIdeaFields {
		if !s.HasCreatorField(f) {
			continue
		}
		if f == domain.IdeaFieldCuriosityPoints {
			raw, _ := json.Marshal(s.Idea.Normalized().CuriosityPoints)
			lines = append(lines, "- "+f+": "+string(raw))
			continue
		}
		lines = append(lines, "- "+f+": "+s.Idea.Field(f))
	}
	return strings.Join(lines, "\n")
}
