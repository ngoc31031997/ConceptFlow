package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"
)

const (
	astAstronaut = "AST-ASTRONAUT-001"
	astBlackHole = "AST-BLACKHOLE-001"
	astSpace     = "AST-SPACE-001"
)

func sbConcept(t *testing.T) []byte {
	t.Helper()
	return seqConcept(t, func(d *Document) {
		d.ColorSystem = ColorSystem{
			Background: "#101010",
			Palette:    []Swatch{{Token: "ink", Hex: "#FFFFFF"}, {Token: "navy", Hex: "#0F1B3D"}, {Token: "gold", Hex: "#E1AD01"}},
			Journey:    []JourneyStep{{BeatKeys: []string{"b1"}, Mood: "tò mò", BackgroundToken: "navy"}},
			Contrast:   Contrast{TextToken: "ink"},
		}
		d.RepresentationRules.Characters.Items = []RepresentedThing{{ConceptID: "astronaut", Term: "phi hành gia"}}
		d.RepresentationRules.Typography = Typography{Labels: Labels{MaxWords: 4, MinPx: 36}}
		d.VisualLanguage.CompositionRules = []string{"quy tắc một phần ba"}
	})
}

func sbSequence(t *testing.T, edit func(*SequenceDocument)) []byte {
	t.Helper()
	d := SequenceDocument{DocumentID: "visual-sequence-v1", SourceVersions: SequenceSourceVersions{Script: 2, VisualConcept: 1, VisualPlan: 1}, Sequence: []DocumentSeqItem{
		{SequenceIndex: 1, UnitID: "VP-001", BeatID: "b1", PurposeInFlow: "mở"},
		{SequenceIndex: 2, UnitID: "VP-002", BeatID: "b2", PurposeInFlow: "so sánh"},
		{SequenceIndex: 3, UnitID: "VP-003", BeatID: "b2", PurposeInFlow: "chốt"},
	}}
	if edit != nil {
		edit(&d)
	}
	raw, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func sbInput(t *testing.T, seq func(*SequenceDocument)) StoryboardInput {
	t.Helper()
	in, err := NewStoryboardInput(scriptVersion(t, nil), 1, sbConcept(t), 1, seqPlan(t, nil), 1, sbSequence(t, seq))
	if err != nil {
		t.Fatal(err)
	}
	return in
}

func sbAssets() []StoryboardAsset {
	return []StoryboardAsset{
		{AssetID: astAstronaut, ConceptID: "astronaut", Name: "Astronaut", Kind: AssetKindFigure, Tier: AssetTierHero, Portrayal: "nhỏ bé", Version: 1, PreviewPath: "storyboard/assets/a/v1.png", Status: AssetApproved},
		{AssetID: astBlackHole, ConceptID: "black_hole", Name: "BlackHole", Kind: AssetKindFigure, Tier: AssetTierSupport, Portrayal: "tròn đen", Version: 1, PreviewPath: "storyboard/assets/b/v1.png", Status: AssetApproved},
		{AssetID: astSpace, Name: "Space", Kind: AssetKindBackdrop, Tier: AssetTierBackground, Portrayal: "nền sao", Version: 1, PreviewPath: "storyboard/assets/s/v1.png", Status: AssetApproved},
	}
}

func sbShot(id string, refs ...string) Shot {
	return Shot{
		ShotID: id, ScriptRefs: refs, ShotPurpose: "so sánh", ActionSummary: "phi hành gia nhìn hố đen",
		Composition:  ShotComposition{Subject: astAstronaut, SubjectPosition: "lower_left", MainObject: astBlackHole, MainObjectPosition: "upper_right", CameraView: CameraWide},
		OnScreenText: []string{"Góc nhìn A"}, AssetRefs: []string{astAstronaut, astBlackHole, astSpace}, Notes: []string{}, Status: UnitApproved,
	}
}

// sbState is a storyboard that can lock: one approved shot per plan unit with a
// frame drawn from its current spec, and one finished review.
func sbState(t *testing.T, in StoryboardInput) StoryboardState {
	t.Helper()
	st := StoryboardState{
		ProjectID: "p", Assets: sbAssets(), Kit: json.RawMessage(`{"stroke":2}`),
		Units: map[string]UnitShots{
			"VP-001": {Shots: []Shot{sbShot("SH-001", "b1.1", "b1.2")}, NextSeq: 4},
			"VP-002": {Shots: []Shot{sbShot("SH-002", "b2.1")}, NextSeq: 4},
			"VP-003": {Shots: []Shot{sbShot("SH-003", "b2.1")}, NextSeq: 4},
		},
		Frames: map[string]DrawnFrame{},
		Review: ReviewState{Rounds: 1, Findings: []Finding{}, Checks: map[string]CheckDecision{}},
	}
	for unitID, u := range st.Units {
		for _, sh := range u.Shots {
			h, err := st.SpecHashOf(in, unitID, sh)
			if err != nil {
				t.Fatal(err)
			}
			st.Frames[sh.ShotID] = DrawnFrame{FramePath: "storyboard/" + sh.ShotID + ".png", SpecHash: h}
		}
	}
	return st
}

func TestNormalizeComposition(t *testing.T) {
	two := ShotComposition{Subject: "a", SubjectPosition: PositionSplitScreen, MainObject: "b", MainObjectPosition: PositionSplitScreen, CameraView: CameraComparison}
	got, err := NormalizeComposition(two)
	if err != nil || got.SubjectPosition != PositionSplitLeft || got.MainObjectPosition != PositionSplitRight {
		t.Fatalf("two objects take the halves: %+v %v", got, err)
	}
	onlyOneSays := ShotComposition{Subject: "a", SubjectPosition: "center", MainObject: "b", MainObjectPosition: PositionSplitScreen}
	if got, err = NormalizeComposition(onlyOneSays); err != nil || got.SubjectPosition != PositionSplitLeft {
		t.Errorf("split_screen on either object splits the frame: %+v %v", got, err)
	}
	for name, c := range map[string]ShotComposition{
		"one object":        {Subject: "a", SubjectPosition: PositionSplitScreen},
		"blank main object": {Subject: "a", SubjectPosition: PositionSplitScreen, MainObject: " ", MainObjectPosition: PositionSplitScreen},
		"no subject":        {MainObject: "b", MainObjectPosition: PositionSplitScreen},
	} {
		if _, err := NormalizeComposition(c); err == nil {
			t.Errorf("%s: split_screen needs exactly two objects", name)
		}
	}
	plain := ShotComposition{Subject: "a", SubjectPosition: "center", CameraView: CameraClose}
	if got, err = NormalizeComposition(plain); err != nil || got != plain {
		t.Errorf("another position is left alone: %+v %v", got, err)
	}
	if contains(ShotPositions, PositionSplitScreen) {
		t.Error("split_screen is not stored, so it is not in the vocabulary")
	}
	for _, p := range []string{PositionSplitLeft, PositionSplitRight, "full", "center"} {
		if !contains(ShotPositions, p) {
			t.Errorf("%s belongs to the vocabulary", p)
		}
	}
	if strings.Join(ShotCameraViews, ",") != "wide,medium,close,comparison" {
		t.Errorf("camera views: %v", ShotCameraViews)
	}
}

func TestShotIDsAreNeverReused(t *testing.T) {
	if ShotID(7) != "SH-007" || ShotSeq("SH-007") != 7 || ShotSeq("SH-7") != 0 || ShotSeq("VP-001") != 0 || ShotSeq("SH-1000") != 1000 {
		t.Error("shot ids are SH- and at least three digits")
	}
	units := map[string]UnitShots{
		"VP-001": {Shots: []Shot{{ShotID: "SH-002"}}, NextSeq: 5},
		"VP-002": {Shots: []Shot{{ShotID: "SH-009"}}, NextSeq: 2},
	}
	if got := NextShotSeq(units); got != 10 {
		t.Errorf("past the highest id and counter, got %d", got)
	}
	if got := NextShotSeq(nil); got != 1 {
		t.Errorf("a project with no shot starts at 1, got %d", got)
	}
	if got := NextShotSeq(map[string]UnitShots{"VP-001": {NextSeq: 12}}); got != 12 {
		t.Errorf("a counter ahead of the ids (deleted shots) holds, got %d", got)
	}
	if ids := SortedShotIDs(map[string]UnitShots{"a": {Shots: []Shot{{ShotID: "SH-010"}}}, "b": {Shots: []Shot{{ShotID: "SH-002"}}}}); strings.Join(ids, ",") != "SH-002,SH-010" {
		t.Errorf("number order: %v", ids)
	}
	if ShotCreatorField("SH-001", "notes") != "SH-001.notes" {
		t.Error("creator field path is <shot>.<field>")
	}
}

func TestUnitShotsCodecAndApproval(t *testing.T) {
	empty, err := DecodeUnitShots(nil)
	if err != nil || len(empty.Shots) != 0 || empty.NextSeq != 1 {
		t.Fatalf("empty content is a unit with no shot: %+v %v", empty, err)
	}
	s, err := DecodeUnitShots([]byte(`{"shots":[{"shot_id":"SH-004","script_refs":null}],"next_seq":2}`))
	if err != nil || s.NextSeq != 5 || s.Shots[0].Status != UnitPending || s.Shots[0].ScriptRefs == nil || s.Shots[0].AssetRefs == nil {
		t.Errorf("null lists become empty, status pending, counter passes the ids: %+v %v", s, err)
	}
	raw, err := s.Encode()
	if err != nil || strings.Contains(string(raw), "null") {
		t.Errorf("encoded content has no null: %s %v", raw, err)
	}
	if _, err := DecodeUnitShots([]byte(`{"shots":[],"next_seq":1,"extra":1}`)); !errors.Is(err, ErrInvalidEdit) {
		t.Errorf("unknown key in stored content: %v", err)
	}
	if _, err := DecodeShot([]byte(`{"shot_id":"SH-001","unit_id":"VP-001"}`)); !errors.Is(err, ErrInvalidEdit) {
		t.Errorf("the unit is not a shot field: %v", err)
	}
	if _, err := DecodeShot([]byte(`{"shot_purpose":3}`)); !errors.Is(err, ErrInvalidEdit) {
		t.Errorf("wrong type: %v", err)
	}
	approved, pending := sbShot("SH-001"), sbShot("SH-002")
	pending.Status = UnitPending
	cases := []struct {
		name  string
		units []UnitShots
		unit  string
		beat  string
	}{
		{"no shot", []UnitShots{{}}, UnitPending, UnitPending},
		{"all approved", []UnitShots{{Shots: []Shot{approved}}, {Shots: []Shot{approved, approved}}}, UnitApproved, UnitApproved},
		{"one pending", []UnitShots{{Shots: []Shot{approved}}, {Shots: []Shot{approved, pending}}}, UnitApproved, UnitPending},
		{"empty beat", nil, UnitPending, UnitPending},
	}
	for _, c := range cases {
		if len(c.units) > 0 {
			if got := UnitApproval(c.units[0]); got != c.unit {
				t.Errorf("%s: first unit %s, want %s", c.name, got, c.unit)
			}
		}
		if got := BeatApproval(c.units); got != c.beat {
			t.Errorf("%s: beat %s, want %s", c.name, got, c.beat)
		}
	}
	if UnitApproval(UnitShots{Shots: []Shot{approved, pending}}) != UnitPending {
		t.Error("a unit is approved only when every shot is")
	}
}

func TestAssetListCodec(t *testing.T) {
	l, err := DecodeAssetList([]byte(`{}`))
	if err != nil || l.NextSeq != 1 || l.AllApproved() {
		t.Fatalf("empty list: %+v %v", l, err)
	}
	l.Assets = []AssetEntry{{AssetID: astAstronaut, Status: UnitApproved}, {AssetID: astSpace, Status: UnitPending}}
	if l.AllApproved() || l.Index(astSpace) != 1 || l.Index("AST-X-001") != -1 {
		t.Error("a list is approved only when every asset is")
	}
	l.Assets[1].Status = UnitApproved
	raw, err := l.Encode()
	if err != nil {
		t.Fatal(err)
	}
	back, err := DecodeAssetList(raw)
	if err != nil || !back.AllApproved() {
		t.Errorf("round trip: %+v %v", back, err)
	}
	if _, err := DecodeAssetList([]byte(`{"assets":[],"x":1}`)); !errors.Is(err, ErrInvalidEdit) {
		t.Errorf("unknown key: %v", err)
	}
	for id, ok := range map[string]bool{astAstronaut: true, "AST-A-B-012": true, "AST-astro-001": false, "AST-ASTRO": false, "SH-001": false} {
		if IsAssetID(id) != ok {
			t.Errorf("IsAssetID(%q) = %v", id, !ok)
		}
	}
}

func TestStoryboardInput_OutdatedWhenAStepUnderTheSequenceMovedOn(t *testing.T) {
	if in := sbInput(t, nil); len(in.Outdated) != 0 || in.SourceVersions() != (StoryboardSourceVersions{Script: 2, VisualConcept: 1, VisualPlan: 1, VisualSequence: 1}) {
		t.Fatalf("a sequence written from the latest steps is current: %v %+v", in.Outdated, in.SourceVersions())
	}
	cases := map[string]func(*SequenceDocument){
		"script":  func(d *SequenceDocument) { d.SourceVersions.Script = 1 },
		"concept": func(d *SequenceDocument) { d.SourceVersions.VisualConcept = 0 },
		"plan":    func(d *SequenceDocument) { d.SourceVersions.VisualPlan = 0 },
	}
	for name, edit := range cases {
		if in := sbInput(t, edit); len(in.Outdated) != 1 {
			t.Errorf("%s: want one reason, got %v", name, in.Outdated)
		}
	}
	if _, err := NewStoryboardInput(scriptVersion(t, nil), 1, sbConcept(t), 1, seqPlan(t, nil), 1, []byte(`{`)); err == nil {
		t.Error("an unreadable sequence document is an error")
	}
}

func TestStoryboardInput_UnitCarriesWhatTheWriterReads(t *testing.T) {
	in := sbInput(t, nil)
	if strings.Join(in.UnitIDs(), ",") != "VP-001,VP-002,VP-003" {
		t.Fatalf("plan order: %v", in.UnitIDs())
	}
	u, ok := in.Unit("VP-001")
	if !ok || u.Item.PurposeInFlow != "mở" || len(u.Lines) != 2 || u.Lines[1].ID != "b1.2" || u.Beat.Fn != "Hook" {
		t.Fatalf("plan unit, sequence item, lines and beat: %+v", u)
	}
	if u.Journey == nil || u.Journey.Mood != "tò mò" || u.BackgroundColor != "#0F1B3D" || u.TextColor != "#FFFFFF" {
		t.Errorf("the journey of the beat sets the background: %+v", u)
	}
	if u.Typography.Labels.MaxWords != 4 || u.Typography.Labels.MinPx != 36 || len(u.CompositionRules) != 1 {
		t.Errorf("typography and composition rules: %+v", u)
	}
	if v, _ := in.Unit("VP-002"); v.Journey != nil || v.BackgroundColor != "#101010" {
		t.Errorf("a beat with no journey step takes the system background: %+v", v)
	}
	if _, ok := in.Unit("VP-009"); ok {
		t.Error("a unit the plan lacks has no input")
	}
	noItem := sbInput(t, func(d *SequenceDocument) { d.Sequence = d.Sequence[:2] })
	if _, ok := noItem.Unit("VP-003"); ok {
		t.Error("a unit the sequence lacks has no input")
	}
}

func TestStoryboardInput_UnitStaleOnlyOnWhatChanged(t *testing.T) {
	assets := map[string]StoryboardAsset{}
	for _, a := range sbAssets() {
		assets[a.AssetID] = a
	}
	shots := []Shot{sbShot("SH-001", "b1.1", "b1.2")}
	in := sbInput(t, nil)
	stored := in.UnitHashes("VP-002", shots, assets)
	if len(stored) != 4 {
		t.Fatalf("one hash per source: %v", stored)
	}
	if got := in.UnitStaleSources("VP-002", stored, shots, assets); len(got) != 0 {
		t.Fatalf("nothing changed: %v", got)
	}
	cases := []struct {
		name string
		in   StoryboardInput
		edit func(map[string]StoryboardAsset)
		want string
	}{
		{"sequence item", sbInput(t, func(d *SequenceDocument) { d.Sequence[1].PurposeInFlow = "khác" }), nil, StoryboardSourceSeq},
		{"plan unit", func() StoryboardInput {
			i, _ := NewStoryboardInput(scriptVersion(t, nil), 1, sbConcept(t), 1, seqPlan(t, func(p *PlanDocument) { p.Units[1].CommunicationGoal = "khác" }), 1, sbSequence(t, nil))
			return i
		}(), nil, StoryboardSourcePlan},
		{"script line", func() StoryboardInput {
			i, _ := NewStoryboardInput(scriptVersion(t, func(m map[string]any) {
				m["script:b2"] = map[string]any{"lines": []Line{{ID: "b2.1", Text: "Câu khác"}}}
			}), 1, sbConcept(t), 1, seqPlan(t, nil), 1, sbSequence(t, nil))
			return i
		}(), nil, StoryboardSourceScript},
		{"asset version", in, func(m map[string]StoryboardAsset) { a := m[astAstronaut]; a.Version = 2; m[astAstronaut] = a }, StoryboardSourceAssets},
	}
	for _, c := range cases {
		a := map[string]StoryboardAsset{}
		for k, v := range assets {
			a[k] = v
		}
		if c.edit != nil {
			c.edit(a)
		}
		if got := c.in.UnitStaleSources("VP-002", stored, shots, a); strings.Join(got, ",") != c.want {
			t.Errorf("%s: want only %s stale, got %v", c.name, c.want, got)
		}
	}
	other := in.UnitHashes("VP-001", shots, assets)
	if got := in.UnitStaleSources("VP-001", other, shots, assets); len(got) != 0 {
		t.Errorf("a unit is not stale for a sibling's change: %v", got)
	}
}

func TestBuildFrameSpecAndStale(t *testing.T) {
	in := sbInput(t, nil)
	ui, _ := in.Unit("VP-001")
	assets := map[string]StoryboardAsset{}
	for _, a := range sbAssets() {
		assets[a.AssetID] = a
	}
	sh := sbShot("SH-001", "b1.1")
	spec, err := BuildFrameSpec("p", sh, ui, assets, json.RawMessage(`{"k":1}`))
	if err != nil {
		t.Fatal(err)
	}
	if spec.Size != (FrameSize{W: 1280, H: 720}) || spec.Background.AssetID != astSpace || spec.Background.Color != "#0F1B3D" || spec.TextColor != "#FFFFFF" || spec.CameraView != CameraWide {
		t.Errorf("size, backdrop, colours, camera: %+v", spec)
	}
	if len(spec.Placements) != 2 || spec.Placements[0] != (FramePlacement{AssetID: astAstronaut, Role: "subject", Position: "lower_left", Version: 1}) || spec.Placements[1].Role != "main" || spec.Placements[1].Position != "upper_right" {
		t.Errorf("subject and main object are placed: %+v", spec.Placements)
	}
	if len(spec.OnScreenText) != 1 || spec.OnScreenText[0] != (FrameText{Text: "Góc nhìn A", MinPx: 36}) {
		t.Errorf("text at the concept's size: %+v", spec.OnScreenText)
	}
	raw, _ := json.Marshal(spec)
	if strings.Contains(string(raw), "version") {
		t.Errorf("the asset version is not sent to rendering: %s", raw)
	}

	free := sh
	free.Composition.Subject = "con ong"
	if s, err := BuildFrameSpec("p", free, ui, assets, nil); err != nil || len(s.Placements) != 1 {
		t.Errorf("a free-name subject is not drawn: %+v %v", s, err)
	}
	missing := sh
	missing.AssetRefs = []string{"AST-NOPE-001"}
	if _, err := BuildFrameSpec("p", missing, ui, assets, nil); err == nil {
		t.Error("an unknown asset cannot be placed")
	}

	base := SpecHash(spec)
	twin, _ := BuildFrameSpec("p", func() Shot { s := sh; s.ShotID = "SH-009"; return s }(), ui, assets, json.RawMessage(`{"k":1}`))
	if SpecHash(twin) != base {
		t.Error("two shots with the same look share a spec hash")
	}
	edits := map[string]func(*Shot, map[string]StoryboardAsset){
		"text":          func(s *Shot, _ map[string]StoryboardAsset) { s.OnScreenText = []string{"Khác"} },
		"position":      func(s *Shot, _ map[string]StoryboardAsset) { s.Composition.SubjectPosition = "center" },
		"camera":        func(s *Shot, _ map[string]StoryboardAsset) { s.Composition.CameraView = CameraClose },
		"asset version": func(_ *Shot, a map[string]StoryboardAsset) { x := a[astBlackHole]; x.Version = 2; a[astBlackHole] = x },
		"backdrop":      func(_ *Shot, a map[string]StoryboardAsset) { x := a[astSpace]; x.Version = 3; a[astSpace] = x },
	}
	for name, edit := range edits {
		s, a := sh, map[string]StoryboardAsset{}
		for k, v := range assets {
			a[k] = v
		}
		edit(&s, a)
		changed, err := BuildFrameSpec("p", s, ui, a, json.RawMessage(`{"k":1}`))
		if err != nil || SpecHash(changed) == base {
			t.Errorf("%s changes the spec hash (err %v)", name, err)
		}
	}
	colour := spec
	colour.TextColor = "#000000"
	if SpecHash(colour) == base {
		t.Error("the text colour changes the spec hash")
	}
	size := spec
	size.OnScreenText = []FrameText{{Text: spec.OnScreenText[0].Text, MinPx: spec.OnScreenText[0].MinPx + 4}}
	if SpecHash(size) == base {
		t.Error("the min_px of a text line changes the spec hash")
	}
	if kitChanged, _ := BuildFrameSpec("p", sh, ui, assets, json.RawMessage(`{"k":2}`)); SpecHash(kitChanged) == base {
		t.Error("the kit changes the spec hash")
	}

	f := DrawnFrame{FramePath: "storyboard/SH-001.png", SpecHash: base}
	if !f.Drawn() || f.Stale(base) || !f.Stale("other") || (DrawnFrame{}).Drawn() || (DrawnFrame{SpecHash: "x"}).Stale("y") {
		t.Error("a frame is stale when drawn from another spec; no image is not stale, it is missing")
	}
}

func sbShotContext(in StoryboardInput, st StoryboardState, unitID string, sh Shot) ShotContext {
	return st.ShotContextFor(in, unitID, sh)
}

func TestCheckShot(t *testing.T) {
	in := sbInput(t, nil)
	st := sbState(t, in)
	good := st.Units["VP-001"].Shots[0]
	if p := CheckShot(good, sbShotContext(in, st, "VP-001", good)); len(p) != 0 {
		t.Fatalf("a valid shot passes: %v", p)
	}
	cases := []struct {
		name  string
		unit  string
		edit  func(*Shot)
		stage func(*StoryboardState)
		check string
		text  string
	}{
		{"unit not in the plan", "VP-099", nil, nil, CheckShotUnit, "không có trong kế hoạch"},
		{"no script ref", "VP-001", func(s *Shot) { s.ScriptRefs = nil }, nil, CheckShotScript, "chưa phủ câu nào"},
		{"script ref of another unit", "VP-001", func(s *Shot) { s.ScriptRefs = []string{"b2.1"} }, nil, CheckShotScript, "b2.1"},
		{"unknown asset", "VP-001", func(s *Shot) { s.AssetRefs = []string{"AST-NOPE-001"} }, nil, CheckShotAssets, "không có"},
		{"asset not approved", "VP-001", nil, func(s *StoryboardState) { s.Assets[0].Status = AssetDrawn }, CheckShotAssets, "chưa duyệt"},
		{"position outside the vocabulary", "VP-001", func(s *Shot) { s.Composition.SubjectPosition = "split_screen" }, nil, CheckShotEnum, "vị trí"},
		{"camera outside the vocabulary", "VP-001", func(s *Shot) { s.Composition.CameraView = "dutch" }, nil, CheckShotEnum, "góc máy"},
		{"no subject", "VP-001", func(s *Shot) { s.Composition.Subject = "" }, nil, CheckShotEnum, "subject"},
		{"main object without position", "VP-001", func(s *Shot) { s.Composition.MainObjectPosition = "" }, nil, CheckShotEnum, "vị trí của vật chính"},
		{"position without main object", "VP-001", func(s *Shot) { s.Composition.MainObject = "" }, nil, CheckShotEnum, "không có vật chính"},
		{"text over the word limit", "VP-001", func(s *Shot) { s.OnScreenText = []string{"một hai ba bốn năm"} }, nil, CheckShotText, "dài quá 4 từ"},
		{"blank text", "VP-001", func(s *Shot) { s.OnScreenText = []string{" "} }, nil, CheckShotText, "trống"},
		{"no frame", "VP-001", nil, func(s *StoryboardState) { delete(s.Frames, "SH-001") }, CheckShotFrame, "chưa có ảnh khung"},
		{"stale frame", "VP-001", nil, func(s *StoryboardState) {
			s.Frames["SH-001"] = DrawnFrame{FramePath: "x.png", SpecHash: "old"}
		}, CheckShotFrame, "cũ"},
	}
	for _, c := range cases {
		s := sbState(t, in)
		if c.stage != nil {
			c.stage(&s)
		}
		sh := good
		if c.edit != nil {
			c.edit(&sh)
		}
		var found []string
		for _, p := range CheckShot(sh, sbShotContext(in, s, c.unit, sh)) {
			if p.Check == c.check {
				found = append(found, p.Detail)
			}
		}
		if !strings.Contains(strings.Join(found, "|"), c.text) {
			t.Errorf("%s: want a %s problem about %q, got %v", c.name, c.check, c.text, found)
		}
	}
	noLimit := st
	in2 := sbInput(t, nil)
	in2.Seq.Concept.RepresentationRules.Typography.Labels.MaxWords = 0
	long := good
	long.OnScreenText = []string{"một hai ba bốn năm sáu"}
	for _, p := range CheckShot(long, noLimit.ShotContextFor(in2, "VP-001", long)) {
		if p.Check == CheckShotText {
			t.Errorf("a concept without a word limit sets none: %v", p)
		}
	}
	if got := ShotChecksBlocking(good, sbShotContext(in, st, "VP-001", good)); len(got) != 0 {
		t.Errorf("no blockers: %v", got)
	}
}

func TestStoryboardChecks_BlockingAndWarnings(t *testing.T) {
	in := sbInput(t, nil)
	st := sbState(t, in)
	byID := func(cs []MachineCheck) map[string]MachineCheck {
		m := map[string]MachineCheck{}
		for _, c := range cs {
			m[c.ID] = c
		}
		return m
	}
	checks := StoryboardChecks(in, st)
	if len(checks) != 9 {
		t.Fatalf("six blocking checks and three warnings, got %d", len(checks))
	}
	for _, c := range checks {
		if !c.OK || c.Detail != "" {
			t.Errorf("a ready storyboard passes %s: %+v", c.ID, c)
		}
	}
	for _, id := range []string{CheckShotUnit, CheckShotScript, CheckShotAssets, CheckShotEnum, CheckShotText, CheckShotFrame} {
		if !IsBlockingCheck(id) {
			t.Errorf("%s blocks", id)
		}
	}
	for _, id := range []string{CheckAssetUnused, CheckUnitCover, CheckAssetCount} {
		if IsBlockingCheck(id) {
			t.Errorf("%s only warns", id)
		}
	}

	// A unit with no shot, an unused asset and an asset used by one shot warn.
	warn := sbState(t, in)
	delete(warn.Units, "VP-003")
	two := warn.Units["VP-002"]
	two.Shots[0].AssetRefs = []string{astAstronaut, astBlackHole}
	one := warn.Units["VP-001"]
	one.Shots[0].AssetRefs = []string{astAstronaut, astBlackHole}
	warn.Units["VP-001"], warn.Units["VP-002"] = one, two
	for unitID, u := range warn.Units {
		h, err := warn.SpecHashOf(in, unitID, u.Shots[0])
		if err != nil {
			t.Fatal(err)
		}
		warn.Frames[u.Shots[0].ShotID] = DrawnFrame{FramePath: "x.png", SpecHash: h}
	}
	warn.Assets = append(warn.Assets, StoryboardAsset{AssetID: "AST-LAB-001", Name: "Lab", Kind: AssetKindBackdrop, Tier: AssetTierBackground, Status: AssetApproved})
	warn.Assets = append(warn.Assets, StoryboardAsset{AssetID: "AST-DRAFT-001", Name: "Draft", Kind: AssetKindFigure, Tier: AssetTierSupport, Status: AssetDrawn})
	m := byID(StoryboardChecks(in, warn))
	if c := m[CheckUnitCover]; c.OK || !strings.Contains(c.Detail, "VP-003") {
		t.Errorf("m-unit-cover names the unit: %+v", c)
	}
	if c := m[CheckAssetUnused]; c.OK || !strings.Contains(c.Detail, "AST-LAB-001") || strings.Contains(c.Detail, "AST-DRAFT-001") {
		t.Errorf("m-asset-unused lists approved assets no shot uses (the drawn one is not yet part of the cast): %+v", c)
	}
	if !m[CheckShotFrame].OK {
		t.Errorf("a missing unit does not break the frame check: %+v", m[CheckShotFrame])
	}

	// Blocking failures name the shot and aggregate.
	bad := sbState(t, in)
	u := bad.Units["VP-001"]
	u.Shots[0].ScriptRefs = []string{"b1.1"}
	u.Shots[0].OnScreenText = []string{"một hai ba bốn năm"}
	bad.Units["VP-001"] = u
	bad.Units["VP-099"] = UnitShots{Shots: []Shot{sbShot("SH-050", "b1.1")}}
	m = byID(StoryboardChecks(in, bad))
	if c := m[CheckShotScript]; c.OK || !strings.Contains(c.Detail, "VP-001: câu b1.2 chưa có shot nào phủ") {
		t.Errorf("a line no shot covers fails m-shot-script: %+v", c)
	}
	if c := m[CheckShotText]; c.OK || !strings.Contains(c.Detail, "SH-001") {
		t.Errorf("m-shot-text names the shot: %+v", c)
	}
	if c := m[CheckShotUnit]; c.OK || !strings.Contains(c.Detail, "SH-050") {
		t.Errorf("a shot of a unit the plan lacks fails m-shot-unit: %+v", c)
	}
}

func TestParseAssetsReply(t *testing.T) {
	ctx := AssetsReplyContext{
		ConceptIDs: map[string]bool{"astronaut": true, "black_hole": true},
		PlanUnits:  map[string]bool{"VP-001": true, "VP-002": true},
		Library:    map[string]string{"ILL-7": AssetKindFigure, "ILL-8": AssetKindBackdrop},
		Current:    map[string]bool{astAstronaut: true, astBlackHole: true},
		InUse:      map[string]bool{astAstronaut: true},
	}
	asset := func(extra string) string {
		return `{"same_as":"","concept_id":"astronaut","name":"Astronaut","kind":"figure","tier":"hero","base_illustration_id":"","units":["VP-001","VP-002"]` + extra + `}`
	}
	good := `{"assets":[` + asset("") + `]}`
	fresh := ctx
	fresh.InUse = nil
	got, err := ParseAssetsReply([]byte(good), fresh)
	if err != nil || len(got) != 1 || got[0].Name != "Astronaut" || len(got[0].Units) != 2 || got[0].Units[1] != "VP-002" {
		t.Fatalf("a valid reply: %+v %v", got, err)
	}
	if got, err := ParseAssetsReply([]byte(`{"assets":[{"same_as":"AST-ASTRONAUT-001","concept_id":"","name":"Space","kind":"backdrop","tier":"background","base_illustration_id":"ILL-8","units":null}]}`), ctx); err != nil || got[0].SameAs != astAstronaut || got[0].Units == nil {
		t.Errorf("a backdrop needs no concept, may keep an asset, take a library backdrop and appear in no unit: %+v %v", got, err)
	}
	r := strings.NewReplacer
	cases := map[string]string{
		"not json":                `nope`,
		"unknown key":             `{"assets":[` + asset("") + `],"extra":1}`,
		"asset_id key":            `{"assets":[` + asset(`,"asset_id":"AST-X-001"`) + `]}`,
		"detail key":              `{"assets":[` + asset(`,"portrayal":"nhỏ bé"`) + `]}`,
		"palette key":             `{"assets":[` + asset(`,"palette_tokens":[]`) + `]}`,
		"no asset":                `{"assets":[]}`,
		"bad kind":                `{"assets":[` + r(`"kind":"figure"`, `"kind":"prop"`).Replace(asset("")) + `]}`,
		"bad tier":                `{"assets":[` + r(`"tier":"hero"`, `"tier":"star"`).Replace(asset("")) + `]}`,
		"bad name":                `{"assets":[` + r(`"name":"Astronaut"`, `"name":"astro naut"`).Replace(asset("")) + `]}`,
		"concept not in rules":    `{"assets":[` + r(`"concept_id":"astronaut"`, `"concept_id":"rocket"`).Replace(asset("")) + `]}`,
		"figure without concept":  `{"assets":[` + r(`"concept_id":"astronaut"`, `"concept_id":""`).Replace(asset("")) + `]}`,
		"unit not in the plan":    `{"assets":[` + r(`"VP-002"`, `"VP-099"`).Replace(asset("")) + `]}`,
		"unit twice":              `{"assets":[` + r(`"VP-002"`, `"VP-001"`).Replace(asset("")) + `]}`,
		"units not an array":      `{"assets":[` + r(`["VP-001","VP-002"]`, `"VP-001"`).Replace(asset("")) + `]}`,
		"base not in library":     `{"assets":[` + r(`"base_illustration_id":""`, `"base_illustration_id":"ILL-1"`).Replace(asset("")) + `]}`,
		"base of another kind":    `{"assets":[` + r(`"base_illustration_id":""`, `"base_illustration_id":"ILL-8"`).Replace(asset("")) + `]}`,
		"same_as of no asset":     `{"assets":[` + r(`"same_as":""`, `"same_as":"AST-X-009"`).Replace(asset("")) + `]}`,
		"same_as twice":           `{"assets":[` + r(`"same_as":""`, `"same_as":"AST-ASTRONAUT-001"`).Replace(asset("")) + `,` + r(`"same_as":""`, `"same_as":"AST-ASTRONAUT-001"`).Replace(asset("")) + `]}`,
		"drops an asset in use":   `{"assets":[` + r(`"concept_id":"astronaut"`, `"concept_id":"black_hole"`).Replace(r(`"same_as":""`, `"same_as":"AST-BLACKHOLE-001"`).Replace(asset(""))) + `]}`,
		"asset list not an array": `{"assets":{}}`,
	}
	for name, raw := range cases {
		if _, err := ParseAssetsReply([]byte(raw), ctx); !errors.Is(err, ErrUnreadableReply) {
			t.Errorf("%s must be refused, got %v", name, err)
		}
	}
	dropUnused := strings.Replace(good, `"same_as":""`, `"same_as":"AST-ASTRONAUT-001"`, 1)
	if _, err := ParseAssetsReply([]byte(dropUnused), ctx); err != nil {
		t.Errorf("dropping an asset no shot uses is fine: %v", err)
	}
}

func TestParseAssetDetailReply(t *testing.T) {
	palette := map[string]bool{"ink": true, "gold": true}
	good := `{"detail":{"portrayal":"nhỏ bé","must_not_imply":["anh hùng"],"palette_tokens":["ink"],"expressions":["ngước nhìn"]}}`
	d, err := ParseAssetDetailReply([]byte(good), palette)
	if err != nil || d.Portrayal != "nhỏ bé" || len(d.PaletteTokens) != 1 || d.Expressions[0] != "ngước nhìn" {
		t.Fatalf("a valid reply: %+v %v", d, err)
	}
	d, err = ParseAssetDetailReply([]byte(`{"detail":{"portrayal":"nền","must_not_imply":null,"palette_tokens":null,"expressions":null}}`), palette)
	if err != nil || d.MustNotImply == nil || d.PaletteTokens == nil || d.Expressions == nil {
		t.Errorf("null lists become empty ones: %+v %v", d, err)
	}
	cases := map[string]string{
		"not json":             `nope`,
		"no detail":            `{}`,
		"null detail":          `{"detail":null}`,
		"unknown key":          `{"detail":{"portrayal":"x","must_not_imply":[],"palette_tokens":[],"expressions":[]},"extra":1}`,
		"unknown key inside":   `{"detail":{"portrayal":"x","must_not_imply":[],"palette_tokens":[],"expressions":[],"tier":"hero"}}`,
		"frame field":          `{"detail":{"portrayal":"x","name":"Astro"}}`,
		"empty portrayal":      `{"detail":{"portrayal":"  ","must_not_imply":[],"palette_tokens":[],"expressions":[]}}`,
		"token not in palette": `{"detail":{"portrayal":"x","must_not_imply":[],"palette_tokens":["neon"],"expressions":[]}}`,
		"list not an array":    `{"detail":{"portrayal":"x","must_not_imply":"a","palette_tokens":[],"expressions":[]}}`,
	}
	for name, raw := range cases {
		if _, err := ParseAssetDetailReply([]byte(raw), palette); !errors.Is(err, ErrUnreadableReply) {
			t.Errorf("%s must be refused, got %v", name, err)
		}
	}
}

func TestStoryboardAssetHasDetailOnlyWithAPortrayal(t *testing.T) {
	if (StoryboardAsset{}).HasDetail() || (StoryboardAsset{Portrayal: "  "}).HasDetail() || !(StoryboardAsset{Portrayal: "x"}).HasDetail() {
		t.Error("has detail means a non-blank portrayal")
	}
}

func TestParseShotsReply(t *testing.T) {
	ctx := ShotsReplyContext{
		UnitRefs:       []string{"b2.1", "b2.2"},
		ApprovedAssets: map[string]bool{astAstronaut: true, astBlackHole: true},
		Typography:     Typography{Labels: Labels{MaxWords: 3}},
	}
	body := func(refs, comp, text, assets string) string {
		return `{"shots":[{"script_refs":` + refs + `,"shot_purpose":"so sánh","composition":` + comp + `,"action_summary":"hai cảnh","on_screen_text":` + text + `,"asset_refs":` + assets + `,"notes":[]}]}`
	}
	comp := `{"subject":"AST-ASTRONAUT-001","subject_position":"lower_left","main_object":"","main_object_position":"","camera_view":"wide"}`
	assets := `["AST-ASTRONAUT-001"]`
	got, err := ParseShotsReply([]byte(body(`["b2.1","b2.2"]`, comp, `["Góc nhìn A"]`, assets)), ctx)
	if err != nil || len(got) != 1 || got[0].ShotID != "" || got[0].Status != UnitPending || got[0].Notes == nil {
		t.Fatalf("a valid reply, shots without ids and pending: %+v %v", got, err)
	}
	split := `{"subject":"AST-ASTRONAUT-001","subject_position":"split_screen","main_object":"AST-BLACKHOLE-001","main_object_position":"split_screen","camera_view":"comparison"}`
	got, err = ParseShotsReply([]byte(body(`["b2.1","b2.2"]`, split, `[]`, `["AST-ASTRONAUT-001","AST-BLACKHOLE-001"]`)), ctx)
	if err != nil || got[0].Composition.SubjectPosition != "split_left" || got[0].Composition.MainObjectPosition != "split_right" {
		t.Errorf("split_screen with two objects becomes the two halves: %+v %v", got, err)
	}
	twoShots := `{"shots":[` + strings.TrimSuffix(strings.TrimPrefix(body(`["b2.1"]`, comp, `[]`, assets), `{"shots":[`), `]}`) + `,` +
		strings.TrimSuffix(strings.TrimPrefix(body(`["b2.2"]`, comp, `[]`, assets), `{"shots":[`), `]}`) + `]}`
	if got, err := ParseShotsReply([]byte(twoShots), ctx); err != nil || len(got) != 2 {
		t.Errorf("shots that together cover the unit: %+v %v", got, err)
	}
	cases := map[string]string{
		"not json":                  `nope`,
		"unknown key":               `{"shots":[],"extra":1}`,
		"shot_id key":               strings.Replace(body(`["b2.1","b2.2"]`, comp, `[]`, assets), `"script_refs"`, `"shot_id":"SH-001","script_refs"`, 1),
		"unit_id key":               strings.Replace(body(`["b2.1","b2.2"]`, comp, `[]`, assets), `"script_refs"`, `"unit_id":"VP-002","script_refs"`, 1),
		"unknown key in comp":       body(`["b2.1","b2.2"]`, strings.Replace(comp, `"camera_view"`, `"lens":"x","camera_view"`, 1), `[]`, assets),
		"no shot":                   `{"shots":[]}`,
		"no script refs":            body(`[]`, comp, `[]`, assets),
		"ref outside the unit":      body(`["b2.1","b2.2","b1.1"]`, comp, `[]`, assets),
		"line not covered":          body(`["b2.1"]`, comp, `[]`, assets),
		"asset not approved":        body(`["b2.1","b2.2"]`, comp, `[]`, `["AST-ASTRONAUT-001","AST-GHOST-001"]`),
		"bad position":              body(`["b2.1","b2.2"]`, strings.Replace(comp, "lower_left", "bottom", 1), `[]`, assets),
		"bad camera":                body(`["b2.1","b2.2"]`, strings.Replace(comp, "wide", "birdseye", 1), `[]`, assets),
		"split_screen one object":   body(`["b2.1","b2.2"]`, strings.Replace(comp, "lower_left", "split_screen", 1), `[]`, assets),
		"asset subject not in refs": body(`["b2.1","b2.2"]`, comp, `[]`, `[]`),
		"text over the limit":       body(`["b2.1","b2.2"]`, comp, `["một hai ba bốn"]`, assets),
		"blank text":                body(`["b2.1","b2.2"]`, comp, `[" "]`, assets),
		"no purpose":                strings.Replace(body(`["b2.1","b2.2"]`, comp, `[]`, assets), `"so sánh"`, `""`, 1),
	}
	for name, raw := range cases {
		if _, err := ParseShotsReply([]byte(raw), ctx); !errors.Is(err, ErrUnreadableReply) {
			t.Errorf("%s must be refused, got %v", name, err)
		}
	}
}

func TestMergeShots_KeepsIdsAndCreatorFieldsByPlaceAndNeverReusesNumbers(t *testing.T) {
	old := UnitShots{NextSeq: 6, Shots: []Shot{
		{ShotID: "SH-003", ShotPurpose: "cũ 1", ActionSummary: "cũ a", Status: UnitApproved, Notes: []string{"tay"}},
		{ShotID: "SH-004", ShotPurpose: "cũ 2", Status: UnitApproved},
		{ShotID: "SH-005", ShotPurpose: "cũ 3", Status: UnitApproved},
	}}
	fresh := func(n int) []Shot {
		out := make([]Shot, n)
		for i := range out {
			out[i] = Shot{ShotPurpose: "mới", ActionSummary: "mới", Status: UnitPending}
		}
		return out
	}
	fields := []string{"SH-003.shot_purpose", "SH-003.notes", "SH-004.bogus", "SH-009.notes"}
	got, kept, err := MergeShots(fresh(4), old, fields, 6)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Shots) != 4 || got.Shots[0].ShotID != "SH-003" || got.Shots[2].ShotID != "SH-005" || got.Shots[3].ShotID != "SH-006" || got.NextSeq != 7 {
		t.Errorf("the n-th shot keeps the n-th id, extra ones take the next numbers: %+v", got)
	}
	if got.Shots[0].ShotPurpose != "cũ 1" || got.Shots[0].ActionSummary != "mới" || len(got.Shots[0].Notes) != 1 || got.Shots[1].ShotPurpose != "mới" {
		t.Errorf("only the Creator's fields stand: %+v", got.Shots[:2])
	}
	if strings.Join(kept, ",") != "SH-003.shot_purpose,SH-003.notes" {
		t.Errorf("kept fields: %v", kept)
	}
	for _, sh := range got.Shots {
		if sh.Status != UnitPending {
			t.Errorf("every shot goes back to review: %+v", sh)
		}
	}
	shrunk, _, err := MergeShots(fresh(2), old, nil, 6)
	if err != nil || len(shrunk.Shots) != 2 || shrunk.NextSeq != 6 {
		t.Errorf("fewer shots drop the last ones and the counter holds: %+v %v", shrunk, err)
	}
	if again, _, _ := MergeShots(fresh(4), UnitShots{Shots: shrunk.Shots, NextSeq: shrunk.NextSeq}, nil, shrunk.NextSeq); again.Shots[2].ShotID != "SH-006" {
		t.Errorf("a dropped number is not given again: %+v", again.Shots)
	}
	if _, _, err := MergeShots(fresh(1), old, []string{"SH-005.notes"}, 6); !errors.Is(err, ErrUnreadableReply) {
		t.Errorf("dropping a shot the Creator edited is refused: %v", err)
	}
	if first, _, err := MergeShots(fresh(2), UnitShots{}, nil, 0); err != nil || first.Shots[0].ShotID != "SH-001" || first.Shots[1].ShotID != "SH-002" || first.NextSeq != 3 {
		t.Errorf("a first write numbers from the counter: %+v %v", first, err)
	}
}

