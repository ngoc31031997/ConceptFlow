package domain

import "testing"

func TestEstimateNarrationDuration_ScalesWithWordCountAndLanguage(t *testing.T) {
	// 150 words at 150 wpm (English) is exactly one minute; Vietnamese reads
	// the same count slower, so it must come out longer.
	text := ""
	for i := 0; i < 150; i++ {
		text += "word "
	}

	english := EstimateNarrationDuration(text, LanguageEnglish)
	if english < 59.9 || english > 60.1 {
		t.Fatalf("expected ~60s for 150 English words, got %v", english)
	}

	vietnamese := EstimateNarrationDuration(text, LanguageVietnamese)
	if vietnamese <= english {
		t.Fatalf("expected Vietnamese (%v) to read slower than English (%v)", vietnamese, english)
	}
}

func TestEstimateNarrationDuration_ShortAndEmptyTextGetMinimumOnScreenTime(t *testing.T) {
	for _, text := range []string{"", "   ", "hi"} {
		if got := EstimateNarrationDuration(text, LanguageEnglish); got != minNarrationSeconds {
			t.Fatalf("text %q: expected floor of %v, got %v", text, minNarrationSeconds, got)
		}
	}
}

func TestSubtitleModeIsValid(t *testing.T) {
	valid := []SubtitleMode{SubtitleModeOff, SubtitleModeTrack, SubtitleModeBurnIn, SubtitleModeBoth}
	for _, m := range valid {
		if !m.IsValid() {
			t.Fatalf("expected %q to be valid", m)
		}
	}
	if SubtitleMode("bogus").IsValid() {
		t.Fatal("expected an unknown mode to be invalid")
	}
	if SubtitleMode("").IsValid() {
		t.Fatal("expected the zero value to be invalid, not silently treated as a real mode")
	}
}

func TestSubtitleModeNeedsCues(t *testing.T) {
	if SubtitleModeOff.NeedsCues() {
		t.Fatal("expected off to need no cues")
	}
	for _, m := range []SubtitleMode{SubtitleModeTrack, SubtitleModeBurnIn, SubtitleModeBoth} {
		if !m.NeedsCues() {
			t.Fatalf("expected %q to need cues", m)
		}
	}
}

func TestSubtitleModeFromLegacy(t *testing.T) {
	if got := SubtitleModeFromLegacy(true); got != SubtitleModeBurnIn {
		t.Fatalf("expected legacy true to map to burn_in, got %q", got)
	}
	if got := SubtitleModeFromLegacy(false); got != SubtitleModeOff {
		t.Fatalf("expected legacy false to map to off, got %q", got)
	}
}
