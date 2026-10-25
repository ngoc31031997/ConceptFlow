package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"authoring/internal/domain"
)

// ErrStoryNeedsBrief is returned when the story is asked for before the brief is approved.
var ErrStoryNeedsBrief = errors.New("duyệt brief ở bước Creative brief trước")

// ErrStoryNeedsResearch is returned when the story is asked for before the
// research step is approved or skipped.
var ErrStoryNeedsResearch = errors.New("duyệt bảng dữ kiện hoặc bỏ qua bước Nghiên cứu trước")

// ErrStoryNothingToWrite is returned when every field of the story is the Creator's.
var ErrStoryNothingToWrite = errors.New("cả ba ô đều do bạn viết, không có ô nào để AI viết")

// StoryStorePort keeps a project's story and reads the research facts it is
// written from and the model choice of the step.
type StoryStorePort interface {
	// GetProjectStory returns the story, or domain.ErrStoryNotFound.
	GetProjectStory(ctx context.Context, projectID string) (domain.StoredStory, error)
	SaveProjectStory(ctx context.Context, story domain.StoredStory) error
	ListResearchFacts(ctx context.Context, projectID string) ([]domain.StoredFact, error)
	GetAuthoringModels(ctx context.Context, projectID string) (domain.AuthoringStepModels, error)
}

// ResearchStandingPort tells whether the research step is approved or skipped.
type ResearchStandingPort interface {
	ResearchStanding(ctx context.Context, projectID string) (resolved, skipped bool, err error)
}

// ApprovedStoryPort reads the story the Creator approved on the fourth step.
type ApprovedStoryPort interface {
	// ApprovedStory returns the approved story and a hash of its content; ok is
	// false when there is none.
	ApprovedStory(ctx context.Context, projectID string) (story domain.StoryStructure, hash string, ok bool, err error)
}

// StoryFactView is one kept fact as the story step lists it.
type StoryFactView struct {
	ID      string          `json:"id"`
	ClaimVI string          `json:"claim_vi"`
	Type    domain.FactType `json:"type"`
	// Must is true for a fact the script has to state.
	Must bool `json:"must"`
	// Cited is true when a beat of the arc rests on the fact.
	Cited bool             `json:"cited"`
	Link  domain.LinkState `json:"link"`
}

// StoryState is everything the story screen shows.
type StoryState struct {
	// Story is nil until the AI or the Creator wrote one.
	Story *domain.StoredStory `json:"-"`
	// Brief is the approved brief the AI reads; nil while the brief step is not approved.
	Brief *domain.ProjectBrief `json:"-"`
	// Facts are the research facts the AI reads, in code order.
	Facts []StoryFactView `json:"-"`
	// ResearchResolved and ResearchSkipped tell where the research step stands.
	ResearchResolved bool
	ResearchSkipped  bool
	// BriefStale and ResearchStale are true when the approved brief or the kept
	// facts differ from the ones the story was written from.
	BriefStale    bool
	ResearchStale bool
}

// StoryDeskUseCase backs the story step: the AI writing the story from the
// approved brief and the kept research facts, the Creator's edits and approval.
type StoryDeskUseCase struct {
	store           StoryStorePort
	briefs          ApprovedBriefPort
	research        ResearchStandingPort
	projects        ProjectReaderPort
	locks           AuthoringLockPort
	prompts         PromptActivePort
	provider        LLMProviderPort
	maxOutputTokens int
	now             func() time.Time
}

// NewStoryDeskUseCase wires the use case. provider may be nil when no language
// model is configured: the Creator then writes the story by hand.
func NewStoryDeskUseCase(
	store StoryStorePort, briefs ApprovedBriefPort, research ResearchStandingPort, projects ProjectReaderPort,
	locks AuthoringLockPort, prompts PromptActivePort,
	provider LLMProviderPort, maxOutputTokens int,
) *StoryDeskUseCase {
	return &StoryDeskUseCase{
		store: store, briefs: briefs, research: research, projects: projects, locks: locks, prompts: prompts,
		provider: provider, maxOutputTokens: maxOutputTokens, now: time.Now,
	}
}

