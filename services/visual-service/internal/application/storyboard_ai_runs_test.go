package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"visual/internal/domain"
)

type fakeAIRunStore struct {
	mu   sync.Mutex
	runs []AIRun
	next int
}

func (s *fakeAIRunStore) CreateAIRun(_ context.Context, run AIRun) (AIRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.next++
	run.ID = fmt.Sprintf("run-%d", s.next)
	run.CreatedAt = time.Unix(int64(s.next), 0)
	s.runs = append(s.runs, run)
	return run, nil
}

func (s *fakeAIRunStore) AIRun(_ context.Context, projectID, id string) (AIRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, r := range s.runs {
		if r.ID == id && r.ProjectID == projectID {
			return r, nil
		}
	}
	return AIRun{}, ErrUnknownAIRun
}

func (s *fakeAIRunStore) SaveAIRun(_ context.Context, run AIRun) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, r := range s.runs {
		if r.ID == run.ID {
			s.runs[i] = run
			return nil
		}
	}
	return ErrUnknownAIRun
}

func (s *fakeAIRunStore) LatestAIRuns(_ context.Context, projectID string) ([]AIRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	latest := map[string]AIRun{}
	for _, r := range s.runs {
		if r.ProjectID == projectID {
			latest[string(r.Task)+"/"+r.Target] = r
		}
	}
	out := []AIRun{}
	for _, r := range latest {
		out = append(out, r)
	}
	return out, nil
}

// newRuns wires the runs over the fixture's desk, with the model call made inline.
func newRuns(f *sbFixture) (*StoryboardAIRuns, *fakeAIRunStore) {
	store := &fakeAIRunStore{}
	runs := NewStoryboardAIRuns(f.desk, store)
	runs.background = func(fn func()) { fn() }
	return runs, store
}

