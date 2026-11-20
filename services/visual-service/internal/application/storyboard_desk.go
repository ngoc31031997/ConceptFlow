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

// StoryboardDesk is step 11 "Storyboard": it reads the locked visual sequence,
// visual plan, visual concept and script, keeps the video's assets at rendering
// and the shots of every unit here, has frames drawn, reviews them and locks
// versions (kept at rendering).
type StoryboardDesk struct {
	authoring AuthoringInputPort
	store     StoryboardStorePort
	rendering StoryboardRenderingPort
	kits      ProjectKitPort
	now       func() time.Time
	running   *runningSet

	// The AI side; unset until WithAI, and then the AI routes fail with
	// ErrLLMNotConfigured.
	llm       LLMPort
	prompts   PromptLibraryPort
	models    ModelCatalogPort
	maxTokens int
}

// NewStoryboardDesk wires the desk without an AI.
func NewStoryboardDesk(authoring AuthoringInputPort, store StoryboardStorePort, rendering StoryboardRenderingPort, kits ProjectKitPort) *StoryboardDesk {
	return &StoryboardDesk{authoring: authoring, store: store, rendering: rendering, kits: kits, now: time.Now,
		running: &runningSet{m: map[string]map[string]bool{}}}
}

// WithAI gives the desk the model, the prompt library and the model catalog it
// writes and reviews with.
func (d *StoryboardDesk) WithAI(llm LLMPort, prompts PromptLibraryPort, models ModelCatalogPort, maxTokens int) *StoryboardDesk {
	d.llm, d.prompts, d.models, d.maxTokens = llm, prompts, models, maxTokens
	return d
}

// storyboardData is everything one request works on.
type storyboardData struct {
	in domain.StoryboardInput
	// assets are the video's assets at rendering, oldest first.
	assets []ProjectAsset
	// listRow holds the approval of the asset list; list its content.
	listRow domain.ReviewUnit
	list    domain.AssetList
	// rows and units are the stored shots by plan unit id.
	rows      map[string]domain.ReviewUnit
	units     map[string]domain.UnitShots
	review    domain.ReviewState
	reviewRow domain.ReviewUnit
	// refID and kit are the reference style the video picked and its drawing kit;
	// kitErr says why there is none.
	refID  string
	kit    json.RawMessage
	kitErr error
}

func (d *StoryboardDesk) load(ctx context.Context, projectID string) (*storyboardData, error) {
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
	seqs, err := d.store.SequenceVersions(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if len(seqs) == 0 {
		return nil, domain.ErrSequenceNotLocked
	}
	sq, err := d.store.SequenceVersion(ctx, projectID, seqs[len(seqs)-1].Version)
	if err != nil {
		return nil, err
	}
	in, err := domain.NewStoryboardInput(sv, cv.Version, cv.Document, pv.Version, pv.Document, sq.Version, sq.Document)
	if err != nil {
		return nil, err
	}
	data := &storyboardData{in: in, rows: map[string]domain.ReviewUnit{}, units: map[string]domain.UnitShots{}}
	if data.assets, err = d.rendering.ListAssets(ctx, projectID); err != nil {
		return nil, err
	}
	if data.listRow, err = d.singleRow(ctx, projectID, domain.UnitKindStoryboardAssets); err != nil {
		return nil, err
	}
	if data.list, err = domain.DecodeAssetList(data.listRow.Content); err != nil {
		return nil, fmt.Errorf("đọc danh sách tài sản đã lưu: %w", err)
	}
	rows, err := d.store.ListUnits(ctx, projectID, domain.UnitKindStoryboardUnit)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		u, err := domain.DecodeUnitShots(r.Content)
		if err != nil {
			return nil, fmt.Errorf("đọc shot đã lưu của unit %s: %w", r.Key, err)
		}
		data.rows[r.Key], data.units[r.Key] = r, u
	}
	if data.reviewRow, err = d.singleRow(ctx, projectID, domain.UnitKindStoryboardReview); err != nil {
		return nil, err
	}
	if data.review, err = domain.ParseReviewState(data.reviewRow.Content); err != nil {
		return nil, err
	}
	d.loadKit(ctx, projectID, data)
	return data, nil
}

