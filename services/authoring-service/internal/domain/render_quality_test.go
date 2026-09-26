package domain

import "testing"

// TestRenderQualityIsValid locks in the four presets Rendering can honour —
// 480p15 was added (bug report) as an explicit "for testing only" tier below
// 720p30, so a Creator can iterate on content/timing without paying for a
// heavier render.
func TestRenderQualityIsValid(t *testing.T) {
	for _, q := range []RenderQuality{Quality480p15, Quality720p30, Quality1080p60, Quality4k60} {
		if !q.IsValid() {
			t.Errorf("expected %q to be valid", q)
		}
	}
	if RenderQuality("480p30").IsValid() {
		t.Error("expected an unknown quality to be invalid")
	}
}
