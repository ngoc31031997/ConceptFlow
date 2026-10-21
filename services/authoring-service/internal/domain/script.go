package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// UnitKindScriptBeat is the dialogue of one beat. Its key is the key of the
// beat it speaks for (b1, b2...), so the dialogue stays with the beat when the
// beats are reordered.
const UnitKindScriptBeat = "script_beat"

// Limits of a beat's dialogue.
const (
	// MaxScriptLines is the most lines one beat may hold.
	MaxScriptLines = 40
	// MaxLineRunes is the longest a single line may be.
	MaxLineRunes = 600
	// MaxLineWords is the most words a spoken line should have; longer lines are flagged.
	MaxLineWords = 28
	// OpeningSeconds is how soon the opening line must set up the question.
	OpeningSeconds = 7.0
)

// ErrScriptReplyUnusable marks a model reply the dialogue cannot be read from.
var ErrScriptReplyUnusable = errors.New("câu trả lời của AI không dùng được làm lời thoại")

// ScriptLine is one spoken sentence. ID is "<beat key>.<n>", never reused
// inside its beat, so a voice file and a suggestion can keep pointing at it.
type ScriptLine struct {
	ID    string   `json:"id"`
	Text  string   `json:"text"`
	Facts []string `json:"facts"`
}

// ScriptBeat is the content of a script_beat unit.
type ScriptBeat struct {
	Lines []ScriptLine `json:"lines"`
	// NextSeq is the number the next line of the beat takes.
	NextSeq int `json:"next_seq"`
}

var scriptLineIDPattern = regexp.MustCompile(`^(b[1-9][0-9]*)\.([1-9][0-9]*)$`)

// ScriptLineID is the id of line n of the beat with key beatKey.
func ScriptLineID(beatKey string, n int) string { return beatKey + "." + strconv.Itoa(n) }

// ScriptLineBeat returns the beat key of a line id, and false when id is not one.
func ScriptLineBeat(id string) (string, bool) {
	m := scriptLineIDPattern.FindStringSubmatch(id)
	if m == nil {
		return "", false
	}
	return m[1], true
}

// LineTextSHA is the hash voice records use to tell whether the text a file
// was spoken from is still the text of the line.
func LineTextSHA(text string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(text)))
	return hex.EncodeToString(sum[:])
}

// ValidateLineText checks the wording of one line.
func ValidateLineText(text string) error {
	t := strings.TrimSpace(text)
	if t == "" {
		return errors.New("câu không được để trống")
	}
	if n := len([]rune(t)); n > MaxLineRunes {
		return fmt.Errorf("câu dài %d ký tự, tối đa %d", n, MaxLineRunes)
	}
	return nil
}

// Validate checks a beat's dialogue the way a hand edit and the AI's reply
// both must satisfy it. known tells which fact codes exist (nil accepts any).
func (s ScriptBeat) Validate(known func(id string) bool) error {
	if len(s.Lines) > MaxScriptLines {
		return fmt.Errorf("beat có %d câu, tối đa %d", len(s.Lines), MaxScriptLines)
	}
	seen := map[string]bool{}
	for _, l := range s.Lines {
		if !scriptLineIDPattern.MatchString(l.ID) || seen[l.ID] {
			return fmt.Errorf("mã câu %q không hợp lệ hoặc trùng", l.ID)
		}
		seen[l.ID] = true
		if err := ValidateLineText(l.Text); err != nil {
			return fmt.Errorf("câu %s: %v", l.ID, err)
		}
		for _, id := range l.Facts {
			if known != nil && !known(id) {
				return fmt.Errorf("câu %s trích dữ kiện %s không có trong bảng dữ kiện", l.ID, id)
			}
		}
	}
	return nil
}

// ScriptLineDraft is a line as the AI wrote it, before it has an id.
type ScriptLineDraft struct {
	Text  string
	Facts []string
}

