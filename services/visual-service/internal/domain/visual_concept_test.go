package domain

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

var r01 = ReferenceStyle{ID: "R01", Name: "Sáng xuyên lớp", Summary: "nền tối ấm",
	Drawing: RefDrawing{Shape: "tròn mềm", Fill: "phẳng", Light: "sau lớp mỏng"}, Layout: RefLayout{Layering: "ba lớp"}}

func TestValidPartsPassEveryCheck(t *testing.T) {
	in := conceptInput(t, r01)
	for k, raw := range validParts() {
		v, _, err := DecodePart(k, []byte(raw))
		if err != nil {
			t.Fatal(err)
		}
		if p := CheckPart(k, v, checkCtx(in, []string{"Be Vietnam Pro"})); len(p) > 0 {
			t.Errorf("%s: %v", k, p)
		}
	}
}

func TestDecodePart_RefusesUnknownFieldsAndWrongTypes(t *testing.T) {
	if _, _, err := DecodePart(UnitVisualLanguage, []byte(`{"shape_rules":["a"],"colour":"x"}`)); !errors.Is(err, ErrInvalidEdit) {
		t.Errorf("unknown field: %v", err)
	}
	if _, _, err := DecodePart(UnitVisualLanguage, []byte(`{"shape_rules":"a"}`)); !errors.Is(err, ErrInvalidEdit) {
		t.Errorf("wrong type: %v", err)
	}
	if _, _, err := DecodePart("nope", []byte(`{}`)); !errors.Is(err, ErrUnknownUnit) {
		t.Errorf("unknown part: %v", err)
	}
	_, canon, err := DecodePart(UnitVisualLanguage, []byte(`{}`))
	if err != nil || strings.Contains(string(canon), "null") {
		t.Errorf("empty lists must be [] not null: %s %v", canon, err)
	}
}

