package domain

import "time"

// Unit kinds of step 8.
const (
	// UnitKindConcept is one part of the visual-concept document.
	UnitKindConcept = "visual_concept"
	// UnitKindReview holds the review of the whole concept under ReviewUnitKey.
	UnitKindReview = "visual_review"
	// ReviewUnitKey is the one key of UnitKindReview.
	ReviewUnitKey = "main"
)

// Statuses of a review unit. UnitRunning is never stored: it is how a unit the
// AI is writing reads while the call is open.
const (
	UnitPending  = "pending"
	UnitApproved = "approved"
	UnitRunning  = "running"
)

// ReviewUnit is one row of review_units.
type ReviewUnit struct {
	ProjectID     string
	Kind          string
	Key           string
	Status        string
	Content       []byte
	CreatorFields []string
	InputHashes   map[string]string
	RedoCount     int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// HasCreatorField reports whether the Creator wrote field.
func (u ReviewUnit) HasCreatorField(field string) bool {
	for _, f := range u.CreatorFields {
		if f == field {
			return true
		}
	}
	return false
}

// AddCreatorFields returns fields added to the unit's own, each once.
func (u ReviewUnit) AddCreatorFields(fields ...string) []string {
	out := append([]string{}, u.CreatorFields...)
	for _, f := range fields {
		if !u.HasCreatorField(f) {
			out = append(out, f)
			u.CreatorFields = out
		}
	}
	return out
}
