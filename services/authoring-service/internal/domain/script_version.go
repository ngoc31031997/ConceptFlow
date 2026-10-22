package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ErrNoLockedVersion is returned when a handoff is asked for before any version is locked.
var ErrNoLockedVersion = errors.New("chưa chốt phiên bản kịch bản nào")

// ErrLockNotReady is returned when a version is locked before the conditions hold.
var ErrLockNotReady = errors.New("chưa đủ điều kiện chốt")

// ErrNothingToLock is returned when a draft has no change against the last locked version.
var ErrNothingToLock = errors.New("bản nháp không có thay đổi so với phiên bản đã chốt")

// MaxVersionNoteRunes bounds the note a version is locked with.
const MaxVersionNoteRunes = 200

// Keys of the parts a locked version freezes. Beat parts carry the beat key
// after a colon.
const (
	PartBrief    = "brief"
	PartFacts    = "facts"
	PartStory    = "story"
	PartOrder    = "order"
	PartGlossary = "glossary"
	PartReview   = "review"
	PartBeat     = "beat:"
	PartScript   = "script:"
	PartHandoff  = "handoff:"
)

// VersionPart is one frozen piece of a version with the hash that tells
// whether it changed since.
type VersionPart struct {
	Key     string          `json:"key"`
	SHA     string          `json:"sha"`
	Content json.RawMessage `json:"content"`
}

// NewVersionPart hashes the JSON form of v.
func NewVersionPart(key string, v any) (VersionPart, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return VersionPart{}, fmt.Errorf("freeze %s: %w", key, err)
	}
	sum := sha256.Sum256(raw)
	return VersionPart{Key: key, SHA: hex.EncodeToString(sum[:]), Content: raw}, nil
}

// ScriptVersion is one locked version: an immutable snapshot of everything
// the steps after the script read.
type ScriptVersion struct {
	ProjectID string        `json:"project_id"`
	Version   int           `json:"version"`
	Note      string        `json:"note"`
	LockedAt  time.Time     `json:"locked_at"`
	Parts     []VersionPart `json:"parts"`
}

// Part returns the part with key.
func (v ScriptVersion) Part(key string) (VersionPart, bool) {
	for _, p := range v.Parts {
		if p.Key == key {
			return p, true
		}
	}
	return VersionPart{}, false
}

// PartLabel names a part for the Creator; beatNo gives the number of a beat key.
func PartLabel(key string, beatNo func(string) int) string {
	switch {
	case key == PartBrief:
		return "Creative brief"
	case key == PartFacts:
		return "Dữ kiện"
	case key == PartStory:
		return "Câu chuyện"
	case key == PartOrder:
		return "Thứ tự beat"
	case key == PartGlossary:
		return "Thuật ngữ"
	case key == PartReview:
		return "Ghi chú rà soát"
	}
	for _, p := range []struct{ prefix, label string }{
		{PartBeat, "Beat sheet"}, {PartScript, "Lời thoại"}, {PartHandoff, "Ghi chú bàn giao"},
	} {
		if strings.HasPrefix(key, p.prefix) {
			bk := strings.TrimPrefix(key, p.prefix)
			if n := beatNo(bk); n > 0 {
				return fmt.Sprintf("%s · B%d", p.label, n)
			}
			return p.label + " · " + bk + " (đã xoá)"
		}
	}
	return key
}

// ChangedParts lists, in the order of now, the parts that differ from locked:
// added, changed, or gone.
func ChangedParts(locked ScriptVersion, now []VersionPart) []string {
	have := map[string]string{}
	for _, p := range locked.Parts {
		have[p.Key] = p.SHA
	}
	var out []string
	seen := map[string]bool{}
	for _, p := range now {
		seen[p.Key] = true
		if sha, ok := have[p.Key]; !ok || sha != p.SHA {
			out = append(out, p.Key)
		}
	}
	for _, p := range locked.Parts {
		if !seen[p.Key] {
			out = append(out, p.Key)
		}
	}
	return out
}

