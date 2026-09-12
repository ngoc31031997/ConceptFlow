package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

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

// --- normalizeTags: a nil slice marshals to JSON `null`, which broke the GUI ---

func TestNormalizeTags_NeverReturnsNil(t *testing.T) {
	// Each of these is a shape a small model actually emits. None may produce
	// nil, because nil serialises to `null` and the GUI does tags.join(", ").
	cases := map[string]string{
		"missing key":   `null`,
		"empty string":  `""`,
		"a number":      `42`,
		"an object":     `{"a":1}`,
		"nested arrays": `[[1,2],[3]]`,
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			got := normalizeTags([]byte(raw))
			if got == nil {
				t.Fatal("normalizeTags returned nil; it must return an empty slice so the JSON stays []")
			}
		})
	}
}

func TestNormalizeTags_ParsesAnArray(t *testing.T) {
	got := normalizeTags([]byte(`["thuật toán","sắp xếp"]`))

	if len(got) != 2 || got[0] != "thuật toán" || got[1] != "sắp xếp" {
		t.Fatalf("expected the two tags back, got %#v", got)
	}
}

func TestNormalizeTags_ParsesACommaSeparatedString(t *testing.T) {
	got := normalizeTags([]byte(`"thuật toán, sắp xếp , "`))

	if len(got) != 2 || got[0] != "thuật toán" || got[1] != "sắp xếp" {
		t.Fatalf("expected trimmed tags with the empty one dropped, got %#v", got)
	}
}

func TestNormalizeTags_EmptyArrayStaysEmptyNotNil(t *testing.T) {
	got := normalizeTags([]byte(`[]`))

	if got == nil {
		t.Fatal("an explicitly empty array must stay an empty slice, not become nil")
	}
	if len(got) != 0 {
		t.Fatalf("expected no tags, got %#v", got)
	}
}

// --- Suggest: valid JSON with useless content must not pass silently ---

func newTestClient(t *testing.T, handler http.HandlerFunc) (*OllamaClient, *int) {
	t.Helper()
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		handler(w, r)
	}))
	t.Cleanup(srv.Close)
	return NewOllamaClient(srv.URL, "test-model", 10*time.Second), &calls
}

func modelReplying(payload string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"response": payload})
	}
}

func TestSuggest_RetriesWhenTheModelReturnsNoTitle(t *testing.T) {
	// The failure seen in production: well-formed JSON, empty title, null tags.
	attempt := 0
	client, calls := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		attempt++
		if attempt == 1 {
			modelReplying(`{"title":"","description":"d","tags":null}`)(w, r)
			return
		}
		modelReplying(`{"title":"Sắp xếp nổi bọt","description":"d","tags":["a"]}`)(w, r)
	})

	title, _, tags, err := client.Suggest(context.Background(), "script", "topic", domain.LanguageVietnamese)
	if err != nil {
		t.Fatalf("expected the retry to recover, got %v", err)
	}
	if title != "Sắp xếp nổi bọt" {
		t.Fatalf("expected the second attempt's title, got %q", title)
	}
	if tags == nil {
		t.Fatal("tags must never be nil")
	}
	if *calls != 2 {
		t.Fatalf("expected exactly 2 attempts, got %d", *calls)
	}
}

func TestSuggest_FailsLoudlyWhenEveryAttemptIsUnusable(t *testing.T) {
	client, calls := newTestClient(t, modelReplying(`{"title":"   ","description":"d","tags":null}`))

	_, _, _, err := client.Suggest(context.Background(), "script", "topic", domain.LanguageVietnamese)

	if err == nil {
		t.Fatal("an empty title must surface as an error, not as blank fields the Creator might publish")
	}
	if *calls != suggestMaxAttempts {
		t.Fatalf("expected %d attempts, got %d", suggestMaxAttempts, *calls)
	}
}

func TestSuggest_DoesNotRetryIntoACancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	client, calls := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		cancel() // caller gave up while the first attempt was in flight
		modelReplying(`{"title":"","description":"d","tags":null}`)(w, r)
	})

	if _, _, _, err := client.Suggest(ctx, "script", "topic", domain.LanguageVietnamese); err == nil {
		t.Fatal("expected an error")
	}
	if *calls != 1 {
		t.Fatalf("expected to stop after 1 attempt once the context died, got %d", *calls)
	}
}

func TestSuggest_ReturnsAnEmptyTagArrayWhenTheModelOmitsTags(t *testing.T) {
	client, _ := newTestClient(t, modelReplying(`{"title":"Có tiêu đề","description":"d"}`))

	_, _, tags, err := client.Suggest(context.Background(), "script", "topic", domain.LanguageVietnamese)
	if err != nil {
		t.Fatalf("a missing tags key is not fatal, got %v", err)
	}
	if tags == nil {
		t.Fatal("tags must be an empty slice so the JSON stays [] and the GUI can call join()")
	}
}

// --- prompt size: the measured cause of the empty-title failures ---

