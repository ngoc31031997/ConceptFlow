package domain

import (
	"fmt"
	"strings"
)

// IDs of the machine checks of the whole sequence. They can be accepted with a
// reason; what fails an item's or a motif's own checks cannot.
const (
	CheckSeqVia     = "m-seq-via"
	CheckMotifCount = "m-motif-count"
)

// minMotifAppearance is how many units a picture has to appear in to be a motif.
const minMotifAppearance = 2

// CheckSeqItem lists what fails in an item; an item with problems cannot be
// approved. first says whether its unit opens the video; motifs are the motif
// ids the video has.
func CheckSeqItem(it SeqItem, first bool, motifs map[string]bool) []string {
	var p problems
	p.need(it.PurposeInFlow, "Thiếu vai trò trong mạch")
	p.need(it.Pacing.Name, "Thiếu tên nhịp")
	p.enum(it.Pacing.Tempo, Tempos, "tốc độ của nhịp")
	switch {
	case first && it.TransitionFromPrevious != nil:
		p.add("Unit đầu video không có chuyển cảnh vào")
	case !first && it.TransitionFromPrevious == nil:
		p.add("Thiếu chuyển cảnh từ unit trước")
	case !first:
		p.need(it.TransitionFromPrevious.Type, "Thiếu kiểu chuyển cảnh")
	}
	seen := map[string]bool{}
	for _, m := range it.MotifRefs {
		if !motifs[m] {
			p.addf("Motif %s không có trong danh sách motif", m)
		}
		if seen[m] {
			p.addf("Motif %s bị ghi hai lần", m)
		}
		seen[m] = true
	}
	if t := it.TransitionFromPrevious; t != nil && t.ViaMotif != "" && !seen[t.ViaMotif] {
		p.addf("Motif nối %s phải có trong motif của unit", t.ViaMotif)
	}
	return p.list
}

// CheckMotif lists what fails in a motif; motifs with problems cannot be
// approved. anchors are the representation rules of the locked visual concept.
func CheckMotif(m Motif, anchors map[string]string) []string {
	var p problems
	if !motifIDPattern.MatchString(m.MotifID) {
		p.addf("mã motif %q không đúng dạng MOTIF-01", m.MotifID)
	}
	p.need(m.Name, "Thiếu tên motif")
	p.need(m.Description, "Thiếu mô tả motif")
	p.need(m.Role, "Thiếu vai trò của motif")
	if strings.TrimSpace(m.AnchorRef) == "" {
		p.add("Motif chưa neo vào quy tắc thể hiện nào của bước 8")
	} else if _, ok := anchors[m.AnchorRef]; !ok {
		p.addf("Neo %q không có trong quy tắc thể hiện đã chốt", m.AnchorRef)
	}
	return p.list
}

// SequenceCoverage says which units of the locked plan have an item.
type SequenceCoverage struct {
	UnitsTotal     int `json:"units_total"`
	UnitsSequenced int `json:"units_sequenced"`
	// Missing lists the plan units with no item yet.
	Missing []string `json:"missing"`
	// Extra lists stored items of units the plan no longer has.
	Extra []string `json:"extra"`
}

// Complete reports whether every plan unit has exactly its item.
func (c SequenceCoverage) Complete() bool {
	return c.UnitsTotal > 0 && len(c.Missing) == 0 && len(c.Extra) == 0
}

// ComputeSequenceCoverage matches the stored items against the plan's units.
func ComputeSequenceCoverage(in SequenceInput, beats map[string]BeatSequence) SequenceCoverage {
	cov := SequenceCoverage{Missing: []string{}, Extra: []string{}}
	planned := map[string]string{}
	for _, u := range in.Plan.Units {
		planned[u.UnitID] = u.BeatID
	}
	for _, u := range in.Plan.Units {
		cov.UnitsTotal++
		if beats[u.BeatID].Index(u.UnitID) >= 0 {
			cov.UnitsSequenced++
		} else {
			cov.Missing = append(cov.Missing, u.UnitID)
		}
	}
	for key, s := range beats {
		for _, it := range s.Items {
			if planned[it.UnitID] != key {
				cov.Extra = append(cov.Extra, it.UnitID)
			}
		}
	}
	return cov
}

// ItemOf returns the stored item of plan unit id.
func ItemOf(in SequenceInput, beats map[string]BeatSequence, unitID string) (SeqItem, bool) {
	beat, _, ok := in.LocateUnit(unitID)
	if !ok {
		return SeqItem{}, false
	}
	s := beats[beat]
	if i := s.Index(unitID); i >= 0 {
		return s.Items[i], true
	}
	return SeqItem{}, false
}

// MotifAppearances lists, for each motif, the plan units whose items name it,
// in plan order.
func MotifAppearances(in SequenceInput, beats map[string]BeatSequence, motifs MotifSet) map[string][]string {
	out := make(map[string][]string, len(motifs.Motifs))
	for _, m := range motifs.Motifs {
		out[m.MotifID] = []string{}
	}
	for _, u := range in.Plan.Units {
		it, ok := ItemOf(in, beats, u.UnitID)
		if !ok {
			continue
		}
		for _, m := range it.MotifRefs {
			if _, known := out[m]; known {
				out[m] = append(out[m], u.UnitID)
			}
		}
	}
	return out
}

// SequenceChecks are the checks of the whole sequence, worked out on every read
// and never stored. Detail names what fails; a Creator's acceptance holds while
// Detail stays the same.
func SequenceChecks(in SequenceInput, beats map[string]BeatSequence, motifs MotifSet) []MachineCheck {
	var via []string
	for i, u := range in.Plan.Units {
		if i == 0 {
			continue
		}
		it, ok := ItemOf(in, beats, u.UnitID)
		if !ok || it.TransitionFromPrevious == nil || it.TransitionFromPrevious.ViaMotif == "" {
			continue
		}
		prevID := in.Plan.Units[i-1].UnitID
		prev, ok := ItemOf(in, beats, prevID)
		if !ok || !prev.HasMotif(it.TransitionFromPrevious.ViaMotif) {
			via = append(via, fmt.Sprintf("%s: nối bằng %s nhưng %s không có motif này", u.UnitID, it.TransitionFromPrevious.ViaMotif, prevID))
		}
	}
	var few []string
	seen := MotifAppearances(in, beats, motifs)
	for _, m := range motifs.Motifs {
		if n := len(seen[m.MotifID]); n < minMotifAppearance {
			few = append(few, fmt.Sprintf("%s xuất hiện ở %d unit", m.MotifID, n))
		}
	}
	return []MachineCheck{
		{ID: CheckSeqVia, Label: "Motif nối chuyển cảnh có ở cả unit trước lẫn unit sau", OK: len(via) == 0, Detail: strings.Join(via, "; ")},
		{ID: CheckMotifCount, Label: fmt.Sprintf("Mỗi motif xuất hiện ở ít nhất %d unit", minMotifAppearance), OK: len(few) == 0, Detail: strings.Join(few, "; ")},
	}
}