// BeatKeysOf returns the beat keys among the part keys, each once, in order of first appearance.
func BeatKeysOf(partKeys []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, k := range partKeys {
		for _, p := range []string{PartBeat, PartScript, PartHandoff} {
			if strings.HasPrefix(k, p) {
				bk := strings.TrimPrefix(k, p)
				if !seen[bk] {
					seen[bk] = true
					out = append(out, bk)
				}
			}
		}
	}
	return out
}

// HandoffBeat is one beat as a role reads it from a locked version.
type HandoffBeat struct {
	Key    string              `json:"key"`
	Number int                 `json:"number"`
	Fn     string              `json:"fn,omitempty"`
	Title  string              `json:"title,omitempty"`
	Goal   string              `json:"goal,omitempty"`
	Next   string              `json:"next,omitempty"`
	Lines  []ScriptLine        `json:"lines,omitempty"`
	Notes  map[string][]string `json:"notes"`
}

// HandoffPackage is what one role reads from a locked version.
type HandoffPackage struct {
	Version  int               `json:"version"`
	LockedAt time.Time         `json:"locked_at"`
	Brief    map[string]string `json:"brief,omitempty"`
	Glossary []GlossaryTerm    `json:"glossary,omitempty"`
	Beats    []HandoffBeat     `json:"beats"`
}

// roleReads says, per role, which parts of the package it reads (FR-32).
type roleReads struct {
	brief, glossary, lines, goal, fn bool
}

var reads = map[string]roleReads{
	HandoffRoleArtWorld:     {brief: true, glossary: true},
	HandoffRoleStoryboard:   {lines: true, goal: true},
	HandoffRoleArtColor:     {fn: true},
	HandoffRoleVoice:        {glossary: true, lines: true},
	HandoffRoleIllustration: {glossary: true},
	HandoffRoleCode:         {},
	HandoffRoleValidate:     {},
}

// HandoffFor returns the part of a locked version the role reads: the sections
// of each beat's note listed for it in HandoffSections, plus the brief, the
// glossary, the lines, the goals or the functions where FR-32 gives them to it.
func HandoffFor(role string, v ScriptVersion) (HandoffPackage, error) {
	r, ok := reads[role]
	if !ok {
		return HandoffPackage{}, fmt.Errorf("%w: vai %q", ErrInvalidUnitEdit, role)
	}
	pkg := HandoffPackage{Version: v.Version, LockedAt: v.LockedAt, Beats: []HandoffBeat{}}
	decode := func(key string, into any) error {
		p, ok := v.Part(key)
		if !ok {
			return nil
		}
		if err := json.Unmarshal(p.Content, into); err != nil {
			return fmt.Errorf("read frozen %s: %w", key, err)
		}
		return nil
	}
	var order []string
	if err := decode(PartOrder, &order); err != nil {
		return pkg, err
	}
	if r.brief {
		var b CreativeBrief
		if err := decode(PartBrief, &b); err != nil {
			return pkg, err
		}
		pkg.Brief = map[string]string{"tone": b.Tone, "out_of_scope": b.OutOfScope}
	}
	if r.glossary {
		var g Glossary
		if err := decode(PartGlossary, &g); err != nil {
			return pkg, err
		}
		pkg.Glossary = g.Terms
	}
	for i, key := range order {
		hb := HandoffBeat{Key: key, Number: i + 1, Notes: map[string][]string{}}
		var beat Beat
		if err := decode(PartBeat+key, &beat); err != nil {
			return pkg, err
		}
		if r.fn {
			hb.Fn = beat.Fn
		}
		if r.goal {
			hb.Fn, hb.Title, hb.Goal, hb.Next = beat.Fn, beat.Title, beat.Goal, beat.Next
		}
		if r.lines {
			var s ScriptBeat
			if err := decode(PartScript+key, &s); err != nil {
				return pkg, err
			}
			hb.Lines = s.Lines
		}
		var note HandoffNote
		if err := decode(PartHandoff+key, &note); err != nil {
			return pkg, err
		}
		note = note.Normalized()
		for _, sec := range HandoffSections {
			for _, rr := range sec.Roles {
				if rr == role {
					hb.Notes[sec.Key] = note.Section(sec.Key)
				}
			}
		}
		pkg.Beats = append(pkg.Beats, hb)
	}
	return pkg, nil
}
