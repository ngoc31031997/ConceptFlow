package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"authoring/internal/domain"
)

// beatStoryRef names the approved story as the input the beats are built from.
// The review unit store resolves it from the project's story, so that a beat
// written from a story that has changed since reads as stale.
const beatStoryRef = domain.StoryInputKind + ":" + domain.StoryInputKey

// ErrBeatsNeedStory is returned when the beat sheet is asked for before the story is approved.
var ErrBeatsNeedStory = errors.New("duyệt câu chuyện ở bước Câu chuyện trước")

// ErrBeatSheetFull is returned when a beat would take the sheet past domain.MaxBeats.
var ErrBeatSheetFull = fmt.Errorf("beat sheet đã có %d beat, không thêm được nữa", domain.MaxBeats)

// BeatFactView is one kept fact as the beat sheet lists it.
type BeatFactView struct {
	ID      string           `json:"id"`
	ClaimVI string           `json:"claim_vi"`
	Type    domain.FactType  `json:"type"`
	Must    bool             `json:"must"`
	Link    domain.LinkState `json:"link"`
}

// BeatView is one beat as the screen shows it.
type BeatView struct {
	Key string `json:"key"`
	domain.Beat
	Status domain.UnitStatus `json:"status"`
	// Mine lists the fields the Creator wrote; an AI run keeps them.
	Mine []string `json:"mine"`
	// Stale is true for a beat the AI wrote from a story that has changed since.
	Stale bool `json:"stale"`
	// OnlyMust lists the facts the script must state that no other beat holds,
	// so deleting this beat leaves them out of the video.
	OnlyMust []string `json:"only_must"`
	// NoFact is true for a beat other than the hook that cites no fact, so
	// nothing backs what it says.
	NoFact bool `json:"no_fact"`
}

// The phases of the step, in the order the Creator goes through them.
const (
	// BeatPhaseEmpty is a sheet with no beat yet.
	BeatPhaseEmpty = "empty"
	// BeatPhaseSkeleton is a sheet whose functions and facts the Creator is still reviewing.
	BeatPhaseSkeleton = "skeleton"
	// BeatPhaseContent is a sheet whose skeleton is approved, so its content is written and reviewed.
	BeatPhaseContent = "content"
)

// BeatState is everything the beat sheet screen shows.
type BeatState struct {
	// Status is approved when the sheet has beats and every one is approved.
	Status        domain.UnitStatus  `json:"status"`
	StoryApproved bool               `json:"story_approved"`
	Beats         []BeatView         `json:"beats"`
	Facts         []BeatFactView     `json:"facts"`
	Checks        []domain.BeatCheck `json:"checks"`
	TotalSeconds  int                `json:"total_seconds"`
	// Phase is one of the BeatPhase constants.
	Phase string `json:"phase"`
	// Unwritten counts the beats whose content is not written yet.
	Unwritten int `json:"unwritten"`
	// Rebalanced is how many beats the last content run had to resize so the
	// total fits the target.
	Rebalanced int `json:"rebalanced"`
	// TargetLabel is the length the video aims for, as the screen says it.
	TargetLabel string `json:"target_label"`
	// ScriptApproved counts the beats whose dialogue is approved; moving a
	// beat changes how it joins its neighbours, so the screen asks first.
	ScriptApproved int `json:"script_approved"`
}

// BeatDeskUseCase backs the beat sheet step: the beats the AI writes from the
// approved story, their order, and the Creator's review of each. The beats and
// the gate that holds their order are review units (UnitKind for
// domain.UnitKindBeat and domain.UnitKindBeatSheet).
type BeatDeskUseCase struct {
	units           StepUnitStorePort
	stories         ApprovedStoryPort
	projects        ProjectReaderPort
	locks           AuthoringLockPort
	prompts         PromptActivePort
	provider        LLMProviderPort
	recorder        *LLMUsageRecorder
	maxOutputTokens int
	now             func() time.Time
}

