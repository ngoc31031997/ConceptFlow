package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// The kinds of unit the handoff stores. The note of a beat is an ordinary
// review unit keyed by the beat; the glossary is one unit outside the generic
// unit API.
const (
	UnitKindHandoffBeat = "handoff_beat"
	UnitKindGlossary    = "glossary"
	GlossaryUnitKey     = "main"
)

// Sections of a beat's handoff note, by key.
const (
	HandoffMust   = "must"
	HandoffTerms  = "terms"
	HandoffExact  = "exact"
	HandoffFree   = "free"
	HandoffUnsure = "unsure"
)

// Roles that read the handoff of a locked script.
const (
	HandoffRoleArtWorld     = "art_world"
	HandoffRoleStoryboard   = "storyboard"
	HandoffRoleArtColor     = "art_color"
	HandoffRoleVoice        = "voice"
	HandoffRoleIllustration = "illustration"
	HandoffRoleCode         = "code"
	HandoffRoleValidate     = "validate"
)

// HandoffSection describes one section of a beat's note.
type HandoffSection struct {
	Key   string   `json:"key"`
	Label string   `json:"label"`
	Roles []string `json:"roles"`
}

// HandoffSections are the five sections in the order the screen shows them,
// each with the roles that read it (FR-32).
var HandoffSections = []HandoffSection{
	{HandoffMust, "Thông tin bắt buộc truyền đạt", []string{HandoffRoleStoryboard, HandoffRoleIllustration}},
	{HandoffTerms, "Thuật ngữ cần giải thích, tránh hiểu sai", []string{HandoffRoleArtWorld, HandoffRoleStoryboard, HandoffRoleVoice}},
	{HandoffExact, "Phải giữ chính xác", []string{HandoffRoleArtWorld, HandoffRoleIllustration, HandoffRoleValidate}},
	{HandoffFree, "Được phép sáng tạo", []string{HandoffRoleStoryboard, HandoffRoleCode}},
	{HandoffUnsure, "Điểm chưa chắc, xử lý cẩn thận", []string{HandoffRoleStoryboard, HandoffRoleIllustration, HandoffRoleValidate}},
}

// RoleLabels name the roles for the Creator.
var RoleLabels = map[string]string{
	HandoffRoleArtWorld: "Mỹ thuật", HandoffRoleStoryboard: "Storyboard", HandoffRoleArtColor: "Mỹ thuật lượt 2",
	HandoffRoleVoice: "Giọng", HandoffRoleIllustration: "Hình minh hoạ", HandoffRoleCode: "Code", HandoffRoleValidate: "Validate",
}

// IsHandoffSection reports whether key is one of HandoffSections.
func IsHandoffSection(key string) bool {
	for _, s := range HandoffSections {
		if s.Key == key {
			return true
		}
	}
	return false
}

// IsHandoffRole reports whether role is a role that reads the handoff.
func IsHandoffRole(role string) bool { _, ok := RoleLabels[role]; return ok }

// Limits of a handoff note.
const (
	MaxHandoffItems     = 12
	MaxHandoffItemRunes = 400
	MaxGlossaryTerms    = 60
	MaxGlossaryRunes    = 200
)

// ErrHandoffReplyUnusable marks a model reply the handoff cannot be read from.
var ErrHandoffReplyUnusable = errors.New("câu trả lời của AI không dùng được làm ghi chú bàn giao")

// ErrHandoffEmpty is returned when a beat's note with nothing in any section is approved.
var ErrHandoffEmpty = errors.New("ghi chú bàn giao chưa có mục nào")

// HandoffNote is the content of a handoff_beat unit.
type HandoffNote struct {
	Must   []string `json:"must"`
	Terms  []string `json:"terms"`
	Exact  []string `json:"exact"`
	Free   []string `json:"free"`
	Unsure []string `json:"unsure"`
}

// Normalized replaces missing sections with empty ones.
func (n HandoffNote) Normalized() HandoffNote {
	for _, p := range []*[]string{&n.Must, &n.Terms, &n.Exact, &n.Free, &n.Unsure} {
		if *p == nil {
			*p = []string{}
		}
	}
	return n
}

// Section returns the items of a section.
func (n HandoffNote) Section(key string) []string {
	switch key {
	case HandoffMust:
		return n.Must
	case HandoffTerms:
		return n.Terms
	case HandoffExact:
		return n.Exact
	case HandoffFree:
		return n.Free
	case HandoffUnsure:
		return n.Unsure
	}
	return nil
}

// With returns the note with the items of a section replaced.
func (n HandoffNote) With(key string, items []string) HandoffNote {
	switch key {
	case HandoffMust:
		n.Must = items
	case HandoffTerms:
		n.Terms = items
	case HandoffExact:
		n.Exact = items
	case HandoffFree:
		n.Free = items
	case HandoffUnsure:
		n.Unsure = items
	}
	return n
}

// Empty reports whether no section holds an item.
func (n HandoffNote) Empty() bool {
	for _, s := range HandoffSections {
		if len(n.Section(s.Key)) > 0 {
			return false
		}
	}
	return true
}