// ParseScriptReply reads the lines out of a model reply {"lines":[...]}. Every
// line needs text and only facts known() knows; a reply that fails any of that
// stores nothing.
func ParseScriptReply(reply string, known func(id string) bool) ([]ScriptLineDraft, error) {
	obj, ok := jsonObjectIn(reply)
	if !ok {
		return nil, fmt.Errorf("%w: không có khối JSON", ErrScriptReplyUnusable)
	}
	var wire struct {
		Lines []struct {
			Text  string   `json:"text"`
			Facts []string `json:"facts"`
		} `json:"lines"`
	}
	if err := json.Unmarshal([]byte(obj), &wire); err != nil {
		return nil, fmt.Errorf("%w: khối JSON không đọc được (%v)", ErrScriptReplyUnusable, err)
	}
	if len(wire.Lines) == 0 {
		return nil, fmt.Errorf("%w: không có câu nào", ErrScriptReplyUnusable)
	}
	if len(wire.Lines) > MaxScriptLines {
		return nil, fmt.Errorf("%w: nhiều hơn %d câu", ErrScriptReplyUnusable, MaxScriptLines)
	}
	out := make([]ScriptLineDraft, 0, len(wire.Lines))
	for i, w := range wire.Lines {
		d := ScriptLineDraft{Text: strings.TrimSpace(w.Text), Facts: []string{}}
		if err := ValidateLineText(d.Text); err != nil {
			return nil, fmt.Errorf("%w: câu thứ %d: %v", ErrScriptReplyUnusable, i+1, err)
		}
		for _, raw := range w.Facts {
			id, ok := NormalizeFactID(raw)
			if !ok {
				return nil, fmt.Errorf("%w: câu thứ %d trích dữ kiện %q không hợp lệ", ErrScriptReplyUnusable, i+1, raw)
			}
			if known != nil && !known(id) {
				return nil, fmt.Errorf("%w: câu thứ %d trích dữ kiện %s không có trong bảng dữ kiện", ErrScriptReplyUnusable, i+1, id)
			}
			d.Facts = uniqueFacts(d.Facts, []string{id})
		}
		out = append(out, d)
	}
	return out, nil
}

// ParseLineSuggestion reads the one replacement sentence out of a model reply {"text":"..."}.
func ParseLineSuggestion(reply string) (string, error) {
	obj, ok := jsonObjectIn(reply)
	if !ok {
		return "", fmt.Errorf("%w: không có khối JSON", ErrScriptReplyUnusable)
	}
	var wire struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal([]byte(obj), &wire); err != nil {
		return "", fmt.Errorf("%w: khối JSON không đọc được (%v)", ErrScriptReplyUnusable, err)
	}
	text := strings.TrimSpace(wire.Text)
	if err := ValidateLineText(text); err != nil {
		return "", fmt.Errorf("%w: %v", ErrScriptReplyUnusable, err)
	}
	return text, nil
}

// Line flag kinds.
const (
	LineFlagLong = "Câu dài"
	LineFlagHard = "Khó nghe"
)

// LineFlag marks a line the Creator should look at, with the reason.
type LineFlag struct {
	Kind string `json:"kind"`
	Msg  string `json:"msg"`
}

var (
	mixedAlnumToken = regexp.MustCompile(`\p{L}+[0-9]+|[0-9]+\p{L}+`)
	longNumberToken = regexp.MustCompile(`[0-9][0-9.,]{4,}`)
	// spokenSymbols are written marks that a voice reads badly or not at all.
	spokenSymbols = "%/&→≈°µ≥≤×±"
)

// FlagLine returns the flag of a line: too long to say in one breath, or
// holding something a voice reads badly. A fixed rule table, no model: the
// first rule that matches names the flag. nil means the line is fine.
func FlagLine(text string) *LineFlag {
	t := strings.TrimSpace(text)
	if words := countWords(t); words > MaxLineWords {
		return &LineFlag{Kind: LineFlagLong, Msg: fmt.Sprintf("%d từ, nên không quá %d từ để đọc trọn một hơi.", words, MaxLineWords)}
	}
	if strings.ContainsAny(t, "()[]") {
		return &LineFlag{Kind: LineFlagHard, Msg: "có ngoặc: giọng đọc không phân biệt được phần trong ngoặc."}
	}
	if m := mixedAlnumToken.FindString(t); m != "" {
		return &LineFlag{Kind: LineFlagHard, Msg: fmt.Sprintf("“%s” trộn chữ và số, giọng dễ đọc sai.", m)}
	}
	if m := longNumberToken.FindString(t); m != "" {
		return &LineFlag{Kind: LineFlagHard, Msg: fmt.Sprintf("số “%s” dài, nên viết gọn hoặc bằng chữ.", m)}
	}
	if m := acronymIn(t); m != "" {
		return &LineFlag{Kind: LineFlagHard, Msg: fmt.Sprintf("“%s” là chữ viết tắt, giọng có thể đánh vần sai.", m)}
	}
	if i := strings.IndexFunc(t, func(r rune) bool { return strings.ContainsRune(spokenSymbols, r) }); i >= 0 {
		r := []rune(t[i:])[0]
		return &LineFlag{Kind: LineFlagHard, Msg: fmt.Sprintf("ký hiệu “%c” nên viết thành chữ.", r)}
	}
	return nil
}

