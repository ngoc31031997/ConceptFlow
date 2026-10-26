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

// ProjectBriefStorePort keeps a project's brief and the AI's review of it.
type ProjectBriefStorePort interface {
	// GetProjectBrief returns the brief, or domain.ErrBriefNotFound.
	GetProjectBrief(ctx context.Context, projectID string) (domain.StoredBrief, error)
	SaveProjectBrief(ctx context.Context, brief domain.StoredBrief) error
	// GetBriefReview returns the latest review, or domain.ErrBriefNotFound.
	GetBriefReview(ctx context.Context, projectID string) (domain.BriefReview, error)
	SaveBriefReview(ctx context.Context, projectID string, review domain.BriefReview) error
}

// ApprovedIdeaPort reads the idea the Creator approved on the first step.
type ApprovedIdeaPort interface {
	// ApprovedIdea returns the approved idea; ok is false when there is none.
	ApprovedIdea(ctx context.Context, projectID string) (idea domain.ProjectIdea, ok bool, err error)
}

// ProjectBriefState is everything the brief screen shows.
type ProjectBriefState struct {
	// Brief is nil until the AI or the Creator wrote one.
	Brief *domain.StoredBrief
	// Idea is the approved idea the brief is written from; nil while the first
	// step is not approved. Subject and Language are the project's input.
	Idea     *domain.ProjectIdea
	Subject  string
	Language domain.ContentLanguage
	// VideoMode and the duration range tell the screen which lengths are valid.
	VideoMode          domain.VideoOutputMode
	MinDurationSeconds int
	MaxDurationSeconds int
	// IdeaChanged is true when the approved idea differs from the one the brief
	// was written from, so the brief may no longer fit it.
	IdeaChanged bool
	Review      *domain.BriefReview
	// ReviewStale is true when the brief changed after the review read it.
	ReviewStale bool
}

// ProjectBriefUseCase backs the second step: the AI writing the brief from the
// approved idea, the Creator's edits and approval, and the AI's review of the
// brief before approval.
type ProjectBriefUseCase struct {
	store           ProjectBriefStorePort
	ideas           ApprovedIdeaPort
	input           IdeaInputPort
	projects        ProjectReaderPort
	models          AuthoringModelsReaderPort
	locks           AuthoringLockPort
	prompts         PromptActivePort
	provider        LLMProviderPort
	maxOutputTokens int
	now             func() time.Time
}

// NewProjectBriefUseCase wires the use case. provider may be nil when no
// language model is configured: the Creator then writes the brief by hand.
func NewProjectBriefUseCase(
	store ProjectBriefStorePort, ideas ApprovedIdeaPort, input IdeaInputPort, projects ProjectReaderPort,
	models AuthoringModelsReaderPort, locks AuthoringLockPort, prompts PromptActivePort,
	provider LLMProviderPort, maxOutputTokens int,
) *ProjectBriefUseCase {
	return &ProjectBriefUseCase{
		store: store, ideas: ideas, input: input, projects: projects, models: models, locks: locks,
		prompts: prompts, provider: provider, maxOutputTokens: maxOutputTokens, now: time.Now,
	}
}

// State returns the brief screen's data.
func (uc *ProjectBriefUseCase) State(ctx context.Context, projectID string) (ProjectBriefState, error) {
	in, err := uc.input.IdeaInput(ctx, projectID)
	if err != nil {
		return ProjectBriefState{}, err
	}
	project, err := uc.projects.Get(ctx, projectID)
	if err != nil {
		return ProjectBriefState{}, fmt.Errorf("load project: %w", err)
	}
	state := ProjectBriefState{Subject: in.Subject, Language: in.Language, VideoMode: project.VideoOutputMode}
	state.MinDurationSeconds, state.MaxDurationSeconds = domain.DurationBounds(project.VideoOutputMode)
	idea, ok, err := uc.ideas.ApprovedIdea(ctx, projectID)
	if err != nil {
		return ProjectBriefState{}, err
	}
	if ok {
		state.Idea = &idea
	}
	stored, err := uc.store.GetProjectBrief(ctx, projectID)
	switch {
	case err == nil:
		state.Brief = &stored
		state.IdeaChanged = ok && stored.IdeaHash != idea.Hash()
	case !errors.Is(err, domain.ErrBriefNotFound):
		return ProjectBriefState{}, err
	}
	review, err := uc.store.GetBriefReview(ctx, projectID)
	switch {
	case err == nil:
		state.Review = &review
		state.ReviewStale = state.Brief == nil || review.BriefHash != state.Brief.Brief.Hash()
	case !errors.Is(err, domain.ErrBriefNotFound):
		return ProjectBriefState{}, err
	}
	return state, nil
}

