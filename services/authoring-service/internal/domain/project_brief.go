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

// ProjectBrief is the second step's output: who the video is for, how long it
// runs, how it sounds, what the viewer should come away understanding and what
// the video must not say. The AI writes it from the approved idea.
type ProjectBrief struct {
	Audience           string   `json:"audience"`
	DurationSeconds    int      `json:"duration_seconds"`
	Tone               string   `json:"tone"`
	LearningObjectives []string `json:"learning_objectives"`
	Avoid              []string `json:"avoid"`
}

// The brief's field names, in the order the screen and the prompts show them.
const (
	BriefFieldAudience           = "audience"
	BriefFieldDuration           = "duration_seconds"
	BriefFieldTone               = "tone"
	BriefFieldLearningObjectives = "learning_objectives"
	BriefFieldAvoid              = "avoid"
)

// ProjectBriefFields lists the field names in screen order.
var ProjectBriefFields = []string{
	BriefFieldAudience, BriefFieldDuration, BriefFieldTone, BriefFieldLearningObjectives, BriefFieldAvoid,
}

var projectBriefFieldLabels = map[string]string{
	BriefFieldAudience:           "Khán giả",
	BriefFieldDuration:           "Thời lượng",
	BriefFieldTone:               "Giọng điệu",
	BriefFieldLearningObjectives: "Mục tiêu học tập",
	BriefFieldAvoid:              "Cần tránh",
}

// Limits of the brief's lists; a video of a few minutes cannot teach more
// objectives or guard against more pitfalls than this.
const (
	MaxLearningObjectives = 6
	MaxAvoidItems         = 8
	// MaxBriefItemRunes keeps each list item a sentence, not a paragraph.
	MaxBriefItemRunes = 300
)

// The length a brief may set, in seconds: a short runs one to two minutes, a
// long video from just over two minutes to half an hour.
const (
	ShortMinSeconds = 60
	ShortMaxSeconds = 120
	LongMinSeconds  = 121
	LongMaxSeconds  = 1800
)

// DurationBounds is the range of seconds a video of the given form may run.
func DurationBounds(mode VideoOutputMode) (min, max int) {
	if mode == ModeShortOnly {
		return ShortMinSeconds, ShortMaxSeconds
	}
	return LongMinSeconds, LongMaxSeconds
}

// IsProjectBriefField reports whether name is one of ProjectBriefFields.
func IsProjectBriefField(name string) bool {
	_, ok := projectBriefFieldLabels[name]
	return ok
}

// Normalized trims every text and drops empty and repeated list items, so what
// is stored and hashed does not depend on stray spaces.
func (b ProjectBrief) Normalized() ProjectBrief {
	return ProjectBrief{
		Audience:           strings.TrimSpace(b.Audience),
		DurationSeconds:    b.DurationSeconds,
		Tone:               strings.TrimSpace(b.Tone),
		LearningObjectives: DistinctLines(b.LearningObjectives),
		Avoid:              DistinctLines(b.Avoid),
	}
}

// DistinctLines trims each line and drops blanks and case-insensitive repeats,
// keeping the first occurrence's order.
func DistinctLines(lines []string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" || seen[strings.ToLower(l)] {
			continue
		}
		seen[strings.ToLower(l)] = true
		out = append(out, l)
	}
	return out
}

// MissingFields lists the fields that hold nothing usable, in screen order.
func (b ProjectBrief) MissingFields() []string {
	n := b.Normalized()
	var missing []string
	if n.Audience == "" {
		missing = append(missing, BriefFieldAudience)
	}
	if n.DurationSeconds <= 0 {
		missing = append(missing, BriefFieldDuration)
	}
	if n.Tone == "" {
		missing = append(missing, BriefFieldTone)
	}
	if len(n.LearningObjectives) == 0 {
		missing = append(missing, BriefFieldLearningObjectives)
	}
	if len(n.Avoid) == 0 {
		missing = append(missing, BriefFieldAvoid)
	}
	return missing
}

// MissingFieldLabels names the missing fields the way the Creator reads them.
func (b ProjectBrief) MissingFieldLabels() string {
	var labels []string
	for _, f := range b.MissingFields() {
		labels = append(labels, projectBriefFieldLabels[f])
	}
	return strings.Join(labels, ", ")
}

