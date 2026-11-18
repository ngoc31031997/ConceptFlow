package domain

import (
	"fmt"
	"time"
)

// SequenceSourceVersions are the locked versions a sequence was written from.
type SequenceSourceVersions struct {
	Script        int `json:"script"`
	VisualConcept int `json:"visual_concept"`
	VisualPlan    int `json:"visual_plan"`
}

// DocumentSeqItem is one unit of a locked sequence.
type DocumentSeqItem struct {
	SequenceIndex          int         `json:"sequence_index"`
	UnitID                 string      `json:"unit_id"`
	BeatID                 string      `json:"beat_id"`
	PurposeInFlow          string      `json:"purpose_in_flow"`
	Pacing                 Pacing      `json:"pacing"`
	TransitionFromPrevious *Transition `json:"transition_from_previous"`
	MotifRefs              []string    `json:"motif_refs"`
	Status                 string      `json:"status"`
}

// DocumentMotif is one motif of a locked sequence; UnitRefs are the units whose
// items name it, in plan order.
type DocumentMotif struct {
	MotifID     string   `json:"motif_id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	AnchorRef   string   `json:"anchor_ref"`
	Role        string   `json:"role"`
	UnitRefs    []string `json:"unit_refs"`
}

// SequenceDocument is a locked visual sequence: the frozen document the later
// visual steps read.
type SequenceDocument struct {
	SchemaVersion  string                 `json:"schema_version"`
	ProjectID      string                 `json:"project_id"`
	DocumentID     string                 `json:"document_id"`
	VisualPlanID   string                 `json:"visual_plan_id"`
	Status         string                 `json:"status"`
	Version        int                    `json:"version"`
	LockedAt       time.Time              `json:"locked_at"`
	SourceVersions SequenceSourceVersions `json:"source_versions"`
	Sequence       []DocumentSeqItem      `json:"sequence"`
	Motifs         []DocumentMotif        `json:"motifs"`
	SequenceNotes  []string               `json:"sequence_notes"`
	Review         DocumentReview         `json:"review"`
}

// BuildSequenceItems returns the item of every plan unit that has one, in plan
// order, numbered by the unit's place in the plan.
func BuildSequenceItems(in SequenceInput, beats map[string]BeatSequence) []DocumentSeqItem {
	out := []DocumentSeqItem{}
	for i, u := range in.Plan.Units {
		it, ok := ItemOf(in, beats, u.UnitID)
		if !ok {
			continue
		}
		out = append(out, DocumentSeqItem{
			SequenceIndex: i + 1, UnitID: u.UnitID, BeatID: u.BeatID, PurposeInFlow: it.PurposeInFlow,
			Pacing: it.Pacing, TransitionFromPrevious: it.TransitionFromPrevious, MotifRefs: it.MotifRefs, Status: it.Status,
		})
	}
	return out
}

// BuildSequenceMotifs returns the motifs with where each appears.
func BuildSequenceMotifs(in SequenceInput, beats map[string]BeatSequence, motifs MotifSet) []DocumentMotif {
	seen := MotifAppearances(in, beats, motifs)
	out := []DocumentMotif{}
	for _, m := range motifs.Motifs {
		out = append(out, DocumentMotif{MotifID: m.MotifID, Name: m.Name, Description: m.Description,
			AnchorRef: m.AnchorRef, Role: m.Role, UnitRefs: seen[m.MotifID]})
	}
	return out
}

// BuildSequenceDocument assembles the locked document. It fails with
// ErrLockNotReady when a plan unit has no item or an item has no plan unit.
func BuildSequenceDocument(projectID string, version int, lockedAt time.Time, in SequenceInput, beats map[string]BeatSequence, motifs MotifSet, review ReviewState) (SequenceDocument, error) {
	if !ComputeSequenceCoverage(in, beats).Complete() {
		return SequenceDocument{}, fmt.Errorf("%w: chuỗi hình chưa có đủ unit của kế hoạch hình", ErrLockNotReady)
	}
	comments := []ReviewComment{}
	for _, f := range review.Findings {
		comments = append(comments, ReviewComment{ID: f.ID, Group: f.Group, Text: f.Text, Decision: f.State, Why: f.Why})
	}
	for _, c := range SequenceChecks(in, beats, motifs) {
		if d, ok := review.CheckAccepted(c); ok && !c.OK {
			comments = append(comments, ReviewComment{ID: c.ID, Group: "machine", Text: c.Label + ": " + c.Detail, Decision: FindingAccepted, Why: d.Why})
		}
	}
	notes := review.Notes
	if notes == nil {
		notes = []string{}
	}
	return SequenceDocument{
		SchemaVersion: SchemaVersion, ProjectID: projectID, DocumentID: fmt.Sprintf("visual-sequence-v%d", version),
		VisualPlanID: fmt.Sprintf("visual-plan-v%d", in.PlanVersion), Status: "locked", Version: version, LockedAt: lockedAt,
		SourceVersions: SequenceSourceVersions{Script: in.Script.ScriptVersion, VisualConcept: in.ConceptVersion, VisualPlan: in.PlanVersion},
		Sequence:       BuildSequenceItems(in, beats), Motifs: BuildSequenceMotifs(in, beats, motifs), SequenceNotes: notes,
		Review: DocumentReview{Reviewer: "creator", ApprovedAt: lockedAt, Comments: comments},
	}, nil
}
