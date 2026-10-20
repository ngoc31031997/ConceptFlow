package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// The kinds of review unit the story step stores.
const (
	// UnitKindStory is the gate of the step: approving it approves the whole story.
	UnitKindStory = "story"
	// StoryUnitKey is the only key of the gate unit.
	StoryUnitKey = "main"
	// UnitKindStoryField is one field of the story frame, keyed by StoryFieldKeys.
	UnitKindStoryField = "story_field"
	// UnitKindStoryApproach is one way of telling the story the AI proposed,
	// keyed A1, A2...
	UnitKindStoryApproach = "story_approach"
)

// StoryFieldKeys are the story frame's fields in the order the screen shows
// them, and the keys of their units.
var StoryFieldKeys = []string{"question", "promise", "chain", "payoff", "logline", "summary"}

// storyFieldLabels names each field for the Creator.
var storyFieldLabels = map[string]string{
	"question": "Câu hỏi mở đầu",
	"promise":  "Lời hứa với khán giả",
	"chain":    "Chuỗi nhân quả",
	"payoff":   "Kết luận · payoff",
	"logline":  "Logline",
	"summary":  "Tóm tắt",
}

// The only fields that cite facts.
const (
	storyFieldChain  = "chain"
	storyFieldPayoff = "payoff"
)

// TextFieldName is the one hand-editable field of a story field unit.
const TextFieldName = "text"

// MaxStoryApproaches is the most approaches one proposal may hold.
const MaxStoryApproaches = 6

// MaxStoryChain is the most steps the chain of cause and effect may hold.
const MaxStoryChain = 12

// IsStoryFieldKey reports whether key is one of StoryFieldKeys.
func IsStoryFieldKey(key string) bool {
	_, ok := storyFieldLabels[key]
	return ok
}

// StoryFieldLabel is the name the screen gives a field ("" for an unknown one).
func StoryFieldLabel(key string) string { return storyFieldLabels[key] }

// ErrStoryReplyUnusable marks a model reply a story field cannot be read from.
var ErrStoryReplyUnusable = errors.New("câu trả lời của AI không dùng được làm khung câu chuyện")

// ErrApproachReplyUnusable marks a model reply the approaches cannot be read from.
var ErrApproachReplyUnusable = errors.New("câu trả lời của AI không dùng được làm danh sách góc tiếp cận")

// StoryField is the content of a story field unit. Text is what the Creator
// reads and edits; the chain holds one step per line and the chain and the
// payoff end a line with the facts it rests on as "[F01, F02]". Facts lists
// those codes, each once, and is derived from Text.
type StoryField struct {
	Text  string   `json:"text"`
	Facts []string `json:"facts"`
}

// StoryGate is the content of the gate unit: the approach the story follows
// and the facts it uses besides the ones it must state.
type StoryGate struct {
	// ApproachKey is the story_approach unit the story follows; "" lets the AI choose.
	ApproachKey string `json:"approach_key"`
	// UsedFacts lists the facts the story uses besides the ones marked as
	// having to be stated.
	UsedFacts []string `json:"used_facts"`
}

// StoryApproach is one way of telling the story the AI proposed from the fact
// table: a kind of video and why it fits.
type StoryApproach struct {
	// Code is the kind's code in the Kiểu video table, or a new one the AI proposes.
	Code  string `json:"code"`
	Name  string `json:"name"`
	Angle string `json:"angle"`
	Why   string `json:"why"`
	// Refs are the facts that make this approach fit.
	Refs []string `json:"refs"`
	// Open is a sample opening line.
	Open string `json:"open"`
	// IsNew is true when Code is not in the Kiểu video table.
	IsNew bool `json:"is_new"`
	// WhenToUse and Playbook describe a new kind, so it can be saved to the table.
	WhenToUse string `json:"when_to_use"`
	Playbook  string `json:"playbook"`
	// ArchetypeID is the table row a new kind was saved as; "" until saved.
	ArchetypeID string `json:"archetype_id"`
}

var trailingRefs = regexp.MustCompile(`\s*\[([^\[\]]*)\]\s*$`)

