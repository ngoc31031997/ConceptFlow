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

// SequenceDesk is step 10 "Chuỗi hình": it reads the locked visual plan, the
// locked visual concept and the locked script, keeps what the step adds to every
// unit of the plan (purpose in the flow, pacing, transition, motifs) and the
// video's motifs, reviews them and locks versions.
type SequenceDesk struct {
	authoring AuthoringInputPort
	store     SequenceStorePort
	now       func() time.Time
	running   *runningSet

	// The AI side; unset until WithAI, and then the AI routes fail with
	// ErrLLMNotConfigured.
	llm       LLMPort
	prompts   PromptLibraryPort
	models    ModelCatalogPort
	maxTokens int
}

// NewSequenceDesk wires the desk without an AI.
func NewSequenceDesk(authoring AuthoringInputPort, store SequenceStorePort) *SequenceDesk {
	return &SequenceDesk{authoring: authoring, store: store, now: time.Now, running: &runningSet{m: map[string]map[string]bool{}}}
}

// WithAI gives the desk the model, the prompt library and the model catalog it
// writes and reviews with.
func (d *SequenceDesk) WithAI(llm LLMPort, prompts PromptLibraryPort, models ModelCatalogPort, maxTokens int) *SequenceDesk {
	d.llm, d.prompts, d.models, d.maxTokens = llm, prompts, models, maxTokens
	return d
}

// sequenceData is everything one request works on.
type sequenceData struct {
	in        domain.SequenceInput
	rows      map[string]domain.ReviewUnit
	beats     map[string]domain.BeatSequence
	motifRow  domain.ReviewUnit
	motifs    domain.MotifSet
	review    domain.ReviewState
	reviewRow domain.ReviewUnit
}

func (d *SequenceDesk) load(ctx context.Context, projectID string) (*sequenceData, error) {
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
	pv, err := d.store.LatestPlanVersion(ctx, projectID)
	if errors.Is(err, domain.ErrNoVersion) {
		return nil, domain.ErrPlanNotLocked
	}
	if err != nil {
		return nil, err
	}
	in, err := domain.NewSequenceInput(sv, cv.Version, cv.Document, pv.Version, pv.Document)
	if err != nil {
		return nil, err
	}
	data := &sequenceData{in: in, rows: map[string]domain.ReviewUnit{}, beats: map[string]domain.BeatSequence{}}
	rows, err := d.store.ListUnits(ctx, projectID, domain.UnitKindSequence)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		s, err := domain.DecodeBeatSequence(r.Content)
		if err != nil {
			return nil, fmt.Errorf("đọc chuỗi hình đã lưu của beat %s: %w", r.Key, err)
		}
		data.rows[r.Key], data.beats[r.Key] = r, s
	}
	if data.motifRow, err = d.singleRow(ctx, projectID, domain.UnitKindSequenceMotifs); err != nil {
		return nil, err
	}
	if data.motifs, err = domain.DecodeMotifSet(data.motifRow.Content); err != nil {
		return nil, fmt.Errorf("đọc motif đã lưu: %w", err)
	}
	if data.reviewRow, err = d.singleRow(ctx, projectID, domain.UnitKindSequenceReview); err != nil {
		return nil, err
	}
	if data.review, err = domain.ParseReviewState(data.reviewRow.Content); err != nil {
		return nil, err
	}
	return data, nil
}

// singleRow returns the one row of kind under ReviewUnitKey, or a new pending
// row with no hashes when there is none.
func (d *SequenceDesk) singleRow(ctx context.Context, projectID, kind string) (domain.ReviewUnit, error) {
	rows, err := d.store.ListUnits(ctx, projectID, kind)
	if err != nil {
		return domain.ReviewUnit{}, err
	}
	for _, r := range rows {
		if r.Key == domain.ReviewUnitKey {
			return r, nil
		}
	}
	return domain.ReviewUnit{ProjectID: projectID, Kind: kind, Key: domain.ReviewUnitKey, Status: domain.UnitPending,
		CreatorFields: []string{}, InputHashes: map[string]string{}}, nil
}

// beatStaleSources lists the sources of beat key that changed since its items
// were written; a beat with no item is never stale.
func (data *sequenceData) beatStaleSources(key string) []string {
	row, ok := data.rows[key]
	if !ok || len(data.beats[key].Items) == 0 {
		return nil
	}
	return data.in.BeatStaleSources(key, row.InputHashes, data.motifs)
}

