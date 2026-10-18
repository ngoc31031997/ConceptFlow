package domain

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func claimJSON(id, question, typ string) string {
	return fmt.Sprintf(`{"id":%q,"question":%q,"claim_vi":"Dữ kiện %s","type":%q,"source_title":"T","publisher":"P","year":2020,"url":"https://example.org/%s","quote":"một đoạn trích đủ dài để kiểm","confidence":"high","note":""}`, id, question, id, typ, id)
}

func answerWith(rows ...string) string {
	return "Đây là kết quả:\n```json\n{\"schema\":\"conceptflow.research.v1\",\"claims\":[" + strings.Join(rows, ",") + "]}\n```\nHết."
}

func anyQuestion(string) bool { return true }

func TestParseResearchPasteReadsTwelveFactsAndRejectsOneBadRow(t *testing.T) {
	var rows []string
	for i := 1; i <= 12; i++ {
		rows = append(rows, claimJSON(fmt.Sprintf("F%02d", i), "Q1", "fact"))
	}
	rows = append(rows, `{"id":"F13","question":"Q1","claim_vi":"thiếu type","url":""}`)
	got, err := ParseResearchPaste(answerWith(rows...), anyQuestion)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Claims) != 12 {
		t.Fatalf("claims = %d, want 12", len(got.Claims))
	}
	if len(got.Rejected) != 1 || got.Rejected[0].Row != 13 || !strings.Contains(got.Rejected[0].Reason, "type") {
		t.Fatalf("rejected = %+v, want row 13 naming the missing type", got.Rejected)
	}
	if !got.Claims[0].Fact.Keep || got.Claims[0].Fact.Year != "2020" {
		t.Fatalf("first claim = %+v, want kept with year 2020", got.Claims[0].Fact)
	}
}

func TestParseResearchPasteFindsBlocks(t *testing.T) {
	bare := `Mở đầu {"schema":"conceptflow.research.v1","claims":[` + claimJSON("F01", "Q1", "fact") + `]} kết`
	got, err := ParseResearchPaste(bare, anyQuestion)
	if err != nil || len(got.Claims) != 1 {
		t.Fatalf("bare object: claims=%d err=%v, want 1 claim", len(got.Claims), err)
	}
	two := answerWith(claimJSON("F01", "Q1", "fact")) + "\n" + answerWith(claimJSON("F02", "Q1", "hypothesis"))
	got, err = ParseResearchPaste(two, anyQuestion)
	if err != nil || len(got.Claims) != 2 {
		t.Fatalf("two blocks: claims=%d err=%v, want 2 claims", len(got.Claims), err)
	}
	if _, err := ParseResearchPaste("chỉ là chữ, có {dấu ngoặc} nhưng không có mẫu", anyQuestion); !errors.Is(err, ErrNoResearchJSON) {
		t.Fatalf("no block err = %v, want ErrNoResearchJSON", err)
	}
	if _, err := ParseResearchPaste(strings.Repeat("x", MaxPasteBytes+1), anyQuestion); !errors.Is(err, ErrPasteTooLarge) {
		t.Fatalf("large paste err = %v, want ErrPasteTooLarge", err)
	}
}

func TestParseResearchPasteRowRules(t *testing.T) {
	cases := []struct {
		name, row, want string
	}{
		{"mã sai dạng", `{"id":"X1","question":"Q1","claim_vi":"a","type":"fact"}`, "sai dạng"},
		{"thiếu claim", `{"id":"F01","question":"Q1","claim_vi":" ","type":"fact"}`, "claim_vi"},
		{"type lạ", `{"id":"F01","question":"Q1","claim_vi":"a","type":"rumor"}`, "ngoài"},
		{"câu hỏi không có", `{"id":"F01","question":"Q9","claim_vi":"a","type":"fact"}`, "Q9"},
		{"link javascript", `{"id":"F01","question":"Q1","claim_vi":"a","type":"fact","url":"javascript:alert(1)"}`, "link"},
		{"link có tài khoản", `{"id":"F01","question":"Q1","claim_vi":"a","type":"fact","url":"https://u:pw@example.org/"}`, "link"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ParseResearchPaste(answerWith(c.row), func(id string) bool { return id == "Q1" })
			if err != nil {
				t.Fatal(err)
			}
			if len(got.Claims) != 0 || len(got.Rejected) != 1 || !strings.Contains(got.Rejected[0].Reason, c.want) {
				t.Fatalf("claims=%d rejected=%+v, want one rejection mentioning %q", len(got.Claims), got.Rejected, c.want)
			}
		})
	}
}

func TestParseResearchPasteNormalisesIDsYearsAndDuplicates(t *testing.T) {
	rows := []string{
		`{"id":"f1","question":"Q1","claim_vi":"một","type":"fact","year":"2019"}`,
		`{"id":"F001","question":"Q1","claim_vi":"hai","type":"fact"}`,
		`{"id":"F02","question":"Q1","claim_vi":"ba","type":"unknown"}`,
	}
	got, err := ParseResearchPaste(answerWith(rows...), anyQuestion)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Claims) != 2 || got.Claims[0].ID != "F01" || got.Claims[0].Fact.ClaimVI != "hai" {
		t.Fatalf("claims = %+v, want F01 taken from the later duplicate then F02", got.Claims)
	}
	if len(got.Rejected) != 1 || !strings.Contains(got.Rejected[0].Reason, "trùng mã F01") {
		t.Fatalf("rejected = %+v, want the earlier duplicate noted", got.Rejected)
	}
}

