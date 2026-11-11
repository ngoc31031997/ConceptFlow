package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"visual/internal/domain"
)

// PlanStep is the step name of the visual plan in llm-service (default model,
// call log).
const PlanStep = "visual_plan"

// PlanDesk is step 9 "Kế hoạch hình": it reads the locked script and the locked
// visual concept, keeps the visual-plan units of every beat and their review,
// has the AI write and review them, and locks versions.
type PlanDesk struct {
	authoring AuthoringInputPort
	store     PlanStorePort
	now       func() time.Time
	running   *runningSet

	// The AI side; unset until WithAI, and then the AI routes fail with
	// ErrLLMNotConfigured.
	llm       LLMPort
	prompts   PromptLibraryPort
	models    ModelCatalogPort
	maxTokens int
}

// NewPlanDesk wires the desk without an AI.
func NewPlanDesk(authoring AuthoringInputPort, store PlanStorePort) *PlanDesk {
	return &PlanDesk{authoring: authoring, store: store, now: time.Now, running: &runningSet{m: map[string]map[string]bool{}}}
}

// WithAI gives the desk the model, the prompt library and the model catalog it
// writes and reviews with.
func (d *PlanDesk) WithAI(llm LLMPort, prompts PromptLibraryPort, models ModelCatalogPort, maxTokens int) *PlanDesk {
	d.llm, d.prompts, d.models, d.maxTokens = llm, prompts, models, maxTokens
	return d
}

// planData is everything one request works on.
type planData struct {
	in      domain.PlanInput
	rows    map[string]domain.ReviewUnit
	plans   map[string]domain.BeatPlan
	review  domain.ReviewState
	revUnit domain.ReviewUnit
}

func (d *PlanDesk) load(ctx context.Context, projectID string) (*planData, error) {
	if strings.TrimSpace(projectID) == "" {
		return nil, ErrNoProjectID
	}
	sv, err := d.authoring.LatestScriptVersion(ctx, projectID)
	if err != nil {
		return nil, err
	}
	cv, err := d.store.LatestConceptVersion(ctx, projectID)
	if errors.Is(err, domain.ErrNoVersion) {
		return nil, domain.ErrConceptNotLocked
	}
	if err != nil {
		return nil, err
	}
	in, err := domain.NewPlanInput(sv, cv.Version, cv.Document)
	if err != nil {
		return nil, err
	}
	data := &planData{in: in, rows: map[string]domain.ReviewUnit{}, plans: map[string]domain.BeatPlan{}}
	rows, err := d.store.ListUnits(ctx, projectID, domain.UnitKindPlan)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		if _, ok := in.Beat(r.Key); !ok {
			continue // a beat the script no longer has
		}
		plan, err := domain.DecodeBeatPlan(r.Content)
		if err != nil {
			return nil, fmt.Errorf("đọc kế hoạch hình đã lưu của beat %s: %w", r.Key, err)
		}
		data.rows[r.Key], data.plans[r.Key] = r, plan
	}
	revs, err := d.store.ListUnits(ctx, projectID, domain.UnitKindPlanReview)
	if err != nil {
		return nil, err
	}
	data.revUnit = domain.ReviewUnit{ProjectID: projectID, Kind: domain.UnitKindPlanReview, Key: domain.ReviewUnitKey,
		Status: domain.UnitPending, CreatorFields: []string{}, InputHashes: map[string]string{}}
	for _, u := range revs {
		if u.Key == domain.ReviewUnitKey {
			data.revUnit = u
		}
	}
	if data.review, err = domain.ParseReviewState(data.revUnit.Content); err != nil {
		return nil, err
	}
	return data, nil
}

// staleSources lists the sources of beat key that changed since it was planned.
func (data *planData) staleSources(key string) []string {
	row, ok := data.rows[key]
	if !ok || len(data.plans[key].Units) == 0 {
		return nil
	}
	return data.in.StaleSources(key, row.InputHashes)
}

// locate finds the beat and position of the unit with id.
func (data *planData) locate(id string) (string, int, bool) {
	for _, b := range data.in.Script.Beats {
		if i := data.plans[b.Key].Index(id); i >= 0 {
			return b.Key, i, true
		}
	}
	return "", -1, false
}

// unitProblems lists what fails in a unit and what else keeps it from approval.
func (data *planData) unitProblems(beatKey string, u domain.PlanUnit) []string {
	return append(domain.CheckPlanUnit(u, data.in.PlanContextFor(beatKey)), domain.ApprovalBlockers(u)...)
}

// --- the screen ---------------------------------------------------------------

