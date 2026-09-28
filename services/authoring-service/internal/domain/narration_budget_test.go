package domain

import (
	"math"
	"strings"
	"testing"
)

// words returns n space-separated words, each counted once by countWords.
func words(n int) string {
	return strings.TrimSpace(strings.Repeat("chữ ", n))
}

// At 140 wpm (the Vietnamese default) 7 words are 3 seconds.
var budgetFormat = VideoFormat{
	ID: "f", Name: "Thử", Beats: []FormatBeat{
		{ID: "hook", MinSeconds: 6, MaxSeconds: 9, Required: true, MaxRepeat: 1},
		{ID: "concrete", MinSeconds: 30, MaxSeconds: 45, Required: true, MaxRepeat: 1},
		{ID: "variation", MinSeconds: 30, MaxSeconds: 45, MaxRepeat: 3},
	},
}

func scene(id string, narration ...string) StoryboardScene {
	s := StoryboardScene{ID: id}
	for _, n := range narration {
		s.Shots = append(s.Shots, StoryboardShot{ID: "1.1", Visual: "hình", Narration: n})
	}
	return s
}

func TestNarrationBudgetWithinBudgetIsQuiet(t *testing.T) {
	// hook: 7+7 words = 6 s (the minimum); concrete: 84 words = 36 s.
	got := CheckNarrationBudgets([]StoryboardScene{
		scene("hook", words(7), words(7)),
		scene("concrete", words(40), words(44)),
	}, budgetFormat, "vi", 0)
	if len(got) != 0 {
		t.Fatalf("want no warnings, got %q", got)
	}
}

func TestNarrationBudgetToleratesTwentyPercent(t *testing.T) {
	// concrete is 30–45 s, so the warning lines are 24 s and 54 s.
	// 125 words = 53.6 s and 57 words = 24.4 s are inside; 127 = 54.4 s and
	// 55 = 23.6 s are outside.
	for _, n := range []int{125, 57} {
		if got := CheckNarrationBudgets([]StoryboardScene{scene("concrete", words(n))}, budgetFormat, "vi", 0); len(got) != 0 {
			t.Errorf("%d words: want no warning inside the tolerance, got %q", n, got)
		}
	}
	for _, n := range []int{127, 55} {
		if got := CheckNarrationBudgets([]StoryboardScene{scene("concrete", words(n))}, budgetFormat, "vi", 0); len(got) != 1 {
			t.Errorf("%d words: want a warning just outside the tolerance, got %q", n, got)
		}
	}
}

func TestNarrationBudgetLongScene(t *testing.T) {
	// 140 words at 140 wpm = 60 s against a 45 s maximum: +33%.
	got := CheckNarrationBudgets([]StoryboardScene{scene("concrete", words(70), words(70))}, budgetFormat, "vi", 0)
	want := "Cảnh concrete: ~60 giây, ngân sách 30–45 giây (+33%)"
	if len(got) != 1 || got[0] != want {
		t.Fatalf("got %q, want [%q]", got, want)
	}
}

func TestNarrationBudgetShortScene(t *testing.T) {
	// 35 words = 15 s against a 30 s minimum: -50%.
	got := CheckNarrationBudgets([]StoryboardScene{scene("concrete", words(35))}, budgetFormat, "vi", 0)
	want := "Cảnh concrete: ~15 giây, ngân sách 30–45 giây (-50%)"
	if len(got) != 1 || got[0] != want {
		t.Fatalf("got %q, want [%q]", got, want)
	}
}

func TestNarrationBudgetUsesCalibratedRate(t *testing.T) {
	// 140 words: 60 s at the 140 wpm default (too long), 40 s at a measured
	// 210 wpm (inside 30–45).
	scenes := []StoryboardScene{scene("concrete", words(140))}
	if got := CheckNarrationBudgets(scenes, budgetFormat, "vi", 0); len(got) != 1 {
		t.Fatalf("default rate: want 1 warning, got %q", got)
	}
	if got := CheckNarrationBudgets(scenes, budgetFormat, "vi", 210); len(got) != 0 {
		t.Fatalf("calibrated rate: want no warning, got %q", got)
	}
}

