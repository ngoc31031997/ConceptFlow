package domain

import (
	"strings"
	"unicode"
)

// ReplaceWord replaces every whole-word occurrence of find in text with
// replacement and reports how many it replaced. A match is whole when the
// characters next to it are not letters or digits, so "TRNG" does not match
// inside "TRNGS". With matchCase false the comparison ignores letter case.
func ReplaceWord(text, find, replacement string, matchCase bool) (string, int) {
	runes, target := []rune(text), []rune(find)
	if len(target) == 0 {
		return text, 0
	}
	var out strings.Builder
	count := 0
	for i := 0; i < len(runes); {
		if matchesAt(runes, target, i, matchCase) {
			out.WriteString(replacement)
			i += len(target)
			count++
			continue
		}
		out.WriteRune(runes[i])
		i++
	}
	if count == 0 {
		return text, 0
	}
	return out.String(), count
}

// matchesAt reports whether target stands as a whole word at runes[at:].
func matchesAt(runes, target []rune, at int, matchCase bool) bool {
	end := at + len(target)
	if end > len(runes) {
		return false
	}
	for j, t := range target {
		r := runes[at+j]
		if r != t && (matchCase || unicode.ToLower(r) != unicode.ToLower(t)) {
			return false
		}
	}
	if at > 0 && isWordRune(runes[at-1]) {
		return false
	}
	return end == len(runes) || !isWordRune(runes[end])
}

func isWordRune(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }
