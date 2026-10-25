package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"authoring/internal/domain"
)

// beatStoryRef names the approved story as the input the beats are built from.
// The review unit store resolves it from the project's story, so that a beat
// written from a story that has changed since reads as stale.
const beatStoryRef = domain.StoryInputKind + ":" + domain.StoryInputKey

// ErrBeatsNeedBrief is returned when the beat sheet is asked for before the brief is approved.
var ErrBeatsNeedBrief = errors.New("duyệt brief ở bước Creative brief trước")

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

// BeatInput is what the AI reads besides the kept facts, as the prompt gets it,
// so the Creator can see it before any call.
type BeatInput struct {
	Topic string `json:"topic"`
	// Brief is the approved brief; empty while the brief step is not approved.
	Brief         string `json:"brief"`
	BriefApproved bool   `json:"brief_approved"`
	// Story is the approved story in the fields the beats are built from.
	Story string `json:"story"`
}

// BeatState is everything the beat sheet screen shows.
type BeatState struct {
	// Status is approved when the sheet has beats and every one is approved.
	Status        domain.UnitStatus  `json:"status"`
	StoryApproved bool               `json:"story_approved"`
	Beats         []BeatView         `json:"beats"`
	Facts         []BeatFactView     `json:"facts"`
	Input         BeatInput          `json:"input"`
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
// the gate that holds their order are one row of project_beat_sheet.
type BeatDeskUseCase struct {
	units           StepUnitStorePort
	sheets          BeatSheetStorePort
	stories         ApprovedStoryPort
	briefs          ApprovedBriefPort
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
	units StepUnitStorePort, sheets BeatSheetStorePort, stories ApprovedStoryPort, briefs ApprovedBriefPort, projects ProjectReaderPort, locks AuthoringLockPort, prompts PromptActivePort,
	provider LLMProviderPort, recorder *LLMUsageRecorder, maxOutputTokens int,
) *BeatDeskUseCase {
	return &BeatDeskUseCase{
		units: units, sheets: sheets, stories: stories, briefs: briefs, projects: projects, locks: locks, prompts: prompts,
		provider: provider, recorder: recorder, maxOutputTokens: maxOutputTokens, now: time.Now,
	}
}

// storedBeat is a beat of the sheet with its content.
type storedBeat struct {
	unit domain.BeatSlot
	domain.Beat
	// stale is true for a beat the AI wrote from a story that has changed since.
	stale bool
}

// beats loads the beats in the order the sheet holds them, with the gate of
// the step. A project with no sheet yet has no beats and an open gate.
func (uc *BeatDeskUseCase) beats(ctx context.Context, projectID string) ([]storedBeat, domain.BeatSheetGate, error) {
	sheet, err := uc.sheets.GetBeatSheet(ctx, projectID)
	if errors.Is(err, domain.ErrBeatSheetNotFound) {
		return []storedBeat{}, domain.BeatSheetGate{NextSeq: 1}, nil
	}
	if err != nil {
		return nil, domain.BeatSheetGate{}, err
	}
	_, storyHash, storyOK, err := uc.stories.ApprovedStory(ctx, projectID)
	if err != nil {
		return nil, domain.BeatSheetGate{}, err
	}
	gate := sheet.BeatSheetGate
	if gate.NextSeq < 1 {
		gate.NextSeq = 1
	}
	out := make([]storedBeat, 0, len(sheet.Beats))
	for _, b := range sheet.Beats {
		if b.Beat.Facts == nil {
			b.Beat.Facts = []string{}
		}
		stale := storyOK && b.StoryHash != "" && b.StoryHash != storyHash
		out = append(out, storedBeat{unit: b.BeatSlot, Beat: b.Beat, stale: stale})
	}
	return out, gate, nil
}

// saveSheet stores the whole sheet: the beats in the order given and the gate.
func (uc *BeatDeskUseCase) saveSheet(ctx context.Context, projectID string, beats []storedBeat, gate domain.BeatSheetGate) error {
	sheet := domain.StoredBeatSheet{ProjectID: projectID, BeatSheetGate: gate, Beats: make([]domain.StoredBeat, 0, len(beats))}
	for _, b := range beats {
		sheet.Beats = append(sheet.Beats, domain.StoredBeat{BeatSlot: b.unit, Beat: b.Beat})
	}
	return uc.sheets.SaveBeatSheet(ctx, sheet)
}

// newBeatSlot is the slot of a beat just built: waiting for review, from the
// story with the given hash ("" for a beat the Creator added).
func newBeatSlot(key, storyHash string) domain.BeatSlot {
	return domain.BeatSlot{Key: key, Status: domain.UnitPending, CreatorFields: []string{}, StoryHash: storyHash}
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
	if state.StoryApproved {
		if state.Input, err = uc.input(ctx, projectID); err != nil {
			return BeatState{}, err
		}
	}
	stored, gate, err := uc.beats(ctx, projectID)
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
		view := BeatView{Key: b.unit.Key, Beat: b.Beat, Status: b.unit.Status, Mine: append([]string{}, b.unit.CreatorFields...), Stale: b.stale, OnlyMust: []string{}, NoFact: b.Fn != domain.BeatFns[0] && len(b.Facts) == 0}
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
		stored, _, err := uc.beats(ctx, projectID)
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
	beats, gate, err := uc.beats(ctx, projectID)
	if err != nil {
		return err
	}
	at := indexOfBeat(beats, key)
	if at < 0 {
		return domain.ErrUnitNotFound
	}
	beats = append(append([]storedBeat{}, beats[:at]...), beats[at+1:]...)
	if len(beats) == 0 {
		gate.SkeletonApproved, gate.Rebalanced = false, 0
	}
	if err := uc.saveSheet(ctx, projectID, beats, gate); err != nil {
		return err
	}
	return uc.units.DeleteUnit(ctx, projectID, domain.UnitKindScriptBeat, key)
}

// Reorder sets the order of the beats. order must name every beat once.
func (uc *BeatDeskUseCase) Reorder(ctx context.Context, projectID string, order []string) error {
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return err
	}
	beats, gate, err := uc.beats(ctx, projectID)
	if err != nil {
		return err
	}
	if len(order) != len(beats) || len(toSet(order)) != len(order) {
		return fmt.Errorf("%w: thứ tự phải nêu mỗi beat đúng một lần", domain.ErrInvalidUnitEdit)
	}
	moved := make([]storedBeat, 0, len(beats))
	for _, key := range order {
		at := indexOfBeat(beats, key)
		if at < 0 {
			return fmt.Errorf("%w: không có beat %s", domain.ErrInvalidUnitEdit, key)
		}
		moved = append(moved, beats[at])
	}
	return uc.saveSheet(ctx, projectID, moved, gate)
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