func TestParseResearchPasteStopsAtClaimLimit(t *testing.T) {
	var rows []string
	for i := 1; i <= MaxPasteClaims+2; i++ {
		rows = append(rows, fmt.Sprintf(`{"id":"F%d","question":"Q1","claim_vi":"a","type":"unknown"}`, i))
	}
	got, err := ParseResearchPaste(answerWith(rows...), anyQuestion)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Claims) != MaxPasteClaims || len(got.Rejected) != 2 {
		t.Fatalf("claims=%d rejected=%d, want %d claims and 2 rejected", len(got.Claims), len(got.Rejected), MaxPasteClaims)
	}
}

func TestParseResearchQuestionsReply(t *testing.T) {
	got, err := ParseResearchQuestionsReply("```json\n{\"questions\":[\" Một? \",\"hai?\",\"MỘT?\",\"\"]}\n```")
	if err != nil || len(got) != 2 || got[0] != "Một?" || got[1] != "hai?" {
		t.Fatalf("got %v, %v, want the two distinct trimmed questions", got, err)
	}
	var many []string
	for i := 0; i <= MaxGeneratedQuestions; i++ {
		many = append(many, fmt.Sprintf("%q", fmt.Sprintf("câu %d?", i)))
	}
	for name, reply := range map[string]string{
		"không có JSON":   "không có gì",
		"JSON hỏng":       `{"questions":[`,
		"danh sách trống": `{"questions":[]}`,
		"toàn câu trắng":  `{"questions":["  ",""]}`,
		"sai kiểu":        `{"questions":"một câu"}`,
		"quá nhiều":       `{"questions":[` + strings.Join(many, ",") + `]}`,
	} {
		if _, err := ParseResearchQuestionsReply(reply); !errors.Is(err, ErrResearchReplyUnusable) {
			t.Errorf("%s: err = %v, want ErrResearchReplyUnusable", name, err)
		}
	}
}

func TestFactAndQuestionCodes(t *testing.T) {
	for in, want := range map[string]string{"F1": "F01", "f01": "F01", " F012 ": "F12", "F100": "F100"} {
		if got, ok := NormalizeFactID(in); !ok || got != want {
			t.Errorf("NormalizeFactID(%q) = %q,%v, want %q", in, got, ok, want)
		}
	}
	for _, in := range []string{"", "F", "F0", "Q1", "FF1"} {
		if _, ok := NormalizeFactID(in); ok {
			t.Errorf("NormalizeFactID(%q) accepted a bad code", in)
		}
	}
	if got := NextFactID(nil); got != "F01" {
		t.Errorf("NextFactID(nil) = %q, want F01", got)
	}
	if got := NextFactID([]string{"F01", "F09", "F03"}); got != "F10" {
		t.Errorf("NextFactID = %q, want F10", got)
	}
	if n, ok := QuestionNumber("Q12"); !ok || n != 12 || QuestionID(12) != "Q12" {
		t.Errorf("question code round trip failed: %d %v", n, ok)
	}
	if _, ok := QuestionNumber("F1"); ok {
		t.Error("QuestionNumber accepted a fact code")
	}
}

func TestCoverageCountsIndependentSources(t *testing.T) {
	facts := map[string]Fact{
		"F01": {Question: "Q1", Keep: true, URL: "https://en.wikipedia.org/wiki/A", Link: FactLink{State: LinkOK}},
		"F02": {Question: "Q1", Keep: true, URL: "https://en.wikipedia.org/wiki/B", Link: FactLink{State: LinkOK}},
		"F03": {Question: "Q2", Keep: true, URL: "https://www.britannica.com/x", Link: FactLink{State: LinkOK}},
		"F04": {Question: "Q2", Keep: true, URL: "https://britannica.com/y", Link: FactLink{State: LinkOK}},
		"F05": {Question: "Q3", Keep: true, URL: "https://a.org/1", Link: FactLink{State: LinkOK}},
		"F06": {Question: "Q3", Keep: true, URL: "https://b.org/1", Link: FactLink{State: LinkOK}},
		"F07": {Question: "Q3", Keep: true, URL: "https://c.org/1", Link: FactLink{State: LinkBad}},
		"F08": {Question: "Q4", Keep: false, URL: "https://d.org/1", Link: FactLink{State: LinkOK}},
	}
	ids := []string{"F08", "F07", "F06", "F05", "F04", "F03", "F02", "F01"}
	got := Coverage([]string{"Q1", "Q2", "Q3", "Q4"}, ids, facts)
	if got[0].Sources != 1 || !got[0].Weak {
		t.Errorf("Q1 = %+v, want one source (same site) and weak", got[0])
	}
	if got[1].Sources != 1 || !got[1].Weak {
		t.Errorf("Q2 = %+v, want www. and bare host counted once", got[1])
	}
	if got[2].Sources != 2 || len(got[2].BadFacts) != 1 || got[2].BadFacts[0] != "F07" || !got[2].Weak {
		t.Errorf("Q3 = %+v, want two sources but weak because of F07", got[2])
	}
	if got[3].Sources != 0 || !got[3].Weak {
		t.Errorf("Q4 = %+v, want a dropped fact not counted", got[3])
	}
	clean := Coverage([]string{"Q"}, []string{"F01", "F02"}, map[string]Fact{
		"F01": {Question: "Q", Keep: true, URL: "https://a.org/", Link: FactLink{State: LinkOK}},
		"F02": {Question: "Q", Keep: true, URL: "https://b.org/", Link: FactLink{State: LinkOK}},
	})
	if clean[0].Weak {
		t.Errorf("two independent sources must not be weak: %+v", clean[0])
	}
}
