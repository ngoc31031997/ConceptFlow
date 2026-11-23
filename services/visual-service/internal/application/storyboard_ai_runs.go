package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// AIRunTask names the AI turn of step 11 a run makes.
type AIRunTask string

const (
	// AIRunAssets lists the video's assets (one turn, no target).
	AIRunAssets AIRunTask = "assets"
	// AIRunAssetDetail writes the detail of the asset named by Target.
	AIRunAssetDetail AIRunTask = "asset_detail"
	// AIRunUnit writes the shots of the plan unit named by Target.
	AIRunUnit AIRunTask = "unit"
	// AIRunReview reviews the whole storyboard (one turn, no target).
	AIRunReview AIRunTask = "review"
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
	// AIRunApplied had its reply stored into the storyboard.
	AIRunApplied AIRunStatus = "applied"
	// AIRunCancelled was dropped by the Creator or replaced by a newer run of the same task.
	AIRunCancelled AIRunStatus = "cancelled"
)

// AIRunInterrupted is the error of a run that was asking the model when
// visual-service stopped.
const AIRunInterrupted = "visual-service khởi động lại khi đang chờ AI; chạy lại lượt này."

var (
	// ErrUnknownAIRun is a run id the video does not have.
	ErrUnknownAIRun = errors.New("không có lượt chạy AI này")
	// ErrAIRunState is an action the run's status does not allow.
	ErrAIRunState = errors.New("lượt chạy AI không ở trạng thái cho phép việc này")
	// ErrBadAIRun is a run request with an unknown task or channel, or a missing target.
	ErrBadAIRun = errors.New("yêu cầu chạy AI không hợp lệ")
)

