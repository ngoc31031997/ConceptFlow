package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// The review of the script is stored as one unit of its own kind, outside the
// generic unit API: it is never approved, only worked through.
const (
	UnitKindScriptReview = "script_review"
	ScriptReviewUnitKey  = "main"
)

// Groups a review finding belongs to, by key. The AI is told the keys; the
// screen shows ReviewGroupLabel.
const (
	ReviewGroupAccuracy   = "accuracy"
	ReviewGroupResearch   = "research"
	ReviewGroupTiming     = "timing"
	ReviewGroupLogic      = "logic"
	ReviewGroupClarity    = "clarity"
	ReviewGroupProduction = "production"
	// ReviewGroupFinal holds the one finding that says the AI has not reviewed
	// the script as it stands now.
	ReviewGroupFinal = "final"
)

// ReviewGroups are the groups the AI may use, in the order the screen lists them.
var ReviewGroups = []string{ReviewGroupAccuracy, ReviewGroupResearch, ReviewGroupTiming, ReviewGroupLogic, ReviewGroupClarity, ReviewGroupProduction}

var reviewGroupLabels = map[string]string{
	ReviewGroupAccuracy:   "Độ chính xác",
	ReviewGroupResearch:   "Nhất quán với nghiên cứu",
	ReviewGroupTiming:     "Thời lượng",
	ReviewGroupLogic:      "Logic và nhịp",
	ReviewGroupClarity:    "Dễ hiểu sai",
	ReviewGroupProduction: "Khả năng sản xuất",
	ReviewGroupFinal:      "Nhất quán cuối",
}

// ReviewGroupLabel names a group for the Creator.
func ReviewGroupLabel(key string) string { return reviewGroupLabels[key] }

// Where a finding stands for the Creator.
const (
	// FindingOpen still needs the Creator: it holds the lock back.
	FindingOpen = "open"
	// FindingPass is a machine check that passed.
	FindingPass = "ok"
	// FindingInfo only informs; it never holds the lock back.
	FindingInfo = "info"
	// FindingFixed was dealt with: its suggestion applied, its note added to the
	// handoff, or a later review no longer found it.
	FindingFixed = "fixed"
	// FindingAccepted was accepted by the Creator with a reason.
	FindingAccepted = "accepted"
)

// Sources of a finding.
const (
	FindingByMachine = "machine"
	FindingByAI      = "ai"
)

// Limits of one review.
const (
	MaxReviewFindings   = 40
	MaxReviewNotes      = 10
	MaxReviewTextRunes  = 600
	MaxCreatorNoteRunes = 1000
	MaxAcceptWhyRunes   = 500
)

// ErrReviewReplyUnusable marks a model reply the review cannot be read from.
var ErrReviewReplyUnusable = errors.New("câu trả lời của AI không dùng được làm kết quả rà soát")

// HandoffAddition is a line a finding proposes for the handoff note of a beat.
type HandoffAddition struct {
	Beat    string `json:"beat"`
	Section string `json:"section"`
	Text    string `json:"text"`
}

