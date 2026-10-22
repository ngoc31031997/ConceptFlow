package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// The creative brief is the idea step's one review unit: brief:main.
const (
	UnitKindBrief = "brief"
	BriefUnitKey  = "main"
)

// CreativeBrief turns an approved idea into something specific enough to
// research and tell: who the video is for, what the viewer will know,
// understand and feel afterwards, the tone, and what it leaves out. The
// length, the form, the language and the platform are chosen before the brief
// is written and belong to the project.
type CreativeBrief struct {
	Audience   string `json:"audience"`
	Know       string `json:"know"`
	Understand string `json:"understand"`
	Feel       string `json:"feel"`
	Tone       string `json:"tone"`
	OutOfScope string `json:"out_of_scope"`
}

// The brief step's platform choice is stored as a unit of its own, since it
// is picked before any brief exists: brief_platform:main.
const (
	UnitKindBriefPlatform = "brief_platform"
	BriefPlatformKey      = "main"
)

// BriefPlatform is the stored platform choice.
type BriefPlatform struct {
	Platform string `json:"platform"`
}

// The platforms a video can be published to.
const (
	PlatformYouTube       = "YouTube"
	PlatformYouTubeShorts = "YouTube Shorts"
	PlatformWebsite       = "Website, khoá học"
)

// Platforms lists the choices in the order the screen shows them.
var Platforms = []string{PlatformYouTube, PlatformYouTubeShorts, PlatformWebsite}

// DefaultPlatform is the platform a project has until the Creator picks one:
// a short goes to YouTube Shorts, a long video to YouTube.
func DefaultPlatform(mode VideoOutputMode) string {
	if mode == ModeShortOnly {
		return PlatformYouTubeShorts
	}
	return PlatformYouTube
}

// ValidPlatform reports whether p is one of Platforms.
func ValidPlatform(p string) bool {
	for _, known := range Platforms {
		if known == p {
			return true
		}
	}
	return false
}

// BriefFields are the brief's field names in the order the screen and the
// prompts show them.
var BriefFields = []string{"audience", "know", "understand", "feel", "tone", "out_of_scope"}

// briefFieldLabels names each field in the text the story step reads.
var briefFieldLabels = map[string]string{
	"audience":     "Khán giả",
	"know":         "Sau video, người xem sẽ biết",
	"understand":   "Sau video, người xem sẽ hiểu",
	"feel":         "Sau video, người xem sẽ cảm thấy",
	"tone":         "Giọng điệu",
	"out_of_scope": "Ngoài phạm vi (video không nói tới)",
}

// IsBriefField reports whether name is one of BriefFields.
func IsBriefField(name string) bool {
	_, ok := briefFieldLabels[name]
	return ok
}

// Field returns the value of the named field ("" for an unknown name).
func (b CreativeBrief) Field(name string) string {
	switch name {
	case "audience":
		return b.Audience
	case "know":
		return b.Know
	case "understand":
		return b.Understand
	case "feel":
		return b.Feel
	case "tone":
		return b.Tone
	case "out_of_scope":
		return b.OutOfScope
	}
	return ""
}

// WithField returns a copy of b with the named field set; an unknown name
// leaves b as it is.
func (b CreativeBrief) WithField(name, value string) CreativeBrief {
	switch name {
	case "audience":
		b.Audience = value
	case "know":
		b.Know = value
	case "understand":
		b.Understand = value
	case "feel":
		b.Feel = value
	case "tone":
		b.Tone = value
	case "out_of_scope":
		b.OutOfScope = value
	}
	return b
}

// MissingFields lists the fields that are empty, in BriefFields order.
func (b CreativeBrief) MissingFields() []string {
	var missing []string
	for _, f := range BriefFields {
		if strings.TrimSpace(b.Field(f)) == "" {
			missing = append(missing, f)
		}
	}
	return missing
}

// MissingFieldLabels names the empty fields as the screen labels them, for a
// message the Creator reads ("Khán giả; Giọng điệu").
func (b CreativeBrief) MissingFieldLabels() string {
	missing := b.MissingFields()
	labels := make([]string, 0, len(missing))
	for _, f := range missing {
		labels = append(labels, briefFieldLabels[f])
	}
	return strings.Join(labels, "; ")
}

