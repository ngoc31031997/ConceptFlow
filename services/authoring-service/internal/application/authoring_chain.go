package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// ErrChainBusy is returned when a chain is already running for the project.
var ErrChainBusy = errors.New("a chain is already running for this project")

// ErrChainInvalid is returned for an empty or unknown step list.
var ErrChainInvalid = errors.New("chain steps must be a non-empty list of story, storyboard, code")

type authoringStepRunner interface {
	Execute(ctx context.Context, projectID, step string) (GeneratedStep, error)
}

// ChainState is what the GUI polls: which steps a chain runs, where it is, and
// how it ended. The last finished state stays until the next Start so a page
// that was closed mid-run finds the outcome when it reopens — the reason the
// chain lives here and not in the browser.
type ChainState struct {
	Running      bool     `json:"running"`
	Steps        []string `json:"steps"`
	CurrentIndex int      `json:"current_index"`
	Finished     bool     `json:"finished"`
	// Error is the Creator-facing reason the chain stopped; ErrorStep is where.
	Error     string `json:"error,omitempty"`
	ErrorStep string `json:"error_step,omitempty"`
	// Note is a non-fatal stop: content saved but the compile check still
	// fails, or the output could not be saved. Tokens were spent either way.
	Note       string     `json:"note,omitempty"`
	StartedAt  time.Time  `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
}

// AuthoringChainRunner runs authoring steps in order on the server, detached
// from any HTTP request. Each step already saves its own result (see
// GenerateAuthoringUseCase.Execute), so a step that fails leaves the earlier
// ones intact and the Creator resumes from the tab that stopped.
//
// State is in memory: a restart of the orchestrator forgets a finished chain
// (the saved content and the project journal remain).
type AuthoringChainRunner struct {
	runner  authoringStepRunner
	errText func(error) string

	mu     sync.Mutex
	chains map[string]*ChainState
}

// NewAuthoringChainRunner builds the runner. errText turns a failed run into
// the sentence shown to the Creator; nil falls back to err.Error().
func NewAuthoringChainRunner(runner authoringStepRunner, errText func(error) string) *AuthoringChainRunner {
	if errText == nil {
		errText = func(err error) string { return err.Error() }
	}
	return &AuthoringChainRunner{runner: runner, errText: errText, chains: map[string]*ChainState{}}
}

var chainSteps = map[string]bool{"story": true, "storyboard": true, "code": true}

// Start launches the chain and returns at once.
func (c *AuthoringChainRunner) Start(projectID string, steps []string) error {
	if projectID == "" || len(steps) == 0 {
		return ErrChainInvalid
	}
	for _, s := range steps {
		if !chainSteps[s] {
			return ErrChainInvalid
		}
	}
	c.mu.Lock()
	if cur := c.chains[projectID]; cur != nil && cur.Running {
		c.mu.Unlock()
		return ErrChainBusy
	}
	c.chains[projectID] = &ChainState{
		Running: true, Steps: append([]string(nil), steps...), StartedAt: time.Now().UTC(),
	}
	c.mu.Unlock()

	go c.run(projectID, append([]string(nil), steps...))
	return nil
}

func (c *AuthoringChainRunner) update(projectID string, fn func(*ChainState)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if st := c.chains[projectID]; st != nil {
		fn(st)
	}
}

func (c *AuthoringChainRunner) finish(projectID string, fn func(*ChainState)) {
	c.update(projectID, func(st *ChainState) {
		fn(st)
		now := time.Now().UTC()
		st.Running, st.Finished, st.FinishedAt = false, true, &now
	})
}

func (c *AuthoringChainRunner) run(projectID string, steps []string) {
	// Detached on purpose: the caller's request ends the moment Start returns.
	ctx := context.Background()
	defer func() {
		if r := recover(); r != nil {
			c.finish(projectID, func(st *ChainState) { st.Error = fmt.Sprintf("lỗi bất ngờ: %v", r) })
		}
	}()
	for i, step := range steps {
		c.update(projectID, func(st *ChainState) { st.CurrentIndex = i })
		out, err := c.runner.Execute(ctx, projectID, step)
		switch {
		case err != nil:
			c.finish(projectID, func(st *ChainState) { st.Error, st.ErrorStep = c.errText(err), step })
			return
		case out.CheckFailed:
			shown := out.Diagnostics
			if len(shown) > 5 {
				shown = shown[:5]
			}
			note := fmt.Sprintf("Đã sinh và lưu code nhưng vẫn lỗi biên dịch sau %d vòng sửa: %s", out.RepairRounds, strings.Join(shown, " · "))
			if extra := len(out.Diagnostics) - len(shown); extra > 0 {
				note += fmt.Sprintf(" (+%d lỗi nữa)", extra)
			}
			c.finish(projectID, func(st *ChainState) { st.Note = note + ". Sửa tay trong ô soạn thảo, hoặc chạy lại." })
			return
		case out.SaveError != "":
			c.finish(projectID, func(st *ChainState) { st.Note, st.ErrorStep = out.SaveError, step })
			return
		}
	}
	c.finish(projectID, func(st *ChainState) { st.CurrentIndex = len(steps) })
}

// State returns the current or last chain state; ok is false if none ran.
func (c *AuthoringChainRunner) State(projectID string) (ChainState, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	st := c.chains[projectID]
	if st == nil {
		return ChainState{}, false
	}
	out := *st
	out.Steps = append([]string(nil), st.Steps...)
	return out, true
}
