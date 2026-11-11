package domain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// Unit kinds of step 9.
const (
	// UnitKindPlan holds the visual-plan units of one beat; its unit_key is the
	// beat key.
	UnitKindPlan = "visual_plan"
	// UnitKindPlanReview holds the review of the whole plan under ReviewUnitKey.
	UnitKindPlanReview = "visual_plan_review"
)

// Methods a visual-plan unit can show its idea with.
var PlanMethods = []string{
	"spatial_composition", "split_perspective_comparison", "diagram", "analogy", "process_sequence",
	"scale_comparison", "data_visual", "character_action", "text_emphasis", "timeline", "cross_section",
}

// Relationships between the narration and the picture.
var NarrationRelationships = []string{"direct_illustration", "contrast", "complement", "metaphor"}

// Priorities of an essential visual item.
const (
	PriorityMust = "must"
	PriorityNice = "nice"
)

// durationEpsilon absorbs float noise when planned seconds meet the minimum.
const durationEpsilon = 1e-9

var planUnitIDPattern = regexp.MustCompile(`^VP-[0-9]{3,}$`)

// PlanUnitID is the id of the n-th visual-plan unit of a video; numbers are
// never reused.
func PlanUnitID(n int) string { return fmt.Sprintf("VP-%03d", n) }

// EssentialInfo is one thing the picture must show.
type EssentialInfo struct {
	Item     string `json:"item"`
	Priority string `json:"priority"`
}

// VisualApproach is how the picture gets its idea across.
type VisualApproach struct {
	Method          string   `json:"method"`
	Description     string   `json:"description"`
	ConceptRuleRefs []string `json:"concept_rule_refs"`
}

// RiskNote is a scientific risk of one fact the unit shows.
type RiskNote struct {
	FactID string `json:"fact_id"`
	Note   string `json:"note"`
}

// OpenQuestion is a question for the Creator; Answer is empty until answered.
type OpenQuestion struct {
	ID       string `json:"id"`
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

// PlannedDuration is how long the picture stays. PlannedS 0 means the narration
// minimum; a longer picture needs ExtraReason.
type PlannedDuration struct {
	PlannedS    float64 `json:"planned_s"`
	ExtraReason string  `json:"extra_reason"`
}

// PlanUnit is one unit of the visual plan: a run of consecutive lines of one beat
// and the picture that goes with them.
type PlanUnit struct {
	UnitID                string          `json:"unit_id"`
	ScriptRefs            []string        `json:"script_refs"`
	NarrationSummary      string          `json:"narration_summary"`
	CommunicationGoal     string          `json:"communication_goal"`
	EssentialVisualInfo   []EssentialInfo `json:"essential_visual_info"`
	OnScreenText          []string        `json:"on_screen_text"`
	VisualApproach        VisualApproach  `json:"visual_approach"`
	NarrationRelationship string          `json:"narration_relationship"`
	RiskNotes             []RiskNote      `json:"risk_notes"`
	OpenQuestions         []OpenQuestion  `json:"open_questions"`
	Duration              PlannedDuration `json:"duration"`
	Status                string          `json:"status"`
}

// BeatPlan is the content of a visual_plan review unit: the units of one beat in
// story order.
type BeatPlan struct {
	Units []PlanUnit `json:"units"`
}

// PlanEditableFields are the fields of a unit the Creator can write. The id and
// the lines change only by splitting and merging, the status by approving.
var PlanEditableFields = []string{
	"narration_summary", "communication_goal", "essential_visual_info", "on_screen_text",
	"visual_approach", "narration_relationship", "risk_notes", "open_questions", "duration",
}

// DecodeBeatPlan reads the stored content of a beat's units. Empty content is a
// beat with no unit yet; an unknown field is an error.
func DecodeBeatPlan(raw []byte) (BeatPlan, error) {
	var p BeatPlan
	if len(bytes.TrimSpace(raw)) > 0 && string(raw) != "{}" {
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&p); err != nil {
			return BeatPlan{}, fmt.Errorf("%w: %v", ErrInvalidEdit, err)
		}
	}
	for i := range p.Units {
		p.Units[i].normalize()
	}
	if p.Units == nil {
		p.Units = []PlanUnit{}
	}
	return p, nil
}

// DecodePlanUnit reads one unit's JSON. An unknown field or a value of the wrong
// type is ErrInvalidEdit.
func DecodePlanUnit(raw []byte) (PlanUnit, error) {
	var u PlanUnit
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&u); err != nil {
		return PlanUnit{}, fmt.Errorf("%w: %v", ErrInvalidEdit, err)
	}
	u.normalize()
	return u, nil
}

// Encode returns the stored content of the beat.
func (p BeatPlan) Encode() ([]byte, error) {
	for i := range p.Units {
		p.Units[i].normalize()
	}
	if p.Units == nil {
		p.Units = []PlanUnit{}
	}
	return json.Marshal(p)
}

