package application

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"visual/internal/domain"
)

// assetsRunKey marks the asset list as being written by the AI.
const assetsRunKey = "assets"

// aiReady fails with ErrLLMNotConfigured when the desk has no AI wired in.
func (d *StoryboardDesk) aiReady() error {
	if d.llm == nil || d.prompts == nil || d.models == nil {
		return ErrLLMNotConfigured
	}
	return nil
}

// modelFor is the model the video's step 11 runs on.
func (d *StoryboardDesk) modelFor(ctx context.Context, projectID string) (string, error) {
	m, err := d.store.StoryboardModel(ctx, projectID)
	if err != nil || m != "" {
		return m, err
	}
	return d.models.StepDefaultModel(ctx, StoryboardStep)
}

// SetModel saves the model the video's step 11 runs on; "" is the step default.
func (d *StoryboardDesk) SetModel(ctx context.Context, projectID, model string) (*StoryboardScreen, error) {
	if strings.TrimSpace(projectID) == "" {
		return nil, ErrNoProjectID
	}
	if err := d.aiReady(); err != nil {
		return nil, err
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
	if err := d.store.SetStoryboardModel(ctx, projectID, model); err != nil {
		return nil, err
	}
	return d.State(ctx, projectID)
}

// ask runs one turn of role on the template's variables and returns the JSON
// object of the reply.
func (d *StoryboardDesk) ask(ctx context.Context, projectID, role string, vars map[string]string) (json.RawMessage, error) {
	model, err := d.modelFor(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return askJSON(ctx, d.llm, d.prompts, askCall{ProjectID: projectID, Role: role, Step: StoryboardStep, Model: model, MaxTokens: d.maxTokens}, vars)
}

// --- the asset list -------------------------------------------------------------

// GenerateAssets has the AI list the video's assets (the first phase: which
// assets, their kind, tier, base drawing and units, no description) from the
// locked concept, plan and sequence, in one model turn so an AI outside the
// system can answer it. An asset the AI keeps through same_as is revised and one
// it leaves out is deleted; the fields the Creator wrote stay. The reply is checked whole before
// rendering is asked to change anything; rendering's own refusal part-way leaves
// the assets already changed as they are.
func (d *StoryboardDesk) GenerateAssets(ctx context.Context, projectID, note string) (*StoryboardScreen, error) {
	if err := d.aiReady(); err != nil {
		return nil, err
	}
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if d.running.has(projectID, assetsRunKey) {
		return nil, domain.ErrUnitRunning
	}
	library, err := d.rendering.ApprovedLibrary(ctx)
	if err != nil {
		return nil, err
	}
	finish := d.running.start(projectID, assetsRunKey)
	reply, err := d.ask(ctx, projectID, domain.PromptRoleStoryboardAssets, assetsVars(data, library, note))
	finish()
	if err != nil {
		return nil, err
	}
	pc := domain.AssetsReplyContext{ConceptIDs: domain.ConceptIDs(data.in.Seq.Concept.RepresentationRules.RepresentationRules),
		PlanUnits: map[string]bool{}, Library: map[string]string{}, Current: map[string]bool{}, InUse: map[string]bool{}}
	for _, id := range data.in.UnitIDs() {
		pc.PlanUnits[id] = true
	}
	for _, l := range library {
		pc.Library[l.ID] = l.Kind
	}
	for _, a := range data.assets {
		pc.Current[a.AssetID] = true
	}
	for id, users := range data.assetUsers() {
		if len(users) > 0 {
			pc.InUse[id] = true
		}
	}
	drafts, err := domain.ParseAssetsReply(reply, pc)
	if err != nil {
		return nil, err
	}
	if err := d.applyAssetDrafts(ctx, data, projectID, drafts); err != nil {
		return nil, err
	}
	return d.State(ctx, projectID)
}

// applyAssetDrafts makes rendering's assets what the AI listed. An asset the AI
// revised keeps the fields the Creator wrote and its library settings; one whose
// fields would not change is not sent, so an approved drawing is not reopened.
func (d *StoryboardDesk) applyAssetDrafts(ctx context.Context, data *storyboardData, projectID string, drafts []domain.AssetDraft) error {
	kept := map[string]bool{}
	for _, dr := range drafts {
		if dr.SameAs == "" {
			continue
		}
		kept[dr.SameAs] = true
		cur, _ := data.findAsset(dr.SameAs)
		in := AssetInputOf(cur)
		creator := map[string]bool{}
		for _, f := range unitCreatorFields(data.listRow, dr.SameAs) {
			creator[f] = true
		}
		put := func(field string, set func()) {
			if !creator[field] {
				set()
			}
		}
		put("concept_id", func() { in.ConceptID = dr.ConceptID })
		put("name", func() { in.Name = dr.Name })
		put("kind", func() { in.Kind = dr.Kind })
		put("tier", func() { in.Tier = dr.Tier })
		put("base_illustration_id", func() { in.BaseIllustrationID = dr.BaseIllustrationID })
		put("units", func() { in.Units = dr.Units })
		if reflect.DeepEqual(in, AssetInputOf(cur)) {
			continue
		}
		if _, err := d.rendering.UpdateAsset(ctx, projectID, dr.SameAs, in); err != nil {
			return fmt.Errorf("%s: %w", dr.SameAs, err)
		}
	}
	for _, dr := range drafts {
		if dr.SameAs != "" {
			continue
		}
		in := AssetInput{ConceptID: dr.ConceptID, Name: dr.Name, Kind: dr.Kind, Tier: dr.Tier, MustNotImply: []string{}, PaletteTokens: []string{},
			Expressions: []string{}, BaseIllustrationID: dr.BaseIllustrationID, Units: dr.Units, ToLibrary: true}
		if _, err := d.rendering.CreateAsset(ctx, projectID, in); err != nil {
			return fmt.Errorf("%s: %w", dr.Name, err)
		}
	}
	creator := []string{}
	for _, a := range data.assets {
		if kept[a.AssetID] {
			creator = append(creator, fieldPaths(data.listRow, a.AssetID)...)
			continue
		}
		if err := d.rendering.DeleteAsset(ctx, projectID, a.AssetID); err != nil {
			return fmt.Errorf("%s: %w", a.AssetID, err)
		}
	}
	// Whatever the AI changed goes back to review, and the list is read anew from
	// rendering on the next screen.
	data.assets = nil
	return d.saveAssetList(ctx, data, []domain.AssetEntry{}, creator)
}

// --- the detail of an asset -------------------------------------------------------

// detailFields are the fields of an asset the detail writer writes.
var detailFields = []string{"portrayal", "must_not_imply", "palette_tokens", "expressions"}

// writeAssetDetail has the AI write the detail of one asset from its concept
// entry, the palette, the units it appears in and a summary of the other assets,
// then stores it at rendering once the answer is in and fits. The detail fields
// the Creator wrote stay; the asset leaves the approved list and, when it was
// drawn, goes back to planned at rendering.
func (d *StoryboardDesk) writeAssetDetail(ctx context.Context, projectID string, data *storyboardData, assetID, note string) error {
	cur, ok := data.findAsset(assetID)
	if !ok {
		return domain.ErrUnknownAsset
	}
	creatorFields := unitCreatorFields(data.listRow, assetID)
	reply, err := d.ask(ctx, projectID, domain.PromptRoleStoryboardAssetDetail, assetDetailVars(data, cur, creatorFields, note))
	if err != nil {
		return err
	}
	detail, err := domain.ParseAssetDetailReply(reply, domain.PaletteTokenSet(data.in.Seq.Concept.ColorSystem))
	if err != nil {
		return err
	}
	kept := map[string]bool{}
	for _, f := range creatorFields {
		kept[f] = containsStr(detailFields, f)
	}
	// A portrayal the Creator emptied is not text to keep.
	if strings.TrimSpace(cur.Portrayal) == "" {
		kept["portrayal"] = false
	}
	in := AssetInputOf(cur)
	if !kept["portrayal"] {
		in.Portrayal = detail.Portrayal
	}
	if !kept["must_not_imply"] {
		in.MustNotImply = detail.MustNotImply
	}
	if !kept["palette_tokens"] {
		in.PaletteTokens = detail.PaletteTokens
	}
	if !kept["expressions"] {
		in.Expressions = detail.Expressions
	}
	updated := cur
	if !reflect.DeepEqual(in, AssetInputOf(cur)) {
		if updated, err = d.rendering.UpdateAsset(ctx, projectID, assetID, in); err != nil {
			return fmt.Errorf("%s: %w", assetID, err)
		}
		for i := range data.assets {
			if data.assets[i].AssetID == assetID {
				data.assets[i] = updated
			}
		}
		return d.saveAssetList(ctx, data, data.listEntries(assetID, domain.UnitPending), data.listRow.CreatorFields)
	}
	return nil
}

// GenerateAssetDetail has the AI write the detail of one asset, with the
// Creator's note. It runs one model turn, so an AI outside the system can answer
// it.
func (d *StoryboardDesk) GenerateAssetDetail(ctx context.Context, projectID, assetID, note string) (*StoryboardScreen, error) {
	if err := d.aiReady(); err != nil {
		return nil, err
	}
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if _, ok := data.findAsset(assetID); !ok {
		return nil, domain.ErrUnknownAsset
	}
	if d.running.has(projectID, assetsRunKey) || d.running.has(projectID, assetID) {
		return nil, domain.ErrUnitRunning
	}
	finish := d.running.start(projectID, assetID)
	err = d.writeAssetDetail(ctx, projectID, data, assetID, note)
	finish()
	if err != nil {
		return nil, err
	}
	return d.State(ctx, projectID)
}

// GenerateAllAssetDetails has the AI write the detail of every asset that has
// none, one model turn per asset in list order, each reading the details written
// before it. A failure stops the run; the details already written stay.
func (d *StoryboardDesk) GenerateAllAssetDetails(ctx context.Context, projectID, note string) (*StoryboardScreen, error) {
	if err := d.aiReady(); err != nil {
		return nil, err
	}
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if d.running.has(projectID, assetsRunKey) {
		return nil, domain.ErrUnitRunning
	}
	var todo []string
	for _, a := range data.assets {
		if d.running.has(projectID, a.AssetID) {
			return nil, domain.ErrUnitRunning
		}
		if !a.HasDetail() {
			todo = append(todo, a.AssetID)
		}
	}
	finish := d.running.start(projectID, todo...)
	defer finish()
	for _, id := range todo {
		if err := d.writeAssetDetail(ctx, projectID, data, id, note); err != nil {
			return nil, fmt.Errorf("%s: %w", id, err)
		}
		d.running.done(projectID, id)
	}
	return d.State(ctx, projectID)
}

// fieldPaths are the Creator's field paths of id in row.
func fieldPaths(row domain.ReviewUnit, id string) []string {
	var out []string
	for _, f := range row.CreatorFields {
		if strings.HasPrefix(f, id+".") {
			out = append(out, f)
		}
	}
	return out
}

// --- the shots of a unit --------------------------------------------------------

// assetsReady fails with ErrAssetsMissing, naming what is not ready, unless the
// video has assets and every one is approved: shots are written only over
// drawings that exist.
func (data *storyboardData) assetsReady() error {
	if len(data.assets) == 0 {
		return fmt.Errorf("%w: chưa có tài sản nào", domain.ErrAssetsMissing)
	}
	var notReady []string
	for _, a := range data.assets {
		if a.Status != domain.AssetApproved {
			notReady = append(notReady, fmt.Sprintf("%s (%s)", a.AssetID, a.Status))
		}
	}
	if len(notReady) > 0 {
		return fmt.Errorf("%w: %s", domain.ErrAssetsMissing, strings.Join(notReady, ", "))
	}
	return nil
}

// previousShots are the shots of the plan unit before unitID; false for the
// first unit of the video.
func (data *storyboardData) previousShots(unitID string) ([]domain.Shot, bool) {
	at := unitPosition(data.in.Seq, unitID)
	if at <= 0 {
		return nil, false
	}
	return data.units[data.in.Seq.Plan.Units[at-1].UnitID].Shots, true
}

// writeUnit has the AI write the shots of one unit, reading the shots of the
// unit before it, stores them once the answer is in and fits, and draws their
// frames. The fields the Creator wrote in a shot stay and every shot goes back to
// review.
func (d *StoryboardDesk) writeUnit(ctx context.Context, projectID string, data *storyboardData, unitID, note string) error {
	ui, ok := data.in.Unit(unitID)
	if !ok {
		return domain.ErrUnknownPlanUnit
	}
	row := data.rows[unitID]
	prev, hasPrev := data.previousShots(unitID)
	vars, err := unitVars(data, unitID, prev, hasPrev, row.CreatorFields, note)
	if err != nil {
		return err
	}
	reply, err := d.ask(ctx, projectID, domain.PromptRoleStoryboardUnit, vars)
	if err != nil {
		return err
	}
	approved := map[string]bool{}
	for _, a := range data.assets {
		if a.Status == domain.AssetApproved {
			approved[a.AssetID] = true
		}
	}
	fresh, err := domain.ParseShotsReply(reply, domain.ShotsReplyContext{UnitRefs: ui.Unit.ScriptRefs, ApprovedAssets: approved, Typography: ui.Typography})
	if err != nil {
		return err
	}
	old := data.units[unitID]
	merged, kept, err := domain.MergeShots(fresh, old, row.CreatorFields, domain.NextShotSeq(data.units))
	if err != nil {
		return err
	}
	merged.Frames = map[string]domain.DrawnFrame{}
	for _, sh := range merged.Shots {
		if fr, ok := old.Frames[sh.ShotID]; ok {
			merged.Frames[sh.ShotID] = fr
		}
	}
	if len(old.Shots) > 0 {
		row.RedoCount++
		data.rows[unitID] = row
	}
	if err := d.saveUnit(ctx, data, projectID, unitID, merged, kept, true); err != nil {
		return err
	}
	ids := make([]string, 0, len(merged.Shots))
	for _, sh := range merged.Shots {
		ids = append(ids, sh.ShotID)
	}
	return d.drawFrames(ctx, data, projectID, unitID, ids...)
}

// GenerateUnit has the AI write or rewrite the shots of one unit, with the
// Creator's note. It runs one model turn, so an AI outside the system can answer
// it.
func (d *StoryboardDesk) GenerateUnit(ctx context.Context, projectID, unitID, note string) (*StoryboardScreen, error) {
	if err := d.aiReady(); err != nil {
		return nil, err
	}
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if err := data.planUnit(unitID); err != nil {
		return nil, err
	}
	if err := data.assetsReady(); err != nil {
		return nil, err
	}
	if d.running.has(projectID, unitID) {
		return nil, domain.ErrUnitRunning
	}
	finish := d.running.start(projectID, unitID)
	err = d.writeUnit(ctx, projectID, data, unitID, note)
	finish()
	if err != nil {
		return nil, err
	}
	return d.State(ctx, projectID)
}

// Generate has the AI write every unit that has no shot yet, one model turn per
// unit in sequence order, each reading the shots written before it. A failure
// stops the run; the units written before it stay.
func (d *StoryboardDesk) Generate(ctx context.Context, projectID, note string) (*StoryboardScreen, error) {
	if err := d.aiReady(); err != nil {
		return nil, err
	}
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if err := data.assetsReady(); err != nil {
		return nil, err
	}
	var todo []string
	for _, u := range data.in.Seq.Plan.Units {
		if d.running.has(projectID, u.UnitID) {
			return nil, domain.ErrUnitRunning
		}
		if len(data.units[u.UnitID].Shots) == 0 {
			todo = append(todo, u.UnitID)
		}
	}
	finish := d.running.start(projectID, todo...)
	defer finish()
	for _, unitID := range todo {
		if err := d.writeUnit(ctx, projectID, data, unitID, note); err != nil {
			return nil, fmt.Errorf("%s: %w", unitID, err)
		}
		d.running.done(projectID, unitID)
	}
	return d.State(ctx, projectID)
}

// Review has the AI review the whole storyboard in one turn. A later review
// compares with the earlier findings: those the AI no longer reports are fixed,
// new ones are added.
func (d *StoryboardDesk) Review(ctx context.Context, projectID, note string) (*StoryboardScreen, error) {
	if err := d.aiReady(); err != nil {
		return nil, err
	}
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if d.running.has(projectID, domain.ReviewUnitKey) {
		return nil, domain.ErrUnitRunning
	}
	shotIDs, unitIDs, assetIDs := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, u := range data.in.Seq.Plan.Units {
		unitIDs[u.UnitID] = true
		for _, sh := range data.units[u.UnitID].Shots {
			shotIDs[sh.ShotID] = true
		}
	}
	if len(shotIDs) == 0 {
		return nil, fmt.Errorf("%w: chưa có shot nào để rà soát", domain.ErrUnitNotWritten)
	}
	for _, a := range data.assets {
		assetIDs[a.AssetID] = true
	}
	st := data.checkState(projectID)
	in := data.in
	vars := map[string]string{
		"brief":             briefText(in.Seq.Script.Brief),
		"do_not_use":        doNotUseText(in.Seq.Concept.DoNotUse),
		"plan":              storyboardPlanText(in),
		"sequence":          storyboardSequenceText(in),
		"assets":            assetsLinesText(data.domainAssets()),
		"shots":             shotLines(data, st),
		"checks":            storyboardChecksText(domain.StoryboardChecks(in, st)),
		"previous_findings": planPreviousFindings(data.review),
		"note":              orNone(note),
	}
	finish := d.running.start(projectID, domain.ReviewUnitKey)
	reply, err := d.ask(ctx, projectID, domain.PromptRoleStoryboardReview, vars)
	finish()
	if err != nil {
		return nil, err
	}
	parsed, err := domain.ParseStoryboardReviewReply(reply, data.review, shotIDs, unitIDs, assetIDs)
	if err != nil {
		return nil, err
	}
	data.review = domain.MergeReview(data.review, parsed)
	if err := d.saveReview(ctx, data); err != nil {
		return nil, err
	}
	return d.State(ctx, projectID)
}
