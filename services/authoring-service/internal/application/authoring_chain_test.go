package application

import (
	"context"
	"errors"
	"strings"
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

func TestChainCheckFailedNoteNamesTheKindOfIssueLeft(t *testing.T) {
	cases := []struct {
		name string
		out  GeneratedStep
		want []string
		not  []string
	}{
		{
			name: "layout only",
			out:  GeneratedStep{CheckFailed: true, Diagnostics: []string{"Shot 2.1: ra ngoài vùng an toàn"}, LayoutIssues: 1, RepairRounds: 2},
			want: []string{"còn 1 lỗi bố cục sau 2 vòng sửa", "không chặn render", "Shot 2.1: ra ngoài vùng an toàn"},
			not:  []string{"lỗi biên dịch"},
		},
		{
			name: "compile only",
			out:  GeneratedStep{CheckFailed: true, Diagnostics: []string{"dòng 4: TS2304"}, CompileIssues: 1, RepairRounds: 3},
			want: []string{"vẫn lỗi biên dịch sau 3 vòng sửa: dòng 4: TS2304"},
			not:  []string{"bố cục", "không chặn render"},
		},
		{
			name: "both",
			out:  GeneratedStep{CheckFailed: true, Diagnostics: []string{"dòng 4: TS2304", "Shot 3.1: lấn"}, CompileIssues: 1, LayoutIssues: 1, RepairRounds: 3},
			want: []string{"vẫn lỗi biên dịch", "trong đó 1 lỗi bố cục"},
			not:  []string{"không chặn render"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := NewAuthoringChainRunner(&fakeStepRunner{result: map[string]GeneratedStep{"code": tc.out}}, nil)
			_ = c.Start("p", []string{"code"})
			st := waitFinished(t, c, "p")
			for _, w := range tc.want {
				if !strings.Contains(st.Note, w) {
					t.Errorf("note %q lacks %q", st.Note, w)
				}
			}
			for _, n := range tc.not {
				if strings.Contains(st.Note, n) {
					t.Errorf("note %q must not say %q", st.Note, n)
				}
			}
		})
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

// blockingRunner holds Execute until its context is cancelled, like a model
// call that is still streaming.
type blockingRunner struct{ started chan struct{} }

func (b *blockingRunner) Execute(ctx context.Context, _, _ string) (GeneratedStep, error) {
	close(b.started)
	<-ctx.Done()
	return GeneratedStep{}, ctx.Err()
}

func TestChainCancelAbortsTheStepAndIsNotAnError(t *testing.T) {
	b := &blockingRunner{started: make(chan struct{})}
	c := NewAuthoringChainRunner(b, nil)
	if err := c.Start("p", []string{"story", "storyboard"}); err != nil {
		t.Fatal(err)
	}
	<-b.started
	if !c.Cancel("p") {
		t.Fatal("Cancel reported nothing running")
	}
	st := waitFinished(t, c, "p")
	if !st.Cancelled || st.Error != "" || st.Running || st.ErrorStep != "story" {
		t.Errorf("state %+v", st)
	}
	if c.Cancel("p") {
		t.Error("second Cancel should report nothing running")
	}
	// The project can start a fresh chain afterwards.
	f := &fakeStepRunner{}
	c.runner = f
	if err := c.Start("p", []string{"story"}); err != nil {
		t.Fatalf("restart after cancel: %v", err)
	}
}

func TestChainCancelUnknownProject(t *testing.T) {
	if NewAuthoringChainRunner(&fakeStepRunner{}, nil).Cancel("nope") {
		t.Error("Cancel of an unknown project must be false")
	}
}

// Drawings waiting for review stop the chain before Code — not an error.
func TestChainStopsForReviewAfterTheIllustrationsStep(t *testing.T) {
	f := &fakeStepRunner{result: map[string]GeneratedStep{
		StepIllustrations: {AwaitingReview: true, Message: "Còn 2 hình chờ bạn duyệt"},
	}}
	c := NewAuthoringChainRunner(f, nil)
	if err := c.Start("p", []string{"story", "storyboard", StepIllustrations, "code"}); err != nil {
		t.Fatal(err)
	}
	st := waitFinished(t, c, "p")
	if st.Error != "" || st.Waiting != "Còn 2 hình chờ bạn duyệt" || st.WaitingStep != StepIllustrations || len(f.ran) != 3 {
		t.Errorf("state %+v ran %v", st, f.ran)
	}
}

// Nothing to wait for when the storyboard is the last step, or
// has no warnings.
func TestChainDoesNotWaitOnAStoryboardItEndsWithOrThatHasNoWarnings(t *testing.T) {
	f := &fakeStepRunner{result: map[string]GeneratedStep{"storyboard": {Warnings: []string{"w"}}}}
	c := NewAuthoringChainRunner(f, nil)
	_ = c.Start("p", []string{"story", "storyboard"})
	if st := waitFinished(t, c, "p"); st.Waiting != "" || len(f.ran) != 2 {
		t.Fatalf("last step: state %+v ran %v", st, f.ran)
	}
	f2 := &fakeStepRunner{}
	c2 := NewAuthoringChainRunner(f2, nil)
	_ = c2.Start("p", []string{"storyboard", "code"})
	if st := waitFinished(t, c2, "p"); st.Waiting != "" || len(f2.ran) != 2 {
		t.Fatalf("no warnings: state %+v ran %v", st, f2.ran)
	}
}

func TestChainGoesOnToCodeWhenNoDrawingWaits(t *testing.T) {
	f := &fakeStepRunner{}
	c := NewAuthoringChainRunner(f, nil)
	_ = c.Start("p", []string{StepIllustrations, "code"})
	st := waitFinished(t, c, "p")
	if st.Waiting != "" || len(f.ran) != 2 || f.ran[1] != "code" {
		t.Errorf("state %+v ran %v", st, f.ran)
	}
}

// A step's warnings reach the polled state; a
// storyboard with warnings stops the chain before the steps after it.
func TestChainKeepsStepWarningsAndWaitsOnAFlaggedStoryboard(t *testing.T) {
	const warning = "Cảnh hook: ~20 giây, ngân sách 6–10 giây (+100%)"
	f := &fakeStepRunner{
		result: map[string]GeneratedStep{"storyboard": {Warnings: []string{warning}}},
	}
	c := NewAuthoringChainRunner(f, nil)
	_ = c.Start("p", []string{"story", "storyboard", StepIllustrations, "code"})
	st := waitFinished(t, c, "p")
	if st.Error != "" || st.WaitingStep != "storyboard" || !strings.Contains(st.Waiting, "1 cảnh báo") || len(f.ran) != 2 {
		t.Fatalf("state %+v ran %v, want a wait at the storyboard before any later step", st, f.ran)
	}
	if got := st.Warnings["storyboard"]; len(got) != 1 || got[0] != warning {
		t.Fatalf("storyboard warnings = %q", got)
	}
	if _, ok := st.Warnings["story"]; ok {
		t.Error("a step with no warnings must not get an entry")
	}
	// State hands out a copy: editing it must not change the runner's state.
	st.Warnings["storyboard"][0] = "changed"
	if again, _ := c.State("p"); again.Warnings["storyboard"][0] != warning {
		t.Error("State leaked its internal warnings slice")
	}
}

// codeOptsRunner also takes the code step's options.
type codeOptsRunner struct {
	fakeStepRunner
	opts []CodeRunOptions
}

func (f *codeOptsRunner) ExecuteCode(_ context.Context, _ string, opts CodeRunOptions) (GeneratedStep, error) {
	f.mu.Lock()
	f.opts = append(f.opts, opts)
	f.ran = append(f.ran, "code*")
	f.mu.Unlock()
	return GeneratedStep{}, nil
}

// A segment re-run and a fresh run go through the chain, so they
// survive the browser closing; without options the code step runs as before.
func TestChainCarriesTheCodeRunOptions(t *testing.T) {
	f := &codeOptsRunner{}
	c := NewAuthoringChainRunner(f, nil)
	if err := c.StartWith("p", []string{"code"}, CodeRunOptions{Segment: "1.4-1.6"}); err != nil {
		t.Fatal(err)
	}
	waitFinished(t, c, "p")
	if err := c.Start("p", []string{"code"}); err != nil {
		t.Fatal(err)
	}
	waitFinished(t, c, "p")
	if len(f.opts) != 1 || f.opts[0].Segment != "1.4-1.6" || strings.Join(f.ran, ",") != "code*,code" {
		t.Errorf("opts %+v ran %v", f.opts, f.ran)
	}
	for _, bad := range []struct {
		steps []string
		opts  CodeRunOptions
	}{
		{[]string{"storyboard", "code"}, CodeRunOptions{Fresh: true}},
		{[]string{"story"}, CodeRunOptions{Segment: "x"}},
		{[]string{"code"}, CodeRunOptions{Segment: "x", Fresh: true}},
	} {
		if err := c.StartWith("p", bad.steps, bad.opts); !errors.Is(err, ErrChainInvalid) {
			t.Errorf("%v %+v: %v", bad.steps, bad.opts, err)
		}
	}
	if err := NewAuthoringChainRunner(&fakeStepRunner{}, nil).StartWith("p", []string{"code"}, CodeRunOptions{Fresh: true}); err == nil {
		t.Error("a runner without ExecuteCode must refuse options rather than ignore them")
	}
}
