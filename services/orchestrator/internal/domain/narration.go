package domain

import "strings"

// LanguageProfile carries everything that varies per content language
// (CR-008 FR21.6). Adding a language is a matter of adding a row here plus a
// TTS voice — never of editing branching logic scattered across the codebase,
// which is what the previous pair of constants and an `if` forced.
type LanguageProfile struct {
	// WordsPerMinute paces animation and subtitles when TTS is disabled
	// (CR-001 FR3.4/FR9.3).
	WordsPerMinute float64
	// EnglishName is how the language is named to an LLM when asking it to
	// produce content in that language (CR-008 FR21.3). English is used
	// because prompts are more reliably followed when the target language is
	// named in the prompt's own language.
	EnglishName string
}

// languageProfiles is the single source of truth for per-language behaviour.
// Vietnamese reads slightly slower than English because its words are shorter
// and more numerous for the same content.
var languageProfiles = map[ContentLanguage]LanguageProfile{
	LanguageEnglish:    {WordsPerMinute: 150.0, EnglishName: "English"},
	LanguageVietnamese: {WordsPerMinute: 140.0, EnglishName: "Vietnamese"},
}

// defaultLanguage is used when a project carries a language this build does
// not know — a forward-compatibility guard, not an expected path.
const defaultLanguage = LanguageEnglish

// DefaultBackgroundMusicVolume is the level used when the Creator has not
// chosen one — the value music was fixed at before CR-005 made it adjustable.
const DefaultBackgroundMusicVolume = 0.2

// minNarrationSeconds keeps a very short line on screen long enough to
// read — without it, a two-word narration would flash by in under a second.
const minNarrationSeconds = 1.5

// ProfileFor returns the profile for a language, falling back to the default
// rather than a zero value: a WordsPerMinute of 0 would divide by zero and
// stall every wait in the script.
func ProfileFor(language ContentLanguage) LanguageProfile {
	if profile, ok := languageProfiles[language]; ok {
		return profile
	}
	return languageProfiles[defaultLanguage]
}

// SupportedLanguages lists the languages this build can produce content in.
func SupportedLanguages() []ContentLanguage {
	return []ContentLanguage{LanguageVietnamese, LanguageEnglish}
}

// EstimateNarrationDuration approximates how long `text` takes to read aloud.
// It is the stand-in for real TTS audio duration when a project has narration
// disabled: the Rendering Service substitutes it into `self.wait(AUTO)` and
// Video Assembly uses it to time subtitle cues, so both stay in lockstep with
// each other exactly as they would with synthesized audio.
func EstimateNarrationDuration(text string, language ContentLanguage) float64 {
	words := countWords(text)
	if words == 0 {
		return minNarrationSeconds
	}

	seconds := float64(words) / ProfileFor(language).WordsPerMinute * 60.0
	if seconds < minNarrationSeconds {
		return minNarrationSeconds
	}
	return seconds
}

// countWords is the single definition of "a word" shared by the plain and the
// calibrated estimate, and mirrored by countWords() in the Web GUI
// (tests/fixtures/narration-duration-vectors.json locks the two together).
func countWords(text string) int {
	return len(strings.Fields(text))
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

// SubtitleMode is how subtitle_cues get delivered to the viewer (CR-015,
// ADR-0027) — a dimension of its own rather than a bolt-on to the CR-001
// on/off toggle, because which delivery is correct depends on the publishing
// surface, not on whether the Creator "wants subtitles":
//
//   - Off:    no subtitles.
//   - Track:  a .srt YouTube caption track (FR38) — searchable,
//     auto-translatable, dismissable by the viewer, never painted
//     over Manim's edge content. The GUI default for new projects
//     (FR41.2).
//   - BurnIn: painted into the video frames — CR-001's original (and, until
//     this CR, only) behaviour. Still required for platforms with
//     no caption-track upload path (Shorts/TikTok, CR-007).
//   - Both:   both at once. Valid (e.g. a repost target with no track
//     upload path) but doubles the text for a viewer with CC on
//     (FR41.3) — the GUI warns rather than blocking it.
type SubtitleMode string

const (
	SubtitleModeOff    SubtitleMode = "off"
	SubtitleModeTrack  SubtitleMode = "track"
	SubtitleModeBurnIn SubtitleMode = "burn_in"
	SubtitleModeBoth   SubtitleMode = "both"
)

// IsValid reports whether m is a mode Video Assembly understands.
func (m SubtitleMode) IsValid() bool {
	switch m {
	case SubtitleModeOff, SubtitleModeTrack, SubtitleModeBurnIn, SubtitleModeBoth:
		return true
	}
	return false
}

// NeedsCues reports whether this mode requires subtitle_cues/subtitle_style
// in the assemble_video payload at all.
func (m SubtitleMode) NeedsCues() bool {
	return m != SubtitleModeOff
}

// SubtitleModeFromLegacy derives a mode from the pre-CR-015 boolean, for a
// project row that predates the subtitle_mode column (project_repository.go)
// or a caller that still only sends subtitles_enabled. It reproduces exactly
// the one behaviour that boolean ever meant: enabled meant burned-in text,
// there being no other kind before this CR.
func SubtitleModeFromLegacy(subtitlesEnabled bool) SubtitleMode {
	if subtitlesEnabled {
		return SubtitleModeBurnIn
	}
	return SubtitleModeOff
}
