package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
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
			latest[r.Path] = r
		}
	}
	out := []AIRun{}
	for _, r := range latest {
		out = append(out, r)
	}
	return out, nil
}

// deskReplayer stands in for the router: it serves the step 11 unit and asset
// list routes by calling the desk, answering like the HTTP layer would.
type deskReplayer struct{ f *sbFixture }

func (d deskReplayer) Replay(ctx context.Context, req AIRunRequest) (int, []byte) {
	var in struct {
		Note string `json:"note"`
	}
	_ = json.Unmarshal([]byte(req.Body), &in)
	prefix := "/v1/projects/" + project + "/storyboard"
	var st *StoryboardScreen
	var err error
	switch {
	case req.Path == prefix+"/assets/generate":
		st, err = d.f.desk.GenerateAssets(ctx, project, in.Note)
	case strings.HasPrefix(req.Path, prefix+"/units/") && strings.HasSuffix(req.Path, "/generate"):
		unit := strings.TrimSuffix(strings.TrimPrefix(req.Path, prefix+"/units/"), "/generate")
		st, err = d.f.desk.GenerateUnit(ctx, project, unit, in.Note)
	case req.Path == prefix+"/model":
		st, err = d.f.desk.State(ctx, project)
	default:
		return http.StatusNotFound, []byte(`{"error":"not found"}`)
	}
	if err != nil {
		status := http.StatusBadGateway
		if errors.Is(err, ErrExternalAIPromptBuilt) || errors.Is(err, ErrExternalAIPreviewReady) {
			status = http.StatusOK
		} else if errors.Is(err, domain.ErrUnitRunning) || errors.Is(err, domain.ErrAssetsMissing) {
			status = http.StatusConflict
		}
		b, _ := json.Marshal(map[string]string{"error": err.Error()})
		return status, b
	}
	b, _ := json.Marshal(st)
	return http.StatusOK, b
}

func newRuns(f *sbFixture) (*AIRuns, *fakeAIRunStore) {
	store := &fakeAIRunStore{}
	return NewAIRuns(deskReplayer{f}, store).WithInlineCalls(), store
}

func unitReq(unit, note string) AIRunRequest {
	b, _ := json.Marshal(map[string]string{"note": note})
	return AIRunRequest{Method: http.MethodPost, Path: "/v1/projects/" + project + "/storyboard/units/" + unit + "/generate", Body: string(b)}
}

func TestAIRuns_APastedReplyIsPreviewedWithoutBeingStoredThenApplied(t *testing.T) {
	f := newSbFixture(t)
	f.approvedAssets()
	runs, _ := newRuns(f)
	run, err := runs.Start(f.ctx, project, "Viết shot VP-001", unitReq("VP-001", "ghi chú"), AIRunManual)
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
	run, state, err := runs.Apply(f.ctx, project, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != AIRunApplied || !strings.Contains(string(state), `"SH-001"`) || len(f.rendering.frames) != 1 {
		t.Fatalf("applying stores the shots, draws their frames and answers the screen: %+v", run)
	}
	if _, _, err := runs.Apply(f.ctx, project, run.ID); !errors.Is(err, ErrAIRunState) {
		t.Errorf("a run is applied once, got %v", err)
	}
}

func TestAIRuns_AnUnreadableReplyIsKeptWithTheReasonAndCanBeSentAgain(t *testing.T) {
	f := newSbFixture(t)
	f.approvedAssets()
	runs, _ := newRuns(f)
	run, err := runs.Start(f.ctx, project, "", unitReq("VP-001", ""), AIRunExtension)
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
	run, err := runs.Start(f.ctx, project, "", unitReq("VP-001", ""), AIRunAPI)
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
}

func TestAIRuns_AModelFailureLeavesAFailedRunWithTheReason(t *testing.T) {
	f := newSbFixture(t)
	runs, store := newRuns(f) // no reply queued: the fake model fails
	b, _ := json.Marshal(map[string]string{"note": ""})
	run, err := runs.Start(f.ctx, project, "", AIRunRequest{Method: http.MethodPost, Path: "/v1/projects/" + project + "/storyboard/assets/generate", Body: string(b)}, AIRunAPI)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := store.AIRun(f.ctx, project, run.ID)
	if got.Status != AIRunFailed || got.Error == "" {
		t.Fatalf("a failed call says why: %+v", got)
	}
}

func TestAIRuns_ANewRunOfTheSamePathCancelsTheOpenOne(t *testing.T) {
	f := newSbFixture(t)
	f.approvedAssets()
	runs, store := newRuns(f)
	first, err := runs.Start(f.ctx, project, "", unitReq("VP-001", ""), AIRunManual)
	if err != nil {
		t.Fatal(err)
	}
	other, err := runs.Start(f.ctx, project, "", unitReq("VP-002", ""), AIRunManual)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runs.Start(f.ctx, project, "", unitReq("VP-001", "lại"), AIRunManual); err != nil {
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
	f.approvedAssets()
	runs, _ := newRuns(f)
	ok := unitReq("VP-001", "")
	for name, c := range map[string]struct {
		req     AIRunRequest
		channel AIRunChannel
	}{
		"channel":       {ok, "fax"},
		"method":        {AIRunRequest{Method: http.MethodGet, Path: ok.Path}, AIRunAPI},
		"other project": {AIRunRequest{Method: http.MethodPost, Path: "/v1/projects/other/storyboard/units/VP-001/generate"}, AIRunAPI},
		"runs route":    {AIRunRequest{Method: http.MethodPost, Path: "/v1/projects/" + project + "/visual/ai-runs"}, AIRunAPI},
		"body":          {AIRunRequest{Method: http.MethodPost, Path: ok.Path, Body: "{"}, AIRunAPI},
	} {
		if _, err := runs.Start(f.ctx, project, "", c.req, c.channel); !errors.Is(err, ErrBadAIRun) {
			t.Errorf("%s: want ErrBadAIRun, got %v", name, err)
		}
	}
	// A request that asks no AI builds no prompt and is refused.
	if _, err := runs.Start(f.ctx, project, "", AIRunRequest{Method: http.MethodPut, Path: "/v1/projects/" + project + "/storyboard/model"}, AIRunAPI); !errors.Is(err, ErrBadAIRun) {
		t.Errorf("no AI: want ErrBadAIRun, got %v", err)
	}
	// The step's own refusal keeps its status.
	var refused *AIRunHTTPError
	if _, err := runs.Start(f.ctx, project, "", AIRunRequest{Method: http.MethodPost, Path: "/v1/projects/" + project + "/storyboard/nope"}, AIRunAPI); !errors.As(err, &refused) || refused.Status != http.StatusNotFound {
		t.Errorf("refusal: %v", err)
	}
	if _, err := runs.Reply(f.ctx, project, "nope", "x"); !errors.Is(err, ErrUnknownAIRun) {
		t.Errorf("unknown run: %v", err)
	}
}

func TestAIRuns_AnAssetListIsPreviewedBeforeRenderingChanges(t *testing.T) {
	f := newSbFixture(t)
	runs, _ := newRuns(f)
	b, _ := json.Marshal(map[string]string{"note": ""})
	run, err := runs.Start(f.ctx, project, "", AIRunRequest{Method: http.MethodPost, Path: "/v1/projects/" + project + "/storyboard/assets/generate", Body: string(b)}, AIRunManual)
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