func TestParseStoryboardReviewReply(t *testing.T) {
	shots := map[string]bool{"SH-001": true}
	units := map[string]bool{"VP-001": true}
	assets := map[string]bool{astAstronaut: true}
	item := func(key, field string) string {
		return `{"findings":[{"group":"clarity","unit_key":"` + key + `","field":"` + field + `","text":"vấn đề","detail":"d","fix":"","same_as":""}],"notes":["ổn"]}`
	}
	for _, ok := range [][2]string{{"SH-001", "shot_purpose"}, {"SH-001", "composition"}, {"SH-001", ""}, {"VP-001", ""}, {astAstronaut, "portrayal"}} {
		if r, err := ParseStoryboardReviewReply([]byte(item(ok[0], ok[1])), ReviewState{}, shots, units, assets); err != nil || len(r.Findings) != 1 {
			t.Errorf("%v must be accepted: %v", ok, err)
		}
	}
	for _, bad := range [][2]string{{"SH-009", ""}, {"MOTIF-01", ""}, {"SH-001", "portrayal"}, {astAstronaut, "action_summary"}, {"VP-001", "notes"}} {
		if _, err := ParseStoryboardReviewReply([]byte(item(bad[0], bad[1])), ReviewState{}, shots, units, assets); !errors.Is(err, ErrUnreadableReply) {
			t.Errorf("%v must be refused: %v", bad, err)
		}
	}
	badGroup := strings.Replace(item("SH-001", ""), "clarity", "style", 1)
	if _, err := ParseStoryboardReviewReply([]byte(badGroup), ReviewState{}, shots, units, assets); !errors.Is(err, ErrUnreadableReply) {
		t.Errorf("a group outside the four: %v", err)
	}
}

