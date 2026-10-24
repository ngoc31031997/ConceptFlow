package domain

import (
	"fmt"
	"regexp"
	"strings"
)

var trailingRefs = regexp.MustCompile(`\s*\[([^\[\]]*)\]\s*$`)

// FormatRefLine writes a text with the facts it cites as a trailing
// "[F01, F02]"; a text citing none is written as it is.
func FormatRefLine(text string, refs []string) string {
	if len(refs) == 0 {
		return text
	}
	return text + " [" + strings.Join(refs, ", ") + "]"
}

// ParseRefLine reads FormatRefLine's form back: the text and the fact codes in
// the trailing brackets. A bracket holding anything but fact codes stays in the
// text. A code known() does not know is an error naming it.
func ParseRefLine(line string, known func(id string) bool) (string, []string, error) {
	line = strings.TrimSpace(line)
	m := trailingRefs.FindStringSubmatchIndex(line)
	if m == nil {
		return line, []string{}, nil
	}
	var refs []string
	for _, part := range strings.Split(line[m[2]:m[3]], ",") {
		id, ok := NormalizeFactID(strings.TrimSpace(part))
		if !ok {
			return line, []string{}, nil
		}
		if known != nil && !known(id) {
			return "", nil, fmt.Errorf("dữ kiện %s không có trong bảng dữ kiện", id)
		}
		refs = append(refs, id)
	}
	return strings.TrimSpace(line[:m[0]]), refs, nil
}

// uniqueFacts returns ids in first-seen order, each once.
func uniqueFacts(lists ...[]string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, ids := range lists {
		for _, id := range ids {
			if !seen[id] {
				seen[id] = true
				out = append(out, id)
			}
		}
	}
	return out
}

// jsonObjectIn returns the text from the first "{" to the last "}" of a
// reply, with or without a ```json fence around it.
func jsonObjectIn(reply string) (string, bool) {
	start := strings.Index(reply, "{")
	end := strings.LastIndex(reply, "}")
	if start < 0 || end <= start {
		return "", false
	}
	return reply[start : end+1], true
}
