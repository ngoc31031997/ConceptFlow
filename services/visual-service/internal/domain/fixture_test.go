package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"
)

// part freezes v the way authoring-service does.
func part(t *testing.T, key string, v any) ScriptVersionPart {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	return ScriptVersionPart{Key: key, SHA: hex.EncodeToString(sum[:]), Content: raw}
}

// scriptVersion is a locked script with two beats, three facts (one a
// hypothesis, one a misconception) and a two-term glossary.
func scriptVersion(t *testing.T, edit func(map[string]any)) ScriptVersion {
	t.Helper()
	m := map[string]any{
		"brief": Brief{Audience: "người xem phổ thông", DurationSeconds: 120, Tone: "tò mò", LearningObjectives: []string{"hiểu A"}, Avoid: []string{"nói quá"}},
		"facts": []map[string]any{
			{"ID": "F01", "type": "fact", "claim_vi": "A là thật", "keep": true},
			{"ID": "F02", "type": "hypothesis", "claim_vi": "B có thể đúng", "keep": true},
			{"ID": "F03", "type": "misconception", "claim_vi": "C là ngộ nhận", "keep": true},
		},
		"story":      map[string]StoryField{"question": {Text: "vì sao?"}, "chain": {Text: "vì A"}, "payoff": {Text: "A"}},
		"order":      []string{"b1", "b2"},
		"beat:b1":    map[string]any{"fn": "Hook", "title": "Mở", "goal": "kéo vào", "dur": 20},
		"beat:b2":    map[string]any{"fn": "Payoff", "title": "Đóng", "goal": "trả lời", "dur": 40},
		"script:b1":  map[string]any{"lines": []Line{{ID: "b1.1", Text: "A là thật", Facts: []string{"F01"}}, {ID: "b1.2", Text: "Có thể B", Facts: []string{"F02"}}}},
		"script:b2":  map[string]any{"lines": []Line{{ID: "b2.1", Text: "Không phải C", Facts: []string{"F03"}}}},
		"handoff:b1": Handoff{Terms: []string{"enzyme"}},
		"handoff:b2": Handoff{},
		"glossary":   map[string]any{"terms": []GlossaryTerm{{Term: "Enzyme"}, {Term: "cửa quả"}}},
	}
	if edit != nil {
		edit(m)
	}
	var parts []ScriptVersionPart
	for _, k := range []string{"brief", "facts", "story", "order", "beat:b1", "beat:b2", "script:b1", "script:b2", "handoff:b1", "handoff:b2", "glossary"} {
		if v, ok := m[k]; ok {
			parts = append(parts, part(t, k, v))
		}
	}
	return ScriptVersion{ProjectID: "test-cr112b-p", Version: 2, Parts: parts}
}

func conceptInput(t *testing.T, refs ...PickedReference) ConceptInput {
	t.Helper()
	in, err := NewConceptInput(scriptVersion(t, nil), refs)
	if err != nil {
		t.Fatal(err)
	}
	return in
}

// ValidParts returns eight parts that pass every check against conceptInput.
func validParts() map[string]string {
	return map[string]string{
		UnitCreativeDirection:   `{"style_name":"Phẳng sáng","description":"d","abstraction_level":"hybrid","tone":["ấm"],"principles":["một ý"],"why":"w","feeling":"f","refs_used":["R01"],"abstraction_map":[{"for":"ong","level":"character","example":"e"}],"examples":[{"do":"a","dont":"b"}]}`,
		UnitColorSystem:         `{"background":"#101010","palette":[{"token":"ink","hex":"#ffffff","role":"chữ","name":"trắng"},{"token":"accent","hex":"#e1ad01","role":"nhấn","name":"vàng"}],"journey":[{"beat_keys":["b1"],"mood":"ấm","accent_token":"accent","background_token":""},{"beat_keys":["b2"],"mood":"dịu","accent_token":"accent","background_token":""}],"contrast":{"text_min":4.5,"text_token":"ink","subject_vs_bg":"sáng hơn","accent_per_frame":1},"exceptions":[],"examples":[]}`,
		UnitVisualLanguage:      `{"shape_rules":["tròn"],"line_rules":["không viền"],"lighting_rules":["sau"],"depth_rules":["ba lớp"],"surface_rules":["phẳng"],"layout_rules":["ba lớp"],"composition_rules":["1/3"],"examples":[]}`,
		UnitRepresentationRules: `{"characters":{"approach":"đơn giản","allowed_variations":["dáng"],"items":[{"concept_id":"wasp","term":"ong","draw":"thân đen","must_not_imply":["ong mật"],"facts":["F01"]}]},"objects":[{"concept_id":"gate","term":"Cửa quả","draw":"lỗ nhỏ","must_not_imply":[],"facts":[]}],"invisible":[],"scientific_concepts":[{"concept_id":"enz","term":"enzyme","representation":"quầng sáng","must_not_imply":["đã chứng minh"],"facts":["F02"]}],"examples":[]}`,
		UnitTypography:          `{"font_direction":"sans","label_style":"ngắn","font":"Be Vietnam Pro","labels":{"max_words":4,"when":"sau lời","min_px":36},"captions":"c","terms":[],"formulas":"không dùng"}`,
		UnitMotionLanguage:      `{"principles":["mềm"],"preferred_transitions":["zoom"],"avoid":["rung"],"pace":[{"beat_keys":["b1"],"pace":"medium"},{"beat_keys":["b2"],"pace":"slow"}],"character_motion":"râu","examples":[]}`,
		UnitDoNotUse:            `{"items":[{"text":"vẽ B như thật","why":"giả thuyết","facts":["F02"],"source":"accuracy"},{"text":"mọi quả có C","why":"ngộ nhận","facts":["F03"],"source":"accuracy"},{"text":"viền đen","why":"phá phong cách","facts":[],"source":"style"}]}`,
		UnitFeasibility:         `{"scope":{"beats":9,"est_shots":8,"settings":3,"characters":9},"checks":[{"item":"glow","assessment":"chậm","verdict":"risk","mitigation":"blur"}]}`,
	}
}

func decodedParts(t *testing.T, raw map[string]string) Parts {
	t.Helper()
	var p Parts
	for k, v := range raw {
		d, _, err := DecodePart(k, []byte(v))
		if err != nil {
			t.Fatalf("%s: %v", k, err)
		}
		p.Set(k, d)
	}
	return p
}

func checkCtx(in ConceptInput, fonts []string) CheckContext {
	facts := map[string]bool{}
	for _, f := range in.Facts {
		facts[f.ID] = true
	}
	refs := map[string]bool{}
	for _, r := range in.Refs {
		refs[r.ID] = true
	}
	surface, layout := RefsNeed(in.Refs)
	return CheckContext{BeatKeys: in.BeatKeys(), FactIDs: facts, RefIDs: refs, Fonts: fonts, NeedSurface: surface, NeedLayout: layout}
}
