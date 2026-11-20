package domain

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Sources of a unit's shots. Each has one hash, so a unit whose source changed
// since its shots were written reads as stale.
const (
	// StoryboardSourceSeq is the unit's item in the locked visual sequence.
	StoryboardSourceSeq = "seq"
	// StoryboardSourcePlan is the unit in the locked visual plan.
	StoryboardSourcePlan = "plan"
	// StoryboardSourceScript is the unit's lines in the locked script.
	StoryboardSourceScript = "script"
	// StoryboardSourceAssets is the versions of the assets the unit's shots use.
	StoryboardSourceAssets = "assets"
)

// StoryboardSourceLabels names each source of a unit for the Creator.
var StoryboardSourceLabels = map[string]string{
	StoryboardSourceSeq:    "Chuỗi hình của unit",
	StoryboardSourcePlan:   "Unit trong kế hoạch hình đã chốt",
	StoryboardSourceScript: "Lời thoại của unit",
	StoryboardSourceAssets: "Phiên bản các tài sản mà shot dùng",
}

var storyboardSourceOrder = []string{StoryboardSourceSeq, StoryboardSourcePlan, StoryboardSourceScript, StoryboardSourceAssets}

// StoryboardSourceVersions are the locked versions a storyboard was written from.
type StoryboardSourceVersions struct {
	Script         int `json:"script"`
	VisualConcept  int `json:"visual_concept"`
	VisualPlan     int `json:"visual_plan"`
	VisualSequence int `json:"visual_sequence"`
}

// StoryboardInput is everything step 11 reads: the latest locked visual
// sequence, visual plan, visual concept and script.
type StoryboardInput struct {
	// Seq holds the locked script, visual concept and visual plan.
	Seq SequenceInput
	// SequenceVersion and Sequence are the latest locked visual sequence.
	SequenceVersion int
	Sequence        SequenceDocument
	// Outdated says why the sequence, or the plan under it, was written from an
	// older script, visual concept or plan than the latest locked ones; empty when
	// it was not.
	Outdated []string

	unitHashes map[string]map[string]string
}

// NewStoryboardInput reads the latest locked script, visual concept, visual plan
// and visual sequence documents.
func NewStoryboardInput(sv ScriptVersion, conceptVersion int, conceptDoc []byte, planVersion int, planDoc []byte, sequenceVersion int, sequenceDoc []byte) (StoryboardInput, error) {
	seq, err := NewSequenceInput(sv, conceptVersion, conceptDoc, planVersion, planDoc)
	if err != nil {
		return StoryboardInput{}, err
	}
	var doc SequenceDocument
	if err := json.Unmarshal(sequenceDoc, &doc); err != nil {
		return StoryboardInput{}, fmt.Errorf("đọc chuỗi hình đã chốt v%d: %w", sequenceVersion, err)
	}
	in := StoryboardInput{Seq: seq, SequenceVersion: sequenceVersion, Sequence: doc, Outdated: append([]string{}, seq.Outdated...)}
	sv2 := doc.SourceVersions
	if sv2.Script != seq.Script.ScriptVersion {
		in.Outdated = append(in.Outdated, fmt.Sprintf("chuỗi hình viết từ kịch bản v%d, kịch bản đã chốt v%d", sv2.Script, seq.Script.ScriptVersion))
	}
	if sv2.VisualConcept != conceptVersion {
		in.Outdated = append(in.Outdated, fmt.Sprintf("chuỗi hình viết từ ngôn ngữ hình ảnh v%d, ngôn ngữ hình ảnh đã chốt v%d", sv2.VisualConcept, conceptVersion))
	}
	if sv2.VisualPlan != planVersion {
		in.Outdated = append(in.Outdated, fmt.Sprintf("chuỗi hình viết từ kế hoạch hình v%d, kế hoạch hình đã chốt v%d", sv2.VisualPlan, planVersion))
	}
	in.hashSources()
	return in, nil
}