// keptResearchFacts lists the facts the Creator has not dropped, in code order.
func (uc *StoryDeskUseCase) keptResearchFacts(ctx context.Context, projectID string) ([]domain.StoredFact, error) {
	all, err := uc.store.ListResearchFacts(ctx, projectID)
	if err != nil {
		return nil, err
	}
	kept := make([]domain.StoredFact, 0, len(all))
	for _, f := range all {
		if f.Fact.Keep {
			kept = append(kept, f)
		}
	}
	return kept, nil
}

func knownStoredFact(facts []domain.StoredFact) func(string) bool {
	set := make(map[string]bool, len(facts))
	for _, f := range facts {
		set[f.ID] = true
	}
	return func(id string) bool { return set[id] }
}

// State returns the story screen's data.
func (uc *StoryDeskUseCase) State(ctx context.Context, projectID string) (StoryState, error) {
	var state StoryState
	var err error
	if state.ResearchResolved, state.ResearchSkipped, err = uc.research.ResearchStanding(ctx, projectID); err != nil {
		return StoryState{}, err
	}
	brief, briefOK, err := uc.briefs.ApprovedBrief(ctx, projectID)
	if err != nil {
		return StoryState{}, err
	}
	if briefOK {
		state.Brief = &brief
	}
	facts, err := uc.keptResearchFacts(ctx, projectID)
	if err != nil {
		return StoryState{}, err
	}
	stored, err := uc.store.GetProjectStory(ctx, projectID)
	switch {
	case err == nil:
		state.Story = &stored
		state.BriefStale = briefOK && stored.BriefHash != brief.Hash()
		state.ResearchStale = state.ResearchResolved && stored.ResearchHash != domain.ResearchHash(facts)
	case !errors.Is(err, domain.ErrStoryNotFound):
		return StoryState{}, err
	}
	cited := map[string]bool{}
	if state.Story != nil {
		for _, id := range state.Story.Story.CitedFacts() {
			cited[id] = true
		}
	}
	state.Facts = make([]StoryFactView, 0, len(facts))
	for _, f := range facts {
		state.Facts = append(state.Facts, StoryFactView{
			ID: f.ID, ClaimVI: f.Fact.ClaimVI, Type: f.Fact.Type, Must: f.Fact.Must, Cited: cited[f.ID], Link: f.Fact.Link.State,
		})
	}
	return state, nil
}

// ApprovedStory returns the story the Creator approved with a hash of its
// content, for the steps after this one; ok is false when there is none.
func (uc *StoryDeskUseCase) ApprovedStory(ctx context.Context, projectID string) (story domain.StoryStructure, hash string, ok bool, err error) {
	stored, err := uc.store.GetProjectStory(ctx, projectID)
	if errors.Is(err, domain.ErrStoryNotFound) {
		return domain.StoryStructure{}, "", false, nil
	}
	if err != nil {
		return domain.StoryStructure{}, "", false, err
	}
	if stored.Status != domain.StoryApproved {
		return domain.StoryStructure{}, "", false, nil
	}
	return stored.Story, stored.Story.Hash(), true, nil
}

// FactDependents lists, for the research step, the beat of the story that rests
// on a fact, so that dropping or changing the fact says what it sends back.
func (uc *StoryDeskUseCase) FactDependents() FactDependentsFinder {
	return func(ctx context.Context, projectID, factID string) ([]string, error) {
		stored, err := uc.store.GetProjectStory(ctx, projectID)
		if errors.Is(err, domain.ErrStoryNotFound) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		var items []string
		for _, b := range stored.Story.NarrativeArc {
			for _, id := range b.Facts {
				if id == factID {
					items = append(items, fmt.Sprintf("Câu chuyện · nhịp «%s» đang trích %s: câu chuyện nên được viết lại", b.Beat, factID))
				}
			}
		}
		return items, nil
	}
}

// requireAI fails unless a language model is configured.
func (uc *StoryDeskUseCase) requireAI() error {
	if uc.provider == nil {
		return ErrLLMNotConfigured
	}
	if r, ok := uc.provider.(interface{ Ready() bool }); ok && !r.Ready() {
		return ErrLLMNotConfigured
	}
	return nil
}

