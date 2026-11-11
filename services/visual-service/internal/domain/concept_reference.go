package domain

import (
	"encoding/json"
	"strings"
)

// FlexText is a text field of a reference that may be stored as one string or
// as a list of strings; a list reads as its items joined by "; ".
type FlexText string

// UnmarshalJSON reads a string, a list of strings, or null.
func (t *FlexText) UnmarshalJSON(b []byte) error {
	var s string
	if json.Unmarshal(b, &s) == nil {
		*t = FlexText(s)
		return nil
	}
	var list []string
	if err := json.Unmarshal(b, &list); err != nil {
		return err
	}
	*t = FlexText(strings.Join(list, "; "))
	return nil
}

// RefSource says where a reference description comes from.
type RefSource struct {
	URL   string `json:"url"`
	Title string `json:"title"`
	Kind  string `json:"kind"`
}

// RefDrawing is how a reference style is drawn.
type RefDrawing struct {
	Medium      FlexText `json:"medium"`
	Shape       FlexText `json:"shape"`
	Outline     FlexText `json:"outline"`
	Fill        FlexText `json:"fill"`
	Shading     FlexText `json:"shading"`
	Light       FlexText `json:"light"`
	Depth       FlexText `json:"depth"`
	Perspective FlexText `json:"perspective"`
	Texture     FlexText `json:"texture"`
}

// RefLayout is how a reference style is laid out.
type RefLayout struct {
	Structure          FlexText `json:"structure"`
	Layering           FlexText `json:"layering"`
	PersistentElements FlexText `json:"persistent_elements"`
}

// PickedReference is one description of the shared reference library as a video's
// visual concept reads it: how a style is drawn and laid out, in words. Colour, characters, text and motion
// are not part of it, since they depend on the topic and the scene. Step 8
// reads the descriptions a video picked; there are no images.
type PickedReference struct {
	ID      string     `json:"ref_id"`
	Name    string     `json:"name"`
	Summary string     `json:"summary"`
	Tags    []string   `json:"tags"`
	Source  RefSource  `json:"source"`
	Drawing RefDrawing `json:"drawing"`
	Layout  RefLayout  `json:"layout"`
}

// DocumentReference is a picked description as the locked document copies it.
type DocumentReference struct {
	ID      string     `json:"ref_id"`
	Name    string     `json:"name"`
	Summary string     `json:"summary"`
	Drawing RefDrawing `json:"drawing"`
	Layout  RefLayout  `json:"layout"`
}

// DocumentReferences copies refs the way the locked document holds them.
func DocumentReferences(refs []PickedReference) []DocumentReference {
	out := make([]DocumentReference, 0, len(refs))
	for _, r := range refs {
		out = append(out, DocumentReference{ID: r.ID, Name: r.Name, Summary: r.Summary, Drawing: r.Drawing, Layout: r.Layout})
	}
	return out
}

// RefsNeed reports whether the picked descriptions say anything about the
// surface (fill, texture, medium) and about the layout, so the visual language
// has to carry rules for them.
func RefsNeed(refs []PickedReference) (surface, layout bool) {
	for _, r := range refs {
		d, l := r.Drawing, r.Layout
		surface = surface || d.Fill != "" || d.Texture != "" || d.Medium != ""
		layout = layout || l.Structure != "" || l.Layering != "" || l.PersistentElements != ""
	}
	return surface, layout
}
