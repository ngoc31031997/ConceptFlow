package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"visual/internal/domain"
)

// beatRow returns the stored row of beat key, or a new one carrying the hashes
// the beat has now.
func (data *sequenceData) beatRow(projectID, key string) domain.ReviewUnit {
	if row, ok := data.rows[key]; ok {
		return row
	}
	return domain.ReviewUnit{ProjectID: projectID, Kind: domain.UnitKindSequence, Key: key, Status: domain.UnitPending,
		CreatorFields: []string{}, InputHashes: data.in.BeatHashes(key, data.motifs)}
}

// saveBeat stores the items of beat key in plan order; an item whose unit the
// beat no longer has is dropped. The row reads approved when every unit of the
// beat has an approved item.
func (d *SequenceDesk) saveBeat(ctx context.Context, data *sequenceData, projectID, key string, seq domain.BeatSequence, creatorFields []string) error {
	beat, ok := data.in.Beat(key)
	if !ok {
		return domain.ErrUnknownPlanBeat
	}
	ordered := domain.BeatSequence{Items: []domain.SeqItem{}}
	approved := 0
	for _, u := range beat.Units {
		if i := seq.Index(u.UnitID); i >= 0 {
			ordered.Items = append(ordered.Items, seq.Items[i])
			if seq.Items[i].Status == domain.UnitApproved {
				approved++
			}
		}
	}
	raw, err := ordered.Encode()
	if err != nil {
		return err
	}
	row := data.beatRow(projectID, key)
	row.Content, row.CreatorFields, row.Status = raw, creatorFields, domain.UnitPending
	if approved == len(beat.Units) {
		row.Status = domain.UnitApproved
	}
	if err := d.store.SaveUnit(ctx, row); err != nil {
		return err
	}
	data.rows[key], data.beats[key] = row, ordered
	return nil
}

// locateItem returns the beat of plan unit id, its item (an empty pending one
// when the unit has none yet) and the item's position in the beat (-1 when new).
func (data *sequenceData) locateItem(unitID string) (string, domain.SeqItem, int, error) {
	beat, _, ok := data.in.LocateUnit(unitID)
	if !ok {
		return "", domain.SeqItem{}, -1, domain.ErrUnknownPlanUnit
	}
	seq := data.beats[beat]
	if i := seq.Index(unitID); i >= 0 {
		return beat, seq.Items[i], i, nil
	}
	return beat, domain.SeqItem{UnitID: unitID, MotifRefs: []string{}, Status: domain.UnitPending}, -1, nil
}

// putItem returns the beat's items with it placed at position at, or added when
// at is -1.
func putItem(seq domain.BeatSequence, at int, it domain.SeqItem) domain.BeatSequence {
	items := append([]domain.SeqItem{}, seq.Items...)
	if at < 0 {
		items = append(items, it)
	} else {
		items[at] = it
	}
	return domain.BeatSequence{Items: items}
}

// Edit writes the Creator's fields into the item of a plan unit, starting from
// an empty item when the unit has none; the fields become the Creator's and the
// item goes back to review.
func (d *SequenceDesk) Edit(ctx context.Context, projectID, unitID string, fields map[string]json.RawMessage) (*SequenceState, error) {
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	beat, cur, at, err := data.locateItem(unitID)
	if err != nil {
		return nil, err
	}
	names, err := editableNames(fields, domain.SequenceEditableFields, "unit")
	if err != nil {
		return nil, err
	}
	merged, err := overlay(cur, fields, names)
	if err != nil {
		return nil, err
	}
	it, err := domain.DecodeSeqItem(merged)
	if err != nil {
		return nil, err
	}
	it.UnitID, it.Status = unitID, domain.UnitPending
	row := data.beatRow(projectID, beat)
	if err := d.saveBeat(ctx, data, projectID, beat, putItem(data.beats[beat], at, it), withCreatorFields(row, unitID, names...)); err != nil {
		return nil, err
	}
	return d.State(ctx, projectID)
}

