package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"visual/internal/domain"
)

// fakeStoryboardStore keeps step 11's data and the locked steps under it in
// memory.
type fakeStoryboardStore struct {
	units     map[string]domain.ReviewUnit
	concepts  []VersionRecord
	plans     []PlanVersionRecord
	sequences []SequenceVersionRecord
	model     string
}

func (s *fakeStoryboardStore) StoryboardModel(context.Context, string) (string, error) {
	return s.model, nil
}
func (s *fakeStoryboardStore) SetStoryboardModel(_ context.Context, _ string, m string) error {
	s.model = m
	return nil
}
func (s *fakeStoryboardStore) ListUnits(_ context.Context, p, kind string) ([]domain.ReviewUnit, error) {
	var out []domain.ReviewUnit
	for _, u := range s.units {
		if u.ProjectID == p && u.Kind == kind {
			out = append(out, u)
		}
	}
	return out, nil
}
func (s *fakeStoryboardStore) SaveUnit(_ context.Context, u domain.ReviewUnit) error {
	s.units[u.Kind+"/"+u.Key] = u
	return nil
}
func (s *fakeStoryboardStore) LatestConceptVersion(context.Context, string) (VersionRecord, error) {
	if len(s.concepts) == 0 {
		return VersionRecord{}, domain.ErrNoVersion
	}
	return s.concepts[len(s.concepts)-1], nil
}
func (s *fakeStoryboardStore) LatestPlanVersion(context.Context, string) (PlanVersionRecord, error) {
	if len(s.plans) == 0 {
		return PlanVersionRecord{}, domain.ErrNoVersion
	}
	return s.plans[len(s.plans)-1], nil
}
func (s *fakeStoryboardStore) SequenceVersions(context.Context, string) ([]SequenceVersionRecord, error) {
	var out []SequenceVersionRecord
	for _, v := range s.sequences {
		v.Document = nil
		out = append(out, v)
	}
	return out, nil
}
func (s *fakeStoryboardStore) SequenceVersion(_ context.Context, _ string, n int) (SequenceVersionRecord, error) {
	for _, v := range s.sequences {
		if v.Version == n {
			return v, nil
		}
	}
	return SequenceVersionRecord{}, domain.ErrNoVersion
}

// fakeKits gives the video a kit, or the reason it has none.
type fakeKits struct{ err error }

func (k fakeKits) KitOfProject(context.Context, string) (ReferenceKitResult, error) {
	if k.err != nil {
		return ReferenceKitResult{}, k.err
	}
	return ReferenceKitResult{RefID: "R12", Kit: domain.ReferenceKit{Outline: 2, Ink: "#101010"}}, nil
}

// fakeRendering is rendering's step 11 API in memory.
type fakeRendering struct {
	assets   []ProjectAsset
	library  []LibraryDrawing
	nextNum  map[string]int
	frames   []FrameRequest
	frameErr error
	locked   []StoryboardVersionRecord
	lockErr  error
	deleted  []string
	updated  []string
	purged   []string
	purgeErr error
	draws    []string
	approves []AssetApproveRequest
}

func newFakeRendering() *fakeRendering { return &fakeRendering{nextNum: map[string]int{}} }

func (r *fakeRendering) find(id string) (int, error) {
	for i, a := range r.assets {
		if a.AssetID == id {
			return i, nil
		}
	}
	return -1, &RenderingError{Status: 404, Message: "không có tài sản " + id}
}

func (r *fakeRendering) ListAssets(context.Context, string) ([]ProjectAsset, error) {
	return append([]ProjectAsset{}, r.assets...), nil
}

func (r *fakeRendering) CreateAsset(_ context.Context, _ string, in AssetInput) (ProjectAsset, error) {
	name := strings.ToUpper(in.Name)
	r.nextNum[name]++
	a := ProjectAsset{StoryboardAsset: domain.StoryboardAsset{AssetID: fmt.Sprintf("AST-%s-%03d", name, r.nextNum[name]), ConceptID: in.ConceptID,
		Name: in.Name, Kind: in.Kind, Tier: in.Tier, Portrayal: in.Portrayal, MustNotImply: in.MustNotImply, PaletteTokens: in.PaletteTokens,
		Expressions: in.Expressions, BaseIllustrationID: in.BaseIllustrationID, Units: in.Units, Status: domain.AssetPlanned}, ToLibrary: in.ToLibrary}
	r.assets = append(r.assets, a)
	return a, nil
}

