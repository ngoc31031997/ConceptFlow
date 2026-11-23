package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// AIRunChannel is where a run's reply comes from.
type AIRunChannel string

const (
	// AIRunAPI asks the system's model in the background.
	AIRunAPI AIRunChannel = "api"
	// AIRunExtension waits for the AI Bridge extension to post the reply.
	AIRunExtension AIRunChannel = "extension"
	// AIRunManual waits for the Creator to paste the reply.
	AIRunManual AIRunChannel = "manual"
)

// AIRunStatus is where a run stands.
type AIRunStatus string

const (
	// AIRunWaiting has its prompt and waits for a reply from the extension or a paste.
	AIRunWaiting AIRunStatus = "waiting"
	// AIRunRunning is asking the system's model.
	AIRunRunning AIRunStatus = "running"
	// AIRunReady has a reply that was read and checked; Preview is what applying stores.
	AIRunReady AIRunStatus = "ready"
	// AIRunUnreadable has a reply the step could not use; Error says why.
	AIRunUnreadable AIRunStatus = "unreadable"
	// AIRunFailed got no reply; Error says why.
	AIRunFailed AIRunStatus = "failed"
	// AIRunApplied had its reply stored by the step.
	AIRunApplied AIRunStatus = "applied"
	// AIRunCancelled was dropped by the Creator or replaced by a newer run of the same request.
	AIRunCancelled AIRunStatus = "cancelled"
)

// AIRunInterrupted is the error of a run that was asking the model when the
// service stopped.
const AIRunInterrupted = "service khởi động lại khi đang chờ AI; chạy lại lượt này."

var (
	// ErrUnknownAIRun is a run id the video does not have.
	ErrUnknownAIRun = errors.New("không có lượt chạy AI này")
	// ErrAIRunState is an action the run's status does not allow.
	ErrAIRunState = errors.New("lượt chạy AI không ở trạng thái cho phép việc này")
	// ErrBadAIRun is a run request with an unknown channel or a request this service does not run.
	ErrBadAIRun = errors.New("yêu cầu chạy AI không hợp lệ")
	// ErrNoAIRunProject is a run request without a project id.
	ErrNoAIRunProject = errors.New("thiếu mã dự án")
)

// AIRunRequest is the step's own AI request a run replays: the route and body
// the page would send to run the step's AI turn.
type AIRunRequest struct {
	Method string `json:"method"`
	Path   string `json:"path"`
	Body   string `json:"body"`
}

