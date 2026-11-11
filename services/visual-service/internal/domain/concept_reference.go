package domain

// PickedReference is one description of the shared reference library as a
// video's visual concept reads it: how a style is drawn, coloured (as
// relations, not a palette) and composed, in words. Step 8 reads the
// descriptions a video picked; there are no images.
type PickedReference struct {
	ID string `json:"ref_id"`
	ReferenceStyleInput
}

// DocumentReference is a picked description as the locked document copies
// it: the parts the AI reads. Tags, source and the image-generation prompts
// are library bookkeeping and are left out.
type DocumentReference struct {
	ID          string               `json:"ref_id"`
	StyleName   string               `json:"style_name"`
	Intent      string               `json:"intent"`
	CoreRules   []string             `json:"core_rules"`
	Drawing     ReferenceDrawing     `json:"drawing"`
	ColorRules  ReferenceColorRules  `json:"color_rules"`
	Composition ReferenceComposition `json:"composition"`
	Avoid       []string             `json:"avoid"`
}

// DocumentReferences copies refs the way the locked document holds them.
func DocumentReferences(refs []PickedReference) []DocumentReference {
	out := make([]DocumentReference, 0, len(refs))
	for _, r := range refs {
		out = append(out, DocumentReference{
			ID: r.ID, StyleName: r.StyleName, Intent: r.Intent, CoreRules: r.CoreRules, Drawing: r.Drawing,
			ColorRules: r.ColorRules, Composition: r.Composition, Avoid: r.Avoid,
		})
	}
	return out
}

// RefsNeed reports whether the picked descriptions say anything about the
// surface (fill, texture, medium) and about the composition, so the visual
// language has to carry rules for them.
func RefsNeed(refs []PickedReference) (surface, layout bool) {
	for _, r := range refs {
		d, c := r.Drawing, r.Composition
		surface = surface || d.Fill != "" || d.Texture != "" || d.Medium != ""
		layout = layout || c.Structure != "" || c.Layering != "" || len(c.OptionalDevices) > 0
	}
	return surface, layout
}
