package domain

import (
	"strings"
	"testing"
)

func chapters(indices ...int) []Chapter {
	out := make([]Chapter, 0, len(indices))
	for i, idx := range indices {
		out = append(out, Chapter{SceneIndex: idx, Title: string(rune('A' + i))})
	}
	return out
}

func TestBuildChapterTimestamps_UsesMeasuredOffsets(t *testing.T) {
	// The offsets are what Rendering measured, not a running sum of narration
	// durations — that is the whole reason chapters can be trusted (CR-002).
	offsets := []float64{0, 40, 95, 160}

	lines := BuildChapterTimestamps(chapters(0, 1, 2, 3), offsets, 200, 0.0)

	want := []string{"0:00 A", "0:40 B", "1:35 C", "2:40 D"}
	if len(lines) != len(want) {
		t.Fatalf("expected %d lines, got %v", len(want), lines)
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Fatalf("line %d: expected %q, got %q", i, want[i], lines[i])
		}
	}
}

func TestBuildChapterTimestamps_ForcesTheFirstChapterToZero(t *testing.T) {
	// YouTube ignores a chapter list whose first entry is not 00:00. A Creator
	// who marked their first chapter slightly in still meant it to open the
	// video, so it is pulled back rather than the whole list discarded.
	lines := BuildChapterTimestamps(chapters(0, 1, 2), []float64{12, 60, 120}, 200, 0.0)

	if len(lines) != 3 || !strings.HasPrefix(lines[0], "0:00 ") {
		t.Fatalf("expected the first chapter at 0:00, got %v", lines)
	}
}

func TestBuildChapterTimestamps_RejectsFewerThanThree(t *testing.T) {
	// Better to publish no chapters than a list YouTube will silently drop.
	if lines := BuildChapterTimestamps(chapters(0, 1), []float64{0, 60}, 200, 0.0); lines != nil {
		t.Fatalf("expected no chapters for a 2-entry list, got %v", lines)
	}
}

func TestBuildChapterTimestamps_RejectsChaptersShorterThanTenSeconds(t *testing.T) {
	lines := BuildChapterTimestamps(chapters(0, 1, 2), []float64{0, 60, 65}, 200, 0.0)

	if lines != nil {
		t.Fatalf("expected no chapters when one runs under 10s, got %v", lines)
	}
}

func TestBuildChapterTimestamps_RejectsAnUnderLengthFinalChapter(t *testing.T) {
	// The last chapter's length is bounded by the video, not by a next entry.
	lines := BuildChapterTimestamps(chapters(0, 1, 2), []float64{0, 60, 120}, 125, 0.0)

	if lines != nil {
		t.Fatalf("expected no chapters when the last runs under 10s, got %v", lines)
	}
}

func TestBuildChapterTimestamps_RejectsOutOfOrderMarkers(t *testing.T) {
	lines := BuildChapterTimestamps(chapters(0, 2, 1), []float64{0, 60, 120}, 200, 0.0)

	if lines != nil {
		t.Fatalf("expected no chapters for a non-monotonic list, got %v", lines)
	}
}

func TestBuildChapterTimestamps_RejectsAnIndexBeyondTheOffsets(t *testing.T) {
	// A chapter pointing past the narration means the script and the render
	// disagree; there is no timestamp to give it.
	lines := BuildChapterTimestamps(chapters(0, 1, 9), []float64{0, 60, 120}, 200, 0.0)

	if lines != nil {
		t.Fatalf("expected no chapters for an out-of-range index, got %v", lines)
	}
}

// --- CR-023 D6: introDuration > 0 ---

func TestBuildChapterTimestamps_WithIntro_PrependsSyntheticIntroChapter(t *testing.T) {
	// The intro itself must be >= MinChapterSeconds to survive as its own
	// chapter, so use 15s; videoSeconds already includes it (caller's job per
	// D6): 15 (intro) + 200 (main) = 215.
	offsets := []float64{0, 40, 95}
	introDuration := 15.0
	videoSeconds := 215.0

	lines := BuildChapterTimestamps(chapters(0, 1, 2), offsets, videoSeconds, introDuration)

	want := []string{"0:00 Intro", "0:15 A", "0:55 B", "1:50 C"}
	if len(lines) != len(want) {
		t.Fatalf("expected %d lines, got %v", len(want), lines)
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Fatalf("line %d: expected %q, got %q", i, want[i], lines[i])
		}
	}
}

