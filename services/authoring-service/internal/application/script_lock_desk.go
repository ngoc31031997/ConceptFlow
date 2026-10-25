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

// Errors of the lock step.
var (
	// ErrLockNeedsScript is returned when the step is used before every beat's dialogue is approved.
	ErrLockNeedsScript = errors.New("duyệt lời thoại ở bước Viết kịch bản trước")
	// ErrHandoffExists is returned when a beat that already has a note is written from scratch.
	ErrHandoffExists = errors.New("beat đã có ghi chú bàn giao: dùng Làm lại để viết lại")
	// ErrFindingNotFound is returned for a finding id the review does not hold.
	ErrFindingNotFound = errors.New("không có mục rà soát này")
	// ErrFindingNotApplicable is returned when an action does not fit a finding.
	ErrFindingNotApplicable = errors.New("mục này không có thao tác đó")
)

// ScriptVersionPort stores the locked versions of a script. LatestScriptVersion
// returns domain.ErrNoLockedVersion when none is locked.
type ScriptVersionPort interface {
	SaveScriptVersion(ctx context.Context, v domain.ScriptVersion) error
	ListScriptVersions(ctx context.Context, projectID string) ([]domain.ScriptVersion, error)
	LatestScriptVersion(ctx context.Context, projectID string) (domain.ScriptVersion, error)
}

// ScriptLockDeskUseCase backs the lock step: the review of the whole script,
// the handoff note of each beat, the shared glossary, and the locking of a
// version. The handoff note of a beat is a review unit (domain.UnitKindHandoffBeat)
// keyed by the beat's key; the review and the glossary are one unit each. The
// unit store keeps them in project_handoff, project_script_review and
// project_glossary.
type ScriptLockDeskUseCase struct {
	units           StepUnitStorePort
	projects        ProjectReaderPort
	locks           AuthoringLockPort
	prompts         PromptActivePort
	provider        LLMProviderPort
	maxOutputTokens int
	beats           *BeatDeskUseCase
	script          *ScriptDeskUseCase
	versions        ScriptVersionPort
	now             func() time.Time
}

// NewScriptLockDeskUseCase wires the use case. provider may be nil when no
// language model is configured: the Creator then writes the notes by hand.
func NewScriptLockDeskUseCase(
	units StepUnitStorePort, projects ProjectReaderPort, locks AuthoringLockPort, prompts PromptActivePort,
	provider LLMProviderPort, maxOutputTokens int,
	beats *BeatDeskUseCase, script *ScriptDeskUseCase, versions ScriptVersionPort,
) *ScriptLockDeskUseCase {
	return &ScriptLockDeskUseCase{
		units: units, projects: projects, locks: locks, prompts: prompts, provider: provider,
		maxOutputTokens: maxOutputTokens, beats: beats, script: script, versions: versions, now: time.Now,
	}
}

// storedHandoff is a handoff_beat unit with its content read.
type storedHandoff struct {
	unit domain.ReviewUnit
	domain.HandoffNote
}

func readHandoff(u domain.ReviewUnit) (domain.HandoffNote, error) {
	var n domain.HandoffNote
	if err := json.Unmarshal(u.Content, &n); err != nil {
		return domain.HandoffNote{}, fmt.Errorf("read stored handoff %s: %w", u.Key, err)
	}
	return n.Normalized(), nil
}

func (uc *ScriptLockDeskUseCase) handoffs(ctx context.Context, projectID string) (map[string]storedHandoff, error) {
	units, err := uc.units.ListUnits(ctx, projectID, domain.UnitKindHandoffBeat)
	if err != nil {
		return nil, err
	}
	out := make(map[string]storedHandoff, len(units))
	for _, u := range units {
		n, err := readHandoff(u)
		if err != nil {
			return nil, err
		}
		out[u.Key] = storedHandoff{unit: u, HandoffNote: n}
	}
	return out, nil
}

