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

// storyBriefRef names the brief as the unit the story fields are built from.
const storyBriefRef = domain.UnitKindBrief + ":" + domain.BriefUnitKey

// ErrStoryNeedsBrief is returned when the story is asked for before the brief is approved.
var ErrStoryNeedsBrief = errors.New("duyệt brief ở bước Ý tưởng trước")

// ErrStoryNeedsResearch is returned when the story is asked for before the
// research step is approved or skipped.
var ErrStoryNeedsResearch = errors.New("duyệt bảng dữ kiện hoặc bỏ qua bước Nghiên cứu trước")

// ErrStoryNothingToWrite is returned when every field of the story is the Creator's.
var ErrStoryNothingToWrite = errors.New("cả sáu ô đều do bạn viết, không có ô nào để AI viết")

// StoryStorePort is what the story step reads and writes: its review units,
// the project's topic and length, and the model choices.
type StoryStorePort interface {
	ReviewUnitPort
	GetAuthoringTopic(ctx context.Context, projectID string) (string, error)
	GetAuthoringModels(ctx context.Context, projectID string) (domain.AuthoringStepModels, error)
	GetTargetDuration(ctx context.Context, projectID string) (domain.TargetDuration, error)
}

// StoryArchetypePort is the Kiểu video table as the story step uses it.
type StoryArchetypePort interface {
	List(ctx context.Context) ([]domain.VideoArchetype, error)
	Create(ctx context.Context, a domain.VideoArchetype) (domain.VideoArchetype, error)
}

// StoryFieldView is one field of the story as the screen shows it.
type StoryFieldView struct {
	Key   string   `json:"key"`
	Label string   `json:"label"`
	Text  string   `json:"text"`
	Facts []string `json:"facts"`
	// Mine is true for a field the Creator wrote or edited; an AI run keeps it.
	Mine bool `json:"mine"`
	// Stale is true for a field the AI wrote from a brief that has changed since.
	Stale bool `json:"stale"`
}

// StoryApproachView is one proposed approach with its key.
type StoryApproachView struct {
	Key string `json:"key"`
	domain.StoryApproach
	Selected bool `json:"selected"`
}

// StoryFactView is one kept fact as the story step lists it.
type StoryFactView struct {
	ID      string          `json:"id"`
	ClaimVI string          `json:"claim_vi"`
	Type    domain.FactType `json:"type"`
	// Must is true for a fact the script has to state; it is always used.
	Must bool `json:"must"`
	// Used is true when the story uses the fact.
	Used bool `json:"used"`
	// Cited is true when the chain or the payoff rests on the fact.
	Cited bool             `json:"cited"`
	Link  domain.LinkState `json:"link"`
}

// StoryState is everything the story screen shows.
type StoryState struct {
	Status      domain.UnitStatus   `json:"status"`
	ApproachKey string              `json:"approach_key"`
	Fields      []StoryFieldView    `json:"fields"`
	Approaches  []StoryApproachView `json:"approaches"`
	Facts       []StoryFactView     `json:"facts"`
	// BriefApproved and ResearchResolved are the two things the step waits for.
	BriefApproved    bool `json:"brief_approved"`
	ResearchResolved bool `json:"research_resolved"`
	// ResearchSkipped is true while the Creator has skipped the research step.
	ResearchSkipped bool `json:"research_skipped"`
}

// StoryDeskUseCase backs the story step: the approaches the AI proposes, the
// six fields of the story frame, the facts the story uses and the Creator's
// approval. The fields, the approaches and the gate are review units (UnitKind
// for domain.UnitKindStoryField and domain.UnitKindStory).
type StoryDeskUseCase struct {
	units           StoryStorePort
	projects        ProjectReaderPort
	locks           AuthoringLockPort
	prompts         PromptActivePort
	archetypes      StoryArchetypePort
	provider        LLMProviderPort
	recorder        *LLMUsageRecorder
	maxOutputTokens int
	now             func() time.Time
}

