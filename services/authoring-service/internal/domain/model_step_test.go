package domain

import "testing"

// The code step and the drawings refuse the local Ollama model.
func TestModelAllowedForStep(t *testing.T) {
	cases := []struct {
		step, model string
		want        bool
	}{
		{"code", "deepseek-ai/deepseek-v4.1-flash", true},
		{"code", "", true}, // server default (a Hive model)
		{"code", "ollama", false},
		{"code", "ollama/llama3.2", false},
		{"illustrations", "ollama", false},
		{"story", "ollama", true},
		{"storyboard", "ollama/llama3.2", true},
	}
	for _, c := range cases {
		if got := ModelAllowedForStep(c.step, c.model); got != c.want {
			t.Errorf("ModelAllowedForStep(%q, %q) = %v, want %v", c.step, c.model, got, c.want)
		}
	}
	for _, opt := range AuthoringModelCatalog {
		if opt.CodeOK != ModelAllowedForStep("code", opt.ID) {
			t.Errorf("catalog %q: CodeOK=%v disagrees with ModelAllowedForStep", opt.ID, opt.CodeOK)
		}
	}
}