// Validate is the machine check a brief passes before the Creator may approve
// it: every field holds something, the length fits the video's form, and each
// list stays within its cap with items of a sentence's length.
func (b ProjectBrief) Validate(mode VideoOutputMode) error {
	if missing := b.MissingFields(); len(missing) > 0 {
		return fmt.Errorf("%w: còn trống %s", ErrProjectBriefInvalid, b.MissingFieldLabels())
	}
	n := b.Normalized()
	if min, max := DurationBounds(mode); n.DurationSeconds < min || n.DurationSeconds > max {
		return fmt.Errorf("%w: thời lượng phải từ %d đến %d giây", ErrProjectBriefInvalid, min, max)
	}
	if err := checkBriefList(n.LearningObjectives, MaxLearningObjectives, BriefFieldLearningObjectives); err != nil {
		return err
	}
	return checkBriefList(n.Avoid, MaxAvoidItems, BriefFieldAvoid)
}

func checkBriefList(items []string, max int, field string) error {
	label := strings.ToLower(projectBriefFieldLabels[field])
	if len(items) > max {
		return fmt.Errorf("%w: tối đa %d dòng %s", ErrProjectBriefInvalid, max, label)
	}
	for _, it := range items {
		if len([]rune(it)) > MaxBriefItemRunes {
			return fmt.Errorf("%w: mỗi dòng %s tối đa %d ký tự", ErrProjectBriefInvalid, label, MaxBriefItemRunes)
		}
	}
	return nil
}

