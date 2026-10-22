package domain

import (
	"errors"
	"strings"
	"testing"
)

const fullIdeaJSON = `{"question":"Vì sao ong chết trong quả sung?","why":"Ai cũng nghe ăn sung là ăn xác ong","known":"Sung là chùm hoa kín","unknown":"Quả ngoài chợ có còn xác ong không"}`

func TestParseIdeaReplyReadsAFencedJSONBlock(t *testing.T) {
	idea, err := ParseIdeaReply("Đây là ý tưởng:\n```json\n" + fullIdeaJSON + "\n```\n")
	if err != nil {
		t.Fatalf("a fenced reply with every field should parse: %v", err)
	}
	if idea.Question != "Vì sao ong chết trong quả sung?" || idea.Unknown != "Quả ngoài chợ có còn xác ong không" {
		t.Errorf("fields read wrong: %+v", idea)
	}
}

func TestParseIdeaReplyRefusesAReplyMissingAField(t *testing.T) {
	_, err := ParseIdeaReply(`{"question":"Q","why":"","known":"K"}`)
	if !errors.Is(err, ErrIdeaReplyUnusable) {
		t.Fatalf("a reply with an empty and a missing field must be unusable, got %v", err)
	}
	if !strings.Contains(err.Error(), "Vì sao câu hỏi này thú vị") || !strings.Contains(err.Error(), "Điều chưa biết") {
		t.Errorf("the error must name the missing fields as the screen labels them, got %q", err)
	}
}

func TestParseIdeaReplyRefusesAReplyWithoutJSON(t *testing.T) {
	if _, err := ParseIdeaReply("Xin lỗi, tôi không làm được."); !errors.Is(err, ErrIdeaReplyUnusable) {
		t.Fatalf("a reply with no JSON must be unusable, got %v", err)
	}
}

func TestKeepIdeaCreatorFieldsKeepsOnlyTheCreatorsFields(t *testing.T) {
	written := Idea{Question: "AI Q", Why: "AI W"}
	current := Idea{Question: "Creator Q", Why: "old W"}
	got := KeepIdeaCreatorFields(written, current, []string{"question"})
	if got.Question != "Creator Q" || got.Why != "AI W" {
		t.Errorf("only the Creator's field survives the redo, got %+v", got)
	}
}

func TestIdeaPromptUsesOnlyTheVariablesTheUseCaseFills(t *testing.T) {
	known := map[string]bool{"topic": true, "idea_language": true, "current_idea": true, "kept_fields": true, "redo_note": true}
	text := aiTemplate(t, RoleIdea)
	for _, m := range placeholderRe.FindAllStringSubmatch(text, -1) {
		if !known[m[1]] {
			t.Errorf("idea uses {{%s}}, which the idea use case does not fill", m[1])
		}
	}
	for _, field := range IdeaFields {
		if !strings.Contains(text, `"`+field+`"`) {
			t.Errorf("the idea prompt does not ask for the %q key", field)
		}
	}
}
