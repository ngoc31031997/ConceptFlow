package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"authoring/internal/domain"
)

// GateView is where the research step stands as the screen shows it.
type GateView struct {
	// Status is approved once the Creator approved the fact table.
	Status domain.UnitStatus `json:"status"`
	// Skipped is true while the Creator has skipped the step.
	Skipped bool `json:"skipped"`
	// Resolved is true when the step no longer holds the flow back: the table
	// is approved or the step is skipped.
	Resolved bool `json:"resolved"`
}

// QuestionView is one research question as the screen shows it.
type QuestionView struct {
	ID   string `json:"id"`
	Text string `json:"text"`
	// Mine is true for a question the Creator wrote or edited; an AI regeneration
	// keeps it.
	Mine bool `json:"mine"`
	// Stale is true for a question the AI wrote from a brief that has changed since.
	Stale bool `json:"stale"`
}

// FactView is one fact with its code and the fields the Creator wrote.
type FactView struct {
	ID string `json:"id"`
	domain.Fact
	// Mine lists the fields the Creator wrote or changed by hand; pasting a
	// new answer keeps them.
	Mine []string `json:"mine"`
}

// ResearchState is everything the research screen shows.
type ResearchState struct {
	Gate      GateView                  `json:"gate"`
	Questions []QuestionView            `json:"questions"`
	Facts     []FactView                `json:"facts"`
	Pastes    []PasteLog                `json:"pastes"`
	Coverage  []domain.QuestionCoverage `json:"coverage"`
	// BriefApproved is true when the idea step's brief is approved.
	BriefApproved bool `json:"brief_approved"`
	// QuestionsStale is true when some AI-written question was written from a
	// brief that has changed.
	QuestionsStale bool `json:"questions_stale"`
}

// ResearchDeskUseCase backs the research step: the questions to research, the
// prompt the Creator takes to an outside research tool, the pasted answers read
// into a table of facts, and the Creator's approval or skipping of the step.
// The questions, the facts and the step's gate are review units (UnitKind for
// domain.UnitKindResearchQuestion, UnitKindFact and UnitKindResearch).
type ResearchDeskUseCase struct {
	units           ResearchStorePort
	projects        ProjectReaderPort
	locks           AuthoringLockPort
	prompts         PromptActivePort
	provider        LLMProviderPort
	recorder        *LLMUsageRecorder
	maxOutputTokens int
	verifier        *SourceVerifier
	dependents      []FactDependentsFinder
	now             func() time.Time
}

// FactDependentsFinder lists, in the Creator's words, the work in later steps
// that uses a fact, so that changing or dropping the fact can say what it sends
// back. Each later step that uses facts registers one.
type FactDependentsFinder func(ctx context.Context, projectID, factID string) ([]string, error)

// WithVerifier lets the use case check the pages facts cite. Without one a
// fact with a link is shown as not checkable.
func (uc *ResearchDeskUseCase) WithVerifier(v *SourceVerifier) *ResearchDeskUseCase {
	uc.verifier = v
	return uc
}

// RegisterFactDependents adds a finder of the work that uses facts.
func (uc *ResearchDeskUseCase) RegisterFactDependents(f FactDependentsFinder) *ResearchDeskUseCase {
	uc.dependents = append(uc.dependents, f)
	return uc
}

// NewResearchDeskUseCase wires the use case. provider may be nil when no
// language model is configured: the Creator then writes the questions by hand.
func NewResearchDeskUseCase(
	units ResearchStorePort, projects ProjectReaderPort, locks AuthoringLockPort, prompts PromptActivePort,
	provider LLMProviderPort, recorder *LLMUsageRecorder, maxOutputTokens int,
) *ResearchDeskUseCase {
	return &ResearchDeskUseCase{
		units: units, projects: projects, locks: locks, prompts: prompts,
		provider: provider, recorder: recorder, maxOutputTokens: maxOutputTokens, now: time.Now,
	}
}

// State returns the whole research screen.
func (uc *ResearchDeskUseCase) State(ctx context.Context, projectID string) (ResearchState, error) {
	var state ResearchState
	_, gate, err := uc.gate(ctx, projectID)
	if err != nil {
		return ResearchState{}, err
	}
	gateUnit, err := uc.units.GetUnit(ctx, projectID, domain.UnitKindResearch, domain.ResearchUnitKey)
	status := domain.UnitPending
	switch {
	case err == nil:
		status = gateUnit.Status
	case !errors.Is(err, domain.ErrUnitNotFound):
		return ResearchState{}, err
	}
	state.Gate = GateView{Status: status, Skipped: gate.Skipped, Resolved: status == domain.UnitApproved || gate.Skipped}

	if state.Questions, err = uc.questions(ctx, projectID); err != nil {
		return ResearchState{}, err
	}
	for _, q := range state.Questions {
		if q.Stale {
			state.QuestionsStale = true
		}
	}
	if state.Facts, err = uc.facts(ctx, projectID); err != nil {
		return ResearchState{}, err
	}
	if state.Pastes, err = uc.units.ListPastes(ctx, projectID); err != nil {
		return ResearchState{}, err
	}
	state.Coverage = coverageOf(state.Questions, state.Facts)
	brief, err := uc.units.GetUnit(ctx, projectID, domain.UnitKindBrief, domain.BriefUnitKey)
	switch {
	case err == nil:
		state.BriefApproved = brief.Status == domain.UnitApproved
	case !errors.Is(err, domain.ErrUnitNotFound):
		return ResearchState{}, err
	}
	return state, nil
}

