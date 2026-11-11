package domain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// Keys of the eight parts of the visual concept, one review unit each, in the
// order the AI writes them (a later part reads the earlier ones).
const (
	UnitCreativeDirection   = "creative_direction"
	UnitColorSystem         = "color_system"
	UnitVisualLanguage      = "visual_language"
	UnitRepresentationRules = "representation_rules"
	UnitTypography          = "typography"
	UnitMotionLanguage      = "motion_language"
	UnitDoNotUse            = "do_not_use"
	UnitFeasibility         = "feasibility"
)

// UnitSpec describes one part: its title and the sources it is written from.
type UnitSpec struct {
	Key   string
	Title string
	Reads []string
}

// ConceptUnits lists the eight parts in writing order.
var ConceptUnits = []UnitSpec{
	{UnitCreativeDirection, "Định hướng sáng tạo", []string{SourceBrief, SourceStory, SourceBeats, SourceRefs}},
	{UnitColorSystem, "Bảng màu và tương phản", []string{SourceStory, SourceBeats}},
	{UnitVisualLanguage, "Hình dạng, nét, ánh sáng, chiều sâu, bề mặt, layout, bố cục", []string{SourceBeats, SourceRefs}},
	{UnitRepresentationRules, "Nhân vật, vật thể, khái niệm vô hình, hiện tượng khoa học", []string{SourceResearch, SourceScript}},
	{UnitTypography, "Chữ, nhãn, chú thích và công thức", []string{SourceBrief, SourceScript}},
	{UnitMotionLanguage, "Nguyên tắc chuyển động", []string{SourceStory, SourceBeats}},
	{UnitDoNotUse, "Cách thể hiện không được dùng", []string{SourceResearch, SourceScript}},
	{UnitFeasibility, "Kiểm tra khả thi", []string{SourceBrief, SourceBeats}},
}

// UnitSpecOf returns the spec of key.
func UnitSpecOf(key string) (UnitSpec, bool) {
	for _, u := range ConceptUnits {
		if u.Key == key {
			return u, true
		}
	}
	return UnitSpec{}, false
}

// UnitNumber is the 1-based position of key, 0 when unknown.
func UnitNumber(key string) int {
	for i, u := range ConceptUnits {
		if u.Key == key {
			return i + 1
		}
	}
	return 0
}

// AbstractionLevels are how far a thing is drawn from how it looks.
var AbstractionLevels = []string{"realistic", "semi_real", "stylized", "character", "diagram", "symbol", "abstract", "hybrid"}

// Paces are the speeds of motion a beat can take.
var Paces = []string{"slow", "medium", "fast"}

// DoNotUseSources say why a representation is forbidden.
var DoNotUseSources = []string{"accuracy", "style"}

// Verdicts of a feasibility check.
var Verdicts = []string{"ok", "risk"}

// --- the parts ------------------------------------------------------------------

// Example is a pair of what to do and what not to do.
type Example struct {
	Do   string `json:"do"`
	Dont string `json:"dont"`
}

// AbstractionEntry says how abstract one kind of thing is drawn.
type AbstractionEntry struct {
	For     string `json:"for"`
	Level   string `json:"level"`
	Example string `json:"example"`
}

// CreativeDirection is part 1.
type CreativeDirection struct {
	StyleName        string             `json:"style_name"`
	Description      string             `json:"description"`
	AbstractionLevel string             `json:"abstraction_level"`
	Tone             []string           `json:"tone"`
	Principles       []string           `json:"principles"`
	Why              string             `json:"why"`
	Feeling          string             `json:"feeling"`
	RefsUsed         []string           `json:"refs_used"`
	AbstractionMap   []AbstractionEntry `json:"abstraction_map"`
	Examples         []Example          `json:"examples"`
}

// Swatch is one colour of the palette.
type Swatch struct {
	Token string `json:"token"`
	Hex   string `json:"hex"`
	Role  string `json:"role"`
	Name  string `json:"name"`
}

