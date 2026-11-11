package domain

import (
	"encoding/json"
	"fmt"
	"time"
)

// SchemaVersion is the version of the visual-concept document format.
const SchemaVersion = "1.0"

// AccuracyNote is one note the steps after read about a fact.
type AccuracyNote struct {
	Fact string `json:"fact"`
	Note string `json:"note"`
}

// DocumentRepresentation is part 4 with part 5 as its typography, as the
// document holds it.
type DocumentRepresentation struct {
	RepresentationRules
	Typography Typography `json:"typography"`
}

// ReviewComment is one review finding as the locked document keeps it.
type ReviewComment struct {
	ID       string `json:"id"`
	Group    string `json:"group"`
	Text     string `json:"text"`
	Decision string `json:"decision"`
	Why      string `json:"why"`
}

// DocumentReview is the Creator's approval and the review it went through.
type DocumentReview struct {
	Reviewer   string          `json:"reviewer"`
	ApprovedAt time.Time       `json:"approved_at"`
	Comments   []ReviewComment `json:"comments"`
}

// Document is a locked visual concept: the frozen document the later visual
// steps read.
type Document struct {
	SchemaVersion       string                 `json:"schema_version"`
	ProjectID           string                 `json:"project_id"`
	DocumentID          string                 `json:"document_id"`
	Status              string                 `json:"status"`
	Version             int                    `json:"version"`
	ScriptVersion       int                    `json:"script_version"`
	LockedAt            time.Time              `json:"locked_at"`
	CreativeDirection   CreativeDirection      `json:"creative_direction"`
	ColorSystem         ColorSystem            `json:"color_system"`
	VisualLanguage      VisualLanguage         `json:"visual_language"`
	RepresentationRules DocumentRepresentation `json:"representation_rules"`
	MotionLanguage      MotionLanguage         `json:"motion_language"`
	DoNotUse            []Forbidden            `json:"do_not_use"`
	Feasibility         Feasibility            `json:"feasibility"`
	AccuracyNotes       []AccuracyNote         `json:"accuracy_notes"`
	ReferenceStyles     []DocumentReference    `json:"reference_styles"`
	Review              DocumentReview         `json:"review"`
}

// Complete reports whether every part exists.
func (p Parts) Complete() bool {
	return p.CreativeDirection != nil && p.ColorSystem != nil && p.VisualLanguage != nil &&
		p.RepresentationRules != nil && p.Typography != nil && p.MotionLanguage != nil &&
		p.DoNotUse != nil && p.Feasibility != nil
}

// AccuracyNotes gathers, in part order, the facts of part 4 and the accuracy
// rows of part 7 into the notes the later steps read.
func AccuracyNotes(p Parts) []AccuracyNote {
	out := []AccuracyNote{}
	add := func(facts []string, note string) {
		for _, f := range facts {
			out = append(out, AccuracyNote{Fact: f, Note: note})
		}
	}
	if r := p.RepresentationRules; r != nil {
		for _, list := range [][]RepresentedThing{r.Characters.Items, r.Objects, r.Invisible} {
			for _, t := range list {
				add(t.Facts, thingNote(t.Term, t.Draw, t.MustNotImply))
			}
		}
		for _, s := range r.ScientificConcepts {
			add(s.Facts, thingNote(s.Term, s.Representation, s.MustNotImply))
		}
	}
	if d := p.DoNotUse; d != nil {
		for _, it := range d.Items {
			if it.Source == "accuracy" {
				note := "Không dùng: " + it.Text
				if it.Why != "" {
					note += " (" + it.Why + ")"
				}
				add(it.Facts, note)
			}
		}
	}
	return out
}

func thingNote(term, draw string, mustNot []string) string {
	note := term + ": " + draw
	if len(mustNot) > 0 {
		b, _ := json.Marshal(mustNot)
		note += "; không được gợi " + string(b)
	}
	return note
}

// BuildDocument assembles the locked document from complete parts.
func BuildDocument(projectID string, version, scriptVersion int, lockedAt time.Time, p Parts, refs []ReferenceStyle, review ReviewState) (Document, error) {
	if !p.Complete() {
		return Document{}, fmt.Errorf("%w: chưa đủ 8 phần", ErrLockNotReady)
	}
	comments := []ReviewComment{}
	for _, f := range review.Findings {
		comments = append(comments, ReviewComment{ID: f.ID, Group: f.Group, Text: f.Text, Decision: f.State, Why: f.Why})
	}
	return Document{
		SchemaVersion: SchemaVersion, ProjectID: projectID,
		DocumentID: fmt.Sprintf("visual-concept-v%d", version), Status: "locked",
		Version: version, ScriptVersion: scriptVersion, LockedAt: lockedAt,
		CreativeDirection:   *p.CreativeDirection,
		ColorSystem:         *p.ColorSystem,
		VisualLanguage:      *p.VisualLanguage,
		RepresentationRules: DocumentRepresentation{RepresentationRules: *p.RepresentationRules, Typography: *p.Typography},
		MotionLanguage:      *p.MotionLanguage,
		DoNotUse:            p.DoNotUse.Items,
		Feasibility:         *p.Feasibility,
		AccuracyNotes:       AccuracyNotes(p),
		ReferenceStyles:     DocumentReferences(refs),
		Review:              DocumentReview{Reviewer: "creator", ApprovedAt: lockedAt, Comments: comments},
	}, nil
}
