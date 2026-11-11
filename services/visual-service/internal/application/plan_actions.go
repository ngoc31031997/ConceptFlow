package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"visual/internal/domain"
)

// beatRow returns the stored row of beat key; a beat with no unit has none.
func (data *planData) beatRow(projectID, key string) domain.ReviewUnit {
	if row, ok := data.rows[key]; ok {
		return row
	}
	return domain.ReviewUnit{ProjectID: projectID, Kind: domain.UnitKindPlan, Key: key, Status: domain.UnitPending,
		CreatorFields: []string{}, InputHashes: data.in.HashesFor(key)}
}

// saveBeat stores the units of beat key. The row reads approved when it has
// units and every one is approved.
func (d *PlanDesk) saveBeat(ctx context.Context, data *planData, projectID, key string, plan domain.BeatPlan, creatorFields []string) error {
	raw, err := plan.Encode()
	if err != nil {
		return err
	}
	row := data.beatRow(projectID, key)
	row.Content, row.CreatorFields, row.Status = raw, creatorFields, domain.UnitPending
	if len(plan.Units) > 0 {
		row.Status = domain.UnitApproved
		for _, u := range plan.Units {
			if u.Status != domain.UnitApproved {
				row.Status = domain.UnitPending
			}
		}
	}
	if row.InputHashes == nil {
		row.InputHashes = data.in.HashesFor(key)
	}
	if err := d.store.SaveUnit(ctx, row); err != nil {
		return err
	}
	data.rows[key], data.plans[key] = row, plan
	return nil
}

// withCreatorFields returns the row's creator fields plus the given paths of
// unit id, each once.
func withCreatorFields(row domain.ReviewUnit, id string, fields ...string) []string {
	paths := make([]string, 0, len(fields))
	for _, f := range fields {
		paths = append(paths, id+"."+f)
	}
	return row.AddCreatorFields(paths...)
}

// Edit writes the Creator's fields into a unit; they become the Creator's, and
// the unit goes back to review.
func (d *PlanDesk) Edit(ctx context.Context, projectID, unitID string, fields map[string]json.RawMessage) (*PlanState, error) {
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	beat, at, ok := data.locate(unitID)
	if !ok {
		return nil, domain.ErrUnknownPlanUnit
	}
	if len(fields) == 0 {
		return nil, fmt.Errorf("%w: không có trường nào", domain.ErrInvalidEdit)
	}
	names := make([]string, 0, len(fields))
	for f := range fields {
		if !containsStr(domain.PlanEditableFields, f) {
			return nil, fmt.Errorf("%w: unit không có trường %q để sửa", domain.ErrInvalidEdit, f)
		}
		names = append(names, f)
	}
	plan := data.plans[beat]
	cur, err := json.Marshal(plan.Units[at])
	if err != nil {
		return nil, err
	}
	over, err := json.Marshal(fields)
	if err != nil {
		return nil, err
	}
	merged, err := domain.OverlayFields(cur, over, names)
	if err != nil {
		return nil, err
	}
	u, err := domain.DecodePlanUnit(merged)
	if err != nil {
		return nil, err
	}
	u.Status = domain.UnitPending
	plan.Units = replaceUnit(plan.Units, at, u)
	if err := d.saveBeat(ctx, data, projectID, beat, plan, withCreatorFields(data.rows[beat], unitID, sortedNames(names)...)); err != nil {
		return nil, err
	}
	return d.State(ctx, projectID)
}

func replaceUnit(units []domain.PlanUnit, at int, u domain.PlanUnit) []domain.PlanUnit {
	out := append([]domain.PlanUnit{}, units...)
	out[at] = u
	return out
}

// SetAnswer records the Creator's answer to an open question; the unit goes
// back to review.
func (d *PlanDesk) SetAnswer(ctx context.Context, projectID, unitID, questionID, answer string) (*PlanState, error) {
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	beat, at, ok := data.locate(unitID)
	if !ok {
		return nil, domain.ErrUnknownPlanUnit
	}
	answer = strings.TrimSpace(answer)
	if answer == "" {
		return nil, fmt.Errorf("%w: câu trả lời để trống", domain.ErrInvalidEdit)
	}
	plan := data.plans[beat]
	u := plan.Units[at]
	u.OpenQuestions = append([]domain.OpenQuestion{}, u.OpenQuestions...)
	found := false
	for i := range u.OpenQuestions {
		if u.OpenQuestions[i].ID == questionID {
			u.OpenQuestions[i].Answer, found = answer, true
		}
	}
	if !found {
		return nil, domain.ErrUnknownQuestion
	}
	u.Status = domain.UnitPending
	plan.Units = replaceUnit(plan.Units, at, u)
	if err := d.saveBeat(ctx, data, projectID, beat, plan, withCreatorFields(data.rows[beat], unitID, "open_questions")); err != nil {
		return nil, err
	}
	return d.State(ctx, projectID)
}