// saveHandoff stores a beat's note, built from the beat's dialogue as it is now.
func (uc *ScriptLockDeskUseCase) saveHandoff(ctx context.Context, projectID, key string, n domain.HandoffNote, base domain.ReviewUnit) (domain.ReviewUnit, error) {
	raw, err := json.Marshal(n.Normalized())
	if err != nil {
		return domain.ReviewUnit{}, err
	}
	sha, err := uc.units.UnitSHA(ctx, projectID, domain.UnitKindScriptBeat, key)
	if err != nil {
		return domain.ReviewUnit{}, err
	}
	base.ProjectID, base.Kind, base.Key, base.Content = projectID, domain.UnitKindHandoffBeat, key, raw
	base.InputRef, base.InputSHA = domain.UnitKindScriptBeat+":"+key, sha
	if base.CreatorFields == nil {
		base.CreatorFields = []string{}
	}
	return uc.units.SaveUnit(ctx, base)
}

func (uc *ScriptLockDeskUseCase) review(ctx context.Context, projectID string) (domain.ReviewUnit, domain.ScriptReview, error) {
	u, err := uc.units.GetUnit(ctx, projectID, domain.UnitKindScriptReview, domain.ScriptReviewUnitKey)
	if errors.Is(err, domain.ErrUnitNotFound) {
		return domain.ReviewUnit{ProjectID: projectID, Kind: domain.UnitKindScriptReview, Key: domain.ScriptReviewUnitKey, Status: domain.UnitPending, CreatorFields: []string{}},
			domain.ScriptReview{}.Normalized(), nil
	}
	if err != nil {
		return domain.ReviewUnit{}, domain.ScriptReview{}, err
	}
	var r domain.ScriptReview
	if err := json.Unmarshal(u.Content, &r); err != nil {
		return domain.ReviewUnit{}, domain.ScriptReview{}, fmt.Errorf("read stored script review: %w", err)
	}
	return u, r.Normalized(), nil
}

func (uc *ScriptLockDeskUseCase) saveReview(ctx context.Context, projectID string, base domain.ReviewUnit, r domain.ScriptReview) error {
	raw, err := json.Marshal(r)
	if err != nil {
		return err
	}
	base.ProjectID, base.Kind, base.Key, base.Content = projectID, domain.UnitKindScriptReview, domain.ScriptReviewUnitKey, raw
	_, err = uc.units.SaveUnit(ctx, base)
	return err
}

func (uc *ScriptLockDeskUseCase) glossary(ctx context.Context, projectID string) (domain.Glossary, error) {
	u, err := uc.units.GetUnit(ctx, projectID, domain.UnitKindGlossary, domain.GlossaryUnitKey)
	if errors.Is(err, domain.ErrUnitNotFound) {
		return domain.Glossary{Terms: []domain.GlossaryTerm{}}, nil
	}
	if err != nil {
		return domain.Glossary{}, err
	}
	var g domain.Glossary
	if err := json.Unmarshal(u.Content, &g); err != nil {
		return domain.Glossary{}, fmt.Errorf("read stored glossary: %w", err)
	}
	if g.Terms == nil {
		g.Terms = []domain.GlossaryTerm{}
	}
	return g, nil
}

func (uc *ScriptLockDeskUseCase) saveGlossary(ctx context.Context, projectID string, g domain.Glossary) error {
	raw, err := json.Marshal(g)
	if err != nil {
		return err
	}
	_, err = uc.units.SaveUnit(ctx, domain.ReviewUnit{
		ProjectID: projectID, Kind: domain.UnitKindGlossary, Key: domain.GlossaryUnitKey,
		Status: domain.UnitPending, Content: raw, CreatorFields: []string{},
	})
	return err
}