// current loads the stored story, or a new empty one when there is none yet.
func (uc *StoryDeskUseCase) current(ctx context.Context, projectID string) (domain.StoredStory, bool, error) {
	stored, err := uc.store.GetProjectStory(ctx, projectID)
	if errors.Is(err, domain.ErrStoryNotFound) {
		now := uc.now()
		return domain.StoredStory{
			ProjectID: projectID, Status: domain.StoryPending, CreatorFields: []string{}, CreatedAt: now, UpdatedAt: now,
		}, false, nil
	}
	if err != nil {
		return domain.StoredStory{}, false, err
	}
	return stored, true, nil
}

// storyInputs are what the AI reads: the approved brief, the kept facts and
// whether the research step was skipped.
type storyInputs struct {
	brief   domain.ProjectBrief
	facts   []domain.StoredFact
	skipped bool
}

// inputs gathers the inputs of an AI run and refuses when the step is not open
// yet: the brief must be approved and the research step approved or skipped.
func (uc *StoryDeskUseCase) inputs(ctx context.Context, projectID string) (storyInputs, error) {
	var in storyInputs
	brief, ok, err := uc.briefs.ApprovedBrief(ctx, projectID)
	if err != nil {
		return in, err
	}
	if !ok {
		return in, fmt.Errorf("%w: %w", domain.ErrInvalidStatus, ErrStoryNeedsBrief)
	}
	resolved, skipped, err := uc.research.ResearchStanding(ctx, projectID)
	if err != nil {
		return in, err
	}
	if !resolved {
		return in, fmt.Errorf("%w: %w", domain.ErrInvalidStatus, ErrStoryNeedsResearch)
	}
	in.brief, in.skipped = brief, skipped
	if in.facts, err = uc.keptResearchFacts(ctx, projectID); err != nil {
		return in, err
	}
	return in, nil
}

// researchText lists the kept facts for the prompt, or says there are none.
func researchText(facts []domain.StoredFact, skipped bool) string {
	if len(facts) == 0 {
		if skipped {
			return "(chưa có dữ kiện nào: bước Nghiên cứu đã bỏ qua)"
		}
		return "(chưa có dữ kiện nào)"
	}
	list := make([]storyFact, len(facts))
	for i, f := range facts {
		list[i] = storyFact{ID: f.ID, Fact: f.Fact}
	}
	return factsText(list)
}

// keptStoryFieldsText lists the fields the Creator wrote, which stay as they are.
func keptStoryFieldsText(stored domain.StoredStory) string {
	var lines []string
	for _, key := range domain.StoryFieldKeys {
		if !stored.HasCreatorField(key) {
			continue
		}
		switch key {
		case domain.StoryFieldCentralQuestion:
			lines = append(lines, "- "+key+": "+stored.Story.CentralQuestion)
		case domain.StoryFieldCoreAnswer:
			lines = append(lines, "- "+key+": "+stored.Story.CoreAnswer)
		case domain.StoryFieldNarrativeArc:
			lines = append(lines, "- "+key+":\n"+stored.Story.ArcText())
		}
	}
	if len(lines) == 0 {
		return "(không có)"
	}
	return strings.Join(lines, "\n")
}