// Split cuts a unit before line atLine into two.
func (d *PlanDesk) Split(ctx context.Context, projectID, unitID, atLine string) (*PlanState, error) {
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	beat, at, ok := data.locate(unitID)
	if !ok {
		return nil, domain.ErrUnknownPlanUnit
	}
	if _, err := domain.SplitPoint(data.plans[beat].Units[at], atLine); err != nil {
		return nil, err
	}
	first, err := d.store.ReserveUnitNumbers(ctx, projectID, 1)
	if err != nil {
		return nil, err
	}
	newID := domain.PlanUnitID(first)
	plan := data.plans[beat]
	a, b, err := domain.SplitUnit(plan.Units[at], atLine, newID, data.in.Script.Lines[beat])
	if err != nil {
		return nil, err
	}
	units := append([]domain.PlanUnit{}, plan.Units[:at]...)
	units = append(append(units, a, b), plan.Units[at+1:]...)
	plan.Units = units
	fields := append([]string{}, data.rows[beat].CreatorFields...)
	for _, f := range unitCreatorFields(data.rows[beat], unitID) {
		fields = append(fields, newID+"."+f)
	}
	if err := d.saveBeat(ctx, data, projectID, beat, plan, fields); err != nil {
		return nil, err
	}
	return d.State(ctx, projectID)
}

// Merge joins the unit right after unitID into it.
func (d *PlanDesk) Merge(ctx context.Context, projectID, unitID, withID string) (*PlanState, error) {
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	beat, at, ok := data.locate(unitID)
	if !ok {
		return nil, domain.ErrUnknownPlanUnit
	}
	plan := data.plans[beat]
	if at+1 >= len(plan.Units) || plan.Units[at+1].UnitID != withID {
		return nil, domain.ErrNotMergeable
	}
	merged, err := domain.MergeUnits(plan.Units[at], plan.Units[at+1])
	if err != nil {
		return nil, err
	}
	units := append([]domain.PlanUnit{}, plan.Units[:at]...)
	units = append(append(units, merged), plan.Units[at+2:]...)
	plan.Units = units
	fields := []string{}
	for _, f := range data.rows[beat].CreatorFields {
		if !strings.HasPrefix(f, withID+".") {
			fields = append(fields, f)
		}
	}
	if err := d.saveBeat(ctx, data, projectID, beat, plan, fields); err != nil {
		return nil, err
	}
	return d.State(ctx, projectID)
}

// Approve approves a unit that passes its checks and has no open question. A
// beat whose sources changed cannot be approved until it is planned again.
func (d *PlanDesk) Approve(ctx context.Context, projectID, unitID string) (*PlanState, error) {
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	beat, at, ok := data.locate(unitID)
	if !ok {
		return nil, domain.ErrUnknownPlanUnit
	}
	if len(data.staleSources(beat)) > 0 {
		return nil, domain.ErrBeatStale
	}
	plan := data.plans[beat]
	if p := data.unitProblems(beat, plan.Units[at]); len(p) > 0 {
		return nil, fmt.Errorf("%w: %s", domain.ErrUnitInvalid, strings.Join(p, "; "))
	}
	u := plan.Units[at]
	u.Status = domain.UnitApproved
	plan.Units = replaceUnit(plan.Units, at, u)
	if err := d.saveBeat(ctx, data, projectID, beat, plan, data.rows[beat].CreatorFields); err != nil {
		return nil, err
	}
	return d.State(ctx, projectID)
}

// Unapprove puts an approved unit back to review.
func (d *PlanDesk) Unapprove(ctx context.Context, projectID, unitID string) (*PlanState, error) {
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	beat, at, ok := data.locate(unitID)
	if !ok {
		return nil, domain.ErrUnknownPlanUnit
	}
	plan := data.plans[beat]
	u := plan.Units[at]
	u.Status = domain.UnitPending
	plan.Units = replaceUnit(plan.Units, at, u)
	if err := d.saveBeat(ctx, data, projectID, beat, plan, data.rows[beat].CreatorFields); err != nil {
		return nil, err
	}
	return d.State(ctx, projectID)
}

// ApproveAll approves every pending unit that passes its checks, has no open
// question and sits in a beat whose sources did not change; the others stay.
func (d *PlanDesk) ApproveAll(ctx context.Context, projectID string) (*PlanState, error) {
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	for _, b := range data.in.Script.Beats {
		plan := data.plans[b.Key]
		if len(plan.Units) == 0 || len(data.staleSources(b.Key)) > 0 {
			continue
		}
		units := append([]domain.PlanUnit{}, plan.Units...)
		changed := false
		for i, u := range units {
			if u.Status != domain.UnitApproved && len(data.unitProblems(b.Key, u)) == 0 {
				units[i].Status, changed = domain.UnitApproved, true
			}
		}
		if !changed {
			continue
		}
		plan.Units = units
		if err := d.saveBeat(ctx, data, projectID, b.Key, plan, data.rows[b.Key].CreatorFields); err != nil {
			return nil, err
		}
	}
	return d.State(ctx, projectID)
}