// Hash identifies the brief's content: the same fields give the same hash, so a
// review can tell whether the brief changed after it read it.
func (b ProjectBrief) Hash() string {
	// Marshalling a struct of strings and an int cannot fail.
	raw, _ := json.Marshal(b.Normalized())
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// FormatDuration writes seconds as the screen and the prompts show them:
// m:ss under an hour, h:mm:ss from an hour.
func FormatDuration(seconds int) string {
	if seconds < 0 {
		seconds = 0
	}
	h, m, s := seconds/3600, seconds%3600/60, seconds%60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

// Section is the brief as a prompt reads it, one labelled line per field and
// one line per list item.
func (b ProjectBrief) Section() string {
	n := b.Normalized()
	lines := []string{
		projectBriefFieldLabels[BriefFieldAudience] + ": " + n.Audience,
		fmt.Sprintf("%s: %d giây (%s)", projectBriefFieldLabels[BriefFieldDuration], n.DurationSeconds, FormatDuration(n.DurationSeconds)),
		projectBriefFieldLabels[BriefFieldTone] + ": " + n.Tone,
		projectBriefFieldLabels[BriefFieldLearningObjectives] + ":",
	}
	for i, o := range n.LearningObjectives {
		lines = append(lines, fmt.Sprintf("%d. %s", i+1, o))
	}
	lines = append(lines, projectBriefFieldLabels[BriefFieldAvoid]+":")
	for _, a := range n.Avoid {
		lines = append(lines, "- "+a)
	}
	return strings.Join(lines, "\n")
}

// BriefStatus is where a brief stands in review.
type BriefStatus string

const (
	BriefPending  BriefStatus = "pending"
	BriefApproved BriefStatus = "approved"
)

// StoredBrief is a project's brief with what review needs around it.
type StoredBrief struct {
	ProjectID string
	Brief     ProjectBrief
	// CreatorFields are the fields the Creator wrote or changed; the AI keeps
	// them when it writes the brief again.
	CreatorFields []string
	Status        BriefStatus
	RedoCount     int
	// Language is the content language the AI wrote the brief in, and IdeaHash
	// identifies the approved idea it wrote from, so the screen can tell when
	// the idea changed since.
	Language   string
	IdeaHash   string
	ApprovedAt *time.Time
	UpdatedAt  time.Time
}

// HasCreatorField reports whether the Creator owns the named field.
func (s StoredBrief) HasCreatorField(name string) bool {
	for _, f := range s.CreatorFields {
		if f == name {
			return true
		}
	}
	return false
}

// KeepProjectBriefCreatorFields returns written with every field the Creator
// owns copied back from current.
func KeepProjectBriefCreatorFields(written, current ProjectBrief, creatorFields []string) ProjectBrief {
	for _, f := range creatorFields {
		switch f {
		case BriefFieldAudience:
			written.Audience = current.Audience
		case BriefFieldDuration:
			written.DurationSeconds = current.DurationSeconds
		case BriefFieldTone:
			written.Tone = current.Tone
		case BriefFieldLearningObjectives:
			written.LearningObjectives = current.LearningObjectives
		case BriefFieldAvoid:
			written.Avoid = current.Avoid
		}
	}
	return written
}

// Errors of the brief step.
var (
	// ErrProjectBriefInvalid marks a brief that fails the machine check.
	ErrProjectBriefInvalid = errors.New("brief chưa hợp lệ")
	// ErrProjectBriefReplyUnusable marks a model reply the brief cannot be read from.
	ErrProjectBriefReplyUnusable = errors.New("câu trả lời của AI không dùng được làm brief")
	// ErrBriefReviewReplyUnusable marks a model reply the review cannot be read from.
	ErrBriefReviewReplyUnusable = errors.New("câu trả lời của AI không dùng được làm đánh giá brief")
	// ErrBriefNotFound is returned for a project that has no brief yet.
	ErrBriefNotFound = errors.New("project chưa có brief")
	// ErrBriefNeedsApprovedIdea is returned when the brief is asked for before
	// the first step's idea is approved.
	ErrBriefNeedsApprovedIdea = errors.New("cần duyệt ý tưởng ở bước 1 trước khi viết brief")
)

// ParseProjectBriefReply reads the brief out of a model reply: the first JSON
// object in it, with or without a ```json fence around it. Every field must be
// there and the brief must pass the machine check for the video's form, so a
// half-written brief is never stored.
func ParseProjectBriefReply(reply string, mode VideoOutputMode) (ProjectBrief, error) {
	raw, err := replyObject(reply)
	if err != nil {
		return ProjectBrief{}, fmt.Errorf("%w: %v", ErrProjectBriefReplyUnusable, err)
	}
	var brief ProjectBrief
	brief.Audience, _ = raw[BriefFieldAudience].(string)
	brief.Tone, _ = raw[BriefFieldTone].(string)
	// JSON numbers decode as float64; a fractional second is rounded down.
	if d, ok := raw[BriefFieldDuration].(float64); ok {
		brief.DurationSeconds = int(d)
	}
	brief.LearningObjectives = stringList(raw[BriefFieldLearningObjectives])
	brief.Avoid = stringList(raw[BriefFieldAvoid])
	brief = brief.Normalized()
	if err := brief.Validate(mode); err != nil {
		return ProjectBrief{}, fmt.Errorf("%w: %v", ErrProjectBriefReplyUnusable, strings.TrimPrefix(err.Error(), ErrProjectBriefInvalid.Error()+": "))
	}
	return brief, nil
}

// BriefReview is the AI's reading of a brief before the Creator approves it:
// whether the audience is clear, whether the length fits the idea's scope,
// whether the objectives are concrete and inside the scope, and whether the
// avoid list guards against the likely misreadings. It informs the Creator's
// decision and never blocks it.
type BriefReview struct {
	AudienceFit IdeaAspect `json:"audience_fit"`
	DurationFit IdeaAspect `json:"duration_fit"`
	Objectives  IdeaAspect `json:"objectives"`
	AvoidList   IdeaAspect `json:"avoid_list"`
	// Unclear lists what is still unclear; Suggestions lists how to improve it.
	Unclear     []string `json:"unclear"`
	Suggestions []string `json:"suggestions"`
	// BriefHash is the hash of the brief this review read.
	BriefHash string    `json:"brief_hash"`
	CreatedAt time.Time `json:"created_at"`
}

// briefReviewAspects are the review's aspect keys in the order the screen shows them.
var briefReviewAspects = []string{"audience_fit", "duration_fit", "objectives", "avoid_list"}

// ParseBriefReviewReply reads the review out of a model reply. Each aspect
// needs a rating of good, fair or weak and a comment.
func ParseBriefReviewReply(reply string) (BriefReview, error) {
	raw, err := replyObject(reply)
	if err != nil {
		return BriefReview{}, fmt.Errorf("%w: %v", ErrBriefReviewReplyUnusable, err)
	}
	var review BriefReview
	targets := map[string]*IdeaAspect{
		"audience_fit": &review.AudienceFit, "duration_fit": &review.DurationFit,
		"objectives": &review.Objectives, "avoid_list": &review.AvoidList,
	}
	for _, name := range briefReviewAspects {
		obj, ok := raw[name].(map[string]any)
		if !ok {
			return BriefReview{}, fmt.Errorf("%w: thiếu mục %s", ErrBriefReviewReplyUnusable, name)
		}
		rating, _ := obj["rating"].(string)
		comment, _ := obj["comment"].(string)
		switch IdeaRating(rating) {
		case RatingGood, RatingFair, RatingWeak:
		default:
			return BriefReview{}, fmt.Errorf("%w: mục %s có mức %q lạ", ErrBriefReviewReplyUnusable, name, rating)
		}
		if strings.TrimSpace(comment) == "" {
			return BriefReview{}, fmt.Errorf("%w: mục %s chưa có nhận xét", ErrBriefReviewReplyUnusable, name)
		}
		*targets[name] = IdeaAspect{Rating: IdeaRating(rating), Comment: strings.TrimSpace(comment)}
	}
	review.Unclear = stringList(raw["unclear"])
	review.Suggestions = stringList(raw["suggestions"])
	return review, nil
}