// FormatRefLine writes a text with the facts it cites as a trailing
// "[F01, F02]"; a text citing none is written as it is.
func FormatRefLine(text string, refs []string) string {
	if len(refs) == 0 {
		return text
	}
	return text + " [" + strings.Join(refs, ", ") + "]"
}

// ParseRefLine reads FormatRefLine's form back: the text and the fact codes in
// the trailing brackets. A bracket holding anything but fact codes stays in the
// text. A code known() does not know is an error naming it.
func ParseRefLine(line string, known func(id string) bool) (string, []string, error) {
	line = strings.TrimSpace(line)
	m := trailingRefs.FindStringSubmatchIndex(line)
	if m == nil {
		return line, []string{}, nil
	}
	var refs []string
	for _, part := range strings.Split(line[m[2]:m[3]], ",") {
		id, ok := NormalizeFactID(strings.TrimSpace(part))
		if !ok {
			return line, []string{}, nil
		}
		if known != nil && !known(id) {
			return "", nil, fmt.Errorf("dữ kiện %s không có trong bảng dữ kiện", id)
		}
		refs = append(refs, id)
	}
	return strings.TrimSpace(line[:m[0]]), refs, nil
}

// uniqueFacts returns ids in first-seen order, each once.
func uniqueFacts(lists ...[]string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, ids := range lists {
		for _, id := range ids {
			if !seen[id] {
				seen[id] = true
				out = append(out, id)
			}
		}
	}
	return out
}

// NewStoryField builds the content of the field key from the text the Creator
// typed. The chain is read one step per line and the payoff as one line; both
// may end in "[F01, F02]". known tells which fact codes exist, and a citation
// of any other is an error. An empty text gives an empty field.
func NewStoryField(key, text string, known func(id string) bool) (StoryField, error) {
	text = strings.TrimSpace(text)
	switch key {
	case storyFieldChain:
		var lines []string
		var lists [][]string
		for _, line := range strings.Split(text, "\n") {
			if strings.TrimSpace(line) == "" {
				continue
			}
			body, refs, err := ParseRefLine(line, known)
			if err != nil {
				return StoryField{}, err
			}
			lines = append(lines, FormatRefLine(body, refs))
			lists = append(lists, refs)
		}
		if len(lines) > MaxStoryChain {
			return StoryField{}, fmt.Errorf("chuỗi nhân quả tối đa %d bước", MaxStoryChain)
		}
		return StoryField{Text: strings.Join(lines, "\n"), Facts: uniqueFacts(lists...)}, nil
	case storyFieldPayoff:
		body, refs, err := ParseRefLine(text, known)
		if err != nil {
			return StoryField{}, err
		}
		return StoryField{Text: FormatRefLine(body, refs), Facts: uniqueFacts(refs)}, nil
	}
	if !IsStoryFieldKey(key) {
		return StoryField{}, fmt.Errorf("khung câu chuyện không có ô %q", key)
	}
	return StoryField{Text: text, Facts: []string{}}, nil
}

// NewStoryFact is a fact the AI listed from its own knowledge when the research
// step gave none.
type NewStoryFact struct {
	ID      string
	ClaimVI string
	Type    FactType
	Note    string
}

// StoryReply is what a story reply holds: the fields asked for, the facts the
// story uses, and the facts the AI listed.
type StoryReply struct {
	Fields    map[string]StoryField
	UsedFacts []string
	// Facts are the facts the AI listed; empty when the fact table was given.
	Facts []NewStoryFact
}

type storyReplyWire struct {
	Question string `json:"question"`
	Promise  string `json:"promise"`
	Chain    []struct {
		Text  string   `json:"text"`
		Facts []string `json:"facts"`
	} `json:"chain"`
	Payoff      string   `json:"payoff"`
	PayoffFacts []string `json:"payoff_facts"`
	Logline     string   `json:"logline"`
	Summary     string   `json:"summary"`
	UsedFacts   []string `json:"used_facts"`
	Facts       []struct {
		ID      string `json:"id"`
		ClaimVI string `json:"claim_vi"`
		Type    string `json:"type"`
		Note    string `json:"note"`
	} `json:"facts"`
}

