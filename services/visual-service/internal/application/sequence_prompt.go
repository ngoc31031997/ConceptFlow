package application

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"visual/internal/domain"
)

// The variables the prompts of step 10 fill.

// sequenceSharedVars are the variables every prompt of step 10 uses.
func sequenceSharedVars(in domain.SequenceInput) map[string]string {
	return map[string]string{
		"brief":           briefText(in.Script.Brief),
		"pace":            paceText(in),
		"motion_language": motionLanguageText(in.Concept.MotionLanguage),
	}
}

// paceText is the locked motion speed of each beat, as "b1, b2 · pace".
func paceText(in domain.SequenceInput) string {
	var lines []string
	for _, p := range in.Concept.MotionLanguage.Pace {
		lines = append(lines, strings.Join(p.BeatKeys, ", ")+" · "+p.Pace)
	}
	if len(lines) == 0 {
		return none
	}
	return strings.Join(lines, "\n")
}

// motionLanguageText is the locked motion language without its pace, which
// has its own block.
func motionLanguageText(m domain.MotionLanguage) string {
	var sb strings.Builder
	list := func(label string, items []string) {
		if len(items) > 0 {
			fmt.Fprintf(&sb, "%s:\n- %s\n", label, strings.Join(items, "\n- "))
		}
	}
	list("Nguyên tắc", m.Principles)
	list("Chuyển cảnh ưa dùng", m.PreferredTransitions)
	list("Tránh", m.Avoid)
	if strings.TrimSpace(m.CharacterMotion) != "" {
		fmt.Fprintf(&sb, "Chuyển động nhân vật: %s\n", m.CharacterMotion)
	}
	list("Ví dụ", m.Examples)
	return orNone(strings.TrimSpace(sb.String()))
}

// anchorsText lists the representation rules a motif can anchor to, in path order.
func anchorsText(anchors map[string]string) string {
	return conceptRulesText(anchors)
}

// seqPlanText is the locked plan in order, one unit a line.
func seqPlanText(in domain.SequenceInput) string {
	var lines []string
	for _, u := range in.Plan.Units {
		lines = append(lines, planUnitLine(u))
	}
	return orNone(strings.Join(lines, "\n"))
}

func planUnitLine(u domain.DocumentUnit) string {
	refs := strings.Join(u.ScriptRefs, ", ")
	return fmt.Sprintf("%s · %s · câu %s · %.1fs · %s · %s: %s", u.UnitID, u.BeatID, refs, u.Duration.PlannedS,
		u.CommunicationGoal, u.VisualApproach.Method, u.VisualApproach.Description)
}

// motifsText lists the video's motifs.
func motifsText(set domain.MotifSet) string {
	var lines []string
	for _, m := range set.Motifs {
		planned := "(không)"
		if len(m.PlannedUnits) > 0 {
			planned = strings.Join(m.PlannedUnits, ", ")
		}
		lines = append(lines, fmt.Sprintf("%s · %s · %s · neo %s · %s · dự định %s", m.MotifID, m.Name, m.Description, m.AnchorRef, m.Role, planned))
	}
	return orNone(strings.Join(lines, "\n"))
}

// creatorFieldsText lists the Creator's fields, or none.
func creatorFieldsText(fields []string) string {
	if len(fields) == 0 {
		return none
	}
	f := append([]string{}, fields...)
	sort.Strings(f)
	return strings.Join(f, ", ")
}

// motifsVars are the variables of the prompt that sets the motifs.
func motifsVars(data *sequenceData, note string) map[string]string {
	vars := sequenceSharedVars(data.in)
	vars["anchors"] = anchorsText(data.in.AnchorRefs)
	vars["plan"] = seqPlanText(data.in)
	vars["current_motifs"], vars["creator_fields"] = "(chưa có)", none
	if len(data.motifs.Motifs) > 0 {
		raw, _ := json.Marshal(data.motifs.Motifs)
		vars["current_motifs"] = string(raw)
		vars["creator_fields"] = creatorFieldsText(data.motifRow.CreatorFields)
	}
	vars["note"] = orNone(note)
	return vars
}

// unitDetailText is one plan unit with its plan fields and the lines it covers.
func unitDetailText(in domain.SequenceInput, u domain.DocumentUnit) string {
	text := map[string]string{}
	for _, l := range in.Script.Lines[u.BeatID] {
		text[l.ID] = l.Text
	}
	var sb strings.Builder
	sb.WriteString(planUnitLine(u) + "\n")
	fmt.Fprintf(&sb, "  quan hệ với lời thoại: %s\n", u.NarrationRelationship)
	if len(u.OnScreenText) > 0 {
		fmt.Fprintf(&sb, "  chữ trên hình: %s\n", strings.Join(u.OnScreenText, " | "))
	}
	for _, id := range u.ScriptRefs {
		fmt.Fprintf(&sb, "  %s %s\n", id, text[id])
	}
	return strings.TrimSpace(sb.String())
}

