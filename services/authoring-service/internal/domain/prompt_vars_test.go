package domain

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

// The golden files in testdata/ were produced by running the SHIPPING
// TypeScript (scriptPrompts.ts) against a fixed format — not written by hand
// to match this Go code. That direction matters: these tests fail if the Go
// port says anything different from what the browser has been sending to the
// Creator's AI all along.
//
// This is the load-bearing test of CR-027 FR77. Everything else in the CR
// assumes the two paths — copy the prompt out, or run it here — are working
// from the same text.

func loadGolden(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("could not read golden file %s: %v", name, err)
	}
	return string(b)
}

func loadFixtureFormat(t *testing.T) VideoFormat {
	t.Helper()
	b, err := os.ReadFile("testdata/format.json")
	if err != nil {
		t.Fatalf("could not read the fixture format: %v", err)
	}
	var f VideoFormat
	if err := json.Unmarshal(b, &f); err != nil {
		t.Fatalf("could not parse the fixture format: %v", err)
	}
	return f
}

// diffAt reports the first differing character, so a failure says where the
// port drifted instead of printing two walls of Vietnamese prose.
func diffAt(want, got string) string {
	w, g := []rune(want), []rune(got)
	for i := 0; i < len(w) && i < len(g); i++ {
		if w[i] != g[i] {
			from := i - 40
			if from < 0 {
				from = 0
			}
			return "first difference at rune " + fmt.Sprint(i) +
				"\n  context: ..." + string(w[from:i]) + "<<<HERE>>>" +
				"\n  want:    " + quoteRune(w[i]) +
				"\n  got:     " + quoteRune(g[i])
		}
	}
	if len(w) != len(g) {
		return "same prefix, different length: want " + fmt.Sprint(len(w)) +
			" runes, got " + fmt.Sprint(len(g))
	}
	return ""
}

func quoteRune(r rune) string {
	return fmt.Sprintf("%q (U+%04X)", r, r)
}

func TestBuildStoryBeatSheetSection_MatchesTheTypeScriptCharacterForCharacter(t *testing.T) {
	format := loadFixtureFormat(t)

	cases := []struct {
		name     string
		language string
		wpm      float64
		golden   string
	}{
		{"Vietnamese at the default 140 wpm", "vi", 0, "beats_vi_default.txt"},
		{"English at the default 150 wpm", "en", 0, "beats_en_default.txt"},
		// A calibrated voice changes every budget in the block, so this case
		// covers the arithmetic rather than just the wording.
		{"a calibrated voice at 117 wpm", "vi", 117, "beats_vi_calibrated.txt"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			want := loadGolden(t, tc.golden)
			got := BuildStoryBeatSheetSection(format, tc.language, tc.wpm)

			if got != want {
				t.Fatalf("the Go port drifted from the shipping TypeScript.\n%s", diffAt(want, got))
			}
		})
	}
}

// TestChannelIdentity_IsTheShippedBlock — embedded, so this guards the
// wiring (right language, no stray trimming) rather than the text.
func TestChannelIdentity_IsTheShippedBlock(t *testing.T) {
	vi := ChannelIdentity("vi")
	en := ChannelIdentity("en")

	if !strings.Contains(vi, "BẢN SẮC KÊNH") {
		t.Fatalf("Vietnamese identity block looks wrong: %.60q", vi)
	}
	if !strings.Contains(en, "CHANNEL IDENTITY") {
		t.Fatalf("English identity block looks wrong: %.60q", en)
	}
	if vi == en {
		t.Fatal("the two languages must not return the same block")
	}
}

// TestChannelIdentity_UnknownLanguageStillReturnsABlock — an empty identity
// section still renders as a valid-looking prompt and quietly produces a
// generic video, which is the worst way for this to fail.
func TestChannelIdentity_UnknownLanguageStillReturnsABlock(t *testing.T) {
	if got := ChannelIdentity("fr"); got == "" {
		t.Fatal("an unknown language must fall back, not return nothing")
	}
	if got := NarrationLanguageRule("fr"); got == "" {
		t.Fatal("an unknown language must fall back, not return nothing")
	}
}

func TestNarrationLanguageRule_NamesTheRightLanguage(t *testing.T) {
	if !strings.Contains(NarrationLanguageRule("vi"), "TIẾNG VIỆT") {
		t.Fatalf("Vietnamese rule looks wrong: %q", NarrationLanguageRule("vi"))
	}
	if !strings.Contains(NarrationLanguageRule("en"), "TIẾNG ANH") {
		t.Fatalf("English rule looks wrong: %q", NarrationLanguageRule("en"))
	}
}

// CR-040 FR113.2: the subtitle-zone text web-gui used to build, for every
// language × mode × size × position, including the unknown size that falls back to medium.
func TestSubtitleZoneFor_MatchesTheTypeScriptCharacterForCharacter(t *testing.T) {
	raw, err := os.ReadFile("testdata/prompt_golden.json")
	if err != nil {
		t.Fatal(err)
	}
	var golden struct {
		Zones []struct {
			Language string `json:"language"`
			Mode     string `json:"mode"`
			FontSize string `json:"font_size"`
			Position string `json:"position"`
			Expected string `json:"expected"`
		} `json:"subtitle_zones"`
	}
	if err := json.Unmarshal(raw, &golden); err != nil {
		t.Fatal(err)
	}
	if len(golden.Zones) == 0 {
		t.Fatal("golden file has no subtitle zones")
	}
	for _, z := range golden.Zones {
		got := SubtitleZoneFor(SubtitleMode(z.Mode), SubtitleStyle{FontSize: z.FontSize, Position: z.Position}, z.Language)
		if got != z.Expected {
			t.Errorf("%s/%s/%s/%s:\n got: %s\nwant: %s", z.Language, z.Mode, z.FontSize, z.Position, got, z.Expected)
		}
	}
}
