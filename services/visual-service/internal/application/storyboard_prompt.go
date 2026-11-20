package application

import (
	"encoding/json"
	"fmt"
	"strings"

	"visual/internal/domain"
)

// The variables the prompts of step 11 fill.

// representationText lists the locked concept entries the writers may point an
// asset at, one a line: id, kind, term and how it is drawn.
func representationText(rr domain.RepresentationRules) string {
	var lines []string
	thing := func(kind string, t domain.RepresentedThing) {
		lines = append(lines, fmt.Sprintf("%s · %s · %s · cách vẽ: %s", t.ConceptID, kind, t.Term, t.Draw))
	}
	for _, t := range rr.Characters.Items {
		thing("nhân vật", t)
	}
	for _, t := range rr.Objects {
		thing("vật thể", t)
	}
	for _, t := range rr.Invisible {
		thing("điều vô hình", t)
	}
	for _, c := range rr.ScientificConcepts {
		lines = append(lines, fmt.Sprintf("%s · khái niệm · %s · cách vẽ: %s", c.ConceptID, c.Term, c.Representation))
	}
	return orNone(strings.Join(lines, "\n"))
}

// conceptEntryText is the locked concept entry conceptID names, in full: term,
// how it is drawn, what it must not imply and its facts. A backdrop with no entry
// says so.
func conceptEntryText(rr domain.RepresentationRules, conceptID string) string {
	entry := func(kind, term, draw string, mustNot, facts []string) string {
		return fmt.Sprintf("%s · %s · %s\ncách vẽ: %s\nkhông được gợi ý: %s\ndữ kiện: %s", conceptID, kind, term, draw,
			orNone(strings.Join(mustNot, "; ")), orNone(strings.Join(facts, "; ")))
	}
	thing := func(kind string, items []domain.RepresentedThing) (string, bool) {
		for _, t := range items {
			if t.ConceptID == conceptID {
				return entry(kind, t.Term, t.Draw, t.MustNotImply, t.Facts), true
			}
		}
		return "", false
	}
	if conceptID == "" {
		return "(tài sản này không gắn mục khái niệm nào)"
	}
	for _, g := range []struct {
		kind  string
		items []domain.RepresentedThing
	}{{"nhân vật", rr.Characters.Items}, {"vật thể", rr.Objects}, {"điều vô hình", rr.Invisible}} {
		if text, ok := thing(g.kind, g.items); ok {
			return text
		}
	}
	for _, c := range rr.ScientificConcepts {
		if c.ConceptID == conceptID {
			return entry("khái niệm", c.Term, c.Representation, c.MustNotImply, c.Facts)
		}
	}
	return "(không có mục khái niệm " + conceptID + ")"
}

// paletteText lists the locked palette: token, role, name, colour code.
func paletteText(cs domain.ColorSystem) string {
	var lines []string
	for _, s := range cs.Palette {
		lines = append(lines, fmt.Sprintf("%s · %s · %s · %s", s.Token, s.Role, s.Name, s.Hex))
	}
	return orNone(strings.Join(lines, "\n"))
}

// storyboardPlanText is the locked plan in order: unit, beat, what the picture
// must carry and how it is shown.
func storyboardPlanText(in domain.StoryboardInput) string {
	var lines []string
	for _, u := range in.Seq.Plan.Units {
		lines = append(lines, planUnitBriefLine(u))
	}
	return orNone(strings.Join(lines, "\n"))
}

// planUnitBriefLine is one plan unit: id, beat, what the picture must carry and
// how it is shown.
func planUnitBriefLine(u domain.DocumentUnit) string {
	var info []string
	for _, e := range u.EssentialVisualInfo {
		info = append(info, e.Item)
	}
	return fmt.Sprintf("%s · %s · thông tin hình bắt buộc: %s · cách thể hiện: %s: %s", u.UnitID, u.BeatID,
		orNone(strings.Join(info, "; ")), u.VisualApproach.Method, u.VisualApproach.Description)
}

