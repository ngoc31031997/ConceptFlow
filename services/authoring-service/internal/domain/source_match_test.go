package domain

import (
	"strings"
	"testing"
)

const fixturePage = `<html><head><title>Fig wasp</title><style>p{color:red}</style><script>var x = "The female wasp loses her wings";</script></head>
<body><!-- note --><p>The fig wasp is small. Females are about 1.5 to 2 mm long [1].</p>
<p>The female wasp loses her wings and most of her antennae as she squeezes through the ostiole.</p>
<p>Wingless males mate with the females &amp; then chew an exit tunnel &ndash; through the fig wall.</p>
<p>The relationship is thought to have existed for 60 to 80 million years.</p>
<noscript>enable javascript to see more</noscript></body></html>`

func TestStripHTMLDropsScriptsStylesAndTags(t *testing.T) {
	text := StripHTML(fixturePage)
	for _, gone := range []string{"color:red", "var x", "note", "enable javascript", "<p>"} {
		if strings.Contains(text, gone) {
			t.Errorf("StripHTML kept %q in %q", gone, text)
		}
	}
	if !strings.Contains(text, "Females are about 1.5 to 2 mm long") || !strings.Contains(text, "& then chew") {
		t.Errorf("StripHTML lost the visible text or did not decode entities: %q", text)
	}
}

func TestMatchQuoteFindsQuoteDespiteTypography(t *testing.T) {
	page := StripHTML(fixturePage)
	quotes := []string{
		"Females are about 1.5 to 2 mm long.",
		"FEMALES ARE ABOUT 1.5 TO 2 MM LONG",
		"Wingless males mate with the females & then chew an exit tunnel – through the fig wall.",
		"Wingless males mate with the females and then chew",
		"The female wasp loses her wings … squeezes through the ostiole.",
	}
	for i, q := range quotes {
		m := MatchQuote(page, q)
		// The "and then chew" variant differs in wording, so it must not match.
		wantFound := i != 3
		if m.Found != wantFound {
			t.Errorf("quote %d %q: Found=%v, want %v", i, q, m.Found, wantFound)
		}
	}
}

func TestMatchQuotePiecesMustFollowEachOther(t *testing.T) {
	page := StripHTML(fixturePage)
	inOrder := "The female wasp loses her wings ... she squeezes through the ostiole"
	reversed := "she squeezes through the ostiole ... the female wasp loses her wings"
	if !MatchQuote(page, inOrder).Found {
		t.Error("pieces in page order must match")
	}
	if MatchQuote(page, reversed).Found {
		t.Error("pieces out of page order must not match")
	}
}

func TestMatchQuoteOffersNearestSentence(t *testing.T) {
	page := StripHTML(fixturePage)
	m := MatchQuote(page, "The relationship is believed to have existed for 60 to 80 million years.")
	if m.Found || m.TooShort {
		t.Fatalf("match = %+v, want not found", m)
	}
	if !strings.Contains(m.Nearest, "thought to have existed for 60 to 80 million years") {
		t.Errorf("Nearest = %q, want the relationship sentence as the page wrote it", m.Nearest)
	}
	far := MatchQuote(page, "Quantum chromodynamics describes the strong interaction between quarks.")
	if far.Found || far.Nearest != "" {
		t.Errorf("unrelated quote = %+v, want not found and no nearest sentence", far)
	}
}

func TestMatchQuoteTooShort(t *testing.T) {
	page := StripHTML(fixturePage)
	for _, q := range []string{"", "fig wasp", "The fig ... wasp is small"} {
		if m := MatchQuote(page, q); !m.TooShort || m.Found {
			t.Errorf("quote %q = %+v, want TooShort", q, m)
		}
	}
}

func TestNormalizeTextFoldsTypography(t *testing.T) {
	got := NormalizeText("  “Hello” world — it’s  fine[3] ​ ")
	if got != `"hello" world - it's fine` {
		t.Errorf("NormalizeText = %q", got)
	}
}

func TestNearestSentenceIsCut(t *testing.T) {
	long := strings.Repeat("alpha beta gamma delta ", 30)
	m := MatchQuote(long+"end.", "alpha beta gamma delta alpha beta gamma delta epsilon zeta")
	if m.Found {
		t.Fatal("quote with extra words must not be found")
	}
	if n := len([]rune(m.Nearest)); n > nearestMaxRunes+1 {
		t.Errorf("Nearest has %d runes, want at most %d", n, nearestMaxRunes+1)
	}
}