func TestAIRuns_APastedReplyIsPreviewedWithoutBeingStoredThenApplied(t *testing.T) {
	f := newSbFixture(t)
	f.approvedAssets()
	runs, _ := newRuns(f)
	run, err := runs.Start(f.ctx, project, AIRunUnit, "VP-001", "ghi chú", AIRunManual)
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != AIRunWaiting || !strings.Contains(run.Prompt, "ghi chú") || len(f.llm.calls) != 0 {
		t.Fatalf("a manual run waits with its prompt and asks no model: %+v", run)
	}
	run, err = runs.Reply(f.ctx, project, run.ID, shotsReply(shotJSON([]string{"b1.1"}, nil, "mở")))
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != AIRunReady || !strings.Contains(string(run.Preview), `"shot_id":"SH-001"`) {
		t.Fatalf("the reply is read into a preview: %+v", run)
	}
	if len(f.store.units) != 0 || len(f.rendering.frames) != 0 {
		t.Fatal("a preview stores nothing and draws no frame")
	}
	run, st, err := runs.Apply(f.ctx, project, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != AIRunApplied || len(unitOf(st, "VP-001").Shots) != 1 || len(f.rendering.frames) != 1 {
		t.Fatalf("applying stores the shots and draws their frames: %+v", run)
	}
	if _, _, err := runs.Apply(f.ctx, project, run.ID); !errors.Is(err, ErrAIRunState) {
		t.Errorf("a run is applied once, got %v", err)
	}
}

func TestAIRuns_AnUnreadableReplyIsKeptWithTheReasonAndCanBeSentAgain(t *testing.T) {
	f := newSbFixture(t)
	f.approvedAssets()
	runs, _ := newRuns(f)
	run, err := runs.Start(f.ctx, project, AIRunUnit, "VP-001", "", AIRunExtension)
	if err != nil {
		t.Fatal(err)
	}
	run, err = runs.Reply(f.ctx, project, run.ID, `{"shots":[]}`)
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != AIRunUnreadable || run.Error == "" || run.Reply != `{"shots":[]}` {
		t.Fatalf("the reply stays with why it cannot be used: %+v", run)
	}
	if _, _, err := runs.Apply(f.ctx, project, run.ID); !errors.Is(err, ErrAIRunState) {
		t.Errorf("an unreadable reply cannot be applied, got %v", err)
	}
	run, err = runs.Reply(f.ctx, project, run.ID, shotsReply(shotJSON([]string{"b1.1"}, nil, "mở")))
	if err != nil || run.Status != AIRunReady {
		t.Fatalf("a corrected reply is read again: %+v %v", run, err)
	}
}

func TestAIRuns_TheSystemModelsReplyIsCapturedAndPreviewed(t *testing.T) {
	f := newSbFixture(t)
	f.approvedAssets()
	f.queue(domain.PromptRoleStoryboardUnit, shotsReply(shotJSON([]string{"b1.1"}, nil, "mở")))
	runs, store := newRuns(f)
	run, err := runs.Start(f.ctx, project, AIRunUnit, "VP-001", "", AIRunAPI)
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != AIRunRunning {
		t.Fatalf("an API run answers while the model is asked: %+v", run)
	}
	got, _ := store.AIRun(f.ctx, project, run.ID)
	if got.Status != AIRunReady || got.Reply == "" || len(got.Preview) == 0 || len(f.store.units) != 0 {
		t.Fatalf("the model's reply is kept and previewed, nothing stored: %+v", got)
	}
	if len(f.llm.calls) != 1 || f.llm.calls[0].Role != domain.PromptRoleStoryboardUnit {
		t.Errorf("one model call: %+v", f.llm.calls)
	}
}

func TestAIRuns_AModelFailureLeavesAFailedRunWithTheReason(t *testing.T) {
	f := newSbFixture(t)
	f.approvedAssets()
	runs, store := newRuns(f) // no reply queued: the fake model fails
	run, err := runs.Start(f.ctx, project, AIRunReview, "", "", AIRunAPI)
	if err == nil {
		t.Fatalf("the review needs shots first, got %+v", run)
	}
	run, err = runs.Start(f.ctx, project, AIRunAssets, "", "", AIRunAPI)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := store.AIRun(f.ctx, project, run.ID)
	if got.Status != AIRunFailed || got.Error == "" {
		t.Fatalf("a failed call says why: %+v", got)
	}
}

func TestAIRuns_ANewRunOfTheSameTaskCancelsTheOpenOne(t *testing.T) {
	f := newSbFixture(t)
	f.approvedAssets()
	runs, store := newRuns(f)
	first, err := runs.Start(f.ctx, project, AIRunUnit, "VP-001", "", AIRunManual)
	if err != nil {
		t.Fatal(err)
	}
	other, err := runs.Start(f.ctx, project, AIRunUnit, "VP-002", "", AIRunManual)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runs.Start(f.ctx, project, AIRunUnit, "VP-001", "", AIRunManual); err != nil {
		t.Fatal(err)
	}
	if got, _ := store.AIRun(f.ctx, project, first.ID); got.Status != AIRunCancelled {
		t.Errorf("the older run of VP-001 is cancelled: %+v", got)
	}
	if got, _ := store.AIRun(f.ctx, project, other.ID); got.Status != AIRunWaiting {
		t.Errorf("the run of another unit stays: %+v", got)
	}
	if _, err := runs.Reply(f.ctx, project, first.ID, "x"); !errors.Is(err, ErrAIRunState) {
		t.Errorf("a cancelled run takes no reply, got %v", err)
	}
}

func TestAIRuns_RequestsAreChecked(t *testing.T) {
	f := newSbFixture(t)
	runs, _ := newRuns(f)
	for _, c := range []struct {
		task    AIRunTask
		target  string
		channel AIRunChannel
	}{{"draw", "", AIRunAPI}, {AIRunUnit, "", AIRunAPI}, {AIRunAssets, "VP-001", AIRunAPI}, {AIRunAssets, "", "fax"}} {
		if _, err := runs.Start(f.ctx, project, c.task, c.target, "", c.channel); !errors.Is(err, ErrBadAIRun) {
			t.Errorf("%+v: want ErrBadAIRun, got %v", c, err)
		}
	}
	if _, err := runs.Reply(f.ctx, project, "nope", "x"); !errors.Is(err, ErrUnknownAIRun) {
		t.Errorf("unknown run: %v", err)
	}
}

func TestAIRuns_AnAssetListAndADetailArePreviewedBeforeRenderingChanges(t *testing.T) {
	f := newSbFixture(t)
	runs, _ := newRuns(f)
	run, err := runs.Start(f.ctx, project, AIRunAssets, "", "", AIRunManual)
	if err != nil {
		t.Fatal(err)
	}
	run, err = runs.Reply(f.ctx, project, run.ID, assetsReply(assetJSON("", "astro", "Astro", "figure", "hero", "VP-001")))
	if err != nil || run.Status != AIRunReady || !strings.Contains(string(run.Preview), `"name":"Astro"`) {
		t.Fatalf("preview %+v %v", run, err)
	}
	if n := len(f.rendering.assets); n != 0 {
		t.Fatalf("a preview creates no asset at rendering, got %d", n)
	}
	if _, _, err := runs.Apply(f.ctx, project, run.ID); err != nil {
		t.Fatal(err)
	}
	if n := len(f.rendering.assets); n != 1 {
		t.Fatalf("applying creates the asset, got %d", n)
	}
}
