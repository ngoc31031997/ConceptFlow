package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Sources are the inputs a part of the visual concept is written from. Each has
// one hash, so a part whose sources changed since it was written reads as stale.
const (
	SourceBrief    = "brief"
	SourceResearch = "research"
	SourceStory    = "story"
	SourceBeats    = "beats"
	SourceScript   = "script"
	SourceRefs     = "refs"
)

// SourceLabels names each source for the Creator.
var SourceLabels = map[string]string{
	SourceBrief:    "Creative brief",
	SourceResearch: "Hồ sơ nghiên cứu",
	SourceStory:    "Cấu trúc câu chuyện",
	SourceBeats:    "Beat sheet",
	SourceScript:   "Kịch bản đã chốt",
	SourceRefs:     "Mô tả tham chiếu",
}

// Fact types of the research dossier that need a warning in the visual rules.
const (
	FactTypeHypothesis    = "hypothesis"
	FactTypeMisconception = "misconception"
)

// ScriptVersionPart is one frozen piece of a locked script version, as
// authoring-service returns it; sha is the hash of content.
type ScriptVersionPart struct {
	Key     string          `json:"key"`
	SHA     string          `json:"sha"`
	Content json.RawMessage `json:"content"`
}

// ScriptVersion is the latest locked script version of a video.
type ScriptVersion struct {
	ProjectID string              `json:"project_id"`
	Version   int                 `json:"version"`
	Note      string              `json:"note"`
	LockedAt  time.Time           `json:"locked_at"`
	Parts     []ScriptVersionPart `json:"parts"`
}

// Brief is the approved creative brief.
type Brief struct {
	Audience           string   `json:"audience"`
	DurationSeconds    int      `json:"duration_seconds"`
	Tone               string   `json:"tone"`
	LearningObjectives []string `json:"learning_objectives"`
	Avoid              []string `json:"avoid"`
}

// Fact is one kept fact of the research dossier.
type Fact struct {
	ID    string `json:"ID"`
	Type  string `json:"type"`
	Claim string `json:"claim_vi"`
	Note  string `json:"note"`
}

// StoryField is one field of the story structure.
type StoryField struct {
	Text  string   `json:"text"`
	Facts []string `json:"facts"`
}

// Beat is one beat of the beat sheet.
type Beat struct {
	Key     string `json:"key"`
	Number  int    `json:"number"`
	Fn      string `json:"fn"`
	Title   string `json:"title"`
	Goal    string `json:"goal"`
	Dur     int    `json:"dur"`
	Content string `json:"content"`
}

// Line is one line of the locked script.
type Line struct {
	ID    string   `json:"id"`
	Text  string   `json:"text"`
	Facts []string `json:"facts"`
}

// Handoff is the handoff note of one beat.
type Handoff struct {
	Must   []string `json:"must"`
	Terms  []string `json:"terms"`
	Exact  []string `json:"exact"`
	Free   []string `json:"free"`
	Unsure []string `json:"unsure"`
}

// GlossaryTerm is one row of the script's shared glossary.
type GlossaryTerm struct {
	Term  string `json:"term"`
	Say   string `json:"say"`
	Draw  string `json:"draw"`
	Avoid string `json:"avoid"`
}

// ConceptInput is everything step 8 reads, taken from the latest locked script
// version and the reference descriptions the video picked.
type ConceptInput struct {
	ScriptVersion   int
	ScriptLockedAt  time.Time
	Brief           Brief
	Facts           []Fact
	ResearchSkipped bool
	Story           map[string]StoryField
	Beats           []Beat
	Lines           map[string][]Line
	Handoff         map[string]Handoff
	Glossary        []GlossaryTerm
	Refs            []ReferenceStyle
	Hashes          map[string]string
}

// BeatKeys lists the beat keys in story order.
func (in ConceptInput) BeatKeys() []string {
	out := make([]string, 0, len(in.Beats))
	for _, b := range in.Beats {
		out = append(out, b.Key)
	}
	return out
}

// FactByID returns the fact with id.
func (in ConceptInput) FactByID(id string) (Fact, bool) {
	for _, f := range in.Facts {
		if f.ID == id {
			return f, true
		}
	}
	return Fact{}, false
}

