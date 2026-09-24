package domain

import "testing"

func TestWizardStepForRoute(t *testing.T) {
	for route, want := range map[string]int{
		"/": WizardStepIdea, "/create/script/settings": WizardStepConfig,
		"/create/script/outline": WizardStepScript, "/create/script/storyboard": WizardStepScript,
		"/create/script/code": WizardStepScript,
	} {
		if got, ok := WizardStepForRoute(route); !ok || got != want {
			t.Errorf("%s: got %d,%v want %d", route, got, ok, want)
		}
	}
	if _, ok := WizardStepForRoute("/projects/x/render"); ok {
		t.Error("non-wizard route must be rejected")
	}
}
