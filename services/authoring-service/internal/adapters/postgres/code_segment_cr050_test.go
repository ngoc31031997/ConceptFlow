package postgres

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"authoring/internal/domain"
)

// CR-050 Unit 2 against a real database (TEST_DATABASE_URL): the segment
// store's rules — a plan resets only what changed, a failure keeps the last
// content, a repair keeps the source, the startup sweep, the chunk size, the
// diagnostics log, and a deleted project takes its segments with it.
func TestCodeSegmentsAgainstPostgres(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := NewPool(ctx, url, 4)
	if err != nil {
		t.Fatal(err)
	}
	r := NewPromptTemplateRepository(pool)
	const pid, step = "cr050-seg", "code"
	t.Cleanup(func() { _ = r.DeleteAuthoring(ctx, pid) })
	_ = r.DeleteAuthoring(ctx, pid)

	seg := func(key, fp string, pos int) domain.CodeSegment {
		kind := domain.SegmentKindShots
		if key == "frame" {
			kind = domain.SegmentKindFrame
		}
		return domain.CodeSegment{Key: key, Kind: kind, Position: pos, Shots: []string{key}, Fingerprint: fp}
	}
	get := func(key string) domain.CodeSegment {
		t.Helper()
		list, err := r.ListSegments(ctx, pid, step)
		if err != nil {
			t.Fatal(err)
		}
		for _, s := range list {
			if s.Key == key {
				return s
			}
		}
		t.Fatalf("no segment %s in %+v", key, list)
		return domain.CodeSegment{}
	}

	if err := r.ApplySegmentPlan(ctx, pid, step, []domain.CodeSegment{seg("frame", "f", 0), seg("a", "fa", 1), seg("b", "fb", 2)}); err != nil {
		t.Fatal(err)
	}
	if s := get("a"); s.Status != domain.SegmentPending || s.Fingerprint != "fa" || s.Position != 1 {
		t.Fatalf("planned = %+v", s)
	}
	done := seg("a", "fa", 1)
	done.Content, done.Source, done.DurationMS = json.RawMessage(`{"shots":{"a":"A"}}`), "external", 70
	if err := r.SaveSegmentDone(ctx, pid, step, done, false); err != nil {
		t.Fatal(err)
	}
	fixed := done
	fixed.Content, fixed.Source, fixed.DurationMS = json.RawMessage(`{"shots":{"a":"FIXED"}}`), "", 0
	if err := r.SaveSegmentDone(ctx, pid, step, fixed, true); err != nil {
		t.Fatal(err)
	}
	if s := get("a"); s.Status != domain.SegmentDone || s.Source != "external" || s.DurationMS != 70 || string(s.Content) != `{"shots": {"a": "FIXED"}}` {
		t.Fatalf("repaired = %+v content=%s", s, s.Content)
	}
	// a failed re-run keeps the last content (R1)
	if err := r.MarkSegmentRunning(ctx, pid, step, "a"); err != nil {
		t.Fatal(err)
	}
	if err := r.SaveSegmentFailed(ctx, pid, step, "a", "timeout", "slow", 5); err != nil {
		t.Fatal(err)
	}
	if s := get("a"); s.Status != domain.SegmentFailed || s.ErrorKind != "timeout" || len(s.Content) == 0 {
		t.Fatalf("failed = %+v", s)
	}
	if err := r.SaveSegmentFailed(ctx, pid, step, "nope", "x", "y", 0); err == nil {
		t.Error("failing an unknown segment must be an error")
	}

	// same fingerprint: untouched; changed fingerprint: back to pending, empty; gone key: deleted
	if err := r.SaveSegmentDone(ctx, pid, step, func() domain.CodeSegment {
		s := seg("b", "fb", 2)
		s.Content, s.Source = json.RawMessage(`{"shots":{"b":"B"}}`), "ai"
		return s
	}(), false); err != nil {
		t.Fatal(err)
	}
	if err := r.ApplySegmentPlan(ctx, pid, step, []domain.CodeSegment{seg("frame", "f", 0), seg("b", "fb", 1), seg("c", "fc", 2)}); err != nil {
		t.Fatal(err)
	}
	list, _ := r.ListSegments(ctx, pid, step)
	if len(list) != 3 || list[0].Key != "frame" || list[1].Key != "b" || list[2].Key != "c" {
		t.Fatalf("after re-plan = %+v", list)
	}
	if s := get("b"); s.Status != domain.SegmentDone || s.Position != 1 || s.Source != "ai" {
		t.Errorf("unchanged segment touched: %+v", s)
	}
	if err := r.ApplySegmentPlan(ctx, pid, step, []domain.CodeSegment{seg("frame", "f", 0), seg("b", "fb2", 1), seg("c", "fc", 2)}); err != nil {
		t.Fatal(err)
	}
	if s := get("b"); s.Status != domain.SegmentPending || len(s.Content) != 0 || s.Source != "" || s.Fingerprint != "fb2" {
		t.Errorf("changed segment not reset: %+v", s)
	}

	// running segments: one project's, then the startup sweep
	_ = r.MarkSegmentRunning(ctx, pid, step, "b")
	_ = r.MarkSegmentRunning(ctx, pid, step, "c")
	if n, err := r.FailRunningSegments(ctx, pid, step, domain.SegmentErrCancelled, "đã huỷ"); err != nil || n != 2 {
		t.Fatalf("fail running: %d %v", n, err)
	}
	_ = r.MarkSegmentRunning(ctx, pid, step, "c")
	if n, err := r.FailAllRunningSegments(ctx, domain.SegmentErrInterrupted, "bị ngắt"); err != nil || n < 1 {
		t.Fatalf("sweep: %d %v", n, err)
	}
	if s := get("c"); s.Status != domain.SegmentFailed || s.ErrorKind != domain.SegmentErrInterrupted {
		t.Errorf("swept = %+v", s)
	}

	// chunk size: default without a row, stored after
	if n, err := r.GetCodeChunkShots(ctx, pid); err != nil || n != domain.DefaultChunkShots {
		t.Fatalf("default chunk shots = %d %v", n, err)
	}
	if err := r.SaveCodeChunkShots(ctx, pid, 5); err != nil {
		t.Fatal(err)
	}
	if n, _ := r.GetCodeChunkShots(ctx, pid); n != 5 {
		t.Errorf("chunk shots = %d", n)
	}

	// diagnostics log, kept after the project is gone
	if _, err := pool.Exec(ctx, `DELETE FROM code_check_diagnostics WHERE project_id = $1`, pid); err != nil {
		t.Fatal(err)
	}
	if err := r.InsertCheckDiagnostics(ctx, []domain.CheckDiagnosticRecord{
		{ProjectID: pid, Engine: "remotion", Phase: "final", Round: 1, SegmentKey: "b", ShotID: "1.2", Kind: "layout", Rule: "safe_area", Message: "m", Line: 40},
		{ProjectID: pid, Engine: "remotion", Phase: "chunk", Kind: "compile", Message: "no line"},
	}); err != nil {
		t.Fatal(err)
	}

	if err := r.DeleteAuthoring(ctx, pid); err != nil {
		t.Fatal(err)
	}
	if list, _ := r.ListSegments(ctx, pid, step); len(list) != 0 {
		t.Errorf("segments left after the project was deleted: %+v", list)
	}
	var rules []string
	rows, err := pool.Query(ctx, `SELECT rule || '/' || COALESCE(line::text, '-') FROM code_check_diagnostics WHERE project_id = $1 ORDER BY id`, pid)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var s string
		_ = rows.Scan(&s)
		rules = append(rules, s)
	}
	rows.Close()
	if len(rules) != 2 || rules[0] != "safe_area/40" || rules[1] != "/-" {
		t.Errorf("diagnostics = %v", rules)
	}
	_, _ = pool.Exec(ctx, `DELETE FROM code_check_diagnostics WHERE project_id = $1`, pid)
}
