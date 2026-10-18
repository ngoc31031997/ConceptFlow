package domain

import (
	"html"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// MinQuoteRunes is the shortest quote (after normalising) a page can be
// checked against; a shorter one would match almost any page.
const MinQuoteRunes = 20

// nearestMinSimilarity is the least word overlap for a page sentence to be
// offered as the closest to a quote that was not found.
const nearestMinSimilarity = 0.5

// nearestMaxRunes cuts the closest sentence shown to the Creator.
const nearestMaxRunes = 240

var (
	htmlComment    = regexp.MustCompile(`(?s)<!--.*?-->`)
	htmlScript     = regexp.MustCompile(`(?is)<script\b.*?</script\s*>`)
	htmlStyle      = regexp.MustCompile(`(?is)<style\b.*?</style\s*>`)
	htmlNoscript   = regexp.MustCompile(`(?is)<noscript\b.*?</noscript\s*>`)
	htmlTag        = regexp.MustCompile(`(?s)<[^>]*>`)
	citationMarker = regexp.MustCompile(`\s*\[(?:\d+|citation needed)\]`)
	sentenceEnd    = regexp.MustCompile(`[.!?]\s+|\n+`)
)

var quoteReplacer = strings.NewReplacer(
	"‘", "'", "’", "'", "‚", "'", "‛", "'",
	"“", `"`, "”", `"`, "„", `"`, "‟", `"`,
	"‐", "-", "‑", "-", "‒", "-", "–", "-", "—", "-", "―", "-", "−", "-",
	"\u00ad", "", "\u200b", "", "\u200c", "", "\u200d", "", "\ufeff", "",
)

// StripHTML turns a page's HTML into its text: scripts, styles, comments and
// tags are dropped and character entities are decoded.
func StripHTML(src string) string {
	for _, re := range []*regexp.Regexp{htmlComment, htmlScript, htmlStyle, htmlNoscript} {
		src = re.ReplaceAllString(src, " ")
	}
	src = htmlTag.ReplaceAllString(src, " ")
	return html.UnescapeString(src)
}

// NormalizeText puts text in the form a quote and a page are compared in:
// compatibility-folded Unicode, lower case, straight quotes and hyphens, no
// zero-width characters or citation markers, single spaces.
func NormalizeText(s string) string {
	s = norm.NFKC.String(s)
	s = strings.ToLower(s)
	s = quoteReplacer.Replace(s)
	s = citationMarker.ReplaceAllString(s, "")
	return strings.Join(strings.Fields(s), " ")
}

// QuoteMatch is the result of looking for a quote in a page.
type QuoteMatch struct {
	// Found is true when the whole quote is in the page.
	Found bool
	// TooShort is true when the quote is too short to check.
	TooShort bool
	// Nearest is the page sentence closest to a quote that was not found, or ""
	// when no sentence is close.
	Nearest string
}

// MatchQuote looks for quote in pageText after normalising both. A quote with
// "..." (or "…") in it is a list of pieces that must all be on the page, in
// order; pieces shorter than MinQuoteRunes are ignored, and a quote with no
// piece that long cannot be checked.
func MatchQuote(pageText, quote string) QuoteMatch {
	normQuote := NormalizeText(quote)
	var pieces []string
	for _, piece := range strings.Split(normQuote, "...") {
		piece = strings.Trim(strings.TrimSpace(piece), `"' `)
		if len([]rune(piece)) >= MinQuoteRunes {
			pieces = append(pieces, piece)
		}
	}
	if len(pieces) == 0 {
		return QuoteMatch{TooShort: true}
	}
	page := NormalizeText(pageText)
	pos := 0
	found := true
	for _, piece := range pieces {
		i := strings.Index(page[pos:], piece)
		if i < 0 {
			found = false
			break
		}
		pos += i + len(piece)
	}
	if found {
		return QuoteMatch{Found: true}
	}
	return QuoteMatch{Nearest: nearestSentence(pageText, normQuote)}
}

// nearestSentence returns the sentence of the page whose words overlap most
// with the quote, shown as the page wrote it, or "" when none overlaps enough.
func nearestSentence(pageText, normQuote string) string {
	quoteWords := wordSet(normQuote)
	if len(quoteWords) == 0 {
		return ""
	}
	best, bestScore := "", 0.0
	start := 0
	consider := func(sentence string) {
		sentence = strings.Join(strings.Fields(sentence), " ")
		if sentence == "" {
			return
		}
		words := wordSet(NormalizeText(sentence))
		if len(words) == 0 {
			return
		}
		shared := 0
		for w := range words {
			if quoteWords[w] {
				shared++
			}
		}
		score := float64(shared) / float64(len(words)+len(quoteWords)-shared)
		if score > bestScore {
			best, bestScore = sentence, score
		}
	}
	for _, loc := range sentenceEnd.FindAllStringIndex(pageText, -1) {
		consider(pageText[start:loc[1]])
		start = loc[1]
	}
	consider(pageText[start:])
	if bestScore < nearestMinSimilarity {
		return ""
	}
	if runes := []rune(best); len(runes) > nearestMaxRunes {
		best = string(runes[:nearestMaxRunes]) + "…"
	}
	return best
}

// wordSet is the set of words (letters and digits) of already normalised text.
func wordSet(s string) map[string]bool {
	set := map[string]bool{}
	for _, w := range strings.FieldsFunc(s, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }) {
		set[w] = true
	}
	return set
}
