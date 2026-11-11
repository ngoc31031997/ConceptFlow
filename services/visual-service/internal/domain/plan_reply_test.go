package domain

import (
	"errors"
	"strings"
	"testing"
)

var replyLines = []Line{{ID: "b1.1", Text: "một"}, {ID: "b1.2", Text: "hai"}, {ID: "b1.3", Text: "ba"}}

func replyUnit(refs string) string {
	return `{"script_refs":[` + refs + `],"narration_summary":"s","communication_goal":"g",
	"essential_visual_info":[{"item":"i","priority":"must"}],"on_screen_text":[],
	"visual_approach":{"method":"diagram","description":"d","concept_rule_refs":[]},
	"narration_relationship":"contrast","duration":{"planned_s":0,"extra_reason":""},"risk_notes":[],"open_questions":[]}`
}

func replyOf(units ...string) []byte { return []byte(`{"units":[` + strings.Join(units, ",") + `]}`) }

func TestParsePlanBeatReply_AcceptsUnitsThatCoverTheBeatOnceInOrder(t *testing.T) {
	units, err := ParsePlanBeatReply(replyOf(replyUnit(`"b1.1"`), replyUnit(`"b1.2","b1.3"`)), replyLines)
	if err != nil || len(units) != 2 {
		t.Fatalf("a reply that partitions the lines must pass: %v", err)
	}
	if units[0].UnitID != "" || units[0].Status != UnitPending || units[1].ScriptRefs[1] != "b1.3" || units[0].VisualApproach.Method != "diagram" {
		t.Errorf("units must come back without ids and pending: %+v", units)
	}
}

func TestParsePlanBeatReply_RefusesWhatCannotBeStored(t *testing.T) {
	good := replyUnit(`"b1.1","b1.2","b1.3"`)
	for name, c := range map[string]struct {
		raw  []byte
		want string
	}{
		"not json":               {[]byte(`ok rồi`), ""},
		"no unit":                {replyOf(), "không có unit"},
		"an unknown key":         {[]byte(strings.Replace(string(replyOf(good)), `"script_refs"`, `"unit_id":"VP-001","script_refs"`, 1)), "unit_id"},
		"a machine's key":        {[]byte(strings.Replace(string(replyOf(good)), `"script_refs"`, `"fact_refs":[],"script_refs"`, 1)), "fact_refs"},
		"a wrong type":           {[]byte(strings.Replace(string(replyOf(good)), `"planned_s":0`, `"planned_s":"dài"`, 1)), ""},
		"a strange method":       {[]byte(strings.Replace(string(replyOf(good)), `"diagram"`, `"hologram"`, 1)), "hologram"},
		"a strange relation":     {[]byte(strings.Replace(string(replyOf(good)), `"contrast"`, `"echo"`, 1)), "echo"},
		"a strange priority":     {[]byte(strings.Replace(string(replyOf(good)), `"must"`, `"urgent"`, 1)), "urgent"},
		"a unit with no line":    {replyOf(replyUnit(``)), "không có câu"},
		"a line left out":        {replyOf(replyUnit(`"b1.1","b1.2"`)), "phủ 2 câu"},
		"a line twice":           {replyOf(replyUnit(`"b1.1","b1.2"`), replyUnit(`"b1.2","b1.3"`)), "phủ 4 câu"},
		"lines out of order":     {replyOf(replyUnit(`"b1.1","b1.3","b1.2"`)), "đúng thứ tự"},
		"a line of another beat": {replyOf(replyUnit(`"b1.1","b1.2","b2.1"`)), "b2.1"},
	} {
		_, err := ParsePlanBeatReply(c.raw, replyLines)
		if !errors.Is(err, ErrUnreadableReply) || (c.want != "" && !strings.Contains(err.Error(), c.want)) {
			t.Errorf("%s: want an unreadable reply mentioning %q, got %v", name, c.want, err)
		}
	}
}

func unitOf(id, summary string, refs ...string) PlanUnit {
	u := PlanUnit{UnitID: id, ScriptRefs: refs, NarrationSummary: summary, Status: UnitApproved}
	u.normalize()
	return u
}

