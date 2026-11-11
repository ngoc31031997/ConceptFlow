package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
	"testing"

	"visual/internal/domain"
)

const project = "test-cr112b-app"

type fakeAuthoring struct {
	v   domain.ScriptVersion
	err error
}

func (f *fakeAuthoring) LatestScriptVersion(context.Context, string) (domain.ScriptVersion, error) {
	return f.v, f.err
}

type fakeStore struct {
	units    map[string]domain.ReviewUnit
	model    string
	library  []domain.ReferenceStyle
	picks    []string
	versions []VersionRecord
}

func newStore() *fakeStore {
	return &fakeStore{units: map[string]domain.ReviewUnit{},
		library: []domain.ReferenceStyle{{ID: "R01", Name: "Sáng", Summary: "nền tối", Drawing: domain.RefDrawing{Shape: "tròn", Fill: "phẳng"}, Layout: domain.RefLayout{Layering: "ba lớp"}}, {ID: "R02", Name: "Sơ đồ", Summary: "nét mảnh"}}}
}

func (s *fakeStore) ListUnits(_ context.Context, p, kind string) ([]domain.ReviewUnit, error) {
	var out []domain.ReviewUnit
	for _, u := range s.units {
		if u.ProjectID == p && u.Kind == kind {
			out = append(out, u)
		}
	}
	return out, nil
}
func (s *fakeStore) SaveUnit(_ context.Context, u domain.ReviewUnit) error {
	s.units[u.Kind+"/"+u.Key] = u
	return nil
}
func (s *fakeStore) ConceptModel(context.Context, string) (string, error) { return s.model, nil }
func (s *fakeStore) SetConceptModel(_ context.Context, _ string, m string) error {
	s.model = m
	return nil
}
func (s *fakeStore) PickedReferences(context.Context, string) ([]domain.ReferenceStyle, error) {
	var out []domain.ReferenceStyle
	for _, r := range s.library {
		for _, id := range s.picks {
			if r.ID == id {
				out = append(out, r)
			}
		}
	}
	return out, nil
}
func (s *fakeStore) ReferenceLibrary(context.Context) ([]domain.ReferenceStyle, error) {
	return s.library, nil
}
func (s *fakeStore) SetPicks(_ context.Context, _ string, ids []string) error {
	for _, id := range ids {
		found := false
		for _, r := range s.library {
			found = found || r.ID == id
		}
		if !found {
			return domain.ErrUnknownReference
		}
	}
	s.picks = ids
	return nil
}
func (s *fakeStore) Versions(context.Context, string) ([]VersionRecord, error) {
	var out []VersionRecord
	for _, v := range s.versions {
		v.Document = nil
		out = append(out, v)
	}
	return out, nil
}
func (s *fakeStore) Version(_ context.Context, _ string, n int) (VersionRecord, error) {
	for _, v := range s.versions {
		if v.Version == n {
			return v, nil
		}
	}
	return VersionRecord{}, domain.ErrNoVersion
}
func (s *fakeStore) InsertVersion(_ context.Context, v VersionRecord) error {
	for _, have := range s.versions {
		if have.Version == v.Version {
			return domain.ErrVersionExists
		}
	}
	s.versions = append(s.versions, v)
	return nil
}

// fakeLLM answers each role with the next reply queued for it, and records
// every request it was sent.
type fakeLLM struct {
	replies map[string][]string
	calls   []ChatRequest
}

func (f *fakeLLM) Chat(_ context.Context, req ChatRequest) (ChatResult, error) {
	f.calls = append(f.calls, req)
	q := f.replies[req.Role]
	if len(q) == 0 {
		return ChatResult{}, ErrLLMFailed
	}
	f.replies[req.Role] = q[1:]
	return ChatResult{Content: q[0], JSON: json.RawMessage(q[0])}, nil
}

func (f *fakeLLM) ChatExternal(_ context.Context, req ChatRequest, reply string) (ChatResult, error) {
	f.calls = append(f.calls, req)
	return ChatResult{Content: reply, JSON: json.RawMessage(reply)}, nil
}

type fakePrompts struct{}

func (fakePrompts) ActiveTemplate(_ context.Context, role string) (string, error) {
	return "ROLE " + role + "\n{{beats}}\n{{refs}}\n{{current}}\n{{creator_fields}}\n{{note}}\n{{parts}}\n{{previous_findings}}\n{{fonts}}", nil
}

type fakeModels struct{}

func (fakeModels) Models(context.Context) ([]ModelOption, error) {
	return []ModelOption{{ID: "ollama", Label: "Ollama", Local: true}, {ID: "deepseek-chat", Label: "DeepSeek"}}, nil
}
func (fakeModels) StepDefaultModel(_ context.Context, step string) (string, error) {
	if step != ConceptStep {
		return "", ErrLLMFailed
	}
	return "ollama", nil
}

type fakeFonts struct{ err error }

func (f *fakeFonts) Fonts(context.Context) ([]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	return []string{"Be Vietnam Pro"}, nil
}

func frozen(t *testing.T, key string, v any) domain.ScriptVersionPart {
	t.Helper()
	raw, _ := json.Marshal(v)
	sum := sha256.Sum256(raw)
	return domain.ScriptVersionPart{Key: key, SHA: hex.EncodeToString(sum[:]), Content: raw}
}

