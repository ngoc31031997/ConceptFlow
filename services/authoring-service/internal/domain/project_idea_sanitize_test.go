package domain

import "testing"

func TestSanitizeIdeaText(t *testing.T) {
	cases := map[string]struct{ in, want string }{
		"quote pair":        {`Vì sao người ta nói "ăn sung là ăn xác ong"? Có thật không?`, "Vì sao người ta nói “ăn sung là ăn xác ong”? Có thật không?"},
		"opening positions": {`"Sung" và ("ong")`, "“Sung” và (“ong”)"},
		"backslash":         {`C:\tmp`, "C:/tmp"},
		"control and CRLF":  {"a\u0000b\u0007\r\nc\rd\u007f", "ab\nc\nd"},
		"trimmed":           {"  hố đen \n", "hố đen"},
		"kept as typed":     {"Rơi vào hố đen: người ở xa thấy gì? It's fine.", "Rơi vào hố đen: người ở xa thấy gì? It's fine."},
		"empty":             {"", ""},
	}
	for name, c := range cases {
		if got := SanitizeIdeaText(c.in); got != c.want {
			t.Errorf("%s: SanitizeIdeaText(%q) = %q, want %q", name, c.in, got, c.want)
		}
	}
}

func TestSanitizeIdeaText_IsStable(t *testing.T) {
	once := SanitizeIdeaText(`nói "a" và \ "b"`)
	if again := SanitizeIdeaText(once); again != once {
		t.Errorf("a second pass changed %q into %q", once, again)
	}
}

func TestHashInputSeesCleanedAndRawTextAsTheSame(t *testing.T) {
	if HashInput(`nói "a" đi `) != HashInput("nói “a” đi") {
		t.Error("the same text before and after cleaning hashes differently")
	}
}