// editableNames checks every field is one of allowed and returns their names
// sorted.
func editableNames(fields map[string]json.RawMessage, allowed []string, what string) ([]string, error) {
	if len(fields) == 0 {
		return nil, fmt.Errorf("%w: không có trường nào", domain.ErrInvalidEdit)
	}
	names := make([]string, 0, len(fields))
	for f := range fields {
		if !containsStr(allowed, f) {
			return nil, fmt.Errorf("%w: %s không có trường %q để sửa", domain.ErrInvalidEdit, what, f)
		}
		names = append(names, f)
	}
	return sortedNames(names), nil
}

// overlay returns cur's JSON with the given fields replaced.
func overlay(cur any, fields map[string]json.RawMessage, names []string) ([]byte, error) {
	base, err := json.Marshal(cur)
	if err != nil {
		return nil, err
	}
	over, err := json.Marshal(fields)
	if err != nil {
		return nil, err
	}
	return domain.OverlayFields(base, over, names)
}

// setItemStatus approves or reopens the item of a plan unit. Approving needs a
// beat whose inputs did not change and an item that passes its checks.
func (d *SequenceDesk) setItemStatus(ctx context.Context, projectID, unitID, status string) (*SequenceState, error) {
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	beat, it, at, err := data.locateItem(unitID)
	if err != nil {
		return nil, err
	}
	if at < 0 {
		return nil, domain.ErrUnitNotWritten
	}
	if status == domain.UnitApproved {
		if len(data.beatStaleSources(beat)) > 0 {
			return nil, domain.ErrBeatStale
		}
		if p := data.itemProblems(unitID, it); len(p) > 0 {
			return nil, fmt.Errorf("%w: %s", domain.ErrUnitInvalid, strings.Join(p, "; "))
		}
	}
	it.Status = status
	if err := d.saveBeat(ctx, data, projectID, beat, putItem(data.beats[beat], at, it), data.rows[beat].CreatorFields); err != nil {
		return nil, err
	}
	return d.State(ctx, projectID)
}

// Approve approves the item of a plan unit.
func (d *SequenceDesk) Approve(ctx context.Context, projectID, unitID string) (*SequenceState, error) {
	return d.setItemStatus(ctx, projectID, unitID, domain.UnitApproved)
}

// Unapprove puts the item of a plan unit back to review.
func (d *SequenceDesk) Unapprove(ctx context.Context, projectID, unitID string) (*SequenceState, error) {
	return d.setItemStatus(ctx, projectID, unitID, domain.UnitPending)
}

// ApproveAll approves every pending item that passes its checks in a beat whose
// inputs did not change; the others stay.
func (d *SequenceDesk) ApproveAll(ctx context.Context, projectID string) (*SequenceState, error) {
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	for _, b := range data.in.Beats {
		seq := data.beats[b.Key]
		if len(seq.Items) == 0 || len(data.beatStaleSources(b.Key)) > 0 {
			continue
		}
		items := append([]domain.SeqItem{}, seq.Items...)
		changed := false
		for i, it := range items {
			if it.Status != domain.UnitApproved && len(data.itemProblems(it.UnitID, it)) == 0 {
				items[i].Status, changed = domain.UnitApproved, true
			}
		}
		if !changed {
			continue
		}
		if err := d.saveBeat(ctx, data, projectID, b.Key, domain.BeatSequence{Items: items}, data.rows[b.Key].CreatorFields); err != nil {
			return nil, err
		}
	}
	return d.State(ctx, projectID)
}

