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

// The fields of the story structure, also the names the Creator's own fields
// are recorded under.
const (
	StoryFieldCentralQuestion = "central_question"
	StoryFieldCoreAnswer      = "core_answer"
	StoryFieldNarrativeArc    = "narrative_arc"
)

// StoryFieldKeys are the story's fields in the order the screen shows them.
var StoryFieldKeys = []string{StoryFieldCentralQuestion, StoryFieldCoreAnswer, StoryFieldNarrativeArc}

var storyFieldLabels = map[string]string{
	StoryFieldCentralQuestion: "Câu hỏi trung tâm",
	StoryFieldCoreAnswer:      "Câu trả lời cốt lõi",
	StoryFieldNarrativeArc:    "Mạch kể",
}

// StoryFieldLabel is the name the screen gives a field ("" for an unknown one).
func StoryFieldLabel(key string) string { return storyFieldLabels[key] }

// IsStoryFieldKey reports whether key is one of StoryFieldKeys.
func IsStoryFieldKey(key string) bool {
	_, ok := storyFieldLabels[key]
	return ok
}

// The number of beats the narrative arc may hold. The AI chooses how many from
// the facts it is given.
const (
	MinArcBeats = 3
	MaxArcBeats = 12
)

// MaxArcBeatNameRunes is the longest name of a beat of the arc.
const MaxArcBeatNameRunes = 60

// Errors of the story step.
var (
	// ErrStoryNotFound is returned for a project that has no story yet.
	ErrStoryNotFound = errors.New("project chưa có câu chuyện")
	// ErrStoryInvalid marks a story that fails the machine check.
	ErrStoryInvalid = errors.New("câu chuyện chưa hợp lệ")
	// ErrStoryReplyUnusable marks a model reply the story cannot be read from.
	ErrStoryReplyUnusable = errors.New("câu trả lời của AI không dùng được làm câu chuyện")
)

// ArcBeat is one beat of the narrative arc: its role in the telling, what it
// says, and the research facts it rests on (possibly none).
type ArcBeat struct {
	Beat  string   `json:"beat"`
	Text  string   `json:"text"`
	Facts []string `json:"facts"`
}

// StoryStructure is the output of the story step: the question the video asks,
// the answer it gives, and the arc that leads from one to the other.
type StoryStructure struct {
	CentralQuestion string    `json:"central_question"`
	CoreAnswer      string    `json:"core_answer"`
	NarrativeArc    []ArcBeat `json:"narrative_arc"`
}

// Normalized trims the texts, drops a beat with neither a name nor a text, and
// gives every beat a list of distinct fact codes, so equal stories compare and
// hash equal.
func (s StoryStructure) Normalized() StoryStructure {
	out := StoryStructure{
		CentralQuestion: strings.TrimSpace(s.CentralQuestion),
		CoreAnswer:      strings.TrimSpace(s.CoreAnswer),
		NarrativeArc:    []ArcBeat{},
	}
	for _, b := range s.NarrativeArc {
		beat := ArcBeat{Beat: strings.TrimSpace(b.Beat), Text: strings.TrimSpace(b.Text), Facts: uniqueFacts(b.Facts)}
		if beat.Beat == "" && beat.Text == "" {
			continue
		}
		out.NarrativeArc = append(out.NarrativeArc, beat)
	}
	return out
}

// MissingFields lists the keys of the fields still empty.
func (s StoryStructure) MissingFields() []string {
	n := s.Normalized()
	var missing []string
	if n.CentralQuestion == "" {
		missing = append(missing, StoryFieldCentralQuestion)
	}
	if n.CoreAnswer == "" {
		missing = append(missing, StoryFieldCoreAnswer)
	}
	if len(n.NarrativeArc) == 0 {
		missing = append(missing, StoryFieldNarrativeArc)
	}
	return missing
}