// Generate has the AI write the story from the approved brief and the kept
// facts, keeping every field the Creator wrote; note is the Creator's
// instruction for this run ("" for the first run). The story goes back to
// waiting for approval. A reply that cannot be read, that misses a field or
// that cites a fact the table does not hold is an error and nothing is stored.
func (uc *StoryDeskUseCase) Generate(ctx context.Context, projectID, note string) (domain.StoredStory, error) {
	if err := uc.requireAI(); err != nil {
		return domain.StoredStory{}, err
	}
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return domain.StoredStory{}, err
	}
	in, err := uc.inputs(ctx, projectID)
	if err != nil {
		return domain.StoredStory{}, err
	}
	project, err := uc.projects.Get(ctx, projectID)
	if err != nil {
		return domain.StoredStory{}, fmt.Errorf("load project: %w", err)
	}
	models, err := uc.store.GetAuthoringModels(ctx, projectID)
	if err != nil {
		return domain.StoredStory{}, fmt.Errorf("load authoring models: %w", err)
	}
	current, exists, err := uc.current(ctx, projectID)
	if err != nil {
		return domain.StoredStory{}, err
	}
	if current.HasCreatorField(domain.StoryFieldCentralQuestion) && current.HasCreatorField(domain.StoryFieldCoreAnswer) &&
		current.HasCreatorField(domain.StoryFieldNarrativeArc) {
		return domain.StoredStory{}, fmt.Errorf("%w: %w", domain.ErrInvalidUnitEdit, ErrStoryNothingToWrite)
	}
	prompt, err := uc.prompts.GetActive(ctx, domain.RoleStoryFrame)
	if err != nil {
		return domain.StoredStory{}, fmt.Errorf("load story prompt: %w", err)
	}
	currentText := "(chưa có)"
	if exists && len(current.Story.MissingFields()) < len(domain.StoryFieldKeys) {
		currentText = current.Story.Section()
	}
	language := briefLanguage(project.ContentLanguage)
	system := domain.RenderPromptTemplate(prompt.TemplateText, map[string]string{
		"brief_language": briefLanguageName(language),
		"brief":          in.brief.Section(),
		"research":       researchText(in.facts, in.skipped),
		"current_story":  currentText,
		"kept_fields":    keptStoryFieldsText(current),
		"redo_note":      noteText(note),
	})

	result, chatErr := uc.provider.Chat(ctx, ChatRequest{
		System:      system,
		User:        userTurnNudge,
		MaxTokens:   uc.maxOutputTokens,
		Temperature: 0.7,
		Model:       models.ModelFor("story_frame"),
		OnProgress:  ProgressFrom(ctx),
		ProjectID:   projectID,
		Role:        string(domain.RoleStoryFrame),
		Step:        "story",
		ExpectJSON:  true,
	})
	if chatErr != nil {
		return domain.StoredStory{}, chatErr
	}
	written, err := domain.ParseStoryReply(string(result.JSON), knownStoredFact(in.facts))
	if err != nil {
		return domain.StoredStory{}, err
	}
	written = domain.KeepStoryCreatorFields(written, current.Story, current.CreatorFields).Normalized()
	if err := written.Validate(); err != nil {
		return domain.StoredStory{}, fmt.Errorf("%w: %v", domain.ErrStoryReplyUnusable, err)
	}
	if err := domain.CheckStoryFacts(written, knownStoredFact(in.facts)); err != nil {
		return domain.StoredStory{}, fmt.Errorf("%w: %v", domain.ErrStoryReplyUnusable, err)
	}
	current.Story = written
	current.Status = domain.StoryPending
	current.BriefHash = in.brief.Hash()
	current.ResearchHash = domain.ResearchHash(in.facts)
	current.UpdatedAt = uc.now()
	if err := uc.store.SaveProjectStory(ctx, current); err != nil {
		return domain.StoredStory{}, err
	}
	return current, nil
}

// StoryPatch is the Creator's hand edit; a nil field is left alone.
type StoryPatch struct {
	CentralQuestion *string
	CoreAnswer      *string
	NarrativeArc    *[]domain.ArcBeat
}

func (p StoryPatch) empty() bool {
	return p.CentralQuestion == nil && p.CoreAnswer == nil && p.NarrativeArc == nil
}

