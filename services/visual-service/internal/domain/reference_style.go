// Package domain holds visual-service's business types and rules.
package domain

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"
)

// Limits of a reference style, counted in characters (runes), not bytes.
const (
	MaxReferenceNameLength   = 80
	MaxReferenceIntentLength = 1000
	MaxReferenceTags         = 8
	MaxReferenceTagLength    = 24
	MaxReferenceFieldLength  = 300
	MaxReferenceListItems    = 12
	MaxReferencePromptLength = 1000
)

// Kinds a reference source may have; empty is allowed.
var referenceSourceKinds = []string{"video", "image", "site", "other"}

// ErrInvalidReferenceStyle wraps every rule a reference style breaks, so the
// HTTP adapter can answer 400 with the specific message.
var ErrInvalidReferenceStyle = errors.New("mô tả tham chiếu không hợp lệ")

// ReferenceSource is where a style was read from.
type ReferenceSource struct {
	URL   string `json:"url"`
	Title string `json:"title"`
	Kind  string `json:"kind"`
}

// ReferenceDrawing describes how the style is drawn.
type ReferenceDrawing struct {
	Medium      string `json:"medium"`
	Shape       string `json:"shape"`
	Outline     string `json:"outline"`
	Fill        string `json:"fill"`
	Shading     string `json:"shading"`
	Light       string `json:"light"`
	Depth       string `json:"depth"`
	Perspective string `json:"perspective"`
	Texture     string `json:"texture"`
}

// ReferenceColorRules describes colour as relations between background,
// subject and accents; the actual palette is chosen by each video's topic.
type ReferenceColorRules struct {
	Background string `json:"background"`
	Subject    string `json:"subject"`
	Accents    string `json:"accents"`
	Note       string `json:"note"`
}

// ReferenceComposition describes how the frame is organised.
type ReferenceComposition struct {
	Structure       string   `json:"structure"`
	OptionalDevices []string `json:"optional_devices"`
	Layering        string   `json:"layering"`
}

// ReferenceStyleInput is what the Creator writes when adding or editing a
// description, or pastes as JSON. Only StyleName and Intent are required.
type ReferenceStyleInput struct {
	StyleName      string               `json:"style_name"`
	Intent         string               `json:"intent"`
	Tags           []string             `json:"tags"`
	Source         ReferenceSource      `json:"source"`
	CoreRules      []string             `json:"core_rules"`
	Drawing        ReferenceDrawing     `json:"drawing"`
	ColorRules     ReferenceColorRules  `json:"color_rules"`
	Composition    ReferenceComposition `json:"composition"`
	Avoid          []string             `json:"avoid"`
	PromptTemplate string               `json:"prompt_template"`
	NegativePrompt string               `json:"negative_prompt"`
}

// ReferenceStyle is a structured text description of a visual style in the
// shared reference library. A video's visual concept reads the ones it picked;
// there is no image, the AI only reads the text.
type ReferenceStyle struct {
	ID string `json:"id"`
	ReferenceStyleInput
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidReferenceStyle, fmt.Sprintf(format, args...))
}

// trimField trims value and checks it is at most max characters, naming path
// in the error.
func trimField(path, value string, max int) (string, error) {
	v := strings.TrimSpace(value)
	if utf8.RuneCountInString(v) > max {
		return "", invalid("%s: dài quá %d ký tự", path, max)
	}
	return v, nil
}

// trimAll applies trimField to each (path, field) pair in place, in order.
func trimAll(max int, fields map[string]*string, order []string) error {
	for _, path := range order {
		v, err := trimField(path, *fields[path], max)
		if err != nil {
			return err
		}
		*fields[path] = v
	}
	return nil
}

// trimList trims every item, drops empty ones and checks the item count and
// each item's length. It always returns a non-nil list on success.
func trimList(path string, items []string) ([]string, error) {
	out := []string{}
	for i, raw := range items {
		v, err := trimField(fmt.Sprintf("%s[%d]", path, i), raw, MaxReferenceFieldLength)
		if err != nil {
			return nil, err
		}
		if v != "" {
			out = append(out, v)
		}
	}
	if len(out) > MaxReferenceListItems {
		return nil, invalid("%s: tối đa %d mục", path, MaxReferenceListItems)
	}
	return out, nil
}

func (s ReferenceSource) normalize() (ReferenceSource, error) {
	var err error
	if s.Title, err = trimField("source.title", s.Title, MaxReferenceFieldLength); err != nil {
		return s, err
	}
	s.URL = strings.TrimSpace(s.URL)
	s.Kind = strings.TrimSpace(s.Kind)
	if s.URL != "" {
		u, perr := url.Parse(s.URL)
		if perr != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return s, invalid("source.url: cần là địa chỉ http hoặc https hợp lệ")
		}
	}
	if s.Kind != "" {
		ok := false
		for _, k := range referenceSourceKinds {
			ok = ok || k == s.Kind
		}
		if !ok {
			return s, invalid("source.kind: chỉ nhận %s", strings.Join(referenceSourceKinds, ", "))
		}
	}
	return s, nil
}

