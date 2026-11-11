package domain

import (
	"fmt"
	"math"
	"strings"
)

// IDs of the machine checks of the whole plan. They can be accepted with a
// reason; what fails a unit's own checks cannot.
const (
	CheckPlanRisk  = "m-plan-risk"
	CheckPlanAvoid = "m-plan-avoid"
)

// PlanContext is what one unit is checked against.
type PlanContext struct {
	// BeatKey is the beat the unit belongs to.
	BeatKey string
	// Lines are the beat's lines in script order.
	Lines []Line
	// Facts are the facts of the research dossier by id.
	Facts map[string]Fact
	// RuleRefs are the rules of the locked visual concept.
	RuleRefs map[string]string
	// SecondsPerWord is the video's speaking pace; 0 when unknown.
	SecondsPerWord float64
}

// PlanContextFor builds the context for a beat's units.
func (in PlanInput) PlanContextFor(beatKey string) PlanContext {
	facts := map[string]Fact{}
	for _, f := range in.Script.Facts {
		facts[f.ID] = f
	}
	return PlanContext{BeatKey: beatKey, Lines: in.Script.Lines[beatKey], Facts: facts, RuleRefs: in.RuleRefs, SecondsPerWord: in.SecondsPerWord}
}

// UnitNarrationMin is the shortest time the unit's lines last.
func (pc PlanContext) UnitNarrationMin(u PlanUnit) float64 {
	return NarrationMin(UnitWords(u, pc.Lines), pc.SecondsPerWord)
}

// CheckPlanUnit lists what fails in a unit; a unit with problems cannot be
// approved. It checks the schema, the lines, the concept references, the risk
// notes and the planned duration.
func CheckPlanUnit(u PlanUnit, pc PlanContext) []string {
	var p problems
	if !planUnitIDPattern.MatchString(u.UnitID) {
		p.addf("mã unit %q không đúng dạng VP-001", u.UnitID)
	}
	p.checkLines(u, pc)
	p.need(u.NarrationSummary, "Thiếu tóm tắt lời thoại")
	p.need(u.CommunicationGoal, "Thiếu mục tiêu truyền đạt")
	if len(u.EssentialVisualInfo) == 0 {
		p.add("Chưa có thông tin hình bắt buộc phải có")
	}
	for i, it := range u.EssentialVisualInfo {
		if strings.TrimSpace(it.Item) == "" {
			p.addf("Thông tin hình %d để trống", i+1)
		}
		p.enum(it.Priority, []string{PriorityMust, PriorityNice}, fmt.Sprintf("mức ưu tiên của thông tin hình %d", i+1))
	}
	for i, t := range u.OnScreenText {
		if strings.TrimSpace(t) == "" {
			p.addf("Chữ trên hình %d để trống", i+1)
		}
	}
	p.enum(u.VisualApproach.Method, PlanMethods, "cách hiển thị")
	p.need(u.VisualApproach.Description, "Thiếu mô tả cách hiển thị")
	for _, ref := range u.VisualApproach.ConceptRuleRefs {
		if _, ok := pc.RuleRefs[ref]; !ok {
			p.addf("Quy tắc %q không có trong ngôn ngữ hình ảnh đã chốt", ref)
		}
	}
	p.enum(u.NarrationRelationship, NarrationRelationships, "quan hệ với lời thoại")
	facts := map[string]bool{}
	for _, f := range FactRefs(u, pc.Lines) {
		facts[f] = true
	}
	for _, r := range u.RiskNotes {
		if !facts[r.FactID] {
			p.addf("Ghi chú rủi ro nói về dữ kiện %q, không có trong lời thoại của unit", r.FactID)
		}
		p.need(r.Note, "Ghi chú rủi ro của "+r.FactID+" để trống")
	}
	ids := map[string]bool{}
	for _, q := range u.OpenQuestions {
		if strings.TrimSpace(q.ID) == "" || strings.TrimSpace(q.Question) == "" {
			p.add("Câu hỏi mở thiếu mã hoặc nội dung")
		}
		if ids[q.ID] {
			p.addf("Câu hỏi mở %q bị trùng mã", q.ID)
		}
		ids[q.ID] = true
	}
	p.checkDuration(u, pc)
	return p.list
}

// checkLines checks that the unit's lines exist in its beat and follow one
// another without a gap.
func (p *problems) checkLines(u PlanUnit, pc PlanContext) {
	if len(u.ScriptRefs) == 0 {
		p.add("Unit chưa có câu lời thoại nào")
		return
	}
	idx := lineIndex(pc.Lines)
	prev := -1
	for _, ref := range u.ScriptRefs {
		at, ok := idx[ref]
		if !ok {
			p.addf("Câu %s không còn trong lời thoại của beat %s", ref, pc.BeatKey)
			return
		}
		if prev >= 0 && at != prev+1 {
			p.addf("Các câu của unit phải liền nhau, nhưng %s không đứng ngay sau câu trước", ref)
			return
		}
		prev = at
	}
}