// Edit stores the Creator's hand edit. A field whose value changes becomes the
// Creator's; a field emptied goes back to the AI, which writes it on the next
// run. Any change sends an approved story back to waiting for approval, since
// the steps after read only the approved story. Editing before any story exists
// creates one, for a Creator working without a language model. A beat may cite
// only facts of the research table.
func (uc *StoryDeskUseCase) Edit(ctx context.Context, projectID string, patch StoryPatch) (domain.StoredStory, error) {
	if patch.empty() {
		return domain.StoredStory{}, fmt.Errorf("%w: không có ô nào để lưu", domain.ErrInvalidUnitEdit)
	}
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return domain.StoredStory{}, err
	}
	stored, exists, err := uc.current(ctx, projectID)
	if err != nil {
		return domain.StoredStory{}, err
	}
	facts, err := uc.keptResearchFacts(ctx, projectID)
	if err != nil {
		return domain.StoredStory{}, err
	}
	if !exists {
		if brief, ok, err := uc.briefs.ApprovedBrief(ctx, projectID); err != nil {
			return domain.StoredStory{}, err
		} else if ok {
			stored.BriefHash = brief.Hash()
		}
		stored.ResearchHash = domain.ResearchHash(facts)
	}
	before := stored.Story.Hash()
	story := stored.Story
	setText := func(name string, value *string, dst *string) {
		if value == nil {
			return
		}
		v := strings.TrimSpace(domain.SanitizeIdeaText(*value))
		if v == strings.TrimSpace(*dst) {
			return
		}
		*dst = v
		stored.CreatorFields = markCreatorField(stored.CreatorFields, name, v != "")
	}
	setText(domain.StoryFieldCentralQuestion, patch.CentralQuestion, &story.CentralQuestion)
	setText(domain.StoryFieldCoreAnswer, patch.CoreAnswer, &story.CoreAnswer)
	if patch.NarrativeArc != nil {
		arc, err := cleanedArc(*patch.NarrativeArc)
		if err != nil {
			return domain.StoredStory{}, err
		}
		if len(arc) > domain.MaxArcBeats {
			return domain.StoredStory{}, fmt.Errorf("%w: mạch kể tối đa %d nhịp", domain.ErrInvalidUnitEdit, domain.MaxArcBeats)
		}
		if err := domain.CheckStoryFacts(domain.StoryStructure{NarrativeArc: arc}, knownStoredFact(facts)); err != nil {
			return domain.StoredStory{}, fmt.Errorf("%w: %v", domain.ErrInvalidUnitEdit, strings.TrimPrefix(err.Error(), domain.ErrStoryInvalid.Error()+": "))
		}
		if !domain.ArcEqual(arc, story.NarrativeArc) {
			story.NarrativeArc = arc
			stored.CreatorFields = markCreatorField(stored.CreatorFields, domain.StoryFieldNarrativeArc, len(arc) > 0)
		}
	}
	stored.Story = story.Normalized()
	if stored.Story.Hash() != before || !exists {
		stored.Status = domain.StoryPending
	}
	stored.UpdatedAt = uc.now()
	if err := uc.store.SaveProjectStory(ctx, stored); err != nil {
		return domain.StoredStory{}, err
	}
	return stored, nil
}

// cleanedArc returns the beats with every text run through
// domain.SanitizeIdeaText, the fact codes written in their canonical form and
// the empty beats dropped; a code that is not a fact code is an error.
func cleanedArc(arc []domain.ArcBeat) ([]domain.ArcBeat, error) {
	raw := make([]domain.ArcBeat, len(arc))
	for i, b := range arc {
		refs := make([]string, 0, len(b.Facts))
		for _, code := range b.Facts {
			id, ok := domain.NormalizeFactID(strings.TrimSpace(code))
			if !ok {
				return nil, fmt.Errorf("%w: %q không phải mã dữ kiện", domain.ErrInvalidUnitEdit, code)
			}
			refs = append(refs, id)
		}
		raw[i] = domain.ArcBeat{Beat: domain.SanitizeIdeaText(b.Beat), Text: domain.SanitizeIdeaText(b.Text), Facts: refs}
	}
	return domain.StoryStructure{NarrativeArc: raw}.Normalized().NarrativeArc, nil
}

// Approve records the Creator's approval. The story must pass the machine
// check; whether it tells a story rather than lists information is the
// Creator's call.
func (uc *StoryDeskUseCase) Approve(ctx context.Context, projectID string) (domain.StoredStory, error) {
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return domain.StoredStory{}, err
	}
	stored, err := uc.store.GetProjectStory(ctx, projectID)
	if err != nil {
		return domain.StoredStory{}, err
	}
	if err := stored.Story.Validate(); err != nil {
		return domain.StoredStory{}, err
	}
	stored.Status = domain.StoryApproved
	stored.UpdatedAt = uc.now()
	if err := uc.store.SaveProjectStory(ctx, stored); err != nil {
		return domain.StoredStory{}, err
	}
	return stored, nil
}

// Unapprove takes the approval back.
func (uc *StoryDeskUseCase) Unapprove(ctx context.Context, projectID string) (domain.StoredStory, error) {
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return domain.StoredStory{}, err
	}
	stored, err := uc.store.GetProjectStory(ctx, projectID)
	if err != nil {
		return domain.StoredStory{}, err
	}
	stored.Status = domain.StoryPending
	stored.UpdatedAt = uc.now()
	if err := uc.store.SaveProjectStory(ctx, stored); err != nil {
		return domain.StoredStory{}, err
	}
	return stored, nil
}
