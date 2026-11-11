package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

// Groups of a review finding.
var FindingGroups = []string{"accuracy", "consistency", "clarity", "feasibility"}

// FindingGroupLabels names each group for the Creator.
var FindingGroupLabels = map[string]string{
	"accuracy":    "Độ chính xác khoa học",
	"consistency": "Nhất quán",
	"clarity":     "Rõ ràng",
	"feasibility": "Khả thi",
}

// States of a finding. Open blocks the lock; the others do not.
const (
	FindingOpen     = "open"
	FindingFixed    = "fixed"
	FindingApplied  = "applied"
	FindingAccepted = "accepted"
)

// Finding is one issue the AI found in the concept.
type Finding struct {
	ID      string `json:"id"`
	Group   string `json:"group"`
	UnitKey string `json:"unit_key"`
	Field   string `json:"field"`
	Text    string `json:"text"`
	Detail  string `json:"detail"`
	Fix     string `json:"fix"`
	// Replaces is the exact existing line of a list field that Fix rewrites; empty
	// when Fix is a new line or replaces a whole text field.
	Replaces string `json:"replaces"`
	State    string `json:"state"`
	Why      string `json:"why"`
}

// CheckDecision is the Creator's acceptance of a failing machine check, valid
// while the check's detail has the same fingerprint.
type CheckDecision struct {
	Why         string `json:"why"`
	Fingerprint string `json:"fingerprint"`
}

// ReviewState is the content of the visual_review unit.
type ReviewState struct {
	Findings []Finding                `json:"findings"`
	NextSeq  int                      `json:"next_seq"`
	Notes    []string                 `json:"notes"`
	Rounds   int                      `json:"rounds"`
	Checks   map[string]CheckDecision `json:"checks"`
}

// ParseReviewState reads the stored review; empty content is a review not run.
func ParseReviewState(raw []byte) (ReviewState, error) {
	r := ReviewState{NextSeq: 1}
	if len(raw) > 0 && string(raw) != "{}" {
		if err := json.Unmarshal(raw, &r); err != nil {
			return ReviewState{}, fmt.Errorf("đọc rà soát đã lưu: %w", err)
		}
	}
	if r.NextSeq < 1 {
		r.NextSeq = 1
	}
	if r.Checks == nil {
		r.Checks = map[string]CheckDecision{}
	}
	if r.Findings == nil {
		r.Findings = []Finding{}
	}
	if r.Notes == nil {
		r.Notes = NoteList{}
	}
	return r, nil
}

// Fingerprint is the hash a check decision is tied to.
func Fingerprint(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:8])
}

// CheckAccepted reports whether the Creator accepted c as it reads now.
func (r ReviewState) CheckAccepted(c MachineCheck) (CheckDecision, bool) {
	d, ok := r.Checks[c.ID]
	return d, ok && d.Fingerprint == Fingerprint(c.Detail)
}

// Find returns the index of the finding with id, or -1.
func (r ReviewState) Find(id string) int {
	for i, f := range r.Findings {
		if f.ID == id {
			return i
		}
	}
	return -1
}

// OpenCount counts open findings.
func (r ReviewState) OpenCount() int {
	n := 0
	for _, f := range r.Findings {
		if f.State == FindingOpen {
			n++
		}
	}
	return n
}

// ReviewReplyItem is one finding as the AI writes it.
type ReviewReplyItem struct {
	Group    string `json:"group"`
	UnitKey  string `json:"unit_key"`
	Field    string `json:"field"`
	Text     string `json:"text"`
	Detail   string `json:"detail"`
	Fix      string `json:"fix"`
	Replaces string `json:"replaces"`
	SameAs   string `json:"same_as"`
}

// ReviewReply is the AI's whole answer to a review.
type ReviewReply struct {
	Findings []ReviewReplyItem `json:"findings"`
	Notes    NoteList          `json:"notes"`
}

// NoteList is the AI's remarks on a review. The prompt asks for a list, but a
// model sometimes writes one remark as a plain string; both are read.
type NoteList []string

// UnmarshalJSON reads a list of strings, a single string or null.
func (n *NoteList) UnmarshalJSON(b []byte) error {
	var list []string
	if err := json.Unmarshal(b, &list); err == nil {
		*n = list
		return nil
	}
	var one string
	if err := json.Unmarshal(b, &one); err != nil {
		return err
	}
	if strings.TrimSpace(one) == "" {
		*n = nil
		return nil
	}
	*n = []string{one}
	return nil
}

// ParseReviewReply reads the AI's review. A group, part or field that does not
// exist, an empty finding, or a same_as naming no earlier finding makes the
// whole reply unusable.
func ParseReviewReply(raw []byte, prev ReviewState) (ReviewReply, error) {
	return parseReviewReply(raw, prev, func(f ReviewReplyItem) string {
		if _, ok := UnitSpecOf(f.UnitKey); !ok {
			return "nói về phần lạ «" + f.UnitKey + "»"
		}
		if f.Field != "" && !contains(PartFields(f.UnitKey), f.Field) {
			return "nói về trường lạ «" + f.Field + "»"
		}
		return ""
	})
}