// Validate is the machine check a story passes before it is approved: the three
// fields are there and the arc has between MinArcBeats and MaxArcBeats beats,
// each with a name and a text. Whether the arc tells a story rather than lists
// information is the Creator's call.
func (s StoryStructure) Validate() error {
	if missing := s.MissingFields(); len(missing) > 0 {
		labels := make([]string, len(missing))
		for i, key := range missing {
			labels[i] = StoryFieldLabel(key)
		}
		return fmt.Errorf("%w: còn trống %s", ErrStoryInvalid, strings.Join(labels, ", "))
	}
	n := s.Normalized()
	if len(n.NarrativeArc) < MinArcBeats {
		return fmt.Errorf("%w: mạch kể cần ít nhất %d nhịp", ErrStoryInvalid, MinArcBeats)
	}
	if len(n.NarrativeArc) > MaxArcBeats {
		return fmt.Errorf("%w: mạch kể tối đa %d nhịp", ErrStoryInvalid, MaxArcBeats)
	}
	for i, b := range n.NarrativeArc {
		if b.Beat == "" || b.Text == "" {
			return fmt.Errorf("%w: nhịp %d thiếu tên hoặc nội dung", ErrStoryInvalid, i+1)
		}
		if len([]rune(b.Beat)) > MaxArcBeatNameRunes {
			return fmt.Errorf("%w: tên nhịp %d dài quá %d ký tự", ErrStoryInvalid, i+1, MaxArcBeatNameRunes)
		}
	}
	return nil
}

// ArcEqual reports whether two arcs hold the same beats once normalized.
func ArcEqual(a, b []ArcBeat) bool {
	return StoryStructure{NarrativeArc: a}.Hash() == StoryStructure{NarrativeArc: b}.Hash()
}