// researchSkipped reports whether the Creator skipped the research step.
func (uc *ScriptLockDeskUseCase) researchSkipped(ctx context.Context, projectID string) (bool, error) {
	u, err := uc.units.GetUnit(ctx, projectID, domain.UnitKindResearch, domain.ResearchUnitKey)
	if errors.Is(err, domain.ErrUnitNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	var gate domain.ResearchGate
	if err := json.Unmarshal(u.Content, &gate); err != nil {
		return false, fmt.Errorf("read research gate: %w", err)
	}
	return gate.Skipped, nil
}

// FindingView is one finding with where it stands.
type FindingView struct {
	domain.ReviewFinding
	GroupLabel string `json:"group_label"`
	State      string `json:"state"`
	Why        string `json:"why,omitempty"`
}

// LockBeatView is a beat with its handoff note.
type LockBeatView struct {
	Key    string `json:"key"`
	Number int    `json:"number"`
	Fn     string `json:"fn"`
	Title  string `json:"title"`
	Goal   string `json:"goal"`
	// Written is true once the beat has a note.
	Written bool               `json:"written"`
	Status  domain.UnitStatus  `json:"status"`
	Mine    []string           `json:"mine"`
	Stale   bool               `json:"stale"`
	Note    domain.HandoffNote `json:"note"`
	Lines   int                `json:"lines"`
}

// LockCondition is one condition for locking a version.
type LockCondition struct {
	OK   bool   `json:"ok"`
	Text string `json:"text"`
}

// ChainStep is one beat in the order and joins the script runs.
type ChainStep struct {
	Fn    string `json:"fn"`
	Title string `json:"title"`
	Next  string `json:"next"`
}

// ScriptPackage is the "A · Kịch bản" set of the package.
type ScriptPackage struct {
	Topic        string      `json:"topic"`
	Audience     string      `json:"audience"`
	TargetLabel  string      `json:"target_label"`
	TotalSeconds float64     `json:"total_seconds"`
	Lines        int         `json:"lines"`
	Beats        int         `json:"beats"`
	Chain        []ChainStep `json:"chain"`
	// Facts are the facts the lines use, with how they stand.
	Facts []PackageFact `json:"facts"`
	// Hypotheses counts used facts that are only hypotheses; Unmatched those whose link does not show the quote.
	Hypotheses int `json:"hypotheses"`
	Unmatched  int `json:"unmatched"`
	// NoSource is true when the research step was skipped.
	NoSource bool `json:"no_source"`
}

// PackageFact is a used fact in the package.
type PackageFact struct {
	ID   string           `json:"id"`
	Type domain.FactType  `json:"type"`
	Link domain.LinkState `json:"link"`
}

// HandoffCounts are the "B · Ghi chú bàn giao" counts.
type HandoffCounts struct {
	Beats    int            `json:"beats"`
	Sections map[string]int `json:"sections"`
	Glossary int            `json:"glossary"`
}

// VersionView is one locked version in the history.
type VersionView struct {
	Version  int       `json:"version"`
	LockedAt time.Time `json:"locked_at"`
	Note     string    `json:"note"`
}

// ScriptLockState is everything the lock screen shows.
type ScriptLockState struct {
	// ScriptReady is true when every beat has approved dialogue; the step is closed before that.
	ScriptReady bool `json:"script_ready"`
	// Reviewed is true once the AI has reviewed the script at least once.
	Reviewed   bool                    `json:"reviewed"`
	Findings   []FindingView           `json:"findings"`
	OpenCount  int                     `json:"open_count"`
	AINotes    []string                `json:"ai_notes"`
	MyNotes    []string                `json:"my_notes"`
	Glossary   []domain.GlossaryTerm   `json:"glossary"`
	Beats      []LockBeatView          `json:"beats"`
	Conditions []LockCondition         `json:"conditions"`
	CanLock    bool                    `json:"can_lock"`
	Package    ScriptPackage           `json:"package"`
	Handoff    HandoffCounts           `json:"handoff"`
	Sections   []domain.HandoffSection `json:"sections"`
	// Versions is the history, oldest first; LockedVersion is the latest, 0 when none.
	Versions      []VersionView `json:"versions"`
	LockedVersion int           `json:"locked_version"`
	// Draft lists what changed since the latest locked version, in the Creator's words.
	Draft []string `json:"draft"`
	// NextVersion is the number the next lock takes.
	NextVersion int `json:"next_version"`
	// SkippedResearch is true while the research step is skipped.
	SkippedResearch bool `json:"skipped_research"`
}

// readiness reads what the lock step stands on.
type lockInputs struct {
	lc       lockContext
	script   ScriptState
	notes    map[string]storedHandoff
	glossary domain.Glossary
	reviewU  domain.ReviewUnit
	review   domain.ScriptReview
	skipped  bool
	ready    bool
}

func (uc *ScriptLockDeskUseCase) inputs(ctx context.Context, projectID string) (lockInputs, error) {
	var in lockInputs
	var err error
	if in.lc.beats, _, err = uc.beats.beats(ctx, projectID); err != nil {
		return in, err
	}
	if in.lc.scripts, err = uc.script.scripts(ctx, projectID); err != nil {
		return in, err
	}
	in.ready = len(in.lc.beats) > 0
	for _, b := range in.lc.beats {
		s, ok := in.lc.scripts[b.unit.Key]
		if !ok || len(s.Lines) == 0 || s.unit.Status != domain.UnitApproved {
			in.ready = false
		}
	}
	if in.notes, err = uc.handoffs(ctx, projectID); err != nil {
		return in, err
	}
	if in.glossary, err = uc.glossary(ctx, projectID); err != nil {
		return in, err
	}
	if in.reviewU, in.review, err = uc.review(ctx, projectID); err != nil {
		return in, err
	}
	if in.skipped, err = uc.researchSkipped(ctx, projectID); err != nil {
		return in, err
	}
	return in, nil
}

func (uc *ScriptLockDeskUseCase) machine(ctx context.Context, projectID string, in lockInputs) ([]domain.ReviewFinding, error) {
	st, err := uc.script.State(ctx, projectID)
	if err != nil {
		return nil, err
	}
	project, err := uc.projects.Get(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("load project: %w", err)
	}
	duration, err := uc.units.GetTargetDuration(ctx, projectID)
	if err != nil {
		return nil, err
	}
	lo, hi := domain.TargetSeconds(project.VideoOutputMode, duration)
	facts, err := loadKeptFacts(ctx, uc.units, projectID)
	if err != nil {
		return nil, err
	}
	ri := domain.ReviewInput{
		ResearchSkipped: in.skipped, TotalSeconds: st.Stats.TotalSeconds, LoSeconds: lo, HiSeconds: hi,
		TargetLabel: st.TargetLabel, ScriptSHA: domain.ScriptSHA(in.lc.lineSHAParts()), ReviewedSHA: in.review.ReviewedSHA,
	}
	for i, b := range in.lc.beats {
		ri.Beats = append(ri.Beats, domain.ReviewBeat{Key: b.unit.Key, Number: i + 1, Lines: in.lc.scripts[b.unit.Key].Lines})
	}
	for _, f := range facts {
		ri.Facts = append(ri.Facts, domain.ReviewFact{ID: f.ID, Must: f.Must, Link: f.Link.State})
	}
	return domain.MachineFindings(ri), nil
}

func (uc *ScriptLockDeskUseCase) findingViews(machine []domain.ReviewFinding, r domain.ScriptReview) ([]FindingView, int) {
	out := []FindingView{}
	open := 0
	add := func(f domain.ReviewFinding) {
		v := FindingView{ReviewFinding: f, GroupLabel: domain.ReviewGroupLabel(f.Group), State: r.StateOf(f)}
		if d, ok := r.Decisions[f.ID]; ok && d.Fingerprint == f.Fingerprint() {
			v.Why = d.Why
		}
		if v.State == domain.FindingOpen {
			open++
		}
		out = append(out, v)
	}
	for _, f := range machine {
		add(f)
	}
	for _, f := range r.Findings {
		add(f)
	}
	return out, open
}

// currentParts freezes what a version would hold if locked now.
func (uc *ScriptLockDeskUseCase) currentParts(ctx context.Context, projectID string, in lockInputs) ([]domain.VersionPart, error) {
	var parts []domain.VersionPart
	add := func(key string, v any) error {
		p, err := domain.NewVersionPart(key, v)
		if err != nil {
			return err
		}
		parts = append(parts, p)
		return nil
	}
	_, brief, _, err := loadApprovedBrief(ctx, uc.units, projectID)
	if err != nil {
		return nil, err
	}
	if err := add(domain.PartBrief, brief); err != nil {
		return nil, err
	}
	facts, err := loadKeptFacts(ctx, uc.units, projectID)
	if err != nil {
		return nil, err
	}
	if err := add(domain.PartFacts, facts); err != nil {
		return nil, err
	}
	fields, err := uc.beats.storyFields(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if err := add(domain.PartStory, fields); err != nil {
		return nil, err
	}
	order := make([]string, 0, len(in.lc.beats))
	for _, b := range in.lc.beats {
		order = append(order, b.unit.Key)
	}
	if err := add(domain.PartOrder, order); err != nil {
		return nil, err
	}
	for _, b := range in.lc.beats {
		k := b.unit.Key
		if err := add(domain.PartBeat+k, b.Beat); err != nil {
			return nil, err
		}
		if err := add(domain.PartScript+k, in.lc.scripts[k].ScriptBeat); err != nil {
			return nil, err
		}
		if err := add(domain.PartHandoff+k, in.notes[k].HandoffNote.Normalized()); err != nil {
			return nil, err
		}
	}
	if err := add(domain.PartGlossary, in.glossary); err != nil {
		return nil, err
	}
	return parts, uc.addReviewPart(&parts, in)
}

// acceptedReason is a finding the Creator accepted, frozen with the reason.
type acceptedReason struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Why   string `json:"why"`
}

func (uc *ScriptLockDeskUseCase) addReviewPart(parts *[]domain.VersionPart, in lockInputs) error {
	accepted := []acceptedReason{}
	for _, f := range in.review.Findings {
		if in.review.StateOf(f) == domain.FindingAccepted {
			accepted = append(accepted, acceptedReason{ID: f.ID, Title: f.Title, Why: in.review.Decisions[f.ID].Why})
		}
	}
	for id, d := range in.review.Decisions {
		if d.State == domain.FindingAccepted && !strings.HasPrefix(id, "r") {
			accepted = append(accepted, acceptedReason{ID: id, Why: d.Why})
		}
	}
	p, err := domain.NewVersionPart(domain.PartReview, map[string]any{
		"accepted": accepted, "ai_notes": in.review.AINotes, "my_notes": in.review.MyNotes,
	})
	if err != nil {
		return err
	}
	*parts = append(*parts, p)
	return nil
}

func (in lockInputs) beatNumber(key string) int {
	for i, b := range in.lc.beats {
		if b.unit.Key == key {
			return i + 1
		}
	}
	return 0
}

// State returns the whole lock screen.
func (uc *ScriptLockDeskUseCase) State(ctx context.Context, projectID string) (ScriptLockState, error) {
	state := ScriptLockState{
		Findings: []FindingView{}, AINotes: []string{}, MyNotes: []string{}, Glossary: []domain.GlossaryTerm{},
		Beats: []LockBeatView{}, Conditions: []LockCondition{}, Versions: []VersionView{}, Draft: []string{},
		Sections: domain.HandoffSections, NextVersion: 1,
	}
	in, err := uc.inputs(ctx, projectID)
	if err != nil {
		return state, err
	}
	state.ScriptReady, state.SkippedResearch, state.Reviewed = in.ready, in.skipped, in.review.ReviewedSHA != ""
	state.AINotes, state.MyNotes, state.Glossary = in.review.AINotes, in.review.MyNotes, in.glossary.Terms

	all, err := uc.versions.ListScriptVersions(ctx, projectID)
	if err != nil {
		return state, err
	}
	for _, v := range all {
		state.Versions = append(state.Versions, VersionView{Version: v.Version, LockedAt: v.LockedAt, Note: v.Note})
	}
	var latest *domain.ScriptVersion
	if n := len(all); n > 0 {
		latest = &all[n-1]
		state.LockedVersion, state.NextVersion = latest.Version, latest.Version+1
	}

	notesDone, staleOrOpen := 0, 0
	for i, b := range in.lc.beats {
		view := LockBeatView{
			Key: b.unit.Key, Number: i + 1, Fn: b.Fn, Title: b.Title, Goal: b.Goal,
			Status: domain.UnitPending, Mine: []string{}, Note: domain.HandoffNote{}.Normalized(),
			Lines: len(in.lc.scripts[b.unit.Key].Lines),
		}
		if n, ok := in.notes[b.unit.Key]; ok {
			view.Written, view.Status, view.Stale, view.Note = true, n.unit.Status, n.unit.Stale, n.HandoffNote
			view.Mine = append(view.Mine, n.unit.CreatorFields...)
			for _, s := range domain.HandoffSections {
				state.Handoff.countSection(s.Key, len(n.Section(s.Key)))
			}
		}
		if view.Written && view.Status == domain.UnitApproved && !view.Stale {
			notesDone++
		} else {
			staleOrOpen++
		}
		state.Beats = append(state.Beats, view)
	}
	state.Handoff.Beats, state.Handoff.Glossary = len(in.lc.beats), len(in.glossary.Terms)

	if in.ready {
		machine, err := uc.machine(ctx, projectID, in)
		if err != nil {
			return state, err
		}
		state.Findings, state.OpenCount = uc.findingViews(machine, in.review)
	}

	scriptPending := 0
	for _, b := range in.lc.beats {
		if s, ok := in.lc.scripts[b.unit.Key]; !ok || s.unit.Status != domain.UnitApproved {
			scriptPending++
		}
	}
	state.Conditions = []LockCondition{
		{OK: state.OpenCount == 0 && in.ready, Text: "Rà soát: không còn mục cần xử lý" + countSuffix(state.OpenCount, "")},
		{OK: scriptPending == 0 && in.ready, Text: "Lời thoại: mọi beat đã duyệt" + countSuffix(scriptPending, "beat")},
		{OK: staleOrOpen == 0 && in.ready, Text: "Ghi chú bàn giao: mọi beat đã duyệt" + countSuffix(staleOrOpen, "beat")},
	}
	state.CanLock = in.ready
	for _, c := range state.Conditions {
		state.CanLock = state.CanLock && c.OK
	}

	if err := uc.fillPackage(ctx, projectID, in, &state); err != nil {
		return state, err
	}
	if latest != nil && in.ready {
		parts, err := uc.currentParts(ctx, projectID, in)
		if err != nil {
			return state, err
		}
		for _, key := range domain.ChangedParts(*latest, parts) {
			state.Draft = append(state.Draft, domain.PartLabel(key, in.beatNumber))
		}
	}
	return state, nil
}

func (h *HandoffCounts) countSection(key string, n int) {
	if h.Sections == nil {
		h.Sections = map[string]int{}
	}
	h.Sections[key] += n
}

func countSuffix(n int, unit string) string {
	if n == 0 {
		return ""
	}
	if unit == "" {
		return fmt.Sprintf(" (còn %d)", n)
	}
	return fmt.Sprintf(" (còn %d %s)", n, unit)
}

func (uc *ScriptLockDeskUseCase) fillPackage(ctx context.Context, projectID string, in lockInputs, state *ScriptLockState) error {
	if !in.ready {
		return nil
	}
	st, err := uc.script.State(ctx, projectID)
	if err != nil {
		return err
	}
	topic, err := uc.units.GetAuthoringTopic(ctx, projectID)
	if err != nil {
		return fmt.Errorf("load topic: %w", err)
	}
	_, brief, _, err := loadApprovedBrief(ctx, uc.units, projectID)
	if err != nil {
		return err
	}
	facts, err := loadKeptFacts(ctx, uc.units, projectID)
	if err != nil {
		return err
	}
	byID := map[string]storyFact{}
	for _, f := range facts {
		byID[f.ID] = f
	}
	pkg := ScriptPackage{
		Topic: topic, Audience: brief.Audience, TargetLabel: st.TargetLabel, TotalSeconds: st.Stats.TotalSeconds,
		Lines: st.Stats.Lines, Beats: st.Stats.Beats, Chain: []ChainStep{}, Facts: []PackageFact{}, NoSource: in.skipped,
	}
	for _, b := range in.lc.beats {
		pkg.Chain = append(pkg.Chain, ChainStep{Fn: b.Fn, Title: b.Title, Next: b.Next})
	}
	seen := map[string]bool{}
	for _, b := range in.lc.beats {
		for _, l := range in.lc.scripts[b.unit.Key].Lines {
			for _, id := range l.Facts {
				f, ok := byID[id]
				if !ok || seen[id] {
					continue
				}
				seen[id] = true
				pkg.Facts = append(pkg.Facts, PackageFact{ID: id, Type: f.Type, Link: f.Link.State})
				if f.Type == domain.FactTypeHypothesis {
					pkg.Hypotheses++
				}
				if f.Link.State != domain.LinkOK && f.Type != domain.FactTypeUnknown {
					pkg.Unmatched++
				}
			}
		}
	}
	state.Package = pkg
	return nil
}

// --- actions on the review -----------------------------------------------------

func (uc *ScriptLockDeskUseCase) finding(ctx context.Context, projectID, id string) (lockInputs, domain.ReviewFinding, error) {
	in, err := uc.inputs(ctx, projectID)
	if err != nil {
		return in, domain.ReviewFinding{}, err
	}
	if !in.ready {
		return in, domain.ReviewFinding{}, fmt.Errorf("%w: %w", domain.ErrInvalidStatus, ErrLockNeedsScript)
	}
	machine, err := uc.machine(ctx, projectID, in)
	if err != nil {
		return in, domain.ReviewFinding{}, err
	}
	for _, f := range append(machine, in.review.Findings...) {
		if f.ID == id {
			return in, f, nil
		}
	}
	return in, domain.ReviewFinding{}, ErrFindingNotFound
}

// ApplyFix replaces the line of an AI finding with the fix it carries. The
// replacement is an ordinary hand edit of that beat's dialogue, so the beat
// goes back to review and its handoff note needs a second look.
func (uc *ScriptLockDeskUseCase) ApplyFix(ctx context.Context, projectID, id string) error {
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return err
	}
	in, f, err := uc.finding(ctx, projectID, id)
	if err != nil {
		return err
	}
	if f.Fix == "" || f.Line == "" || in.review.StateOf(f) != domain.FindingOpen {
		return ErrFindingNotApplicable
	}
	beatKey, ok := domain.ScriptLineBeat(f.Line)
	if !ok {
		return ErrFindingNotApplicable
	}
	if _, err := uc.script.ScriptUnitKind().Edit(ctx, projectID, beatKey, map[string]string{scriptFieldLine + f.Line: f.Fix, scriptFieldSpoken + f.Line: ""}); err != nil {
		return err
	}
	in.review.Decide(f, domain.FindingFixed, "")
	return uc.saveReview(ctx, projectID, in.reviewU, in.review)
}

