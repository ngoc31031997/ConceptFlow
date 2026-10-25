package domain

import (
	"errors"
	"time"
)

// ErrBeatSheetNotFound is returned when a project has no stored beat sheet.
var ErrBeatSheetNotFound = errors.New("beat sheet not found")

// BeatSlot is what the sheet records about a beat besides its content: the key
// it keeps for good, where it stands in review, and what the Creator wrote.
type BeatSlot struct {
	Key    string     `json:"key"`
	Status UnitStatus `json:"status"`
	// CreatorFields lists the fields the Creator wrote; an AI run keeps them.
	CreatorFields []string `json:"creator_fields"`
	// RedoCount is how many times the AI wrote this beat again.
	RedoCount int `json:"redo_count"`
	// StoryHash is the hash of the story the AI wrote the beat from; "" for a
	// beat the Creator wrote or the skeleton built.
	StoryHash string `json:"story_hash"`
}

// HasCreatorField reports whether the Creator wrote field.
func (s BeatSlot) HasCreatorField(field string) bool {
	for _, f := range s.CreatorFields {
		if f == field {
			return true
		}
	}
	return false
}

// StoredBeat is one beat as the project_beat_sheet table holds it. The
// content (Beat) is what a later step hashes to tell that the beat changed;
// the slot is not part of that hash.
type StoredBeat struct {
	BeatSlot
	Beat Beat `json:"beat"`
}

// StoredBeatSheet is a project's beat sheet: its beats in the order the video
// runs them, and the gate that says how far the step has come.
type StoredBeatSheet struct {
	ProjectID string
	// Beats are in the order the video runs them.
	Beats []StoredBeat
	BeatSheetGate
	UpdatedAt time.Time
}