// jsonObjectIn returns the text from the first "{" to the last "}" of a
// reply, with or without a ```json fence around it.
func jsonObjectIn(reply string) (string, bool) {
	start := strings.Index(reply, "{")
	end := strings.LastIndex(reply, "}")
	if start < 0 || end <= start {
		return "", false
	}
	return reply[start : end+1], true
}

// ParseStoryReply reads the fields in want out of a model reply, each of which
// must be there and non-empty. Facts the AI listed are read first (one whose
// type is not a FactType is an error), and every code a field cites must be
// one of known or one of those: a half-written or mis-cited reply is never
// stored.
func ParseStoryReply(reply string, want []string, known func(id string) bool) (StoryReply, error) {
	obj, ok := jsonObjectIn(reply)
	if !ok {
		return StoryReply{}, fmt.Errorf("%w: không có khối JSON", ErrStoryReplyUnusable)
	}
	var wire storyReplyWire
	if err := json.Unmarshal([]byte(obj), &wire); err != nil {
		return StoryReply{}, fmt.Errorf("%w: khối JSON không đọc được (%v)", ErrStoryReplyUnusable, err)
	}
	out := StoryReply{Fields: map[string]StoryField{}}
	listed := map[string]bool{}
	for i, f := range wire.Facts {
		claim := strings.TrimSpace(f.ClaimVI)
		if claim == "" {
			continue
		}
		t := FactType(strings.TrimSpace(f.Type))
		if !ValidFactType(t) {
			return StoryReply{}, fmt.Errorf("%w: dữ kiện thứ %d có loại %q không hợp lệ", ErrStoryReplyUnusable, i+1, f.Type)
		}
		id, ok := NormalizeFactID(f.ID)
		if !ok || listed[id] {
			return StoryReply{}, fmt.Errorf("%w: dữ kiện thứ %d có mã %q không hợp lệ hoặc trùng", ErrStoryReplyUnusable, i+1, f.ID)
		}
		listed[id] = true
		out.Facts = append(out.Facts, NewStoryFact{ID: id, ClaimVI: claim, Type: t, Note: strings.TrimSpace(f.Note)})
	}
	exists := func(id string) bool { return listed[id] || (known != nil && known(id)) }
	cite := func(raw []string) ([]string, error) {
		refs := []string{}
		for _, r := range raw {
			id, ok := NormalizeFactID(strings.TrimSpace(r))
			if !ok || !exists(id) {
				return nil, fmt.Errorf("%w: trích dữ kiện %q không có trong bảng dữ kiện", ErrStoryReplyUnusable, r)
			}
			refs = append(refs, id)
		}
		return refs, nil
	}
	var err error
	if out.UsedFacts, err = cite(wire.UsedFacts); err != nil {
		return StoryReply{}, err
	}
	for _, key := range want {
		field, err := fieldFromWire(key, wire, cite)
		if err != nil {
			return StoryReply{}, err
		}
		if strings.TrimSpace(field.Text) == "" {
			return StoryReply{}, fmt.Errorf("%w: thiếu ô %s", ErrStoryReplyUnusable, storyFieldLabels[key])
		}
		out.Fields[key] = field
	}
	return out, nil
}