func problemsOf(t *testing.T, key, raw string, cc CheckContext) string {
	t.Helper()
	v, _, err := DecodePart(key, []byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	return strings.Join(CheckPart(key, v, cc), " | ")
}

func TestCheckPart_EachPartsRules(t *testing.T) {
	in := conceptInput(t, r01)
	cc := checkCtx(in, []string{"Be Vietnam Pro"})
	cases := []struct {
		key, raw, want string
	}{
		{UnitCreativeDirection, `{"style_name":"s","description":"d","abstraction_level":"photo","abstraction_map":[{"for":"x","level":"odd"}],"refs_used":["R09"]}`, "photo"},
		{UnitCreativeDirection, `{"style_name":"s","description":"d","abstraction_level":"hybrid","refs_used":["R09"]}`, "R09"},
		{UnitColorSystem, `{"background":"red","palette":[{"token":"a","hex":"#fff"},{"token":"a","hex":"#000000"}],"journey":[{"beat_keys":["b1","b9"],"accent_token":"zz"}],"contrast":{"text_min":4.5,"text_token":"a"}}`, "không phải mã hex"},
		{UnitColorSystem, `{"background":"#000000","palette":[{"token":"a","hex":"#ffffff"},{"token":"a","hex":"#000000"}],"journey":[{"beat_keys":["b1"],"accent_token":"a"}],"contrast":{"text_min":4.5,"text_token":"a"}}`, "Token a bị trùng"},
		{UnitColorSystem, `{"background":"#000000","palette":[{"token":"a","hex":"#ffffff"}],"journey":[{"beat_keys":["b1"],"accent_token":"zz"}],"contrast":{"text_min":4.5,"text_token":"a"}}`, "Beat b2 chưa có trong hành trình màu"},
		{UnitColorSystem, `{"background":"#000000","palette":[{"token":"a","hex":"#ffffff"}],"journey":[{"beat_keys":["b1","b2"],"accent_token":"zz"}],"contrast":{"text_min":4.5,"text_token":"a"}}`, "Màu nhấn zz"},
		{UnitVisualLanguage, `{"shape_rules":["a"],"line_rules":[],"lighting_rules":["a"],"depth_rules":["a"],"composition_rules":["a"]}`, "đường nét"},
		{UnitVisualLanguage, `{"shape_rules":["a"],"line_rules":["a"],"lighting_rules":["a"],"depth_rules":["a"],"composition_rules":["a"],"layout_rules":["a"]}`, "quy tắc bề mặt"},
		{UnitVisualLanguage, `{"shape_rules":["a"],"line_rules":["a"],"lighting_rules":["a"],"depth_rules":["a"],"composition_rules":["a"],"surface_rules":["a"]}`, "quy tắc layout"},
		{UnitRepresentationRules, `{"objects":[{"concept_id":"x","facts":["F99"]}],"invisible":[{"concept_id":"x"}]}`, "concept_id x bị trùng"},
		{UnitRepresentationRules, `{"objects":[{"concept_id":"x","facts":["F99"]}]}`, "F99"},
		{UnitTypography, `{"font":"Comic Sans"}`, "Comic Sans"},
		{UnitMotionLanguage, `{"pace":[{"beat_keys":["b1"],"pace":"warp"}]}`, "Beat b2 chưa có trong nhịp"},
		{UnitMotionLanguage, `{"pace":[{"beat_keys":["b1","b2"],"pace":"warp"}]}`, "warp"},
		{UnitDoNotUse, `{"items":[{"text":"x","source":"taste","facts":["F42"]}]}`, "taste"},
		{UnitDoNotUse, `{"items":[{"text":"x","source":"style","facts":["F42"]}]}`, "F42"},
		{UnitFeasibility, `{"checks":[{"item":"glow","verdict":"risk","mitigation":""}]}`, "chưa có cách giảm rủi ro"},
		{UnitFeasibility, `{"checks":[{"item":"glow","verdict":"maybe"}]}`, "maybe"},
	}
	for _, c := range cases {
		if got := problemsOf(t, c.key, c.raw, cc); !strings.Contains(got, c.want) {
			t.Errorf("%s %s: want %q in %q", c.key, c.raw, c.want, got)
		}
	}
	if got := problemsOf(t, UnitVisualLanguage, `{"shape_rules":["a"],"line_rules":["a"],"lighting_rules":["a"],"depth_rules":["a"],"composition_rules":["a"]}`, checkCtx(conceptInput(t), nil)); got != "" {
		t.Errorf("surface and layout rules are not required when no picked reference describes them: %q", got)
	}
	if got := problemsOf(t, UnitTypography, `{"font":"Montserrat"}`, checkCtx(in, []string{"Be Vietnam Pro", "Montserrat"})); got != "" {
		t.Errorf("a font of the list passes: %q", got)
	}
	if got := problemsOf(t, UnitTypography, `{"font":""}`, cc); !strings.Contains(got, "Chưa chọn font") {
		t.Errorf("a missing font fails: %q", got)
	}
}

func TestMachineScopeCountsBeatsAndCharacters(t *testing.T) {
	p := decodedParts(t, validParts())
	MachineScope(p.Feasibility, 2, p.RepresentationRules)
	if p.Feasibility.Scope.Beats != 2 || p.Feasibility.Scope.Characters != 1 || p.Feasibility.Scope.EstShots != 8 {
		t.Errorf("scope %+v", p.Feasibility.Scope)
	}
}

func TestContrastRatio(t *testing.T) {
	r, err := ContrastRatio("#ffffff", "#000000")
	if err != nil || r < 20.99 || r > 21.01 {
		t.Errorf("white on black %v %v", r, err)
	}
	if r, _ := ContrastRatio("#777777", "#777777"); r != 1 {
		t.Errorf("same colour %v", r)
	}
	if _, err := ContrastRatio("#zzzzzz", "#000000"); err == nil {
		t.Error("bad hex must fail")
	}
}

func checkByID(cs []MachineCheck, id string) MachineCheck {
	for _, c := range cs {
		if c.ID == id {
			return c
		}
	}
	return MachineCheck{}
}

func TestMachineChecks_PassOnValidParts(t *testing.T) {
	in := conceptInput(t, r01)
	for _, c := range MachineChecks(in, decodedParts(t, validParts())) {
		if !c.OK {
			t.Errorf("%s: %s", c.ID, c.Detail)
		}
	}
}

func TestMachineChecks_Contrast(t *testing.T) {
	in := conceptInput(t)
	raw := validParts()
	raw[UnitColorSystem] = strings.Replace(raw[UnitColorSystem], `"background_token":""},{"beat_keys":["b2"]`, `"background_token":""},{"beat_keys":["b2"],"background_token":"pale"`, 1)
	raw[UnitColorSystem] = strings.Replace(raw[UnitColorSystem], `"mood":"dịu","accent_token":"accent","background_token":""`, `"mood":"dịu","accent_token":"accent"`, 1)
	raw[UnitColorSystem] = strings.Replace(raw[UnitColorSystem], `{"token":"accent"`, `{"token":"pale","hex":"#f0f0f0","role":"nền sáng","name":"x"},{"token":"accent"`, 1)
	c := checkByID(MachineChecks(in, decodedParts(t, raw)), CheckContrast)
	if c.OK || !strings.Contains(c.Detail, "B2") || !strings.Contains(c.Label, "1/2") {
		t.Errorf("white text on a pale background must fail at B2: %+v", c)
	}
}

func TestMachineChecks_TermsAndAccuracy(t *testing.T) {
	in := conceptInput(t)
	raw := validParts()
	raw[UnitRepresentationRules] = strings.Replace(raw[UnitRepresentationRules], `"term":"enzyme"`, `"term":"men"`, 1)
	raw[UnitDoNotUse] = `{"items":[{"text":"vẽ B như thật","why":"","facts":["F02"],"source":"style"}]}`
	cs := MachineChecks(in, decodedParts(t, raw))
	if c := checkByID(cs, CheckTerms); c.OK || !strings.Contains(c.Detail, "Enzyme") {
		t.Errorf("a glossary term without a drawing must fail: %+v", c)
	}
	if c := checkByID(cs, CheckAccuracy); c.OK || !strings.Contains(c.Detail, "F02") || !strings.Contains(c.Detail, "F03") {
		t.Errorf("a hypothesis and a misconception without an accuracy row must fail: %+v", c)
	}
	none := MachineChecks(in, Parts{})
	if checkByID(none, CheckContrast).OK || checkByID(none, CheckTerms).OK {
		t.Error("checks fail before the parts exist")
	}
}

func TestNewConceptInput_HashesEachSource(t *testing.T) {
	a := conceptInput(t, r01)
	b, err := NewConceptInput(scriptVersion(t, func(m map[string]any) {
		m["script:b2"] = map[string]any{"lines": []Line{{ID: "b2.1", Text: "khác"}}}
	}), []ReferenceStyle{r01})
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{SourceBrief, SourceResearch, SourceStory, SourceBeats, SourceRefs} {
		if a.Hashes[s] != b.Hashes[s] {
			t.Errorf("%s changed though only the script did", s)
		}
	}
	if a.Hashes[SourceScript] == b.Hashes[SourceScript] {
		t.Error("the script source must change")
	}
	edited := r01
	edited.Summary = "khác"
	c := conceptInput(t, edited)
	if c.Hashes[SourceRefs] == a.Hashes[SourceRefs] || c.Hashes[SourceBeats] != a.Hashes[SourceBeats] {
		t.Error("editing a picked description changes only the refs source")
	}
	if a.ResearchSkipped || len(a.Beats) != 2 || a.Beats[1].Number != 2 || a.Beats[1].Dur != 40 {
		t.Errorf("input %+v", a.Beats)
	}
	if _, err := NewConceptInput(scriptVersion(t, func(m map[string]any) { delete(m, "glossary") }), nil); err == nil {
		t.Error("a version missing a part must be an error")
	}
	skipped, _ := NewConceptInput(scriptVersion(t, func(m map[string]any) { m["facts"] = []any{} }), nil)
	if !skipped.ResearchSkipped {
		t.Error("no facts means research was skipped")
	}
}

func TestOverlayFieldsKeepsTheCreatorsFields(t *testing.T) {
	out, err := OverlayFields([]byte(`{"a":1,"b":2}`), []byte(`{"a":9,"b":8}`), []string{"b"})
	if err != nil || !sameJSONT(out, `{"a":1,"b":8}`) {
		t.Errorf("%s %v", out, err)
	}
}

func sameJSONT(a []byte, b string) bool {
	var x, y any
	_ = json.Unmarshal(a, &x)
	_ = json.Unmarshal([]byte(b), &y)
	ax, _ := json.Marshal(x)
	by, _ := json.Marshal(y)
	return string(ax) == string(by)
}

func TestAccuracyNotesAndDocument(t *testing.T) {
	p := decodedParts(t, validParts())
	notes := AccuracyNotes(p)
	facts := []string{}
	for _, n := range notes {
		facts = append(facts, n.Fact)
	}
	if strings.Join(facts, ",") != "F01,F02,F02,F03" {
		t.Errorf("notes %v", facts)
	}
	at := time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)
	review := ReviewState{Findings: []Finding{{ID: "f1", Group: "accuracy", Text: "t", State: FindingAccepted, Why: "ok"}}}
	doc, err := BuildDocument("test-cr112b-p", 2, 3, at, p, []ReferenceStyle{r01}, review)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(doc)
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	if m["document_id"] != "visual-concept-v2" || m["status"] != "locked" || m["schema_version"] != "1.0" {
		t.Errorf("head %v %v", m["document_id"], m["status"])
	}
	rep := m["representation_rules"].(map[string]any)
	if rep["typography"].(map[string]any)["font"] != "Be Vietnam Pro" || rep["characters"] == nil {
		t.Error("typography sits inside representation_rules")
	}
	if _, ok := m["do_not_use"].([]any); !ok {
		t.Error("do_not_use is a plain array in the document")
	}
	refs := m["reference_styles"].([]any)[0].(map[string]any)
	if refs["summary"] != "nền tối ấm" || refs["drawing"].(map[string]any)["shape"] != "tròn mềm" || refs["layout"].(map[string]any)["layering"] != "ba lớp" || refs["description"] != nil {
		t.Errorf("reference copied as summary, drawing and layout: %v", refs)
	}
	comments := m["review"].(map[string]any)["comments"].([]any)
	if len(comments) != 1 || comments[0].(map[string]any)["decision"] != "accepted" {
		t.Errorf("comments %v", comments)
	}
	if _, err := BuildDocument("p", 1, 1, at, Parts{}, nil, ReviewState{}); !errors.Is(err, ErrLockNotReady) {
		t.Error("an incomplete concept cannot be assembled")
	}
}
