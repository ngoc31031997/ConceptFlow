package domain

import "testing"

func TestRenderEngineIsValidAcceptsOnlyRemotion(t *testing.T) {
	if !RenderEngineRemotion.IsValid() {
		t.Error("expected remotion to be valid")
	}
	for _, e := range []RenderEngine{"manim", "", "blender"} {
		if e.IsValid() {
			t.Errorf("expected %q to be refused", e)
		}
	}
}

func TestDefaultRenderEngineIsRemotion(t *testing.T) {
	if DefaultRenderEngine != RenderEngineRemotion {
		t.Errorf("projects must default to remotion, got %q", DefaultRenderEngine)
	}
}
