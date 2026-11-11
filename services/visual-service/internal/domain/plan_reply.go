package domain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// planUnitReply is one unit as the AI writes it. The id, the status, the facts
// and the narration minimum are the machine's, so the AI has no key for them
// and a reply that carries one is refused.
type planUnitReply struct {
	ScriptRefs            []string        `json:"script_refs"`
	NarrationSummary      string          `json:"narration_summary"`
	CommunicationGoal     string          `json:"communication_goal"`
	EssentialVisualInfo   []EssentialInfo `json:"essential_visual_info"`
	OnScreenText          []string        `json:"on_screen_text"`
	VisualApproach        VisualApproach  `json:"visual_approach"`
	NarrationRelationship string          `json:"narration_relationship"`
	Duration              PlannedDuration `json:"duration"`
	RiskNotes             []RiskNote      `json:"risk_notes"`
	OpenQuestions         []OpenQuestion  `json:"open_questions"`
}

type planBeatReply struct {
	Units []planUnitReply `json:"units"`
}

// ParsePlanBeatReply reads the AI's units of one beat. The reply is unusable,
// and nothing is stored, when a key or value has the wrong shape, an enum
// takes a value outside its list, or the units do not cover the beat's lines
// once each, in order. What else fails (a missing rule, a risk note, the
// duration) is left to the unit's own checks. The units come back without ids.
func ParsePlanBeatReply(raw []byte, lines []Line) ([]PlanUnit, error) {
	var r planBeatReply
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&r); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnreadableReply, err)
	}
	bad := func(format string, a ...any) error {
		return fmt.Errorf("%w: %s", ErrUnreadableReply, fmt.Sprintf(format, a...))
	}
	if len(r.Units) == 0 {
		return nil, bad("không có unit nào")
	}
	var covered []string
	units := make([]PlanUnit, 0, len(r.Units))
	for i, ru := range r.Units {
		n := i + 1
		if len(ru.ScriptRefs) == 0 {
			return nil, bad("unit %d không có câu lời thoại nào", n)
		}
		if !contains(PlanMethods, ru.VisualApproach.Method) {
			return nil, bad("unit %d có cách hiển thị lạ «%s»", n, ru.VisualApproach.Method)
		}
		if !contains(NarrationRelationships, ru.NarrationRelationship) {
			return nil, bad("unit %d có quan hệ với lời thoại lạ «%s»", n, ru.NarrationRelationship)
		}
		for _, it := range ru.EssentialVisualInfo {
			if it.Priority != PriorityMust && it.Priority != PriorityNice {
				return nil, bad("unit %d có mức ưu tiên lạ «%s»", n, it.Priority)
			}
		}
		covered = append(covered, ru.ScriptRefs...)
		u := PlanUnit{ScriptRefs: ru.ScriptRefs, NarrationSummary: ru.NarrationSummary, CommunicationGoal: ru.CommunicationGoal,
			EssentialVisualInfo: ru.EssentialVisualInfo, OnScreenText: ru.OnScreenText, VisualApproach: ru.VisualApproach,
			NarrationRelationship: ru.NarrationRelationship, RiskNotes: ru.RiskNotes, OpenQuestions: ru.OpenQuestions,
			Duration: ru.Duration, Status: UnitPending}
		u.normalize()
		units = append(units, u)
	}
	if len(covered) != len(lines) {
		return nil, bad("các unit phủ %d câu, beat có %d câu", len(covered), len(lines))
	}
	for i, l := range lines {
		if covered[i] != l.ID {
			return nil, bad("câu thứ %d của các unit là %s, đáng lẽ là %s: mỗi câu của beat thuộc đúng một unit, đúng thứ tự", i+1, covered[i], l.ID)
		}
	}
	return units, nil
}

// BeatMerge is a beat's new units set against the units it had.
type BeatMerge struct {
	// Units are the new units in story order; the ones in Fresh have no id yet.
	Units []PlanUnit
	// Fresh are the positions in Units that need a new id.
	Fresh []int
	// Fields are the Creator's fields that still stand, as "<unit id>.<field>".
	Fields []string
}