func lockedScript(t *testing.T, version int, b2line string) domain.ScriptVersion {
	t.Helper()
	m := map[string]any{
		"brief": domain.Brief{Audience: "phổ thông", Tone: "tò mò"},
		"facts": []map[string]any{
			{"ID": "F01", "type": "fact", "claim_vi": "A"},
			{"ID": "F02", "type": "hypothesis", "claim_vi": "B"},
			{"ID": "F03", "type": "misconception", "claim_vi": "C"},
		},
		"story":      map[string]domain.StoryField{"question": {Text: "q"}},
		"order":      []string{"b1", "b2"},
		"beat:b1":    map[string]any{"fn": "Hook", "title": "Mở", "dur": 20},
		"beat:b2":    map[string]any{"fn": "Payoff", "title": "Đóng", "dur": 40},
		"script:b1":  map[string]any{"lines": []domain.Line{{ID: "b1.1", Text: "A", Facts: []string{"F01"}}, {ID: "b1.2", Text: "B", Facts: []string{"F02"}}}},
		"script:b2":  map[string]any{"lines": []domain.Line{{ID: "b2.1", Text: b2line, Facts: []string{"F03"}}}},
		"handoff:b1": domain.Handoff{},
		"handoff:b2": domain.Handoff{},
		"glossary":   map[string]any{"terms": []domain.GlossaryTerm{{Term: "Enzyme"}, {Term: "cửa quả"}}},
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var parts []domain.ScriptVersionPart
	for _, k := range keys {
		parts = append(parts, frozen(t, k, m[k]))
	}
	return domain.ScriptVersion{ProjectID: project, Version: version, Parts: parts}
}

// goodReplies are eight parts that pass every check of lockedScript.
var goodReplies = map[string]string{
	domain.UnitCreativeDirection:   `{"style_name":"Phẳng sáng","description":"d","abstraction_level":"hybrid","tone":["ấm"],"principles":["một ý"],"why":"w","feeling":"f","refs_used":["R01"],"abstraction_map":[],"examples":[]}`,
	domain.UnitColorSystem:         `{"background":"#101010","palette":[{"token":"ink","hex":"#ffffff","role":"chữ","name":"trắng"},{"token":"accent","hex":"#e1ad01","role":"nhấn","name":"vàng"}],"journey":[{"beat_keys":["b1","b2"],"mood":"ấm","accent_token":"accent","background_token":""}],"contrast":{"text_min":4.5,"text_token":"ink","subject_vs_bg":"s","accent_per_frame":1},"exceptions":[],"examples":[]}`,
	domain.UnitVisualLanguage:      `{"shape_rules":["tròn"],"line_rules":["không viền"],"lighting_rules":["sau"],"depth_rules":["ba lớp"],"surface_rules":["phẳng"],"layout_rules":["ba lớp"],"composition_rules":["1/3"],"examples":[]}`,
	domain.UnitRepresentationRules: `{"characters":{"approach":"đơn giản","allowed_variations":[],"items":[{"concept_id":"wasp","term":"ong","draw":"thân đen","must_not_imply":[],"facts":["F01"]}]},"objects":[{"concept_id":"gate","term":"cửa quả","draw":"lỗ","must_not_imply":[],"facts":[]}],"invisible":[],"scientific_concepts":[{"concept_id":"enz","term":"enzyme","representation":"quầng","must_not_imply":[],"facts":["F02"]}],"examples":[]}`,
	domain.UnitTypography:          `{"font_direction":"sans","label_style":"ngắn","font":"Be Vietnam Pro","labels":{"max_words":4,"when":"sau","min_px":36},"captions":"c","terms":[],"formulas":"không dùng"}`,
	domain.UnitMotionLanguage:      `{"principles":["mềm"],"preferred_transitions":["zoom"],"avoid":["rung"],"pace":[{"beat_keys":["b1","b2"],"pace":"medium"}],"character_motion":"râu","examples":[]}`,
	domain.UnitDoNotUse:            `{"items":[{"text":"vẽ B như thật","why":"gt","facts":["F02","F03"],"source":"accuracy"}]}`,
	domain.UnitFeasibility:         `{"scope":{"beats":0,"est_shots":8,"settings":3,"characters":0},"checks":[{"item":"glow","assessment":"chậm","verdict":"risk","mitigation":"blur"}]}`,
}

type rig struct {
	desk  *ConceptDesk
	store *fakeStore
	llm   *fakeLLM
	auth  *fakeAuthoring
	fonts *fakeFonts
}

func newRig(t *testing.T) *rig {
	t.Helper()
	r := &rig{store: newStore(), llm: &fakeLLM{replies: map[string][]string{}}, auth: &fakeAuthoring{v: lockedScript(t, 2, "C")}, fonts: &fakeFonts{}}
	r.store.picks = []string{"R01"}
	r.desk = NewConceptDesk(r.auth, r.store, r.store, NewExternalAIProvider(r.llm, r.llm), fakePrompts{}, fakeModels{}, r.fonts, 1000)
	return r
}

// queueAll queues one good reply per part.
func (r *rig) queueAll() {
	for k, v := range goodReplies {
		role := domain.PromptRoleOf(k)
		r.llm.replies[role] = append(r.llm.replies[role], v)
	}
}

func unitView(st *ConceptState, key string) UnitView {
	for _, u := range st.Units {
		if u.Key == key {
			return u
		}
	}
	return UnitView{}
}

func mustState(t *testing.T) func(*ConceptState, error) *ConceptState {
	return func(st *ConceptState, err error) *ConceptState {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return st
	}
}

func hasLine(lines []string, part string) bool {
	for _, l := range lines {
		if strings.Contains(l, part) {
			return true
		}
	}
	return false
}
