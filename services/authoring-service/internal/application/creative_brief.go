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

// BriefStorePort is what the idea and brief steps read and write beside their
// units: the topic, the target length, the model picked for them, and whether
// the saved story outline was written from the brief that is approved now.
type BriefStorePort interface {
	GetAuthoringTopic(ctx context.Context, projectID string) (string, error)
	GetAuthoringModels(ctx context.Context, projectID string) (domain.AuthoringStepModels, error)
	GetTargetDuration(ctx context.Context, projectID string) (domain.TargetDuration, error)
	SaveTargetDuration(ctx context.Context, projectID string, d domain.TargetDuration) error
	StoryBriefLink(ctx context.Context, projectID string) (StoryBriefLink, error)
}

// loadPlatform returns the platform the Creator picked for the project, or
// the default for its video form when none was picked.
func loadPlatform(ctx context.Context, units ReviewUnitPort, projectID string, mode domain.VideoOutputMode) (string, error) {
	unit, err := units.GetUnit(ctx, projectID, domain.UnitKindBriefPlatform, domain.BriefPlatformKey)
	if errors.Is(err, domain.ErrUnitNotFound) {
		return domain.DefaultPlatform(mode), nil
	}
	if err != nil {
		return "", err
	}
	var stored domain.BriefPlatform
	if err := json.Unmarshal(unit.Content, &stored); err != nil {
		return "", fmt.Errorf("read stored platform: %w", err)
	}
	if !domain.ValidPlatform(stored.Platform) {
		return domain.DefaultPlatform(mode), nil
	}
	return stored.Platform, nil
}

// briefSectionFor renders the approved idea and brief with the project's
// target length and platform, as the later steps' prompts read them.
func briefSectionFor(ctx context.Context, units ReviewUnitPort, projectID string, mode domain.VideoOutputMode, idea domain.Idea, brief domain.CreativeBrief, durationLabel string) (string, error) {
	platform, err := loadPlatform(ctx, units, projectID, mode)
	if err != nil {
		return "", err
	}
	return domain.BriefSection(idea, brief, durationLabel, platform), nil
}

// StoryBriefLink ties the story step's saved outline to the brief.
type StoryBriefLink struct {
	// HasStory is true once a story outline is saved.
	HasStory bool
	// FromCurrentBrief is true when that outline was saved while the brief
	// approved now was the approved one.
	FromCurrentBrief bool
}

// BriefState is everything the brief screen shows.
type BriefState struct {
	// Unit is nil until a brief was written or typed.
	Unit           *domain.ReviewUnit
	TargetDuration domain.TargetDuration
	// Platform is the one picked for the video, or the default for its form.
	Platform string
	// StoryStale is true when a story outline exists that was not written from
	// the brief approved now.
	StoryStale bool
	// StoryPromptReadsBrief is false when the active story_architect prompt
	// (one a Creator wrote) has no {{creative_brief}}, so the brief does not
	// reach the story step.
	StoryPromptReadsBrief bool
}

