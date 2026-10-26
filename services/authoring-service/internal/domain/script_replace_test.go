package domain

import "testing"

func TestReplaceWordMatchesWholeWordsOnly(t *testing.T) {
	cases := []struct {
		name, text, find, repl string
		matchCase              bool
		want                   string
		count                  int
	}{
		{"replaces each occurrence", "TRNG chậm, còn TRNG nhanh.", "TRNG", "TI A EN GI", true, "TI A EN GI chậm, còn TI A EN GI nhanh.", 2},
		{"not inside a longer word", "TRNGS và xTRNG và TRNG1", "TRNG", "X", true, "TRNGS và xTRNG và TRNG1", 0},
		{"next to punctuation", "(TRNG), TRNG.", "TRNG", "X", true, "(X), X.", 2},
		{"case matters by default", "trng và TRNG", "TRNG", "X", true, "trng và X", 1},
		{"case ignored when asked", "trng và TRNG", "TRNG", "X", false, "X và X", 2},
		{"vietnamese letters count as letters", "bóng bóng", "bóng", "quả", true, "quả quả", 2},
		{"accented neighbour is a letter", "xbóng", "bóng", "quả", true, "xbóng", 0},
		{"word at the ends", "TRNG", "TRNG", "X", true, "X", 1},
	}
	for _, c := range cases {
		got, n := ReplaceWord(c.text, c.find, c.repl, c.matchCase)
		if got != c.want || n != c.count {
			t.Errorf("%s: ReplaceWord(%q) = %q, %d; want %q, %d", c.name, c.text, got, n, c.want, c.count)
		}
	}
}

func TestReplaceWordWithEmptyFindChangesNothing(t *testing.T) {
	if got, n := ReplaceWord("abc", "", "x", true); got != "abc" || n != 0 {
		t.Errorf("got %q, %d, want the text unchanged", got, n)
	}
}