// PlanUnitView is one unit as the screen shows it.
type PlanUnitView struct {
	domain.PlanUnit
	BeatKey       string        `json:"beat_key"`
	Number        int           `json:"number"`
	Lines         []domain.Line `json:"lines"`
	FactRefs      []string      `json:"fact_refs"`
	NarrationMinS float64       `json:"narration_min_s"`
	EffectiveS    float64       `json:"effective_planned_s"`
	Problems      []string      `json:"problems"`
	CreatorFields []string      `json:"creator_fields"`
}

// PlanBeatView is one beat with its units.
type PlanBeatView struct {
	Key          string             `json:"key"`
	Number       int                `json:"number"`
	Fn           string             `json:"fn"`
	Title        string             `json:"title"`
	Dur          int                `json:"dur"`
	LinesCount   int                `json:"lines_count"`
	Written      bool               `json:"written"`
	Stale        bool               `json:"stale"`
	StaleSources []string           `json:"stale_sources"`
	Timing       *domain.BeatTiming `json:"timing"`
	Units        []PlanUnitView     `json:"units"`
}

// PlanFindingView is one review finding.
type PlanFindingView struct {
	domain.Finding
	GroupLabel string `json:"group_label"`
}

// PlanReviewView is the review of the whole plan.
type PlanReviewView struct {
	Reviewed  bool              `json:"reviewed"`
	Rounds    int               `json:"rounds"`
	Findings  []PlanFindingView `json:"findings"`
	Notes     []string          `json:"notes"`
	OpenCount int               `json:"open_count"`
}

// PlanInputSummary is the head card's line of inputs.
type PlanInputSummary struct {
	ScriptVersion  int     `json:"script_version"`
	ConceptVersion int     `json:"concept_version"`
	FactsCount     int     `json:"facts_count"`
	TermsCount     int     `json:"terms_count"`
	RuleRefsCount  int     `json:"rule_refs_count"`
	SecondsPerWord float64 `json:"seconds_per_word"`
}

// PlanState is the whole screen of step 9.
type PlanState struct {
	Input         PlanInputSummary    `json:"input"`
	Beats         []PlanBeatView      `json:"beats"`
	UnitsTotal    int                 `json:"units_total"`
	UnitsApproved int                 `json:"units_approved"`
	Coverage      domain.PlanCoverage `json:"coverage"`
	Timing        domain.PlanTiming   `json:"timing"`
	Checks        []CheckView         `json:"checks"`
	Review        PlanReviewView      `json:"review"`
	Blockers      []string            `json:"blockers"`
	CanLock       bool                `json:"can_lock"`
	Versions      []PlanVersionRecord `json:"versions"`
	LockedVersion int                 `json:"locked_version"`
	NextVersion   int                 `json:"next_version"`
	Draft         []string            `json:"draft"`
	// Model is the model the video's step 9 runs on; "" is the step default.
	Model        string        `json:"model"`
	DefaultModel string        `json:"default_model"`
	Models       []ModelOption `json:"models"`
}

// State returns the screen.
func (d *PlanDesk) State(ctx context.Context, projectID string) (*PlanState, error) {
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return d.state(ctx, projectID, data)
}