// NewStoryDeskUseCase wires the use case. provider may be nil when no language
// model is configured: the Creator then writes the fields by hand.
func NewStoryDeskUseCase(
	units StoryStorePort, projects ProjectReaderPort, locks AuthoringLockPort, prompts PromptActivePort,
	archetypes StoryArchetypePort, provider LLMProviderPort, recorder *LLMUsageRecorder, maxOutputTokens int,
) *StoryDeskUseCase {
	return &StoryDeskUseCase{
		units: units, projects: projects, locks: locks, prompts: prompts, archetypes: archetypes,
		provider: provider, recorder: recorder, maxOutputTokens: maxOutputTokens, now: time.Now,
	}
}

// storyFact is a kept fact with the content the prompts and the screen need.
type storyFact struct {
	ID string
	domain.Fact
}

// keptFacts loads the facts the Creator has not dropped, in code order.
func (uc *StoryDeskUseCase) keptFacts(ctx context.Context, projectID string) ([]storyFact, error) {
	return loadKeptFacts(ctx, uc.units, projectID)
}

// loadKeptFacts loads the facts of a project the Creator has not dropped, in code order.
func loadKeptFacts(ctx context.Context, units ReviewUnitPort, projectID string) ([]storyFact, error) {
	stored, err := units.ListUnits(ctx, projectID, domain.UnitKindFact)
	if err != nil {
		return nil, err
	}
	out := make([]storyFact, 0, len(stored))
	for _, u := range stored {
		var f domain.Fact
		if err := json.Unmarshal(u.Content, &f); err != nil {
			return nil, fmt.Errorf("read stored fact %s: %w", u.Key, err)
		}
		if f.Keep {
			out = append(out, storyFact{ID: u.Key, Fact: f})
		}
	}
	return out, nil
}

// allFactIDs lists the code of every fact unit, dropped or not.
func (uc *StoryDeskUseCase) allFactIDs(ctx context.Context, projectID string) (map[string]bool, error) {
	units, err := uc.units.ListUnits(ctx, projectID, domain.UnitKindFact)
	if err != nil {
		return nil, err
	}
	ids := make(map[string]bool, len(units))
	for _, u := range units {
		ids[u.Key] = true
	}
	return ids, nil
}

func factKnownTo(facts []storyFact) func(string) bool {
	set := make(map[string]bool, len(facts))
	for _, f := range facts {
		set[f.ID] = true
	}
	return func(id string) bool { return set[id] }
}

// gate loads the step's gate unit, or a new empty one when none is stored yet.
func (uc *StoryDeskUseCase) gate(ctx context.Context, projectID string) (domain.ReviewUnit, domain.StoryGate, error) {
	unit, err := uc.units.GetUnit(ctx, projectID, domain.UnitKindStory, domain.StoryUnitKey)
	if errors.Is(err, domain.ErrUnitNotFound) {
		return domain.ReviewUnit{
			ProjectID: projectID, Kind: domain.UnitKindStory, Key: domain.StoryUnitKey, Status: domain.UnitPending,
		}, domain.StoryGate{UsedFacts: []string{}}, nil
	}
	if err != nil {
		return domain.ReviewUnit{}, domain.StoryGate{}, err
	}
	var gate domain.StoryGate
	if err := json.Unmarshal(unit.Content, &gate); err != nil {
		return domain.ReviewUnit{}, domain.StoryGate{}, fmt.Errorf("read stored story gate: %w", err)
	}
	if gate.UsedFacts == nil {
		gate.UsedFacts = []string{}
	}
	return unit, gate, nil
}

func (uc *StoryDeskUseCase) saveGate(ctx context.Context, unit domain.ReviewUnit, gate domain.StoryGate) (domain.ReviewUnit, error) {
	if gate.UsedFacts == nil {
		gate.UsedFacts = []string{}
	}
	raw, err := json.Marshal(gate)
	if err != nil {
		return domain.ReviewUnit{}, err
	}
	unit.Content = raw
	return uc.units.SaveUnit(ctx, unit)
}

