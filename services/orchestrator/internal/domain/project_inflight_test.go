package domain

import "testing"

func TestProjectStatusIsInFlight(t *testing.T) {
	inFlight := []ProjectStatus{StatusParsingScript, StatusValidatingScript, StatusSynthesizingSpeech,
		StatusRendering, StatusAssemblingVideo, StatusRunningQC, StatusGeneratingClips, StatusPublishing}
	idle := []ProjectStatus{StatusDraft, StatusAwaitingReview, StatusReadyToPublish, StatusPublished,
		StatusFailedRenderScenes, StatusFailedPublishVideo}
	for _, s := range inFlight {
		if !s.IsInFlight() {
			t.Errorf("%s should be in flight", s)
		}
	}
	for _, s := range idle {
		if s.IsInFlight() {
			t.Errorf("%s should not be in flight", s)
		}
	}
}