// CreativeBriefUseCase backs the brief step: the target length, the AI writing
// the brief from the approved idea, and the brief as a review unit (UnitKind
// for domain.UnitKindBrief).
type CreativeBriefUseCase struct {
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

// NewCreativeBriefUseCase wires the use case. provider may be nil when no
// language model is configured: the Creator then writes the brief by hand.
func NewCreativeBriefUseCase(
	units ReviewUnitPort, store BriefStorePort, projects ProjectReaderPort, locks AuthoringLockPort,
	prompts PromptActivePort, provider LLMProviderPort, recorder *LLMUsageRecorder, maxOutputTokens int,
) *CreativeBriefUseCase {
	return &CreativeBriefUseCase{
		units: units, store: store, projects: projects, locks: locks, prompts: prompts,
		provider: provider, recorder: recorder, maxOutputTokens: maxOutputTokens, now: time.Now,
	}
}

// briefIdeaRef names the unit a brief is built from.
const briefIdeaRef = domain.UnitKindIdea + ":" + domain.IdeaUnitKey

// ErrBriefIncomplete is returned when a brief with empty fields is approved.
var ErrBriefIncomplete = errors.New("brief còn ô trống")

// creativeBriefPlaceholder is the {{creative_brief}} text when no brief is approved.
const creativeBriefPlaceholder = "(chưa có brief đã duyệt: dựng dàn ý từ chủ đề)"

// briefVariable is the story_architect placeholder the approved brief fills.
const briefVariable = "{{creative_brief}}"

// State returns the brief screen's data.
func (uc *CreativeBriefUseCase) State(ctx context.Context, projectID string) (BriefState, error) {
	var state BriefState
	unit, err := uc.units.GetUnit(ctx, projectID, domain.UnitKindBrief, domain.BriefUnitKey)
	switch {
	case err == nil:
		state.Unit = &unit
	case !errors.Is(err, domain.ErrUnitNotFound):
		return BriefState{}, err
	}
	if state.TargetDuration, err = uc.store.GetTargetDuration(ctx, projectID); err != nil {
		return BriefState{}, err
	}
	// A project that never picked one is told the length its brief is written for.
	if !domain.ValidTargetDuration(state.TargetDuration) {
		state.TargetDuration = domain.DefaultTargetDuration
	}
	project, err := uc.projects.Get(ctx, projectID)
	if err != nil {
		return BriefState{}, fmt.Errorf("load project: %w", err)
	}
	if state.Platform, err = loadPlatform(ctx, uc.units, projectID, project.VideoOutputMode); err != nil {
		return BriefState{}, err
	}
	link, err := uc.store.StoryBriefLink(ctx, projectID)
	if err != nil {
		return BriefState{}, err
	}
	state.StoryStale = link.HasStory && !link.FromCurrentBrief
	// A library with no story prompt has nothing that could read the brief.
	story, err := uc.prompts.GetActive(ctx, domain.RoleStoryArchitect)
	if err != nil && !errors.Is(err, ErrPromptNotFound) {
		return BriefState{}, fmt.Errorf("load story prompt: %w", err)
	}
	state.StoryPromptReadsBrief = err == nil && strings.Contains(story.TemplateText, briefVariable)
	return state, nil
}

// SetTargetDuration saves the long video's target length. A brief already
// written keeps the length it was written for until the Creator has the AI
// write it again.
func (uc *CreativeBriefUseCase) SetTargetDuration(ctx context.Context, projectID string, d domain.TargetDuration) error {
	if !domain.ValidTargetDuration(d) {
		return fmt.Errorf("%w: thời lượng mục tiêu %q không có trong danh sách", domain.ErrInvalidWizardInput, d)
	}
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return err
	}
	return uc.store.SaveTargetDuration(ctx, projectID, d)
}

// SetPlatform saves the platform the video is published to. A brief already
// written keeps the platform it was written for until the Creator has the AI
// write it again.
func (uc *CreativeBriefUseCase) SetPlatform(ctx context.Context, projectID, platform string) error {
	if !domain.ValidPlatform(platform) {
		return fmt.Errorf("%w: nền tảng %q không có trong danh sách", domain.ErrInvalidWizardInput, platform)
	}
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return err
	}
	raw, err := json.Marshal(domain.BriefPlatform{Platform: platform})
	if err != nil {
		return err
	}
	_, err = uc.units.SaveUnit(ctx, domain.ReviewUnit{
		ProjectID: projectID, Kind: domain.UnitKindBriefPlatform, Key: domain.BriefPlatformKey,
		Status: domain.UnitApproved, Content: raw,
	})
	return err
}