// ApprovedBrief returns the brief the Creator approved, for the steps after this
// one; ok is false when there is none.
func (uc *ProjectBriefUseCase) ApprovedBrief(ctx context.Context, projectID string) (domain.ProjectBrief, bool, error) {
	stored, err := uc.store.GetProjectBrief(ctx, projectID)
	if errors.Is(err, domain.ErrBriefNotFound) {
		return domain.ProjectBrief{}, false, nil
	}
	if err != nil {
		return domain.ProjectBrief{}, false, err
	}
	if stored.Status != domain.BriefApproved {
		return domain.ProjectBrief{}, false, nil
	}
	return stored.Brief, true, nil
}

// requireAI refuses when no language model is ready.
func (uc *ProjectBriefUseCase) requireAI(ctx context.Context) error {
	return requireProvider(ctx, uc.provider)
}

// videoFormText names the video's form for a prompt.
func videoFormText(mode domain.VideoOutputMode) string {
	if mode == domain.ModeShortOnly {
		return "video ngắn (short)"
	}
	return "video dài"
}

// Generate has the AI write the brief from the approved idea, keeping every
// field the Creator wrote; note is the Creator's instruction for this run ("" for
// the first run). The brief goes back to waiting for approval. A reply that
// cannot be read, that misses a field or whose length is outside the video's
// range is an error and nothing is stored.
func (uc *ProjectBriefUseCase) Generate(ctx context.Context, projectID, note string) (domain.StoredBrief, error) {
	if err := uc.requireAI(ctx); err != nil {
		return domain.StoredBrief{}, err
	}
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return domain.StoredBrief{}, err
	}
	idea, ok, err := uc.ideas.ApprovedIdea(ctx, projectID)
	if err != nil {
		return domain.StoredBrief{}, err
	}
	if !ok {
		return domain.StoredBrief{}, fmt.Errorf("%w: %w", domain.ErrInvalidStatus, domain.ErrBriefNeedsApprovedIdea)
	}
	in, err := uc.input.IdeaInput(ctx, projectID)
	if err != nil {
		return domain.StoredBrief{}, fmt.Errorf("load project input: %w", err)
	}
	project, err := uc.projects.Get(ctx, projectID)
	if err != nil {
		return domain.StoredBrief{}, fmt.Errorf("load project: %w", err)
	}
	models, err := uc.models.GetAuthoringModels(ctx, projectID)
	if err != nil {
		return domain.StoredBrief{}, fmt.Errorf("load authoring models: %w", err)
	}
	current, exists, err := uc.current(ctx, projectID)
	if err != nil {
		return domain.StoredBrief{}, err
	}
	prompt, err := uc.prompts.GetActive(ctx, domain.RoleProjectBrief)
	if err != nil {
		return domain.StoredBrief{}, fmt.Errorf("load brief prompt: %w", err)
	}
	mode := project.VideoOutputMode
	minSec, maxSec := domain.DurationBounds(mode)
	system := domain.RenderPromptTemplate(prompt.TemplateText, map[string]string{
		"topic":          strings.TrimSpace(in.Subject),
		"idea":           idea.Section(),
		"video_form":     videoFormText(mode),
		"duration_range": fmt.Sprintf("từ %d đến %d giây", minSec, maxSec),
		"brief_language": briefLanguageName(briefLanguage(in.Language)),
		"current_brief":  currentBriefText(current, exists),
		"kept_fields":    keptProjectBriefFieldsText(current),
		"redo_note":      noteText(note),
	})

	result, chatErr := uc.provider.Chat(ctx, ChatRequest{
		System:      system,
		User:        userTurnNudge,
		MaxTokens:   uc.maxOutputTokens,
		Temperature: 0.7,
		Model:       models.ModelFor("brief"),
		OnProgress:  ProgressFrom(ctx),
		ProjectID:   projectID,
		Role:        string(domain.RoleProjectBrief),
		Step:        "brief",
		ExpectJSON:  true,
	})
	if chatErr != nil {
		return domain.StoredBrief{}, chatErr
	}
	written, err := domain.ParseProjectBriefReply(string(result.JSON), mode)
	if err != nil {
		return domain.StoredBrief{}, err
	}
	written = domain.KeepProjectBriefCreatorFields(written, current.Brief, current.CreatorFields).Normalized()
	if err := written.Validate(mode); err != nil {
		return domain.StoredBrief{}, fmt.Errorf("%w: %v", domain.ErrProjectBriefReplyUnusable, err)
	}

	if exists {
		current.RedoCount++
	}
	current.Brief = written
	current.Status = domain.BriefPending
	current.ApprovedAt = nil
	current.Language = string(in.Language)
	current.IdeaHash = idea.Hash()
	current.UpdatedAt = uc.now()
	if err := uc.store.SaveProjectBrief(ctx, current); err != nil {
		return domain.StoredBrief{}, err
	}
	return current, nil
}

