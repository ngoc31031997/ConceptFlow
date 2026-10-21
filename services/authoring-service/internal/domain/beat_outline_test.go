package domain

import (
	"strings"
	"testing"
)

const skyChain = "Ánh sáng gồm nhiều màu. [F03]\nXanh bị tán xạ mạnh. [F02, F01]\nMắt người nhạy với xanh."

func TestNewBeatOutlineReadsOneStepPerLineWithItsFacts(t *testing.T) {
	o, err := NewBeatOutline(skyChain, 240, 300)
	if err != nil {
		t.Fatalf("NewBeatOutline: %v", err)
	}
	if len(o.Steps) != 3 || o.MinBeats() != 5 {
		t.Fatalf("steps = %d min = %d, want 3 steps and a hook, three beats and a payoff", len(o.Steps), o.MinBeats())
	}
	if strings.Join(o.Steps[1].Facts, ",") != "F02,F01" || o.Steps[1].Text != "Xanh bị tán xạ mạnh." {
		t.Errorf("step 2 = %+v", o.Steps[1])
	}
	if len(o.Steps[2].Facts) != 0 {
		t.Errorf("step 3 cites %v, want no fact", o.Steps[2].Facts)
	}
}

func TestBeatOutlineFeasibleRefusesWhatNoSheetCanHold(t *testing.T) {
	long := strings.Repeat("Một bước.\n", MaxStoryChain)
	if o, _ := NewBeatOutline(long, 0, 60); o.Feasible() == nil {
		t.Error("14 beats of at least 5 s cannot fit 60 s")
	}
	if o, _ := NewBeatOutline(strings.Repeat("Một bước.\n", MaxBeats), 240, 300); o.Feasible() == nil {
		t.Error("22 beats cannot fit a sheet")
	}
	if o, _ := NewBeatOutline(skyChain, 240, 300); o.Feasible() != nil {
		t.Error("a 5 beat outline in 4–5 minutes is fine")
	}
}

func TestSkeletonIsAHookOneBeatPerStepAndAPayoff(t *testing.T) {
	o, _ := NewBeatOutline(skyChain, 240, 300)
	beats, err := o.Skeleton(ChainStep{Text: "Vì sao trời xanh?"}, ChainStep{Text: "Vì tán xạ.", Facts: []string{"F01"}})
	if err != nil {
		t.Fatalf("Skeleton: %v", err)
	}
	var fns []string
	for _, b := range beats {
		fns = append(fns, b.Fn)
		if b.Written() {
			t.Errorf("beat %q is written, want a skeleton beat with no goal or content", b.Title)
		}
	}
	if strings.Join(fns, "|") != "Hook|Đặt vấn đề|Giải thích cơ chế|Cao trào|Payoff" {
		t.Errorf("functions = %v", fns)
	}
	if beats[0].Step != "Vì sao trời xanh?" || strings.Join(beats[2].Facts, ",") != "F02,F01" || strings.Join(beats[4].Facts, ",") != "F01" || len(beats[3].Facts) != 0 {
		t.Errorf("beats = %+v, want each step's text and facts carried over", beats)
	}
	for _, b := range beats {
		if err := b.Validate(nil); err != nil {
			t.Errorf("skeleton beat %q is not valid: %v", b.Title, err)
		}
	}
	if beats[0].Dur != provisionalHookSeconds {
		t.Errorf("hook = %d s, want %d", beats[0].Dur, provisionalHookSeconds)
	}
}

func TestSkeletonKeepsLongTitlesShort(t *testing.T) {
	o, _ := NewBeatOutline(strings.Repeat("rất dài ", 30), 240, 300)
	beats, _ := o.Skeleton(ChainStep{}, ChainStep{})
	if got := []rune(beats[1].Title); len(got) > skeletonTitleRunes+1 {
		t.Errorf("title has %d characters, want at most %d and an ellipsis", len(got), skeletonTitleRunes)
	}
}

func TestSkeletonStepFnForShortChains(t *testing.T) {
	o, _ := NewBeatOutline("Chỉ một bước. [F01]", 240, 300)
	beats, _ := o.Skeleton(ChainStep{}, ChainStep{})
	if beats[1].Fn != "Giải thích cơ chế" {
		t.Errorf("single step function = %q", beats[1].Fn)
	}
}

