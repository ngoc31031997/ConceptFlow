package domain

import (
	"strings"
	"testing"
)

func lines(ids ...string) []ScriptLine {
	out := []ScriptLine{}
	for _, id := range ids {
		out = append(out, ScriptLine{ID: id, Text: "Câu " + id + ".", Facts: []string{}})
	}
	return out
}

func findingByID(fs []ReviewFinding, id string) (ReviewFinding, bool) {
	for _, f := range fs {
		if f.ID == id {
			return f, true
		}
	}
	return ReviewFinding{}, false
}

func baseInput() ReviewInput {
	return ReviewInput{
		Beats:        []ReviewBeat{{Key: "b1", Number: 1, Lines: lines("b1.1", "b1.2")}},
		TotalSeconds: 60, LoSeconds: 30, HiSeconds: 90, TargetLabel: "1–1,5 phút",
		ScriptSHA: "now", ReviewedSHA: "now",
	}
}

func TestMachineFindingsReportMissingMustFactAndPassWhenPresent(t *testing.T) {
	in := baseInput()
	in.Facts = []ReviewFact{{ID: "F01", Must: true, Link: LinkOK}, {ID: "F02", Must: false, Link: LinkOK}}
	f, _ := findingByID(MachineFindings(in), "m-must")
	if f.Pass || !strings.Contains(f.Title, "F01") {
		t.Fatalf("must finding = %+v, want an open finding naming F01", f)
	}
	in.Beats[0].Lines[0].Facts = []string{"F01"}
	f, _ = findingByID(MachineFindings(in), "m-must")
	if !f.Pass {
		t.Fatalf("must finding = %+v, want pass once F01 is in a line", f)
	}
}

func TestMachineFindingsFlagLinesOnFactsWhoseQuoteDoesNotMatch(t *testing.T) {
	in := baseInput()
	in.Beats[0].Lines[1].Facts = []string{"F03"}
	in.Facts = []ReviewFact{{ID: "F03", Link: LinkBad}}
	f, ok := findingByID(MachineFindings(in), "m-link:F03")
	if !ok || f.Line != "b1.2" || !f.Research {
		t.Fatalf("link finding = %+v ok=%v, want one on b1.2 that opens research", f, ok)
	}
	// A bad link no line uses is not a finding.
	in.Beats[0].Lines[1].Facts = nil
	if _, ok := findingByID(MachineFindings(in), "m-link:F03"); ok {
		t.Error("a fact no line uses must not give a link finding")
	}
}

func TestSkippedResearchGivesExactlyOneNoSourceWarning(t *testing.T) {
	in := baseInput()
	in.ResearchSkipped = true
	in.Beats[0].Lines[0].Facts = []string{"F01"}
	in.Facts = []ReviewFact{{ID: "F01", Link: LinkNone}}
	n := 0
	for _, f := range MachineFindings(in) {
		if f.Group == ReviewGroupAccuracy && f.Research {
			n++
		}
		if strings.HasPrefix(f.ID, "m-link") {
			t.Errorf("skipped research must not give link findings, got %s", f.ID)
		}
	}
	if n != 1 {
		t.Errorf("no-source warnings = %d, want 1", n)
	}
}

func TestMachineFindingsFlagFiguresWithoutFactsAndTimeSpans(t *testing.T) {
	in := baseInput()
	in.Beats[0].Lines[0].Text = "Vài tuần sau, trứng nở."
	in.Beats[0].Lines[1].Text = "Con ong dài 2 mm."
	f1, ok1 := findingByID(MachineFindings(in), "m-figure:b1.1")
	f2, ok2 := findingByID(MachineFindings(in), "m-figure:b1.2")
	if !ok1 || !ok2 || !strings.Contains(f1.Title, "vài tuần") || !strings.Contains(f2.Title, "2") {
		t.Fatalf("figure findings = %+v %+v", f1, f2)
	}
	in.Beats[0].Lines[1].Facts = []string{"F01"}
	if _, ok := findingByID(MachineFindings(in), "m-figure:b1.2"); ok {
		t.Error("a line that cites a fact must not be flagged for its figure")
	}
}

func TestMachineLengthAndReviewFreshness(t *testing.T) {
	in := baseInput()
	if f, _ := findingByID(MachineFindings(in), "m-length"); !f.Pass {
		t.Errorf("60s inside 30-90 must pass, got %+v", f)
	}
	in.TotalSeconds = 200
	if f, _ := findingByID(MachineFindings(in), "m-length"); f.Pass {
		t.Errorf("200s outside 30-90 must not pass, got %+v", f)
	}
	if f, _ := findingByID(MachineFindings(in), "m-review"); !f.Pass {
		t.Errorf("same hash must pass, got %+v", f)
	}
	in.ReviewedSHA = ""
	if f, _ := findingByID(MachineFindings(in), "m-review"); f.Pass || !strings.Contains(f.Title, "chưa rà soát") {
		t.Errorf("never reviewed must be open, got %+v", f)
	}
	in.ReviewedSHA = "old"
	if f, _ := findingByID(MachineFindings(in), "m-review"); f.Pass || !strings.Contains(f.Title, "đã đổi") {
		t.Errorf("changed script must be open, got %+v", f)
	}
}

func always(string) bool { return true }
func never(string) bool  { return false }