// current loads the stored brief, or a new empty one when there is none yet.
func (uc *ProjectBriefUseCase) current(ctx context.Context, projectID string) (domain.StoredBrief, bool, error) {
	stored, err := uc.store.GetProjectBrief(ctx, projectID)
	if errors.Is(err, domain.ErrBriefNotFound) {
		return domain.StoredBrief{ProjectID: projectID, Status: domain.BriefPending, CreatorFields: []string{}}, false, nil
	}
	if err != nil {
		return domain.StoredBrief{}, false, err
	}
	return stored, true, nil
}

// BriefPatch is the Creator's hand edit; a nil field is left alone.
type BriefPatch struct {
	Audience           *string
	DurationSeconds    *int
	Tone               *string
	LearningObjectives *[]string
	Avoid              *[]string
}

func (p BriefPatch) empty() bool {
	return p.Audience == nil && p.DurationSeconds == nil && p.Tone == nil &&
		p.LearningObjectives == nil && p.Avoid == nil
}

// Edit stores the Creator's hand edit. A field whose value changes becomes the
// Creator's; a field emptied goes back to the AI, which writes it on the next
// redo. Any change sends an approved brief back to waiting for approval, since
// the steps after read only the approved brief. Editing before any brief exists
// creates one, for a Creator working without a language model.
func (uc *ProjectBriefUseCase) Edit(ctx context.Context, projectID string, patch BriefPatch) (domain.StoredBrief, error) {
	if patch.empty() {
		return domain.StoredBrief{}, fmt.Errorf("%w: không có ô nào để lưu", domain.ErrInvalidUnitEdit)
	}
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return domain.StoredBrief{}, err
	}
	stored, exists, err := uc.current(ctx, projectID)
	if err != nil {
		return domain.StoredBrief{}, err
	}
	if !exists {
		in, err := uc.input.IdeaInput(ctx, projectID)
		if err != nil {
			return domain.StoredBrief{}, fmt.Errorf("load project input: %w", err)
		}
		stored.Language = string(in.Language)
		if idea, ok, err := uc.ideas.ApprovedIdea(ctx, projectID); err != nil {
			return domain.StoredBrief{}, err
		} else if ok {
			stored.IdeaHash = idea.Hash()
		}
	}
	// What the Creator typed is cleaned the same way as the idea's text, so the
	// next redo does not hand a straight quote to the AI.
	patch = cleanedBriefPatch(patch)
	before := stored.Brief.Hash()
	brief := stored.Brief
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
	setText(domain.BriefFieldAudience, patch.Audience, &brief.Audience)
	setText(domain.BriefFieldTone, patch.Tone, &brief.Tone)
	if patch.DurationSeconds != nil && *patch.DurationSeconds != brief.DurationSeconds {
		brief.DurationSeconds = *patch.DurationSeconds
		stored.CreatorFields = markCreatorField(stored.CreatorFields, domain.BriefFieldDuration, brief.DurationSeconds > 0)
	}
	setList := func(name string, value *[]string, max int, dst *[]string) error {
		if value == nil {
			return nil
		}
		items := domain.DistinctLines(*value)
		if len(items) > max {
			return fmt.Errorf("%w: tối đa %d dòng", domain.ErrInvalidUnitEdit, max)
		}
		if strings.Join(items, "\n") != strings.Join(domain.DistinctLines(*dst), "\n") {
			*dst = items
			stored.CreatorFields = markCreatorField(stored.CreatorFields, name, len(items) > 0)
		}
		return nil
	}
	if err := setList(domain.BriefFieldLearningObjectives, patch.LearningObjectives, domain.MaxLearningObjectives, &brief.LearningObjectives); err != nil {
		return domain.StoredBrief{}, err
	}
	if err := setList(domain.BriefFieldAvoid, patch.Avoid, domain.MaxAvoidItems, &brief.Avoid); err != nil {
		return domain.StoredBrief{}, err
	}
	stored.Brief = brief
	if stored.Brief.LearningObjectives == nil {
		stored.Brief.LearningObjectives = []string{}
	}
	if stored.Brief.Avoid == nil {
		stored.Brief.Avoid = []string{}
	}
	if stored.Brief.Hash() != before || !exists {
		stored.Status = domain.BriefPending
		stored.ApprovedAt = nil
	}
	stored.UpdatedAt = uc.now()
	if err := uc.store.SaveProjectBrief(ctx, stored); err != nil {
		return domain.StoredBrief{}, err
	}
	return stored, nil
}