func (r *fakeRendering) UpdateAsset(_ context.Context, _ string, id string, in AssetInput) (ProjectAsset, error) {
	i, err := r.find(id)
	if err != nil {
		return ProjectAsset{}, err
	}
	a := r.assets[i]
	a.ConceptID, a.Name, a.Kind, a.Tier, a.Portrayal = in.ConceptID, in.Name, in.Kind, in.Tier, in.Portrayal
	a.MustNotImply, a.PaletteTokens, a.Expressions, a.BaseIllustrationID = in.MustNotImply, in.PaletteTokens, in.Expressions, in.BaseIllustrationID
	a.Units, a.ToLibrary, a.LibraryFolderID = in.Units, in.ToLibrary, in.LibraryFolderID
	if a.Status != domain.AssetPlanned {
		a.Status = domain.AssetPlanned
	}
	r.assets[i] = a
	r.updated = append(r.updated, id)
	return a, nil
}

func (r *fakeRendering) DeleteAsset(_ context.Context, _ string, id string) error {
	i, err := r.find(id)
	if err != nil {
		return err
	}
	r.assets = append(r.assets[:i], r.assets[i+1:]...)
	r.deleted = append(r.deleted, id)
	return nil
}

func (r *fakeRendering) draw(id string) (ProjectAsset, error) {
	i, err := r.find(id)
	if err != nil {
		return ProjectAsset{}, err
	}
	r.assets[i].Version++
	r.assets[i].Status = domain.AssetDrawn
	r.assets[i].PreviewPath = fmt.Sprintf("storyboard/assets/%s/v%d.png", id, r.assets[i].Version)
	r.draws = append(r.draws, id)
	return r.assets[i], nil
}

func (r *fakeRendering) DrawAsset(_ context.Context, _ string, id string, _ AssetDrawRequest) (ProjectAsset, error) {
	return r.draw(id)
}
func (r *fakeRendering) RedrawAsset(_ context.Context, _ string, id string, _ AssetDrawRequest) (ProjectAsset, error) {
	return r.draw(id)
}
func (r *fakeRendering) ApproveAsset(_ context.Context, _ string, id string, req AssetApproveRequest) (ProjectAsset, error) {
	i, err := r.find(id)
	if err != nil {
		return ProjectAsset{}, err
	}
	if r.assets[i].Status == domain.AssetPlanned {
		return ProjectAsset{}, &RenderingError{Status: 409, Message: "tài sản chưa vẽ"}
	}
	r.assets[i].Status = domain.AssetApproved
	r.approves = append(r.approves, req)
	return r.assets[i], nil
}
func (r *fakeRendering) UnapproveAsset(_ context.Context, _ string, id string) (ProjectAsset, error) {
	i, err := r.find(id)
	if err != nil {
		return ProjectAsset{}, err
	}
	r.assets[i].Status = domain.AssetDrawn
	return r.assets[i], nil
}
func (r *fakeRendering) ApprovedLibrary(context.Context) ([]LibraryDrawing, error) {
	return r.library, nil
}

func (r *fakeRendering) DrawFrame(_ context.Context, _ string, req FrameRequest) (FrameResult, error) {
	r.frames = append(r.frames, req)
	if r.frameErr != nil {
		return FrameResult{}, r.frameErr
	}
	return FrameResult{FramePath: "storyboard/" + req.ShotID + ".png", FrameHash: "png-" + req.ShotID, SpecHash: req.SpecHash}, nil
}
func (r *fakeRendering) AssetPreview(_ context.Context, _ string, id string) ([]byte, error) {
	i, err := r.find(id)
	if err != nil || r.assets[i].PreviewPath == "" {
		return nil, &RenderingError{Status: 404, Message: "chưa có ảnh"}
	}
	return []byte("PNG-" + id), nil
}
func (r *fakeRendering) FrameImage(context.Context, string, string) ([]byte, error) {
	return []byte("PNG"), nil
}

