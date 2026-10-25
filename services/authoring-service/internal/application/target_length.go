package application

import (
	"context"

	"authoring/internal/domain"
)

// approvedTarget is the target length set by the approved brief; ok is false
// when no brief is approved, and the target is then empty.
func approvedTarget(ctx context.Context, briefs ApprovedBriefPort, projectID string) (target domain.TargetLength, ok bool, err error) {
	brief, ok, err := briefs.ApprovedBrief(ctx, projectID)
	if err != nil || !ok {
		return domain.TargetLength{}, false, err
	}
	return domain.NewTargetLength(brief.DurationSeconds), true, nil
}
