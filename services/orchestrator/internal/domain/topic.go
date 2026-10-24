package domain

import (
	"regexp"
	"strings"
)

// collapseWhitespace turns any run of whitespace into a single space, so
// "AI   robot" and "AI robot" normalize the same way.
var collapseWhitespace = regexp.MustCompile(`\s+`)

// SanitizeIdeaText cleans the text the Creator typed before it is stored and
// handed to the AI. The AI answers in JSON and tends to copy a straight double
// quote or a backslash from its input into a JSON string, which then cannot be
// read. So a straight double quote becomes a typographic one (“ at the start or
// after a space, bracket or dash, ” anywhere else), a backslash becomes a
// slash, and control characters other than a new line and a tab are dropped.
// Everything else, Vietnamese letters and single quotes included, is kept. The
// result is trimmed. authoring-service applies the same rule to the idea text it
// receives.
func SanitizeIdeaText(text string) string {
	var out []rune
	for _, r := range strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n") {
		switch {
		case r == '"':
			if len(out) == 0 || strings.ContainsRune(" \t\n([{“‘—–-", out[len(out)-1]) {
				out = append(out, '“')
			} else {
				out = append(out, '”')
			}
		case r == '\\':
			out = append(out, '/')
		case r == '\n' || r == '\t' || r >= ' ' && r != 0x7f:
			out = append(out, r)
		}
	}
	return strings.TrimSpace(string(out))
}

// NormalizeTopic is the comparison key for the topic-collision
// warning: lowercase, trimmed, internal whitespace collapsed. Deliberately
// exact-match only (no fuzzy/similarity matching) — a normalized-equal check
// has no false positives, which matters more here than catching every near
// duplicate; fuzzy matching is deliberately not done.
func NormalizeTopic(topic string) string {
	normalized := strings.ToLower(strings.TrimSpace(topic))
	return collapseWhitespace.ReplaceAllString(normalized, " ")
}
