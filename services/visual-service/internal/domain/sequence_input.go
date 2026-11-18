package domain

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Sources of a beat's sequence. Each has one hash, so a beat whose source
// changed since its items were written reads as stale.
const (
	// SeqSourcePlan is the beat's own units in the locked visual plan.
	SeqSourcePlan = "plan"
	// SeqSourcePrev is the last unit before the beat, which the beat's first
	// transition comes from.
	SeqSourcePrev = "prev"
	// SeqSourceScript is the beat's card, lines and handoff in the locked script.
	SeqSourceScript = "script"
	// SeqSourceMotion is the motion language of the locked visual concept.
	SeqSourceMotion = "motion"
	// SeqSourceMotifs is the video's motifs.
	SeqSourceMotifs = "motifs"
)

// Sources of the motifs.
const (
	// MotifSourcePlan is every unit of the locked visual plan.
	MotifSourcePlan = "plan"
	// MotifSourceRep is the representation rules of the locked visual concept.
	MotifSourceRep = "rep"
	// MotifSourceBeats is the beats' order and functions in the locked script.
	MotifSourceBeats = "beats"
)

// SeqSourceLabels names each source of a beat or of the motifs for the Creator.
var SeqSourceLabels = map[string]string{
	SeqSourcePlan:    "Unit của beat trong kế hoạch hình đã chốt",
	SeqSourcePrev:    "Unit cuối của beat trước",
	SeqSourceScript:  "Lời thoại và ghi chú bàn giao của beat",
	SeqSourceMotion:  "Nguyên tắc chuyển động đã chốt",
	SeqSourceMotifs:  "Danh sách motif",
	MotifSourceRep:   "Quy tắc thể hiện đã chốt",
	MotifSourceBeats: "Thứ tự và chức năng các beat",
}

var (
	seqSourceOrder   = []string{SeqSourcePlan, SeqSourcePrev, SeqSourceScript, SeqSourceMotion, SeqSourceMotifs}
	motifSourceOrder = []string{MotifSourcePlan, MotifSourceRep, MotifSourceBeats}
)

// anchorPrefix is the start of every rule id a motif can anchor to.
const anchorPrefix = "representation_rules."

// SequenceBeat is one beat of the sequence: its card from the locked script and
// its units from the locked plan, in plan order.
type SequenceBeat struct {
	Beat
	// Pace is the visual concept's motion speed of the beat; "" when none is set.
	Pace  string
	Units []DocumentUnit
}

// SequenceInput is everything step 10 reads: the latest locked visual plan, the
// latest locked visual concept and the latest locked script.
type SequenceInput struct {
	// Script is the latest locked script; its picked references are empty.
	Script ConceptInput
	// ConceptVersion and Concept are the latest locked visual concept.
	ConceptVersion int
	Concept        Document
	// AnchorRefs are the representation rules a motif can anchor to, with their text.
	AnchorRefs map[string]string
	// PlanVersion and Plan are the latest locked visual plan.
	PlanVersion int
	Plan        PlanDocument
	// Beats are the beats that have units in the plan, in plan order.
	Beats []SequenceBeat
	// Outdated says why the plan was written from an older script or visual
	// concept than the latest locked ones; empty when it was not.
	Outdated []string

	beatHashes  map[string]map[string]string
	motifHashes map[string]string
}

// NewSequenceInput reads the latest locked script, the latest locked visual
// concept document and the latest locked plan document.
func NewSequenceInput(sv ScriptVersion, conceptVersion int, conceptDoc []byte, planVersion int, planDoc []byte) (SequenceInput, error) {
	script, err := NewConceptInput(sv, nil)
	if err != nil {
		return SequenceInput{}, err
	}
	var concept Document
	if err := json.Unmarshal(conceptDoc, &concept); err != nil {
		return SequenceInput{}, fmt.Errorf("đọc ngôn ngữ hình ảnh đã chốt v%d: %w", conceptVersion, err)
	}
	var plan PlanDocument
	if err := json.Unmarshal(planDoc, &plan); err != nil {
		return SequenceInput{}, fmt.Errorf("đọc kế hoạch hình đã chốt v%d: %w", planVersion, err)
	}
	in := SequenceInput{Script: script, ConceptVersion: conceptVersion, Concept: concept, PlanVersion: planVersion, Plan: plan,
		AnchorRefs: map[string]string{}, Outdated: []string{}}
	for id, text := range RuleRefs(concept) {
		if strings.HasPrefix(id, anchorPrefix) {
			in.AnchorRefs[id] = text
		}
	}
	if plan.SourceVersions.Script != script.ScriptVersion {
		in.Outdated = append(in.Outdated, fmt.Sprintf("kế hoạch hình viết từ kịch bản v%d, kịch bản đã chốt v%d", plan.SourceVersions.Script, script.ScriptVersion))
	}
	if plan.SourceVersions.VisualConcept != conceptVersion {
		in.Outdated = append(in.Outdated, fmt.Sprintf("kế hoạch hình viết từ ngôn ngữ hình ảnh v%d, ngôn ngữ hình ảnh đã chốt v%d", plan.SourceVersions.VisualConcept, conceptVersion))
	}
	in.groupBeats()
	in.hashSources(sv)
	return in, nil
}

