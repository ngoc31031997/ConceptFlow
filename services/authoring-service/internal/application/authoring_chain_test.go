package application

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type fakeStepRunner struct {
	mu      sync.Mutex
	ran     []string
	failAt  string
	release chan struct{}
	result  map[string]GeneratedStep
}

func (f *fakeStepRunner) Execute(_ context.Context, _ string, step string) (GeneratedStep, error) {
	if f.release != nil {
		<-f.release
	}
	f.mu.Lock()
	f.ran = append(f.ran, step)
	f.mu.Unlock()
	if step == f.failAt {
		return GeneratedStep{}, errors.New("boom")
	}
	return f.result[step], nil
}

func waitFinished(t *testing.T, c *AuthoringChainRunner, id string) ChainState {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if st, ok := c.State(id); ok && st.Finished {
			return st
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("chain did not finish")
	return ChainState{}
}

func TestChainRunsStepsInOrder(t *testing.T) {
	f := &fakeStepRunner{}
	c := NewAuthoringChainRunner(f, nil)
	if err := c.Start("p", []string{"story", "storyboard", "code"}); err != nil {
		t.Fatal(err)
	}
	st := waitFinished(t, c, "p")
	if st.Error != "" || st.Running || len(f.ran) != 3 || f.ran[0] != "story" || f.ran[2] != "code" {
		t.Errorf("state %+v ran %v", st, f.ran)
	}
}

func TestChainStopsAtFailureAndKeepsReason(t *testing.T) {
	f := &fakeStepRunner{failAt: "storyboard"}
	c := NewAuthoringChainRunner(f, func(err error) string { return "vì " + err.Error() })
	_ = c.Start("p", []string{"story", "storyboard", "code"})
	st := waitFinished(t, c, "p")
	if st.ErrorStep != "storyboard" || st.Error != "vì boom" || len(f.ran) != 2 {
		t.Errorf("state %+v ran %v", st, f.ran)
	}
	// The outcome is still there for a page that reopens later.
	if again, ok := c.State("p"); !ok || again.Error == "" {
		t.Error("finished state must persist until the next Start")
	}
}

func TestChainCheckFailedIsANoteNotAnError(t *testing.T) {
	f := &fakeStepRunner{result: map[string]GeneratedStep{"code": {CheckFailed: true, Diagnostics: []string{"x"}, RepairRounds: 2}}}
	c := NewAuthoringChainRunner(f, nil)
	_ = c.Start("p", []string{"code"})
	st := waitFinished(t, c, "p")
	if st.Error != "" || st.Note == "" {
		t.Errorf("state %+v", st)
	}
}

func TestChainRejectsSecondStartWhileRunning(t *testing.T) {
	f := &fakeStepRunner{release: make(chan struct{})}
	c := NewAuthoringChainRunner(f, nil)
	if err := c.Start("p", []string{"story"}); err != nil {
		t.Fatal(err)
	}
	if err := c.Start("p", []string{"story"}); !errors.Is(err, ErrChainBusy) {
		t.Errorf("got %v, want ErrChainBusy", err)
	}
	close(f.release)
	waitFinished(t, c, "p")
}

func TestChainValidatesSteps(t *testing.T) {
	c := NewAuthoringChainRunner(&fakeStepRunner{}, nil)
	for _, steps := range [][]string{nil, {}, {"nope"}} {
		if err := c.Start("p", steps); !errors.Is(err, ErrChainInvalid) {
			t.Errorf("%v: got %v", steps, err)
		}
	}
}
