package application

import (
	"context"
	"encoding/json"
	"fmt"

	"authoring/internal/domain"
)

// SetSkipped records that the Creator skips the research step, or comes back to
// do it. Skipping keeps the questions and the facts; it does not approve the
// table. The step no longer holds the flow back while skipped.
func (uc *ResearchDeskUseCase) SetSkipped(ctx context.Context, projectID string, skipped bool) (GateView, error) {
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return GateView{}, err
	}
	unit, gate, err := uc.gate(ctx, projectID)
	if err != nil {
		return GateView{}, err
	}
	gate.Skipped = skipped
	saved, err := uc.saveGate(ctx, unit, gate)
	if err != nil {
		return GateView{}, err
	}
	return GateView{Status: saved.Status, Skipped: gate.Skipped, Resolved: saved.Status == domain.UnitApproved || gate.Skipped}, nil
}

// gateKind makes the step's gate approvable through the generic unit API
// (UnitKind for domain.UnitKindResearch). Approving it approves the fact table.
type gateKind struct{ desk *ResearchDeskUseCase }

// GateUnitKind returns the gate's UnitKind.
func (uc *ResearchDeskUseCase) GateUnitKind() UnitKind { return gateKind{desk: uc} }

// Initial gives the gate of a project that has none stored yet, so that the
// table can be approved before anything else was saved.
func (gateKind) Initial(_ context.Context, projectID, key string) (domain.ReviewUnit, bool) {
	if key != domain.ResearchUnitKey {
		return domain.ReviewUnit{}, false
	}
	return domain.ReviewUnit{
		ProjectID: projectID, Kind: domain.UnitKindResearch, Key: key, Status: domain.UnitPending, Content: json.RawMessage(`{}`),
	}, true
}

// Edit is refused: the gate holds no text.
func (gateKind) Edit(context.Context, string, string, map[string]string) (domain.ReviewUnit, error) {
	return domain.ReviewUnit{}, fmt.Errorf("%w: bảng dữ kiện được duyệt hoặc bỏ qua, không sửa ở đây", domain.ErrInvalidUnitEdit)
}

// Redo is refused: the AI does not write the fact table.
func (gateKind) Redo(context.Context, string, string, string) (domain.ReviewUnit, error) {
	return domain.ReviewUnit{}, fmt.Errorf("%w: bảng dữ kiện không làm lại bằng AI", domain.ErrInvalidUnitEdit)
}

// Impact is empty: nothing is built on the table yet.
func (gateKind) Impact(context.Context, string, string) ([]string, error) { return []string{}, nil }

// CheckApprovable refuses a table with no kept fact, and a step that is
// skipped.
func (k gateKind) CheckApprovable(ctx context.Context, unit domain.ReviewUnit) error {
	var gate domain.ResearchGate
	if err := json.Unmarshal(unit.Content, &gate); err != nil {
		return fmt.Errorf("read research gate: %w", err)
	}
	if gate.Skipped {
		return fmt.Errorf("%w: bước đang bỏ qua, bấm Làm nghiên cứu trước", domain.ErrInvalidStatus)
	}
	facts, err := k.desk.facts(ctx, unit.ProjectID)
	if err != nil {
		return err
	}
	for _, f := range facts {
		if f.Keep {
			return nil
		}
	}
	return domain.ErrResearchEmpty
}