// JourneyStep is the colour mood of a run of beats. BackgroundToken is the
// palette token behind the text in those beats; empty means the system's
// background.
type JourneyStep struct {
	BeatKeys        []string `json:"beat_keys"`
	Mood            string   `json:"mood"`
	AccentToken     string   `json:"accent_token"`
	BackgroundToken string   `json:"background_token"`
}

// Contrast are the contrast rules. TextToken is the palette token of on-screen
// text; TextMin the WCAG contrast ratio text keeps on every background.
type Contrast struct {
	TextMin        float64 `json:"text_min"`
	TextToken      string  `json:"text_token"`
	SubjectVsBg    string  `json:"subject_vs_bg"`
	AccentPerFrame int     `json:"accent_per_frame"`
}

// ColorException lets a token appear only in some beats.
type ColorException struct {
	Token    string   `json:"token"`
	BeatKeys []string `json:"beat_keys"`
	Why      string   `json:"why"`
}

// ColorSystem is part 2.
type ColorSystem struct {
	Background string           `json:"background"`
	Palette    []Swatch         `json:"palette"`
	Journey    []JourneyStep    `json:"journey"`
	Contrast   Contrast         `json:"contrast"`
	Exceptions []ColorException `json:"exceptions"`
	Examples   []string         `json:"examples"`
}

// VisualLanguage is part 3. Shape, line, lighting, depth, surface and layout
// rules carry what the picked references say about how things are drawn and
// laid out; composition rules are the AI's own, set from the topic and scenes.
type VisualLanguage struct {
	ShapeRules       []string `json:"shape_rules"`
	LineRules        []string `json:"line_rules"`
	LightingRules    []string `json:"lighting_rules"`
	DepthRules       []string `json:"depth_rules"`
	SurfaceRules     []string `json:"surface_rules"`
	LayoutRules      []string `json:"layout_rules"`
	CompositionRules []string `json:"composition_rules"`
	Examples         []string `json:"examples"`
}

// RepresentedThing is how one character, object or invisible concept is drawn.
type RepresentedThing struct {
	ConceptID    string   `json:"concept_id"`
	Term         string   `json:"term"`
	Draw         string   `json:"draw"`
	MustNotImply []string `json:"must_not_imply"`
	Facts        []string `json:"facts"`
}

// Characters are the approach to characters and each character.
type Characters struct {
	Approach          string             `json:"approach"`
	AllowedVariations []string           `json:"allowed_variations"`
	Items             []RepresentedThing `json:"items"`
}

// ScientificConcept is how one scientific phenomenon is shown.
type ScientificConcept struct {
	ConceptID      string   `json:"concept_id"`
	Term           string   `json:"term"`
	Representation string   `json:"representation"`
	MustNotImply   []string `json:"must_not_imply"`
	Facts          []string `json:"facts"`
}

// RepresentationRules is part 4 (the typography of the document's
// representation_rules is part 5).
type RepresentationRules struct {
	Characters         Characters          `json:"characters"`
	Objects            []RepresentedThing  `json:"objects"`
	Invisible          []RepresentedThing  `json:"invisible"`
	ScientificConcepts []ScientificConcept `json:"scientific_concepts"`
	Examples           []string            `json:"examples"`
}

// Labels are the rules of on-screen labels.
type Labels struct {
	MaxWords int    `json:"max_words"`
	When     string `json:"when"`
	MinPx    int    `json:"min_px"`
}

// TermDisplay says how a term is said and whether it shows on screen.
type TermDisplay struct {
	Say          string `json:"say"`
	ShowOnScreen bool   `json:"show_on_screen"`
}

// Typography is part 5.
type Typography struct {
	FontDirection string        `json:"font_direction"`
	LabelStyle    string        `json:"label_style"`
	Font          string        `json:"font"`
	Labels        Labels        `json:"labels"`
	Captions      string        `json:"captions"`
	Terms         []TermDisplay `json:"terms"`
	Formulas      string        `json:"formulas"`
}