// AddToHandoff adds the line an AI finding proposes to its beat's handoff
// note. The note goes back to review.
func (uc *ScriptLockDeskUseCase) AddToHandoff(ctx context.Context, projectID, id string) error {
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return err
	}
	in, f, err := uc.finding(ctx, projectID, id)
	if err != nil {
		return err
	}
	if f.Handoff == nil || in.review.StateOf(f) != domain.FindingOpen {
		return ErrFindingNotApplicable
	}
	h := f.Handoff
	cur, has := in.notes[h.Beat]
	note := domain.HandoffNote{}.Normalized()
	unit := domain.ReviewUnit{Status: domain.UnitPending, CreatorFields: []string{}}
	if has {
		note, unit = cur.HandoffNote, cur.unit
	}
	items := append(append([]string{}, note.Section(h.Section)...), h.Text)
	note = note.With(h.Section, items)
	if err := note.Validate(); err != nil {
		return fmt.Errorf("%w: %v", domain.ErrInvalidUnitEdit, err)
	}
	unit.Status = domain.UnitPending
	if _, err := uc.saveHandoff(ctx, projectID, h.Beat, note, unit); err != nil {
		return err
	}
	in.review.Decide(f, domain.FindingFixed, "")
	return uc.saveReview(ctx, projectID, in.reviewU, in.review)
}