// MergeBeatReply sets the AI's units of a beat against the beat's stored
// units. A new unit that covers exactly the lines of an old one is that unit:
// it keeps its id and the fields the Creator wrote, and every other field is
// the AI's. A new unit with no such match is fresh. A reply that drops a unit
// the Creator edited by hand is unusable, so their work is never lost quietly.
func MergeBeatReply(fresh []PlanUnit, old BeatPlan, creatorFields []string) (BeatMerge, error) {
	out := BeatMerge{Units: make([]PlanUnit, len(fresh)), Fields: []string{}}
	taken := map[string]bool{}
	for i, u := range fresh {
		at := -1
		for j, o := range old.Units {
			if !taken[o.UnitID] && sameStrings(o.ScriptRefs, u.ScriptRefs) {
				at = j
				break
			}
		}
		if at < 0 {
			out.Units[i] = u
			out.Fresh = append(out.Fresh, i)
			continue
		}
		o := old.Units[at]
		taken[o.UnitID] = true
		names := fieldsOfUnit(creatorFields, o.UnitID)
		u.UnitID = o.UnitID
		if len(names) > 0 {
			merged, err := overlayUnit(u, o, names)
			if err != nil {
				return BeatMerge{}, err
			}
			u = merged
			for _, f := range names {
				out.Fields = append(out.Fields, o.UnitID+"."+f)
			}
		}
		out.Units[i] = u
	}
	for _, o := range old.Units {
		if !taken[o.UnitID] && len(fieldsOfUnit(creatorFields, o.UnitID)) > 0 {
			return BeatMerge{}, fmt.Errorf("%w: bỏ mất %s, unit Creator đã sửa tay; giữ các câu %s thành một unit",
				ErrUnreadableReply, o.UnitID, strings.Join(o.ScriptRefs, ", "))
		}
	}
	return out, nil
}

// Assign gives the fresh units the ids first, first+1, ... in story order.
func (m *BeatMerge) Assign(first int) {
	for k, at := range m.Fresh {
		m.Units[at].UnitID = PlanUnitID(first + k)
	}
}

// overlayUnit takes the named fields from old over the new unit.
func overlayUnit(fresh, old PlanUnit, names []string) (PlanUnit, error) {
	base, err := json.Marshal(fresh)
	if err != nil {
		return PlanUnit{}, err
	}
	over, err := json.Marshal(old)
	if err != nil {
		return PlanUnit{}, err
	}
	merged, err := OverlayFields(base, over, names)
	if err != nil {
		return PlanUnit{}, err
	}
	u, err := DecodePlanUnit(merged)
	if err != nil {
		return PlanUnit{}, err
	}
	u.Status = UnitPending
	return u, nil
}

// fieldsOfUnit lists the fields the Creator wrote in unit id, from paths of the
// form "<unit id>.<field>".
func fieldsOfUnit(paths []string, id string) []string {
	var out []string
	prefix := id + "."
	for _, p := range paths {
		if strings.HasPrefix(p, prefix) {
			out = append(out, strings.TrimPrefix(p, prefix))
		}
	}
	return out
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// ParsePlanReviewReply reads the AI's review of the plan. Besides what the
// review of the concept checks, a finding must name a unit of the plan, and a
// field the Creator can write.
func ParsePlanReviewReply(raw []byte, prev ReviewState, unitIDs map[string]bool) (ReviewReply, error) {
	return parseReviewReply(raw, prev, func(f ReviewReplyItem) string {
		if !unitIDs[f.UnitKey] {
			return "nói về unit lạ «" + f.UnitKey + "»"
		}
		if f.Field != "" && !contains(PlanEditableFields, f.Field) {
			return "nói về trường lạ «" + f.Field + "»"
		}
		return ""
	})
}