// normalize turns the lists a JSON null left nil into empty lists, so the unit
// always encodes as arrays.
func (u *PlanUnit) normalize() {
	if u.ScriptRefs == nil {
		u.ScriptRefs = []string{}
	}
	if u.EssentialVisualInfo == nil {
		u.EssentialVisualInfo = []EssentialInfo{}
	}
	if u.OnScreenText == nil {
		u.OnScreenText = []string{}
	}
	if u.VisualApproach.ConceptRuleRefs == nil {
		u.VisualApproach.ConceptRuleRefs = []string{}
	}
	if u.RiskNotes == nil {
		u.RiskNotes = []RiskNote{}
	}
	if u.OpenQuestions == nil {
		u.OpenQuestions = []OpenQuestion{}
	}
	if u.Status == "" {
		u.Status = UnitPending
	}
}

// Index returns the position of the unit with id, or -1.
func (p BeatPlan) Index(id string) int {
	for i, u := range p.Units {
		if u.UnitID == id {
			return i
		}
	}
	return -1
}

// UnansweredQuestions lists the ids of the questions still without an answer.
func (u PlanUnit) UnansweredQuestions() []string {
	var out []string
	for _, q := range u.OpenQuestions {
		if strings.TrimSpace(q.Answer) == "" {
			out = append(out, q.ID)
		}
	}
	return out
}

// FactRefs lists the facts of the unit's lines, each once, in the order they
// first appear.
func FactRefs(u PlanUnit, lines []Line) []string {
	byID := map[string]Line{}
	for _, l := range lines {
		byID[l.ID] = l
	}
	seen := map[string]bool{}
	out := []string{}
	for _, ref := range u.ScriptRefs {
		for _, f := range byID[ref].Facts {
			if !seen[f] {
				seen[f] = true
				out = append(out, f)
			}
		}
	}
	return out
}

// UnitWords counts the words the voice speaks over the unit's lines.
func UnitWords(u PlanUnit, lines []Line) int {
	byID := map[string]Line{}
	for _, l := range lines {
		byID[l.ID] = l
	}
	n := 0
	for _, ref := range u.ScriptRefs {
		n += len(strings.Fields(byID[ref].Text))
	}
	return n
}

// NarrationMin is the shortest time the unit can last: its words at the video's
// own speaking pace, rounded up to a tenth of a second. 0 when the pace is unknown.
func NarrationMin(words int, secondsPerWord float64) float64 {
	return math.Ceil(float64(words)*secondsPerWord*10-durationEpsilon) / 10
}

// PlannedSeconds is how long the unit stays: PlannedS, or the narration
// minimum when none is set.
func PlannedSeconds(u PlanUnit, narrationMin float64) float64 {
	if u.Duration.PlannedS > 0 {
		return u.Duration.PlannedS
	}
	return narrationMin
}

// Round1 rounds to a tenth.
func Round1(v float64) float64 { return math.Round(v*10) / 10 }

// UnitSeq is the number inside a unit id (VP-012 is 12), or 0.
func UnitSeq(id string) int {
	if !planUnitIDPattern.MatchString(id) {
		return 0
	}
	n, _ := strconv.Atoi(strings.TrimPrefix(id, "VP-"))
	return n
}

// lineIndex maps each line id of a beat to its position.
func lineIndex(lines []Line) map[string]int {
	idx := make(map[string]int, len(lines))
	for i, l := range lines {
		idx[l.ID] = i
	}
	return idx
}

// SplitPoint returns the position of line atLine in the unit, or ErrNotSplittable
// when the unit cannot be cut before it: the line is not in the unit, or is its
// first.
func SplitPoint(u PlanUnit, atLine string) (int, error) {
	for i, ref := range u.ScriptRefs {
		if ref == atLine && i > 0 {
			return i, nil
		}
	}
	return 0, fmt.Errorf("%w: câu %q không nằm trong unit hoặc là câu đầu của unit", ErrNotSplittable, atLine)
}

// SplitUnit cuts the unit before line atLine. The first unit keeps the lines
// before it, the second (with newID) takes atLine and the lines after. Both go
// back to review with a fresh duration; the text is copied to both for the
// Creator to adjust, and each keeps only the risk notes of its own facts. The
// open questions stay with the first.
func SplitUnit(u PlanUnit, atLine, newID string, lines []Line) (PlanUnit, PlanUnit, error) {
	at, err := SplitPoint(u, atLine)
	if err != nil {
		return PlanUnit{}, PlanUnit{}, err
	}
	first, second := u, u
	first.ScriptRefs = append([]string{}, u.ScriptRefs[:at]...)
	second.ScriptRefs = append([]string{}, u.ScriptRefs[at:]...)
	second.UnitID = newID
	second.OpenQuestions = []OpenQuestion{}
	second.EssentialVisualInfo = append([]EssentialInfo{}, u.EssentialVisualInfo...)
	second.OnScreenText = append([]string{}, u.OnScreenText...)
	second.VisualApproach.ConceptRuleRefs = append([]string{}, u.VisualApproach.ConceptRuleRefs...)
	for _, p := range []*PlanUnit{&first, &second} {
		p.Duration = PlannedDuration{}
		p.Status = UnitPending
		own := map[string]bool{}
		for _, f := range FactRefs(*p, lines) {
			own[f] = true
		}
		kept := []RiskNote{}
		for _, r := range u.RiskNotes {
			if own[r.FactID] {
				kept = append(kept, r)
			}
		}
		p.RiskNotes = kept
	}
	return first, second, nil
}