func validateHandoffItem(text string) error {
	t := strings.TrimSpace(text)
	if t == "" {
		return errors.New("mục bàn giao không được để trống")
	}
	if n := len([]rune(t)); n > MaxHandoffItemRunes {
		return fmt.Errorf("mục bàn giao dài %d ký tự, tối đa %d", n, MaxHandoffItemRunes)
	}
	return nil
}

// Validate checks a note the way a hand edit and the AI's reply both must satisfy it.
func (n HandoffNote) Validate() error {
	for _, s := range HandoffSections {
		items := n.Section(s.Key)
		if len(items) > MaxHandoffItems {
			return fmt.Errorf("%s có %d mục, tối đa %d", s.Label, len(items), MaxHandoffItems)
		}
		for _, it := range items {
			if err := validateHandoffItem(it); err != nil {
				return fmt.Errorf("%s: %v", s.Label, err)
			}
		}
	}
	return nil
}

// SplitItems turns text with one item per line into items, dropping blank lines.
func SplitItems(text string) []string {
	out := []string{}
	for _, l := range strings.Split(text, "\n") {
		if t := strings.TrimSpace(l); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// GlossaryTerm is one row of the shared glossary.
type GlossaryTerm struct {
	Term  string `json:"term"`
	Say   string `json:"say"`
	Draw  string `json:"draw"`
	Avoid string `json:"avoid"`
	// Mine is true for a row the Creator wrote; the AI keeps it.
	Mine bool `json:"mine"`
}

// Glossary is the content of the glossary unit.
type Glossary struct {
	Terms []GlossaryTerm `json:"terms"`
}

// Validate checks one row.
func (t GlossaryTerm) Validate() error {
	if strings.TrimSpace(t.Term) == "" {
		return errors.New("thuật ngữ không được để trống")
	}
	for _, s := range []string{t.Term, t.Say, t.Draw, t.Avoid} {
		if len([]rune(s)) > MaxGlossaryRunes {
			return fmt.Errorf("một ô của thuật ngữ dài quá %d ký tự", MaxGlossaryRunes)
		}
	}
	return nil
}

// Merge adds rows, keeping the rows already there: a term that exists (same
// words, any case) is left as it is.
func (g Glossary) Merge(rows []GlossaryTerm) Glossary {
	have := map[string]bool{}
	for _, t := range g.Terms {
		have[strings.ToLower(strings.TrimSpace(t.Term))] = true
	}
	for _, r := range rows {
		k := strings.ToLower(strings.TrimSpace(r.Term))
		if k == "" || have[k] || len(g.Terms) >= MaxGlossaryTerms {
			continue
		}
		have[k] = true
		g.Terms = append(g.Terms, r)
	}
	return g
}

// HandoffReply is what one AI call for a beat's handoff returned.
type HandoffReply struct {
	Note  HandoffNote
	Terms []GlossaryTerm
}

// ParseHandoffReply reads a model reply {"must":[..],"terms":[..],"exact":[..],
// "free":[..],"unsure":[..],"glossary":[{"term","say","draw","avoid"}]}. A
// section may be empty; an item must have text. Anything unreadable stores nothing.
func ParseHandoffReply(reply string) (HandoffReply, error) {
	var wire struct {
		Must     []string `json:"must"`
		Terms    []string `json:"terms"`
		Exact    []string `json:"exact"`
		Free     []string `json:"free"`
		Unsure   []string `json:"unsure"`
		Glossary []struct {
			Term  string `json:"term"`
			Say   string `json:"say"`
			Draw  string `json:"draw"`
			Avoid string `json:"avoid"`
		} `json:"glossary"`
	}
	if err := json.Unmarshal([]byte(reply), &wire); err != nil {
		return HandoffReply{}, fmt.Errorf("%w: khối JSON không đọc được (%v)", ErrHandoffReplyUnusable, err)
	}
	clean := func(in []string) []string {
		out := []string{}
		for _, s := range in {
			if t := strings.TrimSpace(s); t != "" {
				out = append(out, t)
			}
		}
		return out
	}
	note := HandoffNote{Must: clean(wire.Must), Terms: clean(wire.Terms), Exact: clean(wire.Exact), Free: clean(wire.Free), Unsure: clean(wire.Unsure)}
	if err := note.Validate(); err != nil {
		return HandoffReply{}, fmt.Errorf("%w: %v", ErrHandoffReplyUnusable, err)
	}
	if note.Empty() {
		return HandoffReply{}, fmt.Errorf("%w: không có mục nào", ErrHandoffReplyUnusable)
	}
	out := HandoffReply{Note: note, Terms: []GlossaryTerm{}}
	for _, g := range wire.Glossary {
		t := GlossaryTerm{Term: strings.TrimSpace(g.Term), Say: strings.TrimSpace(g.Say), Draw: strings.TrimSpace(g.Draw), Avoid: strings.TrimSpace(g.Avoid)}
		if err := t.Validate(); err != nil {
			return HandoffReply{}, fmt.Errorf("%w: %v", ErrHandoffReplyUnusable, err)
		}
		out.Terms = append(out.Terms, t)
	}
	return out, nil
}