// NewBeatDeskUseCase wires the use case. provider may be nil when no language
// model is configured: the Creator then adds and writes the beats by hand.
func NewBeatDeskUseCase(
	units StepUnitStorePort, stories ApprovedStoryPort, projects ProjectReaderPort, locks AuthoringLockPort, prompts PromptActivePort,
	provider LLMProviderPort, recorder *LLMUsageRecorder, maxOutputTokens int,
) *BeatDeskUseCase {
	return &BeatDeskUseCase{
		units: units, stories: stories, projects: projects, locks: locks, prompts: prompts,
		provider: provider, recorder: recorder, maxOutputTokens: maxOutputTokens, now: time.Now,
	}
}

// storedBeat is a beat unit with its content read.
type storedBeat struct {
	unit domain.ReviewUnit
	domain.Beat
}

func readBeat(u domain.ReviewUnit) (domain.Beat, error) {
	var b domain.Beat
	if err := json.Unmarshal(u.Content, &b); err != nil {
		return domain.Beat{}, fmt.Errorf("read stored beat %s: %w", u.Key, err)
	}
	if b.Facts == nil {
		b.Facts = []string{}
	}
	return b, nil
}

// sheet loads the gate unit, or a new empty one when none is stored yet.
func (uc *BeatDeskUseCase) sheet(ctx context.Context, projectID string) (domain.ReviewUnit, domain.BeatSheetGate, error) {
	unit, err := uc.units.GetUnit(ctx, projectID, domain.UnitKindBeatSheet, domain.BeatSheetUnitKey)
	if errors.Is(err, domain.ErrUnitNotFound) {
		return domain.ReviewUnit{
			ProjectID: projectID, Kind: domain.UnitKindBeatSheet, Key: domain.BeatSheetUnitKey, Status: domain.UnitPending,
		}, domain.BeatSheetGate{Order: []string{}, NextSeq: 1}, nil
	}
	if err != nil {
		return domain.ReviewUnit{}, domain.BeatSheetGate{}, err
	}
	var gate domain.BeatSheetGate
	if err := json.Unmarshal(unit.Content, &gate); err != nil {
		return domain.ReviewUnit{}, domain.BeatSheetGate{}, fmt.Errorf("read stored beat sheet: %w", err)
	}
	if gate.Order == nil {
		gate.Order = []string{}
	}
	if gate.NextSeq < 1 {
		gate.NextSeq = 1
	}
	return unit, gate, nil
}

func (uc *BeatDeskUseCase) saveSheet(ctx context.Context, unit domain.ReviewUnit, gate domain.BeatSheetGate) error {
	raw, err := json.Marshal(gate)
	if err != nil {
		return err
	}
	unit.Content = raw
	_, err = uc.units.SaveUnit(ctx, unit)
	return err
}

// beats loads the beats in the order the sheet holds them. A stored beat the
// order does not name is left out: it is a leftover of an interrupted change.
func (uc *BeatDeskUseCase) beats(ctx context.Context, projectID string) ([]storedBeat, domain.BeatSheetGate, domain.ReviewUnit, error) {
	sheetUnit, gate, err := uc.sheet(ctx, projectID)
	if err != nil {
		return nil, gate, sheetUnit, err
	}
	units, err := uc.units.ListUnits(ctx, projectID, domain.UnitKindBeat)
	if err != nil {
		return nil, gate, sheetUnit, err
	}
	byKey := make(map[string]domain.ReviewUnit, len(units))
	for _, u := range units {
		byKey[u.Key] = u
	}
	out := make([]storedBeat, 0, len(gate.Order))
	for _, key := range gate.Order {
		u, ok := byKey[key]
		if !ok {
			continue
		}
		b, err := readBeat(u)
		if err != nil {
			return nil, gate, sheetUnit, err
		}
		out = append(out, storedBeat{unit: u, Beat: b})
	}
	return out, gate, sheetUnit, nil
}

