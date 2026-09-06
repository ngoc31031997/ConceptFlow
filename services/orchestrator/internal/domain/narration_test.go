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
