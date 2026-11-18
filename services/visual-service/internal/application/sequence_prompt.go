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

// beatUnitsText lists a beat's units with their plan fields and their lines.
func beatUnitsText(in domain.SequenceInput, b domain.SequenceBeat) string {
	text := map[string]string{}
	for _, l := range in.Script.Lines[b.Key] {
		text[l.ID] = l.Text
	}
	var sb strings.Builder
	for _, u := range b.Units {
		sb.WriteString(planUnitLine(u) + "\n")
		fmt.Fprintf(&sb, "  quan hệ với lời thoại: %s\n", u.NarrationRelationship)
		if len(u.OnScreenText) > 0 {
			fmt.Fprintf(&sb, "  chữ trên hình: %s\n", strings.Join(u.OnScreenText, " | "))
		}
		for _, id := range u.ScriptRefs {
			fmt.Fprintf(&sb, "  %s %s\n", id, text[id])
		}
	}
	return strings.TrimSpace(sb.String())
}

// prevUnitText is the last unit before beat at, with its item when written.
func prevUnitText(data *sequenceData, at int) string {
	if at == 0 {
		return "(không có, unit đầu tiên của beat là unit đầu video)"
	}
	prev := data.in.Beats[at-1]
	u := prev.Units[len(prev.Units)-1]
	line := planUnitLine(u)
	seq := data.beats[prev.Key]
	if i := seq.Index(u.UnitID); i >= 0 {
		raw, _ := json.Marshal(seq.Items[i])
		return line + "\n  chuỗi: " + string(raw)
	}
	return line + "\n  (chưa có chuỗi)"
}

// nextBeatText is the beat after beat at, with its first unit.
func nextBeatText(in domain.SequenceInput, at int) string {
	if at+1 >= len(in.Beats) {
		return none
	}
	b := in.Beats[at+1]
	return beatLine(b.Beat) + "\n  unit đầu: " + planUnitLine(b.Units[0])
}

// seqBeatVars are the variables of the prompt that writes beat at's sequence.
func seqBeatVars(data *sequenceData, at int, creatorFields []string, note string) map[string]string {
	in := data.in
	b := in.Beats[at]
	vars := sequenceSharedVars(in)
	vars["motifs"] = motifsText(data.motifs)
	vars["beat"] = beatLine(b.Beat)
	vars["beat_units"] = beatUnitsText(in, b)
	vars["handoff"] = handoffOfBeat(in.Script, b.Beat)
	vars["prev_unit"] = prevUnitText(data, at)
	vars["next_beat"] = nextBeatText(in, at)
	vars["current"], vars["creator_fields"] = "(chưa có)", none
	if cur := data.beats[b.Key]; len(cur.Items) > 0 {
		raw, _ := cur.Encode()
		vars["current"] = string(raw)
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
