package domain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Unit kinds of step 10.
const (
	// UnitKindSequence holds the sequence items of one beat's units; its unit_key
	// is the beat key.
	UnitKindSequence = "visual_sequence"
	// UnitKindSequenceMotifs holds the motifs of the whole video under ReviewUnitKey.
	UnitKindSequenceMotifs = "visual_sequence_motifs"
	// UnitKindSequenceReview holds the review of the whole sequence under ReviewUnitKey.
	UnitKindSequenceReview = "visual_sequence_review"
)

// Tempos of a unit's pacing. The pacing itself is free text; the tempo is the
// one closed value, so the flow strip can colour it and the review can set it
// beside the visual concept's pace of the beat.
const (
	TempoSlow   = "slow"
	TempoMedium = "medium"
	TempoFast   = "fast"
	TempoCalm   = "calm"
)

// Tempos lists every tempo.
var Tempos = []string{TempoSlow, TempoMedium, TempoFast, TempoCalm}

var motifIDPattern = regexp.MustCompile(`^MOTIF-[0-9]{2,}$`)

// MotifID is the id of the n-th motif of a video; numbers are never reused.
func MotifID(n int) string { return fmt.Sprintf("MOTIF-%02d", n) }

// MotifSeq is the number inside a motif id (MOTIF-07 is 7), or 0.
func MotifSeq(id string) int {
	if !motifIDPattern.MatchString(id) {
		return 0
	}
	n, _ := strconv.Atoi(strings.TrimPrefix(id, "MOTIF-"))
	return n
}

// Pacing is how a unit moves in time: a free name and description, and a tempo.
type Pacing struct {
	Name        string `json:"name"`
	Tempo       string `json:"tempo"`
	Description string `json:"description"`
}

// Transition is how the picture passes from the unit before into this one. Type
// is free text; ViaMotif, when set, is the motif both units share that carries
// the passage.
type Transition struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	ViaMotif    string `json:"via_motif"`
}

// SeqItem is what step 10 adds to one unit of the locked visual plan.
// TransitionFromPrevious is nil on the first unit of the video.
type SeqItem struct {
	UnitID                 string      `json:"unit_id"`
	PurposeInFlow          string      `json:"purpose_in_flow"`
	Pacing                 Pacing      `json:"pacing"`
	TransitionFromPrevious *Transition `json:"transition_from_previous"`
	MotifRefs              []string    `json:"motif_refs"`
	Status                 string      `json:"status"`
}

// BeatSequence is the content of a visual_sequence review unit: the items of one
// beat's units in plan order.
type BeatSequence struct {
	Items []SeqItem `json:"items"`
}