func TestNarrationBudgetRepeatedBeatScalesBudget(t *testing.T) {
	// Two variation scenes of 35 s each: 70 s against 60–90 s is fine, though
	// 70 s alone would be over one variation's 45 s.
	got := CheckNarrationBudgets([]StoryboardScene{
		scene("variation", words(82)),
		scene("variation", words(82)),
	}, budgetFormat, "vi", 0)
	if len(got) != 0 {
		t.Fatalf("want no warning for two in-budget variations, got %q", got)
	}
	got = CheckNarrationBudgets([]StoryboardScene{
		scene("variation", words(200)),
		scene("variation", words(200)),
	}, budgetFormat, "vi", 0)
	if len(got) != 1 || !strings.HasPrefix(got[0], "Cảnh variation (2 cảnh): ~171 giây, ngân sách 60–90 giây (+") {
		t.Fatalf("got %q", got)
	}
}

func TestNarrationBudgetUnknownBeatGetsItsOwnWarning(t *testing.T) {
	got := CheckNarrationBudgets([]StoryboardScene{
		scene("hook", words(17)),
		scene("bonus", words(14)),
		scene("", words(7)),
	}, budgetFormat, "vi", 0)
	if len(got) != 2 {
		t.Fatalf("want 2 warnings (bonus, empty id), got %q", got)
	}
	if !strings.HasPrefix(got[0], `Cảnh bonus: không có beat này trong format "Thử"`) || !strings.Contains(got[0], "~6 giây") {
		t.Errorf("unknown beat warning = %q", got[0])
	}
	if !strings.HasPrefix(got[1], "Cảnh không có id: không có beat này") {
		t.Errorf("empty id warning = %q", got[1])
	}
}

func TestNarrationBudgetOddInputDoesNotPanic(t *testing.T) {
	cases := []struct {
		name   string
		scenes []StoryboardScene
		format VideoFormat
		wpm    float64
	}{
		{"nothing", nil, budgetFormat, 0},
		{"scene without shots", []StoryboardScene{{ID: "hook"}}, budgetFormat, 0},
		{"empty format", []StoryboardScene{scene("hook", "a b c")}, VideoFormat{}, 0},
		{"zero budgets", []StoryboardScene{scene("x", "a")}, VideoFormat{Beats: []FormatBeat{{ID: "x"}}}, 0},
		{"NaN rate", []StoryboardScene{scene("hook", "a")}, budgetFormat, math.NaN()},
		{"negative rate", []StoryboardScene{scene("hook", "a")}, budgetFormat, -5},
		{"infinite rate", []StoryboardScene{scene("hook", "a")}, budgetFormat, math.Inf(1)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			for _, w := range CheckNarrationBudgets(c.scenes, c.format, "xx", c.wpm) {
				if strings.Contains(w, "NaN") || strings.Contains(w, "Inf") {
					t.Errorf("warning carries a non-number: %q", w)
				}
			}
		})
	}
}

func TestParseStoryboardScenes(t *testing.T) {
	doc := `{"hero":"h","palette":[],"scenes":[{"id":"hook","title":"t","shots":[
		{"id":"1.1","camera":"c","visual":"răng","narration":"một hai"}]}],"layout":{}}`
	scenes, err := ParseStoryboardScenes(doc)
	if err != nil {
		t.Fatal(err)
	}
	if len(scenes) != 1 || scenes[0].ID != "hook" || scenes[0].Shots[0].Narration != "một hai" {
		t.Fatalf("parsed %+v", scenes)
	}
	for _, bad := range []string{"", "   ", "not json", `{"scenes":[]}`, `{"scenes":null}`, `[]`, `{"scenes":[{"id":3}]}`} {
		if _, err := ParseStoryboardScenes(bad); err == nil {
			t.Errorf("%q: want an error", bad)
		}
	}
}

// The outline prompt and the length check must convert budgets at one rate.
func TestBeatSheetWordsPerMinute(t *testing.T) {
	cases := []struct {
		lang string
		cal  float64
		want float64
	}{
		{"vi", 0, 140}, {"en", 0, 150}, {"xx", 0, 150}, {"vi", 171, 171}, {"vi", -1, 140}, {"vi", math.NaN(), 140},
	}
	for _, c := range cases {
		if got := BeatSheetWordsPerMinute(c.lang, c.cal); got != c.want {
			t.Errorf("BeatSheetWordsPerMinute(%q, %v) = %v, want %v", c.lang, c.cal, got, c.want)
		}
	}
}
