package domain

import (
	"bytes"
	"encoding/json"
	"errors"
	"math"
	"regexp"
	"strings"
)

// KitFocusLayers are the depth layers a kit's preview look may keep sharp.
var KitFocusLayers = []string{"sky", "far", "mid", "near", "figures"}

var kitColorPattern = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

// KitLight is the main light of a kit's preview look.
type KitLight struct {
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Color     string  `json:"color"`
	Intensity float64 `json:"intensity"`
}

// KitAtmosphere is the air colour and density of a kit's preview look.
type KitAtmosphere struct {
	Color   string  `json:"color"`
	Density float64 `json:"density"`
}

// KitGrade is the colour grade of a kit's preview look.
type KitGrade struct {
	Warmth     float64 `json:"warmth"`
	Contrast   float64 `json:"contrast"`
	Saturation float64 `json:"saturation"`
}

// KitLook is the neutral look a kit uses to preview one illustration; a video
// brings its own look per scene.
type KitLook struct {
	Light      KitLight      `json:"light"`
	Ambient    string        `json:"ambient"`
	Atmosphere KitAtmosphere `json:"atmosphere"`
	Focus      string        `json:"focus"`
	Grade      KitGrade      `json:"grade"`
}

// KitSpecular is the highlight strength per shiny material.
type KitSpecular struct {
	Glossy float64 `json:"glossy"`
	Metal  float64 `json:"metal"`
	Glass  float64 `json:"glass"`
}

// KitTexture is the surface texture strength per material.
type KitTexture struct {
	Fur     float64 `json:"fur"`
	Fabric  float64 `json:"fabric"`
	Wood    float64 `json:"wood"`
	Stone   float64 `json:"stone"`
	Organic float64 `json:"organic"`
	Leaf    float64 `json:"leaf"`
}

// KitHaze is the atmospheric haze per far depth layer.
type KitHaze struct {
	Far float64 `json:"far"`
	Mid float64 `json:"mid"`
}

// KitDepthOfField is the blur radius in px per depth layer.
type KitDepthOfField struct {
	Sky  float64 `json:"sky"`
	Far  float64 `json:"far"`
	Mid  float64 `json:"mid"`
	Near float64 `json:"near"`
}

// ReferenceKit is the set of drawing numbers rendering paints a style with
// (rim width, shadow strength, grain...). It has the shape of the Remotion
// KitTokens interface and is stored in reference_styles.detail.kit. Every
// number is 0..1 except the grade (-1..1) and DOF (a px blur radius of 0 or
// more); colours
// are #RRGGBB. See ADR-0050.
type ReferenceKit struct {
	Outline        float64         `json:"outline"`
	Radius         float64         `json:"radius"`
	RimWidth       float64         `json:"rimWidth"`
	RimStrength    float64         `json:"rimStrength"`
	ShadowStrength float64         `json:"shadowStrength"`
	AOStrength     float64         `json:"aoStrength"`
	Specular       KitSpecular     `json:"specular"`
	Texture        KitTexture      `json:"texture"`
	SSS            float64         `json:"sss"`
	GlowBlur       float64         `json:"glowBlur"`
	Bloom          float64         `json:"bloom"`
	Haze           KitHaze         `json:"haze"`
	DOF            KitDepthOfField `json:"dof"`
	Grain          float64         `json:"grain"`
	Vignette       float64         `json:"vignette"`
	Ink            string          `json:"ink"`
	PreviewLook    KitLook         `json:"previewLook"`
}

// kitShape lists every key a kit object carries; a nested map is an object
// whose keys are listed in turn. Decoding checks the JSON against it, because
// a missing number would otherwise read as a valid 0.
var kitShape = map[string]any{
	"outline": nil, "radius": nil, "rimWidth": nil, "rimStrength": nil, "shadowStrength": nil, "aoStrength": nil,
	"specular": map[string]any{"glossy": nil, "metal": nil, "glass": nil},
	"texture":  map[string]any{"fur": nil, "fabric": nil, "wood": nil, "stone": nil, "organic": nil, "leaf": nil},
	"sss":      nil, "glowBlur": nil, "bloom": nil,
	"haze":  map[string]any{"far": nil, "mid": nil},
	"dof":   map[string]any{"sky": nil, "far": nil, "mid": nil, "near": nil},
	"grain": nil, "vignette": nil, "ink": nil,
	"previewLook": map[string]any{
		"light":      map[string]any{"x": nil, "y": nil, "color": nil, "intensity": nil},
		"ambient":    nil,
		"atmosphere": map[string]any{"color": nil, "density": nil},
		"focus":      nil,
		"grade":      map[string]any{"warmth": nil, "contrast": nil, "saturation": nil},
	},
}