func (d *PlanDesk) state(ctx context.Context, projectID string, data *planData) (*PlanState, error) {
	in := data.in
	st := &PlanState{
		Input: PlanInputSummary{ScriptVersion: in.Script.ScriptVersion, ConceptVersion: in.ConceptVersion,
			FactsCount: len(in.Script.Facts), TermsCount: len(in.Script.Glossary), RuleRefsCount: len(in.RuleRefs), SecondsPerWord: in.SecondsPerWord},
		Coverage: domain.ComputeCoverage(in, data.plans), Timing: domain.ComputeTiming(in, data.plans),
		Beats: []PlanBeatView{}, Blockers: []string{}, Draft: []string{},
	}
	staleBeats, problemUnits, pendingUnits := []string{}, []string{}, 0
	for _, b := range in.Script.Beats {
		pc := in.PlanContextFor(b.Key)
		bv := PlanBeatView{Key: b.Key, Number: b.Number, Fn: b.Fn, Title: b.Title, Dur: b.Dur,
			LinesCount: len(pc.Lines), StaleSources: []string{}, Units: []PlanUnitView{}}
		if t, ok := st.Timing.Beats[b.Key]; ok {
			t := t
			bv.Timing = &t
		}
		for _, s := range data.staleSources(b.Key) {
			bv.StaleSources = append(bv.StaleSources, domain.PlanSourceLabels[s])
		}
		bv.Stale = len(bv.StaleSources) > 0
		if bv.Stale {
			staleBeats = append(staleBeats, fmt.Sprintf("beat %d", b.Number))
		}
		row := data.rows[b.Key]
		for _, u := range data.plans[b.Key].Units {
			min := pc.UnitNarrationMin(u)
			uv := PlanUnitView{PlanUnit: u, BeatKey: b.Key, Number: st.UnitsTotal + 1, FactRefs: domain.FactRefs(u, pc.Lines),
				NarrationMinS: min, EffectiveS: domain.Round1(domain.PlannedSeconds(u, min)),
				Problems: data.unitProblems(b.Key, u), CreatorFields: unitCreatorFields(row, u.UnitID)}
			uv.Lines = linesOf(u, pc.Lines)
			bv.Units = append(bv.Units, uv)
			st.UnitsTotal++
			if u.Status == domain.UnitApproved {
				st.UnitsApproved++
			} else {
				pendingUnits++
			}
			if len(uv.Problems) > 0 {
				problemUnits = append(problemUnits, u.UnitID)
			}
		}
		bv.Written = len(bv.Units) > 0
		st.Beats = append(st.Beats, bv)
	}

	st.Checks = []CheckView{}
	for _, c := range domain.PlanChecks(in, data.plans) {
		cv := CheckView{MachineCheck: c}
		if dec, ok := data.review.CheckAccepted(c); ok && !c.OK {
			cv.Accepted, cv.Why = true, dec.Why
		}
		st.Checks = append(st.Checks, cv)
	}
	st.Review = PlanReviewView{Reviewed: data.review.Rounds > 0, Rounds: data.review.Rounds, Notes: data.review.Notes,
		OpenCount: data.review.OpenCount(), Findings: []PlanFindingView{}}
	for _, f := range data.review.Findings {
		st.Review.Findings = append(st.Review.Findings, PlanFindingView{Finding: f, GroupLabel: domain.FindingGroupLabels[f.Group]})
	}

	versions, err := d.store.PlanVersions(ctx, projectID)
	if err != nil {
		return nil, err
	}
	st.Versions = versions
	if st.Versions == nil {
		st.Versions = []PlanVersionRecord{}
	}
	st.NextVersion = len(versions) + 1
	if n := len(versions); n > 0 {
		st.LockedVersion = versions[n-1].Version
		st.NextVersion = st.LockedVersion + 1
		latest, err := d.store.PlanVersion(ctx, projectID, st.LockedVersion)
		if err != nil {
			return nil, err
		}
		st.Draft = changedUnits(latest, data)
	}
	nothingNew := st.LockedVersion > 0 && len(st.Draft) == 0

	switch {
	case st.UnitsTotal == 0:
		st.Blockers = append(st.Blockers, "Chưa có unit nào trong kế hoạch hình.")
	case !st.Coverage.Complete():
		if n := len(st.Coverage.Uncovered); n > 0 {
			st.Blockers = append(st.Blockers, fmt.Sprintf("Còn %d câu lời thoại chưa thuộc unit nào.", n))
		}
		if n := len(st.Coverage.Duplicated); n > 0 {
			st.Blockers = append(st.Blockers, fmt.Sprintf("%d câu lời thoại thuộc nhiều hơn một unit.", n))
		}
	}
	if len(staleBeats) > 0 {
		st.Blockers = append(st.Blockers, "Kịch bản hoặc ngôn ngữ hình ảnh đã đổi ở "+strings.Join(staleBeats, ", ")+": lập lại kế hoạch của beat đó.")
	}
	if pendingUnits > 0 {
		st.Blockers = append(st.Blockers, fmt.Sprintf("Còn %d unit chưa duyệt.", pendingUnits))
	}
	if len(problemUnits) > 0 {
		st.Blockers = append(st.Blockers, "Unit chưa đạt điều kiện kiểm: "+strings.Join(problemUnits, ", ")+".")
	}
	if !st.Review.Reviewed {
		st.Blockers = append(st.Blockers, "Chưa rà soát kế hoạch hình.")
	}
	if st.Review.OpenCount > 0 {
		st.Blockers = append(st.Blockers, fmt.Sprintf("Còn %d vấn đề rà soát cần xử lý.", st.Review.OpenCount))
	}
	for _, c := range st.Checks {
		if !c.OK && !c.Accepted {
			st.Blockers = append(st.Blockers, "Máy kiểm chưa đạt: "+c.Label+".")
		}
	}
	if nothingNew {
		st.Blockers = append(st.Blockers, "Không có thay đổi so với v"+fmt.Sprint(st.LockedVersion)+".")
	}
	st.CanLock = len(st.Blockers) == 0

	st.Models = []ModelOption{}
	if d.models != nil {
		if st.Model, err = d.store.PlanModel(ctx, projectID); err != nil {
			return nil, err
		}
		if st.DefaultModel, err = d.models.StepDefaultModel(ctx, PlanStep); err != nil {
			return nil, err
		}
		if st.Models, err = d.models.Models(ctx); err != nil {
			return nil, err
		}
	}
	return st, nil
}

