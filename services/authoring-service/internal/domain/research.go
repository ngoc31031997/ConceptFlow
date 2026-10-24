package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// The kinds of review unit the research step stores, and the one fixed key of
// its gate.
const (
	// UnitKindResearch is the gate of the step: approving it approves the fact
	// table; skipping the step is recorded in its content.
	UnitKindResearch = "research"
	// ResearchUnitKey is the only key of the gate unit.
	ResearchUnitKey = "main"
	// UnitKindResearchQuestion is one question the research should answer.
	UnitKindResearchQuestion = "research_question"
	// UnitKindFact is one fact with its source, keyed by its code (F01...).
	UnitKindFact = "fact"
)

// ResearchSchema names the JSON shape a research tool is asked to answer in.
const ResearchSchema = "conceptflow.research.v1"

// Limits on what one pasted answer may hold.
const (
	// MaxPasteBytes is the largest pasted answer read.
	MaxPasteBytes = 1 << 20
	// MaxPasteClaims is the most fact rows read from one pasted answer.
	MaxPasteClaims = 300
	// MaxSourceURLLength is the longest source link accepted.
	MaxSourceURLLength = 2048
)

// FactType says how settled a fact is.
type FactType string

// The kinds of fact a research answer may hold.
const (
	FactTypeFact          FactType = "fact"
	FactTypeHypothesis    FactType = "hypothesis"
	FactTypeUnknown       FactType = "unknown"
	FactTypeMisconception FactType = "misconception"
)

// FactTypes lists every FactType in the order the screen offers them.
var FactTypes = []FactType{FactTypeFact, FactTypeHypothesis, FactTypeUnknown, FactTypeMisconception}

// ValidFactType reports whether t is one of FactTypes.
func ValidFactType(t FactType) bool {
	for _, v := range FactTypes {
		if v == t {
			return true
		}
	}
	return false
}

// LinkState is where the check of a fact's source link stands.
type LinkState string

// The results of checking a source link against its quote.
const (
	// LinkOK: the page was read and holds the quote.
	LinkOK LinkState = "ok"
	// LinkBad: the page was read but does not hold the quote.
	LinkBad LinkState = "bad"
	// LinkUnreachable: the page could not be read (paywall, error, time out).
	LinkUnreachable LinkState = "na"
	// LinkNone: the fact has no source link.
	LinkNone LinkState = "none"
	// LinkRunning: the check is in progress.
	LinkRunning LinkState = "run"
)

// FactLink is the outcome of checking one fact's link.
type FactLink struct {
	State LinkState `json:"state"`
	// Why explains a bad or unreachable link in the Creator's words.
	Why string `json:"why,omitempty"`
	// Nearest is the sentence of the page closest to a quote that was not found.
	Nearest   string `json:"nearest,omitempty"`
	CheckedAt string `json:"checked_at,omitempty"`
}

// Fact is the content of a fact unit: a claim in the video's language with
// where it comes from.
type Fact struct {
	// Question is the code of the research question this fact answers (Q1).
	Question    string   `json:"question"`
	ClaimVI     string   `json:"claim_vi"`
	Type        FactType `json:"type"`
	SourceTitle string   `json:"source_title"`
	Publisher   string   `json:"publisher"`
	Year        string   `json:"year"`
	URL         string   `json:"url"`
	Quote       string   `json:"quote"`
	Confidence  string   `json:"confidence"`
	Note        string   `json:"note"`
	// Must marks a fact the script has to state.
	Must bool `json:"must"`
	// Keep is false for a fact the Creator dropped.
	Keep bool     `json:"keep"`
	Link FactLink `json:"link"`
}

// ResearchGate is the content of the gate unit.
type ResearchGate struct {
	// Skipped is true while the Creator has skipped the whole step.
	Skipped bool `json:"skipped"`
	// NextQuestion is the number the next question code takes; codes are never
	// reused, so a fact keeps pointing at the question it answered.
	NextQuestion int `json:"next_question"`
}

// ResearchQuestion is the content of a question unit. BriefHash identifies the
// approved brief an AI-written question was written from; it is empty for a
// question the Creator wrote or edited.
type ResearchQuestion struct {
	Text      string `json:"text"`
	BriefHash string `json:"brief_hash,omitempty"`
}

// StoredFact is a fact as the project_research table holds it: the fact under
// its code, with the names of the fields the Creator wrote by hand, which a
// pasted answer does not overwrite.
type StoredFact struct {
	ID            string
	Fact          Fact
	CreatorFields []string
}