// Generate has the AI write the brief from the approved idea, keeping every
// field the Creator wrote; note is the Creator's instruction for this run ("" for
// the first run). The brief goes to review. A reply that cannot be read, or
// that misses a field, is an error and nothing is stored.
func (uc *CreativeBriefUseCase) Generate(ctx context.Context, projectID, note string) (domain.ReviewUnit, error) {
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
	ideaUnit, idea, ideaExists, err := loadIdea(ctx, uc.units, projectID)
	if err != nil {
		return domain.ReviewUnit{}, err
	}
	if !ideaExists || ideaUnit.Status != domain.UnitApproved {
		return domain.ReviewUnit{}, fmt.Errorf("%w: %w", domain.ErrInvalidStatus, ErrBriefNeedsIdea)
	}
	ideaSHA, err := uc.units.UnitSHA(ctx, projectID, domain.UnitKindIdea, domain.IdeaUnitKey)
	if err != nil {
		return domain.ReviewUnit{}, err
	}
	duration, err := uc.store.GetTargetDuration(ctx, projectID)
	if err != nil {
		return domain.ReviewUnit{}, err
	}
	models, err := uc.store.GetAuthoringModels(ctx, projectID)
	if err != nil {
		return domain.ReviewUnit{}, fmt.Errorf("load authoring models: %w", err)
	}

	unit, current, exists, err := uc.current(ctx, projectID)
	if err != nil {
		return domain.ReviewUnit{}, err
	}
	prompt, err := uc.prompts.GetActive(ctx, domain.RoleCreativeBrief)
	if err != nil {
		return domain.ReviewUnit{}, fmt.Errorf("load brief prompt: %w", err)
	}
	language := briefLanguage(project.ContentLanguage)
	platform, err := loadPlatform(ctx, uc.units, projectID, project.VideoOutputMode)
	if err != nil {
		return domain.ReviewUnit{}, err
	}
	system := domain.RenderPromptTemplate(prompt.TemplateText, map[string]string{
		"topic":           topic,
		"idea":            domain.IdeaSection(idea),
		"video_kind":      videoKindLabel(project.VideoOutputMode),
		"target_duration": domain.TargetDurationLabel(project.VideoOutputMode, duration, language),
		"brief_language":  briefLanguageName(language),
		"platform":        platform,
		"current_brief":   currentJSONText(current, exists),
		"kept_fields":     keptFieldsText(current, unit.CreatorFields),
		"redo_note":       noteText(note),
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
		uc.recorder.Record(ctx, RecordFor(uc.provider.Name(), string(domain.RoleCreativeBrief), domain.UnitKindBrief, projectID, result.Usage, started, chatErr))
	}
	if chatErr != nil {
		return domain.ReviewUnit{}, chatErr
	}
	written, err := domain.ParseBriefReply(result.Content)
	if err != nil {
		return domain.ReviewUnit{}, err
	}
	written = domain.KeepCreatorFields(written, current, unit.CreatorFields)

	if exists {
		unit.RedoCount++
	}
	unit.Status = domain.UnitPending
	unit.InputRef, unit.InputSHA = briefIdeaRef, ideaSHA
	if unit.Content, err = json.Marshal(written); err != nil {
		return domain.ReviewUnit{}, err
	}
	return uc.units.SaveUnit(ctx, unit)
}

// current loads the brief unit, or a new empty one when none exists yet.
func (uc *CreativeBriefUseCase) current(ctx context.Context, projectID string) (domain.ReviewUnit, domain.CreativeBrief, bool, error) {
	unit, err := uc.units.GetUnit(ctx, projectID, domain.UnitKindBrief, domain.BriefUnitKey)
	if errors.Is(err, domain.ErrUnitNotFound) {
		return domain.ReviewUnit{
			ProjectID: projectID, Kind: domain.UnitKindBrief, Key: domain.BriefUnitKey, Status: domain.UnitPending,
		}, domain.CreativeBrief{}, false, nil
	}
	if err != nil {
		return domain.ReviewUnit{}, domain.CreativeBrief{}, false, err
	}
	var brief domain.CreativeBrief
	if err := json.Unmarshal(unit.Content, &brief); err != nil {
		return domain.ReviewUnit{}, domain.CreativeBrief{}, false, fmt.Errorf("read stored brief: %w", err)
	}
	return unit, brief, true, nil
}

// Edit stores the Creator's hand edit of brief fields. A field whose value
// changes becomes the Creator's; a field emptied goes back to the AI, which
// writes it on the next redo. An approved brief stays approved (the Creator
// wrote the change) unless the edit empties a field. Editing before any brief
// exists creates one, for a Creator working without a language model.
func (uc *CreativeBriefUseCase) Edit(ctx context.Context, projectID, key string, fields map[string]string) (domain.ReviewUnit, error) {
	if key != domain.BriefUnitKey {
		return domain.ReviewUnit{}, domain.ErrUnitNotFound
	}
	if len(fields) == 0 {
		return domain.ReviewUnit{}, fmt.Errorf("%w: không có ô nào để lưu", domain.ErrInvalidUnitEdit)
	}
	for name := range fields {
		if !domain.IsBriefField(name) {
			return domain.ReviewUnit{}, fmt.Errorf("%w: brief không có ô %q", domain.ErrInvalidUnitEdit, name)
		}
	}
	unit, brief, _, err := uc.current(ctx, projectID)
	if err != nil {
		return domain.ReviewUnit{}, err
	}
	for _, name := range domain.BriefFields {
		value, ok := fields[name]
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		if value == brief.Field(name) {
			continue
		}
		brief = brief.WithField(name, value)
		switch {
		case value == "":
			unit.CreatorFields = withoutField(unit.CreatorFields, name)
		case !unit.HasCreatorField(name):
			unit.CreatorFields = append(unit.CreatorFields, name)
		}
	}
	if unit.Status == domain.UnitApproved && len(brief.MissingFields()) > 0 {
		unit.Status = domain.UnitPending
	}
	if unit.Content, err = json.Marshal(brief); err != nil {
		return domain.ReviewUnit{}, err
	}
	return uc.units.SaveUnit(ctx, unit)
}

// Redo has the AI write the brief again with the Creator's note.
func (uc *CreativeBriefUseCase) Redo(ctx context.Context, projectID, key, note string) (domain.ReviewUnit, error) {
	if key != domain.BriefUnitKey {
		return domain.ReviewUnit{}, domain.ErrUnitNotFound
	}
	return uc.Generate(ctx, projectID, note)
}

// Impact lists what a change to the brief sends back.
func (uc *CreativeBriefUseCase) Impact(ctx context.Context, projectID, key string) ([]string, error) {
	if key != domain.BriefUnitKey {
		return nil, domain.ErrUnitNotFound
	}
	return briefDownstream(ctx, uc.units, uc.store, projectID)
}

// briefDownstream lists what a change to the brief sends back: the story
// outline, when one was written, and the research questions the AI wrote from
// the brief.
func briefDownstream(ctx context.Context, units ReviewUnitPort, store BriefStorePort, projectID string) ([]string, error) {
	link, err := store.StoryBriefLink(ctx, projectID)
	if err != nil {
		return nil, err
	}
	items := []string{}
	if link.HasStory {
		items = append(items, "Dàn ý và lời thoại đã viết ở bước Viết kịch bản: cần viết lại theo brief mới")
	}
	questions, err := units.ListUnits(ctx, projectID, domain.UnitKindResearchQuestion)
	if err != nil {
		return nil, err
	}
	for _, q := range questions {
		if !q.HasCreatorField("text") {
			items = append(items, "Câu hỏi nghiên cứu do AI sinh: cần sinh lại theo brief mới (dữ kiện đã có giữ nguyên)")
			break
		}
	}
	return items, nil
}

// CheckApprovable refuses a brief that still has empty fields.
func (uc *CreativeBriefUseCase) CheckApprovable(_ context.Context, unit domain.ReviewUnit) error {
	var brief domain.CreativeBrief
	if err := json.Unmarshal(unit.Content, &brief); err != nil {
		return fmt.Errorf("read stored brief: %w", err)
	}
	if len(brief.MissingFields()) > 0 {
		return fmt.Errorf("%w: %s", ErrBriefIncomplete, brief.MissingFieldLabels())
	}
	return nil
}

// ApprovedBriefSection is what {{creative_brief}} renders to in the story
// prompt: the approved idea and brief with the target length, or a placeholder
// telling the model there is none.
func (uc *CreativeBriefUseCase) ApprovedBriefSection(ctx context.Context, projectID string, project *domain.Project) (string, error) {
	idea, brief, approved, err := loadApprovedBrief(ctx, uc.units, projectID)
	if err != nil {
		return "", err
	}
	if !approved {
		return creativeBriefPlaceholder, nil
	}
	duration, err := uc.store.GetTargetDuration(ctx, projectID)
	if err != nil {
		return "", err
	}
	label := domain.TargetDurationLabel(project.VideoOutputMode, duration, briefLanguage(project.ContentLanguage))
	return briefSectionFor(ctx, uc.units, projectID, project.VideoOutputMode, idea, brief, label)
}

func briefLanguage(lang domain.ContentLanguage) string {
	if lang == domain.LanguageEnglish {
		return "en"
	}
	return "vi"
}

func briefLanguageName(language string) string {
	if language == "en" {
		return "English"
	}
	return "tiếng Việt"
}

func videoKindLabel(mode domain.VideoOutputMode) string {
	if mode == domain.ModeShortOnly {
		return "Shorts 9:16"
	}
	return "Video dài 16:9"
}

func keptFieldsText(brief domain.CreativeBrief, creatorFields []string) string {
	if len(creatorFields) == 0 {
		return "(không có)"
	}
	var lines []string
	for _, f := range domain.BriefFields {
		for _, kept := range creatorFields {
			if kept == f {
				lines = append(lines, "- "+f+": "+brief.Field(f))
			}
		}
	}
	return strings.Join(lines, "\n")
}

func noteText(note string) string {
	if strings.TrimSpace(note) == "" {
		return "(không có)"
	}
	return strings.TrimSpace(note)
}

// withoutField returns fields with name removed.
func withoutField(fields []string, name string) []string {
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if f != name {
			out = append(out, f)
		}
	}
	return out
}
