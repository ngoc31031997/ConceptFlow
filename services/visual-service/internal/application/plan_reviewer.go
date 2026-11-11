package application

import (
	"context"
	"fmt"

	"visual/internal/domain"
)

// Review has the AI review the whole plan in one turn. A later review compares
// with the earlier findings: those the AI no longer reports are fixed, new ones
// are added.
func (d *PlanDesk) Review(ctx context.Context, projectID, note string) (*PlanState, error) {
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
	unitIDs := map[string]bool{}
	for _, p := range data.plans {
		for _, u := range p.Units {
			unitIDs[u.UnitID] = true
		}
	}
	if len(unitIDs) == 0 {
		return nil, fmt.Errorf("%w: chưa có unit nào để rà soát", domain.ErrUnitNotWritten)
	}
	vars := planSharedVars(data.in)
	vars["plan"] = planText(data.in, data.plans, domain.ComputeTiming(data.in, data.plans))
	vars["checks"] = planChecksText(data.in, data.plans)
	vars["previous_findings"] = planPreviousFindings(data.review)
	vars["note"] = orNone(note)

	finish := d.running.start(projectID, domain.ReviewUnitKey)
	reply, err := d.ask(ctx, projectID, domain.PromptRolePlanReview, vars)
	finish()
	if err != nil {
		return nil, err
	}
	parsed, err := domain.ParsePlanReviewReply(reply, data.review, unitIDs)
	if err != nil {
		return nil, err
	}
	if err := d.saveReview(ctx, data, domain.MergeReview(data.review, parsed)); err != nil {
		return nil, err
	}
	return d.State(ctx, projectID)
}