// documentMotifsText lists the motifs of the locked sequence: id and name.
func documentMotifsText(motifs []domain.DocumentMotif) string {
	var lines []string
	for _, m := range motifs {
		lines = append(lines, fmt.Sprintf("%s · %s", m.MotifID, m.Name))
	}
	return orNone(strings.Join(lines, "\n"))
}

// libraryText lists the approved library drawings an asset may take as its base:
// id, name, kind and tags.
func libraryText(drawings []LibraryDrawing) string {
	var lines []string
	for _, l := range drawings {
		lines = append(lines, fmt.Sprintf("%s · %s · %s · thẻ: %s", l.ID, l.Name, l.Kind, orNone(strings.Join(l.Tags, ", "))))
	}
	return orNone(strings.Join(lines, "\n"))
}

// assetFrameText is the frame of an asset as the list writer revises it: id,
// concept, name, kind, tier and the units it appears in.
func assetFrameText(a domain.StoryboardAsset) string {
	return fmt.Sprintf("%s · khái niệm %s · %s · %s · %s · unit: %s", a.AssetID, orNone(a.ConceptID), a.Name, a.Kind, a.Tier,
		orNone(strings.Join(a.Units, ", ")))
}

// currentAssetsText lists the video's assets as the list writer revises them.
func currentAssetsText(assets []ProjectAsset) string {
	if len(assets) == 0 {
		return "(chưa có)"
	}
	var lines []string
	for _, a := range assets {
		lines = append(lines, assetFrameText(a.StoryboardAsset))
	}
	return strings.Join(lines, "\n")
}

// assetsLinesText lists assets one a line: id, name, kind, tier, portrayal,
// what they must not imply, colours and expressions.
func assetsLinesText(assets []domain.StoryboardAsset) string {
	var lines []string
	for _, a := range assets {
		lines = append(lines, fmt.Sprintf("%s · %s · %s · %s · %s · không được gợi ý: %s · màu: %s · biểu cảm: %s", a.AssetID, a.Name, a.Kind, a.Tier,
			a.Portrayal, orNone(strings.Join(a.MustNotImply, "; ")), orNone(strings.Join(a.PaletteTokens, ", ")), orNone(strings.Join(a.Expressions, "; "))))
	}
	return orNone(strings.Join(lines, "\n"))
}

// assetsVars are the variables of the prompt that lists the video's assets.
func assetsVars(data *storyboardData, library []LibraryDrawing, note string) map[string]string {
	in := data.in
	return map[string]string{
		"brief":          briefText(in.Seq.Script.Brief),
		"representation": representationText(in.Seq.Concept.RepresentationRules.RepresentationRules),
		"plan":           storyboardPlanText(in),
		"motifs":         documentMotifsText(in.Sequence.Motifs),
		"library":        libraryText(library),
		"current_assets": currentAssetsText(data.assets),
		"creator_fields": creatorFieldsText(data.listRow.CreatorFields),
		"note":           orNone(note),
	}
}

// assetDetailJSON is the detail an asset has now, as the detail writer revises it.
type assetDetailJSON struct {
	Portrayal     string   `json:"portrayal"`
	MustNotImply  []string `json:"must_not_imply"`
	PaletteTokens []string `json:"palette_tokens"`
	Expressions   []string `json:"expressions"`
}

// currentDetailText is the detail the asset has, or that it has none.
func currentDetailText(a domain.StoryboardAsset) string {
	if !a.HasDetail() {
		return "(chưa có)"
	}
	raw, _ := json.Marshal(assetDetailJSON{Portrayal: a.Portrayal, MustNotImply: nonNilList(a.MustNotImply),
		PaletteTokens: nonNilList(a.PaletteTokens), Expressions: nonNilList(a.Expressions)})
	return string(raw)
}

