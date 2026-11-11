package application

import (
	"context"
	"encoding/json"

	"fmt"
	"strings"
	"sync"
	"time"

	"visual/internal/domain"
)

// ConceptStep is the step name of the visual concept in llm-service (default
// model, call log).
const ConceptStep = "visual_concept"

// MaxLockNoteRunes bounds the note a version is locked with.
const MaxLockNoteRunes = 200

// ConceptDesk is step 8 "Ngôn ngữ hình ảnh": it reads the locked script, keeps
// the eight parts of the visual concept and their review, and locks versions.
type ConceptDesk struct {
	authoring AuthoringInputPort
	store     ConceptStorePort
	refs      ReferenceReader
	llm       LLMPort
	prompts   PromptLibraryPort
	models    ModelCatalogPort
	fonts     FontCatalogPort
	maxTokens int
	now       func() time.Time
	running   *runningSet
}

// NewConceptDesk wires the desk. fonts may be nil: the font of part 5 is then
// not checked against rendering.
func NewConceptDesk(authoring AuthoringInputPort, store ConceptStorePort, refs ReferenceReader, llm LLMPort, prompts PromptLibraryPort,
	models ModelCatalogPort, fonts FontCatalogPort, maxTokens int) *ConceptDesk {
	return &ConceptDesk{authoring: authoring, store: store, refs: refs, llm: llm, prompts: prompts, models: models, fonts: fonts,
		maxTokens: maxTokens, now: time.Now, running: &runningSet{m: map[string]map[string]bool{}}}
}

// runningSet holds the parts the AI is writing now, per video. It lives in the
// process: a unit is stored only once the AI's answer is in (ADR-0047).
type runningSet struct {
	mu sync.Mutex
	m  map[string]map[string]bool
}

func (r *runningSet) start(projectID string, keys ...string) func() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.m[projectID] == nil {
		r.m[projectID] = map[string]bool{}
	}
	for _, k := range keys {
		r.m[projectID][k] = true
	}
	return func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		for _, k := range keys {
			delete(r.m[projectID], k)
		}
	}
}

func (r *runningSet) done(projectID, key string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.m[projectID], key)
}

func (r *runningSet) has(projectID, key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.m[projectID][key]
}

// --- loading -------------------------------------------------------------------

// deskData is everything one request works on.
type deskData struct {
	in      domain.ConceptInput
	units   map[string]domain.ReviewUnit
	parts   domain.Parts
	review  domain.ReviewState
	revUnit domain.ReviewUnit
	fonts   []string
	// fontsErr is why the fonts could not be listed; fonts is empty then.
	fontsErr error
}