func TestMergeBeatReply_KeepsTheCreatorsUnitAndFieldsAndGivesNewIdsInOrder(t *testing.T) {
	old := BeatPlan{Units: []PlanUnit{unitOf("VP-007", "của Creator", "b1.2", "b1.3"), unitOf("VP-002", "cũ của AI", "b1.1")}}
	fresh := []PlanUnit{unitOf("", "mới", "b1.1"), unitOf("", "mới", "b1.2", "b1.3")}
	fresh[1].CommunicationGoal = "mục tiêu của AI"
	old.Units[0].CommunicationGoal = "mục tiêu cũ"
	m, err := MergeBeatReply(fresh, old, []string{"VP-007.narration_summary"})
	if err != nil {
		t.Fatal(err)
	}
	if got := m.Units[1]; got.UnitID != "VP-007" || got.NarrationSummary != "của Creator" || got.CommunicationGoal != "mục tiêu của AI" || got.Status != UnitPending {
		t.Errorf("the unit over the same lines keeps its id and the Creator's field, the rest is the AI's, and it goes back to review: %+v", got)
	}
	if got := m.Units[0]; got.UnitID != "VP-002" || got.NarrationSummary != "mới" {
		t.Errorf("a unit with no Creator field is the AI's, with the id of the unit over the same lines: %+v", got)
	}
	if len(m.Fresh) != 0 || strings.Join(m.Fields, ",") != "VP-007.narration_summary" {
		t.Errorf("fresh %v fields %v", m.Fresh, m.Fields)
	}

	split := []PlanUnit{unitOf("", "a", "b1.1"), unitOf("", "b", "b1.2"), unitOf("", "c", "b1.3")}
	m, err = MergeBeatReply(split, BeatPlan{Units: []PlanUnit{unitOf("VP-001", "x", "b1.1", "b1.2", "b1.3")}}, nil)
	if err != nil || len(m.Fresh) != 3 {
		t.Fatalf("no unit over the same lines means three fresh units: %v %v", m.Fresh, err)
	}
	m.Assign(10)
	if m.Units[0].UnitID != "VP-010" || m.Units[1].UnitID != "VP-011" || m.Units[2].UnitID != "VP-012" {
		t.Errorf("fresh ids run on from the first reserved: %v %v %v", m.Units[0].UnitID, m.Units[1].UnitID, m.Units[2].UnitID)
	}
}

func TestMergeBeatReply_RefusesToDropAUnitTheCreatorEdited(t *testing.T) {
	old := BeatPlan{Units: []PlanUnit{unitOf("VP-001", "tay", "b1.1", "b1.2"), unitOf("VP-002", "ai", "b1.3")}}
	whole := []PlanUnit{unitOf("", "gộp", "b1.1", "b1.2", "b1.3")}
	if _, err := MergeBeatReply(whole, old, []string{"VP-001.narration_summary"}); !errors.Is(err, ErrUnreadableReply) || !strings.Contains(err.Error(), "VP-001") {
		t.Errorf("dropping an edited unit must be refused by name, got %v", err)
	}
	if _, err := MergeBeatReply(whole, old, nil); err != nil {
		t.Errorf("with no Creator field every unit can be redrawn, got %v", err)
	}
}

func TestParsePlanReviewReply_AcceptsOnlyUnitsAndFieldsOfThePlan(t *testing.T) {
	units := map[string]bool{"VP-001": true}
	item := func(group, unit, field string) []byte {
		return []byte(`{"findings":[{"group":"` + group + `","unit_key":"` + unit + `","field":"` + field + `","text":"t","detail":"d","fix":"","same_as":""}],"notes":[]}`)
	}
	if r, err := ParsePlanReviewReply(item("clarity", "VP-001", "communication_goal"), ReviewState{}, units); err != nil || len(r.Findings) != 1 {
		t.Fatalf("a finding on a unit and a field of it must pass: %v", err)
	}
	if _, err := ParsePlanReviewReply(item("clarity", "VP-001", ""), ReviewState{}, units); err != nil {
		t.Errorf("a finding with no field must pass: %v", err)
	}
	for name, raw := range map[string][]byte{
		"a unit not in the plan":               item("clarity", "VP-009", ""),
		"a field the Creator cannot write":     item("clarity", "VP-001", "unit_id"),
		"a group of the concept's review only": item("flow", "VP-001", ""),
		"a part of the concept":                item("clarity", "color_system", ""),
	} {
		if _, err := ParsePlanReviewReply(raw, ReviewState{}, units); !errors.Is(err, ErrUnreadableReply) {
			t.Errorf("%s must be refused, got %v", name, err)
		}
	}
}

func TestPlanSystemPromptsShipWithTheConceptOnes(t *testing.T) {
	ps := SystemPrompts()
	if len(ps) != 12 || ps[10].Role != "visual_plan_beat" || ps[11].Role != "visual_plan_review" {
		t.Fatalf("%d prompts", len(ps))
	}
	for _, p := range ps[10:] {
		if p.Name != SystemPromptName || !strings.Contains(p.TemplateText, "```json") || strings.Contains(p.TemplateText, "{{url}}") {
			t.Errorf("%s is not shipped as written", p.Role)
		}
	}
	for _, m := range PlanMethods {
		if !strings.Contains(ps[10].TemplateText, m) {
			t.Errorf("the beat prompt does not name the method %s", m)
		}
	}
	for _, r := range NarrationRelationships {
		if !strings.Contains(ps[10].TemplateText, r) {
			t.Errorf("the beat prompt does not name the relationship %s", r)
		}
	}
}