// reopen sends an approved story back to review: the Creator approved it as it was.
func (uc *StoryDeskUseCase) reopen(ctx context.Context, projectID string) error {
	unit, gate, err := uc.gate(ctx, projectID)
	if err != nil {
		return err
	}
	if unit.Status == domain.UnitApproved {
		unit.Status = domain.UnitPending
	}
	_, err = uc.saveGate(ctx, unit, gate)
	return err
}

// fieldUnits loads the stored story fields by key.
func (uc *StoryDeskUseCase) fieldUnits(ctx context.Context, projectID string) (map[string]domain.ReviewUnit, error) {
	units, err := uc.units.ListUnits(ctx, projectID, domain.UnitKindStoryField)
	if err != nil {
		return nil, err
	}
	out := make(map[string]domain.ReviewUnit, len(units))
	for _, u := range units {
		out[u.Key] = u
	}
	return out, nil
}

func readStoryField(u domain.ReviewUnit) (domain.StoryField, error) {
	var f domain.StoryField
	if err := json.Unmarshal(u.Content, &f); err != nil {
		return domain.StoryField{}, fmt.Errorf("read stored story field %s: %w", u.Key, err)
	}
	if f.Facts == nil {
		f.Facts = []string{}
	}
	return f, nil
}

// researchStanding reads whether the research step is approved or skipped.
func (uc *StoryDeskUseCase) researchStanding(ctx context.Context, projectID string) (resolved, skipped bool, err error) {
	unit, err := uc.units.GetUnit(ctx, projectID, domain.UnitKindResearch, domain.ResearchUnitKey)
	if errors.Is(err, domain.ErrUnitNotFound) {
		return false, false, nil
	}
	if err != nil {
		return false, false, err
	}
	var gate domain.ResearchGate
	if err := json.Unmarshal(unit.Content, &gate); err != nil {
		return false, false, fmt.Errorf("read research gate: %w", err)
	}
	return unit.Status == domain.UnitApproved || gate.Skipped, gate.Skipped, nil
}

// approvedBrief returns the idea and the brief when both are approved, and
// reports whether they are.
func (uc *StoryDeskUseCase) approvedBrief(ctx context.Context, projectID string) (domain.Idea, domain.CreativeBrief, bool, error) {
	return loadApprovedBrief(ctx, uc.units, projectID)
}

// loadApprovedBrief returns the idea and the brief of a project, and reports
// whether both are approved.
func loadApprovedBrief(ctx context.Context, units ReviewUnitPort, projectID string) (domain.Idea, domain.CreativeBrief, bool, error) {
	ideaUnit, idea, ideaExists, err := loadIdea(ctx, units, projectID)
	if err != nil {
		return domain.Idea{}, domain.CreativeBrief{}, false, err
	}
	unit, err := units.GetUnit(ctx, projectID, domain.UnitKindBrief, domain.BriefUnitKey)
	if errors.Is(err, domain.ErrUnitNotFound) {
		return idea, domain.CreativeBrief{}, false, nil
	}
	if err != nil {
		return domain.Idea{}, domain.CreativeBrief{}, false, err
	}
	var brief domain.CreativeBrief
	if err := json.Unmarshal(unit.Content, &brief); err != nil {
		return domain.Idea{}, domain.CreativeBrief{}, false, fmt.Errorf("read stored brief: %w", err)
	}
	approved := ideaExists && ideaUnit.Status == domain.UnitApproved && unit.Status == domain.UnitApproved
	return idea, brief, approved, nil
}