// RefIDs lists the ids of the picked descriptions.
func (in ConceptInput) RefIDs() []string {
	out := make([]string, 0, len(in.Refs))
	for _, r := range in.Refs {
		out = append(out, r.ID)
	}
	return out
}

// NewConceptInput reads a locked script version and the picked descriptions.
// A part missing from the version, or one that cannot be read, is an error:
// authoring-service always freezes every part.
func NewConceptInput(v ScriptVersion, refs []ReferenceStyle) (ConceptInput, error) {
	parts := map[string]ScriptVersionPart{}
	for _, p := range v.Parts {
		parts[p.Key] = p
	}
	read := func(key string, out any) error {
		p, ok := parts[key]
		if !ok {
			return fmt.Errorf("kịch bản chốt v%d thiếu phần %q", v.Version, key)
		}
		if err := json.Unmarshal(p.Content, out); err != nil {
			return fmt.Errorf("đọc phần %q của kịch bản chốt: %w", key, err)
		}
		return nil
	}
	in := ConceptInput{
		ScriptVersion: v.Version, ScriptLockedAt: v.LockedAt,
		Lines: map[string][]Line{}, Handoff: map[string]Handoff{}, Story: map[string]StoryField{},
	}
	if err := read("brief", &in.Brief); err != nil {
		return ConceptInput{}, err
	}
	if err := read("facts", &in.Facts); err != nil {
		return ConceptInput{}, err
	}
	in.ResearchSkipped = len(in.Facts) == 0
	if err := read("story", &in.Story); err != nil {
		return ConceptInput{}, err
	}
	var order []string
	if err := read("order", &order); err != nil {
		return ConceptInput{}, err
	}
	for i, k := range order {
		var b Beat
		if err := read("beat:"+k, &b); err != nil {
			return ConceptInput{}, err
		}
		b.Key, b.Number = k, i+1
		in.Beats = append(in.Beats, b)
		var s struct {
			Lines []Line `json:"lines"`
		}
		if err := read("script:"+k, &s); err != nil {
			return ConceptInput{}, err
		}
		in.Lines[k] = s.Lines
		var h Handoff
		if err := read("handoff:"+k, &h); err != nil {
			return ConceptInput{}, err
		}
		in.Handoff[k] = h
	}
	var g struct {
		Terms []GlossaryTerm `json:"terms"`
	}
	if err := read("glossary", &g); err != nil {
		return ConceptInput{}, err
	}
	in.Glossary = g.Terms
	in.Refs = append([]ReferenceStyle{}, refs...)
	sort.Slice(in.Refs, func(i, j int) bool { return in.Refs[i].ID < in.Refs[j].ID })

	shaOf := func(keys ...string) string {
		var sb strings.Builder
		for _, k := range keys {
			sb.WriteString(k)
			sb.WriteByte('=')
			sb.WriteString(parts[k].SHA)
			sb.WriteByte('\n')
		}
		return hashText(sb.String())
	}
	beatKeys := []string{"order"}
	scriptKeys := []string{"glossary"}
	for _, k := range order {
		beatKeys = append(beatKeys, "beat:"+k)
		scriptKeys = append(scriptKeys, "script:"+k, "handoff:"+k)
	}
	in.Hashes = map[string]string{
		SourceBrief:    shaOf("brief"),
		SourceResearch: shaOf("facts"),
		SourceStory:    shaOf("story"),
		SourceBeats:    shaOf(beatKeys...),
		SourceScript:   shaOf(scriptKeys...),
		SourceRefs:     RefsHash(in.Refs),
	}
	return in, nil
}

// RefsHash is the hash of what the AI reads of the picked descriptions (name,
// summary, drawing, layout), so editing a picked description makes the parts
// that read references stale.
func RefsHash(refs []ReferenceStyle) string {
	sorted := append([]ReferenceStyle{}, refs...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })
	var sb strings.Builder
	for _, r := range sorted {
		raw, _ := json.Marshal(DocumentReferences([]ReferenceStyle{r}))
		fmt.Fprintf(&sb, "%s\n", raw)
	}
	return hashText(sb.String())
}

func hashText(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
