package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ProjectIdea is what the AI makes of the text the Creator typed on the first
// step: the question at the centre, what makes people curious about it and how
// far the video goes. The text the Creator
// typed is kept by the orchestrator with the project, not here.
type ProjectIdea struct {
	CentralQuestion string   `json:"central_question"`
	CuriosityPoints []string `json:"curiosity_points"`
	Scope           string   `json:"scope"`
}

// The idea's field names, in the order the screen and the prompts show them.
const (
	IdeaFieldCentralQuestion = "central_question"
	IdeaFieldCuriosityPoints = "curiosity_points"
	IdeaFieldScope           = "scope"
)

// ProjectIdeaFields lists the field names in screen order.
var ProjectIdeaFields = []string{IdeaFieldCentralQuestion, IdeaFieldCuriosityPoints, IdeaFieldScope}

var projectIdeaFieldLabels = map[string]string{
	IdeaFieldCentralQuestion: "Câu hỏi trung tâm",
	IdeaFieldCuriosityPoints: "Điểm gây tò mò",
	IdeaFieldScope:           "Phạm vi",
}

// MaxCuriosityPoints caps the list so a runaway reply cannot fill the screen.
const MaxCuriosityPoints = 10

// IsProjectIdeaField reports whether name is one of ProjectIdeaFields.
func IsProjectIdeaField(name string) bool {
	_, ok := projectIdeaFieldLabels[name]
	return ok
}

// Normalized trims every text and drops empty and repeated curiosity points,
// so what is stored and hashed does not depend on stray spaces.
func (i ProjectIdea) Normalized() ProjectIdea {
	out := ProjectIdea{
		CentralQuestion: strings.TrimSpace(i.CentralQuestion),
		Scope:           strings.TrimSpace(i.Scope),
		CuriosityPoints: []string{},
	}
	seen := map[string]bool{}
	for _, p := range i.CuriosityPoints {
		p = strings.TrimSpace(p)
		if p == "" || seen[strings.ToLower(p)] {
			continue
		}
		seen[strings.ToLower(p)] = true
		out.CuriosityPoints = append(out.CuriosityPoints, p)
	}
	return out
}

// MissingFields lists the fields that hold nothing usable, in screen order.
func (i ProjectIdea) MissingFields() []string {
	n := i.Normalized()
	var missing []string
	if n.CentralQuestion == "" {
		missing = append(missing, IdeaFieldCentralQuestion)
	}
	if len(n.CuriosityPoints) == 0 {
		missing = append(missing, IdeaFieldCuriosityPoints)
	}
	if n.Scope == "" {
		missing = append(missing, IdeaFieldScope)
	}
	return missing
}

// MissingFieldLabels names the missing fields the way the Creator reads them.
func (i ProjectIdea) MissingFieldLabels() string {
	var labels []string
	for _, f := range i.MissingFields() {
		labels = append(labels, projectIdeaFieldLabels[f])
	}
	return strings.Join(labels, ", ")
}

// Validate is the machine check an idea passes before the Creator may approve
// it: it matches the schema, every field holds something, and the curiosity
// points are a list of distinct, non-empty lines within the cap and each of a
// sentence's length, not a paragraph.
func (i ProjectIdea) Validate() error {
	if missing := i.MissingFields(); len(missing) > 0 {
		return fmt.Errorf("%w: còn trống %s", ErrProjectIdeaInvalid, i.MissingFieldLabels())
	}
	n := i.Normalized()
	if len(n.CuriosityPoints) > MaxCuriosityPoints {
		return fmt.Errorf("%w: tối đa %d điểm gây tò mò", ErrProjectIdeaInvalid, MaxCuriosityPoints)
	}
	for _, p := range n.CuriosityPoints {
		if len([]rune(p)) > 300 {
			return fmt.Errorf("%w: mỗi điểm gây tò mò tối đa 300 ký tự", ErrProjectIdeaInvalid)
		}
	}
	return nil
}