// State returns the whole story screen.
func (uc *StoryDeskUseCase) State(ctx context.Context, projectID string) (StoryState, error) {
	state := StoryState{Fields: []StoryFieldView{}, Approaches: []StoryApproachView{}, Facts: []StoryFactView{}}
	gateUnit, gate, err := uc.gate(ctx, projectID)
	if err != nil {
		return StoryState{}, err
	}
	state.Status, state.ApproachKey = gateUnit.Status, gate.ApproachKey
	if _, _, state.BriefApproved, err = uc.approvedBrief(ctx, projectID); err != nil {
		return StoryState{}, err
	}
	if state.ResearchResolved, state.ResearchSkipped, err = uc.researchStanding(ctx, projectID); err != nil {
		return StoryState{}, err
	}

	stored, err := uc.fieldUnits(ctx, projectID)
	if err != nil {
		return StoryState{}, err
	}
	var cited []string
	for _, key := range domain.StoryFieldKeys {
		view := StoryFieldView{Key: key, Label: domain.StoryFieldLabel(key), Facts: []string{}}
		if u, ok := stored[key]; ok {
			f, err := readStoryField(u)
			if err != nil {
				return StoryState{}, err
			}
			view.Text, view.Facts = f.Text, f.Facts
			view.Mine, view.Stale = u.HasCreatorField(domain.TextFieldName), u.Stale
			cited = append(cited, f.Facts...)
		}
		state.Fields = append(state.Fields, view)
	}

	if state.Approaches, err = uc.approachViews(ctx, projectID, gate.ApproachKey); err != nil {
		return StoryState{}, err
	}
	facts, err := uc.keptFacts(ctx, projectID)
	if err != nil {
		return StoryState{}, err
	}
	used := toSet(gate.UsedFacts)
	citedSet := toSet(cited)
	for _, f := range facts {
		state.Facts = append(state.Facts, StoryFactView{
			ID: f.ID, ClaimVI: f.ClaimVI, Type: f.Type, Must: f.Must,
			Used: f.Must || used[f.ID] || citedSet[f.ID], Cited: citedSet[f.ID], Link: f.Link.State,
		})
	}
	return state, nil
}

func toSet(ids []string) map[string]bool {
	set := make(map[string]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}
	return set
}

// SetFactUse turns the use of one kept fact in the story on or off. A fact the
// script must state is always used. Changing the use sends an approved story
// back to review.
func (uc *StoryDeskUseCase) SetFactUse(ctx context.Context, projectID, factID string, use bool) error {
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return err
	}
	facts, err := uc.keptFacts(ctx, projectID)
	if err != nil {
		return err
	}
	var fact *storyFact
	for i := range facts {
		if facts[i].ID == factID {
			fact = &facts[i]
		}
	}
	if fact == nil {
		return fmt.Errorf("%w: dữ kiện %s không có trong bảng dữ kiện", domain.ErrUnitNotFound, factID)
	}
	if fact.Must {
		return fmt.Errorf("%w: dữ kiện %s bắt buộc nói nên luôn được dùng", domain.ErrInvalidUnitEdit, factID)
	}
	unit, gate, err := uc.gate(ctx, projectID)
	if err != nil {
		return err
	}
	rest := make([]string, 0, len(gate.UsedFacts)+1)
	for _, id := range gate.UsedFacts {
		if id != factID {
			rest = append(rest, id)
		}
	}
	if use {
		rest = append(rest, factID)
	}
	gate.UsedFacts = rest
	unit.Status = domain.UnitPending
	_, err = uc.saveGate(ctx, unit, gate)
	return err
}