// AIRun is one AI turn of a step run with a preview (ADR-0052): the step's own
// request is replayed to build the prompt, to read a reply without storing it
// (Preview), and, when the Creator applies it, to store it.
type AIRun struct {
	ID        string          `json:"id"`
	ProjectID string          `json:"project_id"`
	Label     string          `json:"label"`
	Channel   AIRunChannel    `json:"channel"`
	Status    AIRunStatus     `json:"status"`
	Method    string          `json:"method"`
	Path      string          `json:"path"`
	Body      string          `json:"body"`
	Prompt    string          `json:"prompt"`
	Reply     string          `json:"reply"`
	Preview   json.RawMessage `json:"preview"`
	Error     string          `json:"error"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

func (r AIRun) request() AIRunRequest {
	return AIRunRequest{Method: r.Method, Path: r.Path, Body: r.Body}
}

// open reports whether the run still waits for something: a reply or an apply.
func (r AIRun) open() bool {
	switch r.Status {
	case AIRunWaiting, AIRunRunning, AIRunReady, AIRunUnreadable, AIRunFailed:
		return true
	}
	return false
}

// AIRunStorePort keeps the runs.
type AIRunStorePort interface {
	// CreateAIRun stores a new run and returns it with its id and times.
	CreateAIRun(ctx context.Context, run AIRun) (AIRun, error)
	// AIRun returns one run of the video, or ErrUnknownAIRun.
	AIRun(ctx context.Context, projectID, id string) (AIRun, error)
	// SaveAIRun writes the run's status, reply, preview and error.
	SaveAIRun(ctx context.Context, run AIRun) error
	// LatestAIRuns returns the newest run of every request path of the video.
	LatestAIRuns(ctx context.Context, projectID string) ([]AIRun, error)
}

// AIRunReplayer serves a step's request inside the service under ctx, which
// carries the external-AI mode, and returns the status and body it answered.
type AIRunReplayer interface {
	Replay(ctx context.Context, req AIRunRequest) (int, []byte)
}

// AIRunHTTPError is a refusal of the replayed step request, kept with its
// status so the caller answers with it.
type AIRunHTTPError struct {
	Status  int
	Message string
	Body    json.RawMessage
}

func (e *AIRunHTTPError) Error() string { return e.Message }

// AIRuns runs the single-prompt AI turns of the steps this service owns with a
// preview: the reply comes from the system's model, the extension or a paste,
// is read and checked by the step's own flow (stopForPreview), and is stored
// only on Apply.
type AIRuns struct {
	replay AIRunReplayer
	store  AIRunStorePort
	// background starts the call to the system's model; tests run it inline.
	background func(func())
}

// NewAIRuns wires the runs over the replayer and the store.
func NewAIRuns(replay AIRunReplayer, store AIRunStorePort) *AIRuns {
	return &AIRuns{replay: replay, store: store, background: func(f func()) { go f() }}
}

// WithInlineCalls makes the model call of an API run inside Start; for tests.
func (s *AIRuns) WithInlineCalls() *AIRuns {
	s.background = func(f func()) { f() }
	return s
}

// call replays the run's request under ctx and returns the step's refusal, if any.
func (s *AIRuns) call(ctx context.Context, req AIRunRequest) error {
	status, body := s.replay.Replay(ctx, req)
	if status >= 200 && status < 300 {
		return nil
	}
	return refusal(status, body)
}

// refusal reads the step's error answer.
func refusal(status int, body []byte) *AIRunHTTPError {
	var e struct {
		Error string `json:"error"`
	}
	_ = json.Unmarshal(body, &e)
	if strings.TrimSpace(e.Error) == "" {
		e.Error = http.StatusText(status)
	}
	return &AIRunHTTPError{Status: status, Message: e.Error, Body: body}
}

func validRequest(projectID string, req AIRunRequest, channel AIRunChannel) error {
	switch channel {
	case AIRunAPI, AIRunExtension, AIRunManual:
	default:
		return fmt.Errorf("%w: kênh %q", ErrBadAIRun, channel)
	}
	if req.Method != http.MethodPost && req.Method != http.MethodPut {
		return fmt.Errorf("%w: method %q", ErrBadAIRun, req.Method)
	}
	prefix := "/v1/projects/" + projectID + "/"
	if !strings.HasPrefix(req.Path, prefix) || strings.Contains(req.Path, "/ai-runs") || strings.ContainsAny(req.Path, "?#") {
		return fmt.Errorf("%w: đường dẫn %q không thuộc dự án này", ErrBadAIRun, req.Path)
	}
	if req.Body != "" && !json.Valid([]byte(req.Body)) {
		return fmt.Errorf("%w: thân yêu cầu không phải JSON", ErrBadAIRun)
	}
	return nil
}

// Start builds the prompt of the step's request as the step stands now and
// stores the run. On the API channel it then asks the system's model in the
// background; on the other two it waits for a reply. An open run of the same
// path is cancelled, so one AI action has one run going at a time.
func (s *AIRuns) Start(ctx context.Context, projectID, label string, req AIRunRequest, channel AIRunChannel) (AIRun, error) {
	if strings.TrimSpace(projectID) == "" {
		return AIRun{}, ErrNoAIRunProject
	}
	if err := validRequest(projectID, req, channel); err != nil {
		return AIRun{}, err
	}
	pr := NewExternalAIRun(ExternalAIPrompt, "")
	err := s.call(WithExternalAI(ctx, pr), req)
	prompt, ok := pr.Prompt()
	if !ok {
		if err == nil {
			err = fmt.Errorf("%w: yêu cầu này không gọi AI", ErrBadAIRun)
		}
		return AIRun{}, err
	}
	if err := s.cancelOpen(ctx, projectID, req.Path); err != nil {
		return AIRun{}, err
	}
	run := AIRun{ProjectID: projectID, Label: strings.TrimSpace(label), Channel: channel, Status: AIRunWaiting,
		Method: req.Method, Path: req.Path, Body: req.Body, Prompt: prompt}
	if channel == AIRunAPI {
		run.Status = AIRunRunning
	}
	run, err = s.store.CreateAIRun(ctx, run)
	if err != nil {
		return AIRun{}, err
	}
	if channel == AIRunAPI {
		started := run
		s.background(func() { s.capture(context.WithoutCancel(ctx), started) })
	}
	return run, nil
}

// cancelOpen cancels the open runs of path.
func (s *AIRuns) cancelOpen(ctx context.Context, projectID, path string) error {
	runs, err := s.store.LatestAIRuns(ctx, projectID)
	if err != nil {
		return err
	}
	for _, r := range runs {
		if r.Path == path && r.open() {
			r.Status = AIRunCancelled
			if err := s.store.SaveAIRun(ctx, r); err != nil {
				return err
			}
		}
	}
	return nil
}

// capture asks the system's model, then reads and checks the reply without
// storing it. A run cancelled meanwhile stays cancelled.
func (s *AIRuns) capture(ctx context.Context, run AIRun) {
	cr := NewExternalAIRun(ExternalAICapture, "")
	err := s.call(WithExternalAI(ctx, cr), run.request())
	reply, answered := cr.Captured()
	run.Reply = reply
	settle(&run, cr, err, answered)
	if cur, gerr := s.store.AIRun(ctx, run.ProjectID, run.ID); gerr == nil && cur.Status == AIRunCancelled {
		return
	}
	_ = s.store.SaveAIRun(ctx, run)
}

// settle sets the run's status from how reading its reply ended. A step that
// stopped at its preview point is ready whatever status its handler answered.
func settle(run *AIRun, er *ExternalAIRun, err error, answered bool) {
	run.Preview, run.Error = nil, ""
	if p, ok := er.Preview(); ok {
		run.Status, run.Preview = AIRunReady, p
		return
	}
	switch {
	case err == nil:
		run.Status, run.Error = AIRunFailed, "bước này chưa dừng được để xem trước kết quả"
	case answered:
		run.Status, run.Error = AIRunUnreadable, err.Error()
	default:
		run.Status, run.Error = AIRunFailed, err.Error()
	}
}

// Reply takes the reply of a run on the extension or manual channel, or a
// corrected reply of any run that is not running, applied or cancelled, and
// reads and checks it without storing it. A reply the step cannot use is kept
// with the reason, so the Creator can fix it and send it again.
func (s *AIRuns) Reply(ctx context.Context, projectID, id, reply string) (AIRun, error) {
	reply = strings.TrimSpace(reply)
	if reply == "" {
		return AIRun{}, fmt.Errorf("%w: câu trả lời trống", ErrBadAIRun)
	}
	run, err := s.store.AIRun(ctx, projectID, id)
	if err != nil {
		return AIRun{}, err
	}
	switch run.Status {
	case AIRunWaiting, AIRunReady, AIRunUnreadable, AIRunFailed:
	default:
		return AIRun{}, fmt.Errorf("%w: lượt đang %s", ErrAIRunState, run.Status)
	}
	pr := NewExternalAIRun(ExternalAIPreview, reply)
	err = s.call(WithExternalAI(ctx, pr), run.request())
	run.Reply = reply
	settle(&run, pr, err, true)
	if err := s.store.SaveAIRun(ctx, run); err != nil {
		return AIRun{}, err
	}
	return run, nil
}

// Apply stores a ready run's reply through the step's own request in paste
// mode, so the Creator's fields are kept as usual, and returns what the step
// answered (its screen). If the step changed since the preview and the reply no
// longer fits, the run becomes unreadable with the reason and nothing is stored.
func (s *AIRuns) Apply(ctx context.Context, projectID, id string) (AIRun, json.RawMessage, error) {
	run, err := s.store.AIRun(ctx, projectID, id)
	if err != nil {
		return AIRun{}, nil, err
	}
	if run.Status != AIRunReady {
		return AIRun{}, nil, fmt.Errorf("%w: chỉ áp dụng được kết quả đã đọc xong (lượt đang %s)", ErrAIRunState, run.Status)
	}
	status, body := s.replay.Replay(WithExternalAI(ctx, NewExternalAIRun(ExternalAIPaste, run.Reply)), run.request())
	if status < 200 || status >= 300 {
		refused := refusal(status, body)
		run.Status, run.Error, run.Preview = AIRunUnreadable, refused.Message, nil
		if serr := s.store.SaveAIRun(ctx, run); serr != nil {
			return AIRun{}, nil, serr
		}
		return run, nil, refused
	}
	run.Status, run.Error = AIRunApplied, ""
	if err := s.store.SaveAIRun(ctx, run); err != nil {
		return AIRun{}, nil, err
	}
	return run, json.RawMessage(body), nil
}

// Cancel drops a run that was not applied. A reply that arrives for it later
// is refused; a model call already under way finishes but is not kept.
func (s *AIRuns) Cancel(ctx context.Context, projectID, id string) (AIRun, error) {
	run, err := s.store.AIRun(ctx, projectID, id)
	if err != nil {
		return AIRun{}, err
	}
	if run.Status == AIRunApplied {
		return AIRun{}, fmt.Errorf("%w: lượt đã áp dụng", ErrAIRunState)
	}
	run.Status = AIRunCancelled
	if err := s.store.SaveAIRun(ctx, run); err != nil {
		return AIRun{}, err
	}
	return run, nil
}

// Get returns one run of the video.
func (s *AIRuns) Get(ctx context.Context, projectID, id string) (AIRun, error) {
	return s.store.AIRun(ctx, projectID, id)
}

// Latest returns the newest run of every request path of the video.
func (s *AIRuns) Latest(ctx context.Context, projectID string) ([]AIRun, error) {
	if strings.TrimSpace(projectID) == "" {
		return nil, ErrNoAIRunProject
	}
	return s.store.LatestAIRuns(ctx, projectID)
}
