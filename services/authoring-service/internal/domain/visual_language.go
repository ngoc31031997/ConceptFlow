package domain

import (
	_ "embed"
	"errors"
	"fmt"
	"strings"
)

// A visual language is how a channel's videos are drawn: the kit that paints
// the forms (kit_key) and the three rule texts the AI roles are held to. A
// project pins one language, by id and version, when its authoring row is
// created; every prompt, check, preview and render of the project then uses
// the pinned version, whatever language is active later.

//go:embed prompts/visual_languages/phatsang_illustration_vi.txt
var phatsangIllustrationRules string

//go:embed prompts/visual_languages/phatsang_backdrop_vi.txt
var phatsangBackdropRules string

//go:embed prompts/visual_languages/phatsang_scene_vi.txt
var phatsangSceneRules string

// Ids and keys of the shipped language.
const (
	PhatsangID     = "phatsang"
	PhatsangKitKey = "phatsang"
)

var (
	ErrVisualLanguageNotFound   = errors.New("visual language not found")
	ErrVisualLanguageReadOnly   = errors.New("ngôn ngữ hệ thống chỉ đọc: sao chép để sửa")
	ErrVisualLanguageActive     = errors.New("không xoá được ngôn ngữ đang bật")
	ErrVisualLanguageInUse      = errors.New("có project đã ghim ngôn ngữ này, không xoá được")
	ErrVisualLanguageNameTaken  = errors.New("tên ngôn ngữ đã được dùng")
	ErrNoActiveVisualLanguage   = errors.New("chưa có ngôn ngữ hình nào đang bật")
	ErrProjectNotPinned         = errors.New("project chưa ghim ngôn ngữ hình")
	ErrInvalidVisualLanguage    = errors.New("ngôn ngữ hình không hợp lệ")
	ErrUnknownKitKey            = errors.New("kit_key không có trong bộ kit của rendering")
	ErrVisualLanguageSampleBusy = errors.New("đang dựng ảnh mẫu")
)

// VisualLanguageRules are the three rule texts of one version of a language.
type VisualLanguageRules struct {
	// Illustration is what the illustration drawer and the library checks follow.
	Illustration string `json:"illustration_rules"`
	// Backdrop is the extra rules of a whole-frame place.
	Backdrop string `json:"backdrop_rules"`
	// Scene is inserted into the Director's prompt: how light is used in a scene.
	Scene string `json:"scene_rules"`
}

// Validate refuses a rule set with an empty text: the roles that read it
// would otherwise run on nothing.
func (r VisualLanguageRules) Validate() error {
	switch {
	case strings.TrimSpace(r.Illustration) == "":
		return fmt.Errorf("%w: thiếu luật vẽ hình", ErrInvalidVisualLanguage)
	case strings.TrimSpace(r.Backdrop) == "":
		return fmt.Errorf("%w: thiếu luật vẽ nền", ErrInvalidVisualLanguage)
	case strings.TrimSpace(r.Scene) == "":
		return fmt.Errorf("%w: thiếu luật cảnh", ErrInvalidVisualLanguage)
	}
	return nil
}

// VisualLanguage is one entry of the language store, at its newest version.
type VisualLanguage struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Note        string `json:"note"`        // reference note: where the look comes from
	Description string `json:"description"` // what the look is
	KitKey      string `json:"kit_key"`
	IsSystem    bool   `json:"is_system"`
	IsActive    bool   `json:"is_active"`
	// Version is the newest version; Rules are that version's texts.
	Version   int                 `json:"version"`
	Rules     VisualLanguageRules `json:"rules"`
	CreatedAt string              `json:"created_at"`
}

// PinnedLanguage is the language version a project is pinned to.
type PinnedLanguage struct {
	ID      string              `json:"visual_language_id"`
	Version int                 `json:"visual_language_version"`
	KitKey  string              `json:"kit_key"`
	Name    string              `json:"name"`
	Rules   VisualLanguageRules `json:"rules"`
}

// LanguageSample is one rendered sample scene of a kit: a still shown on the
// language's page. A sample is current while KitVersion equals the kit's.
type LanguageSample struct {
	KitKey     string `json:"kit_key"`
	KitVersion int    `json:"kit_version"`
	Scene      string `json:"scene"` // "fig" | "space"
	Frame      string `json:"frame"` // "wide" | "tall"
	File       string `json:"file"`  // name in the shared stills directory
}

// SampleScenes and SampleFrames list every sample the language page shows.
var (
	SampleScenes = []string{"fig", "space"}
	SampleFrames = []string{"wide", "tall"}
)

// SampleComposition is the rendering composition id of a sample.
func SampleComposition(scene, frame string) string {
	return "sample-" + scene + "-" + frame
}

// PhatsangRules are the shipped rule texts of the Khoa học phát sáng language.
func PhatsangRules() VisualLanguageRules {
	return VisualLanguageRules{
		Illustration: strings.TrimSpace(phatsangIllustrationRules),
		Backdrop:     strings.TrimSpace(phatsangBackdropRules),
		Scene:        strings.TrimSpace(phatsangSceneRules),
	}
}

// Tier is the role of a drawing in one video: the thing the video is about
// (hero), a thing with a small part (support) or scenery (background).
type Tier string

const (
	TierHero       Tier = "hero"
	TierSupport    Tier = "support"
	TierBackground Tier = "background"
)

// ValidTier reports whether t is one of the three tiers.
func ValidTier(t Tier) bool {
	return t == TierHero || t == TierSupport || t == TierBackground
}

// TierOrSupport is t, or TierSupport for an empty or unknown tier.
func TierOrSupport(t Tier) Tier {
	if ValidTier(t) {
		return t
	}
	return TierSupport
}

// DetailLevel is how much detail a library drawing was drawn with.
type DetailLevel string

const (
	DetailRich       DetailLevel = "rich"       // drawn as a hero
	DetailMedium     DetailLevel = "medium"     // drawn as a supporting figure
	DetailSilhouette DetailLevel = "silhouette" // drawn as scenery
)

// ValidDetail reports whether d is one of the three detail levels.
func ValidDetail(d DetailLevel) bool {
	return d == DetailRich || d == DetailMedium || d == DetailSilhouette
}

// DetailFor is the detail a drawing gets when it is drawn for tier t.
func DetailFor(t Tier) DetailLevel {
	switch t {
	case TierHero:
		return DetailRich
	case TierBackground:
		return DetailSilhouette
	}
	return DetailMedium
}

// TierForDetail is the tier whose limits a drawing of detail level d is made to:
// rich is a hero, silhouette is scenery, anything else (including none) is support.
func TierForDetail(d DetailLevel) Tier {
	switch d {
	case DetailRich:
		return TierHero
	case DetailSilhouette:
		return TierBackground
	}
	return TierSupport
}

// DetailEnough reports whether a drawing at detail level d may play tier t:
// rich serves every tier, medium serves support and background, silhouette
// serves only background.
func DetailEnough(d DetailLevel, t Tier) bool {
	switch d {
	case DetailRich:
		return true
	case DetailMedium:
		return t != TierHero
	case DetailSilhouette:
		return t == TierBackground
	}
	return false
}
