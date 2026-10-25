package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// UnitKindBeat names one beat of the video, keyed b1, b2... The key stays with
// the beat when it moves, so later steps can keep pointing at it. A later
// step records it as the input of the dialogue it wrote ("beat:b2").
const UnitKindBeat = "beat"

// Fields of a beat the Creator can write; an AI run keeps the ones marked as
// the Creator's.
const (
	BeatFieldFn      = "fn"
	BeatFieldTitle   = "title"
	BeatFieldGoal    = "goal"
	BeatFieldContent = "content"
	BeatFieldFacts   = "facts"
	BeatFieldDur     = "dur"
	BeatFieldNext    = "next"
)

// BeatTextFields are the beat fields a hand edit through the unit API sets.
var BeatTextFields = []string{BeatFieldTitle, BeatFieldGoal, BeatFieldContent, BeatFieldNext}

// BeatFns are the functions a beat can have, in the order a video usually runs them.
var BeatFns = []string{"Hook", "Đặt vấn đề", "Giải thích cơ chế", "Cao trào", "Mở rộng hệ quả", "Payoff"}

const (
	beatFnPayoff = "Payoff"
)

// Limits of a beat sheet.
const (
	// MaxBeats is the most beats one sheet may hold.
	MaxBeats = 20
	// MinBeatSeconds is the shortest a beat may be.
	MinBeatSeconds = 5
	// MaxBeatSeconds is the longest a beat may be.
	MaxBeatSeconds = 600
	// BeatDurationStep is how far the duration buttons move a beat.
	BeatDurationStep = 5
	// MaxHookSeconds is the longest a hook should be.
	MaxHookSeconds = 20
)

// ErrBeatReplyUnusable marks a model reply the beats cannot be read from.
var ErrBeatReplyUnusable = errors.New("câu trả lời của AI không dùng được làm beat sheet")

// IsBeatFn reports whether fn is one of BeatFns.
func IsBeatFn(fn string) bool {
	for _, f := range BeatFns {
		if f == fn {
			return true
		}
	}
	return false
}

// Beat is the content of a beat unit. No dialogue is written at this step.
type Beat struct {
	// Fn is the beat's function, one of BeatFns.
	Fn    string `json:"fn"`
	Title string `json:"title"`
	// Goal is what the beat moves forward for the viewer.
	Goal    string `json:"goal"`
	Content string `json:"content"`
	// Facts are the codes of the facts the beat uses, each once.
	Facts []string `json:"facts"`
	// Dur is the beat's target length in seconds.
	Dur int `json:"dur"`
	// Next says how the beat joins the one after it; empty for the last beat.
	Next string `json:"next"`
	// Step is the story line the beat was built from; empty for a beat the
	// Creator added by hand.
	Step string `json:"step"`
}

// Written reports whether the AI or the Creator has written the beat's goal
// and content; a beat of the skeleton is not written yet.
func (b Beat) Written() bool {
	return strings.TrimSpace(b.Goal) != "" && strings.TrimSpace(b.Content) != ""
}

// BeatSheetGate is what the sheet holds besides its beats.
type BeatSheetGate struct {
	// NextSeq is the number the next beat key takes; keys are never reused.
	NextSeq int
	// SkeletonApproved is true once the Creator approved the functions and facts
	// of the beats; only then does the AI write their content.
	SkeletonApproved bool
	// Rebalanced is how many beats had their length changed by the last run to
	// bring the total inside the target.
	Rebalanced int
}

// BeatKey is the key of the beat with sequence number n.
func BeatKey(n int) string { return "b" + strconv.Itoa(n) }

// IsBeatKey reports whether key has the form b1, b2...
func IsBeatKey(key string) bool {
	if len(key) < 2 || key[0] != 'b' {
		return false
	}
	n, err := strconv.Atoi(key[1:])
	return err == nil && n >= 1 && key[1] != '0'
}

// Validate checks a beat the way the Creator's edit and the AI's reply both
// must satisfy it, with known telling which fact codes exist. An empty goal is
// allowed here (the beat just cannot be approved); everything else must hold.
func (b Beat) Validate(known func(id string) bool) error {
	if !IsBeatFn(b.Fn) {
		return fmt.Errorf("chức năng %q không phải một trong: %s", b.Fn, strings.Join(BeatFns, ", "))
	}
	if strings.TrimSpace(b.Title) == "" {
		return errors.New("beat cần có tên")
	}
	if b.Dur < MinBeatSeconds || b.Dur > MaxBeatSeconds {
		return fmt.Errorf("thời lượng beat phải từ %d đến %d giây", MinBeatSeconds, MaxBeatSeconds)
	}
	for _, id := range b.Facts {
		if known != nil && !known(id) {
			return fmt.Errorf("dữ kiện %s không có trong bảng dữ kiện", id)
		}
	}
	return nil
}

// BeatDraft is a beat as the AI wrote it. Key names the stored beat it
// rewrites, or is empty for a new one.
type BeatDraft struct {
	Key string
	Beat
}