func (r *fakeRendering) LockStoryboard(_ context.Context, p string, doc json.RawMessage, seq int, note string) (StoryboardVersionRecord, error) {
	if r.lockErr != nil {
		return StoryboardVersionRecord{}, r.lockErr
	}
	rec := StoryboardVersionRecord{ProjectID: p, Version: len(r.locked) + 1, SequenceVersion: seq, Note: note, LockedAt: time.Unix(1_700_000_000, 0).UTC(), Document: doc}
	r.locked = append(r.locked, rec)
	return rec, nil
}
func (r *fakeRendering) LatestStoryboard(context.Context, string) (StoryboardVersionRecord, error) {
	if len(r.locked) == 0 {
		return StoryboardVersionRecord{}, domain.ErrNoVersion
	}
	return r.locked[len(r.locked)-1], nil
}
func (r *fakeRendering) StoryboardVersion(_ context.Context, _ string, n int) (StoryboardVersionRecord, error) {
	if n < 1 || n > len(r.locked) {
		return StoryboardVersionRecord{}, domain.ErrNoVersion
	}
	return r.locked[n-1], nil
}
func (r *fakeRendering) PurgeStoryboard(_ context.Context, p string) error {
	r.purged = append(r.purged, p)
	return r.purgeErr
}

// sbConceptDoc is concept v1 of script v2: an astronaut and a black hole to
// draw, a palette, a beat journey and the rules for on-screen text.
func sbConceptDoc(t *testing.T) []byte {
	t.Helper()
	d := domain.Document{DocumentID: "visual-concept-v1"}
	d.ColorSystem = domain.ColorSystem{Background: "#101010",
		Palette:  []domain.Swatch{{Token: "ink", Hex: "#FFFFFF"}, {Token: "navy", Hex: "#0F1B3D"}},
		Journey:  []domain.JourneyStep{{BeatKeys: []string{"b1", "b2"}, Mood: "tò mò", BackgroundToken: "navy"}},
		Contrast: domain.Contrast{TextToken: "ink"}}
	d.RepresentationRules.Characters.Items = []domain.RepresentedThing{{ConceptID: "astro", Term: "Phi hành gia", Draw: "bộ đồ trắng", MustNotImply: []string{"siêu nhân"}}}
	d.RepresentationRules.Objects = []domain.RepresentedThing{{ConceptID: "hole", Term: "Hố đen", Draw: "đĩa tối"}}
	d.RepresentationRules.Typography = domain.Typography{Labels: domain.Labels{MaxWords: 4, MinPx: 28}}
	d.VisualLanguage.CompositionRules = []string{"quy tắc một phần ba"}
	raw, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// sbSequenceDoc is sequence v1 over plan v1: VP-001, VP-002 and VP-003.
func sbSequenceDoc(t *testing.T) []byte {
	t.Helper()
	d := domain.SequenceDocument{DocumentID: "visual-sequence-v1", Version: 1,
		SourceVersions: domain.SequenceSourceVersions{Script: 2, VisualConcept: 1, VisualPlan: 1},
		Sequence: []domain.DocumentSeqItem{
			{SequenceIndex: 1, UnitID: "VP-001", BeatID: "b1", PurposeInFlow: "mở", MotifRefs: []string{}},
			{SequenceIndex: 2, UnitID: "VP-002", BeatID: "b1", PurposeInFlow: "giữa", MotifRefs: []string{}},
			{SequenceIndex: 3, UnitID: "VP-003", BeatID: "b2", PurposeInFlow: "đóng", MotifRefs: []string{}},
		}}
	raw, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// sbFixture is step 11 over locked steps 7 to 10, with a fake rendering and a
// fake AI.
type sbFixture struct {
	t         *testing.T
	ctx       context.Context
	desk      *StoryboardDesk
	store     *fakeStoryboardStore
	rendering *fakeRendering
	llm       *fakeLLM
}

type sbModels struct{ fakeModels }

func (sbModels) StepDefaultModel(_ context.Context, step string) (string, error) {
	if step != StoryboardStep {
		return "", ErrLLMFailed
	}
	return "ollama", nil
}

func newSbFixtureWith(t *testing.T, kits ProjectKitPort) *sbFixture {
	t.Helper()
	store := &fakeStoryboardStore{units: map[string]domain.ReviewUnit{},
		concepts:  []VersionRecord{{Version: 1, ScriptVersion: 2, Document: sbConceptDoc(t)}},
		plans:     []PlanVersionRecord{{Version: 1, ScriptVersion: 2, ConceptVersion: 1, Document: seqPlanDoc(t, "mở")}},
		sequences: []SequenceVersionRecord{{Version: 1, PlanVersion: 1, ConceptVersion: 1, ScriptVersion: 2, Document: sbSequenceDoc(t)}}}
	rend := newFakeRendering()
	llm := &fakeLLM{replies: map[string][]string{}}
	desk := NewStoryboardDesk(&fakeAuthoring{v: planScript(t, 2, "Không phải C")}, store, rend, kits).
		WithAI(NewExternalAIProvider(llm, llm), seedPrompts{}, sbModels{}, 1000)
	desk.now = func() time.Time { return time.Unix(1_700_000_000, 0) }
	return &sbFixture{t: t, ctx: context.Background(), desk: desk, store: store, rendering: rend, llm: llm}
}

func newSbFixture(t *testing.T) *sbFixture { return newSbFixtureWith(t, fakeKits{}) }

func (f *sbFixture) must(st *StoryboardScreen, err error) *StoryboardScreen {
	f.t.Helper()
	if err != nil {
		f.t.Fatal(err)
	}
	return st
}

func (f *sbFixture) queue(role string, replies ...string) { f.llm.replies[role] = replies }

// approvedAssets puts an approved astronaut figure and an approved space
// backdrop at rendering.
func (f *sbFixture) approvedAssets() (figure, backdrop string) {
	f.t.Helper()
	for _, in := range []AssetInput{
		{ConceptID: "astro", Name: "Astro", Kind: domain.AssetKindFigure, Tier: domain.AssetTierHero, Portrayal: "phi hành gia", PaletteTokens: []string{"ink"}, ToLibrary: true},
		{Name: "Space", Kind: domain.AssetKindBackdrop, Tier: domain.AssetTierBackground, Portrayal: "vũ trụ", PaletteTokens: []string{"navy"}, ToLibrary: true},
	} {
		a, err := f.rendering.CreateAsset(f.ctx, project, in)
		if err != nil {
			f.t.Fatal(err)
		}
		if _, err := f.rendering.DrawAsset(f.ctx, project, a.AssetID, AssetDrawRequest{}); err != nil {
			f.t.Fatal(err)
		}
		if _, err := f.rendering.ApproveAsset(f.ctx, project, a.AssetID, AssetApproveRequest{}); err != nil {
			f.t.Fatal(err)
		}
	}
	return "AST-ASTRO-001", "AST-SPACE-001"
}

// shotJSON is one shot of an AI reply: the astronaut at the lower left over the
// space backdrop.
func shotJSON(refs []string, text []string, purpose string) string {
	raw := func(v any) string { b, _ := json.Marshal(v); return string(b) }
	return fmt.Sprintf(`{"script_refs":%s,"shot_purpose":%q,"composition":{"subject":"AST-ASTRO-001","subject_position":"lower_left","main_object":"","main_object_position":"","camera_view":"wide"},"action_summary":"cảnh","on_screen_text":%s,"asset_refs":["AST-ASTRO-001","AST-SPACE-001"],"notes":[]}`,
		raw(refs), purpose, raw(text))
}

func shotsReply(shots ...string) string { return `{"shots":[` + strings.Join(shots, ",") + `]}` }

// writeUnitShots has the fake AI write VP-001 with one shot.
func (f *sbFixture) writeUnit1() *StoryboardScreen {
	f.t.Helper()
	f.queue(domain.PromptRoleStoryboardUnit, shotsReply(shotJSON([]string{"b1.1"}, []string{"Góc nhìn A"}, "mở")))
	return f.must(f.desk.GenerateUnit(f.ctx, project, "VP-001", ""))
}

func unitOf(st *StoryboardScreen, id string) *StoryboardUnitView {
	for _, b := range st.Beats {
		for i := range b.Units {
			if b.Units[i].UnitID == id {
				return &b.Units[i]
			}
		}
	}
	return nil
}

func jfields(t *testing.T, m map[string]any) map[string]json.RawMessage {
	t.Helper()
	return fields(t, m)
}