func (d ReferenceDrawing) normalize() (ReferenceDrawing, error) {
	err := trimAll(MaxReferenceFieldLength, map[string]*string{
		"drawing.medium": &d.Medium, "drawing.shape": &d.Shape, "drawing.outline": &d.Outline,
		"drawing.fill": &d.Fill, "drawing.shading": &d.Shading, "drawing.light": &d.Light,
		"drawing.depth": &d.Depth, "drawing.perspective": &d.Perspective, "drawing.texture": &d.Texture,
	}, []string{"drawing.medium", "drawing.shape", "drawing.outline", "drawing.fill", "drawing.shading",
		"drawing.light", "drawing.depth", "drawing.perspective", "drawing.texture"})
	return d, err
}

func (c ReferenceColorRules) normalize() (ReferenceColorRules, error) {
	err := trimAll(MaxReferenceFieldLength, map[string]*string{
		"color_rules.background": &c.Background, "color_rules.subject": &c.Subject,
		"color_rules.accents": &c.Accents, "color_rules.note": &c.Note,
	}, []string{"color_rules.background", "color_rules.subject", "color_rules.accents", "color_rules.note"})
	return c, err
}

func (c ReferenceComposition) normalize() (ReferenceComposition, error) {
	err := trimAll(MaxReferenceFieldLength, map[string]*string{
		"composition.structure": &c.Structure, "composition.layering": &c.Layering,
	}, []string{"composition.structure", "composition.layering"})
	if err != nil {
		return c, err
	}
	c.OptionalDevices, err = trimList("composition.optional_devices", c.OptionalDevices)
	return c, err
}

// normalizeTags drops empty and repeated tags (compared case-insensitively,
// keeping the first spelling) and checks the tag limits.
func normalizeTags(in []string) ([]string, error) {
	out := []string{}
	seen := map[string]bool{}
	for _, raw := range in {
		tag := strings.TrimSpace(raw)
		key := strings.ToLower(tag)
		if tag == "" || seen[key] {
			continue
		}
		if utf8.RuneCountInString(tag) > MaxReferenceTagLength {
			return nil, invalid("tags: nhãn %q dài quá %d ký tự", tag, MaxReferenceTagLength)
		}
		seen[key] = true
		out = append(out, tag)
	}
	if len(out) > MaxReferenceTags {
		return nil, invalid("tags: tối đa %d nhãn", MaxReferenceTags)
	}
	return out, nil
}

// Normalize trims every field, drops empty list items and empty or repeated
// tags, and checks the limits. It returns the cleaned input or an error
// wrapping ErrInvalidReferenceStyle that names the field at fault.
func (in ReferenceStyleInput) Normalize() (ReferenceStyleInput, error) {
	var out ReferenceStyleInput
	out.StyleName = strings.TrimSpace(in.StyleName)
	if n := utf8.RuneCountInString(out.StyleName); n == 0 || n > MaxReferenceNameLength {
		return ReferenceStyleInput{}, invalid("style_name: cần 1–%d ký tự", MaxReferenceNameLength)
	}
	out.Intent = strings.TrimSpace(in.Intent)
	if n := utf8.RuneCountInString(out.Intent); n == 0 || n > MaxReferenceIntentLength {
		return ReferenceStyleInput{}, invalid("intent: cần 1–%d ký tự", MaxReferenceIntentLength)
	}
	var err error
	if out.Tags, err = normalizeTags(in.Tags); err != nil {
		return ReferenceStyleInput{}, err
	}
	if out.Source, err = in.Source.normalize(); err != nil {
		return ReferenceStyleInput{}, err
	}
	if out.CoreRules, err = trimList("core_rules", in.CoreRules); err != nil {
		return ReferenceStyleInput{}, err
	}
	if out.Drawing, err = in.Drawing.normalize(); err != nil {
		return ReferenceStyleInput{}, err
	}
	if out.ColorRules, err = in.ColorRules.normalize(); err != nil {
		return ReferenceStyleInput{}, err
	}
	if out.Composition, err = in.Composition.normalize(); err != nil {
		return ReferenceStyleInput{}, err
	}
	if out.Avoid, err = trimList("avoid", in.Avoid); err != nil {
		return ReferenceStyleInput{}, err
	}
	if out.PromptTemplate, err = trimField("prompt_template", in.PromptTemplate, MaxReferencePromptLength); err != nil {
		return ReferenceStyleInput{}, err
	}
	if out.NegativePrompt, err = trimField("negative_prompt", in.NegativePrompt, MaxReferencePromptLength); err != nil {
		return ReferenceStyleInput{}, err
	}
	return out, nil
}

// ReferenceStyleID formats the id of the style numbered seq: R plus the
// number padded to two digits (R01, R12, R105).
func ReferenceStyleID(seq int64) string {
	return fmt.Sprintf("R%02d", seq)
}