// Hash identifies the idea's content: the same fields give the same hash, so a
// review can tell whether the idea changed after it was written.
func (i ProjectIdea) Hash() string {
	// Marshalling a struct of strings cannot fail.
	raw, _ := json.Marshal(i.Normalized())
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// Field returns the named text field; the list is read through
// CuriosityPoints. An unknown name or the list gives "".
func (i ProjectIdea) Field(name string) string {
	switch name {
	case IdeaFieldCentralQuestion:
		return i.CentralQuestion
	case IdeaFieldScope:
		return i.Scope
	}
	return ""
}

// Section is the idea as a prompt reads it, one labelled line per field.
func (i ProjectIdea) Section() string {
	n := i.Normalized()
	lines := []string{
		projectIdeaFieldLabels[IdeaFieldCentralQuestion] + ": " + n.CentralQuestion,
		projectIdeaFieldLabels[IdeaFieldCuriosityPoints] + ":",
	}
	for _, p := range n.CuriosityPoints {
		lines = append(lines, "- "+p)
	}
	lines = append(lines, projectIdeaFieldLabels[IdeaFieldScope]+": "+n.Scope)
	return strings.Join(lines, "\n")
}

// IdeaStatus is where an idea stands in review.
type IdeaStatus string

const (
	IdeaPending  IdeaStatus = "pending"
	IdeaApproved IdeaStatus = "approved"
)

// StoredIdea is a project's idea with what review needs around it.
type StoredIdea struct {
	ProjectID string
	Idea      ProjectIdea
	// CreatorFields are the fields the Creator wrote or changed; the AI keeps
	// them when it writes the idea again.
	CreatorFields []string
	Status        IdeaStatus
	RedoCount     int
	// Language is the content language the AI wrote the idea in, and InputHash
	// identifies the text it wrote the idea from. They tell the screen when the
	// Creator changed either since.
	Language   string
	InputHash  string
	ApprovedAt *time.Time
	UpdatedAt  time.Time
}

// HasCreatorField reports whether the Creator owns the named field.
func (s StoredIdea) HasCreatorField(name string) bool {
	for _, f := range s.CreatorFields {
		if f == name {
			return true
		}
	}
	return false
}

// SanitizeIdeaText cleans text the Creator typed before it is stored or handed
// to the AI. The AI answers in JSON and tends to copy a straight double quote or
// a backslash from its input into a JSON string, which then cannot be read. So a
// straight double quote becomes a typographic one (“ at the start or after a
// space, bracket or dash, ” anywhere else), a backslash becomes a slash, and
// control characters other than a new line and a tab are dropped. Everything
// else, Vietnamese letters and single quotes included, is kept. The result is
// trimmed. The orchestrator applies the same rule to the text it keeps.
func SanitizeIdeaText(text string) string {
	var out []rune
	for _, r := range strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n") {
		switch {
		case r == '"':
			if len(out) == 0 || strings.ContainsRune(" \t\n([{“‘—–-", out[len(out)-1]) {
				out = append(out, '“')
			} else {
				out = append(out, '”')
			}
		case r == '\\':
			out = append(out, '/')
		case r == '\n' || r == '\t' || r >= ' ' && r != 0x7f:
			out = append(out, r)
		}
	}
	return strings.TrimSpace(string(out))
}

// HashInput identifies the text the Creator typed once it is cleaned, so text
// kept before the cleaning existed and the same text kept after it match.
func HashInput(text string) string {
	sum := sha256.Sum256([]byte(SanitizeIdeaText(text)))
	return hex.EncodeToString(sum[:])
}

// KeepIdeaCreatorFields returns written with every field the Creator owns
// copied back from current.
func KeepProjectIdeaCreatorFields(written, current ProjectIdea, creatorFields []string) ProjectIdea {
	for _, f := range creatorFields {
		switch f {
		case IdeaFieldCentralQuestion:
			written.CentralQuestion = current.CentralQuestion
		case IdeaFieldCuriosityPoints:
			written.CuriosityPoints = current.CuriosityPoints
		case IdeaFieldScope:
			written.Scope = current.Scope
		}
	}
	return written
}

// Errors of the idea step.
var (
	// ErrProjectIdeaInvalid marks an idea that fails the machine check.
	ErrProjectIdeaInvalid = errors.New("ý tưởng chưa hợp lệ")
	// ErrProjectIdeaReplyUnusable marks a model reply the idea cannot be read from.
	ErrProjectIdeaReplyUnusable = errors.New("câu trả lời của AI không dùng được làm ý tưởng")
	// ErrIdeaReviewReplyUnusable marks a model reply the review cannot be read from.
	ErrIdeaReviewReplyUnusable = errors.New("câu trả lời của AI không dùng được làm đánh giá ý tưởng")
	// ErrIdeaNotFound is returned for a project that has no idea yet.
	ErrIdeaNotFound = errors.New("project chưa có ý tưởng")
	// ErrIdeaNoInput is returned when the project has no text to write an idea from.
	ErrIdeaNoInput = errors.New("project chưa có nội dung ý tưởng")
)