func TestParseReviewReplyRejectsWhatItCannotTrust(t *testing.T) {
	bad := map[string]string{
		"no json":       "xin chào",
		"unknown group": "{\"findings\":[{\"group\":\"zzz\",\"title\":\"x\"}]}",
		"no title":      "{\"findings\":[{\"group\":\"logic\",\"title\":\" \"}]}",
		"fix no line":   "{\"findings\":[{\"group\":\"logic\",\"title\":\"x\",\"fix\":\"Câu mới.\"}]}",
		"final group":   "{\"findings\":[{\"group\":\"final\",\"title\":\"x\"}]}",
	}
	for name, reply := range bad {
		if _, err := ParseReviewReply(reply, always, always, always); err == nil {
			t.Errorf("%s: want error, got none", name)
		}
	}
	line := "{\"findings\":[{\"group\":\"logic\",\"title\":\"x\",\"line\":\"b9.9\"}]}"
	if _, err := ParseReviewReply(line, never, always, always); err == nil {
		t.Error("an unknown line must be rejected")
	}
	ho := "{\"findings\":[{\"group\":\"clarity\",\"title\":\"x\",\"handoff\":{\"beat\":\"b1\",\"section\":\"bogus\",\"text\":\"t\"}}]}"
	if _, err := ParseReviewReply(ho, always, always, always); err == nil {
		t.Error("an unknown handoff section must be rejected")
	}
	// A same_as that names no earlier finding makes a new finding, not an unusable review.
	same := "{\"findings\":[{\"group\":\"logic\",\"title\":\"x\",\"same_as\":\"b24.1\"}]}"
	r, err := ParseReviewReply(same, always, always, never)
	if err != nil || len(r.Findings) != 1 || r.Findings[0].SameAs != "" {
		t.Errorf("unknown same_as: reply %+v err %v, want one new finding", r, err)
	}
}

func parse(t *testing.T, reply string, earlier ...string) ReviewReply {
	t.Helper()
	set := map[string]bool{}
	for _, e := range earlier {
		set[e] = true
	}
	r, err := ParseReviewReply(reply, always, always, func(id string) bool { return set[id] })
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestSecondReviewKeepsIdsFixesVanishedAndReopensRepeated(t *testing.T) {
	first := ApplyReview(ScriptReview{}, parse(t,
		"{\"findings\":[{\"group\":\"logic\",\"title\":\"A\",\"line\":\"b1.1\"},{\"group\":\"clarity\",\"title\":\"B\"},{\"group\":\"timing\",\"title\":\"C\"}],\"notes\":[\"ổn\"]}"), "sha1")
	if len(first.Findings) != 3 || first.Findings[0].ID != "r1" || first.NextSeq != 4 || first.Rounds != 1 {
		t.Fatalf("first review = %+v", first)
	}
	// The Creator marks A fixed (wrongly) and accepts B.
	first.Decide(first.Findings[0], FindingFixed, "")
	first.Decide(first.Findings[1], FindingAccepted, "đã biết")

	second := ApplyReview(first, parse(t,
		"{\"findings\":[{\"group\":\"logic\",\"title\":\"A\",\"line\":\"b1.1\",\"same_as\":\"r1\"},{\"group\":\"production\",\"title\":\"D\"}]}", "r1", "r2", "r3"), "sha2")
	byID := map[string]ReviewFinding{}
	for _, f := range second.Findings {
		byID[f.ID] = f
	}
	if second.StateOf(byID["r1"]) != FindingOpen {
		t.Errorf("r1 was repeated, so a Creator's 'fixed' must reopen it, got %s", second.StateOf(byID["r1"]))
	}
	if second.StateOf(byID["r2"]) != FindingAccepted || second.Decisions["r2"].Why != "đã biết" {
		t.Errorf("r2 not reported again but accepted: it must stay accepted, got %+v", second.Decisions["r2"])
	}
	if second.StateOf(byID["r3"]) != FindingFixed {
		t.Errorf("r3 is gone from the new review, want fixed, got %s", second.StateOf(byID["r3"]))
	}
	if d, ok := byID["r4"]; !ok || second.StateOf(d) != FindingOpen || d.Title != "D" {
		t.Errorf("new finding must take r4 and be open, got %+v", byID["r4"])
	}
	if second.ReviewedSHA != "sha2" || second.Rounds != 2 || len(second.AINotes) != 0 {
		t.Errorf("second = %+v", second)
	}
	// A third review drops what was already fixed.
	third := ApplyReview(second, parse(t, "{\"findings\":[]}"), "sha3")
	for _, f := range third.Findings {
		if f.ID == "r3" {
			t.Error("a finding already fixed before the review must be dropped")
		}
	}
}

func TestDecisionDoesNotCarryToReworkedFinding(t *testing.T) {
	r := ScriptReview{}.Normalized()
	f := ReviewFinding{ID: "m-flag:b1.1", Group: ReviewGroupClarity, Title: "t", Detail: "a", Line: "b1.1"}
	r.Decide(f, FindingAccepted, "ok")
	if r.StateOf(f) != FindingAccepted {
		t.Fatal("decision lost")
	}
	f.Detail = "khác"
	if r.StateOf(f) != FindingOpen {
		t.Error("a finding worded differently must be open again")
	}
}
