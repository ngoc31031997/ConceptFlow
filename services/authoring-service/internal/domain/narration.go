package domain

import (
	"fmt"
	"strings"
)

// LanguageProfile carries everything that varies per content language.
// Adding a language is a matter of adding a row here plus a
// TTS voice — never of editing branching logic scattered across the codebase,
// which is what the previous pair of constants and an `if` forced.
type LanguageProfile struct {
	// WordsPerMinute paces animation and subtitles when TTS is disabled.
	WordsPerMinute float64
	// EnglishName is how the language is named to an LLM when asking it to
	// produce content in that language. English is used
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
// chosen one.
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

// SubtitleStyle is the Creator-chosen appearance of burned-in subtitles.
// It is persisted on Project and passed through to Video
// Assembly unchanged; the zero value is not meaningful, use DefaultSubtitleStyle.
type SubtitleStyle struct {
	// FontFamily is one of SubtitleFontFamilies. Empty on every row saved
	// before the field existed; Video Assembly treats that as the font it
	// always used (DejaVu Sans), so old projects render unchanged.
	FontFamily        string  `json:"font_family,omitempty"`
	FontSize          string  `json:"font_size"`          // small | medium | large
	TextColor         string  `json:"text_color"`         // hex, e.g. "#FFFFFF"
	BackgroundOpacity float64 `json:"background_opacity"` // 0.0 (no box) .. 1.0
	Position          string  `json:"position"`           // bottom | top
}

// DefaultSubtitleStyle is applied when subtitles are enabled but the Creator
// never opened the style panel.
func DefaultSubtitleStyle() SubtitleStyle {
	return SubtitleStyle{
		FontFamily:        "DejaVu Sans",
		FontSize:          "medium",
		TextColor:         "#FFFFFF",
		BackgroundOpacity: 0.6,
		Position:          "bottom",
	}
}

// SubtitleFontFamilies are the fonts the Video Assembly image installs for
// burn-in (services/video-assembly/Dockerfile). libass silently substitutes a
// font it cannot find, so offering one that is not installed would render in
// something else with nothing to say so.
var SubtitleFontFamilies = []string{"DejaVu Sans", "Be Vietnam Pro", "Montserrat"}

// VideoFontFamilies are the fonts a Creator can pick for text drawn INSIDE a
// Remotion video (labels, numbers, titles — not the subtitles, which have
// their own SubtitleStyle.FontFamily). Every one is installed in the
// rendering image (services/rendering/Dockerfile) and covers Vietnamese
// diacritics; headless Chrome would otherwise fall back to a generic sans
// without a word.
var VideoFontFamilies = []string{"Be Vietnam Pro", "Montserrat", "Cormorant Garamond"}

// DefaultVideoFont is what an empty Project.VideoFont means — the font
// conceptflow-mini hardcoded before the choice existed.
const DefaultVideoFont = "Be Vietnam Pro"

// ValidVideoFont reports whether font is "" (use the default) or one of
// VideoFontFamilies.
func ValidVideoFont(font string) bool {
	if font == "" {
		return true
	}
	for _, f := range VideoFontFamilies {
		if f == font {
			return true
		}
	}
	return false
}

// subtitleBandPx is how tall a strip, measured from the frame edge, burned-in
// subtitles can occupy (the same on either frame; the subtitle file is laid
// out in the video's own pixels): Video Assembly's vertical margin
// (60) + two wrapped lines at its font size (subtitle_file.py FONT_SIZES) +
// the background box's padding.
var subtitleBandPx = map[string]int{"small": 200, "medium": 240, "large": 280}

// SubtitleZone renders {{subtitle_zone}} for the Remotion Engineer: which band
// of the frame burned-in subtitles will cover, so the composition keeps its
// own drawing out of it. Only burn-in paints over the frame; a caption track
// is drawn by the player, off the video, and needs no room.
func SubtitleZone(project *Project, language string) string {
	mode, style := projectSubtitleSettings(project)
	return SubtitleZoneIn(mode, style, FrameFor(project.VideoOutputMode), language)
}

// projectSubtitleSettings reads a project's subtitle mode and style, falling
// back to the legacy on/off flag and the default style.
func projectSubtitleSettings(project *Project) (SubtitleMode, SubtitleStyle) {
	mode := project.SubtitleMode
	if !mode.IsValid() {
		mode = SubtitleModeFromLegacy(project.SubtitlesEnabled)
	}
	style := DefaultSubtitleStyle()
	if project.SubtitleStyle != nil {
		style = *project.SubtitleStyle
	}
	return mode, style
}

// SubtitleBand is the strip of the frame that burned-in subtitles cover: Px pixels measured from the Edge ("top" or "bottom") of the frame.
type SubtitleBand struct {
	Edge string
	Px   int
}

// SubtitleBandFor is the strip burned-in subtitles cover, or false when
// nothing is painted over the frame (off, or a caption track only). It is the
// one source of the numbers both {{subtitle_zone}} (what the Remotion
// Engineer is told) and the rendering layout check (what the drawn frame is
// held to) use.
func SubtitleBandFor(mode SubtitleMode, style SubtitleStyle) (SubtitleBand, bool) {
	if mode != SubtitleModeBurnIn && mode != SubtitleModeBoth {
		return SubtitleBand{}, false
	}
	band, ok := subtitleBandPx[style.FontSize]
	if !ok {
		band = subtitleBandPx["medium"]
	}
	edge := "bottom"
	if style.Position == "top" {
		edge = "top"
	}
	return SubtitleBand{Edge: edge, Px: band}, true
}

// ProjectSubtitleBand is SubtitleBandFor for a saved project, with the same
// legacy fallbacks SubtitleZone applies.
func ProjectSubtitleBand(project *Project) (SubtitleBand, bool) {
	return SubtitleBandFor(projectSubtitleSettings(project))
}

// SubtitleZoneFor is SubtitleZone for explicit settings on the landscape
// frame — what the wizard has in its draft before anything is saved.
func SubtitleZoneFor(mode SubtitleMode, style SubtitleStyle, language string) string {
	return SubtitleZoneIn(mode, style, LandscapeFrame, language)
}

// SubtitleZoneIn is SubtitleZoneFor on the given frame.
func SubtitleZoneIn(mode SubtitleMode, style SubtitleStyle, frame Frame, language string) string {
	sb, burned := SubtitleBandFor(mode, style)
	if !burned {
		if language == "vi" {
			return "video này KHÔNG in phụ đề lên hình — được dùng toàn bộ vùng an toàn."
		}
		return "this video has NO burned-in subtitles — the whole safe area is yours."
	}
	band := sb.Px
	if sb.Edge == "top" {
		if language == "vi" {
			return fmt.Sprintf("phụ đề được in ở MÉP TRÊN khung — dải y từ 0 đến %d px phải để TRỐNG hoàn toàn (không chữ, không vật có nghĩa). Vùng an toàn của bạn bắt đầu từ y = %d.", band, band+24)
		}
		return fmt.Sprintf("subtitles are burned in at the TOP of the frame — the strip from y = 0 to %d px must stay completely EMPTY (no text, no meaningful object). Your safe area starts at y = %d.", band, band+24)
	}
	h := frame.Height
	if language == "vi" {
		return fmt.Sprintf("phụ đề được in ở MÉP DƯỚI khung — dải y từ %d đến %d px phải để TRỐNG hoàn toàn (không chữ, không vật có nghĩa). Vùng an toàn của bạn kết thúc ở y = %d.", h-band, h, h-band-24)
	}
	return fmt.Sprintf("subtitles are burned in at the BOTTOM of the frame — the strip from y = %d to %d px must stay completely EMPTY (no text, no meaningful object). Your safe area ends at y = %d.", h-band, h, h-band-24)
}

// SubtitleMode is how subtitle_cues get delivered to the viewer (see
// ADR-0027) — a dimension of its own rather than a bolt-on to an on/off
// toggle, because which delivery is correct depends on the publishing
// surface, not on whether the Creator "wants subtitles":
//
//   - Off:    no subtitles.
//   - Track:  a .srt YouTube caption track — searchable,
//     auto-translatable, dismissable by the viewer, never painted
//     over Manim's edge content. The GUI default for new projects.
//   - BurnIn: painted into the video frames. Required for platforms with
//     no caption-track upload path (Shorts/TikTok).
//   - Both:   both at once. Valid (e.g. a repost target with no track
//     upload path) but doubles the text for a viewer with CC on —
//     the GUI warns rather than blocking it.
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

// SubtitleModeFromLegacy derives a mode from the legacy boolean, for a
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
