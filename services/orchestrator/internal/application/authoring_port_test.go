package application

import (
	"encoding/json"
	"testing"

	"orchestrator/internal/domain"
)

// Where a draft stands depends on illustrations_ready, and an
// authoring-service that does not send the field must leave the placement at
// Code.
func TestAuthoringSummaryPlacesTheIllustrationsStep(t *testing.T) {
	cases := []struct {
		name string
		body string
		want int
	}{
		{"drawings not ready", `{"story":true,"storyboard":true,"illustrations_ready":false}`, domain.FlowIllustrations},
		{"drawings ready", `{"story":true,"storyboard":true,"illustrations_ready":true}`, domain.FlowCode},
		{"field missing", `{"story":true,"storyboard":true}`, domain.FlowCode},
	}
	for _, c := range cases {
		var s AuthoringSummary
		if err := json.Unmarshal([]byte(c.body), &s); err != nil {
			t.Fatal(err)
		}
		got := domain.FlowStateFor(domain.StatusDraft, domain.WizardStepScript, s.Content()).Step
		if got != c.want {
			t.Errorf("%s: step %d, want %d", c.name, got, c.want)
		}
	}
}