// otherAssetsText lists the video's other assets: name, tier and, when written,
// the description, so a detail does not repeat another asset's colours or role.
func otherAssetsText(assets []ProjectAsset, exceptID string) string {
	var lines []string
	for _, a := range assets {
		if a.AssetID == exceptID {
			continue
		}
		line := fmt.Sprintf("%s · %s", a.Name, a.Tier)
		if a.HasDetail() {
			line += " · " + a.Portrayal
			if len(a.PaletteTokens) > 0 {
				line += " · màu: " + strings.Join(a.PaletteTokens, ", ")
			}
		}
		lines = append(lines, line)
	}
	return orNone(strings.Join(lines, "\n"))
}

// assetUnitsText lists the plan units an asset appears in, each with its plan
// line and its script lines. A unit the plan does not have is left out.
func assetUnitsText(in domain.StoryboardInput, unitIDs []string) string {
	var blocks []string
	for _, id := range unitIDs {
		ui, ok := in.Unit(id)
		if !ok {
			continue
		}
		var sb strings.Builder
		sb.WriteString(planUnitBriefLine(ui.Unit))
		for _, l := range ui.Lines {
			fmt.Fprintf(&sb, "\n  %s %s", l.ID, l.Text)
		}
		blocks = append(blocks, sb.String())
	}
	return orNone(strings.Join(blocks, "\n"))
}

// assetDetailVars are the variables of the prompt that writes one asset's detail.
func assetDetailVars(data *storyboardData, a ProjectAsset, creatorFields []string, note string) map[string]string {
	in := data.in
	base := a.BaseIllustrationID
	return map[string]string{
		"brief":          briefText(in.Seq.Script.Brief),
		"asset":          fmt.Sprintf("%s · khái niệm %s · %s · %s · %s · hình gốc: %s", a.AssetID, orNone(a.ConceptID), a.Name, a.Kind, a.Tier, orNone(base)),
		"concept":        conceptEntryText(in.Seq.Concept.RepresentationRules.RepresentationRules, a.ConceptID),
		"palette":        paletteText(in.Seq.Concept.ColorSystem),
		"units":          assetUnitsText(in, a.Units),
		"other_assets":   otherAssetsText(data.assets, a.AssetID),
		"current":        currentDetailText(a.StoryboardAsset),
		"creator_fields": creatorFieldsText(creatorFields),
		"note":           orNone(note),
	}
}

// typographyText is the locked rules for on-screen text.
func typographyText(t domain.Typography) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Số từ tối đa một dòng: %d\n", t.Labels.MaxWords)
	fmt.Fprintf(&sb, "Cỡ chữ nhỏ nhất: %d px\n", t.Labels.MinPx)
	if strings.TrimSpace(t.Labels.When) != "" {
		fmt.Fprintf(&sb, "Khi nào dùng chữ: %s\n", t.Labels.When)
	}
	if strings.TrimSpace(t.LabelStyle) != "" {
		fmt.Fprintf(&sb, "Kiểu nhãn: %s\n", t.LabelStyle)
	}
	return strings.TrimSpace(sb.String())
}

// journeyText is the colour mood of the unit's beat.
func journeyText(ui domain.UnitInput) string {
	if ui.Journey == nil {
		return "Màu nền: " + orNone(ui.BackgroundColor)
	}
	return fmt.Sprintf("Tâm trạng: %s · màu nền %s (%s) · token nhấn: %s", ui.Journey.Mood, orNone(ui.BackgroundColor),
		orNone(ui.Journey.BackgroundToken), orNone(ui.Journey.AccentToken))
}

// shotsJSONText is shots without their status, one a line, or none.
func shotsJSONText(shots []domain.Shot, missing string) string {
	if len(shots) == 0 {
		return missing
	}
	var lines []string
	for _, sh := range shots {
		sh.Status = ""
		raw, _ := json.Marshal(sh)
		lines = append(lines, string(raw))
	}
	return strings.Join(lines, "\n")
}