func TestBuildChapterTimestamps_WithIntro_DoesNotForceOtherChaptersToZero(t *testing.T) {
	// Only the synthetic Intro chapter sits at 0:00 — the original first
	// marker keeps its shifted (not zeroed) start.
	lines := BuildChapterTimestamps(chapters(0, 1, 2), []float64{12, 60, 120}, 220, 15.0)

	if len(lines) != 4 {
		t.Fatalf("expected 4 lines (synthetic intro + 3 markers), got %v", lines)
	}
	if lines[0] != "0:00 Intro" {
		t.Fatalf("expected synthetic intro chapter at 0:00, got %q", lines[0])
	}
	if !strings.HasPrefix(lines[1], "0:27 ") { // 12 + 15
		t.Fatalf("expected the first Creator chapter shifted (not zeroed), got %v", lines)
	}
}

func TestBuildChapterTimestamps_WithIntro_ValidatesMinLengthAgainstShiftedTotal(t *testing.T) {
	// The intro chapter itself must satisfy MinChapterSeconds against the
	// *shifted* start of the next chapter, using the total duration the
	// caller passed in (which already includes intro+outro per D6).
	introDuration := 3.0 // under MinChapterSeconds — the Intro chapter is only 3s long
	lines := BuildChapterTimestamps(chapters(0, 1, 2), []float64{0, 60, 120}, 200+introDuration, introDuration)

	if lines != nil {
		t.Fatalf("expected no chapters when the synthetic Intro chapter runs under 10s, got %v", lines)
	}
}

func TestBuildChapterTimestamps_ZeroIntroDuration_IsByteIdenticalToOldBehaviour(t *testing.T) {
	offsets := []float64{0, 40, 95, 160}

	withZero := BuildChapterTimestamps(chapters(0, 1, 2, 3), offsets, 200, 0.0)
	forced := BuildChapterTimestamps(chapters(0, 1, 2, 3), offsets, 200, 0)

	if len(withZero) != len(forced) {
		t.Fatalf("expected identical output for introDuration=0.0 vs 0, got %v vs %v", withZero, forced)
	}
	for i := range withZero {
		if withZero[i] != forced[i] {
			t.Fatalf("line %d differs: %q vs %q", i, withZero[i], forced[i])
		}
	}
}

func TestFormatTimestamp_SwitchesToHoursOnlyWhenNeeded(t *testing.T) {
	cases := map[float64]string{
		0:    "0:00",
		9:    "0:09",
		95:   "1:35",
		3599: "59:59",
		3600: "1:00:00",
		3725: "1:02:05",
	}
	for seconds, want := range cases {
		if got := FormatTimestamp(seconds); got != want {
			t.Fatalf("%.0fs: expected %q, got %q", seconds, want, got)
		}
	}
}

func TestComposeDescription_AssemblesTheFourParts(t *testing.T) {
	got := ComposeDescription(
		"A summary.",
		[]string{"0:00 Intro", "1:00 Body"},
		"Subscribe!",
		[]string{"go", "concurrency"},
	)

	for _, want := range []string{"A summary.", "0:00 Intro", "Subscribe!", "#go #concurrency"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in:\n%s", want, got)
		}
	}
}

func TestComposeDescription_SkipsMissingPartsWithoutBlankGaps(t *testing.T) {
	got := ComposeDescription("A summary.", nil, "", nil)

	if got != "A summary." {
		t.Fatalf("expected just the summary, got %q", got)
	}
}

func TestComposeDescription_CapsHashtagsAtThree(t *testing.T) {
	// YouTube only surfaces the first three above the title; more reads as
	// keyword stuffing.
	got := ComposeDescription("", nil, "", []string{"a", "b", "c", "d", "e"})

	if got != "#a #b #c" {
		t.Fatalf("expected three hashtags, got %q", got)
	}
}

func TestComposeDescription_NormalisesTagsIntoUsableHashtags(t *testing.T) {
	got := ComposeDescription("", nil, "", []string{" vòng lặp for ", "#golang"})

	if !strings.Contains(got, "#vònglặpfor") || !strings.Contains(got, "#golang") {
		t.Fatalf("expected cleaned hashtags, got %q", got)
	}
}