func TestStoryboardLockBlockersAndDocument(t *testing.T) {
	in := sbInput(t, nil)
	at := time.Unix(1_700_000_000, 0).UTC()
	st := sbState(t, in)
	if b := StoryboardLockBlockers(in, st); len(b) != 0 {
		t.Fatalf("a ready storyboard has no blocker: %v", b)
	}
	cases := []struct {
		name string
		edit func(*StoryboardInput, *StoryboardState)
		want string
	}{
		{"outdated steps", func(i *StoryboardInput, _ *StoryboardState) { i.Outdated = []string{"kịch bản đổi"} }, "đã cũ"},
		{"asset not approved", func(_ *StoryboardInput, s *StoryboardState) { s.Assets[2].Status = AssetDrawn }, "1 tài sản chưa duyệt (" + astSpace + ")"},
		{"unit without shot", func(_ *StoryboardInput, s *StoryboardState) { delete(s.Units, "VP-003") }, "1 unit chưa có shot (VP-003)"},
		{"shot not approved", func(_ *StoryboardInput, s *StoryboardState) { u := s.Units["VP-002"]; u.Shots[0].Status = UnitPending }, "1 shot chưa duyệt (SH-002)"},
		{"frame stale", func(_ *StoryboardInput, s *StoryboardState) {
			s.Frames["SH-003"] = DrawnFrame{FramePath: "x.png", SpecHash: "old"}
		}, "ảnh khung cũ"},
		{"frame missing", func(_ *StoryboardInput, s *StoryboardState) { delete(s.Frames, "SH-001") }, "chưa có ảnh khung"},
		{"never reviewed", func(_ *StoryboardInput, s *StoryboardState) { s.Review.Rounds = 0 }, "Chưa rà soát"},
		{"open finding", func(_ *StoryboardInput, s *StoryboardState) {
			s.Review.Findings = []Finding{{ID: "f1", State: FindingOpen}}
		}, "1 vấn đề rà soát chưa xử lý"},
		{"warning not accepted", func(_ *StoryboardInput, s *StoryboardState) {
			s.Assets = append(s.Assets, StoryboardAsset{AssetID: "AST-LAB-001", Kind: AssetKindBackdrop, Tier: AssetTierBackground, Status: AssetApproved})
		}, "1 mục chưa đạt (AST-LAB-001)"},
	}
	for _, c := range cases {
		i, s := sbInput(t, nil), sbState(t, in)
		c.edit(&i, &s)
		b := strings.Join(StoryboardLockBlockers(i, s), "|")
		if !strings.Contains(b, c.want) {
			t.Errorf("%s: want a blocker %q, got %q", c.name, c.want, b)
		}
		if _, err := BuildStoryboardDocument("p", 1, at, i, s); !errors.Is(err, ErrLockNotReady) {
			t.Errorf("%s: cannot lock, got %v", c.name, err)
		}
	}

	// An accepted warning does not block; an accepted blocking check still does.
	warn := sbState(t, in)
	warn.Assets = append(warn.Assets, StoryboardAsset{AssetID: "AST-LAB-001", Name: "Lab", Kind: AssetKindBackdrop, Tier: AssetTierBackground, Status: AssetApproved})
	var unused MachineCheck
	for _, c := range StoryboardChecks(in, warn) {
		if c.ID == CheckAssetUnused {
			unused = c
		}
	}
	warn.Review.Checks[CheckAssetUnused] = CheckDecision{Why: "để dành", Fingerprint: Fingerprint(unused.Detail)}
	if b := StoryboardLockBlockers(in, warn); len(b) != 0 {
		t.Errorf("an accepted warning lets the storyboard lock: %v", b)
	}
	stale := sbState(t, in)
	delete(stale.Frames, "SH-001")
	stale.Review.Checks[CheckShotFrame] = CheckDecision{Why: "kệ", Fingerprint: Fingerprint("SH-001 chưa có ảnh khung")}
	if b := StoryboardLockBlockers(in, stale); len(b) == 0 {
		t.Error("a blocking check cannot be accepted")
	}

	doc, err := BuildStoryboardDocument("p", 2, at, in, func() StoryboardState {
		s := sbState(t, in)
		s.Review.Notes = []string{"liền mạch"}
		s.Review.Findings = []Finding{{ID: "f1", Group: "clarity", Text: "chữ dài", State: FindingFixed}}
		// Shot ids out of order inside a unit sort by number; units keep sequence order.
		s.Units["VP-002"] = UnitShots{Shots: []Shot{sbShot("SH-020", "b2.1"), sbShot("SH-010", "b2.1")}}
		for _, id := range []string{"SH-020", "SH-010"} {
			s.Frames[id] = s.Frames["SH-002"]
		}
		return s
	}())
	if err != nil {
		t.Fatal(err)
	}
	if doc.DocumentID != "storyboard-v2" || doc.Status != "locked" || doc.Version != 2 || doc.SchemaVersion != SchemaVersion ||
		doc.SourceVersions != (StoryboardSourceVersions{Script: 2, VisualConcept: 1, VisualPlan: 1, VisualSequence: 1}) {
		t.Errorf("ids, status and sources: %+v", doc)
	}
	var order []string
	for _, sh := range doc.Shots {
		order = append(order, sh.ShotID+"@"+sh.UnitID+"/"+sh.BeatID)
	}
	if strings.Join(order, " ") != "SH-001@VP-001/b1 SH-010@VP-002/b2 SH-020@VP-002/b2 SH-003@VP-003/b2" {
		t.Errorf("shots follow the sequence, then shot number: %v", order)
	}
	if doc.Shots[0].FramePath != "storyboard/v2/SH-001.png" || len(doc.Assets) != 3 || doc.Assets[0].PreviewPath == "" {
		t.Errorf("frame path of the version and assets: %+v %+v", doc.Shots[0], doc.Assets)
	}
	if len(doc.Notes) != 1 || len(doc.Review.Comments) != 1 || doc.Review.Comments[0].Decision != FindingFixed {
		t.Errorf("review notes and comments: %+v", doc)
	}
	raw, err := json.Marshal(doc)
	if err != nil || strings.Contains(string(raw), "null") {
		t.Errorf("no null in the document: %s %v", raw, err)
	}

	// Locking twice without a change is nothing to lock; any change is one.
	same, _ := BuildStoryboardDocument("p", 1, at, in, st)
	next, _ := BuildStoryboardDocument("p", 2, at.Add(time.Hour), in, st)
	if !SameStoryboardContent(same, next) {
		t.Error("the same storyboard in another version is the same content")
	}
	changed := sbState(t, in)
	u := changed.Units["VP-001"]
	u.Shots[0].ShotPurpose = "đổi"
	changed.Units["VP-001"] = u
	diff, _ := BuildStoryboardDocument("p", 2, at, in, changed)
	if SameStoryboardContent(same, diff) {
		t.Error("an edited shot is a change")
	}
	redrawn := sbState(t, in)
	redrawn.Assets[0].PreviewPath = "storyboard/assets/a/v2.png"
	diff, _ = BuildStoryboardDocument("p", 2, at, in, redrawn)
	if SameStoryboardContent(same, diff) {
		t.Error("a redrawn asset is a change")
	}
}

