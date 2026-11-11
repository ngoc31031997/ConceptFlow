package domain

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Sources of the visual plan. A beat's units are written from the beat's own
// script, handoff and beat card, from the brief, the facts and the glossary, and
// from the locked visual concept; each has one hash, so a beat whose source
// changed since it was planned reads as stale.
const (
	PlanSourceBeat     = "beat"
	PlanSourceBrief    = "brief"
	PlanSourceFacts    = "facts"
	PlanSourceGlossary = "glossary"
	PlanSourceConcept  = "concept"
)

// PlanSourceLabels names each source for the Creator.
var PlanSourceLabels = map[string]string{
	PlanSourceBeat:     "Lời thoại và ghi chú bàn giao của beat",
	PlanSourceBrief:    "Creative brief",
	PlanSourceFacts:    "Hồ sơ nghiên cứu",
	PlanSourceGlossary: "Bảng thuật ngữ",
	PlanSourceConcept:  "Ngôn ngữ hình ảnh đã chốt",
}

// planSourceOrder is the order sources are listed in.
var planSourceOrder = []string{PlanSourceBeat, PlanSourceBrief, PlanSourceFacts, PlanSourceGlossary, PlanSourceConcept}

// PlanInput is everything step 9 reads: the locked script, the locked visual
// concept and the video's own speaking pace.
type PlanInput struct {
	// Script is the locked script as step 8 reads it; its picked references are empty.
	Script ConceptInput
	// ConceptVersion is the version of the locked visual concept.
	ConceptVersion int
	// Concept is that locked visual concept.
	Concept Document
	// RuleRefs are the rules of the concept a unit can point at, with their text.
	RuleRefs map[string]string
	// SecondsPerWord is the speaking pace: the brief's duration over the words of
	// the whole script. 0 when it cannot be worked out.
	SecondsPerWord float64
	// BeatHashes is the hash of each beat's own sources, by beat key.
	BeatHashes map[string]string
	// Hashes is the hash of each shared source, by Source constant.
	Hashes map[string]string
}

// NewPlanInput reads the locked script version and the locked visual concept
// document (with its version number).
func NewPlanInput(sv ScriptVersion, conceptVersion int, conceptDoc []byte) (PlanInput, error) {
	script, err := NewConceptInput(sv, nil)
	if err != nil {
		return PlanInput{}, err
	}
	var doc Document
	if err := json.Unmarshal(conceptDoc, &doc); err != nil {
		return PlanInput{}, fmt.Errorf("đọc ngôn ngữ hình ảnh đã chốt v%d: %w", conceptVersion, err)
	}
	in := PlanInput{Script: script, ConceptVersion: conceptVersion, Concept: doc, RuleRefs: RuleRefs(doc),
		BeatHashes: map[string]string{}}
	words := 0
	for _, lines := range script.Lines {
		for _, l := range lines {
			words += len(strings.Fields(l.Text))
		}
	}
	if words > 0 && script.Brief.DurationSeconds > 0 {
		in.SecondsPerWord = float64(script.Brief.DurationSeconds) / float64(words)
	}
	shas := map[string]string{}
	for _, p := range sv.Parts {
		shas[p.Key] = p.SHA
	}
	for _, b := range script.Beats {
		in.BeatHashes[b.Key] = hashText("beat:" + shas["beat:"+b.Key] + "\nscript:" + shas["script:"+b.Key] + "\nhandoff:" + shas["handoff:"+b.Key])
	}
	in.Hashes = map[string]string{
		PlanSourceBrief:    script.Hashes[SourceBrief],
		PlanSourceFacts:    script.Hashes[SourceResearch],
		PlanSourceGlossary: hashText("glossary=" + shas["glossary"]),
		PlanSourceConcept:  hashText(string(conceptDoc)),
	}
	return in, nil
}

// HashesFor is the hash of every source the beat's units are written from.
func (in PlanInput) HashesFor(beatKey string) map[string]string {
	out := map[string]string{PlanSourceBeat: in.BeatHashes[beatKey]}
	for k, v := range in.Hashes {
		out[k] = v
	}
	return out
}

// StaleSources lists the sources, by Source constant, that differ between the
// hashes a beat was planned from and the hashes now. A beat stored without
// hashes reads as stale on every source.
func (in PlanInput) StaleSources(beatKey string, stored map[string]string) []string {
	now := in.HashesFor(beatKey)
	var out []string
	for _, s := range planSourceOrder {
		if stored[s] != now[s] {
			out = append(out, s)
		}
	}
	return out
}

// Beat returns the beat with key.
func (in PlanInput) Beat(key string) (Beat, bool) {
	for _, b := range in.Script.Beats {
		if b.Key == key {
			return b, true
		}
	}
	return Beat{}, false
}

// BeatOfLine returns the beat key of a line id such as b2.3.
func BeatOfLine(lineID string) string {
	if i := strings.IndexByte(lineID, '.'); i > 0 {
		return lineID[:i]
	}
	return ""
}