func TestBuildSuggestPrompt_CapsTheScriptItSendsToTheModel(t *testing.T) {
	// A real ten-minute project's script is ~17k characters, which overflows
	// Ollama's default 2048-token context and made the model return an empty
	// object every time.
	huge := strings.Repeat("Đây là một câu trong kịch bản. ", 2000)

	prompt := buildSuggestPrompt(huge, "topic", domain.LanguageVietnamese)

	if len([]rune(prompt)) > maxScriptChars+2000 {
		t.Fatalf("prompt is %d runes; the script should have been capped near %d",
			len([]rune(prompt)), maxScriptChars)
	}
}

func TestTruncateScript_CutsByRunesNotBytes(t *testing.T) {
	script := strings.Repeat("ế", maxScriptChars+500)

	got := truncateScript(script)

	if len([]rune(got)) != maxScriptChars {
		t.Fatalf("expected %d runes, got %d", maxScriptChars, len([]rune(got)))
	}
	if !utf8.ValidString(got) {
		t.Fatal("truncation produced invalid UTF-8; the model would receive broken text")
	}
}

func TestTruncateScript_LeavesShortScriptsAlone(t *testing.T) {
	script := "Kịch bản ngắn."

	if got := truncateScript(script); got != script {
		t.Fatalf("expected the script unchanged, got %q", got)
	}
}

// --- SuggestShortScript (CR-026 FR71) ---

func TestStripCodeFence_GoMirrorOfTheTypeScriptHelper(t *testing.T) {
	wrapped := "```python\nfrom conceptflow import *\n```"
	if got := stripCodeFence(wrapped); got != "from conceptflow import *" {
		t.Fatalf("expected the fence stripped, got %q", got)
	}
}

func TestStripCodeFence_LeavesUnfencedScriptAlone(t *testing.T) {
	script := "from conceptflow import *"
	if got := stripCodeFence(script); got != script {
		t.Fatalf("expected the script unchanged, got %q", got)
	}
}

func TestBuildShortScriptSuggestionPrompt_RequiresTheClipWrapper(t *testing.T) {
	// FR70.1/FR71: the entire point of this prompt is that the pipeline's
	// existing generate_clips (CR-007) picks the result up with zero new
	// code — that only works if the model is told, unambiguously, to wrap
	// everything in self.clip("short").
	prompt := buildShortScriptSuggestionPrompt("Vòng lặp for", "", domain.LanguageVietnamese)

	if !strings.Contains(prompt, `self.clip("short")`) {
		t.Fatalf(`expected the prompt to require self.clip("short"), got:\n%s`, prompt)
	}
	if !strings.Contains(prompt, "written in Vietnamese") {
		t.Fatalf("expected the prompt to name the project's content language, got:\n%s", prompt)
	}
}

func TestBuildShortScriptSuggestionPrompt_UsesSourceScriptAsContextOnly(t *testing.T) {
	prompt := buildShortScriptSuggestionPrompt("", "print('long script body')", domain.LanguageEnglish)

	if !strings.Contains(prompt, "print('long script body')") {
		t.Fatal("expected the source script to reach the model as context")
	}
	if !strings.Contains(prompt, "do not summarize or transform its code") {
		t.Fatal("expected the prompt to warn against transforming the long script's code")
	}
}

func TestSuggestShortScript_StripsMarkdownFenceFromTheModelReply(t *testing.T) {
	client, _ := newTestClient(t, modelReplying("```python\nfrom conceptflow import *\n```"))

	script, err := client.SuggestShortScript(context.Background(), "topic", "", domain.LanguageVietnamese)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if script != "from conceptflow import *" {
		t.Fatalf("expected the fence stripped from the model's reply, got %q", script)
	}
}

func TestSuggestShortScript_RetriesOnEmptyReply(t *testing.T) {
	attempt := 0
	client, calls := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		attempt++
		if attempt == 1 {
			modelReplying("")(w, r)
			return
		}
		modelReplying("from conceptflow import *")(w, r)
	})

	script, err := client.SuggestShortScript(context.Background(), "topic", "", domain.LanguageVietnamese)
	if err != nil {
		t.Fatalf("expected the retry to recover, got %v", err)
	}
	if script != "from conceptflow import *" {
		t.Fatalf("expected the second attempt's script, got %q", script)
	}
	if *calls != 2 {
		t.Fatalf("expected exactly 2 attempts, got %d", *calls)
	}
}

func TestSuggestShortScript_DoesNotForceJSONFormat(t *testing.T) {
	// Unlike Suggest (SEO metadata), the response here is multi-line Python
	// source — format:"json" would make the model escape every newline and
	// quote, which is a much easier way to get invalid Python back than
	// asking for plain text is.
	var gotFormat string
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var req generateRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		gotFormat = req.Format
		modelReplying("from conceptflow import *")(w, r)
	})

	if _, err := client.SuggestShortScript(context.Background(), "topic", "", domain.LanguageVietnamese); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotFormat != "" {
		t.Fatalf(`expected no format constraint, got %q`, gotFormat)
	}
}