// Motif is a picture repeated on purpose across units to tie the flow together.
// AnchorRef points at what step 8 locked on how to draw it. PlannedUnits are the
// units the AI meant it for when it proposed it, a hint for writing the beats;
// where a motif really appears is read from the items' MotifRefs.
type Motif struct {
	MotifID      string   `json:"motif_id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	AnchorRef    string   `json:"anchor_ref"`
	Role         string   `json:"role"`
	PlannedUnits []string `json:"planned_units"`
}

// MotifSet is the content of the visual_sequence_motifs unit. NextSeq is the
// number the next motif takes.
type MotifSet struct {
	Motifs  []Motif `json:"motifs"`
	NextSeq int     `json:"next_seq"`
}

// SequenceEditableFields are the fields of an item the Creator can write. The
// unit id comes from the plan and the status from approving.
var SequenceEditableFields = []string{"purpose_in_flow", "pacing", "transition_from_previous", "motif_refs"}

// MotifEditableFields are the fields of a motif the Creator can write.
var MotifEditableFields = []string{"name", "description", "anchor_ref", "role"}

// strictDecode reads raw into v and refuses an unknown field or a wrong type.
func strictDecode(raw []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidEdit, err)
	}
	return nil
}

// isEmptyContent reports whether stored content holds nothing yet.
func isEmptyContent(raw []byte) bool {
	t := bytes.TrimSpace(raw)
	return len(t) == 0 || string(t) == "{}"
}

// DecodeBeatSequence reads the stored items of a beat. Empty content is a beat
// with no item yet; an unknown field is an error.
func DecodeBeatSequence(raw []byte) (BeatSequence, error) {
	var s BeatSequence
	if !isEmptyContent(raw) {
		if err := strictDecode(raw, &s); err != nil {
			return BeatSequence{}, err
		}
	}
	for i := range s.Items {
		s.Items[i].normalize()
	}
	if s.Items == nil {
		s.Items = []SeqItem{}
	}
	return s, nil
}

// DecodeSeqItem reads one item's JSON. An unknown field or a value of the wrong
// type is ErrInvalidEdit.
func DecodeSeqItem(raw []byte) (SeqItem, error) {
	var it SeqItem
	if err := strictDecode(raw, &it); err != nil {
		return SeqItem{}, err
	}
	it.normalize()
	return it, nil
}

// Encode returns the stored content of the beat.
func (s BeatSequence) Encode() ([]byte, error) {
	for i := range s.Items {
		s.Items[i].normalize()
	}
	if s.Items == nil {
		s.Items = []SeqItem{}
	}
	return json.Marshal(s)
}

// Index returns the position of the item of unit id, or -1.
func (s BeatSequence) Index(unitID string) int {
	for i, it := range s.Items {
		if it.UnitID == unitID {
			return i
		}
	}
	return -1
}

// normalize turns a list a JSON null left nil into an empty list and gives an
// item without status the pending one.
func (it *SeqItem) normalize() {
	if it.MotifRefs == nil {
		it.MotifRefs = []string{}
	}
	if it.Status == "" {
		it.Status = UnitPending
	}
}

// HasMotif reports whether the item names motif id.
func (it SeqItem) HasMotif(id string) bool {
	for _, m := range it.MotifRefs {
		if m == id {
			return true
		}
	}
	return false
}

// WithoutMotif returns the item with motif id taken out of its motifs and of its
// transition, and whether anything changed.
func (it SeqItem) WithoutMotif(id string) (SeqItem, bool) {
	changed := false
	kept := []string{}
	for _, m := range it.MotifRefs {
		if m == id {
			changed = true
			continue
		}
		kept = append(kept, m)
	}
	it.MotifRefs = kept
	if it.TransitionFromPrevious != nil && it.TransitionFromPrevious.ViaMotif == id {
		t := *it.TransitionFromPrevious
		t.ViaMotif = ""
		it.TransitionFromPrevious = &t
		changed = true
	}
	return it, changed
}

// DecodeMotifSet reads the stored motifs. Empty content is a video with no motif
// yet, whose first motif takes number 1.
func DecodeMotifSet(raw []byte) (MotifSet, error) {
	var s MotifSet
	if !isEmptyContent(raw) {
		if err := strictDecode(raw, &s); err != nil {
			return MotifSet{}, err
		}
	}
	for i := range s.Motifs {
		s.Motifs[i].normalize()
	}
	if s.Motifs == nil {
		s.Motifs = []Motif{}
	}
	if s.NextSeq < 1 {
		s.NextSeq = 1
	}
	for _, m := range s.Motifs {
		if n := MotifSeq(m.MotifID); n >= s.NextSeq {
			s.NextSeq = n + 1
		}
	}
	return s, nil
}

// DecodeMotif reads one motif's JSON strictly.
func DecodeMotif(raw []byte) (Motif, error) {
	var m Motif
	if err := strictDecode(raw, &m); err != nil {
		return Motif{}, err
	}
	m.normalize()
	return m, nil
}

// Encode returns the stored content of the motifs.
func (s MotifSet) Encode() ([]byte, error) {
	for i := range s.Motifs {
		s.Motifs[i].normalize()
	}
	if s.Motifs == nil {
		s.Motifs = []Motif{}
	}
	return json.Marshal(s)
}

// Index returns the position of motif id, or -1.
func (s MotifSet) Index(id string) int {
	for i, m := range s.Motifs {
		if m.MotifID == id {
			return i
		}
	}
	return -1
}

// IDs returns the set of motif ids.
func (s MotifSet) IDs() map[string]bool {
	out := make(map[string]bool, len(s.Motifs))
	for _, m := range s.Motifs {
		out[m.MotifID] = true
	}
	return out
}

func (m *Motif) normalize() {
	if m.PlannedUnits == nil {
		m.PlannedUnits = []string{}
	}
}