// motifStaleSources lists the sources that changed since the motifs were
// written; no motif is never stale.
func (data *sequenceData) motifStaleSources() []string {
	if len(data.motifs.Motifs) == 0 {
		return nil
	}
	return data.in.MotifStaleSources(data.motifRow.InputHashes)
}

// itemProblems lists what fails in the item of plan unit id.
func (data *sequenceData) itemProblems(unitID string, it domain.SeqItem) []string {
	_, first, _ := data.in.LocateUnit(unitID)
	return append([]string{}, domain.CheckSeqItem(it, first, data.motifs.IDs())...)
}

// --- the screen ---------------------------------------------------------------

// SequenceUnitView is one plan unit with what step 10 adds to it.
type SequenceUnitView struct {
	UnitID                string                `json:"unit_id"`
	Order                 int                   `json:"order"`
	ScriptRefs            []string              `json:"script_refs"`
	PlannedS              float64               `json:"planned_s"`
	CommunicationGoal     string                `json:"communication_goal"`
	VisualApproach        domain.VisualApproach `json:"visual_approach"`
	NarrationRelationship string                `json:"narration_relationship"`
	PreviousUnit          string                `json:"previous_unit"`
	// Item is nil while the unit has no sequence item yet.
	Item          *domain.SeqItem `json:"item"`
	Problems      []string        `json:"problems"`
	CreatorFields []string        `json:"creator_fields"`
}

// SequenceBeatView is one beat with its units.
type SequenceBeatView struct {
	Key          string             `json:"key"`
	Number       int                `json:"number"`
	Fn           string             `json:"fn"`
	Title        string             `json:"title"`
	Pace         string             `json:"pace"`
	Written      bool               `json:"written"`
	Approved     bool               `json:"approved"`
	Stale        bool               `json:"stale"`
	StaleSources []string           `json:"stale_sources"`
	Units        []SequenceUnitView `json:"units"`
}

// MotifView is one motif as the screen shows it.
type MotifView struct {
	domain.Motif
	// AnchorText is what step 8 locked on how to draw the anchor; "" when the
	// anchor does not exist.
	AnchorText    string   `json:"anchor_text"`
	UnitRefs      []string `json:"unit_refs"`
	Problems      []string `json:"problems"`
	CreatorFields []string `json:"creator_fields"`
}

// MotifsView is the motifs of the video, approved together.
type MotifsView struct {
	Status       string      `json:"status"`
	Stale        bool        `json:"stale"`
	StaleSources []string    `json:"stale_sources"`
	Motifs       []MotifView `json:"motifs"`
	// Anchors are the representation rules a motif can anchor to, with their text.
	Anchors map[string]string `json:"anchors"`
}

// SequenceInputSummary is the head card's line of inputs.
type SequenceInputSummary struct {
	PlanVersion    int      `json:"plan_version"`
	ConceptVersion int      `json:"concept_version"`
	ScriptVersion  int      `json:"script_version"`
	UnitsCount     int      `json:"units_count"`
	Outdated       []string `json:"outdated"`
}

// SequenceState is the whole screen of step 10.
type SequenceState struct {
	Input         SequenceInputSummary    `json:"input"`
	Motifs        MotifsView              `json:"motifs"`
	Beats         []SequenceBeatView      `json:"beats"`
	ItemsApproved int                     `json:"items_approved"`
	Coverage      domain.SequenceCoverage `json:"coverage"`
	Checks        []CheckView             `json:"checks"`
	Review        PlanReviewView          `json:"review"`
	Blockers      []string                `json:"blockers"`
	CanLock       bool                    `json:"can_lock"`
	Versions      []SequenceVersionRecord `json:"versions"`
	LockedVersion int                     `json:"locked_version"`
	NextVersion   int                     `json:"next_version"`
	Draft         []string                `json:"draft"`
	// PendingUnits are the units whose item waits for review, in plan order.
	PendingUnits []string `json:"pending_units"`
	// Model is the model the video's step 10 runs on; "" is the step default.
	Model        string        `json:"model"`
	DefaultModel string        `json:"default_model"`
	Models       []ModelOption `json:"models"`
}

// maxNamedUnits is how many units a blocker names before it counts the rest.
const maxNamedUnits = 10