// cleanedBriefPatch returns the patch with every text run through
// domain.SanitizeIdeaText; a nil field stays nil.
func cleanedBriefPatch(p BriefPatch) BriefPatch {
	clean := func(s *string) *string {
		if s == nil {
			return nil
		}
		v := domain.SanitizeIdeaText(*s)
		return &v
	}
	cleanList := func(l *[]string) *[]string {
		if l == nil {
			return nil
		}
		out := make([]string, len(*l))
		for i, it := range *l {
			out[i] = domain.SanitizeIdeaText(it)
		}
		return &out
	}
	return BriefPatch{
		Audience: clean(p.Audience), DurationSeconds: p.DurationSeconds, Tone: clean(p.Tone),
		LearningObjectives: cleanList(p.LearningObjectives), Avoid: cleanList(p.Avoid),
	}
}

// Redo has the AI write the brief again with the Creator's note.
func (uc *ProjectBriefUseCase) Redo(ctx context.Context, projectID, note string) (domain.StoredBrief, error) {
	return uc.Generate(ctx, projectID, note)
}

// Approve records the Creator's approval. The brief must pass the machine check
// for the video's form; the AI review informs the decision and does not gate it.
func (uc *ProjectBriefUseCase) Approve(ctx context.Context, projectID string) (domain.StoredBrief, error) {
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return domain.StoredBrief{}, err
	}
	stored, err := uc.store.GetProjectBrief(ctx, projectID)
	if err != nil {
		return domain.StoredBrief{}, err
	}
	project, err := uc.projects.Get(ctx, projectID)
	if err != nil {
		return domain.StoredBrief{}, fmt.Errorf("load project: %w", err)
	}
	if err := stored.Brief.Validate(project.VideoOutputMode); err != nil {
		return domain.StoredBrief{}, err
	}
	now := uc.now()
	stored.Status = domain.BriefApproved
	stored.ApprovedAt = &now
	stored.UpdatedAt = now
	if err := uc.store.SaveProjectBrief(ctx, stored); err != nil {
		return domain.StoredBrief{}, err
	}
	return stored, nil
}

// Unapprove takes the approval back.
func (uc *ProjectBriefUseCase) Unapprove(ctx context.Context, projectID string) (domain.StoredBrief, error) {
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return domain.StoredBrief{}, err
	}
	stored, err := uc.store.GetProjectBrief(ctx, projectID)
	if err != nil {
		return domain.StoredBrief{}, err
	}
	stored.Status = domain.BriefPending
	stored.ApprovedAt = nil
	stored.UpdatedAt = uc.now()
	if err := uc.store.SaveProjectBrief(ctx, stored); err != nil {
		return domain.StoredBrief{}, err
	}
	return stored, nil
}