// ConfirmBeat records that a stale beat's items still fit its changed inputs:
// the beat takes the inputs' hashes of now and every item goes back to review.
func (d *SequenceDesk) ConfirmBeat(ctx context.Context, projectID, beatKey string) (*SequenceState, error) {
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if _, ok := data.in.Beat(beatKey); !ok {
		return nil, domain.ErrUnknownPlanBeat
	}
	seq := data.beats[beatKey]
	if len(seq.Items) == 0 {
		return nil, domain.ErrUnitNotWritten
	}
	items := append([]domain.SeqItem{}, seq.Items...)
	for i := range items {
		items[i].Status = domain.UnitPending
	}
	row := data.rows[beatKey]
	row.InputHashes = data.in.BeatHashes(beatKey, data.motifs)
	data.rows[beatKey] = row
	if err := d.saveBeat(ctx, data, projectID, beatKey, domain.BeatSequence{Items: items}, row.CreatorFields); err != nil {
		return nil, err
	}
	return d.State(ctx, projectID)
}

// --- motifs ---------------------------------------------------------------------

// saveMotifs stores the motifs; any change sends them back to review. A first
// save takes the hashes the motifs' inputs have now.
func (d *SequenceDesk) saveMotifs(ctx context.Context, data *sequenceData, set domain.MotifSet, status string, creatorFields []string) error {
	raw, err := set.Encode()
	if err != nil {
		return err
	}
	row := data.motifRow
	row.Content, row.Status, row.CreatorFields = raw, status, creatorFields
	if len(row.InputHashes) == 0 {
		row.InputHashes = data.in.MotifHashes()
	}
	if err := d.store.SaveUnit(ctx, row); err != nil {
		return err
	}
	data.motifRow, data.motifs = row, set
	return nil
}

// motifFromFields reads a motif from the Creator's fields over base.
func motifFromFields(base domain.Motif, fields map[string]json.RawMessage) (domain.Motif, []string, error) {
	names, err := editableNames(fields, domain.MotifEditableFields, "motif")
	if err != nil {
		return domain.Motif{}, nil, err
	}
	merged, err := overlay(base, fields, names)
	if err != nil {
		return domain.Motif{}, nil, err
	}
	m, err := domain.DecodeMotif(merged)
	if err != nil {
		return domain.Motif{}, nil, err
	}
	m.MotifID, m.PlannedUnits = base.MotifID, base.PlannedUnits
	return m, names, nil
}

// AddMotif adds a motif the Creator wrote; it takes the next motif number and
// every field it was given becomes the Creator's.
func (d *SequenceDesk) AddMotif(ctx context.Context, projectID string, fields map[string]json.RawMessage) (*SequenceState, error) {
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	set := data.motifs
	m, names, err := motifFromFields(domain.Motif{MotifID: domain.MotifID(set.NextSeq), PlannedUnits: []string{}}, fields)
	if err != nil {
		return nil, err
	}
	set.Motifs = append(append([]domain.Motif{}, set.Motifs...), m)
	set.NextSeq++
	if err := d.saveMotifs(ctx, data, set, domain.UnitPending, withCreatorFields(data.motifRow, m.MotifID, names...)); err != nil {
		return nil, err
	}
	return d.State(ctx, projectID)
}

// EditMotif writes the Creator's fields into a motif.
func (d *SequenceDesk) EditMotif(ctx context.Context, projectID, motifID string, fields map[string]json.RawMessage) (*SequenceState, error) {
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	at := data.motifs.Index(motifID)
	if at < 0 {
		return nil, domain.ErrUnknownMotif
	}
	m, names, err := motifFromFields(data.motifs.Motifs[at], fields)
	if err != nil {
		return nil, err
	}
	set := data.motifs
	set.Motifs = append([]domain.Motif{}, set.Motifs...)
	set.Motifs[at] = m
	if err := d.saveMotifs(ctx, data, set, domain.UnitPending, withCreatorFields(data.motifRow, motifID, names...)); err != nil {
		return nil, err
	}
	return d.State(ctx, projectID)
}