// loadKit reads the kit of the style the video picked. A video without one can
// still be read and edited; only drawing needs it.
func (d *StoryboardDesk) loadKit(ctx context.Context, projectID string, data *storyboardData) {
	res, err := d.kits.KitOfProject(ctx, projectID)
	if err != nil {
		if errors.Is(err, ErrNoReferencePicked) || errors.Is(err, ErrReferenceKitMissing) {
			data.kitErr = fmt.Errorf("%w: %v", domain.ErrNoKit, err)
		} else {
			data.kitErr = err
		}
		return
	}
	raw, err := json.Marshal(res.Kit)
	if err != nil {
		data.kitErr = err
		return
	}
	data.refID, data.kit = res.RefID, raw
}

// needKit fails with the reason the video has no kit to draw with.
func (data *storyboardData) needKit() error { return data.kitErr }

// singleRow returns the one row of kind under ReviewUnitKey, or a new pending
// row with no hashes when there is none.
func (d *StoryboardDesk) singleRow(ctx context.Context, projectID, kind string) (domain.ReviewUnit, error) {
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

// domainAssets are the assets as the checks read them.
func (data *storyboardData) domainAssets() []domain.StoryboardAsset {
	out := make([]domain.StoryboardAsset, 0, len(data.assets))
	for _, a := range data.assets {
		out = append(out, a.StoryboardAsset)
	}
	return out
}

func (data *storyboardData) assetMap() map[string]domain.StoryboardAsset {
	out := make(map[string]domain.StoryboardAsset, len(data.assets))
	for _, a := range data.assets {
		out[a.AssetID] = a.StoryboardAsset
	}
	return out
}

func (data *storyboardData) findAsset(id string) (ProjectAsset, bool) {
	for _, a := range data.assets {
		if a.AssetID == id {
			return a, true
		}
	}
	return ProjectAsset{}, false
}

// checkState is the stored step as the machine checks and the lock read it.
func (data *storyboardData) checkState(projectID string) domain.StoryboardState {
	frames := map[string]domain.DrawnFrame{}
	for _, u := range data.units {
		for id, f := range u.Frames {
			frames[id] = f
		}
	}
	return domain.StoryboardState{ProjectID: projectID, Assets: data.domainAssets(), Units: data.units, Frames: frames,
		Kit: data.kit, Review: data.review}
}

// shotUnit returns the plan unit that holds shot id and its position in it.
func (data *storyboardData) shotUnit(shotID string) (string, int, bool) {
	for unitID, u := range data.units {
		if i := u.Index(shotID); i >= 0 {
			return unitID, i, true
		}
	}
	return "", -1, false
}

// assetUsers lists, by asset id, the shots that name the asset.
func (data *storyboardData) assetUsers() map[string][]string {
	out := map[string][]string{}
	for _, u := range data.in.UnitIDs() {
		for _, sh := range data.units[u].Shots {
			for _, id := range sh.AssetRefs {
				out[id] = append(out[id], sh.ShotID)
			}
		}
	}
	return out
}

// --- the screen ---------------------------------------------------------------

// FrameView is the last drawing of a shot's frame as the screen shows it.
type FrameView struct {
	Drawn bool `json:"drawn"`
	// Stale says the frame was drawn from another spec than the shot has now.
	Stale bool `json:"stale"`
	// Path is the image relative to the video's shared folder; the image itself
	// is read through GET …/frames/{shot}.
	Path string `json:"path"`
	// Error is why the last drawing failed; empty when it did not.
	Error string `json:"error"`
}

// ShotView is one shot with its checks and frame.
type ShotView struct {
	domain.Shot
	UnitID        string               `json:"unit_id"`
	BeatID        string               `json:"beat_id"`
	CreatorFields []string             `json:"creator_fields"`
	Problems      []domain.ShotProblem `json:"problems"`
	// MissingAssets are the assets the shot names that are not approved or not
	// there.
	MissingAssets []string  `json:"missing_assets"`
	Frame         FrameView `json:"frame"`
}

// StoryboardUnitView is one plan unit with its shots.
type StoryboardUnitView struct {
	UnitID            string                `json:"unit_id"`
	Order             int                   `json:"order"`
	BeatID            string                `json:"beat_id"`
	ScriptRefs        []string              `json:"script_refs"`
	Lines             []domain.Line         `json:"lines"`
	PlannedS          float64               `json:"planned_s"`
	CommunicationGoal string                `json:"communication_goal"`
	VisualApproach    domain.VisualApproach `json:"visual_approach"`
	PlannedText       []string              `json:"planned_text"`
	PurposeInFlow     string                `json:"purpose_in_flow"`
	Written           bool                  `json:"written"`
	Approved          bool                  `json:"approved"`
	Stale             bool                  `json:"stale"`
	StaleSources      []string              `json:"stale_sources"`
	Shots             []ShotView            `json:"shots"`
	CreatorFields     []string              `json:"creator_fields"`
}

// StoryboardBeatView is one beat with its units.
type StoryboardBeatView struct {
	Key      string               `json:"key"`
	Number   int                  `json:"number"`
	Fn       string               `json:"fn"`
	Title    string               `json:"title"`
	Written  bool                 `json:"written"`
	Approved bool                 `json:"approved"`
	Units    []StoryboardUnitView `json:"units"`
}

// AssetView is one asset with its approval in the list and the shots using it.
type AssetView struct {
	ProjectAsset
	// ListStatus is whether the Creator approved the list with this asset in it.
	ListStatus    string   `json:"list_status"`
	CreatorFields []string `json:"creator_fields"`
	UsedBy        []string `json:"used_by"`
	// HasDetail says the asset has its detail (a portrayal) and can be drawn.
	HasDetail bool `json:"has_detail"`
}

// AssetsView is the asset tab.
type AssetsView struct {
	Approved bool        `json:"approved"`
	Items    []AssetView `json:"items"`
	// ApprovedCount is how many assets are approved (drawn and accepted).
	ApprovedCount int `json:"approved_count"`
	// WithDetail and WithoutDetail count the assets that have their detail and
	// those that wait for it.
	WithDetail    int `json:"with_detail"`
	WithoutDetail int `json:"without_detail"`
}

// KitView says whether the video can draw.
type KitView struct {
	RefID   string `json:"ref_id"`
	Missing bool   `json:"missing"`
	Message string `json:"message"`
}

// StoryboardInputSummary is the head card's line of inputs.
type StoryboardInputSummary struct {
	SequenceVersion int      `json:"sequence_version"`
	PlanVersion     int      `json:"plan_version"`
	ConceptVersion  int      `json:"concept_version"`
	ScriptVersion   int      `json:"script_version"`
	UnitsCount      int      `json:"units_count"`
	Outdated        []string `json:"outdated"`
}

// StoryboardScreen is the whole screen of step 11.
type StoryboardScreen struct {
	Input         StoryboardInputSummary `json:"input"`
	Kit           KitView                `json:"kit"`
	Assets        AssetsView             `json:"assets"`
	Beats         []StoryboardBeatView   `json:"beats"`
	ShotsApproved int                    `json:"shots_approved"`
	ShotsTotal    int                    `json:"shots_total"`
	Checks        []CheckView            `json:"checks"`
	Review        PlanReviewView         `json:"review"`
	Blockers      []string               `json:"blockers"`
	CanLock       bool                   `json:"can_lock"`
	LockedVersion int                    `json:"locked_version"`
	LockedNote    string                 `json:"locked_note"`
	LockedAt      *time.Time             `json:"locked_at"`
	NextVersion   int                    `json:"next_version"`
	// Draft names the assets and shots that differ from the locked version; it is
	// worked out only when nothing else keeps the storyboard from locking.
	Draft []string `json:"draft"`
	// PendingUnits are the units whose shots wait for review or are not written,
	// in plan order; EmptyUnits those with no shot yet.
	PendingUnits []string `json:"pending_units"`
	EmptyUnits   []string `json:"empty_units"`
	// Model is the model the video's step 11 runs on; "" is the step default.
	Model        string        `json:"model"`
	DefaultModel string        `json:"default_model"`
	Models       []ModelOption `json:"models"`
}

// State returns the screen.
func (d *StoryboardDesk) State(ctx context.Context, projectID string) (*StoryboardScreen, error) {
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return d.state(ctx, projectID, data)
}

func (d *StoryboardDesk) latestRecord(ctx context.Context, projectID string) (*StoryboardVersionRecord, error) {
	rec, err := d.rendering.LatestStoryboard(ctx, projectID)
	if errors.Is(err, domain.ErrNoVersion) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

func (d *StoryboardDesk) state(ctx context.Context, projectID string, data *storyboardData) (*StoryboardScreen, error) {
	in := data.in
	st := data.checkState(projectID)
	sc := &StoryboardScreen{
		Input: StoryboardInputSummary{SequenceVersion: in.SequenceVersion, PlanVersion: in.Seq.PlanVersion, ConceptVersion: in.Seq.ConceptVersion,
			ScriptVersion: in.Seq.Script.ScriptVersion, UnitsCount: len(in.Seq.Plan.Units), Outdated: in.Outdated},
		Beats: []StoryboardBeatView{}, Blockers: []string{}, Draft: []string{}, PendingUnits: []string{}, EmptyUnits: []string{},
	}
	if data.kitErr != nil {
		sc.Kit = KitView{Missing: true, Message: data.kitErr.Error()}
	} else {
		sc.Kit = KitView{RefID: data.refID}
	}
	sc.Assets = data.assetsView()
	assets := data.assetMap()
	for _, b := range in.Seq.Beats {
		bv := StoryboardBeatView{Key: b.Key, Number: b.Number, Fn: b.Fn, Title: b.Title, Units: []StoryboardUnitView{}}
		approved := 0
		for _, u := range b.Units {
			uv := data.unitView(u, st, assets)
			bv.Units = append(bv.Units, uv)
			sc.ShotsTotal += len(uv.Shots)
			for _, sh := range uv.Shots {
				if sh.Status == domain.UnitApproved {
					sc.ShotsApproved++
				}
			}
			switch {
			case !uv.Written:
				sc.EmptyUnits = append(sc.EmptyUnits, uv.UnitID)
				sc.PendingUnits = append(sc.PendingUnits, uv.UnitID)
			case !uv.Approved:
				sc.PendingUnits = append(sc.PendingUnits, uv.UnitID)
			default:
				approved++
			}
			bv.Written = bv.Written || uv.Written
		}
		bv.Approved = approved == len(b.Units)
		sc.Beats = append(sc.Beats, bv)
	}
	for _, c := range domain.StoryboardChecks(in, st) {
		cv := CheckView{MachineCheck: c}
		if dec, ok := data.review.CheckAccepted(c); ok && !c.OK {
			cv.Accepted, cv.Why = true, dec.Why
		}
		sc.Checks = append(sc.Checks, cv)
	}
	sc.Review = PlanReviewView{Reviewed: data.review.Rounds > 0, Rounds: data.review.Rounds, Notes: data.review.Notes,
		OpenCount: data.review.OpenCount(), Findings: []PlanFindingView{}}
	for _, f := range data.review.Findings {
		sc.Review.Findings = append(sc.Review.Findings, PlanFindingView{Finding: f, GroupLabel: domain.FindingGroupLabels[f.Group]})
	}

	latest, err := d.latestRecord(ctx, projectID)
	if err != nil {
		return nil, err
	}
	sc.NextVersion = 1
	var latestDoc domain.StoryboardDocument
	if latest != nil {
		sc.LockedVersion, sc.LockedNote, sc.NextVersion = latest.Version, latest.Note, latest.Version+1
		at := latest.LockedAt
		sc.LockedAt = &at
		if err := json.Unmarshal(latest.Document, &latestDoc); err != nil {
			return nil, fmt.Errorf("đọc bản chốt v%d: %w", latest.Version, err)
		}
	}
	sc.Blockers = domain.StoryboardLockBlockers(in, st)
	if len(sc.Blockers) == 0 {
		doc, err := domain.BuildStoryboardDocument(projectID, sc.NextVersion, d.now().UTC(), in, st)
		if err != nil {
			return nil, err
		}
		if latest != nil {
			if domain.SameStoryboardContent(latestDoc, doc) {
				sc.Blockers = append(sc.Blockers, fmt.Sprintf("Không có thay đổi so với v%d.", latest.Version))
			} else {
				sc.Draft = changedStoryboard(latestDoc, doc)
			}
		}
	}
	sc.CanLock = len(sc.Blockers) == 0

	sc.Models = []ModelOption{}
	if d.models != nil {
		var err error
		if sc.Model, err = d.store.StoryboardModel(ctx, projectID); err != nil {
			return nil, err
		}
		if sc.DefaultModel, err = d.models.StepDefaultModel(ctx, StoryboardStep); err != nil {
			return nil, err
		}
		if sc.Models, err = d.models.Models(ctx); err != nil {
			return nil, err
		}
	}
	return sc, nil
}

func (data *storyboardData) assetsView() AssetsView {
	av := AssetsView{Items: []AssetView{}}
	listStatus := map[string]string{}
	for _, e := range data.list.Assets {
		listStatus[e.AssetID] = e.Status
	}
	users := data.assetUsers()
	approvedList := len(data.assets) > 0
	for _, a := range data.assets {
		status := listStatus[a.AssetID]
		if status == "" {
			status = domain.UnitPending
		}
		if status != domain.UnitApproved {
			approvedList = false
		}
		if a.Status == domain.AssetApproved {
			av.ApprovedCount++
		}
		used := users[a.AssetID]
		if used == nil {
			used = []string{}
		}
		if a.HasDetail() {
			av.WithDetail++
		} else {
			av.WithoutDetail++
		}
		a.Units = nonNilList(a.Units)
		av.Items = append(av.Items, AssetView{ProjectAsset: a, ListStatus: status,
			CreatorFields: unitCreatorFields(data.listRow, a.AssetID), UsedBy: used, HasDetail: a.HasDetail()})
	}
	av.Approved = approvedList
	return av
}

func (data *storyboardData) unitView(u domain.DocumentUnit, st domain.StoryboardState, assets map[string]domain.StoryboardAsset) StoryboardUnitView {
	ui, _ := data.in.Unit(u.UnitID)
	stored := data.units[u.UnitID]
	uv := StoryboardUnitView{UnitID: u.UnitID, Order: u.Order, BeatID: u.BeatID, ScriptRefs: u.ScriptRefs, Lines: ui.Lines,
		PlannedS: u.Duration.PlannedS, CommunicationGoal: u.CommunicationGoal, VisualApproach: u.VisualApproach,
		PlannedText: nonNilList(u.OnScreenText), PurposeInFlow: ui.Item.PurposeInFlow, StaleSources: []string{}, Shots: []ShotView{},
		CreatorFields: []string{}}
	if uv.Lines == nil {
		uv.Lines = []domain.Line{}
	}
	if len(stored.Shots) == 0 {
		return uv
	}
	uv.Written = true
	uv.Approved = domain.UnitApproval(stored) == domain.UnitApproved
	row := data.rows[u.UnitID]
	for _, s := range data.in.UnitStaleSources(u.UnitID, row.InputHashes, stored.Shots, assets) {
		uv.StaleSources = append(uv.StaleSources, domain.StoryboardSourceLabels[s])
	}
	uv.Stale = len(uv.StaleSources) > 0
	uv.CreatorFields = row.CreatorFields
	if uv.CreatorFields == nil {
		uv.CreatorFields = []string{}
	}
	for _, sh := range stored.Shots {
		ctx := st.ShotContextFor(data.in, u.UnitID, sh)
		sv := ShotView{Shot: sh, UnitID: u.UnitID, BeatID: u.BeatID, CreatorFields: shotCreatorFields(row, sh.ShotID),
			Problems: domain.CheckShot(sh, ctx), MissingAssets: []string{}}
		if sv.Problems == nil {
			sv.Problems = []domain.ShotProblem{}
		}
		for _, id := range sh.AssetRefs {
			if a, ok := assets[id]; !ok || a.Status != domain.AssetApproved {
				sv.MissingAssets = append(sv.MissingAssets, id)
			}
		}
		fr := stored.Frames[sh.ShotID]
		sv.Frame = FrameView{Drawn: fr.Drawn(), Path: fr.FramePath, Error: fr.Error, Stale: ctx.SpecHash != "" && fr.Stale(ctx.SpecHash)}
		uv.Shots = append(uv.Shots, sv)
	}
	return uv
}

// shotCreatorFields lists the fields the Creator wrote in shot id.
func shotCreatorFields(row domain.ReviewUnit, shotID string) []string {
	return unitCreatorFields(row, shotID)
}

// changedStoryboard names the assets and shots that differ from the locked
// document.
func changedStoryboard(old, now domain.StoryboardDocument) []string {
	out := []string{}
	if old.SourceVersions.VisualSequence != now.SourceVersions.VisualSequence {
		out = append(out, fmt.Sprintf("Chuỗi hình v%d → v%d", old.SourceVersions.VisualSequence, now.SourceVersions.VisualSequence))
	}
	assets := map[string]domain.DocumentAsset{}
	for _, a := range old.Assets {
		assets[a.AssetID] = a
	}
	for _, a := range now.Assets {
		out = appendChange(out, a.AssetID, a, assets)
	}
	for id := range assets {
		out = append(out, id+" · bỏ")
	}
	shots := map[string]domain.DocumentShot{}
	for _, s := range old.Shots {
		s.FramePath = ""
		shots[s.ShotID] = s
	}
	for _, s := range now.Shots {
		s.FramePath = ""
		out = appendChange(out, s.ShotID, s, shots)
	}
	for id := range shots {
		out = append(out, id+" · bỏ")
	}
	return out
}

// --- locking ---------------------------------------------------------------------

// Lock freezes the storyboard into a new locked version kept at rendering. A
// sequence written from older steps is domain.ErrSequenceOutdated.
func (d *StoryboardDesk) Lock(ctx context.Context, projectID, note string) (*StoryboardScreen, error) {
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if len(data.in.Outdated) > 0 {
		return nil, fmt.Errorf("%w: %s", domain.ErrSequenceOutdated, strings.Join(data.in.Outdated, "; "))
	}
	st := data.checkState(projectID)
	if blockers := domain.StoryboardLockBlockers(data.in, st); len(blockers) > 0 {
		return nil, fmt.Errorf("%w: %s", domain.ErrLockNotReady, strings.Join(blockers, "; "))
	}
	latest, err := d.latestRecord(ctx, projectID)
	if err != nil {
		return nil, err
	}
	next, changes := 1, 0
	var old domain.StoryboardDocument
	if latest != nil {
		next = latest.Version + 1
		if err := json.Unmarshal(latest.Document, &old); err != nil {
			return nil, fmt.Errorf("đọc bản chốt v%d: %w", latest.Version, err)
		}
	}
	doc, err := domain.BuildStoryboardDocument(projectID, next, d.now().UTC(), data.in, st)
	if err != nil {
		return nil, err
	}
	if latest != nil {
		if domain.SameStoryboardContent(old, doc) {
			return nil, domain.ErrNothingToLock
		}
		changes = len(changedStoryboard(old, doc))
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}
	rec, err := d.rendering.LockStoryboard(ctx, projectID, raw, data.in.SequenceVersion, lockNote(note, next-1, changes))
	if err != nil {
		return nil, err
	}
	if rec.Version != next {
		return nil, fmt.Errorf("%w: rendering ghi bản chốt v%d nhưng visual-service tính v%d", ErrRenderingFailed, rec.Version, next)
	}
	return d.State(ctx, projectID)
}

// Version returns a locked version with its document.
func (d *StoryboardDesk) Version(ctx context.Context, projectID string, version int) (StoryboardVersionRecord, error) {
	if strings.TrimSpace(projectID) == "" {
		return StoryboardVersionRecord{}, ErrNoProjectID
	}
	return d.rendering.StoryboardVersion(ctx, projectID, version)
}

// LatestDocument returns the latest locked storyboard document for a later
// step, or domain.ErrNoVersion.
func (d *StoryboardDesk) LatestDocument(ctx context.Context, projectID string) (json.RawMessage, error) {
	if strings.TrimSpace(projectID) == "" {
		return nil, ErrNoProjectID
	}
	rec, err := d.rendering.LatestStoryboard(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return rec.Document, nil
}

// FrameImage returns the PNG of a shot's last drawn frame.
func (d *StoryboardDesk) FrameImage(ctx context.Context, projectID, shotID string) ([]byte, error) {
	if strings.TrimSpace(projectID) == "" {
		return nil, ErrNoProjectID
	}
	return d.rendering.FrameImage(ctx, projectID, shotID)
}
