package llm

import (
	"strings"
	"testing"

	"orchestrator/internal/domain"
)

// TestBuildSuggestPrompt_NamesTheProjectsContentLanguage is the CR-008
// regression. The prompt used to hardcode "tiếng Việt" and Suggest took no
// language at all, so an English channel got English narration and subtitles
// alongside Vietnamese metadata.
func TestBuildSuggestPrompt_NamesTheProjectsContentLanguage(t *testing.T) {
	prompt := buildSuggestPrompt("some script", "concept", domain.LanguageEnglish)

	if !strings.Contains(prompt, "written in English") {
		t.Fatalf("expected the prompt to ask for English, got:\n%s", prompt)
	}
	if strings.Contains(prompt, "Vietnamese") || strings.Contains(prompt, "tiếng Việt") {
		t.Fatalf("an English project's prompt must not mention Vietnamese:\n%s", prompt)
	}
}

func TestBuildSuggestPrompt_SwitchesLanguageWithTheProject(t *testing.T) {
	prompt := buildSuggestPrompt("some script", "concept", domain.LanguageVietnamese)

	if !strings.Contains(prompt, "written in Vietnamese") {
		t.Fatalf("expected the prompt to ask for Vietnamese, got:\n%s", prompt)
	}
	if strings.Contains(prompt, "written in English") {
		t.Fatalf("a Vietnamese project's prompt must not ask for English:\n%s", prompt)
	}
}

func TestBuildSuggestPrompt_CarriesScriptAndTopic(t *testing.T) {
	prompt := buildSuggestPrompt("SCRIPT-BODY", "algorithms", domain.LanguageEnglish)

	if !strings.Contains(prompt, "SCRIPT-BODY") {
		t.Fatal("the script content must reach the model")
	}
	if !strings.Contains(prompt, "algorithms") {
		t.Fatal("the category hint must reach the model")
	}
}

// TestTruncateTitle_CutsByRunesNotBytes covers the CR-008 FR18.3 bug: len()
// counts bytes and title[:100] slices bytes, so a Vietnamese title (2-3 bytes
// per accented character) was cut mid-character and invalid UTF-8 went to the
// YouTube API. YouTube's limit is in characters, not bytes.
func TestTruncateTitle_CutsByRunesNotBytes(t *testing.T) {
	// 150 accented characters — well over 100 runes and far over 100 bytes.
	long := strings.Repeat("ế", 150)

	got := truncateTitle(long)

	if runes := []rune(got); len(runes) != maxTitleLength {
		t.Fatalf("expected %d runes, got %d", maxTitleLength, len(runes))
	}
	if !isValidUTF8(got) {
		t.Fatal("truncation produced invalid UTF-8")
	}
	// Byte-slicing would have kept only ~33 characters of a 3-byte-per-char
	// string, and split the last one.
	if len(got) <= maxTitleLength {
		t.Fatalf("expected the byte length to exceed %d for a multi-byte title, got %d",
			maxTitleLength, len(got))
	}
}

func TestTruncateTitle_LeavesShortTitlesAlone(t *testing.T) {
	title := "Vòng lặp for trong Java"

	if got := truncateTitle(title); got != title {
		t.Fatalf("expected %q unchanged, got %q", title, got)
	}
}

func isValidUTF8(s string) bool {
	for _, r := range s {
		if r == '�' {
			return false
		}
	}
	return true
}
