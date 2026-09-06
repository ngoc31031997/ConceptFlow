package domain

import "strings"

// Reading speeds used to pace animation and subtitles when TTS is disabled
// (CR-001 FR3.4/FR9.3). Vietnamese is slightly slower than English because
// its words are shorter and more numerous for the same content.
const (
	wordsPerMinuteEnglish    = 150.0
	wordsPerMinuteVietnamese = 140.0

	// minNarrationSeconds keeps a very short line on screen long enough to
	// read — without it, a two-word narration would flash by in under a second.
	minNarrationSeconds = 1.5
)

// EstimateNarrationDuration approximates how long `text` takes to read aloud.
// It is the stand-in for real TTS audio duration when a project has narration
// disabled: the Rendering Service substitutes it into `self.wait(AUTO)` and
// Video Assembly uses it to time subtitle cues, so both stay in lockstep with
// each other exactly as they would with synthesized audio.
func EstimateNarrationDuration(text string, language VoiceLanguage) float64 {
	words := len(strings.Fields(text))
	if words == 0 {
		return minNarrationSeconds
	}

	wpm := wordsPerMinuteEnglish
	if language == LanguageVietnamese {
		wpm = wordsPerMinuteVietnamese
	}

	seconds := float64(words) / wpm * 60.0
	if seconds < minNarrationSeconds {
		return minNarrationSeconds
	}
	return seconds
}

// SubtitleStyle is the Creator-chosen appearance of burned-in subtitles
// (CR-001 FR9.4). It is persisted on Project and passed through to Video
// Assembly unchanged; the zero value is not meaningful, use DefaultSubtitleStyle.
type SubtitleStyle struct {
	FontSize          string  `json:"font_size"`          // small | medium | large
	TextColor         string  `json:"text_color"`         // hex, e.g. "#FFFFFF"
	BackgroundOpacity float64 `json:"background_opacity"` // 0.0 (no box) .. 1.0
	Position          string  `json:"position"`           // bottom | top
}

// DefaultSubtitleStyle is applied when subtitles are enabled but the Creator
// never opened the style panel.
func DefaultSubtitleStyle() SubtitleStyle {
	return SubtitleStyle{
		FontSize:          "medium",
		TextColor:         "#FFFFFF",
		BackgroundOpacity: 0.6,
		Position:          "bottom",
	}
}
