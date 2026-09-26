package domain

import (
	"errors"
	"strings"
	"testing"
)

const editScript = `from conceptflow import *

class DemoScene(ConceptFlowScene):
    def construct(self):
        self.narrate("câu một")
        self.narrate('câu hai')
`

func TestReplaceNarrationLiteral_RewritesExactlyOneLine(t *testing.T) {
	out, err := ReplaceNarrationLiteral(editScript, "câu một", "câu một đã sửa")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `self.narrate("câu một đã sửa")`) {
		t.Fatalf("chưa thay: %s", out)
	}
	// Phần còn lại của script phải nguyên vẹn từng byte.
	if !strings.Contains(out, `self.narrate('câu hai')`) {
		t.Fatal("đã đụng vào dòng khác")
	}
}

func TestReplaceNarrationLiteral_HandlesSingleQuotes(t *testing.T) {
	out, err := ReplaceNarrationLiteral(editScript, "câu hai", "hai mới")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `self.narrate("hai mới")`) {
		t.Fatalf("chưa thay: %s", out)
	}
}

func TestReplaceNarrationLiteral_RefusesWhenAmbiguous(t *testing.T) {
	// Sau CR-018 một literal có thể sinh ra nhiều dòng dàn ý (vòng lặp), nên
	// đoán xem Creator muốn sửa lần xuất hiện nào là cách làm hỏng script.
	script := editScript + `        self.narrate("câu một")` + "\n"
	if _, err := ReplaceNarrationLiteral(script, "câu một", "x"); !errors.Is(err, ErrNarrationNotEditable) {
		t.Fatalf("muốn từ chối, có %v", err)
	}
}

func TestReplaceNarrationLiteral_RefusesTextNotInSource(t *testing.T) {
	// Lời thoại dựng bằng f-string không xuất hiện nguyên văn trong mã nguồn.
	if _, err := ReplaceNarrationLiteral(editScript, "Ví dụ số 1", "x"); !errors.Is(err, ErrNarrationNotEditable) {
		t.Fatalf("muốn từ chối, có %v", err)
	}
}

func TestReplaceNarrationLiteral_RefusesUnsafeReplacement(t *testing.T) {
	// Dấu nháy trong chuỗi thay thế sẽ đóng literal sớm và làm script không
	// còn parse được.
	for _, bad := range []string{`có "nháy"`, "xuống\ndòng", `dấu \ chéo`} {
		if _, err := ReplaceNarrationLiteral(editScript, "câu một", bad); !errors.Is(err, ErrNarrationNotEditable) {
			t.Errorf("muốn từ chối %q, có %v", bad, err)
		}
	}
}
