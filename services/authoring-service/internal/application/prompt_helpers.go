package application

import (
	"strings"

	"authoring/internal/domain"
)

// briefLanguage is the language code ("vi" or "en") a project writes in.
func briefLanguage(lang domain.ContentLanguage) string {
	if lang == domain.LanguageEnglish {
		return "en"
	}
	return "vi"
}

// briefLanguageName names a language code for a prompt.
func briefLanguageName(language string) string {
	if language == "en" {
		return "English"
	}
	return "tiếng Việt"
}

// noteText is the Creator's note for a prompt, or "(không có)" when there is none.
func noteText(note string) string {
	if strings.TrimSpace(note) == "" {
		return "(không có)"
	}
	return strings.TrimSpace(note)
}

// withoutField returns fields with name removed.
func withoutField(fields []string, name string) []string {
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if f != name {
			out = append(out, f)
		}
	}
	return out
}