// Accept records that the Creator accepts a finding as it stands, with the reason.
func (uc *ScriptLockDeskUseCase) Accept(ctx context.Context, projectID, id, why string) error {
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return err
	}
	in, f, err := uc.finding(ctx, projectID, id)
	if err != nil {
		return err
	}
	if f.Pass || f.Info {
		return ErrFindingNotApplicable
	}
	why = strings.TrimSpace(why)
	if why == "" {
		why = "không ghi lý do"
	}
	if len([]rune(why)) > domain.MaxAcceptWhyRunes {
		return fmt.Errorf("%w: lý do dài quá %d ký tự", domain.ErrInvalidUnitEdit, domain.MaxAcceptWhyRunes)
	}
	in.review.Decide(f, domain.FindingAccepted, why)
	return uc.saveReview(ctx, projectID, in.reviewU, in.review)
}

// AddNote adds a note of the Creator's to the review.
func (uc *ScriptLockDeskUseCase) AddNote(ctx context.Context, projectID, text string) error {
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return err
	}
	text = strings.TrimSpace(text)
	if text == "" || len([]rune(text)) > domain.MaxCreatorNoteRunes {
		return fmt.Errorf("%w: ghi chú phải có chữ và không quá %d ký tự", domain.ErrInvalidUnitEdit, domain.MaxCreatorNoteRunes)
	}
	u, r, err := uc.review(ctx, projectID)
	if err != nil {
		return err
	}
	r.MyNotes = append(r.MyNotes, text)
	return uc.saveReview(ctx, projectID, u, r)
}

