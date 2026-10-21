package domain

import "testing"

// Every configurable step is known, and a name outside the list is refused.
func TestValidModelStep(t *testing.T) {
	for _, step := range ConfigurableModelSteps {
		if !ValidModelStep(step) {
			t.Errorf("ValidModelStep(%q) = false, want true", step)
		}
	}
	for _, step := range []string{"", "render", "Brief"} {
		if ValidModelStep(step) {
			t.Errorf("ValidModelStep(%q) = true, want false", step)
		}
	}
}

// Every project step the model picker shows is configurable.
func TestConfigurableStepsCoverProjectSteps(t *testing.T) {
	for _, step := range []string{"brief", "research", "story_frame", "story", "storyboard", "code", "publish"} {
		if !ValidModelStep(step) {
			t.Errorf("project step %q is not configurable", step)
		}
	}
}

// A step with no saved model gets the first model of the catalog, which is the
// local one; a saved model wins; a name outside the steps is dropped.
func TestWithStepModelFallback(t *testing.T) {
	got := WithStepModelFallback(map[string]string{"research": "zai-org/glm-5.3-flash", "render": "ollama"})
	if len(got) != len(ConfigurableModelSteps) {
		t.Fatalf("got %d steps, want %d: %v", len(got), len(ConfigurableModelSteps), got)
	}
	if got["research"] != "zai-org/glm-5.3-flash" {
		t.Errorf("research = %q, want the saved model", got["research"])
	}
	if got["brief"] != "ollama" || AuthoringModelCatalog[0].ID != "ollama" {
		t.Errorf("brief = %q, catalog first = %q; both must be ollama", got["brief"], AuthoringModelCatalog[0].ID)
	}
	if _, ok := got["render"]; ok {
		t.Error("a step outside ConfigurableModelSteps must be dropped")
	}
}

// Only the local model is flagged, so the GUI warns about it and nothing else.
func TestCatalogFlagsLocalModelOnly(t *testing.T) {
	for _, opt := range AuthoringModelCatalog {
		want := opt.ID == "ollama"
		if opt.Local != want {
			t.Errorf("catalog %q: Local=%v, want %v", opt.ID, opt.Local, want)
		}
	}
}