func unitCreatorFields(row domain.ReviewUnit, unitID string) []string {
	out := []string{}
	prefix := unitID + "."
	for _, f := range row.CreatorFields {
		if strings.HasPrefix(f, prefix) {
			out = append(out, strings.TrimPrefix(f, prefix))
		}
	}
	return out
}

func linesOf(u domain.PlanUnit, lines []domain.Line) []domain.Line {
	byID := map[string]domain.Line{}
	for _, l := range lines {
		byID[l.ID] = l
	}
	out := []domain.Line{}
	for _, ref := range u.ScriptRefs {
		if l, ok := byID[ref]; ok {
			out = append(out, l)
		}
	}
	return out
}

// changedUnits names the units and sources that differ from the locked document.
func changedUnits(latest PlanVersionRecord, data *planData) []string {
	var old domain.PlanDocument
	if err := json.Unmarshal(latest.Document, &old); err != nil {
		return []string{"Không đọc được bản chốt trước"}
	}
	var out []string
	if old.SourceVersions.Script != data.in.Script.ScriptVersion {
		out = append(out, fmt.Sprintf("Kịch bản chốt v%d → v%d", old.SourceVersions.Script, data.in.Script.ScriptVersion))
	}
	if old.SourceVersions.VisualConcept != data.in.ConceptVersion {
		out = append(out, fmt.Sprintf("Ngôn ngữ hình ảnh v%d → v%d", old.SourceVersions.VisualConcept, data.in.ConceptVersion))
	}
	before := map[string]domain.DocumentUnit{}
	for _, u := range old.Units {
		u.Order = 0
		before[u.UnitID] = u
	}
	for _, u := range domain.BuildPlanUnits(data.in, data.plans) {
		u.Order = 0
		prev, ok := before[u.UnitID]
		delete(before, u.UnitID)
		a, _ := json.Marshal(u)
		b, _ := json.Marshal(prev)
		switch {
		case !ok:
			out = append(out, u.UnitID+" · mới")
		case !sameJSON(a, b):
			out = append(out, u.UnitID+" · đổi")
		}
	}
	for id := range before {
		out = append(out, id+" · bỏ")
	}
	return out
}

// --- locking ---------------------------------------------------------------------

// Lock freezes the units into a new locked version.
func (d *PlanDesk) Lock(ctx context.Context, projectID, note string) (*PlanState, error) {
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	st, err := d.state(ctx, projectID, data)
	if err != nil {
		return nil, err
	}
	if !st.CanLock {
		if st.LockedVersion > 0 && len(st.Draft) == 0 {
			return nil, domain.ErrNothingToLock
		}
		return nil, fmt.Errorf("%w: %s", domain.ErrLockNotReady, strings.Join(st.Blockers, " "))
	}
	note = strings.TrimSpace(note)
	if r := []rune(note); len(r) > MaxLockNoteRunes {
		note = string(r[:MaxLockNoteRunes])
	}
	if note == "" {
		if st.LockedVersion == 0 {
			note = "Chốt lần đầu"
		} else {
			note = fmt.Sprintf("Chốt %d thay đổi", len(st.Draft))
		}
	}
	lockedAt := d.now().UTC()
	doc, err := domain.BuildPlanDocument(projectID, st.NextVersion, lockedAt, data.in, data.plans, data.review)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}
	if err := d.store.InsertPlanVersion(ctx, PlanVersionRecord{ProjectID: projectID, Version: st.NextVersion,
		ScriptVersion: data.in.Script.ScriptVersion, ConceptVersion: data.in.ConceptVersion, Document: raw, Note: note, LockedAt: lockedAt}); err != nil {
		return nil, err
	}
	return d.State(ctx, projectID)
}

// Version returns a locked version with its document.
func (d *PlanDesk) Version(ctx context.Context, projectID string, version int) (PlanVersionRecord, error) {
	if strings.TrimSpace(projectID) == "" {
		return PlanVersionRecord{}, ErrNoProjectID
	}
	return d.store.PlanVersion(ctx, projectID, version)
}

// LatestDocument returns the latest locked plan document for a later step.
func (d *PlanDesk) LatestDocument(ctx context.Context, projectID string) (json.RawMessage, error) {
	if strings.TrimSpace(projectID) == "" {
		return nil, ErrNoProjectID
	}
	versions, err := d.store.PlanVersions(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if len(versions) == 0 {
		return nil, domain.ErrNoVersion
	}
	v, err := d.store.PlanVersion(ctx, projectID, versions[len(versions)-1].Version)
	if err != nil {
		return nil, err
	}
	return v.Document, nil
}