// AddTerm adds a row to the shared glossary; the row is the Creator's.
func (uc *ScriptLockDeskUseCase) AddTerm(ctx context.Context, projectID string, t domain.GlossaryTerm) error {
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return err
	}
	t.Term, t.Say, t.Draw, t.Avoid, t.Mine = strings.TrimSpace(t.Term), strings.TrimSpace(t.Say), strings.TrimSpace(t.Draw), strings.TrimSpace(t.Avoid), true
	if err := t.Validate(); err != nil {
		return fmt.Errorf("%w: %v", domain.ErrInvalidUnitEdit, err)
	}
	g, err := uc.glossary(ctx, projectID)
	if err != nil {
		return err
	}
	before := len(g.Terms)
	g = g.Merge([]domain.GlossaryTerm{t})
	if len(g.Terms) == before {
		return fmt.Errorf("%w: thuật ngữ đã có hoặc bảng đã đầy", domain.ErrInvalidUnitEdit)
	}
	return uc.saveGlossary(ctx, projectID, g)
}

// ConfirmHandoff keeps a handoff note as it is after the dialogue it was built
// from changed, and approves it.
func (uc *ScriptLockDeskUseCase) ConfirmHandoff(ctx context.Context, projectID, key string) error {
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return err
	}
	notes, err := uc.handoffs(ctx, projectID)
	if err != nil {
		return err
	}
	n, ok := notes[key]
	if !ok {
		return domain.ErrUnitNotFound
	}
	if n.Empty() {
		return domain.ErrHandoffEmpty
	}
	unit := n.unit
	unit.Status = domain.UnitApproved
	_, err = uc.saveHandoff(ctx, projectID, key, n.HandoffNote, unit)
	return err
}