// AutoFix repairs, on the Creator's request, the units the checks refuse for a
// time below the narration minimum or a risk note on a fact outside the unit's
// lines. Stale beats are left alone. The answer lists each change.
func (d *PlanDesk) AutoFix(ctx context.Context, projectID string) (*PlanState, error) {
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	did := []string{}
	for _, b := range data.in.Script.Beats {
		plan := data.plans[b.Key]
		if len(plan.Units) == 0 || len(data.staleSources(b.Key)) > 0 {
			continue
		}
		units := append([]domain.PlanUnit{}, plan.Units...)
		changed := false
		for i, u := range units {
			if len(data.unitProblems(b.Key, u)) == 0 {
				continue
			}
			fixed, notes := domain.AutoFixUnit(u, data.in.PlanContextFor(b.Key), unitCreatorFields(data.rows[b.Key], u.UnitID))
			if len(notes) == 0 {
				continue
			}
			units[i], changed = fixed, true
			for _, n := range notes {
				did = append(did, u.UnitID+": "+n)
			}
		}
		if !changed {
			continue
		}
		plan.Units = units
		if err := d.saveBeat(ctx, data, projectID, b.Key, plan, data.rows[b.Key].CreatorFields); err != nil {
			return nil, err
		}
	}
	st, err := d.State(ctx, projectID)
	if err != nil {
		return nil, err
	}
	st.AutoFixes = did
	return st, nil
}

// saveReview stores the review of the whole plan.
func (d *PlanDesk) saveReview(ctx context.Context, data *planData, r domain.ReviewState) error {
	raw, err := json.Marshal(r)
	if err != nil {
		return err
	}
	u := data.revUnit
	u.Content = raw
	if u.CreatorFields == nil {
		u.CreatorFields = []string{}
	}
	if u.InputHashes == nil {
		u.InputHashes = map[string]string{}
	}
	return d.store.SaveUnit(ctx, u)
}

// ApplyFinding writes a finding's fix into its field of the unit it names: the
// field becomes the Creator's and the unit goes back to review.
func (d *PlanDesk) ApplyFinding(ctx context.Context, projectID, findingID string) (*PlanState, error) {
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	i := data.review.Find(findingID)
	if i < 0 {
		return nil, domain.ErrUnknownFinding
	}
	f := data.review.Findings[i]
	if f.State != domain.FindingOpen || f.Field == "" || strings.TrimSpace(f.Fix) == "" || !containsStr(domain.PlanEditableFields, f.Field) {
		return nil, domain.ErrFindingNotApplicable
	}
	beat, at, ok := data.locate(f.UnitKey)
	if !ok {
		return nil, domain.ErrUnknownPlanUnit
	}
	plan := data.plans[beat]
	cur, err := json.Marshal(plan.Units[at])
	if err != nil {
		return nil, err
	}
	fixed, err := domain.ApplyFix(cur, f.Field, f.Fix, f.Replaces)
	if err != nil {
		return nil, err
	}
	u, err := domain.DecodePlanUnit(fixed)
	if err != nil {
		return nil, domain.ErrFindingNotApplicable
	}
	u.Status = domain.UnitPending
	plan.Units = replaceUnit(plan.Units, at, u)
	if err := d.saveBeat(ctx, data, projectID, beat, plan, withCreatorFields(data.rows[beat], f.UnitKey, f.Field)); err != nil {
		return nil, err
	}
	data.review.Findings[i].State = domain.FindingApplied
	if err := d.saveReview(ctx, data, data.review); err != nil {
		return nil, err
	}
	return d.State(ctx, projectID)
}

// AcceptFinding accepts a finding, or a failing machine check, with the
// Creator's reason. A machine check accepted opens again when what fails changes.
func (d *PlanDesk) AcceptFinding(ctx context.Context, projectID, findingID, why string) (*PlanState, error) {
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	why = strings.TrimSpace(why)
	if why == "" {
		why = "không ghi lý do"
	}
	if strings.HasPrefix(findingID, "m-") {
		for _, c := range domain.PlanChecks(data.in, data.plans) {
			if c.ID != findingID {
				continue
			}
			if c.OK {
				return nil, domain.ErrFindingNotApplicable
			}
			data.review.Checks[c.ID] = domain.CheckDecision{Why: why, Fingerprint: domain.Fingerprint(c.Detail)}
			if err := d.saveReview(ctx, data, data.review); err != nil {
				return nil, err
			}
			return d.State(ctx, projectID)
		}
		return nil, domain.ErrUnknownFinding
	}
	i := data.review.Find(findingID)
	if i < 0 {
		return nil, domain.ErrUnknownFinding
	}
	if data.review.Findings[i].State != domain.FindingOpen {
		return nil, domain.ErrFindingNotApplicable
	}
	data.review.Findings[i].State, data.review.Findings[i].Why = domain.FindingAccepted, why
	if err := d.saveReview(ctx, data, data.review); err != nil {
		return nil, err
	}
	return d.State(ctx, projectID)
}