// FactDependents lists, for the research step, the story fields that rest on a
// fact and whether the story uses it, so that dropping or changing the fact
// says what it sends back.
func (uc *StoryDeskUseCase) FactDependents() FactDependentsFinder {
	return func(ctx context.Context, projectID, factID string) ([]string, error) {
		stored, err := uc.fieldUnits(ctx, projectID)
		if err != nil {
			return nil, err
		}
		var items []string
		for _, key := range domain.StoryFieldKeys {
			u, ok := stored[key]
			if !ok {
				continue
			}
			f, err := readStoryField(u)
			if err != nil {
				return nil, err
			}
			for _, id := range f.Facts {
				if id == factID {
					items = append(items, fmt.Sprintf("Câu chuyện · %s đang trích %s: câu chuyện được duyệt lại", strings.ToLower(domain.StoryFieldLabel(key)), factID))
				}
			}
		}
		if len(items) > 0 {
			return items, nil
		}
		_, gate, err := uc.gate(ctx, projectID)
		if err != nil {
			return nil, err
		}
		for _, id := range gate.UsedFacts {
			if id == factID {
				return []string{fmt.Sprintf("Câu chuyện dùng %s: câu chuyện được duyệt lại", factID)}, nil
			}
		}
		return nil, nil
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

// storyContext is what every AI call of the step reads.
type storyContext struct {
	project  *domain.Project
	topic    string
	brief    string
	language string
	facts    []storyFact
	skipped  bool
	// model is the model the project chose for this step; "" is the server default.
	model string
}

// context gathers the inputs of an AI call and refuses when the step is not
// open yet: the brief must be approved and the research step approved or skipped.
func (uc *StoryDeskUseCase) context(ctx context.Context, projectID string) (storyContext, error) {
	var sc storyContext
	idea, brief, approved, err := uc.approvedBrief(ctx, projectID)
	if err != nil {
		return sc, err
	}
	if !approved {
		return sc, fmt.Errorf("%w: %w", domain.ErrInvalidStatus, ErrStoryNeedsBrief)
	}
	resolved, skipped, err := uc.researchStanding(ctx, projectID)
	if err != nil {
		return sc, err
	}
	if !resolved {
		return sc, fmt.Errorf("%w: %w", domain.ErrInvalidStatus, ErrStoryNeedsResearch)
	}
	if sc.project, err = uc.projects.Get(ctx, projectID); err != nil {
		return sc, fmt.Errorf("load project: %w", err)
	}
	if sc.topic, err = uc.units.GetAuthoringTopic(ctx, projectID); err != nil {
		return sc, fmt.Errorf("load topic: %w", err)
	}
	duration, err := uc.units.GetTargetDuration(ctx, projectID)
	if err != nil {
		return sc, err
	}
	sc.language = briefLanguage(sc.project.ContentLanguage)
	if sc.brief, err = briefSectionFor(ctx, uc.units, projectID, sc.project.VideoOutputMode, idea, brief, domain.TargetDurationLabel(sc.project.VideoOutputMode, duration, sc.language)); err != nil {
		return sc, err
	}
	if sc.facts, err = uc.keptFacts(ctx, projectID); err != nil {
		return sc, err
	}
	sc.skipped = skipped
	models, err := uc.units.GetAuthoringModels(ctx, projectID)
	if err != nil {
		return sc, fmt.Errorf("load authoring models: %w", err)
	}
	sc.model = models.ModelFor("story_frame")
	return sc, nil
}

// factsText lists the kept facts for a prompt: code, kind, whether it must be
// stated, whether it has a source, and the claim.
func factsText(facts []storyFact) string {
	if len(facts) == 0 {
		return "(chưa có dữ kiện nào)"
	}
	var sb strings.Builder
	for i, f := range facts {
		if i > 0 {
			sb.WriteString("\n")
		}
		tags := []string{string(f.Type)}
		if f.Must {
			tags = append(tags, "bắt buộc nói")
		}
		if f.Link.State == domain.LinkNone {
			tags = append(tags, "chưa có nguồn")
		}
		sb.WriteString(f.ID + " [" + strings.Join(tags, " · ") + "]: " + f.ClaimVI)
		if f.Note != "" {
			sb.WriteString(" (ghi chú: " + f.Note + ")")
		}
	}
	return sb.String()
}

// chat sends one system prompt and records the usage.
func (uc *StoryDeskUseCase) chat(ctx context.Context, role domain.PromptRole, step, projectID, model, system string) (string, error) {
	started := uc.now()
	result, err := uc.provider.Chat(ctx, ChatRequest{
		System:      system,
		User:        userTurnNudge,
		MaxTokens:   uc.maxOutputTokens,
		Temperature: 0.7,
		Model:       model,
		OnProgress:  ProgressFrom(ctx),
	})
	if uc.recorder != nil {
		uc.recorder.Record(ctx, RecordFor(uc.provider.Name(), string(role), step, projectID, result.Usage, started, err))
	}
	if err != nil {
		return "", err
	}
	return result.Content, nil
}