func TestCheckSkeletonNamesWhatIsMissing(t *testing.T) {
	beats := []Beat{{Fn: "Giải thích cơ chế", Facts: []string{"F01"}}, {Fn: "Cao trào", Facts: []string{}}}
	checks := CheckSkeleton(beats, []string{"F01", "F02"})
	if checks[0].OK || checks[1].OK || checks[2].OK || !strings.Contains(checks[2].Text, "thiếu F02") {
		t.Errorf("checks = %+v, want no Hook, no Payoff and F02 missing", checks)
	}
	ok := CheckSkeleton([]Beat{{Fn: "Hook"}, {Fn: "Payoff", Facts: []string{"F01"}}}, []string{"F01"})
	if !ok[0].OK || !ok[1].OK || !ok[2].OK {
		t.Errorf("checks = %+v, want all three to pass", ok)
	}
}

func sheetOf(durs ...int) []Beat {
	beats := make([]Beat, len(durs))
	for i, d := range durs {
		beats[i] = Beat{Fn: "Giải thích cơ chế", Dur: d}
	}
	beats[0].Fn = "Hook"
	return beats
}

func total(durs []int) int {
	sum := 0
	for _, d := range durs {
		sum += d
	}
	return sum
}

func TestBalanceSecondsLeavesAFittingSheetAlone(t *testing.T) {
	got, err := BalanceSeconds(sheetOf(18, 123, 77), []bool{false, false, false}, 200, 300)
	if err != nil || got[0] != 18 || got[1] != 123 || got[2] != 77 {
		t.Errorf("got %v, %v, want the lengths unchanged", got, err)
	}
}

func TestBalanceSecondsScalesAnOverlongSheetIntoTheTarget(t *testing.T) {
	got, err := BalanceSeconds(sheetOf(20, 500, 200, 100), []bool{false, false, false, false}, 240, 300)
	if err != nil {
		t.Fatal(err)
	}
	if sum := total(got); sum < 240 || sum > 300 {
		t.Errorf("total = %d, want 240–300", sum)
	}
	for i, d := range got {
		if d%BeatDurationStep != 0 || d < MinBeatSeconds {
			t.Errorf("beat %d = %d s, want a multiple of %d and at least %d", i, d, BeatDurationStep, MinBeatSeconds)
		}
	}
	if got[1] <= got[2] || got[2] <= got[3] {
		t.Errorf("got %v, want the order of lengths kept", got)
	}
}

func TestBalanceSecondsStretchesAShortSheetAndCapsTheHook(t *testing.T) {
	got, err := BalanceSeconds(sheetOf(15, 10, 10), []bool{false, false, false}, 240, 300)
	if err != nil {
		t.Fatal(err)
	}
	if sum := total(got); sum < 240 || sum > 300 {
		t.Errorf("total = %d, want 240–300", sum)
	}
	if got[0] > MaxHookSeconds {
		t.Errorf("hook = %d s, want at most %d", got[0], MaxHookSeconds)
	}
}

func TestBalanceSecondsKeepsTheLengthsTheCreatorSet(t *testing.T) {
	got, err := BalanceSeconds(sheetOf(20, 100, 300), []bool{false, true, false}, 240, 300)
	if err != nil {
		t.Fatal(err)
	}
	if got[1] != 100 {
		t.Errorf("fixed beat = %d s, want 100 unchanged", got[1])
	}
	if sum := total(got); sum < 240 || sum > 300 {
		t.Errorf("total = %d, want 240–300", sum)
	}
}

func TestBalanceSecondsFailsWhenNoLengthsFit(t *testing.T) {
	if _, err := BalanceSeconds(sheetOf(15, 30, 30), []bool{true, true, true}, 10, 20); err == nil {
		t.Error("every beat fixed and over the target must fail")
	}
	if _, err := BalanceSeconds(sheetOf(20, 300, 300), []bool{false, true, false}, 240, 300); err == nil {
		t.Error("a fixed beat that takes the whole target leaves no room for the others")
	}
	if _, err := BalanceSeconds(sheetOf(20, 100, 100, 100), []bool{false, false, false, false}, 0, 10); err == nil {
		t.Error("four beats of at least 5 s cannot fit 10 s")
	}
}