// Review has the AI read the brief for audience, length, objectives and the
// avoid list, and stores what it found. The review is advice: it never changes
// the brief or its approval.
func (uc *ProjectBriefUseCase) Review(ctx context.Context, projectID string) (domain.BriefReview, error) {
	if err := uc.requireAI(ctx); err != nil {
		return domain.BriefReview{}, err
	}
	stored, err := uc.store.GetProjectBrief(ctx, projectID)
	if err != nil {
		return domain.BriefReview{}, err
	}
	project, err := uc.projects.Get(ctx, projectID)
	if err != nil {
		return domain.BriefReview{}, fmt.Errorf("load project: %w", err)
	}
	if err := stored.Brief.Validate(project.VideoOutputMode); err != nil {
		return domain.BriefReview{}, err
	}
	idea, ok, err := uc.ideas.ApprovedIdea(ctx, projectID)
	if err != nil {
		return domain.BriefReview{}, err
	}
	if !ok {
		return domain.BriefReview{}, fmt.Errorf("%w: %w", domain.ErrInvalidStatus, domain.ErrBriefNeedsApprovedIdea)
	}
	in, err := uc.input.IdeaInput(ctx, projectID)
	if err != nil {
		return domain.BriefReview{}, fmt.Errorf("load project input: %w", err)
	}
	models, err := uc.models.GetAuthoringModels(ctx, projectID)
	if err != nil {
		return domain.BriefReview{}, fmt.Errorf("load authoring models: %w", err)
	}
	prompt, err := uc.prompts.GetActive(ctx, domain.RoleProjectBriefReview)
	if err != nil {
		return domain.BriefReview{}, fmt.Errorf("load brief review prompt: %w", err)
	}
	system := domain.RenderPromptTemplate(prompt.TemplateText, map[string]string{
		"idea":           idea.Section(),
		"video_form":     videoFormText(project.VideoOutputMode),
		"brief":          stored.Brief.Section(),
		"brief_language": briefLanguageName(briefLanguage(in.Language)),
	})
	result, chatErr := uc.provider.Chat(ctx, ChatRequest{
		System:      system,
		User:        userTurnNudge,
		MaxTokens:   uc.maxOutputTokens,
		Temperature: 0.3,
		Model:       models.ModelFor("brief"),
		OnProgress:  ProgressFrom(ctx),
		ProjectID:   projectID,
		Role:        string(domain.RoleProjectBriefReview),
		Step:        "brief_review",
		ExpectJSON:  true,
	})
	if chatErr != nil {
		return domain.BriefReview{}, chatErr
	}
	review, err := domain.ParseBriefReviewReply(string(result.JSON))
	if err != nil {
		return domain.BriefReview{}, err
	}
	review.BriefHash = stored.Brief.Hash()
	review.CreatedAt = uc.now()
	if err := uc.store.SaveBriefReview(ctx, projectID, review); err != nil {
		return domain.BriefReview{}, err
	}
	return review, nil
}

func currentBriefText(s domain.StoredBrief, exists bool) string {
	if !exists {
		return "(chưa có)"
	}
	// Marshalling a struct of strings and an int cannot fail.
	raw, _ := json.Marshal(s.Brief.Normalized())
	return string(raw)
}

func keptProjectBriefFieldsText(s domain.StoredBrief) string {
	if len(s.CreatorFields) == 0 {
		return "(không có)"
	}
	n := s.Brief.Normalized()
	var lines []string
	for _, f := range domain.ProjectBriefFields {
		if !s.HasCreatorField(f) {
			continue
		}
		var value any
		switch f {
		case domain.BriefFieldAudience:
			value = n.Audience
		case domain.BriefFieldDuration:
			value = n.DurationSeconds
		case domain.BriefFieldTone:
			value = n.Tone
		case domain.BriefFieldLearningObjectives:
			value = n.LearningObjectives
		case domain.BriefFieldAvoid:
			value = n.Avoid
		}
		raw, _ := json.Marshal(value)
		lines = append(lines, "- "+f+": "+string(raw))
	}
	return strings.Join(lines, "\n")
}