// DeleteMotif removes a motif: it leaves every item that named it, an item that
// changes goes back to review, and the motif's number is not used again.
func (d *SequenceDesk) DeleteMotif(ctx context.Context, projectID, motifID string) (*SequenceState, error) {
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	at := data.motifs.Index(motifID)
	if at < 0 {
		return nil, domain.ErrUnknownMotif
	}
	set := data.motifs
	set.Motifs = append(append([]domain.Motif{}, set.Motifs[:at]...), set.Motifs[at+1:]...)
	kept := []string{}
	for _, f := range data.motifRow.CreatorFields {
		if !strings.HasPrefix(f, motifID+".") {
			kept = append(kept, f)
		}
	}
	if err := d.saveMotifs(ctx, data, set, domain.UnitPending, kept); err != nil {
		return nil, err
	}
	for _, b := range data.in.Beats {
		items := append([]domain.SeqItem{}, data.beats[b.Key].Items...)
		changed := false
		for i, it := range items {
			if out, did := it.WithoutMotif(motifID); did {
				out.Status = domain.UnitPending
				items[i], changed = out, true
			}
		}
		if !changed {
			continue
		}
		if err := d.saveBeat(ctx, data, projectID, b.Key, domain.BeatSequence{Items: items}, data.rows[b.Key].CreatorFields); err != nil {
			return nil, err
		}
	}
	return d.State(ctx, projectID)
}

// ApproveMotifs approves the video's motifs when there is at least one and each
// passes its checks. Approving also confirms they still fit inputs that changed:
// the motifs take the inputs' hashes of now.
func (d *SequenceDesk) ApproveMotifs(ctx context.Context, projectID string) (*SequenceState, error) {
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if len(data.motifs.Motifs) == 0 {
		return nil, domain.ErrNoMotif
	}
	var p []string
	for _, m := range data.motifs.Motifs {
		for _, prob := range domain.CheckMotif(m, data.in.AnchorRefs) {
			p = append(p, m.MotifID+": "+prob)
		}
	}
	if len(p) > 0 {
		return nil, fmt.Errorf("%w: %s", domain.ErrUnitInvalid, strings.Join(p, "; "))
	}
	data.motifRow.InputHashes = data.in.MotifHashes()
	if err := d.saveMotifs(ctx, data, data.motifs, domain.UnitApproved, data.motifRow.CreatorFields); err != nil {
		return nil, err
	}
	return d.State(ctx, projectID)
}

// UnapproveMotifs puts the motifs back to review.
func (d *SequenceDesk) UnapproveMotifs(ctx context.Context, projectID string) (*SequenceState, error) {
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if len(data.motifs.Motifs) == 0 {
		return nil, domain.ErrNoMotif
	}
	if err := d.saveMotifs(ctx, data, data.motifs, domain.UnitPending, data.motifRow.CreatorFields); err != nil {
		return nil, err
	}
	return d.State(ctx, projectID)
}

// --- review findings ------------------------------------------------------------

// saveReview stores the review of the whole sequence.
func (d *SequenceDesk) saveReview(ctx context.Context, data *sequenceData) error {
	raw, err := json.Marshal(data.review)
	if err != nil {
		return err
	}
	row := data.reviewRow
	row.Content = raw
	return d.store.SaveUnit(ctx, row)
}

// ApplyFinding writes a finding's fix into its field of the item or motif it
// names: the field becomes the Creator's and the item or motifs go back to review.
func (d *SequenceDesk) ApplyFinding(ctx context.Context, projectID, findingID string) (*SequenceState, error) {
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	i := data.review.Find(findingID)
	if i < 0 {
		return nil, domain.ErrUnknownFinding
	}
	f := data.review.Findings[i]
	if f.State != domain.FindingOpen || f.Field == "" || strings.TrimSpace(f.Fix) == "" {
		return nil, domain.ErrFindingNotApplicable
	}
	if at := data.motifs.Index(f.UnitKey); at >= 0 {
		err = d.applyToMotif(ctx, data, at, f)
	} else {
		err = d.applyToItem(ctx, data, projectID, f)
	}
	if err != nil {
		return nil, err
	}
	data.review.Findings[i].State = domain.FindingApplied
	if err := d.saveReview(ctx, data); err != nil {
		return nil, err
	}
	return d.State(ctx, projectID)
}