// BeatPace is the motion speed of a run of beats.
type BeatPace struct {
	BeatKeys []string `json:"beat_keys"`
	Pace     string   `json:"pace"`
}

// MotionLanguage is part 6.
type MotionLanguage struct {
	Principles           []string   `json:"principles"`
	PreferredTransitions []string   `json:"preferred_transitions"`
	Avoid                []string   `json:"avoid"`
	Pace                 []BeatPace `json:"pace"`
	CharacterMotion      string     `json:"character_motion"`
	Examples             []string   `json:"examples"`
}

// Forbidden is one representation that must not be used.
type Forbidden struct {
	Text   string   `json:"text"`
	Why    string   `json:"why"`
	Facts  []string `json:"facts"`
	Source string   `json:"source"`
}

// DoNotUse is part 7; the document holds Items as a plain array.
type DoNotUse struct {
	Items []Forbidden `json:"items"`
}

// Scope is what the video has to produce. Beats and Characters are counted by
// the machine; EstShots and Settings are the AI's estimate.
type Scope struct {
	Beats      int `json:"beats"`
	EstShots   int `json:"est_shots"`
	Settings   int `json:"settings"`
	Characters int `json:"characters"`
}

// FeasibilityCheck is one item weighed against the production means.
type FeasibilityCheck struct {
	Item       string `json:"item"`
	Assessment string `json:"assessment"`
	Verdict    string `json:"verdict"`
	Mitigation string `json:"mitigation"`
}

// Feasibility is part 8.
type Feasibility struct {
	Scope  Scope              `json:"scope"`
	Checks []FeasibilityCheck `json:"checks"`
}

// newPart returns an empty value of the part's type.
func newPart(key string) (any, error) {
	switch key {
	case UnitCreativeDirection:
		return &CreativeDirection{}, nil
	case UnitColorSystem:
		return &ColorSystem{}, nil
	case UnitVisualLanguage:
		return &VisualLanguage{}, nil
	case UnitRepresentationRules:
		return &RepresentationRules{}, nil
	case UnitTypography:
		return &Typography{}, nil
	case UnitMotionLanguage:
		return &MotionLanguage{}, nil
	case UnitDoNotUse:
		return &DoNotUse{}, nil
	case UnitFeasibility:
		return &Feasibility{}, nil
	}
	return nil, ErrUnknownUnit
}

// DecodePart reads raw as the part key, refusing fields the part does not have
// and values of the wrong type. It returns the part in its canonical JSON form.
func DecodePart(key string, raw []byte) (any, []byte, error) {
	v, err := newPart(key)
	if err != nil {
		return nil, nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrInvalidEdit, err)
	}
	normalizeNils(v)
	out, err := json.Marshal(v)
	if err != nil {
		return nil, nil, err
	}
	return v, out, nil
}

