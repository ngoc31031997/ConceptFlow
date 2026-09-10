package domain

import (
	"fmt"
	"strings"
)

// ErrNarrationNotEditable means the outline line cannot be traced back to a
// single place in the source.
var ErrNarrationNotEditable = fmt.Errorf("narration line cannot be edited here")

// ReplaceNarrationLiteral rewrites one `self.narrate("...")` literal in the
// script (CR-024 FR70.2).
//
// ## Why this can refuse
//
// The outline comes from *running* the script, while this edit has to change
// the *source*. Those two are not always one-to-one, and CR-018 is the reason:
// narration is now allowed inside loops and helpers, so a single literal can
// produce three outline lines, and an f-string produces text that appears
// nowhere in the source at all.
//
// Rather than guess which occurrence the Creator meant — and silently rewrite
// the wrong line — this refuses unless the old text appears exactly once. The
// GUI then shows that line as read-only with the reason. Being unable to edit
// a line in place is a small inconvenience; corrupting someone's script is not.
func ReplaceNarrationLiteral(script, oldText, newText string) (string, error) {
	oldText, newText = strings.TrimSpace(oldText), strings.TrimSpace(newText)
	if oldText == "" || newText == "" {
		return "", ErrNarrationNotEditable
	}
	if strings.ContainsAny(newText, "\"'\n\\") {
		// The replacement is spliced into a Python string literal, so a quote
		// or a backslash in it would end the literal early and produce a script
		// that no longer parses.
		return "", fmt.Errorf(`%w: lời thoại không được chứa dấu nháy, xuống dòng hay dấu \`, ErrNarrationNotEditable)
	}

	candidates := []string{
		fmt.Sprintf("self.narrate(%q)", oldText),
		fmt.Sprintf("self.narrate('%s')", oldText),
	}

	matched := ""
	total := 0
	for _, candidate := range candidates {
		if n := strings.Count(script, candidate); n > 0 {
			total += n
			matched = candidate
		}
	}
	if total != 1 {
		return "", fmt.Errorf(
			"%w: câu này xuất hiện %d lần trong script (lời thoại sinh trong vòng lặp "+
				"hoặc bằng f-string thì phải sửa trực tiếp trong script)",
			ErrNarrationNotEditable, total)
	}

	replacement := fmt.Sprintf("self.narrate(%q)", newText)
	return strings.Replace(script, matched, replacement, 1), nil
}