// gate loads the step's gate unit, or a new empty one when none is stored yet.
func (uc *ResearchDeskUseCase) gate(ctx context.Context, projectID string) (domain.ReviewUnit, domain.ResearchGate, error) {
	unit, err := uc.units.GetUnit(ctx, projectID, domain.UnitKindResearch, domain.ResearchUnitKey)
	if errors.Is(err, domain.ErrUnitNotFound) {
		return domain.ReviewUnit{
			ProjectID: projectID, Kind: domain.UnitKindResearch, Key: domain.ResearchUnitKey, Status: domain.UnitPending,
		}, domain.ResearchGate{}, nil
	}
	if err != nil {
		return domain.ReviewUnit{}, domain.ResearchGate{}, err
	}
	var gate domain.ResearchGate
	if err := json.Unmarshal(unit.Content, &gate); err != nil {
		return domain.ReviewUnit{}, domain.ResearchGate{}, fmt.Errorf("read research gate: %w", err)
	}
	return unit, gate, nil
}

// saveGate stores the gate unit with its content.
func (uc *ResearchDeskUseCase) saveGate(ctx context.Context, unit domain.ReviewUnit, gate domain.ResearchGate) (domain.ReviewUnit, error) {
	raw, err := json.Marshal(gate)
	if err != nil {
		return domain.ReviewUnit{}, err
	}
	unit.Content = raw
	return uc.units.SaveUnit(ctx, unit)
}

// facts loads every fact of the project in code order.
func (uc *ResearchDeskUseCase) facts(ctx context.Context, projectID string) ([]FactView, error) {
	units, err := uc.units.ListUnits(ctx, projectID, domain.UnitKindFact)
	if err != nil {
		return nil, err
	}
	out := make([]FactView, 0, len(units))
	for _, u := range units {
		var fact domain.Fact
		if err := json.Unmarshal(u.Content, &fact); err != nil {
			return nil, fmt.Errorf("read stored fact %s: %w", u.Key, err)
		}
		mine := u.CreatorFields
		if mine == nil {
			mine = []string{}
		}
		out = append(out, FactView{ID: u.Key, Fact: fact, Mine: mine})
	}
	return out, nil
}

// coverageOf counts the independent sources of each question.
func coverageOf(questions []QuestionView, facts []FactView) []domain.QuestionCoverage {
	ids := make([]string, 0, len(questions))
	for _, q := range questions {
		ids = append(ids, q.ID)
	}
	factIDs := make([]string, 0, len(facts))
	byID := make(map[string]domain.Fact, len(facts))
	for _, f := range facts {
		factIDs = append(factIDs, f.ID)
		byID[f.ID] = f.Fact
	}
	return domain.Coverage(ids, factIDs, byID)
}

// durationLabel is the project's target length as the prompts say it.
func (uc *ResearchDeskUseCase) durationLabel(ctx context.Context, projectID string, project *domain.Project) (string, error) {
	duration, err := uc.units.GetTargetDuration(ctx, projectID)
	if err != nil {
		return "", err
	}
	return domain.TargetDurationLabel(project.VideoOutputMode, duration, briefLanguage(project.ContentLanguage)), nil
}

// brief loads the idea step's brief; a project without one cannot be researched.
func (uc *ResearchDeskUseCase) brief(ctx context.Context, projectID string) (domain.ReviewUnit, domain.CreativeBrief, error) {
	unit, err := uc.units.GetUnit(ctx, projectID, domain.UnitKindBrief, domain.BriefUnitKey)
	if errors.Is(err, domain.ErrUnitNotFound) {
		return domain.ReviewUnit{}, domain.CreativeBrief{}, fmt.Errorf("%w: chưa có brief, viết brief ở bước Ý tưởng trước", domain.ErrInvalidStatus)
	}
	if err != nil {
		return domain.ReviewUnit{}, domain.CreativeBrief{}, err
	}
	var brief domain.CreativeBrief
	if err := json.Unmarshal(unit.Content, &brief); err != nil {
		return domain.ReviewUnit{}, domain.CreativeBrief{}, fmt.Errorf("read stored brief: %w", err)
	}
	return unit, brief, nil
}
