package postgres

import (
	"context"
	"encoding/json"
	"testing"

	"visual/internal/application"
)

func TestAIRunRepository_KeepsTheStructuredErrorOfAnUnreadableRun(t *testing.T) {
	data := openTestPool(t)
	repo := NewAIRunRepository(data.pool)
	ctx := context.Background()
	const p = "test-ai-run-error-detail"
	t.Cleanup(func() { _ = data.DeleteProjectData(ctx, p) })

	run, err := repo.CreateAIRun(ctx, application.AIRun{ProjectID: p, Channel: application.AIRunAPI, Status: application.AIRunRunning,
		Method: "POST", Path: "/v1/projects/" + p + "/storyboard/assets/AST-A-001/draw", Body: "{}", Prompt: "vẽ"})
	if err != nil {
		t.Fatal(err)
	}
	if run.ErrorDetail != nil {
		t.Fatalf("a new run has no error detail: %s", run.ErrorDetail)
	}
	detail := json.RawMessage(`{"kind":"syntax","message":"Thiếu dấu đóng thẻ <g>.","line":12,"snippet":"12: > <g>"}`)
	run.Status, run.Reply, run.Error, run.ErrorDetail = application.AIRunUnreadable, "reply", "Thiếu dấu đóng thẻ <g>.", detail
	if err := repo.SaveAIRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	got, err := repo.AIRun(ctx, p, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	var d struct {
		Kind string `json:"kind"`
		Line int    `json:"line"`
	}
	if err := json.Unmarshal(got.ErrorDetail, &d); err != nil || d.Kind != "syntax" || d.Line != 12 || got.Status != application.AIRunUnreadable {
		t.Fatalf("the stored detail reads back: %s %v", got.ErrorDetail, err)
	}
	got.Status, got.ErrorDetail = application.AIRunReady, nil
	if err := repo.SaveAIRun(ctx, got); err != nil {
		t.Fatal(err)
	}
	latest, err := repo.LatestAIRuns(ctx, p)
	if err != nil || len(latest) != 1 || latest[0].ErrorDetail != nil {
		t.Fatalf("a cleared detail is NULL again: %+v %v", latest, err)
	}
}