// acronymIn returns the first word of two or more letters that are all capitals, or "".
func acronymIn(text string) string {
	for _, w := range strings.FieldsFunc(text, func(r rune) bool { return !unicode.IsLetter(r) }) {
		if r := []rune(w); len(r) >= 2 && strings.ToUpper(w) == w && strings.ToLower(w) != w {
			return w
		}
	}
	return ""
}

// ScriptCheck is one line of the checks over the whole script.
type ScriptCheck struct {
	OK   bool   `json:"ok"`
	Text string `json:"text"`
}

// CheckOpening reports whether the first line of the script sets up the
// question quickly: it is short enough to be said inside OpeningSeconds, or
// a question is asked by a line that starts inside that time. first is the
// lines of the first beat, secs the estimated seconds of each.
func CheckOpening(first []ScriptLine, secs []float64) ScriptCheck {
	if len(first) == 0 {
		return ScriptCheck{Text: "Chưa có câu mở"}
	}
	start := 0.0
	for i, l := range first {
		if i == 0 && secs[0] <= OpeningSeconds {
			return ScriptCheck{OK: true, Text: fmt.Sprintf("Câu mở (%s) đủ ngắn: %.0f giây", l.ID, secs[0])}
		}
		if start >= OpeningSeconds {
			break
		}
		if strings.Contains(l.Text, "?") {
			return ScriptCheck{OK: true, Text: fmt.Sprintf("Câu mở (%s) đặt câu hỏi trong %.0f giây đầu", l.ID, OpeningSeconds)}
		}
		start += secs[i]
	}
	return ScriptCheck{Text: fmt.Sprintf("Câu mở (%s) dài %.0f giây và chưa đặt câu hỏi trong %.0f giây đầu", first[0].ID, secs[0], OpeningSeconds)}
}

// CheckClosing reports whether the last line of the script lands: it ends the
// sentence and is not flagged as too long.
func CheckClosing(last *ScriptLine) ScriptCheck {
	if last == nil {
		return ScriptCheck{Text: "Chưa có câu kết"}
	}
	t := strings.TrimSpace(last.Text)
	ended := t != "" && strings.ContainsRune(".!?…", []rune(t)[len([]rune(t))-1])
	flag := FlagLine(t)
	if ended && (flag == nil || flag.Kind != LineFlagLong) {
		return ScriptCheck{OK: true, Text: fmt.Sprintf("Câu kết (%s) ngắn gọn và khép câu", last.ID)}
	}
	return ScriptCheck{Text: fmt.Sprintf("Câu kết (%s) quá dài hoặc chưa có dấu kết câu", last.ID)}
}

// JoinFlag says whether the first line of a beat follows the last line of the
// beat before it. A join is flagged when the new beat opens with a word that
// repeats the sentence it follows, or when the previous beat ends mid-thought.
type JoinFlag struct {
	Msg string `json:"msg"`
}

// FlagJoin checks the join between two beats on the lines alone: the same
// opening word twice running, or a previous line that ends without a
// sentence mark. nil means the join reads fine.
func FlagJoin(prevLast, nextFirst string) *JoinFlag {
	prev, next := strings.TrimSpace(prevLast), strings.TrimSpace(nextFirst)
	if prev == "" || next == "" {
		return nil
	}
	if last := []rune(prev)[len([]rune(prev))-1]; !strings.ContainsRune(".!?…", last) {
		return &JoinFlag{Msg: "câu cuối beat trước chưa có dấu kết câu, chỗ nối dễ bị cụt."}
	}
	pw, nw := firstWord(prev), firstWord(next)
	if pw != "" && strings.EqualFold(pw, nw) && len([]rune(pw)) > 2 {
		return &JoinFlag{Msg: fmt.Sprintf("hai câu liền nhau cùng mở bằng “%s”, nghe lặp.", pw)}
	}
	return nil
}

func firstWord(s string) string {
	f := strings.FieldsFunc(s, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
	if len(f) == 0 {
		return ""
	}
	return f[0]
}
