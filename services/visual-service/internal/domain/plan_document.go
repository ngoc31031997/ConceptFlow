package domain

import (
	"fmt"
	"time"
)

// PlanSourceVersions are the locked versions a plan was written from. The
// brief, facts and beats are frozen inside the script version.
type PlanSourceVersions struct {
	Script        int `json:"script"`
	VisualConcept int `json:"visual_concept"`
}

// DocumentDuration is how long a locked unit lasts.
type DocumentDuration struct {
	NarrationMinS float64 `json:"narration_min_s"`
	PlannedS      float64 `json:"planned_s"`
	ExtraReason   string  `json:"extra_reason"`
}

// DocumentUnit is one unit of a locked plan.
type DocumentUnit struct {
	UnitID                string           `json:"unit_id"`
	Order                 int              `json:"order"`
	BeatID                string           `json:"beat_id"`
	ScriptRefs            []string         `json:"script_refs"`
	Duration              DocumentDuration `json:"duration"`
	NarrationSummary      string           `json:"narration_summary"`
	CommunicationGoal     string           `json:"communication_goal"`
	EssentialVisualInfo   []EssentialInfo  `json:"essential_visual_info"`
	OnScreenText          []string         `json:"on_screen_text"`
	VisualApproach        VisualApproach   `json:"visual_approach"`
	NarrationRelationship string           `json:"narration_relationship"`
	FactRefs              []string         `json:"fact_refs"`
	RiskNotes             []RiskNote       `json:"risk_notes"`
	OpenQuestions         []OpenQuestion   `json:"open_questions"`
	Status                string           `json:"status"`
}

// PlanDocument is a locked visual plan: the frozen document the later visual
// steps read.
type PlanDocument struct {
	SchemaVersion   string             `json:"schema_version"`
	ProjectID       string             `json:"project_id"`
	DocumentID      string             `json:"document_id"`
	VisualConceptID string             `json:"visual_concept_id"`
	Status          string             `json:"status"`
	Version         int                `json:"version"`
	LockedAt        time.Time          `json:"locked_at"`
	SourceVersions  PlanSourceVersions `json:"source_versions"`
	Coverage        PlanCoverage       `json:"coverage"`
	Units           []DocumentUnit     `json:"units"`
	Review          DocumentReview     `json:"review"`
}

// BuildPlanUnits returns the units of every beat in story order, numbered from 1.
func BuildPlanUnits(in PlanInput, plans map[string]BeatPlan) []DocumentUnit {
	out := []DocumentUnit{}
	for _, b := range in.Script.Beats {
		pc := in.PlanContextFor(b.Key)
		for _, u := range plans[b.Key].Units {
			min := pc.UnitNarrationMin(u)
			out = append(out, DocumentUnit{
				UnitID: u.UnitID, Order: len(out) + 1, BeatID: b.Key, ScriptRefs: u.ScriptRefs,
				Duration:         DocumentDuration{NarrationMinS: min, PlannedS: Round1(PlannedSeconds(u, min)), ExtraReason: u.Duration.ExtraReason},
				NarrationSummary: u.NarrationSummary, CommunicationGoal: u.CommunicationGoal,
				EssentialVisualInfo: u.EssentialVisualInfo, OnScreenText: u.OnScreenText, VisualApproach: u.VisualApproach,
				NarrationRelationship: u.NarrationRelationship, FactRefs: FactRefs(u, pc.Lines),
				RiskNotes: u.RiskNotes, OpenQuestions: u.OpenQuestions, Status: u.Status,
			})
		}
	}
	return out
}

// BuildPlanDocument assembles the locked document. It fails with ErrLockNotReady
// when no unit exists or the lines are not each in exactly one unit.
func BuildPlanDocument(projectID string, version int, lockedAt time.Time, in PlanInput, plans map[string]BeatPlan, review ReviewState) (PlanDocument, error) {
	units := BuildPlanUnits(in, plans)
	cov := ComputeCoverage(in, plans)
	if len(units) == 0 || !cov.Complete() {
		return PlanDocument{}, fmt.Errorf("%w: kế hoạch chưa phủ hết lời thoại", ErrLockNotReady)
	}
	comments := []ReviewComment{}
	for _, f := range review.Findings {
		comments = append(comments, ReviewComment{ID: f.ID, Group: f.Group, Text: f.Text, Decision: f.State, Why: f.Why})
	}
	for _, c := range PlanChecks(in, plans) {
		if d, ok := review.CheckAccepted(c); ok && !c.OK {
			comments = append(comments, ReviewComment{ID: c.ID, Group: "machine", Text: c.Label + ": " + c.Detail, Decision: FindingAccepted, Why: d.Why})
		}
	}
	return PlanDocument{
		SchemaVersion: SchemaVersion, ProjectID: projectID, DocumentID: fmt.Sprintf("visual-plan-v%d", version),
		VisualConceptID: fmt.Sprintf("visual-concept-v%d", in.ConceptVersion), Status: "locked",
		Version: version, LockedAt: lockedAt,
		SourceVersions: PlanSourceVersions{Script: in.Script.ScriptVersion, VisualConcept: in.ConceptVersion},
		Coverage:       cov, Units: units,
		Review: DocumentReview{Reviewer: "creator", ApprovedAt: lockedAt, Comments: comments},
	}, nil
}