// parseReviewReply reads a review reply; place says what is wrong with where a
// finding points (the part or unit and its field), or "" when it is fine.
func parseReviewReply(raw []byte, prev ReviewState, place func(ReviewReplyItem) string) (ReviewReply, error) {
	var r ReviewReply
	if err := json.Unmarshal(raw, &r); err != nil {
		return ReviewReply{}, fmt.Errorf("%w: %v", ErrUnreadableReply, err)
	}
	seen := map[string]bool{}
	for i, f := range r.Findings {
		bad := func(why string) error {
			return fmt.Errorf("%w: vấn đề %d %s", ErrUnreadableReply, i+1, why)
		}
		if !contains(FindingGroups, f.Group) {
			return ReviewReply{}, bad("có nhóm lạ «" + f.Group + "»")
		}
		if why := place(f); why != "" {
			return ReviewReply{}, bad(why)
		}
		if strings.TrimSpace(f.Text) == "" {
			return ReviewReply{}, bad("không có nội dung")
		}
		if f.SameAs != "" {
			if prev.Find(f.SameAs) < 0 {
				return ReviewReply{}, bad("nhắc lại vấn đề không có «" + f.SameAs + "»")
			}
			if seen[f.SameAs] {
				return ReviewReply{}, bad("nhắc lại «" + f.SameAs + "» hai lần")
			}
			seen[f.SameAs] = true
		}
	}
	if r.Notes == nil {
		r.Notes = NoteList{}
	}
	return r, nil
}

// MergeReview folds a new review round into prev: a finding the AI reports again
// keeps its id and opens again unless the Creator accepted it; a new one takes
// the next id; one the AI no longer reports is fixed.
func MergeReview(prev ReviewState, reply ReviewReply) ReviewState {
	out := prev
	out.Findings = nil
	again := map[string]ReviewReplyItem{}
	for _, f := range reply.Findings {
		if f.SameAs != "" {
			again[f.SameAs] = f
		}
	}
	for _, old := range prev.Findings {
		f, ok := again[old.ID]
		switch {
		case ok:
			upd := Finding{ID: old.ID, Group: f.Group, UnitKey: f.UnitKey, Field: f.Field, Text: f.Text, Detail: f.Detail, Fix: f.Fix, Replaces: f.Replaces, State: FindingOpen}
			if old.State == FindingAccepted {
				upd.State, upd.Why = FindingAccepted, old.Why
			}
			out.Findings = append(out.Findings, upd)
		case old.State == FindingAccepted:
			out.Findings = append(out.Findings, old)
		default:
			old.State = FindingFixed
			out.Findings = append(out.Findings, old)
		}
	}
	for _, f := range reply.Findings {
		if f.SameAs != "" {
			continue
		}
		out.Findings = append(out.Findings, Finding{
			ID: fmt.Sprintf("f%d", out.NextSeq), Group: f.Group, UnitKey: f.UnitKey, Field: f.Field,
			Text: f.Text, Detail: f.Detail, Fix: f.Fix, Replaces: f.Replaces, State: FindingOpen,
		})
		out.NextSeq++
	}
	if out.Findings == nil {
		out.Findings = []Finding{}
	}
	out.Notes = []string(reply.Notes)
	out.Rounds++
	return out
}

// CanApplyFix reports whether ApplyFix would write f's fix into content as it is
// now, so the Creator is offered a direct apply only when it works.
func CanApplyFix(content []byte, f Finding) bool {
	if f.Field == "" || strings.TrimSpace(f.Fix) == "" {
		return false
	}
	_, err := ApplyFix(content, f.Field, f.Fix, f.Replaces)
	return err == nil
}

// ApplyFix writes fix into field of a part's JSON content: a text field takes
// it; a list of texts gets it as a new line, or, when replaces names an existing
// line, in place of that line. Any other field is not something one line can fix.
func ApplyFix(content []byte, field, fix, replaces string) ([]byte, error) {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(content, &m); err != nil {
		return nil, err
	}
	cur, ok := m[field]
	if !ok || strings.TrimSpace(fix) == "" {
		return nil, ErrFindingNotApplicable
	}
	var s string
	var list []string
	switch {
	case json.Unmarshal(cur, &s) == nil:
		m[field], _ = json.Marshal(fix)
	case json.Unmarshal(cur, &list) == nil:
		if strings.TrimSpace(replaces) == "" {
			m[field], _ = json.Marshal(append(list, fix))
			break
		}
		at := -1
		for i, line := range list {
			if strings.TrimSpace(line) == strings.TrimSpace(replaces) {
				at = i
				break
			}
		}
		if at < 0 {
			return nil, ErrFindingNotApplicable
		}
		list[at] = fix
		m[field], _ = json.Marshal(list)
	default:
		return nil, ErrFindingNotApplicable
	}
	return json.Marshal(m)
}