// checkDuration checks the planned time against the narration minimum.
func (p *problems) checkDuration(u PlanUnit, pc PlanContext) {
	if pc.SecondsPerWord <= 0 {
		p.add(ErrNoNarrationPace.Error())
		return
	}
	min := pc.UnitNarrationMin(u)
	planned := u.Duration.PlannedS
	switch {
	case planned < 0:
		p.add("Thời lượng dự kiến không được âm")
	case planned > 0 && planned < min-durationEpsilon:
		p.addf("Thời lượng dự kiến %.1fs ngắn hơn lời thoại (tối thiểu %.1fs)", planned, min)
	case planned > min+durationEpsilon && strings.TrimSpace(u.Duration.ExtraReason) == "":
		p.addf("Giữ hình %.1fs, dài hơn lời thoại %.1fs, cần ghi lý do", planned, min)
	}
}

// AutoFixUnit repairs the two failures that need no judgement: a planned time
// shorter than the narration needs is raised to that minimum, and a risk note
// about a fact the unit's own lines do not carry is dropped. A field the Creator
// wrote (creatorFields) is left as it is. It returns the unit and what it changed.
func AutoFixUnit(u PlanUnit, pc PlanContext, creatorFields []string) (PlanUnit, []string) {
	var did []string
	mine := map[string]bool{}
	for _, f := range creatorFields {
		mine[f] = true
	}
	if pc.SecondsPerWord > 0 && !mine["duration"] {
		min := pc.UnitNarrationMin(u)
		if u.Duration.PlannedS > 0 && u.Duration.PlannedS < min-durationEpsilon {
			raised := math.Ceil(min*100) / 100
			did = append(did, fmt.Sprintf("Đã nâng thời lượng %.2fs → %.2fs (mức tối thiểu theo lời thoại)", u.Duration.PlannedS, raised))
			u.Duration.PlannedS = raised
		}
	}
	if !mine["risk_notes"] {
		facts := map[string]bool{}
		for _, f := range FactRefs(u, pc.Lines) {
			facts[f] = true
		}
		kept := []RiskNote{}
		for _, r := range u.RiskNotes {
			if facts[r.FactID] {
				kept = append(kept, r)
				continue
			}
			did = append(did, fmt.Sprintf("Đã bỏ ghi chú rủi ro %s vì %s không nằm trong lời thoại của unit", r.FactID, r.FactID))
		}
		u.RiskNotes = kept
	}
	return u, did
}

// ApprovalBlockers lists what keeps a unit from being approved besides its
// checks: questions still waiting for an answer.
func ApprovalBlockers(u PlanUnit) []string {
	var out []string
	for _, id := range u.UnansweredQuestions() {
		out = append(out, "Câu hỏi "+id+" chưa trả lời")
	}
	return out
}

// PlanCoverage says how much of the locked script the units cover.
type PlanCoverage struct {
	LinesTotal   int      `json:"lines_total"`
	LinesCovered int      `json:"lines_covered"`
	Uncovered    []string `json:"uncovered"`
	// Duplicated lists lines that more than one unit claims.
	Duplicated []string `json:"duplicated"`
}

// Complete reports whether every line belongs to exactly one unit.
func (c PlanCoverage) Complete() bool {
	return len(c.Uncovered) == 0 && len(c.Duplicated) == 0
}

// ComputeCoverage counts the lines of the whole script that belong to exactly
// one unit.
func ComputeCoverage(in PlanInput, plans map[string]BeatPlan) PlanCoverage {
	cov := PlanCoverage{Uncovered: []string{}, Duplicated: []string{}}
	count := map[string]int{}
	for _, p := range plans {
		for _, u := range p.Units {
			for _, ref := range u.ScriptRefs {
				count[ref]++
			}
		}
	}
	for _, b := range in.Script.Beats {
		for _, l := range in.Script.Lines[b.Key] {
			cov.LinesTotal++
			switch count[l.ID] {
			case 0:
				cov.Uncovered = append(cov.Uncovered, l.ID)
			case 1:
				cov.LinesCovered++
			default:
				cov.Duplicated = append(cov.Duplicated, l.ID)
			}
		}
	}
	return cov
}

