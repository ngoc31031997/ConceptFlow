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
	MaxReferenceNameLength    = 80
	MaxReferenceSummaryLength = 1000
	MaxReferenceTags          = 8
	MaxReferenceTagLength     = 24
	MaxReferenceFieldLength   = 300
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

// ReferenceLayout describes how the frame is laid out.
type ReferenceLayout struct {
	Structure          string `json:"structure"`
	Layering           string `json:"layering"`
	PersistentElements string `json:"persistent_elements"`
}

// ReferenceStyleInput is what the Creator writes when adding or editing a
// description, or pastes as JSON. Only Name and Summary are required.
type ReferenceStyleInput struct {
	Name    string           `json:"name"`
	Summary string           `json:"summary"`
	Tags    []string         `json:"tags"`
	Source  ReferenceSource  `json:"source"`
	Drawing ReferenceDrawing `json:"drawing"`
	Layout  ReferenceLayout  `json:"layout"`
}

// ReferenceStyle is a structured text description of a visual style in the
// shared reference library. A video's visual concept reads the ones it picked;
// there is no image, the AI only reads the text.
type ReferenceStyle struct {
	ID        string           `json:"id"`
	Name      string           `json:"name"`
	Summary   string           `json:"summary"`
	Tags      []string         `json:"tags"`
	Source    ReferenceSource  `json:"source"`
	Drawing   ReferenceDrawing `json:"drawing"`
	Layout    ReferenceLayout  `json:"layout"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
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

// trimAll applies trimField to each (path, field) pair in place.
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

func (l ReferenceLayout) normalize() (ReferenceLayout, error) {
	err := trimAll(MaxReferenceFieldLength, map[string]*string{
		"layout.structure": &l.Structure, "layout.layering": &l.Layering, "layout.persistent_elements": &l.PersistentElements,
	}, []string{"layout.structure", "layout.layering", "layout.persistent_elements"})
	return l, err
}

// Normalize trims every field, drops empty and repeated tags (keeping the
// first spelling, compared case-insensitively) and checks the limits. It
// returns the cleaned input or an error wrapping ErrInvalidReferenceStyle
// that names the field at fault.
func (in ReferenceStyleInput) Normalize() (ReferenceStyleInput, error) {
	out := ReferenceStyleInput{Tags: []string{}}
	out.Name = strings.TrimSpace(in.Name)
	if n := utf8.RuneCountInString(out.Name); n == 0 || n > MaxReferenceNameLength {
		return ReferenceStyleInput{}, invalid("name: cần 1–%d ký tự", MaxReferenceNameLength)
	}
	out.Summary = strings.TrimSpace(in.Summary)
	if n := utf8.RuneCountInString(out.Summary); n == 0 || n > MaxReferenceSummaryLength {
		return ReferenceStyleInput{}, invalid("summary: cần 1–%d ký tự", MaxReferenceSummaryLength)
	}
	seen := map[string]bool{}
	for _, raw := range in.Tags {
		tag := strings.TrimSpace(raw)
		key := strings.ToLower(tag)
		if tag == "" || seen[key] {
			continue
		}
		if utf8.RuneCountInString(tag) > MaxReferenceTagLength {
			return ReferenceStyleInput{}, invalid("tags: nhãn %q dài quá %d ký tự", tag, MaxReferenceTagLength)
		}
		seen[key] = true
		out.Tags = append(out.Tags, tag)
	}
	if len(out.Tags) > MaxReferenceTags {
		return ReferenceStyleInput{}, invalid("tags: tối đa %d nhãn", MaxReferenceTags)
	}
	var err error
	if out.Source, err = in.Source.normalize(); err != nil {
		return ReferenceStyleInput{}, err
	}
	if out.Drawing, err = in.Drawing.normalize(); err != nil {
		return ReferenceStyleInput{}, err
	}
	if out.Layout, err = in.Layout.normalize(); err != nil {
		return ReferenceStyleInput{}, err
	}
	return out, nil
}

// ReferenceStyleID formats the id of the style numbered seq: R plus the
// number padded to two digits (R01, R12, R105).
func ReferenceStyleID(seq int64) string {
	return fmt.Sprintf("R%02d", seq)
}
