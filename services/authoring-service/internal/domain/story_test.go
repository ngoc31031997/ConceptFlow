package domain

import (
	"strings"
	"testing"
)

func knownFacts(ids ...string) func(string) bool {
	set := map[string]bool{}
	for _, id := range ids {
		set[id] = true
	}
	return func(id string) bool { return set[id] }
}

func TestParseRefLineReadsTrailingFactCodes(t *testing.T) {
	text, refs, err := ParseRefLine("Sung là chùm hoa kín. [f1, F02]", knownFacts("F01", "F02"))
	if err != nil {
		t.Fatalf("ParseRefLine: %v", err)
	}
	if text != "Sung là chùm hoa kín." || strings.Join(refs, ",") != "F01,F02" {
		t.Errorf("got %q %v, want the text without the brackets and codes F01,F02", text, refs)
	}
}

func TestParseRefLineKeepsABracketThatIsNotFactCodes(t *testing.T) {
	text, refs, err := ParseRefLine("Một câu [chú thích]", knownFacts("F01"))
	if err != nil || text != "Một câu [chú thích]" || len(refs) != 0 {
		t.Errorf("got %q %v %v, want the line unchanged without refs", text, refs, err)
	}
}

func TestParseRefLineRejectsUnknownFact(t *testing.T) {
	if _, _, err := ParseRefLine("Một câu [F09]", knownFacts("F01")); err == nil || !strings.Contains(err.Error(), "F09") {
		t.Errorf("err = %v, want one naming F09", err)
	}
}
