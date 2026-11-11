package application

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"visual/internal/domain"
)

// The variables the prompts of step 9 fill.

// planSharedVars are the variables both prompts use.
func planSharedVars(in domain.PlanInput) map[string]string {
	return map[string]string{
		"brief":         briefText(in.Script.Brief),
		"facts":         factsText(in.Script),
		"glossary":      glossaryText(in.Script),
		"concept_rules": conceptRulesText(in.RuleRefs),
		"do_not_use":    doNotUseText(in.Concept.DoNotUse),
	}
}

// conceptRulesText lists every rule of the locked visual concept a unit can
// point at, as "id · text", in id order.
func conceptRulesText(rules map[string]string) string {
	if len(rules) == 0 {
		return none
	}
	ids := make([]string, 0, len(rules))
	for id := range rules {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	lines := make([]string, 0, len(ids))
	for _, id := range ids {
		lines = append(lines, id+" · "+rules[id])
	}
	return strings.Join(lines, "\n")
}

func doNotUseText(items []domain.Forbidden) string {
	if len(items) == 0 {
		return none
	}
	lines := make([]string, 0, len(items))
	for _, f := range items {
		lines = append(lines, "- "+f.Text+" (vì sao: "+f.Why+")")
	}
	return strings.Join(lines, "\n")
}

// beatScriptText lists a beat's lines with the fewest seconds each lasts.
func beatScriptText(in domain.PlanInput, beatKey string) string {
	var lines []string
	for _, l := range in.Script.Lines[beatKey] {
		facts := ""
		if len(l.Facts) > 0 {
			facts = " [" + strings.Join(l.Facts, ", ") + "]"
		}
		sec := float64(len(strings.Fields(l.Text))) * in.SecondsPerWord
		lines = append(lines, fmt.Sprintf("%s %s%s · tối thiểu %.2f giây", l.ID, l.Text, facts, sec))
	}
	return strings.Join(lines, "\n")
}

// beatMinTotal is the fewest seconds the whole beat's lines last.
func beatMinTotal(in domain.PlanInput, beatKey string) float64 {
	words := 0
	for _, l := range in.Script.Lines[beatKey] {
		words += len(strings.Fields(l.Text))
	}
	return domain.NarrationMin(words, in.SecondsPerWord)
}

// neighbourText describes a beat next to the one being written: its card and
// the units it has, so the new units join up with them.
func neighbourText(in domain.PlanInput, b domain.Beat, ok bool, plan domain.BeatPlan) string {
	if !ok {
		return none
	}
	var sb strings.Builder
	sb.WriteString(beatLine(b))
	if len(plan.Units) == 0 {
		sb.WriteString("\n  (chưa có kế hoạch hình)")
	}
	for _, u := range plan.Units {
		fmt.Fprintf(&sb, "\n  %s · %s · %s · %s", u.UnitID, strings.Join(u.ScriptRefs, ", "), u.VisualApproach.Method, u.NarrationSummary)
	}
	return sb.String()
}

// beatVars are the variables of the prompt that writes the units of beat key.
func beatVars(in domain.PlanInput, plans map[string]domain.BeatPlan, key string, current domain.BeatPlan, creatorFields []string, note string) map[string]string {
	vars := planSharedVars(in)
	at := -1
	for i, b := range in.Script.Beats {
		if b.Key == key {
			at = i
		}
	}
	b := in.Script.Beats[at]
	vars["beat"] = beatLine(b)
	vars["script"] = beatScriptText(in, key)
	vars["beat_min_total"] = fmt.Sprintf("%.1f", beatMinTotal(in, key))
	vars["handoff"] = handoffOfBeat(in.Script, b)
	var prev, next domain.Beat
	hasPrev, hasNext := at > 0, at+1 < len(in.Script.Beats)
	if hasPrev {
		prev = in.Script.Beats[at-1]
	}
	if hasNext {
		next = in.Script.Beats[at+1]
	}
	vars["prev_beat"] = neighbourText(in, prev, hasPrev, plans[prev.Key])
	vars["next_beat"] = neighbourText(in, next, hasNext, plans[next.Key])
	vars["current"], vars["creator_fields"] = "(chưa có)", none
	if len(current.Units) > 0 {
		raw, _ := current.Encode()
		vars["current"] = string(raw)
		if len(creatorFields) > 0 {
			vars["creator_fields"] = strings.Join(creatorFields, ", ")
		}
	}
	vars["note"] = orNone(note)
	return vars
}

// planText is the whole plan as the reviewer reads it: each beat with its
// timing, each unit with its lines and content.
func planText(in domain.PlanInput, plans map[string]domain.BeatPlan, timing domain.PlanTiming) string {
	var sb strings.Builder
	for _, b := range in.Script.Beats {
		t := timing.Beats[b.Key]
		fmt.Fprintf(&sb, "%s · B%d · %s · beat sheet %d giây · kế hoạch %.1f giây\n", b.Key, b.Number, b.Title, b.Dur, t.PlannedS)
		units := plans[b.Key].Units
		if len(units) == 0 {
			sb.WriteString("  (chưa có kế hoạch hình)\n")
			continue
		}
		pc := in.PlanContextFor(b.Key)
		for _, u := range units {
			fmt.Fprintf(&sb, "  %s · tối thiểu %.1f giây\n", u.UnitID, pc.UnitNarrationMin(u))
			for _, l := range linesOf(u, pc.Lines) {
				fmt.Fprintf(&sb, "    %s %s\n", l.ID, l.Text)
			}
			raw, _ := json.Marshal(u)
			fmt.Fprintf(&sb, "    %s\n", raw)
		}
	}
	return strings.TrimSpace(sb.String())
}

// planChecksText lists what the machine already checked in the plan.
func planChecksText(in domain.PlanInput, plans map[string]domain.BeatPlan) string {
	var lines []string
	cov := domain.ComputeCoverage(in, plans)
	state := "đạt"
	if !cov.Complete() {
		state = "chưa đạt"
	}
	lines = append(lines, fmt.Sprintf("- [%s] Mỗi câu lời thoại thuộc đúng một unit: phủ %d/%d câu", state, cov.LinesCovered, cov.LinesTotal))
	for _, c := range domain.PlanChecks(in, plans) {
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
	for _, b := range in.Script.Beats {
		for _, u := range plans[b.Key].Units {
			if p := domain.CheckPlanUnit(u, in.PlanContextFor(b.Key)); len(p) > 0 {
				lines = append(lines, fmt.Sprintf("- [chưa đạt] %s: %s", u.UnitID, strings.Join(p, "; ")))
			}
		}
	}
	return strings.Join(lines, "\n")
}

// planPreviousFindings lists the findings of the earlier rounds that still stand.
func planPreviousFindings(r domain.ReviewState) string {
	var prev []string
	for _, f := range r.Findings {
		if f.State != domain.FindingFixed {
			prev = append(prev, fmt.Sprintf("%s · %s · %s: %s", f.ID, f.Group, f.UnitKey, f.Text))
		}
	}
	if len(prev) == 0 {
		return "(lần rà soát đầu)"
	}
	return strings.Join(prev, "\n")
}
