package domain

import (
	"fmt"
	"strings"
)

// ChainStep is one line of the approved story's chain of cause and effect, or
// its payoff line, with the fact codes it cites. Facts is empty for a line that
// cites none.
type ChainStep struct {
	Text  string
	Facts []string
}

// BeatOutline is the skeleton a sheet is built on: a hook, one beat for each
// step of the story's chain, and a payoff, inside a target length. The number
// of beats comes from the story, not from a model.
type BeatOutline struct {
	Steps []ChainStep
	// LoSeconds and HiSeconds bound the total length of the sheet.
	LoSeconds, HiSeconds int
}

// NewBeatOutline reads the chain text (one step per line, each possibly ending
// in "[F01, F02]") into an outline for a sheet of lo to hi seconds.
func NewBeatOutline(chain string, lo, hi int) (BeatOutline, error) {
	outline := BeatOutline{LoSeconds: lo, HiSeconds: hi}
	for _, line := range strings.Split(chain, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		body, refs, err := ParseRefLine(line, nil)
		if err != nil {
			return BeatOutline{}, err
		}
		outline.Steps = append(outline.Steps, ChainStep{Text: body, Facts: refs})
	}
	return outline, nil
}

// MinBeats is the number of beats of the skeleton: the hook, one beat per
// chain step and the payoff.
func (o BeatOutline) MinBeats() int { return len(o.Steps) + 2 }

// Feasible reports an outline no sheet can satisfy: more beats than a sheet
// holds, or more than fit in the target length at the shortest beat.
func (o BeatOutline) Feasible() error {
	if o.MinBeats() > MaxBeats {
		return fmt.Errorf("chuỗi nhân quả có %d bước, cần %d beat nhưng một bảng chỉ có tối đa %d", len(o.Steps), o.MinBeats(), MaxBeats)
	}
	if o.MinBeats()*MinBeatSeconds > o.HiSeconds {
		return fmt.Errorf("chuỗi nhân quả có %d bước, cần %d beat, mỗi beat từ %d giây trở lên, vượt thời lượng mục tiêu %s",
			len(o.Steps), o.MinBeats(), MinBeatSeconds, FormatSeconds(o.HiSeconds))
	}
	return nil
}

// Layout of the skeleton.
const (
	skeletonHookTitle   = "Mở đầu"
	skeletonPayoffTitle = "Câu trả lời"
	// skeletonTitleRunes is how many characters of a chain line a skeleton title keeps.
	skeletonTitleRunes = 48
	// provisionalHookSeconds is the length a skeleton gives the hook.
	provisionalHookSeconds = 15
)

// stepFn is the function of the i-th of n chain steps: the first opens the
// problem, the last is the climax, the rest explain the mechanism. The Creator
// reviews the choice when approving the skeleton.
func stepFn(i, n int) string {
	switch {
	case n >= 2 && i == 0:
		return "Đặt vấn đề"
	case n >= 2 && i == n-1:
		return "Cao trào"
	default:
		return "Giải thích cơ chế"
	}
}

func shortTitle(text string) string {
	r := []rune(strings.TrimSpace(text))
	if len(r) <= skeletonTitleRunes {
		return string(r)
	}
	return strings.TrimSpace(string(r[:skeletonTitleRunes])) + "…"
}

// Skeleton builds the beats of the outline: a Hook from the story's question,
// one beat per chain step carrying the step's facts, and a Payoff from the
// payoff line. Goal and content are empty until the content is written, and
// the lengths are an even provisional split the content run replaces.
func (o BeatOutline) Skeleton(question, payoff ChainStep) ([]Beat, error) {
	if err := o.Feasible(); err != nil {
		return nil, err
	}
	n := o.MinBeats()
	rest := MinBeatSeconds
	if share := ((o.LoSeconds+o.HiSeconds)/2 - provisionalHookSeconds) / (n - 1); share > rest {
		rest = share / BeatDurationStep * BeatDurationStep
	}
	beats := make([]Beat, 0, n)
	beats = append(beats, Beat{Fn: BeatFns[0], Title: skeletonHookTitle, Facts: []string{}, Dur: provisionalHookSeconds, Step: question.Text})
	for i, s := range o.Steps {
		facts := append([]string{}, s.Facts...)
		beats = append(beats, Beat{Fn: stepFn(i, len(o.Steps)), Title: shortTitle(s.Text), Facts: facts, Dur: rest, Step: s.Text})
	}
	beats = append(beats, Beat{Fn: beatFnPayoff, Title: skeletonPayoffTitle, Facts: append([]string{}, payoff.Facts...), Dur: rest, Step: payoff.Text})
	return beats, nil
}