// positionsText lists the positions the writer may use; split_screen is the
// way to ask for a split frame.
func positionsText() string {
	return strings.Join(domain.ShotPositions, ", ") + ", " + domain.PositionSplitScreen +
		" (khung chia đôi: cần đúng hai đối tượng, đối tượng chính ở nửa trái và vật chính ở nửa phải)"
}

// unitVars are the variables of the prompt that writes one unit's shots.
func unitVars(data *storyboardData, unitID string, prev []domain.Shot, hasPrev bool, creatorFields []string, note string) (map[string]string, error) {
	in := data.in
	ui, ok := in.Unit(unitID)
	if !ok {
		return nil, domain.ErrUnknownPlanUnit
	}
	item := ui.Item
	item.Status = ""
	rawItem, _ := json.Marshal(item)
	var approved []domain.StoryboardAsset
	for _, a := range data.assets {
		if a.Status == domain.AssetApproved {
			approved = append(approved, a.StoryboardAsset)
		}
	}
	prevMissing := "(không có: unit đang viết là unit đầu video)"
	if hasPrev {
		prevMissing = "(unit liền trước chưa có shot)"
	}
	cur := data.units[unitID]
	vars := map[string]string{
		"brief":             briefText(in.Seq.Script.Brief),
		"composition_rules": orNone(strings.Join(ui.CompositionRules, "\n")),
		"typography":        typographyText(ui.Typography),
		"journey":           journeyText(ui),
		"beat":              beatLine(ui.Beat.Beat),
		"unit":              unitDetailText(in.Seq, ui.Unit),
		"sequence_item":     string(rawItem),
		"assets":            assetsLinesText(approved),
		"positions":         positionsText(),
		"camera_views":      strings.Join(domain.ShotCameraViews, ", "),
		"prev_shots":        shotsJSONText(prev, prevMissing),
		"current_shots":     shotsJSONText(cur.Shots, "(chưa có)"),
		"creator_fields":    none,
		"note":              orNone(note),
	}
	if len(cur.Shots) > 0 {
		vars["creator_fields"] = creatorFieldsText(creatorFields)
	}
	return vars, nil
}

// shotLines is every shot in viewing order for the reviewer: the shot with its
// unit, beat and frame state.
func shotLines(data *storyboardData, st domain.StoryboardState) string {
	var lines []string
	for _, u := range data.in.Seq.Plan.Units {
		for _, sh := range data.units[u.UnitID].Shots {
			frame := "chưa có ảnh khung"
			if fr := st.Frames[sh.ShotID]; fr.Drawn() {
				frame = "có ảnh khung"
				if hash, err := st.SpecHashOf(data.in, u.UnitID, sh); err == nil && fr.Stale(hash) {
					frame = "ảnh khung cũ"
				}
			}
			sh.Status = ""
			raw, _ := json.Marshal(sh)
			lines = append(lines, fmt.Sprintf("%s · %s · %s · %s · %s", sh.ShotID, u.UnitID, u.BeatID, frame, raw))
		}
	}
	return orNone(strings.Join(lines, "\n"))
}

// storyboardSequenceText is the locked sequence in order, one item a line.
func storyboardSequenceText(in domain.StoryboardInput) string {
	var lines []string
	for _, it := range in.Sequence.Sequence {
		it.Status = ""
		raw, _ := json.Marshal(it)
		lines = append(lines, string(raw))
	}
	return orNone(strings.Join(lines, "\n"))
}

// storyboardChecksText lists what the machine already checked.
func storyboardChecksText(checks []domain.MachineCheck) string {
	var lines []string
	for _, c := range checks {
		state := "đạt"
		if !c.OK {
			state = "chưa đạt"
		}
		line := fmt.Sprintf("- [%s] %s", state, c.Label)
		if c.Detail != "" {
			line += ": " + c.Detail
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}