// ReviewFinding is one line of the review.
type ReviewFinding struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Group  string `json:"group"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
	// Line is the id of the line the finding is about, if any.
	Line string `json:"line,omitempty"`
	// Fix is a replacement for Line the Creator can apply.
	Fix string `json:"fix,omitempty"`
	// Handoff is a line the Creator can add to a beat's handoff note.
	Handoff *HandoffAddition `json:"handoff,omitempty"`
	// Research says the finding is dealt with at the research step.
	Research bool `json:"research,omitempty"`
	// Pass is set on a machine check that passed; Info on a finding that only informs.
	Pass bool `json:"pass,omitempty"`
	Info bool `json:"info,omitempty"`
}

// Fingerprint identifies what a finding says; a decision taken on one wording
// does not carry over to another.
func (f ReviewFinding) Fingerprint() string {
	sum := sha256.Sum256([]byte(f.Group + "\x00" + f.Title + "\x00" + f.Detail + "\x00" + f.Line))
	return hex.EncodeToString(sum[:8])
}

// FindingDecision is what the Creator, or a later review, decided on a finding.
type FindingDecision struct {
	State       string `json:"state"`
	Why         string `json:"why,omitempty"`
	Fingerprint string `json:"fingerprint"`
}

// ScriptReview is the content of the script_review unit.
type ScriptReview struct {
	// Findings are the AI's findings; machine findings are worked out on every read.
	Findings []ReviewFinding `json:"findings"`
	// NextSeq is the number the next AI finding takes (r1, r2...); never reused.
	NextSeq   int                        `json:"next_seq"`
	Decisions map[string]FindingDecision `json:"decisions"`
	AINotes   []string                   `json:"ai_notes"`
	MyNotes   []string                   `json:"my_notes"`
	// ReviewedSHA is the hash of the script the AI last reviewed; "" before the first review.
	ReviewedSHA string `json:"reviewed_sha"`
	Rounds      int    `json:"rounds"`
}

// Normalized fills the empty parts of a review read from storage.
func (r ScriptReview) Normalized() ScriptReview {
	if r.Findings == nil {
		r.Findings = []ReviewFinding{}
	}
	if r.Decisions == nil {
		r.Decisions = map[string]FindingDecision{}
	}
	if r.AINotes == nil {
		r.AINotes = []string{}
	}
	if r.MyNotes == nil {
		r.MyNotes = []string{}
	}
	if r.NextSeq < 1 {
		r.NextSeq = len(r.Findings) + 1
	}
	return r
}

// StateOf is where a finding stands, given the decisions taken.
func (r ScriptReview) StateOf(f ReviewFinding) string {
	switch {
	case f.Pass:
		return FindingPass
	case f.Info:
		return FindingInfo
	}
	if d, ok := r.Decisions[f.ID]; ok && d.Fingerprint == f.Fingerprint() {
		return d.State
	}
	return FindingOpen
}

// Decide records a decision on a finding as it is worded now.
func (r *ScriptReview) Decide(f ReviewFinding, state, why string) {
	if r.Decisions == nil {
		r.Decisions = map[string]FindingDecision{}
	}
	r.Decisions[f.ID] = FindingDecision{State: state, Why: why, Fingerprint: f.Fingerprint()}
}

// ScriptSHA is the hash of the script's wording, in beat order. A review is
// current while the script's hash is the one it reviewed.
func ScriptSHA(beats [][]ScriptLine) string {
	h := sha256.New()
	for _, lines := range beats {
		for _, l := range lines {
			h.Write([]byte(l.ID + "\x00" + strings.TrimSpace(l.Text) + "\x00" + strings.TrimSpace(l.Spoken) + "\x00" + strings.Join(l.Facts, ",") + "\n"))
		}
		h.Write([]byte("|"))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// ReviewBeat is one beat as the machine checks read it.
type ReviewBeat struct {
	Key    string
	Number int
	Lines  []ScriptLine
}

// ReviewFact is one kept fact as the machine checks read it.
type ReviewFact struct {
	ID   string
	Must bool
	Link LinkState
}

// ReviewInput is everything the machine checks read.
type ReviewInput struct {
	Beats           []ReviewBeat
	Facts           []ReviewFact
	ResearchSkipped bool
	TotalSeconds    float64
	LoSeconds       int
	HiSeconds       int
	TargetLabel     string
	// ScriptSHA is the hash of the script now; ReviewedSHA the one the AI last reviewed.
	ScriptSHA   string
	ReviewedSHA string
}

// MachineFindings works out the findings that need no model: the facts the
// script must state, lines resting on a fact whose quote was not found on its
// page, the length against the target, figures and time spans no fact backs,
// the lines and joins the script step flags, and whether the AI has reviewed
// the script as it stands. A skipped research step gives one warning instead
// of the link findings.
func MachineFindings(in ReviewInput) []ReviewFinding {
	out := []ReviewFinding{}
	usedBy := map[string][]string{}
	for _, b := range in.Beats {
		for _, l := range b.Lines {
			for _, id := range l.Facts {
				usedBy[id] = append(usedBy[id], l.ID)
			}
		}
	}

	out = append(out, reviewFinal(in))

	if in.ResearchSkipped {
		out = append(out, ReviewFinding{
			ID: "m-skipped", Source: FindingByMachine, Group: ReviewGroupAccuracy,
			Title:    "Mọi dữ kiện trong lời thoại đều chưa có nguồn",
			Detail:   "Bước Nghiên cứu đang bỏ qua. Chấp nhận để chốt, hoặc quay lại làm nghiên cứu.",
			Research: true,
		})
	} else {
		bad := 0
		for _, f := range in.Facts {
			if f.Link != LinkBad || len(usedBy[f.ID]) == 0 {
				continue
			}
			bad++
			out = append(out, ReviewFinding{
				ID: "m-link:" + f.ID, Source: FindingByMachine, Group: ReviewGroupResearch,
				Title:    fmt.Sprintf("Câu %s dùng %s, nhưng đoạn trích không khớp trang nguồn", strings.Join(usedBy[f.ID], ", "), f.ID),
				Detail:   "Sửa nguồn của dữ kiện ở bước Nghiên cứu, hoặc chấp nhận và ghi lý do.",
				Line:     usedBy[f.ID][0],
				Research: true,
			})
		}
		if bad == 0 {
			out = append(out, ReviewFinding{
				ID: "m-link", Source: FindingByMachine, Group: ReviewGroupResearch, Pass: true,
				Title:  "Không câu thoại nào dùng dữ kiện có đoạn trích không khớp",
				Detail: "Mọi dữ kiện dùng trong lời thoại đều khớp nguồn hoặc đã ghi rõ loại.",
			})
		}
	}

	var must, missing []string
	for _, f := range in.Facts {
		if f.Must {
			must = append(must, f.ID)
			if len(usedBy[f.ID]) == 0 {
				missing = append(missing, f.ID)
			}
		}
	}
	mustFinding := ReviewFinding{ID: "m-must", Source: FindingByMachine, Group: ReviewGroupAccuracy}
	if len(missing) > 0 {
		mustFinding.Title = "Dữ kiện bắt buộc chưa có trong lời thoại: " + strings.Join(missing, ", ")
		mustFinding.Detail = "Thêm vào lời thoại một câu nói dữ kiện đó, hoặc bỏ dấu “bắt buộc nói” ở bước Nghiên cứu."
	} else {
		mustFinding.Pass = true
		mustFinding.Title = "Dữ kiện bắt buộc đều có trong lời thoại"
		mustFinding.Detail = fmt.Sprintf("%d/%d", len(must), len(must))
		if len(must) > 0 {
			mustFinding.Detail += ": " + strings.Join(must, ", ")
		}
		mustFinding.Detail += "."
	}
	out = append(out, mustFinding)

	for _, b := range in.Beats {
		for _, l := range b.Lines {
			if len(l.Facts) > 0 {
				continue
			}
			if m := figureIn(l.Text); m != "" {
				out = append(out, ReviewFinding{
					ID: "m-figure:" + l.ID, Source: FindingByMachine, Group: ReviewGroupAccuracy, Line: l.ID,
					Title:  fmt.Sprintf("Câu %s có “%s” nhưng chưa gắn dữ kiện nào", l.ID, m),
					Detail: "Gắn một dữ kiện có nguồn, hoặc nói định tính.",
				})
			}
		}
	}

	total := int(math.Round(in.TotalSeconds))
	length := ReviewFinding{ID: "m-length", Source: FindingByMachine, Group: ReviewGroupTiming}
	if total >= in.LoSeconds && total <= in.HiSeconds {
		length.Pass = true
		length.Title = fmt.Sprintf("Tổng %s, trong khung mục tiêu %s", clock(total), in.TargetLabel)
	} else {
		length.Title = fmt.Sprintf("Tổng %s, ngoài khung mục tiêu %s", clock(total), in.TargetLabel)
		length.Detail = "Rút gọn hoặc thêm lời thoại ở bước Viết kịch bản, hoặc chấp nhận và ghi lý do."
	}
	out = append(out, length)

	prevLast := ""
	for i, b := range in.Beats {
		for _, l := range b.Lines {
			flag := FlagLine(l.SpokenText())
			if flag == nil {
				continue
			}
			group := ReviewGroupClarity
			if flag.Kind == LineFlagLong {
				group = ReviewGroupTiming
			}
			out = append(out, ReviewFinding{
				ID: "m-flag:" + l.ID, Source: FindingByMachine, Group: group, Line: l.ID,
				Title:  fmt.Sprintf("Câu %s · %s", l.ID, strings.ToLower(flag.Kind)),
				Detail: flag.Msg,
			})
		}
		if len(b.Lines) == 0 {
			prevLast = ""
			continue
		}
		if i > 0 && prevLast != "" {
			if j := FlagJoin(prevLast, b.Lines[0].SpokenText()); j != nil {
				out = append(out, ReviewFinding{
					ID: "m-join:" + b.Key, Source: FindingByMachine, Group: ReviewGroupLogic, Line: b.Lines[0].ID,
					Title:  fmt.Sprintf("Chỗ nối từ beat B%d sang beat B%d", in.Beats[i-1].Number, b.Number),
					Detail: j.Msg,
				})
			}
		}
		prevLast = b.Lines[len(b.Lines)-1].SpokenText()
	}
	return out
}

// reviewFinal says whether the AI has reviewed the script as it stands.
func reviewFinal(in ReviewInput) ReviewFinding {
	f := ReviewFinding{ID: "m-review", Source: FindingByMachine, Group: ReviewGroupFinal}
	switch {
	case in.ReviewedSHA == "":
		f.Title = "AI chưa rà soát kịch bản này"
		f.Detail = "Bấm “AI rà soát”, hoặc chấp nhận và ghi lý do nếu không dùng AI."
	case in.ReviewedSHA != in.ScriptSHA:
		f.Title = "Lời thoại đã đổi sau lần AI rà soát gần nhất"
		f.Detail = "Bấm “Rà soát lại” để AI so bản hiện tại với các mục đã phát hiện."
	default:
		f.Pass = true
		f.Title = "AI đã rà soát đúng bản lời thoại hiện tại"
	}
	return f
}

func clock(sec int) string {
	if sec < 0 {
		sec = 0
	}
	return fmt.Sprintf("%d:%02d", sec/60, sec%60)
}

var (
	digitRun = regexp.MustCompile(`\p{Nd}+(?:[.,]\p{Nd}+)?`)
	// numberWords are the words that state an amount; "một" is left out
	// because it is mostly an article.
	numberWords = toWordSet("hai ba bốn năm sáu bảy tám chín mười mươi chục trăm nghìn ngàn vạn triệu tỷ tỉ vài mấy dăm nửa rưỡi")
	// unitWords are the words that turn an amount into a figure a fact should back.
	unitWords = toWordSet("giây phút giờ ngày tuần tháng năm thế kỷ thập niên mét milimét xentimét centimét kilômét km cân kilôgam gam tấn độ phần lần loài con quả cây người")
)

func toWordSet(words string) map[string]bool {
	out := map[string]bool{}
	for _, w := range strings.Fields(words) {
		out[w] = true
	}
	return out
}

// figureIn returns the first figure or time span in text — a number written in
// digits, or an amount word followed by a unit ("vài tuần", "hai milimét") —
// or "" when there is none.
func figureIn(text string) string {
	if m := digitRun.FindString(text); m != "" {
		return m
	}
	words := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool { return !unicode.IsLetter(r) })
	for i := 0; i+1 < len(words); i++ {
		if numberWords[words[i]] && unitWords[words[i+1]] && words[i] != words[i+1] {
			return words[i] + " " + words[i+1]
		}
	}
	return ""
}

// ReviewFindingDraft is a finding as the AI wrote it, before it has an id.
type ReviewFindingDraft struct {
	ReviewFinding
	// SameAs is the id of an earlier finding this one repeats, on a second review.
	SameAs string
}

// ReviewReply is what one AI review returned.
type ReviewReply struct {
	Findings []ReviewFindingDraft
	Notes    []string
}

// ParseReviewReply reads a model reply {"findings":[...],"notes":[...]}. A
// finding needs a known group and a title; a line it names must be a line of
// the script, a beat its handoff names must be a beat, and an earlier finding
// it repeats that is not one of earlier makes it a new finding. Any other failure stores nothing.
func ParseReviewReply(reply string, lineKnown, beatKnown, earlierKnown func(string) bool) (ReviewReply, error) {
	var wire struct {
		Findings []struct {
			Group   string `json:"group"`
			Title   string `json:"title"`
			Detail  string `json:"detail"`
			Line    string `json:"line"`
			Fix     string `json:"fix"`
			Handoff *struct {
				Beat    string `json:"beat"`
				Section string `json:"section"`
				Text    string `json:"text"`
			} `json:"handoff"`
			SameAs string `json:"same_as"`
		} `json:"findings"`
		Notes []string `json:"notes"`
	}
	if err := json.Unmarshal([]byte(reply), &wire); err != nil {
		return ReviewReply{}, fmt.Errorf("%w: khối JSON không đọc được (%v)", ErrReviewReplyUnusable, err)
	}
	if len(wire.Findings) > MaxReviewFindings {
		return ReviewReply{}, fmt.Errorf("%w: nhiều hơn %d mục", ErrReviewReplyUnusable, MaxReviewFindings)
	}
	out := ReviewReply{Findings: []ReviewFindingDraft{}, Notes: []string{}}
	for i, w := range wire.Findings {
		n := i + 1
		d := ReviewFindingDraft{ReviewFinding: ReviewFinding{
			Source: FindingByAI, Group: strings.TrimSpace(w.Group), Title: strings.TrimSpace(w.Title),
			Detail: strings.TrimSpace(w.Detail), Line: strings.TrimSpace(w.Line), Fix: strings.TrimSpace(w.Fix),
		}, SameAs: strings.TrimSpace(w.SameAs)}
		if _, ok := reviewGroupLabels[d.Group]; !ok || d.Group == ReviewGroupFinal {
			return ReviewReply{}, fmt.Errorf("%w: mục thứ %d có nhóm %q không có", ErrReviewReplyUnusable, n, w.Group)
		}
		if d.Title == "" {
			return ReviewReply{}, fmt.Errorf("%w: mục thứ %d thiếu tiêu đề", ErrReviewReplyUnusable, n)
		}
		for _, t := range []string{d.Title, d.Detail, d.Fix} {
			if len([]rune(t)) > MaxReviewTextRunes {
				return ReviewReply{}, fmt.Errorf("%w: mục thứ %d dài quá %d ký tự", ErrReviewReplyUnusable, n, MaxReviewTextRunes)
			}
		}
		if d.Line != "" && !lineKnown(d.Line) {
			return ReviewReply{}, fmt.Errorf("%w: mục thứ %d nói tới câu %s không có trong kịch bản", ErrReviewReplyUnusable, n, d.Line)
		}
		if d.Fix != "" {
			if d.Line == "" {
				return ReviewReply{}, fmt.Errorf("%w: mục thứ %d có câu gợi ý nhưng không nêu câu nào", ErrReviewReplyUnusable, n)
			}
			if err := ValidateLineText(d.Fix); err != nil {
				return ReviewReply{}, fmt.Errorf("%w: câu gợi ý của mục thứ %d: %v", ErrReviewReplyUnusable, n, err)
			}
		}
		if w.Handoff != nil {
			h := HandoffAddition{Beat: strings.TrimSpace(w.Handoff.Beat), Section: strings.TrimSpace(w.Handoff.Section), Text: strings.TrimSpace(w.Handoff.Text)}
			if !beatKnown(h.Beat) {
				return ReviewReply{}, fmt.Errorf("%w: mục thứ %d nói tới beat %q không có", ErrReviewReplyUnusable, n, h.Beat)
			}
			if !IsHandoffSection(h.Section) {
				return ReviewReply{}, fmt.Errorf("%w: mục thứ %d có mục bàn giao %q không có", ErrReviewReplyUnusable, n, h.Section)
			}
			if err := validateHandoffItem(h.Text); err != nil {
				return ReviewReply{}, fmt.Errorf("%w: mục thứ %d: %v", ErrReviewReplyUnusable, n, err)
			}
			d.Handoff = &h
		}
		// A model that names no earlier finding (it may put a line id here, or fill
		// the field on a first review) has found a new one: that is not worth
		// throwing the whole review away.
		if d.SameAs != "" && !earlierKnown(d.SameAs) {
			d.SameAs = ""
		}
		out.Findings = append(out.Findings, d)
	}
	for _, note := range wire.Notes {
		if t := strings.TrimSpace(note); t != "" {
			if len([]rune(t)) > MaxReviewTextRunes {
				return ReviewReply{}, fmt.Errorf("%w: một ghi chú rà soát dài quá %d ký tự", ErrReviewReplyUnusable, MaxReviewTextRunes)
			}
			out.Notes = append(out.Notes, t)
		}
	}
	if len(out.Notes) > MaxReviewNotes {
		out.Notes = out.Notes[:MaxReviewNotes]
	}
	return out, nil
}

// ApplyReview folds one AI review of the script with hash sha into the stored
// review. A finding that repeats an earlier one keeps its id: an acceptance
// stays, and one the Creator marked fixed opens again because it is still
// there. An earlier finding the review no longer reports counts as fixed; one
// that was already fixed before this review is dropped. New findings take the
// next ids. The AI's notes are replaced; the Creator's stay.
func ApplyReview(prev ScriptReview, reply ReviewReply, sha string) ScriptReview {
	prev = prev.Normalized()
	next := ScriptReview{
		Findings: []ReviewFinding{}, NextSeq: prev.NextSeq, Decisions: map[string]FindingDecision{},
		AINotes: append([]string{}, reply.Notes...), MyNotes: prev.MyNotes, ReviewedSHA: sha, Rounds: prev.Rounds + 1,
	}
	for id, d := range prev.Decisions {
		if !strings.HasPrefix(id, "r") {
			next.Decisions[id] = d // machine findings keep their decisions
		}
	}
	repeated := map[string]bool{}
	for _, d := range reply.Findings {
		f := d.ReviewFinding
		if d.SameAs != "" && !repeated[d.SameAs] {
			repeated[d.SameAs] = true
			f.ID = d.SameAs
			if old, ok := prev.Decisions[d.SameAs]; ok && old.State == FindingAccepted {
				next.Decide(f, FindingAccepted, old.Why)
			}
		} else {
			f.ID = "r" + strconv.Itoa(next.NextSeq)
			next.NextSeq++
		}
		next.Findings = append(next.Findings, f)
	}
	for _, old := range prev.Findings {
		if repeated[old.ID] {
			continue
		}
		state := prev.StateOf(old)
		if state == FindingFixed {
			continue
		}
		if state == FindingAccepted {
			next.Decisions[old.ID] = prev.Decisions[old.ID]
		} else {
			next.Decide(old, FindingFixed, "AI rà soát lại không còn thấy")
		}
		next.Findings = append(next.Findings, old)
	}
	return next
}