func TestStoryboardPromptsShipWithTheOtherSteps(t *testing.T) {
	ps := SystemPrompts()
	if len(ps) != 19 {
		t.Fatalf("%d prompts", len(ps))
	}
	want := map[string][]string{
		PromptRoleStoryboardAssets:      {"brief", "representation", "plan", "motifs", "library", "current_assets", "creator_fields", "note"},
		PromptRoleStoryboardAssetDetail: {"brief", "asset", "concept", "palette", "units", "other_assets", "current", "creator_fields", "note"},
		PromptRoleStoryboardUnit: {"brief", "composition_rules", "typography", "journey", "beat", "unit", "sequence_item", "assets", "positions",
			"camera_views", "prev_shots", "current_shots", "creator_fields", "note"},
		PromptRoleStoryboardReview: {"brief", "do_not_use", "plan", "sequence", "assets", "shots", "checks", "previous_findings", "note"},
	}
	if PromptRoleStoryboardAssets != "storyboard_assets" || PromptRoleStoryboardAssetDetail != "storyboard_asset_detail" || PromptRoleStoryboardUnit != "storyboard_unit" || PromptRoleStoryboardReview != "storyboard_review" {
		t.Error("role names")
	}
	vars := regexp.MustCompile(`\{\{([a-z_]+)\}\}`)
	for i, role := range []string{PromptRoleStoryboardAssets, PromptRoleStoryboardAssetDetail, PromptRoleStoryboardUnit, PromptRoleStoryboardReview} {
		p := ps[15+i]
		if p.Role != role || p.Name != SystemPromptName || !strings.Contains(p.TemplateText, "```json") {
			t.Fatalf("%d: %s is not shipped as written", 15+i, p.Role)
		}
		got := map[string]bool{}
		for _, m := range vars.FindAllStringSubmatch(p.TemplateText, -1) {
			got[m[1]] = true
		}
		if len(got) != len(want[role]) {
			t.Errorf("%s fills %v, want %v", role, got, want[role])
		}
		for _, v := range want[role] {
			if !got[v] {
				t.Errorf("%s lacks the variable {{%s}}", role, v)
			}
		}
	}
}