func (uc *BeatDeskUseCase) saveBeat(ctx context.Context, projectID, key string, b domain.Beat, base domain.ReviewUnit) (domain.ReviewUnit, error) {
	raw, err := json.Marshal(b)
	if err != nil {
		return domain.ReviewUnit{}, err
	}
	base.ProjectID, base.Kind, base.Key, base.Content = projectID, domain.UnitKindBeat, key, raw
	return uc.units.SaveUnit(ctx, base)
}

// storyApproved reports whether the story is approved.
func (uc *BeatDeskUseCase) storyApproved(ctx context.Context, projectID string) (bool, error) {
	_, _, ok, err := uc.stories.ApprovedStory(ctx, projectID)
	return ok, err
}

// State returns the whole beat sheet screen.
func (uc *BeatDeskUseCase) State(ctx context.Context, projectID string) (BeatState, error) {
	state := BeatState{Status: domain.UnitPending, Beats: []BeatView{}, Facts: []BeatFactView{}}
	var err error
	if state.StoryApproved, err = uc.storyApproved(ctx, projectID); err != nil {
		return BeatState{}, err
	}
	stored, gate, _, err := uc.beats(ctx, projectID)
	if err != nil {
		return BeatState{}, err
	}
	state.Rebalanced = gate.Rebalanced
	switch {
	case len(stored) == 0:
		state.Phase = BeatPhaseEmpty
	case !gate.SkeletonApproved:
		state.Phase = BeatPhaseSkeleton
	default:
		state.Phase = BeatPhaseContent
	}
	facts, err := loadKeptFacts(ctx, uc.units, projectID)
	if err != nil {
		return BeatState{}, err
	}
	var must []string
	for _, f := range facts {
		state.Facts = append(state.Facts, BeatFactView{ID: f.ID, ClaimVI: f.ClaimVI, Type: f.Type, Must: f.Must, Link: f.Link.State})
		if f.Must {
			must = append(must, f.ID)
		}
	}
	holders := map[string]int{}
	beats := make([]domain.Beat, 0, len(stored))
	approved := 0
	for _, b := range stored {
		beats = append(beats, b.Beat)
		for _, id := range b.Facts {
			holders[id]++
		}
		if b.unit.Status == domain.UnitApproved {
			approved++
		}
	}
	scriptUnits, err := uc.units.ListUnits(ctx, projectID, domain.UnitKindScriptBeat)
	if err != nil {
		return BeatState{}, err
	}
	for _, u := range scriptUnits {
		if u.Status == domain.UnitApproved && indexOfBeat(stored, u.Key) >= 0 {
			state.ScriptApproved++
		}
	}
	mustSet := toSet(must)
	for _, b := range stored {
		view := BeatView{Key: b.unit.Key, Beat: b.Beat, Status: b.unit.Status, Mine: append([]string{}, b.unit.CreatorFields...), Stale: b.unit.Stale, OnlyMust: []string{}, NoFact: b.Fn != domain.BeatFns[0] && len(b.Facts) == 0}
		for _, id := range b.Facts {
			if mustSet[id] && holders[id] == 1 {
				view.OnlyMust = append(view.OnlyMust, id)
			}
		}
		state.Beats = append(state.Beats, view)
		state.TotalSeconds += b.Dur
		if !b.Written() {
			state.Unwritten++
		}
	}
	if state.Phase == BeatPhaseContent && approved == len(stored) {
		state.Status = domain.UnitApproved
	}
	project, err := uc.projects.Get(ctx, projectID)
	if err != nil {
		return BeatState{}, fmt.Errorf("load project: %w", err)
	}
	duration, err := uc.units.GetTargetDuration(ctx, projectID)
	if err != nil {
		return BeatState{}, err
	}
	lo, hi := domain.TargetSeconds(project.VideoOutputMode, duration)
	state.TargetLabel = domain.TargetDurationLabel(project.VideoOutputMode, duration, "vi")
	if state.Phase == BeatPhaseContent {
		state.Checks = domain.CheckBeatSheet(beats, must, lo, hi, state.TargetLabel)
	} else {
		state.Checks = domain.CheckSkeleton(beats, must)
	}
	return state, nil
}