func (d *SequenceDesk) applyToMotif(ctx context.Context, data *sequenceData, at int, f domain.Finding) error {
	if !containsStr(domain.MotifEditableFields, f.Field) {
		return domain.ErrFindingNotApplicable
	}
	m := data.motifs.Motifs[at]
	cur, err := json.Marshal(m)
	if err != nil {
		return err
	}
	fixed, err := domain.ApplyFix(cur, f.Field, f.Fix, f.Replaces)
	if err != nil {
		return err
	}
	out, err := domain.DecodeMotif(fixed)
	if err != nil {
		return domain.ErrFindingNotApplicable
	}
	set := data.motifs
	set.Motifs = append([]domain.Motif{}, set.Motifs...)
	set.Motifs[at] = out
	return d.saveMotifs(ctx, data, set, domain.UnitPending, withCreatorFields(data.motifRow, m.MotifID, f.Field))
}

func (d *SequenceDesk) applyToItem(ctx context.Context, data *sequenceData, projectID string, f domain.Finding) error {
	if !containsStr(domain.SequenceEditableFields, f.Field) {
		return domain.ErrFindingNotApplicable
	}
	beat, it, at, err := data.locateItem(f.UnitKey)
	if err != nil {
		return err
	}
	if at < 0 {
		return domain.ErrFindingNotApplicable
	}
	cur, err := json.Marshal(it)
	if err != nil {
		return err
	}
	fixed, err := domain.ApplyFix(cur, f.Field, f.Fix, f.Replaces)
	if err != nil {
		return err
	}
	out, err := domain.DecodeSeqItem(fixed)
	if err != nil {
		return domain.ErrFindingNotApplicable
	}
	out.Status = domain.UnitPending
	return d.saveBeat(ctx, data, projectID, beat, putItem(data.beats[beat], at, out), withCreatorFields(data.rows[beat], f.UnitKey, f.Field))
}

// AcceptFinding accepts a finding, or a failing machine check, with the
// Creator's reason. A machine check accepted opens again when what fails changes.
func (d *SequenceDesk) AcceptFinding(ctx context.Context, projectID, findingID, why string) (*SequenceState, error) {
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	why = strings.TrimSpace(why)
	if why == "" {
		why = "không ghi lý do"
	}
	if strings.HasPrefix(findingID, "m-") {
		if err := acceptCheck(&data.review, domain.SequenceChecks(data.in, data.beats, data.motifs), findingID, why); err != nil {
			return nil, err
		}
	} else {
		i := data.review.Find(findingID)
		if i < 0 {
			return nil, domain.ErrUnknownFinding
		}
		if data.review.Findings[i].State != domain.FindingOpen {
			return nil, domain.ErrFindingNotApplicable
		}
		data.review.Findings[i].State, data.review.Findings[i].Why = domain.FindingAccepted, why
	}
	if err := d.saveReview(ctx, data); err != nil {
		return nil, err
	}
	return d.State(ctx, projectID)
}

// acceptCheck records the Creator's reason on the failing machine check id.
func acceptCheck(review *domain.ReviewState, checks []domain.MachineCheck, id, why string) error {
	for _, c := range checks {
		if c.ID != id {
			continue
		}
		if c.OK {
			return domain.ErrFindingNotApplicable
		}
		if review.Checks == nil {
			review.Checks = map[string]domain.CheckDecision{}
		}
		review.Checks[c.ID] = domain.CheckDecision{Why: why, Fingerprint: domain.Fingerprint(c.Detail)}
		return nil
	}
	return domain.ErrUnknownFinding
}