// --- locking -------------------------------------------------------------------

// Lock freezes the script, its facts and its handoff as the next version. It
// refuses unless every condition holds, and refuses a draft that changes
// nothing against the latest version.
func (uc *ScriptLockDeskUseCase) Lock(ctx context.Context, projectID, note string) (VersionView, error) {
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return VersionView{}, err
	}
	state, err := uc.State(ctx, projectID)
	if err != nil {
		return VersionView{}, err
	}
	if !state.CanLock {
		return VersionView{}, fmt.Errorf("%w: %w", domain.ErrInvalidStatus, domain.ErrLockNotReady)
	}
	if state.LockedVersion > 0 && len(state.Draft) == 0 {
		return VersionView{}, fmt.Errorf("%w: %w", domain.ErrInvalidStatus, domain.ErrNothingToLock)
	}
	note = strings.TrimSpace(note)
	if note == "" {
		if state.LockedVersion == 0 {
			note = "Chốt lần đầu"
		} else {
			note = fmt.Sprintf("Chốt %d thay đổi", len(state.Draft))
		}
	}
	if len([]rune(note)) > domain.MaxVersionNoteRunes {
		return VersionView{}, fmt.Errorf("%w: ghi chú dài quá %d ký tự", domain.ErrInvalidUnitEdit, domain.MaxVersionNoteRunes)
	}
	in, err := uc.inputs(ctx, projectID)
	if err != nil {
		return VersionView{}, err
	}
	parts, err := uc.currentParts(ctx, projectID, in)
	if err != nil {
		return VersionView{}, err
	}
	v := domain.ScriptVersion{ProjectID: projectID, Version: state.NextVersion, Note: note, LockedAt: uc.now().UTC(), Parts: parts}
	if err := uc.versions.SaveScriptVersion(ctx, v); err != nil {
		return VersionView{}, err
	}
	return VersionView{Version: v.Version, LockedAt: v.LockedAt, Note: v.Note}, nil
}

// HandoffForRole returns what a role reads from the latest locked version.
func (uc *ScriptLockDeskUseCase) HandoffForRole(ctx context.Context, projectID, role string) (domain.HandoffPackage, error) {
	v, err := uc.versions.LatestScriptVersion(ctx, projectID)
	if err != nil {
		return domain.HandoffPackage{}, err
	}
	return domain.HandoffFor(role, v)
}
