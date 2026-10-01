package domain

import (
	"regexp"
	"sort"
	"strings"
	"testing"
)

// kitComponents lists the components prompts/illustration_kit_vi.txt documents:
// every `<Name ` / `<Name/` tag on a bullet line (¤ stands for a backtick in
// that file). A bare `<AbsoluteFill>` is Remotion's own, not the kit's.
func kitComponents(t *testing.T) []string {
	t.Helper()
	tag := regexp.MustCompile(`¤<([A-Z][A-Za-z]*)[ /]`)
	seen := map[string]bool{}
	for _, line := range strings.Split(illustrationKitVI, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "- ") {
			continue
		}
		for _, m := range tag.FindAllStringSubmatch(line, -1) {
			if !IsSceneKitComponent(m[1]) {
				seen[m[1]] = true
			}
		}
	}
	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	sort.Strings(out)
	if len(out) < 20 {
		t.Fatalf("found only %d components in the kit (%v) — did its bullet format change?", len(out), out)
	}
	return out
}

// The keyword table must cover the kit exactly, like the Lottie
// catalog is held to its clips (lottie_catalog_test.go). A component added to
// the kit needs keywords or an entry in illustrationKitUnkeyed saying why not;
// a component removed from it must leave the table.
func TestIllustrationKeywordsCoverTheKit(t *testing.T) {
	inKit := map[string]bool{}
	for _, c := range kitComponents(t) {
		inKit[c] = true
	}
	covered := map[string]bool{}
	for _, g := range illustrationKeywords {
		if covered[g.Component] {
			t.Errorf("%s appears twice in illustrationKeywords", g.Component)
		}
		covered[g.Component] = true
		if !inKit[g.Component] {
			t.Errorf("illustrationKeywords has %s, which illustration_kit_vi.txt does not document", g.Component)
		}
		if len(g.Spoken) == 0 {
			t.Errorf("%s has no spoken keyword, so it can never trigger", g.Component)
		}
		for _, kw := range append(append([]string{}, g.Spoken...), g.Shown...) {
			if len(keywordTokens(kw)) == 0 || kw != strings.ToLower(kw) {
				t.Errorf("%s: keyword %q must be non-empty lower-case words", g.Component, kw)
			}
		}
	}
	for c := range illustrationKitUnkeyed {
		if covered[c] {
			t.Errorf("%s is both keyed and in illustrationKitUnkeyed", c)
		}
		if !inKit[c] {
			t.Errorf("illustrationKitUnkeyed has %s, which illustration_kit_vi.txt does not document", c)
		}
		covered[c] = true
	}
	for c := range inKit {
		if !covered[c] {
			t.Errorf("kit component %s has no keywords and is not in illustrationKitUnkeyed", c)
		}
	}
}

func shotScene(shots ...StoryboardShot) []StoryboardScene {
	return []StoryboardScene{{ID: "concrete", Shots: shots}}
}

// The plan's acceptance case: one violating shot, one correct shot.
func TestIllustratedNarrationFlagsOnlyTheShotThatDoesNotShowIt(t *testing.T) {
	got := CheckIllustratedNarration(shotScene(
		StoryboardShot{ID: "2.2", Visual: "Vi khuẩn màu xanh trượt vào, vây quanh chiếc răng.", Narration: "Vi khuẩn bám lên răng."},
		StoryboardShot{ID: "2.3", Visual: "Chiếc răng chuyển sang lo lắng, một mảng nâu lan dần.", Narration: "Những con vi khuẩn này ăn đường."},
	))
	want := []string{"Shot 2.3: lời thoại nhắc 'vi khuẩn' nhưng HÌNH không có"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestIllustratedNarrationMatchesWholeWords(t *testing.T) {
	// "trăng" contains "răng" as letters but is another word; "danh sách" is a
	// list, not a book; "tìm" is not "tim".
	got := CheckIllustratedNarration(shotScene(StoryboardShot{
		ID: "1.1", Visual: "Một vòng tròn.", Narration: "Ánh trăng, danh sách, đi tìm lời giải.",
	}))
	if len(got) != 0 {
		t.Fatalf("want no warnings, got %q", got)
	}
}

func TestIllustratedNarrationShownWordsCount(t *testing.T) {
	// The narration says "bác sĩ"; the visual shows "một người mặc áo blouse".
	got := CheckIllustratedNarration(shotScene(StoryboardShot{
		ID: "1.2", Visual: "Một người mặc áo blouse trắng giơ tay.", Narration: "Bác sĩ nói gì?",
	}))
	if len(got) != 0 {
		t.Fatalf("want no warnings, got %q", got)
	}
}

func TestIllustratedNarrationReportsTheLongerOverlap(t *testing.T) {
	// "kẹo mút" is both Candy ("kẹo") and Lollipop; one warning, the specific one.
	got := CheckIllustratedNarration(shotScene(StoryboardShot{
		ID: "3.1", Visual: "Nền vàng, em bé cười.", Narration: "Một cây kẹo mút thật to!",
	}))
	want := "Shot 3.1: lời thoại nhắc 'kẹo mút' nhưng HÌNH không có"
	if len(got) != 1 || got[0] != want {
		t.Fatalf("got %q, want [%q]", got, want)
	}
}

func TestIllustratedNarrationNormalisesUnicode(t *testing.T) {
	// "răng" written decomposed (a + combining breve) still matches.
	decomposed := "răng"
	got := CheckIllustratedNarration(shotScene(StoryboardShot{
		ID: "1.3", Visual: "Chiếc " + decomposed + " trắng.", Narration: "RĂNG của bạn.",
	}))
	if len(got) != 0 {
		t.Fatalf("want no warnings, got %q", got)
	}
}

func TestIllustratedNarrationOddInputDoesNotPanic(t *testing.T) {
	for _, scenes := range [][]StoryboardScene{
		nil,
		{{ID: "x"}},
		shotScene(StoryboardShot{}),
		shotScene(StoryboardShot{ID: " ", Narration: "răng"}),
	} {
		_ = CheckIllustratedNarration(scenes)
	}
	got := CheckIllustratedNarration(shotScene(StoryboardShot{Narration: "răng"}))
	if len(got) != 1 || !strings.HasPrefix(got[0], "Shot ?:") {
		t.Fatalf("got %q", got)
	}
}