// HasCreatorField reports whether the Creator wrote or changed the named field.
func (s StoredFact) HasCreatorField(name string) bool {
	for _, f := range s.CreatorFields {
		if f == name {
			return true
		}
	}
	return false
}

// ErrNoResearchJSON is returned when a pasted answer holds no research JSON block.
var ErrNoResearchJSON = errors.New("không tìm thấy khối JSON conceptflow.research.v1 trong đoạn đã dán")

// ErrPasteTooLarge is returned when a pasted answer is over MaxPasteBytes.
var ErrPasteTooLarge = errors.New("đoạn dán quá lớn (tối đa 1 MB)")

// ErrResearchEmpty is returned when a fact table with no kept fact is approved.
var ErrResearchEmpty = errors.New("chưa có dữ kiện nào để duyệt: dán kết quả hoặc thêm dữ kiện, hoặc bỏ qua bước")

// ErrResearchReplyUnusable marks a model reply the research questions cannot be read from.
var ErrResearchReplyUnusable = errors.New("câu trả lời của AI không dùng được làm câu hỏi nghiên cứu")

// MaxGeneratedQuestions is the most questions one AI reply may hold.
const MaxGeneratedQuestions = 12

// ParseResearchQuestionsReply reads the questions out of a model reply: the
// first JSON object in it, with a "questions" list of strings, with or without
// a ```json fence. Blank and repeated questions are dropped; a reply with none
// left, or with more than MaxGeneratedQuestions, is an error so that a half
// usable list is never stored.
func ParseResearchQuestionsReply(reply string) ([]string, error) {
	start := strings.Index(reply, "{")
	end := strings.LastIndex(reply, "}")
	if start < 0 || end <= start {
		return nil, fmt.Errorf("%w: không có khối JSON", ErrResearchReplyUnusable)
	}
	var doc struct {
		Questions []string `json:"questions"`
	}
	if err := json.Unmarshal([]byte(reply[start:end+1]), &doc); err != nil {
		return nil, fmt.Errorf("%w: khối JSON không đọc được (%v)", ErrResearchReplyUnusable, err)
	}
	seen := map[string]bool{}
	var out []string
	for _, q := range doc.Questions {
		q = strings.TrimSpace(q)
		key := strings.ToLower(q)
		if q == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, q)
	}
	switch {
	case len(out) == 0:
		return nil, fmt.Errorf("%w: danh sách câu hỏi trống", ErrResearchReplyUnusable)
	case len(out) > MaxGeneratedQuestions:
		return nil, fmt.Errorf("%w: quá %d câu hỏi", ErrResearchReplyUnusable, MaxGeneratedQuestions)
	}
	return out, nil
}

var factIDPattern = regexp.MustCompile(`^[Ff]0*(\d+)$`)

// NormalizeFactID reads a fact code written as F1, f01 or F001 and returns it
// as F01; false when s is not a fact code.
func NormalizeFactID(s string) (string, bool) {
	m := factIDPattern.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return "", false
	}
	n, err := strconv.Atoi(m[1])
	if err != nil || n < 1 {
		return "", false
	}
	return fmt.Sprintf("F%02d", n), true
}

// NextFactID returns the code after the largest of ids (F01 when there is none).
func NextFactID(ids []string) string {
	highest := 0
	for _, id := range ids {
		if m := factIDPattern.FindStringSubmatch(id); m != nil {
			if n, err := strconv.Atoi(m[1]); err == nil && n > highest {
				highest = n
			}
		}
	}
	return fmt.Sprintf("F%02d", highest+1)
}

// QuestionID returns the code of the n-th question.
func QuestionID(n int) string { return "Q" + strconv.Itoa(n) }

// QuestionNumber reads the number out of a question code (Q3 -> 3).
func QuestionNumber(id string) (int, bool) {
	if len(id) < 2 || id[0] != 'Q' {
		return 0, false
	}
	n, err := strconv.Atoi(id[1:])
	if err != nil || n < 1 {
		return 0, false
	}
	return n, true
}