// SourceVersions are the locked versions the storyboard reads now.
func (in StoryboardInput) SourceVersions() StoryboardSourceVersions {
	return StoryboardSourceVersions{
		Script: in.Seq.Script.ScriptVersion, VisualConcept: in.Seq.ConceptVersion,
		VisualPlan: in.Seq.PlanVersion, VisualSequence: in.SequenceVersion,
	}
}

// UnitIDs lists the plan units in plan order, which is the order of the sequence.
func (in StoryboardInput) UnitIDs() []string {
	out := make([]string, 0, len(in.Seq.Plan.Units))
	for _, u := range in.Seq.Plan.Units {
		out = append(out, u.UnitID)
	}
	return out
}

// seqItem returns the sequence item of unit id.
func (in StoryboardInput) seqItem(unitID string) (DocumentSeqItem, bool) {
	for _, it := range in.Sequence.Sequence {
		if it.UnitID == unitID {
			return it, true
		}
	}
	return DocumentSeqItem{}, false
}

// planUnit returns the plan unit id.
func (in StoryboardInput) planUnit(unitID string) (DocumentUnit, bool) {
	for _, u := range in.Seq.Plan.Units {
		if u.UnitID == unitID {
			return u, true
		}
	}
	return DocumentUnit{}, false
}

// unitLines returns the script lines unit refs name, in the order of the refs.
func (in StoryboardInput) unitLines(refs []string) []Line {
	byID := map[string]Line{}
	for _, lines := range in.Seq.Script.Lines {
		for _, l := range lines {
			byID[l.ID] = l
		}
	}
	out := make([]Line, 0, len(refs))
	for _, r := range refs {
		if l, ok := byID[r]; ok {
			out = append(out, l)
		}
	}
	return out
}

func (in *StoryboardInput) hashSources() {
	in.unitHashes = map[string]map[string]string{}
	for _, u := range in.Seq.Plan.Units {
		h := map[string]string{StoryboardSourcePlan: unitHash(u)}
		if it, ok := in.seqItem(u.UnitID); ok {
			it.SequenceIndex = 0
			raw, _ := json.Marshal(it)
			h[StoryboardSourceSeq] = hashText(string(raw))
		} else {
			h[StoryboardSourceSeq] = hashText("")
		}
		var b strings.Builder
		for _, l := range in.unitLines(u.ScriptRefs) {
			b.WriteString(l.ID + "=" + l.Text + "\n")
		}
		h[StoryboardSourceScript] = hashText(b.String())
		in.unitHashes[u.UnitID] = h
	}
}

