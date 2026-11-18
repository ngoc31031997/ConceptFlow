package domain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// --- motifs ------------------------------------------------------------------------

// motifReply is one motif as the AI writes it. The id is the machine's: the AI
// names an existing motif through same_as, and a reply with a motif_id key is
// refused.
type motifReply struct {
	SameAs       string   `json:"same_as"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	AnchorRef    string   `json:"anchor_ref"`
	Role         string   `json:"role"`
	PlannedUnits []string `json:"planned_units"`
}

type motifsReply struct {
	Motifs []motifReply `json:"motifs"`
}

// MotifMerge is the AI's motifs set against the motifs the video had.
type MotifMerge struct {
	Set MotifSet
	// Fields are the Creator's fields that still stand, as "<motif id>.<field>".
	Fields []string
}

// unreadable wraps a reason as ErrUnreadableReply.
func unreadable(format string, a ...any) error {
	return fmt.Errorf("%w: %s", ErrUnreadableReply, fmt.Sprintf(format, a...))
}

func decodeReply(raw []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("%w: %v", ErrUnreadableReply, err)
	}
	return nil
}

// ParseMotifsReply reads the AI's motifs and merges them with the stored ones.
// A motif whose same_as names a stored motif keeps that motif's id and the
// fields the Creator wrote in it; a motif with an empty same_as takes the next
// number, and numbers are never used again. The reply is unusable, and nothing
// is stored, when it has an unknown key, names a stored motif that does not
// exist or names one twice, plans a motif on a unit the plan does not have, or
// drops a motif that an item uses or that the Creator edited. planUnits are the
// unit ids of the locked plan; inUse are the motif ids items name.
func ParseMotifsReply(raw []byte, cur MotifSet, creatorFields []string, planUnits, inUse map[string]bool) (MotifMerge, error) {
	var r motifsReply
	if err := decodeReply(raw, &r); err != nil {
		return MotifMerge{}, err
	}
	if len(r.Motifs) == 0 {
		return MotifMerge{}, unreadable("không có motif nào")
	}
	out := MotifMerge{Set: MotifSet{Motifs: make([]Motif, 0, len(r.Motifs)), NextSeq: cur.NextSeq}, Fields: []string{}}
	if out.Set.NextSeq < 1 {
		out.Set.NextSeq = 1
	}
	kept := map[string]bool{}
	for i, rm := range r.Motifs {
		n := i + 1
		for _, u := range rm.PlannedUnits {
			if !planUnits[u] {
				return MotifMerge{}, unreadable("motif %d dự định xuất hiện ở unit lạ «%s»", n, u)
			}
		}
		m := Motif{Name: rm.Name, Description: rm.Description, AnchorRef: rm.AnchorRef, Role: rm.Role, PlannedUnits: rm.PlannedUnits}
		m.normalize()
		if rm.SameAs == "" {
			m.MotifID = MotifID(out.Set.NextSeq)
			out.Set.NextSeq++
			out.Set.Motifs = append(out.Set.Motifs, m)
			continue
		}
		at := cur.Index(rm.SameAs)
		if at < 0 {
			return MotifMerge{}, unreadable("motif %d ghi same_as «%s», không có motif này", n, rm.SameAs)
		}
		if kept[rm.SameAs] {
			return MotifMerge{}, unreadable("motif %s được ghi hai lần", rm.SameAs)
		}
		kept[rm.SameAs] = true
		m.MotifID = rm.SameAs
		old := cur.Motifs[at]
		for _, f := range fieldsOfUnit(creatorFields, old.MotifID) {
			switch f {
			case "name":
				m.Name = old.Name
			case "description":
				m.Description = old.Description
			case "anchor_ref":
				m.AnchorRef = old.AnchorRef
			case "role":
				m.Role = old.Role
			default:
				continue
			}
			out.Fields = append(out.Fields, old.MotifID+"."+f)
		}
		out.Set.Motifs = append(out.Set.Motifs, m)
	}
	for _, old := range cur.Motifs {
		if kept[old.MotifID] {
			continue
		}
		if inUse[old.MotifID] {
			return MotifMerge{}, unreadable("bỏ mất %s, motif đang được unit dùng; giữ lại với same_as «%s»", old.MotifID, old.MotifID)
		}
		if len(fieldsOfUnit(creatorFields, old.MotifID)) > 0 {
			return MotifMerge{}, unreadable("bỏ mất %s, motif Creator đã sửa tay; giữ lại với same_as «%s»", old.MotifID, old.MotifID)
		}
	}
	return out, nil
}

// --- one beat's sequence -------------------------------------------------------------

// seqItemReply is one item as the AI writes it; the status is the machine's.
type seqItemReply struct {
	UnitID                 string      `json:"unit_id"`
	PurposeInFlow          string      `json:"purpose_in_flow"`
	Pacing                 Pacing      `json:"pacing"`
	TransitionFromPrevious *Transition `json:"transition_from_previous"`
	MotifRefs              []string    `json:"motif_refs"`
}

type beatSeqReply struct {
	Items []seqItemReply `json:"items"`
}

// ParseBeatSequenceReply reads the AI's items of one beat. The reply is
// unusable, and nothing is stored, when a key or value has the wrong shape, the
// items do not cover the beat's units once each in plan order, a tempo is
// outside its list, a motif is not one of the video's, the unit that opens the
// video has a transition into it, another unit has none, or a transition goes
// through a motif the unit does not carry. Whether via_motif is also on the
// unit before is the machine check of the whole sequence. firstOfVideo says
// whether the beat's first unit opens the video.
func ParseBeatSequenceReply(raw []byte, beat SequenceBeat, firstOfVideo bool, motifs map[string]bool) ([]SeqItem, error) {
	var r beatSeqReply
	if err := decodeReply(raw, &r); err != nil {
		return nil, err
	}
	if len(r.Items) != len(beat.Units) {
		return nil, unreadable("có %d item, beat có %d unit", len(r.Items), len(beat.Units))
	}
	items := make([]SeqItem, 0, len(r.Items))
	for i, ri := range r.Items {
		want := beat.Units[i].UnitID
		if ri.UnitID != want {
			return nil, unreadable("item thứ %d là %s, đáng lẽ là %s: mỗi unit của beat có đúng một item, đúng thứ tự", i+1, ri.UnitID, want)
		}
		if !contains(Tempos, ri.Pacing.Tempo) {
			return nil, unreadable("%s có tempo lạ «%s»", want, ri.Pacing.Tempo)
		}
		for _, m := range ri.MotifRefs {
			if !motifs[m] {
				return nil, unreadable("%s dùng motif lạ «%s»", want, m)
			}
		}
		t := ri.TransitionFromPrevious
		switch {
		case i == 0 && firstOfVideo && t != nil:
			return nil, unreadable("%s mở đầu video, transition_from_previous phải là null", want)
		case !(i == 0 && firstOfVideo) && (t == nil || strings.TrimSpace(t.Type) == ""):
			return nil, unreadable("%s thiếu chuyển cảnh từ unit trước", want)
		}
		if t != nil && t.ViaMotif != "" {
			if !motifs[t.ViaMotif] {
				return nil, unreadable("%s chuyển cảnh qua motif lạ «%s»", want, t.ViaMotif)
			}
			if !contains(ri.MotifRefs, t.ViaMotif) {
				return nil, unreadable("%s chuyển cảnh qua %s nhưng motif_refs không có nó", want, t.ViaMotif)
			}
		}
		it := SeqItem{UnitID: want, PurposeInFlow: ri.PurposeInFlow, Pacing: ri.Pacing, TransitionFromPrevious: t,
			MotifRefs: ri.MotifRefs, Status: UnitPending}
		it.normalize()
		items = append(items, it)
	}
	return items, nil
}

// MergeSequenceItems keeps, in the AI's items, the fields the Creator wrote in
// the beat's stored items, and returns the Creator's fields that still stand.
// Every item goes back to review.
func MergeSequenceItems(fresh []SeqItem, old BeatSequence, creatorFields []string) ([]SeqItem, []string) {
	kept := []string{}
	out := make([]SeqItem, len(fresh))
	for i, it := range fresh {
		at := old.Index(it.UnitID)
		if at >= 0 {
			o := old.Items[at]
			for _, f := range fieldsOfUnit(creatorFields, it.UnitID) {
				switch f {
				case "purpose_in_flow":
					it.PurposeInFlow = o.PurposeInFlow
				case "pacing":
					it.Pacing = o.Pacing
				case "transition_from_previous":
					it.TransitionFromPrevious = o.TransitionFromPrevious
				case "motif_refs":
					it.MotifRefs = o.MotifRefs
				default:
					continue
				}
				kept = append(kept, it.UnitID+"."+f)
			}
		}
		it.Status = UnitPending
		it.normalize()
		out[i] = it
	}
	return out, kept
}

// --- review --------------------------------------------------------------------------

// ParseSequenceReviewReply reads the AI's review of the sequence. A finding
// names a unit of the plan or a motif of the video, and a field the Creator can
// write in it.
func ParseSequenceReviewReply(raw []byte, prev ReviewState, unitIDs, motifIDs map[string]bool) (ReviewReply, error) {
	return parseReviewReply(raw, prev, func(f ReviewReplyItem) string {
		var fields []string
		switch {
		case unitIDs[f.UnitKey]:
			fields = SequenceEditableFields
		case motifIDs[f.UnitKey]:
			fields = MotifEditableFields
		default:
			return "nói về unit hay motif lạ «" + f.UnitKey + "»"
		}
		if f.Field != "" && !contains(fields, f.Field) {
			return "nói về trường lạ «" + f.Field + "»"
		}
		return ""
	})
}
