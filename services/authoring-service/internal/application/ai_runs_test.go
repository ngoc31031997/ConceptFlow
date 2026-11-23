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
)

type runsStore struct {
	mu   sync.Mutex
	runs []AIRun
}

func (s *runsStore) CreateAIRun(_ context.Context, run AIRun) (AIRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	run.ID = fmt.Sprintf("run-%d", len(s.runs)+1)
	run.CreatedAt = time.Unix(int64(len(s.runs)+1), 0)
	s.runs = append(s.runs, run)
	return run, nil
}

func (s *runsStore) AIRun(_ context.Context, projectID, id string) (AIRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, r := range s.runs {
		if r.ID == id && r.ProjectID == projectID {
			return r, nil
		}
	}
	return AIRun{}, ErrUnknownAIRun
}

func (s *runsStore) SaveAIRun(_ context.Context, run AIRun) error {
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

func (s *runsStore) LatestAIRuns(_ context.Context, projectID string) ([]AIRun, error) {
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

type scriptedModel struct{ reply string }

func (m scriptedModel) Name() string { return "fake" }
func (m scriptedModel) Chat(context.Context, ChatRequest) (ChatResult, error) {
	if m.reply == "" {
		return ChatResult{}, errors.New("model down")
	}
	return ChatResult{Content: m.reply, JSON: json.RawMessage(m.reply)}, nil
}

type echoReplier struct{}

func (echoReplier) ChatExternal(_ context.Context, _ ChatRequest, reply string) (ChatResult, error) {
	return ChatResult{Content: reply, JSON: json.RawMessage(reply)}, nil
}

// toyStep is a step flow the way every step is built: one model turn, the
// reply read and checked, stopForPreview, then the store.
type toyStep struct {
	provider *ExternalAIProvider
	stored   []string
}

func (s *toyStep) Replay(ctx context.Context, req AIRunRequest) (int, []byte) {
	var in struct {
		Note string `json:"note"`
	}
	_ = json.Unmarshal([]byte(req.Body), &in)
	res, err := s.provider.Chat(ctx, ChatRequest{System: "Viết ý tưởng. " + in.Note})
	if errors.Is(err, ErrExternalAIPromptBuilt) {
		return http.StatusOK, []byte(`{}`)
	}
	if err != nil {
		return http.StatusBadGateway, []byte(`{"error":"` + err.Error() + `"}`)
	}
	var parsed struct {
		Title string `json:"title"`
	}
	if json.Unmarshal(res.JSON, &parsed) != nil || parsed.Title == "" {
		return http.StatusBadGateway, []byte(`{"error":"AI trả về không đọc được: thiếu title"}`)
	}
	if err := stopForPreview(ctx, parsed); err != nil {
		return http.StatusOK, []byte(`{}`)
	}
	s.stored = append(s.stored, parsed.Title)
	return http.StatusOK, []byte(`{"title":"` + parsed.Title + `"}`)
}

func toyRuns(reply string) (*AIRuns, *toyStep, *runsStore) {
	step := &toyStep{provider: NewExternalAIProvider(scriptedModel{reply: reply}, echoReplier{})}
	store := &runsStore{}
	return NewAIRuns(step, store).WithInlineCalls(), step, store
}

var ideaReq = AIRunRequest{Method: http.MethodPost, Path: "/v1/projects/p1/idea/generate", Body: `{"note":"ngắn"}`}

func TestAIRuns_PasteIsPreviewedThenApplied(t *testing.T) {
	runs, step, _ := toyRuns("")
	run, err := runs.Start(context.Background(), "p1", "Ý tưởng", ideaReq, AIRunManual)
	if err != nil || run.Status != AIRunWaiting || !strings.Contains(run.Prompt, "ngắn") {
		t.Fatalf("start: %+v %v", run, err)
	}
	run, err = runs.Reply(context.Background(), "p1", run.ID, `{"title":""}`)
	if err != nil || run.Status != AIRunUnreadable || !strings.Contains(run.Error, "thiếu title") {
		t.Fatalf("unreadable: %+v %v", run, err)
	}
	run, err = runs.Reply(context.Background(), "p1", run.ID, `{"title":"Hố đen"}`)
	if err != nil || run.Status != AIRunReady || string(run.Preview) != `{"title":"Hố đen"}` || len(step.stored) != 0 {
		t.Fatalf("preview: %+v %v stored %v", run, err, step.stored)
	}
	run, state, err := runs.Apply(context.Background(), "p1", run.ID)
	if err != nil || run.Status != AIRunApplied || len(step.stored) != 1 || !strings.Contains(string(state), "Hố đen") {
		t.Fatalf("apply: %+v %v %s", run, err, state)
	}
}

func TestAIRuns_TheModelsReplyIsCapturedOrTheFailureKept(t *testing.T) {
	runs, step, store := toyRuns(`{"title":"Sao neutron"}`)
	run, err := runs.Start(context.Background(), "p1", "", ideaReq, AIRunAPI)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := store.AIRun(context.Background(), "p1", run.ID)
	if got.Status != AIRunReady || got.Reply == "" || len(step.stored) != 0 {
		t.Fatalf("capture: %+v", got)
	}
	runs, _, store = toyRuns("")
	run, _ = runs.Start(context.Background(), "p1", "", ideaReq, AIRunAPI)
	got, _ = store.AIRun(context.Background(), "p1", run.ID)
	if got.Status != AIRunFailed || !strings.Contains(got.Error, "model down") {
		t.Fatalf("failure: %+v", got)
	}
}

func TestAIRuns_RequestsOutsideTheProjectAreRefused(t *testing.T) {
	runs, _, _ := toyRuns("")
	for _, req := range []AIRunRequest{
		{Method: http.MethodPost, Path: "/v1/projects/p2/idea/generate"},
		{Method: http.MethodGet, Path: "/v1/projects/p1/idea"},
		{Method: http.MethodPost, Path: "/v1/projects/p1/ai-runs"},
	} {
		if _, err := runs.Start(context.Background(), "p1", "", req, AIRunAPI); !errors.Is(err, ErrBadAIRun) {
			t.Errorf("%+v: want ErrBadAIRun, got %v", req, err)
		}
	}
}
