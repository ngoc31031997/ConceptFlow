package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"visual/internal/domain"
)

func (d *ConceptDesk) saveReview(ctx context.Context, data *deskData, r domain.ReviewState) error {
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

// Review has the AI review the whole concept in one turn. A later review
// compares with the earlier findings: those the AI no longer reports are fixed,
// new ones are added.
func (d *ConceptDesk) Review(ctx context.Context, projectID, note string) (*ConceptState, error) {
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if d.running.has(projectID, domain.ReviewUnitKey) {
		return nil, domain.ErrUnitRunning
	}
	written := 0
	for _, spec := range domain.ConceptUnits {
		if u, ok := data.units[spec.Key]; ok && len(u.Content) > 0 {
			written++
		}
	}
	if written == 0 {
		return nil, fmt.Errorf("%w: chưa có phần nào để rà soát", domain.ErrUnitNotWritten)
	}
	vars := inputVars(data.in, data.fonts)
	vars["parts"] = partsText(data.units, "")
	vars["note"] = orNone(note)
	var checks []string
	for _, c := range domain.MachineChecks(data.in, data.parts) {
		state := "đạt"
		if !c.OK {
			state = "chưa đạt"
		}
		line := fmt.Sprintf("- [%s] %s", state, c.Label)
		if c.Detail != "" {
			line += ": " + c.Detail
		}
		checks = append(checks, line)
	}
	vars["checks"] = strings.Join(checks, "\n")
	var prev []string
	for _, f := range data.review.Findings {
		if f.State != domain.FindingFixed {
			prev = append(prev, fmt.Sprintf("%s · %s · %s: %s", f.ID, f.Group, f.UnitKey, f.Text))
		}
	}
	vars["previous_findings"] = "(lần rà soát đầu)"
	if len(prev) > 0 {
		vars["previous_findings"] = strings.Join(prev, "\n")
	}

	finish := d.running.start(projectID, domain.ReviewUnitKey)
	reply, err := d.ask(ctx, projectID, domain.PromptRoleReview, vars)
	finish()
	if err != nil {
		return nil, err
	}
	parsed, err := domain.ParseReviewReply(reply, data.review)
	if err != nil {
		return nil, err
	}
	if err := d.saveReview(ctx, data, domain.MergeReview(data.review, parsed)); err != nil {
		return nil, err
	}
	return d.State(ctx, projectID)
}

// ApplyFinding writes a finding's fix into its field: the field becomes the
// Creator's and the part goes back to review.
func (d *ConceptDesk) ApplyFinding(ctx context.Context, projectID, findingID string) (*ConceptState, error) {
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
	u, err := d.unitOf(data, projectID, f.UnitKey)
	if err != nil {
		return nil, err
	}
	fixed, err := domain.ApplyFix(u.Content, f.Field, f.Fix, f.Replaces)
	if err != nil {
		return nil, err
	}
	_, canon, err := domain.DecodePart(f.UnitKey, fixed)
	if err != nil {
		return nil, domain.ErrFindingNotApplicable
	}
	u.Content, u.Status = canon, domain.UnitPending
	u.CreatorFields = u.AddCreatorFields(f.Field)
	if err := d.store.SaveUnit(ctx, u); err != nil {
		return nil, err
	}
	data.review.Findings[i].State = domain.FindingApplied
	if err := d.saveReview(ctx, data, data.review); err != nil {
		return nil, err
	}
	return d.State(ctx, projectID)
}

// MarkFindingRewritten records that the AI rewrote the part a finding names, so
// the finding reads as applied; the next review decides whether it is really gone.
// A finding that is not open, or names another part, is left as it is.
func (d *ConceptDesk) MarkFindingRewritten(ctx context.Context, projectID, findingID, unitKey string) (*ConceptState, error) {
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	i := data.review.Find(findingID)
	if i < 0 {
		return nil, domain.ErrUnknownFinding
	}
	if f := data.review.Findings[i]; f.State == domain.FindingOpen && f.UnitKey == unitKey {
		data.review.Findings[i].State = domain.FindingApplied
		if err := d.saveReview(ctx, data, data.review); err != nil {
			return nil, err
		}
	}
	return d.State(ctx, projectID)
}

// AcceptFinding accepts a finding, or a failing machine check, with the
// Creator's reason. A machine check accepted opens again when what fails changes.
func (d *ConceptDesk) AcceptFinding(ctx context.Context, projectID, findingID, why string) (*ConceptState, error) {
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	why = strings.TrimSpace(why)
	if why == "" {
		why = "không ghi lý do"
	}
	if strings.HasPrefix(findingID, "m-") {
		for _, c := range domain.MachineChecks(data.in, data.parts) {
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