// CheckSkeleton runs the checks that apply before any content is written: a
// Hook first, a Payoff last and every fact the script must state in some beat.
func CheckSkeleton(beats []Beat, mustFacts []string) []BeatCheck {
	used := map[string]bool{}
	for _, b := range beats {
		for _, id := range b.Facts {
			used[id] = true
		}
	}
	var missing []string
	for _, id := range mustFacts {
		if !used[id] {
			missing = append(missing, id)
		}
	}
	mustLine := fmt.Sprintf("Dữ kiện bắt buộc có trong beat: %d/%d", len(mustFacts)-len(missing), len(mustFacts))
	if len(missing) > 0 {
		mustLine += " · thiếu " + strings.Join(missing, ", ")
	}
	first, last := "", ""
	if len(beats) > 0 {
		first, last = beats[0].Fn, beats[len(beats)-1].Fn
	}
	return []BeatCheck{
		{OK: first == BeatFns[0], Text: "Beat đầu là Hook"},
		{OK: last == beatFnPayoff, Text: "Beat cuối là Payoff"},
		{OK: len(missing) == 0, Text: mustLine},
	}
}

// BalanceSeconds returns the lengths in beats with their sum inside lo to hi.
// Lengths whose sum already fits are returned as they are. Otherwise every
// beat not in fixed is scaled by one factor toward the nearest bound, rounded
// to BeatDurationStep and kept within MinBeatSeconds and MaxBeatSeconds, the
// opening Hook within MaxHookSeconds, and the rounding leftover is moved in
// whole steps from the longest or onto the shortest free beat. fixed marks
// the beats whose length the Creator set. It fails when no lengths fit.
func BalanceSeconds(beats []Beat, fixed []bool, lo, hi int) ([]int, error) {
	out := make([]int, len(beats))
	total := 0
	for i, b := range beats {
		out[i] = b.Dur
		total += b.Dur
	}
	limit := func(i int) int {
		if i == 0 && beats[0].Fn == BeatFns[0] {
			return MaxHookSeconds
		}
		return MaxBeatSeconds
	}
	if total >= lo && total <= hi {
		return out, nil
	}
	target := lo
	if total > hi {
		target = hi
	}
	fixedSum, freeSum, free := 0, 0, 0
	for i := range beats {
		if fixed[i] {
			fixedSum += out[i]
			continue
		}
		freeSum += out[i]
		free++
	}
	if free == 0 || target-fixedSum < free*MinBeatSeconds {
		return nil, fmt.Errorf("không có cách chia thời lượng để tổng nằm trong %s đến %s", FormatSeconds(lo), FormatSeconds(hi))
	}
	scale := float64(target-fixedSum) / float64(freeSum)
	sum := fixedSum
	for i := range beats {
		if fixed[i] {
			continue
		}
		v := int(float64(out[i])*scale/float64(BeatDurationStep)+0.5) * BeatDurationStep
		out[i] = min(max(v, MinBeatSeconds), limit(i))
		sum += out[i]
	}
	for sum > hi {
		at := -1
		for i := range out {
			if !fixed[i] && out[i]-BeatDurationStep >= MinBeatSeconds && (at < 0 || out[i] > out[at]) {
				at = i
			}
		}
		if at < 0 {
			return nil, fmt.Errorf("không có cách chia thời lượng để tổng nằm trong %s đến %s", FormatSeconds(lo), FormatSeconds(hi))
		}
		out[at] -= BeatDurationStep
		sum -= BeatDurationStep
	}
	for sum < lo {
		at := -1
		for i := range out {
			if !fixed[i] && out[i]+BeatDurationStep <= limit(i) && (at < 0 || out[i] < out[at]) {
				at = i
			}
		}
		if at < 0 {
			return nil, fmt.Errorf("không có cách chia thời lượng để tổng nằm trong %s đến %s", FormatSeconds(lo), FormatSeconds(hi))
		}
		out[at] += BeatDurationStep
		sum += BeatDurationStep
	}
	return out, nil
}
