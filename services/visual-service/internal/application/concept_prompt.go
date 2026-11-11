package application

import (
	"fmt"
	"strings"

	"visual/internal/domain"
)

// The texts the prompts and the "Xem đầu vào của AI" view show of each input.

const none = "(không có)"

func briefText(b domain.Brief) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Khán giả: %s\n", b.Audience)
	if b.DurationSeconds > 0 {
		fmt.Fprintf(&sb, "Thời lượng: %d giây\n", b.DurationSeconds)
	}
	fmt.Fprintf(&sb, "Giọng điệu: %s\n", b.Tone)
	sb.WriteString("Người xem cần hiểu:\n")
	for _, o := range b.LearningObjectives {
		sb.WriteString("- " + o + "\n")
	}
	sb.WriteString("Cần tránh:\n")
	for _, a := range b.Avoid {
		sb.WriteString("- " + a + "\n")
	}
	return strings.TrimSpace(sb.String())
}

var factTypeLabels = map[string]string{
	"fact": "sự thật", "hypothesis": "giả thuyết", "unknown": "chưa rõ", "misconception": "ngộ nhận",
}

func factsText(in domain.ConceptInput) string {
	if in.ResearchSkipped {
		return "(Video bỏ qua bước Nghiên cứu: không có dữ kiện.)"
	}
	var lines []string
	for _, f := range in.Facts {
		t := factTypeLabels[f.Type]
		if t == "" {
			t = f.Type
		}
		line := fmt.Sprintf("%s [%s] %s", f.ID, t, f.Claim)
		if strings.TrimSpace(f.Note) != "" {
			line += " (ghi chú: " + f.Note + ")"
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func storyText(story map[string]domain.StoryField) string {
	var sb strings.Builder
	for _, f := range []struct{ key, label string }{
		{"question", "Câu hỏi trung tâm"}, {"chain", "Mạch kể"}, {"payoff", "Câu trả lời cốt lõi"},
	} {
		if v, ok := story[f.key]; ok {
			fmt.Fprintf(&sb, "%s:\n%s\n\n", f.label, v.Text)
		}
	}
	return strings.TrimSpace(sb.String())
}

func beatsText(in domain.ConceptInput) string {
	var lines []string
	for _, b := range in.Beats {
		lines = append(lines, fmt.Sprintf("%s · B%d · %s · %s · %d giây\n  Mục tiêu: %s", b.Key, b.Number, b.Fn, b.Title, b.Dur, b.Goal))
	}
	return strings.Join(lines, "\n")
}

func scriptText(in domain.ConceptInput) string {
	var sb strings.Builder
	for _, b := range in.Beats {
		fmt.Fprintf(&sb, "%s · B%d · %s\n", b.Key, b.Number, b.Title)
		for _, l := range in.Lines[b.Key] {
			facts := ""
			if len(l.Facts) > 0 {
				facts = " [" + strings.Join(l.Facts, ", ") + "]"
			}
			fmt.Fprintf(&sb, "  %s %s%s\n", l.ID, l.Text, facts)
		}
	}
	return strings.TrimSpace(sb.String())
}

func handoffText(in domain.ConceptInput) string {
	var sb strings.Builder
	for _, b := range in.Beats {
		h := in.Handoff[b.Key]
		fmt.Fprintf(&sb, "%s · B%d · %s\n", b.Key, b.Number, b.Title)
		for _, s := range []struct {
			label string
			list  []string
		}{{"Bắt buộc truyền đạt", h.Must}, {"Thuật ngữ", h.Terms}, {"Phải giữ chính xác", h.Exact}, {"Được phép sáng tạo", h.Free}, {"Điểm chưa chắc", h.Unsure}} {
			if len(s.list) == 0 {
				continue
			}
			fmt.Fprintf(&sb, "  %s:\n", s.label)
			for _, l := range s.list {
				fmt.Fprintf(&sb, "  - %s\n", l)
			}
		}
	}
	return strings.TrimSpace(sb.String())
}

func glossaryText(in domain.ConceptInput) string {
	if len(in.Glossary) == 0 {
		return none
	}
	var lines []string
	for _, g := range in.Glossary {
		lines = append(lines, fmt.Sprintf("- %s · nói: %s · vẽ: %s · tránh: %s", g.Term, g.Say, g.Draw, g.Avoid))
	}
	return strings.Join(lines, "\n")
}

func refsText(refs []domain.ReferenceStyle) string {
	if len(refs) == 0 {
		return "(Creator chưa chọn mô tả tham chiếu nào.)"
	}
	var blocks []string
	for _, r := range refs {
		var sb strings.Builder
		fmt.Fprintf(&sb, "%s · %s: %s", r.ID, r.Name, r.Summary)
		for _, f := range []struct {
			label string
			v     domain.FlexText
		}{
			{"chất liệu", r.Drawing.Medium}, {"hình dạng", r.Drawing.Shape}, {"đường nét", r.Drawing.Outline}, {"tô", r.Drawing.Fill},
			{"đổ bóng", r.Drawing.Shading}, {"ánh sáng", r.Drawing.Light}, {"chiều sâu", r.Drawing.Depth}, {"phối cảnh", r.Drawing.Perspective},
			{"kết cấu", r.Drawing.Texture}, {"layout · cấu trúc", r.Layout.Structure}, {"layout · lớp", r.Layout.Layering},
			{"layout · thành phần cố định", r.Layout.PersistentElements},
		} {
			if f.v != "" {
				fmt.Fprintf(&sb, "\n  - %s: %s", f.label, f.v)
			}
		}
		blocks = append(blocks, sb.String())
	}
	return strings.Join(blocks, "\n")
}

func fontsText(fonts []string) string {
	return strings.Join(fonts, "\n")
}

// partsText shows the written parts, skipping skip.
func partsText(units map[string]domain.ReviewUnit, skip string) string {
	var sb strings.Builder
	for i, u := range domain.ConceptUnits {
		if u.Key == skip {
			continue
		}
		ru, ok := units[u.Key]
		if !ok || len(ru.Content) == 0 {
			continue
		}
		fmt.Fprintf(&sb, "### %d · %s (%s)\n%s\n\n", i+1, u.Title, u.Key, string(ru.Content))
	}
	if sb.Len() == 0 {
		return "(chưa có phần nào)"
	}
	return strings.TrimSpace(sb.String())
}

func orNone(s string) string {
	if strings.TrimSpace(s) == "" {
		return none
	}
	return s
}

// renderTemplate fills every {{name}} of vars into text.
func renderTemplate(text string, vars map[string]string) string {
	for k, v := range vars {
		text = strings.ReplaceAll(text, "{{"+k+"}}", v)
	}
	return text
}

// inputVars are the variables every prompt of the step can use.
func inputVars(in domain.ConceptInput, fonts []string) map[string]string {
	return map[string]string{
		"brief":     briefText(in.Brief),
		"facts":     factsText(in),
		"story":     storyText(in.Story),
		"beats":     beatsText(in),
		"script":    scriptText(in),
		"handoff":   handoffText(in),
		"glossary":  glossaryText(in),
		"refs":      refsText(in.Refs),
		"fonts":     fontsText(fonts),
		"beat_keys": strings.Join(in.BeatKeys(), ", "),
	}
}

// InputSection is one input as the "Xem đầu vào của AI" view shows it.
type InputSection struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Text  string `json:"text"`
}

func inputSections(in domain.ConceptInput) []InputSection {
	return []InputSection{
		{domain.SourceBrief, domain.SourceLabels[domain.SourceBrief], briefText(in.Brief)},
		{domain.SourceResearch, domain.SourceLabels[domain.SourceResearch], factsText(in)},
		{domain.SourceStory, domain.SourceLabels[domain.SourceStory], storyText(in.Story)},
		{domain.SourceBeats, domain.SourceLabels[domain.SourceBeats], beatsText(in)},
		{domain.SourceScript, fmt.Sprintf("Kịch bản đã chốt v%d", in.ScriptVersion), scriptText(in) + "\n\nGhi chú bàn giao:\n" + handoffText(in) + "\n\nThuật ngữ:\n" + glossaryText(in)},
		{domain.SourceRefs, domain.SourceLabels[domain.SourceRefs], refsText(in.Refs)},
	}
}
