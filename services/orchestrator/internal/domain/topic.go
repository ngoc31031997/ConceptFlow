package domain

import (
	"regexp"
	"strings"
)

// collapseWhitespace turns any run of whitespace into a single space, so
// "AI   robot" and "AI robot" normalize the same way.
var collapseWhitespace = regexp.MustCompile(`\s+`)

// NormalizeTopic is the CR-028 FR85.1 comparison key for the topic-collision
// warning: lowercase, trimmed, internal whitespace collapsed. Deliberately
// exact-match only (no fuzzy/similarity matching) — a normalized-equal check
// has no false positives, which matters more here than catching every near
// duplicate; fuzzy matching is explicitly out of scope (CR-028 "Không thuộc
// phạm vi CR này").
func NormalizeTopic(topic string) string {
	normalized := strings.ToLower(strings.TrimSpace(topic))
	return collapseWhitespace.ReplaceAllString(normalized, " ")
}