// FactDependents lists, for the research step, the beats that use a fact, so
// that dropping or changing the fact says what it sends back.
func (uc *BeatDeskUseCase) FactDependents() FactDependentsFinder {
	return func(ctx context.Context, projectID, factID string) ([]string, error) {
		stored, _, _, err := uc.beats(ctx, projectID)
		if err != nil {
			return nil, err
		}
		var items []string
		for i, b := range stored {
			for _, id := range b.Facts {
				if id == factID {
					items = append(items, fmt.Sprintf("Beat sheet · B%d %q dùng %s: beat được duyệt lại", i+1, b.Title, factID))
				}
			}
		}
		return items, nil
	}
}

// Delete removes a beat from the sheet. The facts it held stay in the fact
// table; the sheet's checks say if a fact the script must state is left out.
func (uc *BeatDeskUseCase) Delete(ctx context.Context, projectID, key string) error {
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return err
	}
	_, gate, sheetUnit, err := uc.beats(ctx, projectID)
	if err != nil {
		return err
	}
	at := indexOf(gate.Order, key)
	if at < 0 {
		return domain.ErrUnitNotFound
	}
	gate.Order = append(append([]string{}, gate.Order[:at]...), gate.Order[at+1:]...)
	if len(gate.Order) == 0 {
		gate.SkeletonApproved, gate.Rebalanced = false, 0
	}
	if err := uc.saveSheet(ctx, sheetUnit, gate); err != nil {
		return err
	}
	if err := uc.units.DeleteUnit(ctx, projectID, domain.UnitKindScriptBeat, key); err != nil {
		return err
	}
	return uc.units.DeleteUnit(ctx, projectID, domain.UnitKindBeat, key)
}

// Reorder sets the order of the beats. order must name every beat once.
func (uc *BeatDeskUseCase) Reorder(ctx context.Context, projectID string, order []string) error {
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return err
	}
	_, gate, sheetUnit, err := uc.beats(ctx, projectID)
	if err != nil {
		return err
	}
	if len(order) != len(gate.Order) || len(toSet(order)) != len(order) {
		return fmt.Errorf("%w: thứ tự phải nêu mỗi beat đúng một lần", domain.ErrInvalidUnitEdit)
	}
	have := toSet(gate.Order)
	for _, key := range order {
		if !have[key] {
			return fmt.Errorf("%w: không có beat %s", domain.ErrInvalidUnitEdit, key)
		}
	}
	gate.Order = append([]string{}, order...)
	return uc.saveSheet(ctx, sheetUnit, gate)
}

func indexOfBeat(beats []storedBeat, key string) int {
	for i, b := range beats {
		if b.unit.Key == key {
			return i
		}
	}
	return -1
}

func indexOf(list []string, key string) int {
	for i, k := range list {
		if k == key {
			return i
		}
	}
	return -1
}

// requireAI fails unless a language model is configured.
func (uc *BeatDeskUseCase) requireAI() error {
	if uc.provider == nil {
		return ErrLLMNotConfigured
	}
	if r, ok := uc.provider.(interface{ Ready() bool }); ok && !r.Ready() {
		return ErrLLMNotConfigured
	}
	return nil
}

// chat sends one system prompt and records the usage.
func (uc *BeatDeskUseCase) chat(ctx context.Context, projectID, model, system string) (string, error) {
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
		uc.recorder.Record(ctx, RecordFor(uc.provider.Name(), string(domain.RoleBeatSheet), domain.UnitKindBeat, projectID, result.Usage, started, err))
	}
	if err != nil {
		return "", err
	}
	return result.Content, nil
}