// checkKitShape reports the first key of value that is missing from, or not
// part of, shape, naming it by its dotted path.
func checkKitShape(path string, shape map[string]any, value map[string]json.RawMessage) error {
	for key := range value {
		if _, ok := shape[key]; !ok {
			return invalid("%s.%s: không có trong bộ thông số vẽ", path, key)
		}
	}
	for key, sub := range shape {
		raw, ok := value[key]
		if !ok || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			return invalid("%s.%s: thiếu", path, key)
		}
		nested, isObject := sub.(map[string]any)
		if !isObject {
			continue
		}
		var inner map[string]json.RawMessage
		if err := json.Unmarshal(raw, &inner); err != nil {
			return invalid("%s.%s: cần là một object", path, key)
		}
		if err := checkKitShape(path+"."+key, nested, inner); err != nil {
			return err
		}
	}
	return nil
}

// UnmarshalJSON decodes a kit and fails, wrapping ErrInvalidReferenceStyle,
// when a key is missing, unknown or of the wrong type.
func (k *ReferenceKit) UnmarshalJSON(data []byte) error {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(data, &top); err != nil {
		return invalid("kit: cần là một object")
	}
	if err := checkKitShape("kit", kitShape, top); err != nil {
		return err
	}
	type plain ReferenceKit
	var out plain
	if err := json.Unmarshal(data, &out); err != nil {
		var typeErr *json.UnmarshalTypeError
		if errors.As(err, &typeErr) {
			return invalid("kit.%s: sai kiểu dữ liệu", typeErr.Field)
		}
		return invalid("kit: %v", err)
	}
	*k = ReferenceKit(out)
	return nil
}

type kitNumber struct {
	path  string
	value float64
}

func (k ReferenceKit) unitNumbers() []kitNumber {
	l := k.PreviewLook
	return []kitNumber{
		{"outline", k.Outline}, {"radius", k.Radius}, {"rimWidth", k.RimWidth}, {"rimStrength", k.RimStrength},
		{"shadowStrength", k.ShadowStrength}, {"aoStrength", k.AOStrength},
		{"specular.glossy", k.Specular.Glossy}, {"specular.metal", k.Specular.Metal}, {"specular.glass", k.Specular.Glass},
		{"texture.fur", k.Texture.Fur}, {"texture.fabric", k.Texture.Fabric}, {"texture.wood", k.Texture.Wood},
		{"texture.stone", k.Texture.Stone}, {"texture.organic", k.Texture.Organic}, {"texture.leaf", k.Texture.Leaf},
		{"sss", k.SSS}, {"glowBlur", k.GlowBlur}, {"bloom", k.Bloom},
		{"haze.far", k.Haze.Far}, {"haze.mid", k.Haze.Mid},
		{"grain", k.Grain}, {"vignette", k.Vignette},
		{"previewLook.light.x", l.Light.X}, {"previewLook.light.y", l.Light.Y},
		{"previewLook.light.intensity", l.Light.Intensity}, {"previewLook.atmosphere.density", l.Atmosphere.Density},
	}
}

// signedNumbers are the numbers that run from -1 to 1.
func (k ReferenceKit) signedNumbers() []kitNumber {
	g := k.PreviewLook.Grade
	return []kitNumber{
		{"previewLook.grade.warmth", g.Warmth}, {"previewLook.grade.contrast", g.Contrast},
		{"previewLook.grade.saturation", g.Saturation},
	}
}

func (k ReferenceKit) depthNumbers() []kitNumber {
	return []kitNumber{{"dof.sky", k.DOF.Sky}, {"dof.far", k.DOF.Far}, {"dof.mid", k.DOF.Mid}, {"dof.near", k.DOF.Near}}
}

// Normalize checks every number's range, the colours and the focus layer, and
// returns the kit with its colours and focus trimmed, or an error wrapping
// ErrInvalidReferenceStyle that names the field at fault.
func (k ReferenceKit) Normalize() (ReferenceKit, error) {
	for _, n := range k.unitNumbers() {
		if math.IsNaN(n.value) || n.value < 0 || n.value > 1 {
			return k, invalid("kit.%s: cần là số từ 0 đến 1", n.path)
		}
	}
	for _, n := range k.signedNumbers() {
		if math.IsNaN(n.value) || n.value < -1 || n.value > 1 {
			return k, invalid("kit.%s: cần là số từ -1 đến 1", n.path)
		}
	}
	for _, n := range k.depthNumbers() {
		if math.IsNaN(n.value) || math.IsInf(n.value, 0) || n.value < 0 {
			return k, invalid("kit.%s: cần là số px từ 0 trở lên", n.path)
		}
	}
	colors := []struct {
		path  string
		value *string
	}{
		{"ink", &k.Ink}, {"previewLook.light.color", &k.PreviewLook.Light.Color},
		{"previewLook.ambient", &k.PreviewLook.Ambient}, {"previewLook.atmosphere.color", &k.PreviewLook.Atmosphere.Color},
	}
	for _, c := range colors {
		v := strings.TrimSpace(*c.value)
		if !kitColorPattern.MatchString(v) {
			return k, invalid("kit.%s: cần là màu #RRGGBB", c.path)
		}
		*c.value = v
	}
	k.PreviewLook.Focus = strings.TrimSpace(k.PreviewLook.Focus)
	for _, f := range KitFocusLayers {
		if f == k.PreviewLook.Focus {
			return k, nil
		}
	}
	return k, invalid("kit.previewLook.focus: chỉ nhận %s", strings.Join(KitFocusLayers, ", "))
}