// State returns the screen.
func (d *SequenceDesk) State(ctx context.Context, projectID string) (*SequenceState, error) {
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return d.state(ctx, projectID, data)
}

func (d *SequenceDesk) state(ctx context.Context, projectID string, data *sequenceData) (*SequenceState, error) {
	in := data.in
	st := &SequenceState{
		Input: SequenceInputSummary{PlanVersion: in.PlanVersion, ConceptVersion: in.ConceptVersion, ScriptVersion: in.Script.ScriptVersion,
			UnitsCount: len(in.Plan.Units), Outdated: in.Outdated},
		Coverage: domain.ComputeSequenceCoverage(in, data.beats),
		Beats:    []SequenceBeatView{}, Blockers: []string{}, Draft: []string{},
	}
	st.Motifs = data.motifsView()
	staleBeats, problemUnits := []string{}, []string{}
	st.PendingUnits = []string{}
	for _, b := range in.Beats {
		bv := data.beatView(b)
		if bv.Stale {
			staleBeats = append(staleBeats, fmt.Sprintf("beat %d", b.Number))
		}
		for _, uv := range bv.Units {
			if uv.Item == nil {
				continue
			}
			if uv.Item.Status == domain.UnitApproved {
				st.ItemsApproved++
			} else {
				st.PendingUnits = append(st.PendingUnits, uv.UnitID)
			}
			if len(uv.Problems) > 0 {
				problemUnits = append(problemUnits, uv.UnitID)
			}
		}
		st.Beats = append(st.Beats, bv)
	}

	st.Checks = []CheckView{}
	for _, c := range domain.SequenceChecks(in, data.beats, data.motifs) {
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

	versions, err := d.store.SequenceVersions(ctx, projectID)
	if err != nil {
		return nil, err
	}
	st.Versions = versions
	if st.Versions == nil {
		st.Versions = []SequenceVersionRecord{}
	}
	st.NextVersion = 1
	if n := len(versions); n > 0 {
		st.LockedVersion = versions[n-1].Version
		st.NextVersion = st.LockedVersion + 1
		latest, err := d.store.SequenceVersion(ctx, projectID, st.LockedVersion)
		if err != nil {
			return nil, err
		}
		st.Draft = changedSequence(latest, data)
	}

	if len(in.Outdated) > 0 {
		st.Blockers = append(st.Blockers, "Kế hoạch hình dựa trên bản cũ ("+strings.Join(in.Outdated, "; ")+"): chốt lại bước 9.")
	}
	st.Blockers = append(st.Blockers, motifBlockers(st.Motifs)...)
	if n := len(st.Coverage.Missing); n > 0 {
		st.Blockers = append(st.Blockers, fmt.Sprintf("Còn %d unit chưa có chuỗi.", n))
	}
	if n := len(st.Coverage.Extra); n > 0 {
		st.Blockers = append(st.Blockers, fmt.Sprintf("%d unit không còn trong kế hoạch hình: viết lại beat của chúng.", n))
	}
	if len(staleBeats) > 0 {
		st.Blockers = append(st.Blockers, "Đầu vào đã đổi ở "+strings.Join(staleBeats, ", ")+": viết lại hoặc xác nhận beat đó.")
	}
	if n := len(st.PendingUnits); n > 0 {
		names := st.PendingUnits
		if n > maxNamedUnits {
			names = names[:maxNamedUnits]
		}
		more := ""
		if n > maxNamedUnits {
			more = fmt.Sprintf(" và %d unit khác", n-maxNamedUnits)
		}
		st.Blockers = append(st.Blockers, fmt.Sprintf("Còn %d unit chưa duyệt: %s%s.", n, strings.Join(names, ", "), more))
	}
	if len(problemUnits) > 0 {
		st.Blockers = append(st.Blockers, "Unit chưa đạt điều kiện kiểm: "+strings.Join(problemUnits, ", ")+".")
	}
	if !st.Review.Reviewed {
		st.Blockers = append(st.Blockers, "Chưa rà soát chuỗi hình.")
	}
	if st.Review.OpenCount > 0 {
		st.Blockers = append(st.Blockers, fmt.Sprintf("Còn %d vấn đề rà soát cần xử lý.", st.Review.OpenCount))
	}
	for _, c := range st.Checks {
		if !c.OK && !c.Accepted {
			st.Blockers = append(st.Blockers, "Máy kiểm chưa đạt: "+c.Label+".")
		}
	}
	if st.LockedVersion > 0 && len(st.Draft) == 0 {
		st.Blockers = append(st.Blockers, fmt.Sprintf("Không có thay đổi so với v%d.", st.LockedVersion))
	}
	st.CanLock = len(st.Blockers) == 0

	st.Models = []ModelOption{}
	if d.models != nil {
		var err error
		if st.Model, err = d.store.SequenceModel(ctx, projectID); err != nil {
			return nil, err
		}
		if st.DefaultModel, err = d.models.StepDefaultModel(ctx, SequenceStep); err != nil {
			return nil, err
		}
		if st.Models, err = d.models.Models(ctx); err != nil {
			return nil, err
		}
	}
	return st, nil
}

// motifBlockers says why the motifs keep the sequence from locking.
func motifBlockers(m MotifsView) []string {
	switch {
	case len(m.Motifs) == 0:
		return []string{"Chưa có motif nào."}
	case m.Stale:
		return []string{"Kế hoạch hình hoặc ngôn ngữ hình ảnh đã đổi từ lúc đặt motif: xem lại và duyệt motif."}
	case m.Status != domain.UnitApproved:
		return []string{"Motif chưa duyệt."}
	}
	return nil
}

func (data *sequenceData) motifsView() MotifsView {
	mv := MotifsView{Status: data.motifRow.Status, StaleSources: []string{}, Motifs: []MotifView{}, Anchors: data.in.AnchorRefs}
	for _, s := range data.motifStaleSources() {
		mv.StaleSources = append(mv.StaleSources, domain.SeqSourceLabels[s])
	}
	mv.Stale = len(mv.StaleSources) > 0
	seen := domain.MotifAppearances(data.in, data.beats, data.motifs)
	for _, m := range data.motifs.Motifs {
		mv.Motifs = append(mv.Motifs, MotifView{Motif: m, AnchorText: data.in.AnchorRefs[m.AnchorRef], UnitRefs: seen[m.MotifID],
			Problems:      append([]string{}, domain.CheckMotif(m, data.in.AnchorRefs)...),
			CreatorFields: unitCreatorFields(data.motifRow, m.MotifID)})
	}
	return mv
}

func (data *sequenceData) beatView(b domain.SequenceBeat) SequenceBeatView {
	bv := SequenceBeatView{Key: b.Key, Number: b.Number, Fn: b.Fn, Title: b.Title, Pace: b.Pace,
		StaleSources: []string{}, Units: []SequenceUnitView{}}
	for _, s := range data.beatStaleSources(b.Key) {
		bv.StaleSources = append(bv.StaleSources, domain.SeqSourceLabels[s])
	}
	bv.Stale = len(bv.StaleSources) > 0
	row, seq := data.rows[b.Key], data.beats[b.Key]
	approved := 0
	for _, u := range b.Units {
		uv := SequenceUnitView{UnitID: u.UnitID, Order: u.Order, ScriptRefs: u.ScriptRefs, PlannedS: u.Duration.PlannedS,
			CommunicationGoal: u.CommunicationGoal, VisualApproach: u.VisualApproach, NarrationRelationship: u.NarrationRelationship,
			PreviousUnit: data.in.PreviousUnit(u.UnitID), Problems: []string{}, CreatorFields: unitCreatorFields(row, u.UnitID)}
		if i := seq.Index(u.UnitID); i >= 0 {
			it := seq.Items[i]
			uv.Item = &it
			uv.Problems = data.itemProblems(u.UnitID, it)
			bv.Written = true
			if it.Status == domain.UnitApproved {
				approved++
			}
		}
		bv.Units = append(bv.Units, uv)
	}
	bv.Approved = approved == len(b.Units)
	return bv
}

// changedSequence names the items, motifs and sources that differ from the
// locked document.
func changedSequence(latest SequenceVersionRecord, data *sequenceData) []string {
	var old domain.SequenceDocument
	if err := json.Unmarshal(latest.Document, &old); err != nil {
		return []string{"Không đọc được bản chốt trước"}
	}
	in := data.in
	var out []string
	if old.SourceVersions.VisualPlan != in.PlanVersion {
		out = append(out, fmt.Sprintf("Kế hoạch hình v%d → v%d", old.SourceVersions.VisualPlan, in.PlanVersion))
	}
	before := map[string]domain.DocumentSeqItem{}
	for _, it := range old.Sequence {
		it.SequenceIndex = 0
		before[it.UnitID] = it
	}
	for _, it := range domain.BuildSequenceItems(in, data.beats) {
		it.SequenceIndex = 0
		out = appendChange(out, it.UnitID, it, before)
	}
	for id := range before {
		out = append(out, id+" · bỏ")
	}
	oldMotifs := map[string]domain.DocumentMotif{}
	for _, m := range old.Motifs {
		oldMotifs[m.MotifID] = m
	}
	for _, m := range domain.BuildSequenceMotifs(in, data.beats, data.motifs) {
		out = appendChange(out, m.MotifID, m, oldMotifs)
	}
	for id := range oldMotifs {
		out = append(out, id+" · bỏ")
	}
	return out
}

// appendChange adds "id · mới" or "id · đổi" when now is new or differs from
// its entry in before, and takes the entry out of before.
func appendChange[T any](out []string, id string, now T, before map[string]T) []string {
	prev, ok := before[id]
	delete(before, id)
	a, _ := json.Marshal(now)
	b, _ := json.Marshal(prev)
	switch {
	case !ok:
		return append(out, id+" · mới")
	case !sameJSON(a, b):
		return append(out, id+" · đổi")
	}
	return out
}

// --- locking ---------------------------------------------------------------------

// Lock freezes the sequence into a new locked version. A plan written from an
// older script or visual concept is domain.ErrPlanOutdated.
func (d *SequenceDesk) Lock(ctx context.Context, projectID, note string) (*SequenceState, error) {
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	st, err := d.state(ctx, projectID, data)
	if err != nil {
		return nil, err
	}
	if len(data.in.Outdated) > 0 {
		return nil, fmt.Errorf("%w: %s", domain.ErrPlanOutdated, strings.Join(data.in.Outdated, "; "))
	}
	if !st.CanLock {
		if st.LockedVersion > 0 && len(st.Draft) == 0 {
			return nil, domain.ErrNothingToLock
		}
		return nil, fmt.Errorf("%w: %s", domain.ErrLockNotReady, strings.Join(st.Blockers, " "))
	}
	note = lockNote(note, st.LockedVersion, len(st.Draft))
	lockedAt := d.now().UTC()
	doc, err := domain.BuildSequenceDocument(projectID, st.NextVersion, lockedAt, data.in, data.beats, data.motifs, data.review)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}
	if err := d.store.InsertSequenceVersion(ctx, SequenceVersionRecord{ProjectID: projectID, Version: st.NextVersion,
		PlanVersion: data.in.PlanVersion, ConceptVersion: data.in.ConceptVersion, ScriptVersion: data.in.Script.ScriptVersion,
		Document: raw, Note: note, LockedAt: lockedAt}); err != nil {
		return nil, err
	}
	return d.State(ctx, projectID)
}

