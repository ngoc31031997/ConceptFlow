package domain

import (
	"errors"
	"fmt"
	"strings"
)

// The idea step's one review unit: idea:main.
const (
	UnitKindIdea = "idea"
	IdeaUnitKey  = "main"
)

// Idea turns a topic into a question worth a video: the question at the
// centre, why it is interesting, and what is and is not known about it. It
// holds no audience, length or tone; the creative brief decides those.
type Idea struct {
	Question string `json:"question"`
	Why      string `json:"why"`
	Known    string `json:"known"`
	Unknown  string `json:"unknown"`
}

// IdeaFields are the idea's field names in the order the screen and the
// prompts show them.
var IdeaFields = []string{"question", "why", "known", "unknown"}

// ideaFieldLabels names each field in the text later steps read and in
// messages the Creator reads.
var ideaFieldLabels = map[string]string{
	"question": "Câu hỏi trung tâm cần trả lời",
	"why":      "Vì sao câu hỏi này thú vị",
	"known":    "Điều đã biết",
	"unknown":  "Điều chưa biết",
}

// IsIdeaField reports whether name is one of IdeaFields.
func IsIdeaField(name string) bool {
	_, ok := ideaFieldLabels[name]
	return ok
}

// Field returns the value of the named field ("" for an unknown name).
func (i Idea) Field(name string) string {
	switch name {
	case "question":
		return i.Question
	case "why":
		return i.Why
	case "known":
		return i.Known
	case "unknown":
		return i.Unknown
	}
	return ""
}

// WithField returns a copy of i with the named field set; an unknown name
// leaves i as it is.
func (i Idea) WithField(name, value string) Idea {
	switch name {
	case "question":
		i.Question = value
	case "why":
		i.Why = value
	case "known":
		i.Known = value
	case "unknown":
		i.Unknown = value
	}
	return i
}

// MissingFields lists the fields that are empty, in IdeaFields order.
func (i Idea) MissingFields() []string {
	var missing []string
	for _, f := range IdeaFields {
		if strings.TrimSpace(i.Field(f)) == "" {
			missing = append(missing, f)
		}
	}
	return missing
}

// MissingFieldLabels names the empty fields as the screen labels them.
func (i Idea) MissingFieldLabels() string {
	missing := i.MissingFields()
	labels := make([]string, 0, len(missing))
	for _, f := range missing {
		labels = append(labels, ideaFieldLabels[f])
	}
	return strings.Join(labels, "; ")
}

// KeepIdeaCreatorFields returns written with every field in creatorFields
// taken from current: what the Creator wrote survives an AI redo verbatim.
func KeepIdeaCreatorFields(written, current Idea, creatorFields []string) Idea {
	for _, f := range creatorFields {
		written = written.WithField(f, current.Field(f))
	}
	return written
}

// ErrIdeaReplyUnusable marks a model reply the idea cannot be read from.
var ErrIdeaReplyUnusable = errors.New("câu trả lời của AI không dùng được làm ý tưởng")

// ParseIdeaReply reads the idea out of the JSON object llm-service returned
// for the reply. Every field must be there
// and non-empty; a reply missing one is an error naming the fields, so a
// half-written idea is never stored.
func ParseIdeaReply(reply string) (Idea, error) {
	raw, err := replyObject(reply)
	if err != nil {
		return Idea{}, fmt.Errorf("%w: %v", ErrIdeaReplyUnusable, err)
	}
	var idea Idea
	for _, f := range IdeaFields {
		if s, ok := raw[f].(string); ok {
			idea = idea.WithField(f, strings.TrimSpace(s))
		}
	}
	if len(idea.MissingFields()) > 0 {
		return Idea{}, fmt.Errorf("%w: thiếu ô %s", ErrIdeaReplyUnusable, idea.MissingFieldLabels())
	}
	return idea, nil
}

// IdeaSection is the idea as later steps read it, one labelled line per field.
func IdeaSection(i Idea) string {
	lines := make([]string, 0, len(IdeaFields))
	for _, f := range IdeaFields {
		lines = append(lines, ideaFieldLabels[f]+": "+i.Field(f))
	}
	return strings.Join(lines, "\n")
}
