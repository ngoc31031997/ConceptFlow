package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"visual/internal/domain"
)

// SequenceStep is the step of the prompt library the prompts of step 10 belong to.
const SequenceStep = "visual_sequence"

// motifsRunKey marks the motifs as being written by the AI.
const motifsRunKey = "motifs"

// aiReady fails with ErrLLMNotConfigured when the desk has no AI wired in.
func (d *SequenceDesk) aiReady() error {
	if d.llm == nil || d.prompts == nil || d.models == nil {
		return ErrLLMNotConfigured
	}
	return nil
}

// modelFor is the model the video's step 10 runs on.
func (d *SequenceDesk) modelFor(ctx context.Context, projectID string) (string, error) {
	m, err := d.store.SequenceModel(ctx, projectID)
	if err != nil || m != "" {
		return m, err
	}
	return d.models.StepDefaultModel(ctx, SequenceStep)
}

// SetModel saves the model the video's step 10 runs on; "" is the step default.
func (d *SequenceDesk) SetModel(ctx context.Context, projectID, model string) (*SequenceState, error) {
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
	if err := d.store.SetSequenceModel(ctx, projectID, model); err != nil {
		return nil, err
	}
	return d.State(ctx, projectID)
}

// ask runs one turn of role on the template's variables and returns the JSON
// object of the reply.
func (d *SequenceDesk) ask(ctx context.Context, projectID, role string, vars map[string]string) (json.RawMessage, error) {
	model, err := d.modelFor(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return askJSON(ctx, d.llm, d.prompts, askCall{ProjectID: projectID, Role: role, Step: SequenceStep, Model: model, MaxTokens: d.maxTokens}, vars)
}

// motifsInUse are the motif ids some item names.
func (data *sequenceData) motifsInUse() map[string]bool {
	out := map[string]bool{}
	for _, s := range data.beats {
		for _, it := range s.Items {
			for _, m := range it.MotifRefs {
				out[m] = true
			}
		}
	}
	return out
}

// GenerateMotifs has the AI set the video's motifs from the locked plan and
// visual concept, in one model turn, so an AI outside the system can answer it.
// The motifs the Creator edited keep their id and fields; the motifs go back to
// review and take the hashes their inputs have now.
func (d *SequenceDesk) GenerateMotifs(ctx context.Context, projectID, note string) (*SequenceState, error) {
	if err := d.aiReady(); err != nil {
		return nil, err
	}
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if d.running.has(projectID, motifsRunKey) {
		return nil, domain.ErrUnitRunning
	}
	finish := d.running.start(projectID, motifsRunKey)
	reply, err := d.ask(ctx, projectID, domain.PromptRoleSequenceMotifs, motifsVars(data, note))
	finish()
	if err != nil {
		return nil, err
	}
	planUnits := map[string]bool{}
	for _, u := range data.in.Plan.Units {
		planUnits[u.UnitID] = true
	}
	merge, err := domain.ParseMotifsReply(reply, data.motifs, data.motifRow.CreatorFields, planUnits, data.motifsInUse())
	if err != nil {
		return nil, err
	}
	if err := stopForPreview(ctx, merge.Set); err != nil {
		return nil, err
	}
	data.motifRow.InputHashes = data.in.MotifHashes()
	if err := d.saveMotifs(ctx, data, merge.Set, domain.UnitPending, merge.Fields); err != nil {
		return nil, err
	}
	return d.State(ctx, projectID)
}

// motifsReady fails with ErrMotifsNotApproved unless the motifs are approved and
// still fit their inputs.
func (data *sequenceData) motifsReady() error {
	if len(data.motifs.Motifs) == 0 || data.motifRow.Status != domain.UnitApproved || len(data.motifStaleSources()) > 0 {
		return domain.ErrMotifsNotApproved
	}
	return nil
}

// writeUnit has the AI write the item of one plan unit, reading the beat's
// other items and the units on both sides, and stores it once the answer is in
// and fits. The fields the Creator wrote in that item stay and the item goes
// back to review. A beat's first item takes the hashes its inputs have now; a
// beat that already had items keeps its hashes, so a stale beat stays stale
// until it is confirmed or rewritten whole.
func (d *SequenceDesk) writeUnit(ctx context.Context, projectID string, data *sequenceData, beatKey, unitID, note string) error {
	b, ok := data.in.Beat(beatKey)
	if !ok {
		return domain.ErrUnknownPlanBeat
	}
	row := data.beatRow(projectID, beatKey)
	var mine, others []string
	for _, f := range row.CreatorFields {
		if strings.HasPrefix(f, unitID+".") {
			mine = append(mine, f)
		} else {
			others = append(others, f)
		}
	}
	reply, err := d.ask(ctx, projectID, domain.PromptRoleSequenceUnit, seqUnitVars(data, b, unitID, mine, note))
	if err != nil {
		return err
	}
	first := unitPosition(data.in, unitID) == 0
	fresh, err := domain.ParseUnitSequenceReply(reply, unitID, first, data.motifs.IDs())
	if err != nil {
		return err
	}
	seq := data.beats[beatKey]
	items, kept := domain.MergeSequenceItems([]domain.SeqItem{fresh}, seq, mine)
	if err := stopForPreview(ctx, items[0]); err != nil {
		return err
	}
	at := seq.Index(unitID)
	if len(seq.Items) == 0 {
		row.InputHashes = data.in.BeatHashes(beatKey, data.motifs)
	}
	if at >= 0 {
		row.RedoCount++
	}
	data.rows[beatKey] = row
	return d.saveBeat(ctx, data, projectID, beatKey, putItem(seq, at, items[0]), append(others, kept...))
}

// GenerateUnit has the AI write or rewrite the item of one unit, with the
// Creator's note. It runs one model turn, so an AI outside the system can
// answer it.
func (d *SequenceDesk) GenerateUnit(ctx context.Context, projectID, unitID, note string) (*SequenceState, error) {
	if err := d.aiReady(); err != nil {
		return nil, err
	}
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	beatKey, _, ok := data.in.LocateUnit(unitID)
	if !ok {
		return nil, domain.ErrUnknownPlanUnit
	}
	if err := data.motifsReady(); err != nil {
		return nil, err
	}
	if d.running.has(projectID, unitID) || d.running.has(projectID, beatKey) {
		return nil, domain.ErrUnitRunning
	}
	finish := d.running.start(projectID, unitID)
	err = d.writeUnit(ctx, projectID, data, beatKey, unitID, note)
	finish()
	if err != nil {
		return nil, err
	}
	return d.State(ctx, projectID)
}

// GenerateBeat has the AI rewrite every unit of one beat, one model turn per
// unit in plan order, each reading the items written before it. Once every unit
// is written the beat takes the hashes its inputs have now. A failure stops the
// run; the units written before it stay.
func (d *SequenceDesk) GenerateBeat(ctx context.Context, projectID, beatKey, note string) (*SequenceState, error) {
	if err := d.aiReady(); err != nil {
		return nil, err
	}
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	b, ok := data.in.Beat(beatKey)
	if !ok {
		return nil, domain.ErrUnknownPlanBeat
	}
	if err := data.motifsReady(); err != nil {
		return nil, err
	}
	if d.running.has(projectID, beatKey) {
		return nil, domain.ErrUnitRunning
	}
	for _, u := range b.Units {
		if d.running.has(projectID, u.UnitID) {
			return nil, domain.ErrUnitRunning
		}
	}
	finish := d.running.start(projectID, beatKey)
	defer finish()
	for _, u := range b.Units {
		if err := d.writeUnit(ctx, projectID, data, beatKey, u.UnitID, note); err != nil {
			return nil, fmt.Errorf("%s: %w", u.UnitID, err)
		}
	}
	row := data.rows[beatKey]
	row.InputHashes = data.in.BeatHashes(beatKey, data.motifs)
	data.rows[beatKey] = row
	if err := d.saveBeat(ctx, data, projectID, beatKey, data.beats[beatKey], row.CreatorFields); err != nil {
		return nil, err
	}
	return d.State(ctx, projectID)
}

// Generate has the AI write every unit that has no item yet, one model turn per
// unit in plan order, each reading the items written before it. A failure stops
// the run; the units written before it stay.
func (d *SequenceDesk) Generate(ctx context.Context, projectID, note string) (*SequenceState, error) {
	if err := d.aiReady(); err != nil {
		return nil, err
	}
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if err := data.motifsReady(); err != nil {
		return nil, err
	}
	var todo []domain.DocumentUnit
	var keys []string
	for _, u := range data.in.Plan.Units {
		if d.running.has(projectID, u.UnitID) || d.running.has(projectID, u.BeatID) {
			return nil, domain.ErrUnitRunning
		}
		if data.beats[u.BeatID].Index(u.UnitID) < 0 {
			todo, keys = append(todo, u), append(keys, u.UnitID)
		}
	}
	finish := d.running.start(projectID, keys...)
	defer finish()
	for _, u := range todo {
		if err := d.writeUnit(ctx, projectID, data, u.BeatID, u.UnitID, note); err != nil {
			return nil, fmt.Errorf("%s: %w", u.UnitID, err)
		}
		d.running.done(projectID, u.UnitID)
	}
	return d.State(ctx, projectID)
}

// Review has the AI review the whole sequence in one turn. A later review
// compares with the earlier findings: those the AI no longer reports are fixed,
// new ones are added.
func (d *SequenceDesk) Review(ctx context.Context, projectID, note string) (*SequenceState, error) {
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
	written := 0
	for _, s := range data.beats {
		written += len(s.Items)
	}
	if written == 0 {
		return nil, fmt.Errorf("%w: chưa có chuỗi nào để rà soát", domain.ErrUnitNotWritten)
	}
	unitIDs := map[string]bool{}
	for _, u := range data.in.Plan.Units {
		unitIDs[u.UnitID] = true
	}
	vars := sequenceSharedVars(data.in)
	vars["anchors"] = anchorsText(data.in.AnchorRefs)
	vars["plan"] = seqPlanText(data.in)
	vars["motifs"] = motifsText(data.motifs)
	vars["sequence"] = sequenceText(data)
	vars["checks"] = sequenceChecksText(data)
	vars["previous_findings"] = planPreviousFindings(data.review)
	vars["note"] = orNone(note)

	finish := d.running.start(projectID, domain.ReviewUnitKey)
	reply, err := d.ask(ctx, projectID, domain.PromptRoleSequenceReview, vars)
	finish()
	if err != nil {
		return nil, err
	}
	parsed, err := domain.ParseSequenceReviewReply(reply, data.review, unitIDs, data.motifs.IDs())
	if err != nil {
		return nil, err
	}
	merged := domain.MergeReview(data.review, parsed)
	if err := stopForPreview(ctx, merged); err != nil {
		return nil, err
	}
	data.review = merged
	if err := d.saveReview(ctx, data); err != nil {
		return nil, err
	}
	return d.State(ctx, projectID)
}