// groupBeats puts the plan's units under their beats, in plan order.
func (in *SequenceInput) groupBeats() {
	cards := map[string]Beat{}
	for _, b := range in.Script.Beats {
		cards[b.Key] = b
	}
	pace := map[string]string{}
	for _, p := range in.Concept.MotionLanguage.Pace {
		for _, k := range p.BeatKeys {
			pace[k] = p.Pace
		}
	}
	at := map[string]int{}
	for _, u := range in.Plan.Units {
		i, ok := at[u.BeatID]
		if !ok {
			card, found := cards[u.BeatID]
			if !found {
				card = Beat{Key: u.BeatID}
			}
			in.Beats = append(in.Beats, SequenceBeat{Beat: card, Pace: pace[u.BeatID]})
			i = len(in.Beats) - 1
			at[u.BeatID] = i
		}
		in.Beats[i].Units = append(in.Beats[i].Units, u)
	}
}

// unitHash hashes a plan unit without its position, so units moving because an
// earlier beat gained or lost one do not read as changed.
func unitHash(units ...DocumentUnit) string {
	var b strings.Builder
	for _, u := range units {
		u.Order = 0
		raw, _ := json.Marshal(u)
		b.Write(raw)
		b.WriteByte('\n')
	}
	return hashText(b.String())
}

func (in *SequenceInput) hashSources(sv ScriptVersion) {
	shas := map[string]string{}
	for _, p := range sv.Parts {
		shas[p.Key] = p.SHA
	}
	motion, _ := json.Marshal(in.Concept.MotionLanguage)
	in.beatHashes = map[string]map[string]string{}
	var prev *DocumentUnit
	for _, b := range in.Beats {
		h := map[string]string{
			SeqSourcePlan:   unitHash(b.Units...),
			SeqSourceScript: hashText("beat:" + shas["beat:"+b.Key] + "\nscript:" + shas["script:"+b.Key] + "\nhandoff:" + shas["handoff:"+b.Key]),
			SeqSourceMotion: hashText(string(motion)),
		}
		if prev != nil {
			h[SeqSourcePrev] = unitHash(*prev)
		} else {
			h[SeqSourcePrev] = hashText("")
		}
		in.beatHashes[b.Key] = h
		last := b.Units[len(b.Units)-1]
		prev = &last
	}
	rep, _ := json.Marshal(in.Concept.RepresentationRules)
	var beats strings.Builder
	for _, b := range in.Script.Beats {
		beats.WriteString(b.Key + "=" + b.Fn + "\n")
	}
	in.motifHashes = map[string]string{
		MotifSourcePlan:  unitHash(in.Plan.Units...),
		MotifSourceRep:   hashText(string(rep)),
		MotifSourceBeats: hashText(beats.String()),
	}
}

// MotifsHash is the hash of the motifs a beat's items are written against: what
// each motif is and where it anchors, not the AI's hint of where it goes.
func MotifsHash(s MotifSet) string {
	var b strings.Builder
	for _, m := range s.Motifs {
		b.WriteString(strings.Join([]string{m.MotifID, m.Name, m.Description, m.AnchorRef, m.Role}, "\x1f"))
		b.WriteByte('\n')
	}
	return hashText(b.String())
}

// BeatHashes is the hash of every source beat key's items are written from.
func (in SequenceInput) BeatHashes(beatKey string, motifs MotifSet) map[string]string {
	out := map[string]string{SeqSourceMotifs: MotifsHash(motifs)}
	for k, v := range in.beatHashes[beatKey] {
		out[k] = v
	}
	return out
}

// BeatStaleSources lists the sources, by constant, that differ between the
// hashes a beat's items were written from and the hashes now.
func (in SequenceInput) BeatStaleSources(beatKey string, stored map[string]string, motifs MotifSet) []string {
	return staleSources(seqSourceOrder, stored, in.BeatHashes(beatKey, motifs))
}

// MotifHashes is the hash of every source the motifs are written from.
func (in SequenceInput) MotifHashes() map[string]string {
	out := make(map[string]string, len(in.motifHashes))
	for k, v := range in.motifHashes {
		out[k] = v
	}
	return out
}

// MotifStaleSources lists the sources that differ between the hashes the motifs
// were written from and the hashes now.
func (in SequenceInput) MotifStaleSources(stored map[string]string) []string {
	return staleSources(motifSourceOrder, stored, in.motifHashes)
}

func staleSources(order []string, stored, now map[string]string) []string {
	var out []string
	for _, s := range order {
		if stored[s] != now[s] {
			out = append(out, s)
		}
	}
	return out
}

// Beat returns the sequence beat with key.
func (in SequenceInput) Beat(key string) (SequenceBeat, bool) {
	for _, b := range in.Beats {
		if b.Key == key {
			return b, true
		}
	}
	return SequenceBeat{}, false
}

// LocateUnit returns the beat key of plan unit id and whether it is the first
// unit of the video.
func (in SequenceInput) LocateUnit(unitID string) (string, bool, bool) {
	for i, u := range in.Plan.Units {
		if u.UnitID == unitID {
			return u.BeatID, i == 0, true
		}
	}
	return "", false, false
}

// PreviousUnit returns the id of the plan unit right before unitID, or "".
func (in SequenceInput) PreviousUnit(unitID string) string {
	for i, u := range in.Plan.Units {
		if u.UnitID == unitID && i > 0 {
			return in.Plan.Units[i-1].UnitID
		}
	}
	return ""
}