// PartFields lists the top-level fields of the part key, the paths the
// Creator's edits are kept by.
func PartFields(key string) []string {
	v, err := newPart(key)
	if err != nil {
		return nil
	}
	raw, _ := json.Marshal(v)
	var m map[string]json.RawMessage
	_ = json.Unmarshal(raw, &m)
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// OverlayFields returns base with each field in fields taken from over.
// Both are JSON objects of the same part.
func OverlayFields(base, over []byte, fields []string) ([]byte, error) {
	var b, o map[string]json.RawMessage
	if err := json.Unmarshal(base, &b); err != nil {
		return nil, err
	}
	if len(over) > 0 {
		if err := json.Unmarshal(over, &o); err != nil {
			return nil, err
		}
	}
	for _, f := range fields {
		if v, ok := o[f]; ok {
			b[f] = v
		}
	}
	return json.Marshal(b)
}

// CheckContext is what a part is checked against.
type CheckContext struct {
	BeatKeys []string
	FactIDs  map[string]bool
	RefIDs   map[string]bool
	// Fonts lists the fonts rendering can draw.
	Fonts []string
	// NeedSurface and NeedLayout say the picked references describe the surface
	// and the layout, so the visual language must have rules for them.
	NeedSurface, NeedLayout bool
}

var hexColor = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// CheckPart lists what is wrong with a decoded part, in Vietnamese; empty when
// the part passes.
func CheckPart(key string, part any, cc CheckContext) []string {
	var p problems
	switch v := part.(type) {
	case *CreativeDirection:
		p.need(v.StyleName, "Chưa có tên phong cách")
		p.need(v.Description, "Chưa có mô tả")
		p.enum(v.AbstractionLevel, AbstractionLevels, "Mức trừu tượng")
		for _, e := range v.AbstractionMap {
			p.enum(e.Level, AbstractionLevels, "Mức trừu tượng của «"+e.For+"»")
		}
		for _, r := range v.RefsUsed {
			if !cc.RefIDs[r] {
				p.addf("Tham chiếu %s không nằm trong các mô tả đang chọn", r)
			}
		}
	case *ColorSystem:
		if !hexColor.MatchString(v.Background) {
			p.addf("Màu nền «%s» không phải mã hex #rrggbb", v.Background)
		}
		tokens := map[string]bool{}
		for _, s := range v.Palette {
			if !hexColor.MatchString(s.Hex) {
				p.addf("Màu %s «%s» không phải mã hex #rrggbb", s.Token, s.Hex)
			}
			if strings.TrimSpace(s.Token) == "" {
				p.add("Có màu chưa có token")
			} else if tokens[s.Token] {
				p.addf("Token %s bị trùng", s.Token)
			}
			tokens[s.Token] = true
		}
		if len(v.Palette) == 0 {
			p.add("Bảng màu trống")
		}
		inJourney := map[string]int{}
		for _, j := range v.Journey {
			for _, b := range j.BeatKeys {
				inJourney[b]++
			}
			if !tokens[j.AccentToken] {
				p.addf("Màu nhấn %s của hành trình không có trong bảng màu", j.AccentToken)
			}
			if j.BackgroundToken != "" && !tokens[j.BackgroundToken] {
				p.addf("Màu nền %s của hành trình không có trong bảng màu", j.BackgroundToken)
			}
		}
		p.beatsCovered(inJourney, cc.BeatKeys, "hành trình màu")
		if v.Contrast.TextMin <= 0 {
			p.add("Chưa có mức tương phản tối thiểu của chữ")
		}
		if !tokens[v.Contrast.TextToken] {
			p.addf("Màu chữ %s không có trong bảng màu", v.Contrast.TextToken)
		}
		for _, e := range v.Exceptions {
			if !tokens[e.Token] {
				p.addf("Ngoại lệ dùng token %s không có trong bảng màu", e.Token)
			}
			p.knownBeats(e.BeatKeys, cc.BeatKeys)
		}
	case *VisualLanguage:
		for _, r := range []struct {
			list  []string
			label string
		}{{v.ShapeRules, "hình dạng"}, {v.LineRules, "đường nét"}, {v.LightingRules, "ánh sáng"}, {v.DepthRules, "chiều sâu"}, {v.CompositionRules, "bố cục"}} {
			if len(r.list) == 0 {
				p.addf("Chưa có quy tắc %s", r.label)
			}
		}
		if cc.NeedSurface && len(v.SurfaceRules) == 0 {
			p.add("Chưa có quy tắc bề mặt (tham chiếu đã chọn có nói về nền màu, chất liệu hay kết cấu)")
		}
		if cc.NeedLayout && len(v.LayoutRules) == 0 {
			p.add("Chưa có quy tắc layout (tham chiếu đã chọn có nói về layout)")
		}
	case *RepresentationRules:
		seen := map[string]bool{}
		id := func(cid string) {
			if strings.TrimSpace(cid) == "" {
				p.add("Có khái niệm chưa có concept_id")
				return
			}
			if seen[cid] {
				p.addf("concept_id %s bị trùng", cid)
			}
			seen[cid] = true
		}
		for _, list := range [][]RepresentedThing{v.Characters.Items, v.Objects, v.Invisible} {
			for _, t := range list {
				id(t.ConceptID)
				p.knownFacts(t.Facts, cc.FactIDs)
			}
		}
		for _, s := range v.ScientificConcepts {
			id(s.ConceptID)
			p.knownFacts(s.Facts, cc.FactIDs)
		}
	case *Typography:
		p.need(v.Font, "Chưa chọn font")
		if v.Font != "" && !contains(cc.Fonts, v.Font) {
			p.addf("Font «%s» không có trong danh sách font của rendering", v.Font)
		}
	case *MotionLanguage:
		paced := map[string]int{}
		for _, bp := range v.Pace {
			p.enum(bp.Pace, Paces, "Nhịp")
			for _, b := range bp.BeatKeys {
				paced[b]++
			}
		}
		p.beatsCovered(paced, cc.BeatKeys, "nhịp chuyển động")
	case *DoNotUse:
		for _, it := range v.Items {
			p.need(it.Text, "Có dòng chưa nói điều không được dùng")
			p.enum(it.Source, DoNotUseSources, "Loại của «"+it.Text+"»")
			p.knownFacts(it.Facts, cc.FactIDs)
		}
	case *Feasibility:
		for _, c := range v.Checks {
			p.enum(c.Verdict, Verdicts, "Kết luận của «"+c.Item+"»")
			if c.Verdict == "risk" && strings.TrimSpace(c.Mitigation) == "" {
				p.addf("«%s» là rủi ro nhưng chưa có cách giảm rủi ro", c.Item)
			}
		}
		if len(v.Checks) == 0 {
			p.add("Chưa có hạng mục kiểm khả thi")
		}
	}
	return p.list
}

type problems struct{ list []string }

func (p *problems) add(s string)            { p.list = append(p.list, s) }
func (p *problems) addf(f string, a ...any) { p.add(fmt.Sprintf(f, a...)) }
func (p *problems) need(v, msg string) {
	if strings.TrimSpace(v) == "" {
		p.add(msg)
	}
}
func (p *problems) enum(v string, allowed []string, label string) {
	if !contains(allowed, v) {
		p.addf("%s «%s» phải là một trong: %s", label, v, strings.Join(allowed, ", "))
	}
}
func (p *problems) knownFacts(ids []string, known map[string]bool) {
	for _, id := range ids {
		if !known[id] {
			p.addf("Dữ kiện %s không có trong hồ sơ nghiên cứu", id)
		}
	}
}
func (p *problems) knownBeats(keys, beats []string) {
	for _, k := range keys {
		if !contains(beats, k) {
			p.addf("Beat %s không có trong beat sheet", k)
		}
	}
}
func (p *problems) beatsCovered(count map[string]int, beats []string, what string) {
	for k, n := range count {
		if !contains(beats, k) {
			p.addf("Beat %s trong %s không có trong beat sheet", k, what)
		} else if n > 1 {
			p.addf("Beat %s nằm ở nhiều chỗ trong %s", k, what)
		}
	}
	for _, b := range beats {
		if count[b] == 0 {
			p.addf("Beat %s chưa có trong %s", b, what)
		}
	}
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// normalizeNils turns nil slices into empty ones so the JSON has [] not null.
func normalizeNils(v any) {
	raw, _ := json.Marshal(v)
	fixed := nullArrays.ReplaceAll(raw, []byte(`"$1":[]`))
	_ = json.Unmarshal(fixed, v)
}

var nullArrays = regexp.MustCompile(`"([a-z_]+)":null`)

// MachineScope sets the counts of a feasibility part the machine owns: the
// beats of the beat sheet and the characters of part 4.
func MachineScope(f *Feasibility, beats int, rep *RepresentationRules) {
	f.Scope.Beats = beats
	f.Scope.Characters = 0
	if rep != nil {
		f.Scope.Characters = len(rep.Characters.Items)
	}
}