type beatWire struct {
	Key     string   `json:"key"`
	Fn      string   `json:"fn"`
	Title   string   `json:"title"`
	Goal    string   `json:"goal"`
	Content string   `json:"content"`
	Facts   []string `json:"facts"`
	Dur     int      `json:"dur"`
	Next    string   `json:"next"`
}

// ParseBeatsReply reads the beats out of a model reply {"beats":[...]}. Every
// beat must have a known function, a title, a goal, a content, a duration in
// range and only facts known() knows; a key, when given, must have the form
// b1 and appear once. A reply that fails any of that stores nothing.
func ParseBeatsReply(reply string, known func(id string) bool) ([]BeatDraft, error) {
	var wire struct {
		Beats []beatWire `json:"beats"`
	}
	if err := json.Unmarshal([]byte(reply), &wire); err != nil {
		return nil, fmt.Errorf("%w: khối JSON không đọc được (%v)", ErrBeatReplyUnusable, err)
	}
	if len(wire.Beats) == 0 {
		return nil, fmt.Errorf("%w: không có beat nào", ErrBeatReplyUnusable)
	}
	if len(wire.Beats) > MaxBeats {
		return nil, fmt.Errorf("%w: nhiều hơn %d beat", ErrBeatReplyUnusable, MaxBeats)
	}
	out := make([]BeatDraft, 0, len(wire.Beats))
	seen := map[string]bool{}
	for i, w := range wire.Beats {
		d := BeatDraft{Key: strings.TrimSpace(w.Key), Beat: Beat{
			Fn: strings.TrimSpace(w.Fn), Title: strings.TrimSpace(w.Title), Goal: strings.TrimSpace(w.Goal),
			Content: strings.TrimSpace(w.Content), Dur: w.Dur, Next: strings.TrimSpace(w.Next), Facts: []string{},
		}}
		if d.Key != "" {
			if !IsBeatKey(d.Key) || seen[d.Key] {
				return nil, fmt.Errorf("%w: beat thứ %d có khoá %q không hợp lệ hoặc trùng", ErrBeatReplyUnusable, i+1, w.Key)
			}
			seen[d.Key] = true
		}
		if d.Goal == "" || d.Content == "" {
			return nil, fmt.Errorf("%w: beat thứ %d thiếu mục tiêu hoặc nội dung", ErrBeatReplyUnusable, i+1)
		}
		for _, raw := range w.Facts {
			id, ok := NormalizeFactID(raw)
			if !ok {
				return nil, fmt.Errorf("%w: beat thứ %d trích dữ kiện %q không hợp lệ", ErrBeatReplyUnusable, i+1, raw)
			}
			d.Facts = uniqueFacts(d.Facts, []string{id})
		}
		if err := d.Validate(known); err != nil {
			return nil, fmt.Errorf("%w: beat thứ %d: %v", ErrBeatReplyUnusable, i+1, err)
		}
		out = append(out, d)
	}
	return out, nil
}

// BeatCheck is one line of the structure check.
type BeatCheck struct {
	OK   bool   `json:"ok"`
	Text string `json:"text"`
}

// FormatSeconds writes a length as "4:28".
func FormatSeconds(s int) string { return fmt.Sprintf("%d:%02d", s/60, s%60) }

// CheckBeatSheet runs the structure checks over beats in the order they run:
// the total length inside the target, a hook of at most 20 seconds, every
// fact the script must state in some beat, a goal in every beat and a payoff
// last. targetLabel names the target for the first line.
func CheckBeatSheet(beats []Beat, mustFacts []string, lo, hi int, targetLabel string) []BeatCheck {
	total := 0
	used := map[string]bool{}
	allGoals := len(beats) > 0
	for _, b := range beats {
		total += b.Dur
		for _, id := range b.Facts {
			used[id] = true
		}
		if strings.TrimSpace(b.Goal) == "" {
			allGoals = false
		}
	}
	var missing []string
	for _, id := range mustFacts {
		if !used[id] {
			missing = append(missing, id)
		}
	}
	hook, last := 0, ""
	if len(beats) > 0 {
		hook, last = beats[0].Dur, beats[len(beats)-1].Fn
	}
	mustLine := fmt.Sprintf("Dữ kiện bắt buộc có trong beat: %d/%d", len(mustFacts)-len(missing), len(mustFacts))
	if len(missing) > 0 {
		mustLine += " · thiếu " + strings.Join(missing, ", ")
	}
	return []BeatCheck{
		{OK: total >= lo && total <= hi, Text: fmt.Sprintf("Tổng %s · mục tiêu %s", FormatSeconds(total), targetLabel)},
		{OK: len(beats) > 0 && hook <= MaxHookSeconds, Text: fmt.Sprintf("Hook %d s · nên ≤ %d s", hook, MaxHookSeconds)},
		{OK: len(missing) == 0, Text: mustLine},
		{OK: allGoals, Text: "Beat nào cũng có mục tiêu"},
		{OK: last == beatFnPayoff, Text: "Beat cuối là payoff"},
	}
}