// MergeUnits joins unit b, the one right after a, into a. The result keeps a's
// id and text, takes both units' lines, items, on-screen text, risk notes and
// questions (each once), and goes back to review with a fresh duration.
func MergeUnits(a, b PlanUnit) (PlanUnit, error) {
	out := a
	out.ScriptRefs = append(append([]string{}, a.ScriptRefs...), b.ScriptRefs...)
	seenItem := map[string]bool{}
	out.EssentialVisualInfo = []EssentialInfo{}
	for _, it := range append(append([]EssentialInfo{}, a.EssentialVisualInfo...), b.EssentialVisualInfo...) {
		if !seenItem[it.Item] {
			seenItem[it.Item] = true
			out.EssentialVisualInfo = append(out.EssentialVisualInfo, it)
		}
	}
	out.OnScreenText = dedupe(append(append([]string{}, a.OnScreenText...), b.OnScreenText...))
	out.VisualApproach.ConceptRuleRefs = dedupe(append(append([]string{}, a.VisualApproach.ConceptRuleRefs...), b.VisualApproach.ConceptRuleRefs...))
	seenRisk := map[string]bool{}
	out.RiskNotes = []RiskNote{}
	for _, r := range append(append([]RiskNote{}, a.RiskNotes...), b.RiskNotes...) {
		if !seenRisk[r.FactID] {
			seenRisk[r.FactID] = true
			out.RiskNotes = append(out.RiskNotes, r)
		}
	}
	out.OpenQuestions = append([]OpenQuestion{}, a.OpenQuestions...)
	taken := map[string]bool{}
	for _, q := range a.OpenQuestions {
		taken[q.ID] = true
	}
	for _, q := range b.OpenQuestions {
		for taken[q.ID] {
			q.ID += "b"
		}
		taken[q.ID] = true
		out.OpenQuestions = append(out.OpenQuestions, q)
	}
	out.Duration = PlannedDuration{}
	out.Status = UnitPending
	return out, nil
}

func dedupe(in []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// RuleRefs lists the rules of a locked visual concept a plan unit can point
// at, each with a short description. The ids are:
//
//	color_system.palette.<token>
//	visual_language.<group>[<index>]
//	representation_rules.characters.items.<concept_id>
//	representation_rules.objects.<concept_id>
//	representation_rules.invisible.<concept_id>
//	representation_rules.scientific_concepts.<concept_id>
//	motion_language.preferred_transitions[<index>]
//	creative_direction.principles[<index>]
func RuleRefs(d Document) map[string]string {
	out := map[string]string{}
	for _, s := range d.ColorSystem.Palette {
		out["color_system.palette."+s.Token] = strings.TrimSpace(s.Name + " " + s.Hex + " · " + s.Role)
	}
	groups := []struct {
		name  string
		rules []string
	}{
		{"shape_rules", d.VisualLanguage.ShapeRules}, {"line_rules", d.VisualLanguage.LineRules},
		{"lighting_rules", d.VisualLanguage.LightingRules}, {"depth_rules", d.VisualLanguage.DepthRules},
		{"surface_rules", d.VisualLanguage.SurfaceRules}, {"layout_rules", d.VisualLanguage.LayoutRules},
		{"composition_rules", d.VisualLanguage.CompositionRules},
	}
	for _, g := range groups {
		for i, r := range g.rules {
			out[fmt.Sprintf("visual_language.%s[%d]", g.name, i)] = r
		}
	}
	for _, t := range d.RepresentationRules.Characters.Items {
		out["representation_rules.characters.items."+t.ConceptID] = t.Term + ": " + t.Draw
	}
	for _, t := range d.RepresentationRules.Objects {
		out["representation_rules.objects."+t.ConceptID] = t.Term + ": " + t.Draw
	}
	for _, t := range d.RepresentationRules.Invisible {
		out["representation_rules.invisible."+t.ConceptID] = t.Term + ": " + t.Draw
	}
	for _, s := range d.RepresentationRules.ScientificConcepts {
		out["representation_rules.scientific_concepts."+s.ConceptID] = s.Term + ": " + s.Representation
	}
	for i, t := range d.MotionLanguage.PreferredTransitions {
		out[fmt.Sprintf("motion_language.preferred_transitions[%d]", i)] = t
	}
	for i, p := range d.CreativeDirection.Principles {
		out[fmt.Sprintf("creative_direction.principles[%d]", i)] = p
	}
	return out
}