func TestAppendGroupedNamesThreeIdsAndCount(t *testing.T) {
	ids := make([]string, 67)
	for i := range ids {
		ids[i] = fmt.Sprintf("VP-%03d", i+1)
	}
	cases := []struct {
		name  string
		what  string
		noun  string
		items []string
		want  []string
	}{
		{"67 units without shot", "unit chưa có shot", "unit", ids, []string{"67 unit chưa có shot (VP-001, VP-002, VP-003 và 64 unit khác)"}},
		{"three shots", "shot chưa duyệt", "shot", ids[:3], []string{"3 shot chưa duyệt (VP-001, VP-002, VP-003)"}},
		{"67 assets", "tài sản chưa duyệt", "tài sản", ids, []string{"67 tài sản chưa duyệt (VP-001, VP-002, VP-003 và 64 tài sản khác)"}},
		{"check items", "Mỗi shot có ảnh khung", "mục", ids, []string{"Mỗi shot có ảnh khung: 67 mục chưa đạt (VP-001, VP-002, VP-003 và 64 mục khác)"}},
		{"none", "shot chưa duyệt", "shot", nil, nil},
	}
	for _, c := range cases {
		got := appendGrouped(nil, c.what, c.noun, c.items)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: got %q want %q", c.name, got, c.want)
		}
	}
}

func TestStoryboardLockBlockersGroupManyUnitsWithoutShots(t *testing.T) {
	in := sbInput(t, nil)
	st := sbState(t, in)
	for _, id := range in.UnitIDs() {
		delete(st.Units, id)
	}
	b := StoryboardLockBlockers(in, st)
	n := 0
	for _, m := range b {
		if strings.HasPrefix(m, "Unit ") {
			t.Errorf("one blocker per unit: %q", m)
		}
		if strings.Contains(m, "unit chưa có shot") {
			n++
		}
	}
	if n != 1 {
		t.Errorf("want exactly one grouped no-shot blocker, got %d in %q", n, b)
	}
	if len(b) > 8 {
		t.Errorf("blockers list must stay short, got %d: %q", len(b), b)
	}
}