// ParseProjectIdeaReply reads the idea out of a model reply: the first JSON
// object in it, with or without a ```json fence around it. Every field must
// be there and hold something, so a half-written idea is never stored.
func ParseProjectIdeaReply(reply string) (ProjectIdea, error) {
	raw, err := replyObject(reply)
	if err != nil {
		return ProjectIdea{}, fmt.Errorf("%w: %v", ErrProjectIdeaReplyUnusable, err)
	}
	var idea ProjectIdea
	idea.CentralQuestion, _ = raw[IdeaFieldCentralQuestion].(string)
	idea.Scope, _ = raw[IdeaFieldScope].(string)
	if list, ok := raw[IdeaFieldCuriosityPoints].([]any); ok {
		for _, item := range list {
			if s, ok := item.(string); ok {
				idea.CuriosityPoints = append(idea.CuriosityPoints, s)
			}
		}
	}
	idea = idea.Normalized()
	if err := idea.Validate(); err != nil {
		return ProjectIdea{}, fmt.Errorf("%w: %v", ErrProjectIdeaReplyUnusable, strings.TrimPrefix(err.Error(), ErrProjectIdeaInvalid.Error()+": "))
	}
	return idea, nil
}

// IdeaRating is how an aspect of the idea fares in the AI's review.
type IdeaRating string

const (
	RatingGood IdeaRating = "good"
	RatingFair IdeaRating = "fair"
	RatingWeak IdeaRating = "weak"
)

// IdeaAspect is the AI's reading of one aspect of the idea.
type IdeaAspect struct {
	Rating  IdeaRating `json:"rating"`
	Comment string     `json:"comment"`
}

// IdeaReview is the AI's reading of an idea before the Creator approves it:
// whether the question is clear, whether it can be researched and whether the
// scope fits one video. It informs the Creator's decision and never blocks it.
type IdeaReview struct {
	Clarity         IdeaAspect `json:"clarity"`
	Researchability IdeaAspect `json:"researchability"`
	ScopeFit        IdeaAspect `json:"scope_fit"`
	// Unclear lists what is still unclear; Suggestions lists how to improve it.
	Unclear     []string `json:"unclear"`
	Suggestions []string `json:"suggestions"`
	// IdeaHash is the hash of the idea this review read.
	IdeaHash  string    `json:"idea_hash"`
	CreatedAt time.Time `json:"created_at"`
}

// ParseIdeaReviewReply reads the review out of a model reply. Each aspect needs
// a rating of good, fair or weak and a comment.
func ParseIdeaReviewReply(reply string) (IdeaReview, error) {
	raw, err := replyObject(reply)
	if err != nil {
		return IdeaReview{}, fmt.Errorf("%w: %v", ErrIdeaReviewReplyUnusable, err)
	}
	var review IdeaReview
	for name, dst := range map[string]*IdeaAspect{
		"clarity": &review.Clarity, "researchability": &review.Researchability, "scope_fit": &review.ScopeFit,
	} {
		obj, ok := raw[name].(map[string]any)
		if !ok {
			return IdeaReview{}, fmt.Errorf("%w: thiếu mục %s", ErrIdeaReviewReplyUnusable, name)
		}
		rating, _ := obj["rating"].(string)
		comment, _ := obj["comment"].(string)
		switch IdeaRating(rating) {
		case RatingGood, RatingFair, RatingWeak:
		default:
			return IdeaReview{}, fmt.Errorf("%w: mục %s có mức %q lạ", ErrIdeaReviewReplyUnusable, name, rating)
		}
		if strings.TrimSpace(comment) == "" {
			return IdeaReview{}, fmt.Errorf("%w: mục %s chưa có nhận xét", ErrIdeaReviewReplyUnusable, name)
		}
		*dst = IdeaAspect{Rating: IdeaRating(rating), Comment: strings.TrimSpace(comment)}
	}
	review.Unclear = stringList(raw["unclear"])
	review.Suggestions = stringList(raw["suggestions"])
	return review, nil
}

// stringList reads a JSON array of strings, dropping blanks; anything else is
// an empty list.
func stringList(v any) []string {
	out := []string{}
	list, _ := v.([]any)
	for _, item := range list {
		if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
			out = append(out, strings.TrimSpace(s))
		}
	}
	return out
}
