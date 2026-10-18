package domain

import (
	"encoding/json"
	"errors"
	"time"
)

// UnitStatus is where a review unit stands in the Creator's review.
type UnitStatus string

const (
	// UnitPending waits for the Creator to approve it.
	UnitPending UnitStatus = "pending"
	// UnitApproved is approved; the step can move on once all its units are.
	UnitApproved UnitStatus = "approved"
)

// ErrUnitNotFound is returned when a project has no unit of that kind and key.
var ErrUnitNotFound = errors.New("review unit not found")

// ErrUnknownUnitKind is returned for a unit kind no step registered.
var ErrUnknownUnitKind = errors.New("unknown review unit kind")

// ErrInvalidUnitEdit marks a manual edit whose fields the unit kind rejects.
var ErrInvalidUnitEdit = errors.New("invalid review unit edit")

// ReviewUnit is the smallest piece of a project's content the Creator
// approves, edits by hand or sends back to the AI with a note, addressed as
// Kind:Key (for example brief:main). Content is the kind's own JSON.
// CreatorFields names the parts of Content the Creator wrote; an AI redo keeps
// them as they are. RedoCount counts AI redos, for the first-time-approval
// rate. See ADR-0035.
type ReviewUnit struct {
	ProjectID     string          `json:"project_id"`
	Kind          string          `json:"kind"`
	Key           string          `json:"key"`
	Status        UnitStatus      `json:"status"`
	Content       json.RawMessage `json:"content"`
	CreatorFields []string        `json:"creator_fields"`
	RedoCount     int             `json:"redo_count"`
	UpdatedAt     time.Time       `json:"updated_at"`
	// InputRef names the unit this one was built from (kind:key), or "" when it
	// was not built from another unit.
	InputRef string `json:"input_ref"`
	// InputSHA is the hash of the InputRef unit's content when this unit was
	// built; Stale compares it with the hash now.
	InputSHA string `json:"-"`
	// Stale is true when the unit it was built from has changed since. Set by
	// the store when reading; never written.
	Stale bool `json:"stale"`
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