// AIRun is one AI turn of step 11 run through the AI run dialog: its prompt is
// built when it starts, its reply is read and checked without being stored
// (Preview), and it is stored only when the Creator applies it.
type AIRun struct {
	ID        string          `json:"id"`
	ProjectID string          `json:"project_id"`
	Task      AIRunTask       `json:"task"`
	Target    string          `json:"target"`
	Note      string          `json:"note"`
	Channel   AIRunChannel    `json:"channel"`
	Status    AIRunStatus     `json:"status"`
	Prompt    string          `json:"prompt"`
	Reply     string          `json:"reply"`
	Preview   json.RawMessage `json:"preview"`
	Error     string          `json:"error"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// open reports whether the run still waits for something: a reply or an apply.
func (r AIRun) open() bool {
	switch r.Status {
	case AIRunWaiting, AIRunRunning, AIRunReady, AIRunUnreadable:
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
	// LatestAIRuns returns the newest run of every task and target of the video.
	LatestAIRuns(ctx context.Context, projectID string) ([]AIRun, error)
}

// StoryboardAIRuns runs the four single-turn AI tasks of step 11 with a preview:
// the reply comes from the system's model, the extension or a paste, is read
// and checked by the same flow that stores it, and is stored only on Apply.
type StoryboardAIRuns struct {
	desk  *StoryboardDesk
	store AIRunStorePort
	// background starts the call to the system's model; tests run it inline.
	background func(func())
}

// NewStoryboardAIRuns wires the runs over the desk.
func NewStoryboardAIRuns(desk *StoryboardDesk, store AIRunStorePort) *StoryboardAIRuns {
	return &StoryboardAIRuns{desk: desk, store: store, background: func(f func()) { go f() }}
}

// call runs the task's flow under ctx, which carries the external-AI mode.
func (s *StoryboardAIRuns) call(ctx context.Context, run AIRun) error {
	var err error
	switch run.Task {
	case AIRunAssets:
		_, err = s.desk.GenerateAssets(ctx, run.ProjectID, run.Note)
	case AIRunAssetDetail:
		_, err = s.desk.GenerateAssetDetail(ctx, run.ProjectID, run.Target, run.Note)
	case AIRunUnit:
		_, err = s.desk.GenerateUnit(ctx, run.ProjectID, run.Target, run.Note)
	case AIRunReview:
		_, err = s.desk.Review(ctx, run.ProjectID, run.Note)
	default:
		err = ErrBadAIRun
	}
	return err
}

func validRun(task AIRunTask, target string, channel AIRunChannel) error {
	switch channel {
	case AIRunAPI, AIRunExtension, AIRunManual:
	default:
		return fmt.Errorf("%w: kênh %q", ErrBadAIRun, channel)
	}
	switch task {
	case AIRunAssets, AIRunReview:
		if target != "" {
			return fmt.Errorf("%w: việc %q không nhận mục", ErrBadAIRun, task)
		}
	case AIRunAssetDetail, AIRunUnit:
		if target == "" {
			return fmt.Errorf("%w: việc %q cần mã mục", ErrBadAIRun, task)
		}
	default:
		return fmt.Errorf("%w: việc %q", ErrBadAIRun, task)
	}
	return nil
}

// Start builds the prompt of the task as the step stands now and stores the
// run. On the API channel it then asks the system's model in the background;
// on the other two it waits for a reply. An open run of the same task and
// target is cancelled, so one task has one run going at a time.
func (s *StoryboardAIRuns) Start(ctx context.Context, projectID string, task AIRunTask, target, note string, channel AIRunChannel) (AIRun, error) {
	if strings.TrimSpace(projectID) == "" {
		return AIRun{}, ErrNoProjectID
	}
	target, note = strings.TrimSpace(target), strings.TrimSpace(note)
	if err := validRun(task, target, channel); err != nil {
		return AIRun{}, err
	}
	run := AIRun{ProjectID: projectID, Task: task, Target: target, Note: note, Channel: channel}
	pr := NewExternalAIRun(ExternalAIPrompt, "")
	err := s.call(WithExternalAI(ctx, pr), run)
	prompt, ok := pr.Prompt()
	if !errors.Is(err, ErrExternalAIPromptBuilt) || !ok {
		if err == nil {
			err = fmt.Errorf("%w: việc %q không dựng prompt", ErrBadAIRun, task)
		}
		return AIRun{}, err
	}
	if err := s.cancelOpen(ctx, projectID, task, target); err != nil {
		return AIRun{}, err
	}
	run.Prompt = prompt
	run.Status = AIRunWaiting
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

// cancelOpen cancels the open runs of task and target.
func (s *StoryboardAIRuns) cancelOpen(ctx context.Context, projectID string, task AIRunTask, target string) error {
	runs, err := s.store.LatestAIRuns(ctx, projectID)
	if err != nil {
		return err
	}
	for _, r := range runs {
		if r.Task == task && r.Target == target && r.open() {
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
func (s *StoryboardAIRuns) capture(ctx context.Context, run AIRun) {
	cr := NewExternalAIRun(ExternalAICapture, "")
	err := s.call(WithExternalAI(ctx, cr), run)
	reply, answered := cr.Captured()
	run.Reply = reply
	s.settle(&run, cr, err, answered)
	if cur, gerr := s.store.AIRun(ctx, run.ProjectID, run.ID); gerr == nil && cur.Status == AIRunCancelled {
		return
	}
	_ = s.store.SaveAIRun(ctx, run)
}

// settle sets the run's status from how reading its reply ended.
func (s *StoryboardAIRuns) settle(run *AIRun, er *ExternalAIRun, err error, answered bool) {
	run.Preview, run.Error = nil, ""
	switch {
	case errors.Is(err, ErrExternalAIPreviewReady):
		run.Status = AIRunReady
		run.Preview, _ = er.Preview()
	case err == nil:
		run.Status, run.Error = AIRunFailed, "bước không trả kết quả để xem trước"
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
func (s *StoryboardAIRuns) Reply(ctx context.Context, projectID, id, reply string) (AIRun, error) {
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
	err = s.call(WithExternalAI(ctx, pr), run)
	run.Reply = reply
	s.settle(&run, pr, err, true)
	if err := s.store.SaveAIRun(ctx, run); err != nil {
		return AIRun{}, err
	}
	return run, nil
}

// Apply stores a ready run's reply into the storyboard through the same flow a
// pasted reply takes, so the Creator's fields are kept and frames are drawn as
// usual. If the step changed since the preview and the reply no longer fits,
// the run becomes unreadable with the reason and nothing is stored.
func (s *StoryboardAIRuns) Apply(ctx context.Context, projectID, id string) (AIRun, *StoryboardScreen, error) {
	run, err := s.store.AIRun(ctx, projectID, id)
	if err != nil {
		return AIRun{}, nil, err
	}
	if run.Status != AIRunReady {
		return AIRun{}, nil, fmt.Errorf("%w: chỉ áp dụng được kết quả đã đọc xong (lượt đang %s)", ErrAIRunState, run.Status)
	}
	if err := s.call(WithExternalAI(ctx, NewExternalAIRun(ExternalAIPaste, run.Reply)), run); err != nil {
		run.Status, run.Error, run.Preview = AIRunUnreadable, err.Error(), nil
		if serr := s.store.SaveAIRun(ctx, run); serr != nil {
			return AIRun{}, nil, serr
		}
		return run, nil, err
	}
	run.Status, run.Error = AIRunApplied, ""
	if err := s.store.SaveAIRun(ctx, run); err != nil {
		return AIRun{}, nil, err
	}
	st, err := s.desk.State(ctx, projectID)
	return run, st, err
}

// Cancel drops a run that was not applied. A reply that arrives for it later
// is refused; a model call already under way finishes but is not kept.
func (s *StoryboardAIRuns) Cancel(ctx context.Context, projectID, id string) (AIRun, error) {
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
func (s *StoryboardAIRuns) Get(ctx context.Context, projectID, id string) (AIRun, error) {
	return s.store.AIRun(ctx, projectID, id)
}

// Latest returns the newest run of every task and target of the video.
func (s *StoryboardAIRuns) Latest(ctx context.Context, projectID string) ([]AIRun, error) {
	if strings.TrimSpace(projectID) == "" {
		return nil, ErrNoProjectID
	}
	return s.store.LatestAIRuns(ctx, projectID)
}