// assetsHash hashes the versions of the assets shots use, in id order.
func assetsHash(shots []Shot, assets map[string]StoryboardAsset) string {
	used := map[string]bool{}
	for _, sh := range shots {
		for _, a := range sh.AssetRefs {
			used[a] = true
		}
	}
	ids := make([]string, 0, len(used))
	for id := range used {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	var b strings.Builder
	for _, id := range ids {
		fmt.Fprintf(&b, "%s@%d\n", id, assets[id].Version)
	}
	return hashText(b.String())
}

// UnitHashes is the hash of every source the shots of unit unitID are written
// from; assets are the project's assets by id.
func (in StoryboardInput) UnitHashes(unitID string, shots []Shot, assets map[string]StoryboardAsset) map[string]string {
	out := map[string]string{StoryboardSourceAssets: assetsHash(shots, assets)}
	for k, v := range in.unitHashes[unitID] {
		out[k] = v
	}
	return out
}

// UnitStaleSources lists the sources, by constant, that differ between the hashes
// a unit's shots were written from and the hashes now.
func (in StoryboardInput) UnitStaleSources(unitID string, stored map[string]string, shots []Shot, assets map[string]StoryboardAsset) []string {
	return staleSources(storyboardSourceOrder, stored, in.UnitHashes(unitID, shots, assets))
}

// UnitInput is what the writer of one unit's shots reads.
type UnitInput struct {
	Unit DocumentUnit
	Beat SequenceBeat
	// Item is the unit's item in the locked sequence.
	Item DocumentSeqItem
	// Lines are the script lines the unit's references name.
	Lines []Line
	// Journey is the colour mood of the unit's beat; nil when none is set.
	Journey *JourneyStep
	// Typography and CompositionRules are the locked visual concept's.
	Typography       Typography
	CompositionRules []string
	// BackgroundColor is the beat's background colour; TextColor the colour of
	// on-screen text. Both are hex codes, empty when the palette does not give one.
	BackgroundColor string
	TextColor       string
}

// paletteHex returns the colour of palette token, or "".
func (in StoryboardInput) paletteHex(token string) string {
	for _, s := range in.Seq.Concept.ColorSystem.Palette {
		if s.Token == token {
			return s.Hex
		}
	}
	return ""
}

// Unit returns the input of plan unit id; false when the plan has no such unit
// or the sequence has no item for it.
func (in StoryboardInput) Unit(unitID string) (UnitInput, bool) {
	u, ok := in.planUnit(unitID)
	if !ok {
		return UnitInput{}, false
	}
	item, ok := in.seqItem(unitID)
	if !ok {
		return UnitInput{}, false
	}
	beat, _ := in.Seq.Beat(u.BeatID)
	cs := in.Seq.Concept.ColorSystem
	out := UnitInput{
		Unit: u, Beat: beat, Item: item, Lines: in.unitLines(u.ScriptRefs),
		Typography:       in.Seq.Concept.RepresentationRules.Typography,
		CompositionRules: in.Seq.Concept.VisualLanguage.CompositionRules,
		BackgroundColor:  cs.Background,
		TextColor:        in.paletteHex(cs.Contrast.TextToken),
	}
	for i := range cs.Journey {
		if contains(cs.Journey[i].BeatKeys, u.BeatID) {
			step := cs.Journey[i]
			out.Journey = &step
			if hex := in.paletteHex(step.BackgroundToken); hex != "" {
				out.BackgroundColor = hex
			}
			break
		}
	}
	return out, true
}

// Frame size in pixels.
const (
	FrameWidth  = 1280
	FrameHeight = 720
)

// FrameSize is the size of a frame image.
type FrameSize struct {
	W int `json:"w"`
	H int `json:"h"`
}

// FrameBackground is what fills the frame behind the placements: a backdrop
// asset when the shot names one, else the colour.
type FrameBackground struct {
	AssetID string `json:"asset_id"`
	Color   string `json:"color"`
}

// FramePlacement puts one asset at a position of the frame. Role is subject or
// main. Version is the asset's version when the spec was built; it is not sent to
// rendering but is part of the spec hash.
type FramePlacement struct {
	AssetID  string `json:"asset_id"`
	Role     string `json:"role"`
	Position string `json:"position"`
	Version  int    `json:"-"`
}

// FrameText is one line of on-screen text and the smallest size it is drawn at.
type FrameText struct {
	Text  string `json:"text"`
	MinPx int    `json:"min_px"`
}

// Roles of a placement.
const (
	PlacementSubject = "subject"
	PlacementMain    = "main"
)

// FrameSpec is what visual-service sends rendering to draw one shot's frame.
type FrameSpec struct {
	ProjectID    string           `json:"project_id"`
	ShotID       string           `json:"shot_id"`
	Kit          json.RawMessage  `json:"kit"`
	Size         FrameSize        `json:"size"`
	Background   FrameBackground  `json:"background"`
	Placements   []FramePlacement `json:"placements"`
	CameraView   string           `json:"camera_view"`
	OnScreenText []FrameText      `json:"on_screen_text"`
	TextColor    string           `json:"text_color"`

	// BackgroundVersion is the version of the backdrop asset; part of the hash only.
	BackgroundVersion int `json:"-"`
}

// BuildFrameSpec builds the spec of one shot. A subject or main object that is an
// asset id of the shot's asset references becomes a placement; a free name is not
// drawn. The first backdrop among the references fills the background. kit is the
// reference kit of the project, sent as is. Rendering decides the size of each
// placement. It fails when a named asset is not in assets.
func BuildFrameSpec(projectID string, sh Shot, ui UnitInput, assets map[string]StoryboardAsset, kit json.RawMessage) (FrameSpec, error) {
	spec := FrameSpec{
		ProjectID: projectID, ShotID: sh.ShotID, Kit: kit, Size: FrameSize{W: FrameWidth, H: FrameHeight},
		Background: FrameBackground{Color: ui.BackgroundColor}, Placements: []FramePlacement{},
		CameraView: sh.Composition.CameraView, OnScreenText: []FrameText{}, TextColor: ui.TextColor,
	}
	for _, id := range sh.AssetRefs {
		a, ok := assets[id]
		if !ok {
			return FrameSpec{}, fmt.Errorf("tài sản %s không có", id)
		}
		if a.Kind == AssetKindBackdrop {
			spec.Background.AssetID, spec.BackgroundVersion = id, a.Version
			break
		}
	}
	for _, p := range []struct{ ref, pos, role string }{
		{sh.Composition.Subject, sh.Composition.SubjectPosition, PlacementSubject},
		{sh.Composition.MainObject, sh.Composition.MainObjectPosition, PlacementMain},
	} {
		if p.ref == "" || !contains(sh.AssetRefs, p.ref) {
			continue
		}
		a, ok := assets[p.ref]
		if !ok {
			return FrameSpec{}, fmt.Errorf("tài sản %s không có", p.ref)
		}
		if a.Kind == AssetKindBackdrop {
			continue
		}
		spec.Placements = append(spec.Placements, FramePlacement{AssetID: p.ref, Role: p.role, Position: p.pos, Version: a.Version})
	}
	for _, t := range sh.OnScreenText {
		spec.OnScreenText = append(spec.OnScreenText, FrameText{Text: t, MinPx: ui.Typography.Labels.MinPx})
	}
	return spec, nil
}

// SpecHash is the hash of what a frame is drawn from, over exactly what
// visual-service knows: the positions and camera view of the composition, the
// on-screen text (each line with its min_px) and its colour, each placed asset's id and version, the kit and the background
// (colour, backdrop asset and its version). Two shots with the same look share a
// hash. Rendering stores it as the frame's spec_hash when it draws.
func SpecHash(spec FrameSpec) string {
	type placed struct {
		AssetID, Role, Position string
		Version                 int
	}
	in := struct {
		Placements        []placed
		CameraView        string
		Text              []FrameText
		TextColor         string
		Kit               json.RawMessage
		BackgroundAsset   string
		BackgroundColor   string
		BackgroundVersion int
	}{CameraView: spec.CameraView, Kit: spec.Kit, BackgroundAsset: spec.Background.AssetID,
		BackgroundColor: spec.Background.Color, BackgroundVersion: spec.BackgroundVersion, TextColor: spec.TextColor}
	for _, p := range spec.Placements {
		in.Placements = append(in.Placements, placed{p.AssetID, p.Role, p.Position, p.Version})
	}
	for _, t := range spec.OnScreenText {
		in.Text = append(in.Text, t)
	}
	raw, _ := json.Marshal(in)
	return hashText(string(raw))
}

// DrawnFrame is what rendering recorded for a shot's last drawing: where the
// image is, the SpecHash it was drawn from (what staleness compares), the SHA-256
// of the PNG (informational only) and the last drawing error.
type DrawnFrame struct {
	FramePath string `json:"frame_path"`
	SpecHash  string `json:"spec_hash"`
	FrameHash string `json:"frame_hash"`
	Error     string `json:"error"`
}

// Drawn reports whether the shot has a frame image.
func (f DrawnFrame) Drawn() bool { return f.FramePath != "" }

// Stale reports whether the frame was drawn from another spec than specHash, the
// hash of the shot as it is now.
func (f DrawnFrame) Stale(specHash string) bool { return f.Drawn() && f.SpecHash != specHash }
