package application

import (
	"encoding/json"
	"testing"

	"orchestrator/internal/domain"
)

// Where a draft stands depends on illustrations_ready only for a
// Remotion project, and an authoring-service that does not send the field yet
// must leave the old placement (Code) in place.
func TestAuthoringSummaryPlacesTheIllustrationsStep(t *testing.T) {
	cases := []struct {
		name   string
		body   string
		engine domain.RenderEngine
		want   int
	}{
		{"remotion, drawings not ready", `{"story":true,"storyboard":true,"illustrations_ready":false}`, domain.RenderEngineRemotion, domain.FlowIllustrations},
		{"remotion, drawings ready", `{"story":true,"storyboard":true,"illustrations_ready":true}`, domain.RenderEngineRemotion, domain.FlowCode},
		{"remotion, field missing (older authoring)", `{"story":true,"storyboard":true}`, domain.RenderEngineRemotion, domain.FlowCode},
		{"manim never stops at illustrations", `{"story":true,"storyboard":true,"illustrations_ready":false}`, domain.RenderEngineManim, domain.FlowCode},
	}
	for _, c := range cases {
		var s AuthoringSummary
		if err := json.Unmarshal([]byte(c.body), &s); err != nil {
			t.Fatal(err)
		}
		got := domain.FlowStateFor(domain.StatusDraft, domain.WizardStepScript, s.Content(c.engine)).Step
		if got != c.want {
			t.Errorf("%s: step %d, want %d", c.name, got, c.want)
		}
	}
}