// KeepCreatorFields returns written with every field in creatorFields taken
// from current: what the Creator wrote survives an AI redo verbatim.
func KeepCreatorFields(written, current CreativeBrief, creatorFields []string) CreativeBrief {
	for _, f := range creatorFields {
		written = written.WithField(f, current.Field(f))
	}
	return written
}

// ErrBriefReplyUnusable marks a model reply the brief cannot be read from.
var ErrBriefReplyUnusable = errors.New("câu trả lời của AI không dùng được làm brief")

// ParseBriefReply reads the brief out of a model reply: the first JSON object
// in it, with or without a ```json fence around it. Every field must be there
// and non-empty; a reply missing one is an error naming the fields, so a half-written brief is never stored.
func ParseBriefReply(reply string) (CreativeBrief, error) {
	raw, err := replyObject(reply)
	if err != nil {
		return CreativeBrief{}, fmt.Errorf("%w: %v", ErrBriefReplyUnusable, err)
	}
	var brief CreativeBrief
	for _, f := range BriefFields {
		if s, ok := raw[f].(string); ok {
			brief = brief.WithField(f, strings.TrimSpace(s))
		}
	}
	if len(brief.MissingFields()) > 0 {
		return CreativeBrief{}, fmt.Errorf("%w: thiếu ô %s", ErrBriefReplyUnusable, brief.MissingFieldLabels())
	}
	return brief, nil
}

// replyObject reads the first JSON object out of a model reply, with or
// without a ```json fence around it.
func replyObject(reply string) (map[string]any, error) {
	start := strings.Index(reply, "{")
	end := strings.LastIndex(reply, "}")
	if start < 0 || end <= start {
		return nil, errors.New("không có khối JSON")
	}
	var raw map[string]any
	if err := json.Unmarshal([]byte(reply[start:end+1]), &raw); err != nil {
		return nil, fmt.Errorf("khối JSON không đọc được (%v)", err)
	}
	return raw, nil
}

// TargetDuration is the length a long video aims for, in minutes. A short has
// none: it is always under a minute.
type TargetDuration string

// The target lengths the idea step offers for a long video.
const (
	Duration3to4  TargetDuration = "3-4"
	Duration4to5  TargetDuration = "4-5"
	Duration5to7  TargetDuration = "5-7"
	Duration7to10 TargetDuration = "7-10"
)

// TargetDurations lists the choices in the order the screen shows them.
var TargetDurations = []TargetDuration{Duration3to4, Duration4to5, Duration5to7, Duration7to10}

// DefaultTargetDuration is what a long video gets until the Creator picks one.
const DefaultTargetDuration = Duration4to5

// ValidTargetDuration reports whether d is one of TargetDurations.
func ValidTargetDuration(d TargetDuration) bool {
	for _, t := range TargetDurations {
		if t == d {
			return true
		}
	}
	return false
}

// TargetDurationLabel is the length a video aims for as prompts and screens
// say it: "4–5 phút" for a long video, "dưới 60 giây" for a short. A long
// video with no valid choice saved reads as the default.
func TargetDurationLabel(mode VideoOutputMode, d TargetDuration, language string) string {
	if mode == ModeShortOnly {
		if language == "en" {
			return "under 60 seconds"
		}
		return "dưới 60 giây"
	}
	if !ValidTargetDuration(d) {
		d = DefaultTargetDuration
	}
	span := strings.Replace(string(d), "-", "–", 1)
	if language == "en" {
		return span + " minutes"
	}
	return span + " phút"
}

// BriefSection is the approved idea and brief as the later steps read them,
// one labelled line per field, plus the target length and the platform.
func BriefSection(i Idea, b CreativeBrief, durationLabel, platform string) string {
	var sb strings.Builder
	sb.WriteString(IdeaSection(i))
	sb.WriteString("\nThời lượng mục tiêu: " + durationLabel)
	sb.WriteString("\nNền tảng phát hành: " + platform)
	for _, f := range BriefFields {
		sb.WriteString("\n" + briefFieldLabels[f] + ": " + b.Field(f))
	}
	return sb.String()
}