func (d *ConceptDesk) load(ctx context.Context, projectID string) (*deskData, error) {
	if strings.TrimSpace(projectID) == "" {
		return nil, ErrNoProjectID
	}
	sv, err := d.authoring.LatestScriptVersion(ctx, projectID)
	if err != nil {
		return nil, err
	}
	refs, err := d.refs.PickedReferences(ctx, projectID)
	if err != nil {
		return nil, err
	}
	in, err := domain.NewConceptInput(sv, refs)
	if err != nil {
		return nil, err
	}
	data := &deskData{in: in, units: map[string]domain.ReviewUnit{}}
	// A rendering that does not answer does not stop the screen: it blocks the
	// font check, so the typography part cannot be approved or locked until
	// rendering lists its fonts.
	data.fonts, data.fontsErr = d.fonts.Fonts(ctx)
	list, err := d.store.ListUnits(ctx, projectID, domain.UnitKindConcept)
	if err != nil {
		return nil, err
	}
	for _, u := range list {
		if _, ok := domain.UnitSpecOf(u.Key); !ok {
			continue
		}
		data.units[u.Key] = u
		if v, _, err := domain.DecodePart(u.Key, u.Content); err == nil {
			data.parts.Set(u.Key, v)
		}
	}
	if data.parts.Feasibility != nil {
		domain.MachineScope(data.parts.Feasibility, len(in.Beats), data.parts.RepresentationRules)
	}
	revs, err := d.store.ListUnits(ctx, projectID, domain.UnitKindReview)
	if err != nil {
		return nil, err
	}
	data.revUnit = domain.ReviewUnit{ProjectID: projectID, Kind: domain.UnitKindReview, Key: domain.ReviewUnitKey,
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

func (data *deskData) checkContext() domain.CheckContext {
	facts := map[string]bool{}
	for _, f := range data.in.Facts {
		facts[f.ID] = true
	}
	refs := map[string]bool{}
	for _, r := range data.in.Refs {
		refs[r.ID] = true
	}
	surface, layout := domain.RefsNeed(data.in.Refs)
	return domain.CheckContext{BeatKeys: data.in.BeatKeys(), FactIDs: facts, RefIDs: refs, Fonts: data.fonts, NeedSurface: surface, NeedLayout: layout}
}

// unitProblems lists what fails in the stored part key.
func (data *deskData) unitProblems(key string) []string {
	u, ok := data.units[key]
	if !ok || len(u.Content) == 0 {
		return nil
	}
	if key == domain.UnitTypography && data.fontsErr != nil {
		return []string{"Không kiểm được font: " + data.fontsErr.Error()}
	}
	v, _, err := domain.DecodePart(key, u.Content)
	if err != nil {
		return []string{err.Error()}
	}
	if f, ok := v.(*domain.Feasibility); ok {
		domain.MachineScope(f, len(data.in.Beats), data.parts.RepresentationRules)
	}
	return domain.CheckPart(key, v, data.checkContext())
}

// staleSources lists the sources of key that changed since it was written.
func (data *deskData) staleSources(key string) []string {
	u, ok := data.units[key]
	spec, _ := domain.UnitSpecOf(key)
	if !ok || len(u.Content) == 0 {
		return nil
	}
	var out []string
	for _, s := range spec.Reads {
		if u.InputHashes[s] != data.in.Hashes[s] {
			out = append(out, s)
		}
	}
	return out
}

func (data *deskData) hashesFor(key string) map[string]string {
	spec, _ := domain.UnitSpecOf(key)
	out := map[string]string{}
	for _, s := range spec.Reads {
		out[s] = data.in.Hashes[s]
	}
	return out
}

// --- the screen --------------------------------------------------------------------

// UnitView is one part as the screen shows it.
type UnitView struct {
	Key           string          `json:"key"`
	Number        int             `json:"number"`
	Title         string          `json:"title"`
	Status        string          `json:"status"`
	Written       bool            `json:"written"`
	Content       json.RawMessage `json:"content"`
	CreatorFields []string        `json:"creator_fields"`
	Reads         []string        `json:"reads"`
	Stale         bool            `json:"stale"`
	StaleSources  []string        `json:"stale_sources"`
	Problems      []string        `json:"problems"`
	RedoCount     int             `json:"redo_count"`
}

// CheckView is one machine check with the Creator's decision on it.
type CheckView struct {
	domain.MachineCheck
	Accepted bool   `json:"accepted"`
	Why      string `json:"why"`
}

// FindingView is one review finding.
type FindingView struct {
	domain.Finding
	GroupLabel string `json:"group_label"`
	UnitNumber int    `json:"unit_number"`
	UnitTitle  string `json:"unit_title"`
}

// ReviewView is the review of the whole concept.
type ReviewView struct {
	Reviewed  bool          `json:"reviewed"`
	Rounds    int           `json:"rounds"`
	Running   bool          `json:"running"`
	Findings  []FindingView `json:"findings"`
	Notes     []string      `json:"notes"`
	OpenCount int           `json:"open_count"`
}

// Criterion is one of the five completion criteria.
type Criterion struct {
	Label  string `json:"label"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
}

// BeatView is one beat as the head card's colour strip shows it.
type BeatView struct {
	Key    string `json:"key"`
	Number int    `json:"number"`
	Fn     string `json:"fn"`
	Title  string `json:"title"`
	Dur    int    `json:"dur"`
}

// RefView is one description of the library with whether the video picked it.
type RefView struct {
	domain.ReferenceStyle
	Picked bool `json:"picked"`
}

// InputSummary is the head card's line of inputs.
type InputSummary struct {
	ScriptVersion   int        `json:"script_version"`
	FactsCount      int        `json:"facts_count"`
	ResearchSkipped bool       `json:"research_skipped"`
	Beats           []BeatView `json:"beats"`
	TermsCount      int        `json:"terms_count"`
	RefsCount       int        `json:"refs_count"`
}

// ConceptState is the whole screen of step 8.
type ConceptState struct {
	Input         InputSummary    `json:"input"`
	InputView     []InputSection  `json:"input_view"`
	Refs          []RefView       `json:"refs"`
	Units         []UnitView      `json:"units"`
	ApprovedCount int             `json:"approved_count"`
	Checks        []CheckView     `json:"checks"`
	FontsError    string          `json:"fonts_error"`
	Fonts         []string        `json:"fonts"`
	Review        ReviewView      `json:"review"`
	Criteria      []Criterion     `json:"criteria"`
	Blockers      []string        `json:"blockers"`
	CanLock       bool            `json:"can_lock"`
	Versions      []VersionRecord `json:"versions"`
	LockedVersion int             `json:"locked_version"`
	NextVersion   int             `json:"next_version"`
	Draft         []string        `json:"draft"`
	Model         string          `json:"model"`
	DefaultModel  string          `json:"default_model"`
	Models        []ModelOption   `json:"models"`
}

// State returns the screen.
func (d *ConceptDesk) State(ctx context.Context, projectID string) (*ConceptState, error) {
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return d.state(ctx, projectID, data)
}

func (d *ConceptDesk) state(ctx context.Context, projectID string, data *deskData) (*ConceptState, error) {
	in := data.in
	st := &ConceptState{
		InputView: inputSections(in), Fonts: data.fonts,
		Input: InputSummary{ScriptVersion: in.ScriptVersion, FactsCount: len(in.Facts), ResearchSkipped: in.ResearchSkipped,
			TermsCount: len(in.Glossary), RefsCount: len(in.Refs)},
	}
	if st.Fonts == nil {
		st.Fonts = []string{}
	}
	for _, b := range in.Beats {
		st.Input.Beats = append(st.Input.Beats, BeatView{Key: b.Key, Number: b.Number, Fn: b.Fn, Title: b.Title, Dur: b.Dur})
	}
	library, err := d.refs.ReferenceLibrary(ctx)
	if err != nil {
		return nil, err
	}
	picked := map[string]bool{}
	for _, r := range in.Refs {
		picked[r.ID] = true
	}
	st.Refs = []RefView{}
	for _, r := range library {
		st.Refs = append(st.Refs, RefView{ReferenceStyle: r, Picked: picked[r.ID]})
	}

	allApprovedFresh := true
	for i, spec := range domain.ConceptUnits {
		u, ok := data.units[spec.Key]
		v := UnitView{Key: spec.Key, Number: i + 1, Title: spec.Title, Status: domain.UnitPending,
			CreatorFields: []string{}, StaleSources: []string{}, Problems: []string{}}
		for _, s := range spec.Reads {
			v.Reads = append(v.Reads, domain.SourceLabels[s])
		}
		if ok && len(u.Content) > 0 {
			v.Written, v.Status, v.RedoCount = true, u.Status, u.RedoCount
			v.Content = u.Content
			if spec.Key == domain.UnitFeasibility && data.parts.Feasibility != nil {
				v.Content, _ = json.Marshal(data.parts.Feasibility)
			}
			v.CreatorFields = append(v.CreatorFields, u.CreatorFields...)
			for _, s := range data.staleSources(spec.Key) {
				v.StaleSources = append(v.StaleSources, domain.SourceLabels[s])
			}
			v.Stale = len(v.StaleSources) > 0
			v.Problems = append(v.Problems, data.unitProblems(spec.Key)...)
		}
		if d.running.has(projectID, spec.Key) {
			v.Status = domain.UnitRunning
		}
		if v.Status == domain.UnitApproved {
			st.ApprovedCount++
		}
		if v.Status != domain.UnitApproved || v.Stale {
			allApprovedFresh = false
		}
		st.Units = append(st.Units, v)
	}

	checksOK := map[string]bool{}
	for _, c := range domain.MachineChecks(in, data.parts) {
		cv := CheckView{MachineCheck: c}
		if dec, ok := data.review.CheckAccepted(c); ok && !c.OK {
			cv.Accepted, cv.Why = true, dec.Why
		}
		checksOK[c.ID] = c.OK || cv.Accepted
		st.Checks = append(st.Checks, cv)
	}

	st.Review = ReviewView{Reviewed: data.review.Rounds > 0, Rounds: data.review.Rounds, Notes: data.review.Notes,
		OpenCount: data.review.OpenCount(), Running: d.running.has(projectID, domain.ReviewUnitKey)}
	openIn := map[string]bool{}
	for _, f := range data.review.Findings {
		spec, _ := domain.UnitSpecOf(f.UnitKey)
		st.Review.Findings = append(st.Review.Findings, FindingView{Finding: f, GroupLabel: domain.FindingGroupLabels[f.Group],
			UnitNumber: domain.UnitNumber(f.UnitKey), UnitTitle: spec.Title})
		if f.State == domain.FindingOpen {
			openIn[f.Group] = true
		}
	}
	if st.Review.Findings == nil {
		st.Review.Findings = []FindingView{}
	}

	versions, err := d.store.Versions(ctx, projectID)
	if err != nil {
		return nil, err
	}
	st.Versions = versions
	if st.Versions == nil {
		st.Versions = []VersionRecord{}
	}
	st.NextVersion = len(versions) + 1
	st.Draft = []string{}
	if n := len(versions); n > 0 {
		st.LockedVersion = versions[n-1].Version
		st.NextVersion = st.LockedVersion + 1
		latest, err := d.store.Version(ctx, projectID, st.LockedVersion)
		if err != nil {
			return nil, err
		}
		st.Draft = changedParts(latest.Document, data)
	}

	unitOK := func(key string) bool {
		for _, u := range st.Units {
			if u.Key == key {
				return u.Status == domain.UnitApproved && !u.Stale
			}
		}
		return false
	}
	nothingNew := st.LockedVersion > 0 && len(st.Draft) == 0
	st.Criteria = []Criterion{
		{"Quy tắc đủ rõ để các bước sau không tự tạo phong cách khác", allApprovedFresh,
			fmt.Sprintf("%d/%d phần đã duyệt và không cần làm lại", st.ApprovedCount, len(domain.ConceptUnits))},
		{"Các khái niệm quan trọng có cách biểu diễn phù hợp", unitOK(domain.UnitRepresentationRules) && checksOK[domain.CheckTerms],
			"phần 4 đã duyệt; máy kiểm: mọi thuật ngữ có cách vẽ"},
		{"Không có quy tắc hình dễ gây hiểu lầm khoa học", !openIn["accuracy"] && checksOK[domain.CheckAccuracy],
			"không còn vấn đề «Độ chính xác khoa học»; giả thuyết và ngộ nhận có dòng cảnh báo"},
		{"Phong cách khả thi với nguồn lực sản xuất", unitOK(domain.UnitFeasibility) && !openIn["feasibility"],
			"phần 8 đã duyệt, không còn vấn đề «Khả thi»"},
		{"Được người phụ trách sáng tạo duyệt", nothingNew, "bấm Chốt ngôn ngữ hình ảnh"},
	}

	st.Blockers = []string{}
	if !allApprovedFresh {
		st.Blockers = append(st.Blockers, "Duyệt hết 8 phần, không phần nào cần làm lại.")
	}
	if st.Review.OpenCount > 0 {
		st.Blockers = append(st.Blockers, fmt.Sprintf("Còn %d vấn đề rà soát cần xử lý.", st.Review.OpenCount))
	}
	for _, c := range st.Checks {
		if !c.OK && !c.Accepted {
			st.Blockers = append(st.Blockers, "Máy kiểm chưa đạt: "+c.Label+".")
		}
	}
	if data.fontsErr != nil {
		st.Blockers = append(st.Blockers, "Không kiểm được font của phần 5: "+data.fontsErr.Error())
		st.FontsError = data.fontsErr.Error()
	}
	if nothingNew {
		st.Blockers = append(st.Blockers, "Không có thay đổi so với v"+fmt.Sprint(st.LockedVersion)+".")
	}
	st.CanLock = len(st.Blockers) == 0

	if st.Model, err = d.store.ConceptModel(ctx, projectID); err != nil {
		return nil, err
	}
	if st.DefaultModel, err = d.models.StepDefaultModel(ctx, ConceptStep); err != nil {
		return nil, err
	}
	if st.Models, err = d.models.Models(ctx); err != nil {
		return nil, err
	}
	return st, nil
}

// partSection is the JSON of part key as a locked document holds it.
func partSection(doc map[string]json.RawMessage, key string) []byte {
	switch key {
	case domain.UnitTypography:
		var rep map[string]json.RawMessage
		_ = json.Unmarshal(doc["representation_rules"], &rep)
		return rep["typography"]
	case domain.UnitRepresentationRules:
		var rep map[string]json.RawMessage
		_ = json.Unmarshal(doc["representation_rules"], &rep)
		delete(rep, "typography")
		b, _ := json.Marshal(rep)
		return b
	case domain.UnitDoNotUse:
		b, _ := json.Marshal(map[string]json.RawMessage{"items": doc["do_not_use"]})
		return b
	}
	return doc[key]
}

// changedParts names the parts and inputs that differ from the locked document.
func changedParts(document []byte, data *deskData) []string {
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(document, &doc); err != nil {
		return []string{"Không đọc được bản chốt trước"}
	}
	var out []string
	for i, spec := range domain.ConceptUnits {
		u, ok := data.units[spec.Key]
		if !ok {
			continue
		}
		cur := u.Content
		if spec.Key == domain.UnitFeasibility && data.parts.Feasibility != nil {
			cur, _ = json.Marshal(data.parts.Feasibility)
		}
		if !sameJSON(cur, partSection(doc, spec.Key)) {
			out = append(out, fmt.Sprintf("Phần %d · %s", i+1, spec.Title))
		}
	}
	var sv int
	_ = json.Unmarshal(doc["script_version"], &sv)
	if sv != data.in.ScriptVersion {
		out = append(out, fmt.Sprintf("Kịch bản chốt v%d → v%d", sv, data.in.ScriptVersion))
	}
	refs, _ := json.Marshal(domain.DocumentReferences(data.in.Refs))
	if !sameJSON(refs, doc["reference_styles"]) {
		out = append(out, "Mô tả tham chiếu")
	}
	return out
}

func sameJSON(a, b []byte) bool {
	var x, y any
	if json.Unmarshal(a, &x) != nil || json.Unmarshal(b, &y) != nil {
		return false
	}
	ax, _ := json.Marshal(x)
	by, _ := json.Marshal(y)
	return string(ax) == string(by)
}

// --- the Creator's actions -------------------------------------------------------

func (d *ConceptDesk) unitOf(data *deskData, projectID, key string) (domain.ReviewUnit, error) {
	if _, ok := domain.UnitSpecOf(key); !ok {
		return domain.ReviewUnit{}, domain.ErrUnknownUnit
	}
	if d.running.has(projectID, key) {
		return domain.ReviewUnit{}, domain.ErrUnitRunning
	}
	u, ok := data.units[key]
	if !ok || len(u.Content) == 0 {
		return domain.ReviewUnit{}, domain.ErrUnitNotWritten
	}
	return u, nil
}

// Edit writes the Creator's fields into a part; they become the Creator's,
// and the part goes back to review. A part not written yet starts empty.
func (d *ConceptDesk) Edit(ctx context.Context, projectID, key string, fields map[string]json.RawMessage) (*ConceptState, error) {
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if _, ok := domain.UnitSpecOf(key); !ok {
		return nil, domain.ErrUnknownUnit
	}
	if d.running.has(projectID, key) {
		return nil, domain.ErrUnitRunning
	}
	if len(fields) == 0 {
		return nil, fmt.Errorf("%w: không có trường nào", domain.ErrInvalidEdit)
	}
	known := domain.PartFields(key)
	names := make([]string, 0, len(fields))
	for f := range fields {
		if !containsStr(known, f) {
			return nil, fmt.Errorf("%w: phần không có trường %q", domain.ErrInvalidEdit, f)
		}
		names = append(names, f)
	}
	u, ok := data.units[key]
	if !ok || len(u.Content) == 0 {
		_, empty, _ := domain.DecodePart(key, []byte("{}"))
		u = domain.ReviewUnit{ProjectID: projectID, Kind: domain.UnitKindConcept, Key: key, Content: empty,
			CreatorFields: []string{}, InputHashes: data.hashesFor(key)}
	}
	over, err := json.Marshal(fields)
	if err != nil {
		return nil, err
	}
	merged, err := domain.OverlayFields(u.Content, over, names)
	if err != nil {
		return nil, err
	}
	_, canon, err := domain.DecodePart(key, merged)
	if err != nil {
		return nil, err
	}
	u.Content, u.Status = canon, domain.UnitPending
	u.CreatorFields = u.AddCreatorFields(sortedNames(names)...)
	if err := d.store.SaveUnit(ctx, u); err != nil {
		return nil, err
	}
	return d.State(ctx, projectID)
}

// Approve approves a written part that passes its checks. Approving a part
// whose inputs changed confirms it still fits them.
func (d *ConceptDesk) Approve(ctx context.Context, projectID, key string) (*ConceptState, error) {
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	u, err := d.unitOf(data, projectID, key)
	if err != nil {
		return nil, err
	}
	if p := data.unitProblems(key); len(p) > 0 {
		return nil, fmt.Errorf("%w: %s", domain.ErrUnitInvalid, strings.Join(p, "; "))
	}
	u.Status, u.InputHashes = domain.UnitApproved, data.hashesFor(key)
	if err := d.store.SaveUnit(ctx, u); err != nil {
		return nil, err
	}
	return d.State(ctx, projectID)
}

// Unapprove puts an approved part back to review.
func (d *ConceptDesk) Unapprove(ctx context.Context, projectID, key string) (*ConceptState, error) {
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	u, err := d.unitOf(data, projectID, key)
	if err != nil {
		return nil, err
	}
	u.Status = domain.UnitPending
	if err := d.store.SaveUnit(ctx, u); err != nil {
		return nil, err
	}
	return d.State(ctx, projectID)
}

// ApproveAll approves every written part still pending that passes its checks
// and whose inputs did not change; the others stay as they are.
func (d *ConceptDesk) ApproveAll(ctx context.Context, projectID string) (*ConceptState, error) {
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	for _, spec := range domain.ConceptUnits {
		u, err := d.unitOf(data, projectID, spec.Key)
		if err != nil || u.Status == domain.UnitApproved || len(data.unitProblems(spec.Key)) > 0 || len(data.staleSources(spec.Key)) > 0 {
			continue
		}
		u.Status = domain.UnitApproved
		if err := d.store.SaveUnit(ctx, u); err != nil {
			return nil, err
		}
	}
	return d.State(ctx, projectID)
}

// SetRefs replaces the descriptions the video picked.
func (d *ConceptDesk) SetRefs(ctx context.Context, projectID string, refIDs []string) (*ConceptState, error) {
	if strings.TrimSpace(projectID) == "" {
		return nil, ErrNoProjectID
	}
	seen := map[string]bool{}
	ids := []string{}
	for _, id := range refIDs {
		if id = strings.TrimSpace(id); id != "" && !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	if err := d.store.SetPicks(ctx, projectID, ids); err != nil {
		return nil, err
	}
	return d.State(ctx, projectID)
}

// SetModel saves the model the video's step 8 runs on; "" is the step default.
func (d *ConceptDesk) SetModel(ctx context.Context, projectID, model string) (*ConceptState, error) {
	if strings.TrimSpace(projectID) == "" {
		return nil, ErrNoProjectID
	}
	model = strings.TrimSpace(model)
	if model != "" {
		models, err := d.models.Models(ctx)
		if err != nil {
			return nil, err
		}
		found := false
		for _, m := range models {
			found = found || m.ID == model
		}
		if !found {
			return nil, ErrUnknownModel
		}
	}
	if err := d.store.SetConceptModel(ctx, projectID, model); err != nil {
		return nil, err
	}
	return d.State(ctx, projectID)
}

// Lock freezes the parts into a new locked version.
func (d *ConceptDesk) Lock(ctx context.Context, projectID, note string) (*ConceptState, error) {
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
	doc, err := domain.BuildDocument(projectID, st.NextVersion, data.in.ScriptVersion, lockedAt, data.parts, data.in.Refs, data.review)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}
	if err := d.store.InsertVersion(ctx, VersionRecord{ProjectID: projectID, Version: st.NextVersion,
		ScriptVersion: data.in.ScriptVersion, Document: raw, Note: note, LockedAt: lockedAt}); err != nil {
		return nil, err
	}
	return d.State(ctx, projectID)
}

// Version returns a locked version with its document.
func (d *ConceptDesk) Version(ctx context.Context, projectID string, version int) (VersionRecord, error) {
	if strings.TrimSpace(projectID) == "" {
		return VersionRecord{}, ErrNoProjectID
	}
	return d.store.Version(ctx, projectID, version)
}

// LatestDocument returns the latest locked document for a later step. No step
// after 8 reads it yet, so only the whole document (step "") is served.
func (d *ConceptDesk) LatestDocument(ctx context.Context, projectID, step string) (json.RawMessage, error) {
	if strings.TrimSpace(step) != "" {
		return nil, domain.ErrUnknownStep
	}
	versions, err := d.store.Versions(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if len(versions) == 0 {
		return nil, domain.ErrNoVersion
	}
	v, err := d.store.Version(ctx, projectID, versions[len(versions)-1].Version)
	if err != nil {
		return nil, err
	}
	return v.Document, nil
}

func containsStr(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func sortedNames(names []string) []string {
	out := append([]string{}, names...)
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}