// itemText is a unit's stored item without its status, or that it has none.
func (data *sequenceData) itemText(beatKey, unitID string) string {
	seq := data.beats[beatKey]
	i := seq.Index(unitID)
	if i < 0 {
		return "(chưa có chuỗi)"
	}
	it := seq.Items[i]
	it.Status = ""
	raw, _ := json.Marshal(it)
	return string(raw)
}

// neighbourUnitText is the plan unit at position at of the video with its item,
// or none when there is no unit there.
func neighbourUnitText(data *sequenceData, at int, missing string) string {
	units := data.in.Plan.Units
	if at < 0 || at >= len(units) {
		return missing
	}
	u := units[at]
	return planUnitLine(u) + "\n  chuỗi: " + data.itemText(u.BeatID, u.UnitID)
}

// beatItemsText lists the units of a beat with their items, marking the one
// being written.
func beatItemsText(data *sequenceData, b domain.SequenceBeat, writing string) string {
	var lines []string
	for _, u := range b.Units {
		if u.UnitID == writing {
			lines = append(lines, planUnitLine(u)+"\n  (unit đang viết)")
			continue
		}
		lines = append(lines, planUnitLine(u)+"\n  chuỗi: "+data.itemText(b.Key, u.UnitID))
	}
	return strings.Join(lines, "\n")
}

// unitPosition is where plan unit id stands in the video, or -1.
func unitPosition(in domain.SequenceInput, id string) int {
	for i, u := range in.Plan.Units {
		if u.UnitID == id {
			return i
		}
	}
	return -1
}

// seqUnitVars are the variables of the prompt that writes one unit's item.
// creatorFields are the fields the Creator wrote in that item.
func seqUnitVars(data *sequenceData, b domain.SequenceBeat, unitID string, creatorFields []string, note string) map[string]string {
	in := data.in
	at := unitPosition(in, unitID)
	vars := sequenceSharedVars(in)
	vars["motifs"] = motifsText(data.motifs)
	vars["beat"] = beatLine(b.Beat)
	vars["handoff"] = handoffOfBeat(in.Script, b.Beat)
	vars["beat_items"] = beatItemsText(data, b, unitID)
	vars["prev_unit"] = neighbourUnitText(data, at-1, "(không có: unit đang viết là unit đầu video)")
	vars["unit"] = unitDetailText(in, in.Plan.Units[at])
	vars["next_unit"] = neighbourUnitText(data, at+1, "(không có: unit đang viết là unit cuối video)")
	vars["current"], vars["creator_fields"] = "(chưa có)", none
	if data.beats[b.Key].Index(unitID) >= 0 {
		vars["current"] = data.itemText(b.Key, unitID)
		vars["creator_fields"] = creatorFieldsText(creatorFields)
	}
	vars["note"] = orNone(note)
	return vars
}

// sequenceText is the whole sequence in plan order as the reviewer reads it.
func sequenceText(data *sequenceData) string {
	var sb strings.Builder
	for _, b := range data.in.Beats {
		seq := data.beats[b.Key]
		for _, u := range b.Units {
			i := seq.Index(u.UnitID)
			if i < 0 {
				fmt.Fprintf(&sb, "%s · %s · (chưa có chuỗi)\n", u.UnitID, b.Key)
				continue
			}
			it := seq.Items[i]
			it.Status = ""
			raw, _ := json.Marshal(it)
			fmt.Fprintf(&sb, "%s · %s · %s\n", u.UnitID, b.Key, raw)
		}
	}
	return strings.TrimSpace(sb.String())
}

// sequenceChecksText lists what the machine already checked in the sequence.
func sequenceChecksText(data *sequenceData) string {
	in := data.in
	var lines []string
	cov := domain.ComputeSequenceCoverage(in, data.beats)
	state := "đạt"
	if !cov.Complete() {
		state = "chưa đạt"
	}
	lines = append(lines, fmt.Sprintf("- [%s] Mỗi unit của kế hoạch hình có đúng một item", state))
	for _, c := range domain.SequenceChecks(in, data.beats, data.motifs) {
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
	for _, m := range data.motifs.Motifs {
		if p := domain.CheckMotif(m, in.AnchorRefs); len(p) > 0 {
			lines = append(lines, fmt.Sprintf("- [chưa đạt] %s: %s", m.MotifID, strings.Join(p, "; ")))
		}
	}
	for _, b := range in.Beats {
		for _, it := range data.beats[b.Key].Items {
			if p := data.itemProblems(it.UnitID, it); len(p) > 0 {
				lines = append(lines, fmt.Sprintf("- [chưa đạt] %s: %s", it.UnitID, strings.Join(p, "; ")))
			}
		}
	}
	return strings.Join(lines, "\n")
}