// ValidSourceURL reports whether raw is a link a source check may follow:
// http or https, a host, no credentials, at most MaxSourceURLLength characters.
func ValidSourceURL(raw string) bool {
	if raw == "" || len(raw) > MaxSourceURLLength {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}

// PasteRejected is one row of a pasted answer that was not read, and why.
type PasteRejected struct {
	Row    int    `json:"row"`
	Reason string `json:"reason"`
}

// PastedClaim is one fact row read from a pasted answer.
type PastedClaim struct {
	ID   string
	Fact Fact
}

// PasteParse is what a pasted answer holds: the rows read, in order, and the
// rows left out.
type PasteParse struct {
	Claims   []PastedClaim
	Rejected []PasteRejected
}

type researchClaimWire struct {
	ID          string          `json:"id"`
	Question    string          `json:"question"`
	ClaimVI     string          `json:"claim_vi"`
	Type        string          `json:"type"`
	SourceTitle string          `json:"source_title"`
	Publisher   string          `json:"publisher"`
	Year        json.RawMessage `json:"year"`
	URL         string          `json:"url"`
	Quote       string          `json:"quote"`
	Confidence  string          `json:"confidence"`
	Note        string          `json:"note"`
}

// ParseResearchPaste reads the facts out of a research tool's answer. It looks
// for ```json blocks holding the schema name, or else for bare {...} objects
// holding it, and reads every block it finds. A row that cannot be read is
// left out with its reason; validQuestion says whether a question code exists.
func ParseResearchPaste(raw string, validQuestion func(id string) bool) (PasteParse, error) {
	if len(raw) > MaxPasteBytes {
		return PasteParse{}, ErrPasteTooLarge
	}
	blocks := researchBlocks(stripMarkdownEscapes(raw))
	if len(blocks) == 0 {
		return PasteParse{}, ErrNoResearchJSON
	}
	var out PasteParse
	row := 0
	indexOf := map[string]int{}
	for bi, block := range blocks {
		var doc struct {
			Claims []json.RawMessage `json:"claims"`
		}
		if err := json.Unmarshal([]byte(block), &doc); err != nil {
			out.Rejected = append(out.Rejected, PasteRejected{Reason: fmt.Sprintf("khối JSON thứ %d không đọc được (%v)", bi+1, err)})
			continue
		}
		for _, item := range doc.Claims {
			row++
			if row > MaxPasteClaims {
				out.Rejected = append(out.Rejected, PasteRejected{Row: row, Reason: fmt.Sprintf("quá %d dòng, dòng này không đọc", MaxPasteClaims)})
				continue
			}
			claim, reason := readClaim(item, validQuestion)
			if reason != "" {
				out.Rejected = append(out.Rejected, PasteRejected{Row: row, Reason: reason})
				continue
			}
			if prev, dup := indexOf[claim.ID]; dup {
				out.Rejected = append(out.Rejected, PasteRejected{Row: prev + 1, Reason: fmt.Sprintf("trùng mã %s, lấy dòng sau", claim.ID)})
				out.Claims[prev] = claim
				continue
			}
			indexOf[claim.ID] = len(out.Claims)
			out.Claims = append(out.Claims, claim)
		}
	}
	return out, nil
}

// stripMarkdownEscapes removes the backslash a chat page puts before a
// punctuation mark when it renders markdown (`https\://`, `research\.v1`), which
// copying the rendered text carries along. A backslash that starts a JSON
// escape (`\"`, `\\`, `\/`, `\b`, `\f`, `\n`, `\r`, `\t`, `\u`) is kept with the
// character it escapes, so valid JSON is returned unchanged.
func stripMarkdownEscapes(s string) string {
	if !strings.Contains(s, `\`) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c != '\\' || i+1 == len(s) {
			b.WriteByte(c)
			continue
		}
		next := s[i+1]
		if strings.IndexByte(`"\/bfnrtu`, next) >= 0 {
			b.WriteByte(c)
			b.WriteByte(next)
			i++
			continue
		}
		if next > ' ' && next < 0x7f && !isASCIIAlnum(next) {
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}

func isASCIIAlnum(c byte) bool {
	return c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

// readClaim reads one row; the second result is why it was left out ("" when read).
func readClaim(item json.RawMessage, validQuestion func(string) bool) (PastedClaim, string) {
	var w researchClaimWire
	if err := json.Unmarshal(item, &w); err != nil {
		return PastedClaim{}, "dòng không đọc được (" + err.Error() + ")"
	}
	id, ok := NormalizeFactID(w.ID)
	if !ok {
		return PastedClaim{}, fmt.Sprintf("mã dữ kiện %q sai dạng (cần F01, F02...)", w.ID)
	}
	if strings.TrimSpace(w.ClaimVI) == "" {
		return PastedClaim{}, fmt.Sprintf("dữ kiện %s thiếu claim_vi", id)
	}
	t := FactType(strings.TrimSpace(w.Type))
	if t == "" {
		return PastedClaim{}, fmt.Sprintf("dữ kiện %s thiếu trường type", id)
	}
	if !ValidFactType(t) {
		return PastedClaim{}, fmt.Sprintf("dữ kiện %s có type %q ngoài fact, hypothesis, unknown, misconception", id, w.Type)
	}
	question := strings.TrimSpace(w.Question)
	if validQuestion != nil && !validQuestion(question) {
		return PastedClaim{}, fmt.Sprintf("dữ kiện %s thuộc câu hỏi %q không có", id, question)
	}
	link := strings.TrimSpace(w.URL)
	if link != "" && !ValidSourceURL(link) {
		return PastedClaim{}, fmt.Sprintf("dữ kiện %s có link không dùng được (cần http hoặc https, không kèm tài khoản)", id)
	}
	fact := Fact{
		Question:    question,
		ClaimVI:     strings.TrimSpace(w.ClaimVI),
		Type:        t,
		SourceTitle: strings.TrimSpace(w.SourceTitle),
		Publisher:   strings.TrimSpace(w.Publisher),
		Year:        yearText(w.Year),
		URL:         link,
		Quote:       strings.TrimSpace(w.Quote),
		Confidence:  strings.TrimSpace(w.Confidence),
		Note:        strings.TrimSpace(w.Note),
		Keep:        true,
	}
	return PastedClaim{ID: id, Fact: fact}, ""
}

// yearText reads a year written as a number or a string.
func yearText(raw json.RawMessage) string {
	s := strings.TrimSpace(string(raw))
	if s == "" || s == "null" {
		return ""
	}
	var str string
	if err := json.Unmarshal(raw, &str); err == nil {
		return strings.TrimSpace(str)
	}
	return s
}

var fencedBlock = regexp.MustCompile("(?s)```[a-zA-Z]*\\s*(.*?)```")

// researchBlocks returns the JSON texts of an answer that hold ResearchSchema.
func researchBlocks(raw string) []string {
	var blocks []string
	for _, m := range fencedBlock.FindAllStringSubmatch(raw, -1) {
		if strings.Contains(m[1], ResearchSchema) {
			blocks = append(blocks, strings.TrimSpace(m[1]))
		}
	}
	if len(blocks) > 0 {
		return blocks
	}
	for _, obj := range balancedObjects(raw) {
		if strings.Contains(obj, ResearchSchema) {
			blocks = append(blocks, obj)
		}
	}
	return blocks
}

// balancedObjects returns the top-level {...} spans of s, skipping braces
// inside JSON strings.
func balancedObjects(s string) []string {
	var out []string
	depth, start := 0, -1
	inString, escaped := false, false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inString {
			switch {
			case escaped:
				escaped = false
			case c == '\\':
				escaped = true
			case c == '"':
				inString = false
			}
			continue
		}
		switch c {
		case '"':
			if depth > 0 {
				inString = true
			}
		case '{':
			if depth == 0 {
				start = i
			}
			depth++
		case '}':
			if depth > 0 {
				depth--
				if depth == 0 {
					out = append(out, s[start:i+1])
				}
			}
		}
	}
	return out
}

// QuestionCoverage says how well one question is sourced.
type QuestionCoverage struct {
	Question string   `json:"question"`
	Sources  int      `json:"sources"`
	BadFacts []string `json:"bad_facts"`
	// Weak is true when the question needs another round of research: fewer
	// than two independent sources, or a fact whose quote was not found.
	Weak bool `json:"weak"`
}

// Coverage counts, for each question, the independent sources (distinct site
// names) of its kept facts whose link checked out, and lists its kept facts
// whose quote was not found. factIDs fixes the order of BadFacts.
func Coverage(questions []string, factIDs []string, facts map[string]Fact) []QuestionCoverage {
	ids := append([]string(nil), factIDs...)
	sort.Strings(ids)
	out := make([]QuestionCoverage, 0, len(questions))
	for _, q := range questions {
		c := QuestionCoverage{Question: q, BadFacts: []string{}}
		hosts := map[string]bool{}
		for _, id := range ids {
			f := facts[id]
			if !f.Keep || f.Question != q {
				continue
			}
			switch f.Link.State {
			case LinkOK:
				if h := sourceHost(f.URL); h != "" {
					hosts[h] = true
				}
			case LinkBad:
				c.BadFacts = append(c.BadFacts, id)
			}
		}
		c.Sources = len(hosts)
		c.Weak = c.Sources < 2 || len(c.BadFacts) > 0
		out = append(out, c)
	}
	return out
}

// sourceHost is the site a link points to, lower case without a leading www.
func sourceHost(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
}