// Hash identifies the story's content, so that a later step can tell whether
// the story it was written from has changed.
func (s StoryStructure) Hash() string {
	// Marshalling a struct of strings cannot fail.
	raw, _ := json.Marshal(s.Normalized())
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// CitedFacts lists the fact codes the arc cites, each once, in order.
func (s StoryStructure) CitedFacts() []string {
	lists := make([][]string, 0, len(s.NarrativeArc))
	for _, b := range s.NarrativeArc {
		lists = append(lists, b.Facts)
	}
	return uniqueFacts(lists...)
}

// ArcText writes the arc one beat per line as "Name: text [F01, F02]", which is
// how a prompt reads it and how the Creator's current arc is shown to the AI.
func (s StoryStructure) ArcText() string {
	lines := make([]string, 0, len(s.NarrativeArc))
	for i, b := range s.Normalized().NarrativeArc {
		lines = append(lines, fmt.Sprintf("%d. %s: %s", i+1, b.Beat, FormatRefLine(b.Text, b.Facts)))
	}
	return strings.Join(lines, "\n")
}

// Section is the story as a later prompt reads it.
func (s StoryStructure) Section() string {
	n := s.Normalized()
	return StoryFieldLabel(StoryFieldCentralQuestion) + ": " + n.CentralQuestion + "\n" +
		StoryFieldLabel(StoryFieldCoreAnswer) + ": " + n.CoreAnswer + "\n" +
		StoryFieldLabel(StoryFieldNarrativeArc) + ":\n" + n.ArcText()
}

// StoryStatus is where a story stands in review.
type StoryStatus string

const (
	StoryPending  StoryStatus = "pending"
	StoryApproved StoryStatus = "approved"
)

// StoredStory is a project's story with what review needs around it.
type StoredStory struct {
	ProjectID string
	Story     StoryStructure
	// CreatorFields are the fields the Creator wrote or changed; the AI keeps
	// them when it writes the story again.
	CreatorFields []string
	Status        StoryStatus
	// BriefHash and ResearchHash identify the approved brief and the kept
	// research facts the story was written from, so the screen can tell when
	// either changed since.
	BriefHash    string
	ResearchHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// HasCreatorField reports whether the Creator owns the named field.
func (s StoredStory) HasCreatorField(name string) bool {
	for _, f := range s.CreatorFields {
		if f == name {
			return true
		}
	}
	return false
}

// KeepStoryCreatorFields returns written with every field the Creator owns
// copied back from current.
func KeepStoryCreatorFields(written, current StoryStructure, creatorFields []string) StoryStructure {
	for _, f := range creatorFields {
		switch f {
		case StoryFieldCentralQuestion:
			written.CentralQuestion = current.CentralQuestion
		case StoryFieldCoreAnswer:
			written.CoreAnswer = current.CoreAnswer
		case StoryFieldNarrativeArc:
			written.NarrativeArc = current.NarrativeArc
		}
	}
	return written
}

// ResearchHash identifies the facts a story is written from: the kept facts'
// codes, claims and types in code order.
func ResearchHash(facts []StoredFact) string {
	type line struct {
		ID, Claim, Type string
	}
	var lines []line
	for _, f := range facts {
		if f.Fact.Keep {
			lines = append(lines, line{f.ID, strings.TrimSpace(f.Fact.ClaimVI), string(f.Fact.Type)})
		}
	}
	raw, _ := json.Marshal(lines)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

type storyReplyWire struct {
	CentralQuestion string `json:"central_question"`
	CoreAnswer      string `json:"core_answer"`
	NarrativeArc    []struct {
		Beat  string   `json:"beat"`
		Text  string   `json:"text"`
		Facts []string `json:"facts"`
	} `json:"narrative_arc"`
}

// ParseStoryReply reads the story out of the JSON object llm-service returned
// for the reply. The story must pass the
// machine check and every fact code it cites must be one known() accepts, so a
// half-written or mis-cited story is never stored.
func ParseStoryReply(reply string, known func(id string) bool) (StoryStructure, error) {
	var wire storyReplyWire
	if err := json.Unmarshal([]byte(reply), &wire); err != nil {
		return StoryStructure{}, fmt.Errorf("%w: khối JSON không đọc được (%v)", ErrStoryReplyUnusable, err)
	}
	story := StoryStructure{CentralQuestion: wire.CentralQuestion, CoreAnswer: wire.CoreAnswer}
	for i, b := range wire.NarrativeArc {
		refs := make([]string, 0, len(b.Facts))
		for _, raw := range b.Facts {
			id, ok := NormalizeFactID(strings.TrimSpace(raw))
			if !ok || known == nil || !known(id) {
				return StoryStructure{}, fmt.Errorf("%w: nhịp %d trích dữ kiện %q không có trong bảng dữ kiện", ErrStoryReplyUnusable, i+1, raw)
			}
			refs = append(refs, id)
		}
		story.NarrativeArc = append(story.NarrativeArc, ArcBeat{Beat: b.Beat, Text: b.Text, Facts: refs})
	}
	story = story.Normalized()
	if err := story.Validate(); err != nil {
		return StoryStructure{}, fmt.Errorf("%w: %v", ErrStoryReplyUnusable, strings.TrimPrefix(err.Error(), ErrStoryInvalid.Error()+": "))
	}
	return story, nil
}

// CheckStoryFacts reports the first fact code the story cites that known()
// does not accept, as an error naming the code.
func CheckStoryFacts(s StoryStructure, known func(id string) bool) error {
	for _, id := range s.CitedFacts() {
		if known == nil || !known(id) {
			return fmt.Errorf("%w: dữ kiện %s không có trong bảng dữ kiện", ErrStoryInvalid, id)
		}
	}
	return nil
}

// StoryField is one field of the story as the beat sheet reads it: the text
// and the fact codes it cites.
type StoryField struct {
	Text  string   `json:"text"`
	Facts []string `json:"facts"`
}

// StoryInputKind and StoryInputKey name the approved story as the input a later
// step records in the reference of a unit it writes (kind:key); the review unit
// store resolves the reference from the project's story.
const (
	StoryInputKind = "story"
	StoryInputKey  = "main"
)

// Keys of the story fields the beat sheet is built from.
const (
	BeatFieldQuestion = "question"
	BeatFieldChain    = "chain"
	BeatFieldPayoff   = "payoff"
)

// MaxStoryChain is the most steps the beat sheet reads from a story's chain.
const MaxStoryChain = 12

// BeatFields presents the story in the three fields the beat sheet is built
// from: the question, the chain (one arc beat per line, with its facts as a
// trailing "[F01, F02]") and the payoff.
func (s StoryStructure) BeatFields() map[string]StoryField {
	n := s.Normalized()
	lines := make([]string, 0, len(n.NarrativeArc))
	for _, b := range n.NarrativeArc {
		lines = append(lines, FormatRefLine(b.Text, b.Facts))
	}
	return map[string]StoryField{
		BeatFieldQuestion: {Text: n.CentralQuestion, Facts: []string{}},
		BeatFieldChain:    {Text: strings.Join(lines, "\n"), Facts: n.CitedFacts()},
		BeatFieldPayoff:   {Text: n.CoreAnswer, Facts: []string{}},
	}
}