// lockNote trims the Creator's note to MaxLockNoteRunes, or names the lock when
// it is empty.
func lockNote(note string, lockedVersion, changes int) string {
	note = strings.TrimSpace(note)
	if r := []rune(note); len(r) > MaxLockNoteRunes {
		note = string(r[:MaxLockNoteRunes])
	}
	switch {
	case note != "":
		return note
	case lockedVersion == 0:
		return "Chốt lần đầu"
	default:
		return fmt.Sprintf("Chốt %d thay đổi", changes)
	}
}

// Version returns a locked version with its document.
func (d *SequenceDesk) Version(ctx context.Context, projectID string, version int) (SequenceVersionRecord, error) {
	if strings.TrimSpace(projectID) == "" {
		return SequenceVersionRecord{}, ErrNoProjectID
	}
	return d.store.SequenceVersion(ctx, projectID, version)
}

// LatestDocument returns the latest locked sequence document for a later step,
// or domain.ErrNoVersion.
func (d *SequenceDesk) LatestDocument(ctx context.Context, projectID string) (json.RawMessage, error) {
	if strings.TrimSpace(projectID) == "" {
		return nil, ErrNoProjectID
	}
	versions, err := d.store.SequenceVersions(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if len(versions) == 0 {
		return nil, domain.ErrNoVersion
	}
	v, err := d.store.SequenceVersion(ctx, projectID, versions[len(versions)-1].Version)
	if err != nil {
		return nil, err
	}
	return v.Document, nil
}