// avoidPhrases are the wordings the glossary says not to use, split from each
// term's avoid cell at semicolons and line breaks.
func avoidPhrases(terms []GlossaryTerm) []struct{ Term, Phrase string } {
	var out []struct{ Term, Phrase string }
	for _, t := range terms {
		for _, ph := range strings.FieldsFunc(t.Avoid, func(r rune) bool { return r == ';' || r == '\n' }) {
			if ph = strings.TrimSpace(ph); len([]rune(ph)) >= 2 {
				out = append(out, struct{ Term, Phrase string }{t.Term, ph})
			}
		}
	}
	return out
}

// PlanChecks are the checks of the whole plan, worked out on every read and
// never stored. Detail names what fails; a Creator's acceptance holds while
// Detail stays the same.
func PlanChecks(in PlanInput, plans map[string]BeatPlan) []MachineCheck {
	var risk, avoid []string
	phrases := avoidPhrases(in.Script.Glossary)
	for _, b := range in.Script.Beats {
		pc := in.PlanContextFor(b.Key)
		for _, u := range plans[b.Key].Units {
			have := map[string]bool{}
			for _, r := range u.RiskNotes {
				have[r.FactID] = true
			}
			for _, fid := range FactRefs(u, pc.Lines) {
				f, ok := pc.Facts[fid]
				if ok && (f.Type == FactTypeHypothesis || f.Type == FactTypeMisconception) && !have[fid] {
					risk = append(risk, fmt.Sprintf("%s: dữ kiện %s (%s) chưa có ghi chú rủi ro", u.UnitID, fid, factTypeLabel(f.Type)))
				}
			}
			for _, text := range u.OnScreenText {
				for _, a := range phrases {
					if strings.Contains(strings.ToLower(text), strings.ToLower(a.Phrase)) {
						avoid = append(avoid, fmt.Sprintf("%s: chữ «%s» dùng cách nói bảng thuật ngữ bảo tránh «%s» (%s)", u.UnitID, text, a.Phrase, a.Term))
					}
				}
			}
		}
	}
	return []MachineCheck{
		{ID: CheckPlanRisk, Label: "Giả thuyết và ngộ nhận đều có ghi chú rủi ro", OK: len(risk) == 0, Detail: strings.Join(risk, "; ")},
		{ID: CheckPlanAvoid, Label: "Chữ trên hình không dùng cách nói bảng thuật ngữ bảo tránh", OK: len(avoid) == 0, Detail: strings.Join(avoid, "; ")},
	}
}

func factTypeLabel(t string) string {
	if t == FactTypeHypothesis {
		return "giả thuyết"
	}
	return "ngộ nhận"
}

// BeatTiming sets a beat's planned time beside the beat sheet's.
type BeatTiming struct {
	// NarrationMinS is the narration of all the beat's units.
	NarrationMinS float64 `json:"narration_min_s"`
	// PlannedS is the time the beat's units plan to hold.
	PlannedS float64 `json:"planned_s"`
	// BeatSheetS is the beat's length in the beat sheet.
	BeatSheetS int `json:"beat_sheet_s"`
	// DeltaS is PlannedS minus BeatSheetS; no threshold is applied.
	DeltaS float64 `json:"delta_s"`
}

// PlanTiming is the time of the whole plan against the brief.
type PlanTiming struct {
	Beats map[string]BeatTiming `json:"beats"`
	// PlannedS is the time of every unit written so far.
	PlannedS float64 `json:"planned_s"`
	// BriefS is the duration the creative brief asks for.
	BriefS int `json:"brief_s"`
	// OverBrief is true when PlannedS exceeds BriefS. It warns and never blocks.
	OverBrief bool `json:"over_brief"`
}

// ComputeTiming works out each beat's planned time and the total.
func ComputeTiming(in PlanInput, plans map[string]BeatPlan) PlanTiming {
	t := PlanTiming{Beats: map[string]BeatTiming{}, BriefS: in.Script.Brief.DurationSeconds}
	for _, b := range in.Script.Beats {
		units := plans[b.Key].Units
		if len(units) == 0 {
			continue
		}
		pc := in.PlanContextFor(b.Key)
		bt := BeatTiming{BeatSheetS: b.Dur}
		for _, u := range units {
			min := pc.UnitNarrationMin(u)
			bt.NarrationMinS += min
			bt.PlannedS += PlannedSeconds(u, min)
		}
		bt.NarrationMinS, bt.PlannedS = Round1(bt.NarrationMinS), Round1(bt.PlannedS)
		bt.DeltaS = Round1(bt.PlannedS - float64(b.Dur))
		t.Beats[b.Key] = bt
		t.PlannedS += bt.PlannedS
	}
	t.PlannedS = Round1(t.PlannedS)
	t.OverBrief = t.BriefS > 0 && t.PlannedS > float64(t.BriefS)+durationEpsilon
	return t
}
