package domain

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// IDs of the machine checks of the whole concept.
const (
	CheckContrast = "m-contrast"
	CheckTerms    = "m-terms"
	CheckAccuracy = "m-accuracy"
)

// MachineCheck is one check of the whole concept, worked out on every read and
// never stored. Detail names what fails; a Creator's acceptance holds while
// Detail stays the same.
type MachineCheck struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
}

// Parts are the decoded parts of a concept that exist.
type Parts struct {
	CreativeDirection   *CreativeDirection
	ColorSystem         *ColorSystem
	VisualLanguage      *VisualLanguage
	RepresentationRules *RepresentationRules
	Typography          *Typography
	MotionLanguage      *MotionLanguage
	DoNotUse            *DoNotUse
	Feasibility         *Feasibility
}

// Set stores a decoded part under its key.
func (p *Parts) Set(key string, v any) {
	switch key {
	case UnitCreativeDirection:
		p.CreativeDirection, _ = v.(*CreativeDirection)
	case UnitColorSystem:
		p.ColorSystem, _ = v.(*ColorSystem)
	case UnitVisualLanguage:
		p.VisualLanguage, _ = v.(*VisualLanguage)
	case UnitRepresentationRules:
		p.RepresentationRules, _ = v.(*RepresentationRules)
	case UnitTypography:
		p.Typography, _ = v.(*Typography)
	case UnitMotionLanguage:
		p.MotionLanguage, _ = v.(*MotionLanguage)
	case UnitDoNotUse:
		p.DoNotUse, _ = v.(*DoNotUse)
	case UnitFeasibility:
		p.Feasibility, _ = v.(*Feasibility)
	}
}

// MachineChecks runs the three checks of the whole concept.
func MachineChecks(in ConceptInput, p Parts) []MachineCheck {
	return []MachineCheck{contrastCheck(in, p), termsCheck(in, p), accuracyCheck(in, p)}
}

func contrastCheck(in ConceptInput, p Parts) MachineCheck {
	c := MachineCheck{ID: CheckContrast, Label: "Chữ đủ tương phản trên nền của từng đoạn hành trình màu"}
	cs := p.ColorSystem
	if cs == nil {
		c.Detail = "Chưa có bảng màu (phần 2)."
		return c
	}
	hexOf := map[string]string{}
	for _, s := range cs.Palette {
		hexOf[s.Token] = s.Hex
	}
	text, ok := hexOf[cs.Contrast.TextToken]
	if !ok || cs.Contrast.TextMin <= 0 {
		c.Detail = "Bảng màu chưa nói màu chữ hoặc mức tương phản tối thiểu."
		return c
	}
	var fails []string
	passed := 0
	for _, j := range cs.Journey {
		bg := cs.Background
		if j.BackgroundToken != "" {
			bg = hexOf[j.BackgroundToken]
		}
		ratio, err := ContrastRatio(text, bg)
		if err != nil {
			fails = append(fails, fmt.Sprintf("%s: %v", beatList(in, j.BeatKeys), err))
			continue
		}
		if ratio+1e-9 < cs.Contrast.TextMin {
			fails = append(fails, fmt.Sprintf("%s: %s:1", beatList(in, j.BeatKeys), strconv.FormatFloat(math.Floor(ratio*10)/10, 'f', 1, 64)))
			continue
		}
		passed++
	}
	min := strconv.FormatFloat(cs.Contrast.TextMin, 'f', -1, 64)
	c.Label = fmt.Sprintf("Chữ đạt %s:1 ở %d/%d đoạn hành trình màu", min, passed, len(cs.Journey))
	if len(cs.Journey) == 0 {
		c.Detail = "Hành trình màu trống."
		return c
	}
	if len(fails) > 0 {
		c.Detail = "Chưa đạt " + min + ":1 ở " + strings.Join(fails, "; ") + "."
		return c
	}
	c.OK = true
	return c
}

func beatList(in ConceptInput, keys []string) string {
	var out []string
	for _, k := range keys {
		label := k
		for _, b := range in.Beats {
			if b.Key == k {
				label = fmt.Sprintf("B%d", b.Number)
			}
		}
		out = append(out, label)
	}
	return strings.Join(out, ", ")
}

func normTerm(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(s)), " ")
}

func termsCheck(in ConceptInput, p Parts) MachineCheck {
	c := MachineCheck{ID: CheckTerms}
	have := map[string]bool{}
	if r := p.RepresentationRules; r != nil {
		for _, list := range [][]RepresentedThing{r.Characters.Items, r.Objects, r.Invisible} {
			for _, t := range list {
				have[normTerm(t.Term)] = true
			}
		}
		for _, s := range r.ScientificConcepts {
			have[normTerm(s.Term)] = true
		}
	}
	var missing []string
	for _, g := range in.Glossary {
		if !have[normTerm(g.Term)] {
			missing = append(missing, g.Term)
		}
	}
	n := len(in.Glossary)
	c.Label = fmt.Sprintf("%d/%d thuật ngữ của kịch bản có cách vẽ ở phần 4", n-len(missing), n)
	if p.RepresentationRules == nil && n > 0 {
		c.Detail = "Chưa có phần 4."
		return c
	}
	if len(missing) > 0 {
		c.Detail = "Chưa có cách vẽ: " + strings.Join(missing, ", ") + "."
		return c
	}
	c.OK = true
	return c
}

func accuracyCheck(in ConceptInput, p Parts) MachineCheck {
	c := MachineCheck{ID: CheckAccuracy}
	used := map[string]bool{}
	var order []string
	for _, b := range in.Beats {
		for _, l := range in.Lines[b.Key] {
			for _, id := range l.Facts {
				f, ok := in.FactByID(id)
				if ok && (f.Type == FactTypeHypothesis || f.Type == FactTypeMisconception) && !used[id] {
					used[id] = true
					order = append(order, id)
				}
			}
		}
	}
	covered := map[string]bool{}
	if p.DoNotUse != nil {
		for _, it := range p.DoNotUse.Items {
			if it.Source == "accuracy" {
				for _, id := range it.Facts {
					covered[id] = true
				}
			}
		}
	}
	var missing []string
	for _, id := range order {
		if !covered[id] {
			missing = append(missing, id)
		}
	}
	c.Label = fmt.Sprintf("%d/%d giả thuyết và ngộ nhận trong lời thoại có dòng cảnh báo ở phần 7", len(order)-len(missing), len(order))
	if len(missing) > 0 {
		c.Detail = "Chưa có dòng loại accuracy cho: " + strings.Join(missing, ", ") + "."
		return c
	}
	c.OK = true
	return c
}

// ContrastRatio is the WCAG 2 contrast ratio of two #rrggbb colours.
func ContrastRatio(a, b string) (float64, error) {
	la, err := luminance(a)
	if err != nil {
		return 0, err
	}
	lb, err := luminance(b)
	if err != nil {
		return 0, err
	}
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05), nil
}

func luminance(hex string) (float64, error) {
	if !hexColor.MatchString(hex) {
		return 0, fmt.Errorf("màu «%s» không phải mã hex", hex)
	}
	var rgb [3]float64
	for i := 0; i < 3; i++ {
		n, _ := strconv.ParseUint(hex[1+2*i:3+2*i], 16, 8)
		c := float64(n) / 255
		if c <= 0.03928 {
			rgb[i] = c / 12.92
		} else {
			rgb[i] = math.Pow((c+0.055)/1.055, 2.4)
		}
	}
	return 0.2126*rgb[0] + 0.7152*rgb[1] + 0.0722*rgb[2], nil
}