// fieldFromWire builds one field out of the decoded reply.
func fieldFromWire(key string, wire storyReplyWire, cite func([]string) ([]string, error)) (StoryField, error) {
	switch key {
	case "question":
		return NewStoryField(key, wire.Question, nil)
	case "promise":
		return NewStoryField(key, wire.Promise, nil)
	case "logline":
		return NewStoryField(key, wire.Logline, nil)
	case "summary":
		return NewStoryField(key, wire.Summary, nil)
	case storyFieldPayoff:
		refs, err := cite(wire.PayoffFacts)
		if err != nil {
			return StoryField{}, err
		}
		body := strings.TrimSpace(wire.Payoff)
		if body == "" {
			return StoryField{}, nil
		}
		return StoryField{Text: FormatRefLine(body, refs), Facts: uniqueFacts(refs)}, nil
	case storyFieldChain:
		var lines []string
		var lists [][]string
		for _, step := range wire.Chain {
			body := strings.TrimSpace(step.Text)
			if body == "" {
				continue
			}
			refs, err := cite(step.Facts)
			if err != nil {
				return StoryField{}, err
			}
			lines = append(lines, FormatRefLine(body, refs))
			lists = append(lists, refs)
		}
		if len(lines) > MaxStoryChain {
			return StoryField{}, fmt.Errorf("%w: chuỗi nhân quả dài hơn %d bước", ErrStoryReplyUnusable, MaxStoryChain)
		}
		return StoryField{Text: strings.Join(lines, "\n"), Facts: uniqueFacts(lists...)}, nil
	}
	return StoryField{}, fmt.Errorf("khung câu chuyện không có ô %q", key)
}

type approachWire struct {
	Code      string   `json:"code"`
	Name      string   `json:"name"`
	Angle     string   `json:"angle"`
	Why       string   `json:"why"`
	Refs      []string `json:"refs"`
	Open      string   `json:"open"`
	WhenToUse string   `json:"when_to_use"`
	Playbook  string   `json:"playbook"`
}

// ParseApproachesReply reads the proposed approaches out of a model reply.
// existingCodes are the codes of the Kiểu video table: an approach whose code
// is not among them is a new kind and must carry when_to_use and playbook so
// that it can be saved. Citations of facts known() does not know are an error.
func ParseApproachesReply(reply string, existingCodes []string, known func(id string) bool) ([]StoryApproach, error) {
	obj, ok := jsonObjectIn(reply)
	if !ok {
		return nil, fmt.Errorf("%w: không có khối JSON", ErrApproachReplyUnusable)
	}
	var wire struct {
		Approaches []approachWire `json:"approaches"`
	}
	if err := json.Unmarshal([]byte(obj), &wire); err != nil {
		return nil, fmt.Errorf("%w: khối JSON không đọc được (%v)", ErrApproachReplyUnusable, err)
	}
	exists := map[string]bool{}
	for _, c := range existingCodes {
		exists[strings.ToUpper(c)] = true
	}
	var out []StoryApproach
	seen := map[string]bool{}
	for i, a := range wire.Approaches {
		approach := StoryApproach{
			Code: strings.ToUpper(strings.TrimSpace(a.Code)), Name: strings.TrimSpace(a.Name), Angle: strings.TrimSpace(a.Angle),
			Why: strings.TrimSpace(a.Why), Open: strings.TrimSpace(a.Open),
			WhenToUse: strings.TrimSpace(a.WhenToUse), Playbook: strings.TrimSpace(a.Playbook), Refs: []string{},
		}
		if approach.Code == "" || approach.Name == "" || approach.Angle == "" || approach.Why == "" || approach.Open == "" {
			return nil, fmt.Errorf("%w: góc thứ %d thiếu ô", ErrApproachReplyUnusable, i+1)
		}
		if seen[approach.Code] {
			continue
		}
		seen[approach.Code] = true
		for _, r := range a.Refs {
			id, ok := NormalizeFactID(strings.TrimSpace(r))
			if !ok || (known != nil && !known(id)) {
				return nil, fmt.Errorf("%w: góc %s trích dữ kiện %q không có trong bảng dữ kiện", ErrApproachReplyUnusable, approach.Code, r)
			}
			approach.Refs = append(approach.Refs, id)
		}
		approach.IsNew = !exists[approach.Code]
		if approach.IsNew && (approach.WhenToUse == "" || approach.Playbook == "") {
			return nil, fmt.Errorf("%w: kiểu mới %s thiếu when_to_use hoặc playbook", ErrApproachReplyUnusable, approach.Code)
		}
		out = append(out, approach)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%w: không có góc nào", ErrApproachReplyUnusable)
	}
	if len(out) > MaxStoryApproaches {
		out = out[:MaxStoryApproaches]
	}
	return out, nil
}
