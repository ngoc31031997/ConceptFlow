package application

import (
	"context"
	"testing"

	"orchestrator/internal/domain"
)

func newTestUseCase() (*HandleStepEventUseCase, *fakeRepo, *fakePublisher, *fakeProgress) {
	repo := newFakeRepo()
	pub := &fakePublisher{}
	prog := &fakeProgress{}
	return NewHandleStepEventUseCase(repo, pub, prog, nil), repo, pub, prog
}

// TestHandleStepEventUseCase_ScriptParsed_SkipsClassifyAndDispatchesSynthesizeSpeech
// guards the Manim-script input mode's saga shape: classify_scenes has no
// external round-trip anymore (no per-scene template to classify), so
// script_parsed must mark it completed synchronously and dispatch
// synthesize_speech directly — not wait for a scenes_classified event.
func TestHandleStepEventUseCase_ScriptParsed_DispatchesValidateScript(t *testing.T) {
	// CR-020: script_parsed giờ chỉ mang tên class Scene, và mở ra bước
	// validate_script — cổng chạy TRƯỚC TTS, nên script sai không tiêu quota giọng đọc.
	uc, repo, pub, _ := newTestUseCase()
	repo.projects["proj-1"] = &domain.Project{ProjectID: "proj-1", Status: domain.StatusParsingScript, ContentLanguage: domain.LanguageVietnamese, TTSEnabled: true, ScriptContent: "from conceptflow import *", RenderQuality: domain.Quality1080p60}
	repo.steps[stepKey("saga-1", domain.StepParseScript)] = &domain.SagaStep{SagaID: "saga-1", StepName: domain.StepParseScript, Status: domain.SagaStepInProgress}

	err := uc.Execute(context.Background(), StepEvent{
		SagaID: "saga-1", ProjectID: "proj-1", EventType: "script_parsed",
		Payload: map[string]interface{}{"scene_class_name": "DemoScene"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	project, _ := repo.Get(context.Background(), "proj-1")
	if project.Status != domain.StatusValidatingScript {
		t.Fatalf("expected validating_script, got %s", project.Status)
	}
	if project.ManimSceneClassName != "DemoScene" {
		t.Fatalf("expected scene_class_name stored, got %q", project.ManimSceneClassName)
	}

	last := pub.last()
	if last == nil || last.routingKey != "rendering" {
		t.Fatalf("expected validate_script dispatched to rendering, got %+v", last)
	}
	if last.envelope.EventType != string(domain.StepValidateScript) {
		t.Fatalf("expected validate_script command, got %q", last.envelope.EventType)
	}
	if last.envelope.Payload["script_content"] != "from conceptflow import *" {
		t.Fatalf("validate_script must carry the script, got %+v", last.envelope.Payload)
	}
}

func TestHandleStepEventUseCase_ScriptValidated_StoresScenesAndDispatchesSynthesizeSpeech(t *testing.T) {
	// Lời thoại đến từ việc CHẠY script (thứ tự runtime), nên bước này là nơi
	// Project.Scenes được điền, chứ không phải bước parse nữa.
	uc, repo, pub, _ := newTestUseCase()
	repo.projects["proj-1"] = &domain.Project{ProjectID: "proj-1", Status: domain.StatusValidatingScript, ContentLanguage: domain.LanguageVietnamese, TTSEnabled: true}
	repo.steps[stepKey("saga-1", domain.StepValidateScript)] = &domain.SagaStep{SagaID: "saga-1", StepName: domain.StepValidateScript, Status: domain.SagaStepInProgress}

	err := uc.Execute(context.Background(), StepEvent{
		SagaID: "saga-1", ProjectID: "proj-1", EventType: "script_validated",
		Payload: map[string]interface{}{
			"scenes": []interface{}{
				map[string]interface{}{"scene_index": float64(0), "narration_text": "n0"},
				map[string]interface{}{"scene_index": float64(1), "narration_text": "n1"},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	project, _ := repo.Get(context.Background(), "proj-1")
	if project.Status != domain.StatusSynthesizingSpeech {
		t.Fatalf("expected synthesizing_speech, got %s", project.Status)
	}
	if len(project.Scenes) != 2 {
		t.Fatalf("expected 2 scenes stored, got %d", len(project.Scenes))
	}

	last := pub.last()
	if last == nil || last.routingKey != "tts" {
		t.Fatalf("expected synthesize_speech dispatched to tts, got %+v", last)
	}
}

func TestHandleStepEventUseCase_Rule1_SceneIndexMismatch(t *testing.T) {
	uc, repo, pub, prog := newTestUseCase()
	repo.projects["proj-1"] = &domain.Project{
		ProjectID: "proj-1",
		Status:    domain.StatusSynthesizingSpeech,
		Scenes: []domain.Scene{
			{SceneIndex: 0, NarrationText: "n0"},
			{SceneIndex: 1, NarrationText: "n1"},
		},
	}
	repo.steps[stepKey("saga-1", domain.StepSynthesizeSpeech)] = &domain.SagaStep{SagaID: "saga-1", StepName: domain.StepSynthesizeSpeech, Status: domain.SagaStepInProgress}

	err := uc.Execute(context.Background(), StepEvent{
		SagaID: "saga-1", ProjectID: "proj-1", EventType: "speech_synthesized",
		Payload: map[string]interface{}{
			"scenes": []interface{}{
				// only scene_index 0 present — mismatch against the 2 accumulated scenes
				map[string]interface{}{"scene_index": float64(0), "audio_path": "a0.wav", "duration_seconds": float64(1.5)},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	project, _ := repo.Get(context.Background(), "proj-1")
	if project.Status != domain.StatusFailedRenderScenes {
		t.Fatalf("expected failed_at_render_scenes on mismatch, got %s", project.Status)
	}
	if len(pub.published) != 0 {
		t.Fatalf("expected no render_scenes command dispatched on mismatch, got %d published", len(pub.published))
	}
	if prog.last() == nil || prog.last().Status != "failed" || prog.last().ErrorMessage == nil {
		t.Fatalf("expected a failed progress message with error_message, got %+v", prog.last())
	}
}

func TestHandleStepEventUseCase_Rule4_UnexpectedEventSkipped(t *testing.T) {
	uc, repo, pub, prog := newTestUseCase()
	repo.projects["proj-1"] = &domain.Project{ProjectID: "proj-1", Status: domain.StatusRendering}
	// synthesize_speech step already completed — a redelivered/out-of-order event should be skipped.
	repo.steps[stepKey("saga-1", domain.StepSynthesizeSpeech)] = &domain.SagaStep{SagaID: "saga-1", StepName: domain.StepSynthesizeSpeech, Status: domain.SagaStepCompleted}

	err := uc.Execute(context.Background(), StepEvent{
		SagaID: "saga-1", ProjectID: "proj-1", EventType: "speech_synthesized",
		Payload: map[string]interface{}{},
	})
	if err != nil {
		t.Fatalf("expected no error (skip + ack), got %v", err)
	}
	if len(pub.published) != 0 {
		t.Fatalf("expected no command dispatched for unexpected event, got %d", len(pub.published))
	}
	if len(prog.messages) != 0 {
		t.Fatalf("expected no progress message for skipped unexpected event, got %d", len(prog.messages))
	}
	// status must remain unchanged
	project, _ := repo.Get(context.Background(), "proj-1")
	if project.Status != domain.StatusRendering {
		t.Fatalf("expected status unchanged, got %s", project.Status)
	}
}

func TestHandleStepEventUseCase_FailureEvent(t *testing.T) {
	uc, repo, _, prog := newTestUseCase()
	repo.projects["proj-1"] = &domain.Project{ProjectID: "proj-1", Status: domain.StatusRendering}
	repo.steps[stepKey("saga-1", domain.StepRenderScenes)] = &domain.SagaStep{SagaID: "saga-1", StepName: domain.StepRenderScenes, Status: domain.SagaStepInProgress}

	err := uc.Execute(context.Background(), StepEvent{
		SagaID: "saga-1", ProjectID: "proj-1", EventType: "rendering_failed",
		Payload: map[string]interface{}{"error_message": "manim template crashed"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	project, _ := repo.Get(context.Background(), "proj-1")
	if project.Status != domain.StatusFailedRenderScenes {
		t.Fatalf("expected failed_at_render_scenes, got %s", project.Status)
	}
	step, _ := repo.GetStep(context.Background(), "saga-1", domain.StepRenderScenes)
	if step.Status != domain.SagaStepFailed {
		t.Fatalf("expected step failed, got %s", step.Status)
	}
	if prog.last() == nil || *prog.last().ErrorMessage != "manim template crashed" {
		t.Fatalf("expected error_message forwarded in progress message, got %+v", prog.last())
	}
}

func TestHandleStepEventUseCase_SceneRenderedProgressOnly(t *testing.T) {
	uc, repo, pub, prog := newTestUseCase()
	repo.projects["proj-1"] = &domain.Project{ProjectID: "proj-1", Status: domain.StatusRendering}

	err := uc.Execute(context.Background(), StepEvent{
		SagaID: "saga-1", ProjectID: "proj-1", EventType: "scene_rendered",
		Payload: map[string]interface{}{"scene_index": float64(2), "scene_total": float64(5)},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pub.published) != 0 {
		t.Fatalf("expected scene_rendered to never dispatch a command, got %d", len(pub.published))
	}
	last := prog.last()
	if last == nil || last.Status != "in_progress" || last.Step != string(domain.StepRenderScenes) {
		t.Fatalf("expected in_progress render_scenes progress message, got %+v", last)
	}
	if last.SceneIndex == nil || *last.SceneIndex != 2 || last.SceneTotal == nil || *last.SceneTotal != 5 {
		t.Fatalf("expected scene_index=2 scene_total=5, got %+v", last)
	}
	// state machine must not advance for a progress-only event
	project, _ := repo.Get(context.Background(), "proj-1")
	if project.Status != domain.StatusRendering {
		t.Fatalf("expected status unchanged by scene_rendered, got %s", project.Status)
	}
}

// TestHandleStepEventUseCase_RenderingCompleted_DispatchesAssembleVideoWithNarrationSegments
// guards the Manim-script input mode's assemble_video contract: rendering
// produces one video_path for the whole script (not per-scene clips), and
// Video Assembly needs the narration audio from step 3 (Rule 2), each paired
// with the offset Rendering measured for it (CR-002).
func TestHandleStepEventUseCase_RenderingCompleted_DispatchesAssembleVideoWithNarrationSegments(t *testing.T) {
	uc, repo, pub, _ := newTestUseCase()
	repo.projects["proj-1"] = &domain.Project{
		ProjectID: "proj-1",
		Status:    domain.StatusRendering,
		Scenes: []domain.Scene{
			{SceneIndex: 0, AudioPath: "from-step3-0.wav"},
			{SceneIndex: 1, AudioPath: "from-step3-1.wav"},
		},
	}
	repo.steps[stepKey("saga-1", domain.StepRenderScenes)] = &domain.SagaStep{SagaID: "saga-1", StepName: domain.StepRenderScenes, Status: domain.SagaStepInProgress}

	err := uc.Execute(context.Background(), StepEvent{
		SagaID: "saga-1", ProjectID: "proj-1", EventType: "rendering_completed",
		Payload: map[string]interface{}{
			"video_path":             "rendered.mp4",
			"wait_offsets":           []interface{}{0.0, 8.5},
			"video_duration_seconds": 15.0,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	last := pub.last()
	if last == nil || last.routingKey != "video_assembly" {
		t.Fatalf("expected assemble_video dispatched, got %+v", last)
	}
	if last.envelope.Payload["video_path"] != "rendered.mp4" {
		t.Fatalf("expected video_path from rendering_completed, got %v", last.envelope.Payload["video_path"])
	}
	segments, ok := last.envelope.Payload["narration_segments"].([]map[string]interface{})
	if !ok || len(segments) != 2 {
		t.Fatalf("expected 2 narration_segments in assemble_video payload, got %v", last.envelope.Payload["narration_segments"])
	}
	if segments[0]["audio_path"] != "from-step3-0.wav" || segments[1]["audio_path"] != "from-step3-1.wav" {
		t.Fatalf("expected narration audio sourced from step 3 in scene_index order, got %v", segments)
	}
	if segments[0]["start_time"].(float64) != 0 || segments[1]["start_time"].(float64) != 8.5 {
		t.Fatalf("expected the measured offsets to be forwarded, got %v", segments)
	}

	project, _ := repo.Get(context.Background(), "proj-1")
	if project.RenderedVideoPath == nil || *project.RenderedVideoPath != "rendered.mp4" {
		t.Fatalf("expected RenderedVideoPath persisted, got %v", project.RenderedVideoPath)
	}
}

// TestHandleStepEventUseCase_VideoAssembled_StoresCaptionPath is CR-015
// FR38.4: a video_assembled event carrying caption_path must persist it on
// the project so the Publish Saga can pick it up later.
func TestHandleStepEventUseCase_VideoAssembled_StoresCaptionPath(t *testing.T) {
	uc, repo, _, _ := newTestUseCase()
	repo.projects["proj-1"] = &domain.Project{ProjectID: "proj-1", Status: domain.StatusAssemblingVideo}
	repo.steps[stepKey("saga-1", domain.StepAssembleVideo)] = &domain.SagaStep{SagaID: "saga-1", StepName: domain.StepAssembleVideo, Status: domain.SagaStepInProgress}

	err := uc.Execute(context.Background(), StepEvent{
		SagaID: "saga-1", ProjectID: "proj-1", EventType: "video_assembled",
		Payload: map[string]interface{}{
			"video_path":   "/shared/proj-1/video/final.mp4",
			"caption_path": "/shared/proj-1/video/final.srt",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	project, _ := repo.Get(context.Background(), "proj-1")
	if project.CaptionPath == nil || *project.CaptionPath != "/shared/proj-1/video/final.srt" {
		t.Fatalf("expected CaptionPath persisted, got %v", project.CaptionPath)
	}
}

// TestHandleStepEventUseCase_VideoAssembled_NoCaptionPathLeavesItNil mirrors
// the burn-in-only / subtitles-off case: no caption_path key in the event
// must not fabricate one.
func TestHandleStepEventUseCase_VideoAssembled_NoCaptionPathLeavesItNil(t *testing.T) {
	uc, repo, _, _ := newTestUseCase()
	repo.projects["proj-1"] = &domain.Project{ProjectID: "proj-1", Status: domain.StatusAssemblingVideo}
	repo.steps[stepKey("saga-1", domain.StepAssembleVideo)] = &domain.SagaStep{SagaID: "saga-1", StepName: domain.StepAssembleVideo, Status: domain.SagaStepInProgress}

	err := uc.Execute(context.Background(), StepEvent{
		SagaID: "saga-1", ProjectID: "proj-1", EventType: "video_assembled",
		Payload: map[string]interface{}{"video_path": "/shared/proj-1/video/final.mp4"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	project, _ := repo.Get(context.Background(), "proj-1")
	if project.CaptionPath != nil {
		t.Fatalf("expected no CaptionPath, got %v", *project.CaptionPath)
	}
}

// TestAssembleVideoPayload_SubtitlesEnabledSendsExplicitBurnInMode is CR-015:
// the assemble_video command must say subtitle_mode explicitly rather than
// leaning on Video Assembly's own default, so the wire contract does not
// depend on a default that a future change could alter out from under it.
func TestAssembleVideoPayload_SubtitlesEnabledSendsExplicitBurnInMode(t *testing.T) {
	rendered := "/shared/proj-1/video.mp4"
	project := &domain.Project{
		ProjectID: "proj-1", RenderedVideoPath: &rendered,
		TTSEnabled: false, SubtitlesEnabled: true,
	}

	payload := assembleVideoPayload(project)

	if payload["subtitle_mode"] != "burn_in" {
		t.Fatalf("expected explicit subtitle_mode=burn_in, got %v", payload["subtitle_mode"])
	}
}

// TestAssembleVideoPayload_TrackModeSendsCuesWithoutForcingBurnIn is CR-015:
// "track" needs subtitle_cues (Video Assembly writes them to .srt) but the
// mode string itself must say "track", not the "burn_in" the earlier,
// pre-GUI-wiring version of this code hardcoded.
func TestAssembleVideoPayload_TrackModeSendsCuesWithoutForcingBurnIn(t *testing.T) {
	rendered := "/shared/proj-1/video.mp4"
	project := &domain.Project{
		ProjectID: "proj-1", RenderedVideoPath: &rendered,
		TTSEnabled: false, SubtitleMode: domain.SubtitleModeTrack,
		Scenes: []domain.Scene{{SceneIndex: 0, NarrationText: "hi", DurationSeconds: 2}},
	}

	payload := assembleVideoPayload(project)

	if payload["subtitle_mode"] != "track" {
		t.Fatalf("expected subtitle_mode track, got %v", payload["subtitle_mode"])
	}
	if _, ok := payload["subtitle_cues"]; !ok {
		t.Fatal("expected subtitle_cues even for track mode — Video Assembly needs them to write the .srt")
	}
}

// TestAssembleVideoPayload_OffModeSendsNoCues guards the opposite: an
// explicit "off" must not leak cues into the payload even if SubtitleStyle
// happens to be set from an earlier UI state.
func TestAssembleVideoPayload_OffModeSendsNoCues(t *testing.T) {
	rendered := "/shared/proj-1/video.mp4"
	project := &domain.Project{
		ProjectID: "proj-1", RenderedVideoPath: &rendered,
		SubtitleMode: domain.SubtitleModeOff,
		Scenes:       []domain.Scene{{SceneIndex: 0, NarrationText: "hi", DurationSeconds: 2}},
	}

	payload := assembleVideoPayload(project)

	if _, ok := payload["subtitle_cues"]; ok {
		t.Fatal("expected no subtitle_cues when subtitle_mode is off")
	}
	if _, ok := payload["subtitle_mode"]; ok {
		t.Fatal("expected no subtitle_mode key at all when off")
	}
}

// TestHandleStepEventUseCase_VideoPublished_StoresCaptionStatus is CR-015
// FR39.4: a caption_status carried on video_published must persist onto the
// project so the GUI can show a silently skipped or failed caption.
func TestHandleStepEventUseCase_VideoPublished_StoresCaptionStatus(t *testing.T) {
	uc, repo, _, _ := newTestUseCase()
	repo.projects["proj-1"] = &domain.Project{ProjectID: "proj-1", Status: domain.StatusPublishing}
	repo.steps[stepKey("saga-1", domain.StepPublishVideo)] = &domain.SagaStep{
		SagaID: "saga-1", StepName: domain.StepPublishVideo, Status: domain.SagaStepInProgress,
	}

	err := uc.Execute(context.Background(), StepEvent{
		SagaID: "saga-1", ProjectID: "proj-1", EventType: "video_published",
		Payload: map[string]interface{}{
			"youtube_video_url": "https://youtu.be/abc",
			"caption_status":    "skipped_no_scope",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	project, _ := repo.Get(context.Background(), "proj-1")
	if project.CaptionStatus == nil || *project.CaptionStatus != "skipped_no_scope" {
		t.Fatalf("expected CaptionStatus persisted, got %v", project.CaptionStatus)
	}
}

// guards CR-001's branch: with narration off, no synthesize_speech command may
// reach the TTS Service, yet render_scenes must still carry a duration per
// scene so each `self.narrate(...)` still holds the animation for the right
// length. The branch now hangs off script_validated, since that is where the
// narration lines arrive (CR-018).
func TestHandleStepEventUseCase_ScriptValidated_TTSDisabled_SkipsSynthesisAndEstimatesDurations(t *testing.T) {
	uc, repo, pub, _ := newTestUseCase()
	repo.projects["proj-1"] = &domain.Project{
		ProjectID: "proj-1", Status: domain.StatusValidatingScript,
		ContentLanguage: domain.LanguageEnglish, TTSEnabled: false,
	}
	repo.steps[stepKey("saga-1", domain.StepValidateScript)] = &domain.SagaStep{SagaID: "saga-1", StepName: domain.StepValidateScript, Status: domain.SagaStepInProgress}

	err := uc.Execute(context.Background(), StepEvent{
		SagaID: "saga-1", ProjectID: "proj-1", EventType: "script_validated",
		Payload: map[string]interface{}{
			"scene_class_name": "DemoScene",
			"scenes": []interface{}{
				map[string]interface{}{"scene_index": float64(0), "narration_text": "the first narration line"},
				map[string]interface{}{"scene_index": float64(1), "narration_text": "the second narration line"},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, cmd := range pub.published {
		if cmd.routingKey == "tts" {
			t.Fatalf("expected no tts command when narration is disabled, got %+v", cmd)
		}
	}

	last := pub.last()
	if last == nil || last.routingKey != "rendering" {
		t.Fatalf("expected render_scenes dispatched to rendering, got %+v", last)
	}

	project, _ := repo.Get(context.Background(), "proj-1")
	if project.Status != domain.StatusRendering {
		t.Fatalf("expected rendering, got %s", project.Status)
	}
	for _, scene := range project.Scenes {
		if scene.DurationSeconds <= 0 {
			t.Fatalf("scene %d has no estimated duration", scene.SceneIndex)
		}
		if scene.AudioPath != "" {
			t.Fatalf("scene %d must have no audio when narration is disabled, got %q", scene.SceneIndex, scene.AudioPath)
		}
	}

	synthStep, _ := repo.GetStep(context.Background(), "saga-1", domain.StepSynthesizeSpeech)
	if synthStep.Status != domain.SagaStepCompleted {
		t.Fatalf("expected synthesize_speech closed as completed without dispatch, got %s", synthStep.Status)
	}
}

// TestAssembleVideoPayload_SubtitlesAndSilentVideo covers the assemble_video
// contract for a subtitled, narration-free project: no narration segments to
// mux, and one cue per scene placed at the offset Rendering measured.
func TestAssembleVideoPayload_SubtitlesAndSilentVideo(t *testing.T) {
	rendered := "/shared/proj-1/video.mp4"
	project := &domain.Project{
		ProjectID: "proj-1", RenderedVideoPath: &rendered,
		TTSEnabled: false, SubtitlesEnabled: true,
		Scenes: []domain.Scene{
			{SceneIndex: 1, NarrationText: "second", DurationSeconds: 3},
			{SceneIndex: 0, NarrationText: "first", DurationSeconds: 2},
		},
		// 9s of animation runs between the two narrations, so the second cue
		// starts at 11s, not at 2s.
		WaitOffsets:          []float64{0, 11},
		RenderedVideoSeconds: 20,
	}

	payload := assembleVideoPayload(project)

	if segments := payload["narration_segments"].([]map[string]interface{}); len(segments) != 0 {
		t.Fatalf("expected no narration segments for a silent video, got %v", segments)
	}
	if _, ok := payload["subtitle_style"]; !ok {
		t.Fatal("expected subtitle_style to fall back to the default style")
	}

	cues := payload["subtitle_cues"].([]map[string]interface{})
	if len(cues) != 2 {
		t.Fatalf("expected 2 cues, got %d", len(cues))
	}
	if cues[0]["text"] != "first" || cues[0]["start_time"].(float64) != 0 || cues[0]["end_time"].(float64) != 2 {
		t.Fatalf("unexpected first cue: %+v", cues[0])
	}
	if cues[1]["start_time"].(float64) != 11 || cues[1]["end_time"].(float64) != 14 {
		t.Fatalf("expected second cue at its measured offset, got %+v", cues[1])
	}
}

// TestAssembleVideoPayload_NarrationCarriesMeasuredOffsets is the CR-002
// regression on the Orchestrator side: each narration segment must go out with
// the offset Rendering measured, never the running total of durations that the
// old audio_segments payload implied.
func TestAssembleVideoPayload_NarrationCarriesMeasuredOffsets(t *testing.T) {
	rendered := "/shared/proj-1/video.mp4"
	project := &domain.Project{
		ProjectID: "proj-1", RenderedVideoPath: &rendered,
		TTSEnabled: true,
		Scenes: []domain.Scene{
			{SceneIndex: 0, NarrationText: "first", DurationSeconds: 2, AudioPath: "/shared/a0.wav"},
			{SceneIndex: 1, NarrationText: "second", DurationSeconds: 3, AudioPath: "/shared/a1.wav"},
			{SceneIndex: 2, NarrationText: "third", DurationSeconds: 4, AudioPath: "/shared/a2.wav"},
		},
		WaitOffsets:          []float64{0, 11.5, 30.25},
		RenderedVideoSeconds: 48,
	}

	payload := assembleVideoPayload(project)

	segments := payload["narration_segments"].([]map[string]interface{})
	if len(segments) != 3 {
		t.Fatalf("expected 3 narration segments, got %d", len(segments))
	}
	wantStarts := []float64{0, 11.5, 30.25}
	for i, seg := range segments {
		if got := seg["start_time"].(float64); got != wantStarts[i] {
			// Summing durations would have produced 0, 2, 5 here.
			t.Fatalf("segment %d: expected start_time %v, got %v", i, wantStarts[i], got)
		}
	}
	if payload["video_duration_seconds"].(float64) != 48 {
		t.Fatalf("expected the rendered duration to be forwarded, got %v", payload["video_duration_seconds"])
	}
	if _, ok := payload["audio_segments"]; ok {
		t.Fatal("the pre-CR-002 audio_segments key must not be sent anymore")
	}
}

// TestOnRenderingCompleted_RejectsMismatchedOffsetCount covers CR-002 FR10.5:
// a wait_offsets list that does not line up with the scenes must fail the saga
// rather than silently assembling a video whose audio drifts.
func TestOnRenderingCompleted_RejectsMismatchedOffsetCount(t *testing.T) {
	uc, repo, publisher, _ := newTestUseCase()
	rendered := "/shared/proj-1/video.mp4"
	project := &domain.Project{
		ProjectID: "proj-1", SagaID: "saga-1", RenderedVideoPath: &rendered,
		Scenes: []domain.Scene{
			{SceneIndex: 0, NarrationText: "first", DurationSeconds: 2},
			{SceneIndex: 1, NarrationText: "second", DurationSeconds: 3},
		},
	}
	_ = repo.Save(context.Background(), project)
	_ = repo.UpdateStep(context.Background(), &domain.SagaStep{
		SagaID: "saga-1", StepName: domain.StepRenderScenes, Status: domain.SagaStepInProgress,
	})

	err := uc.Execute(context.Background(), StepEvent{
		SagaID: "saga-1", ProjectID: "proj-1", EventType: "rendering_completed",
		Payload: map[string]interface{}{
			"video_path":   rendered,
			"wait_offsets": []interface{}{0.0}, // only one, but there are two scenes
		},
	})
	if err != nil {
		t.Fatalf("expected the mismatch to be handled, not returned: %v", err)
	}

	step, _ := repo.GetStep(context.Background(), "saga-1", domain.StepRenderScenes)
	if step.Status != domain.SagaStepFailed {
		t.Fatalf("expected render_scenes to be marked failed, got %s", step.Status)
	}
	for _, cmd := range publisher.published {
		if cmd.envelope.EventType == string(domain.StepAssembleVideo) {
			t.Fatal("assemble_video must not be dispatched with mismatched offsets")
		}
	}
}

func TestHandleStepEventUseCase_SpeechSynthesized_RecordsVoiceCalibration(t *testing.T) {
	// CR-016 FR43.1: đây là thời điểm duy nhất có đủ cả hai nửa của phép đo —
	// văn bản đã gửi đi, và thời lượng thật đọc ra.
	uc, repo, _, _ := newTestUseCase()
	repo.projects["proj-1"] = &domain.Project{
		ProjectID: "proj-1", Status: domain.StatusSynthesizingSpeech,
		ContentLanguage: domain.LanguageVietnamese, TTSEnabled: true, VoiceID: "vi-NamMinh",
		Scenes: []domain.Scene{
			{SceneIndex: 0, NarrationText: "một hai ba bốn năm"},
			{SceneIndex: 1, NarrationText: "sáu bảy tám chín mười"},
		},
	}
	repo.steps[stepKey("saga-1", domain.StepSynthesizeSpeech)] = &domain.SagaStep{SagaID: "saga-1", StepName: domain.StepSynthesizeSpeech, Status: domain.SagaStepInProgress}

	err := uc.Execute(context.Background(), StepEvent{
		SagaID: "saga-1", ProjectID: "proj-1", EventType: "speech_synthesized",
		Payload: map[string]interface{}{
			"scenes": []interface{}{
				map[string]interface{}{"scene_index": float64(0), "audio_path": "/a/0.wav", "duration_seconds": float64(2.0)},
				map[string]interface{}{"scene_index": float64(1), "audio_path": "/a/1.wav", "duration_seconds": float64(3.0)},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	c, _ := repo.GetVoiceCalibration(context.Background(), "vi-NamMinh")
	if c.SampleCount != 1 || c.TotalWords != 10 || c.TotalSecond != 5.0 {
		t.Fatalf("số đo sai: %+v", c)
	}
}

func TestHandleStepEventUseCase_TTSDisabled_UsesCalibratedRateWhenAvailable(t *testing.T) {
	// FR43.2: đủ mẫu thì dùng số đo thật thay cho hằng số theo ngôn ngữ.
	uc, repo, _, _ := newTestUseCase()
	for i := 0; i < domain.MinCalibrationSamples; i++ {
		// 300 wpm — nhanh gấp đôi hằng số tiếng Việt (140).
		_ = repo.RecordVoiceSamples(context.Background(), "vi-Fast", 1000, 200)
	}
	repo.projects["proj-1"] = &domain.Project{
		ProjectID: "proj-1", Status: domain.StatusValidatingScript,
		ContentLanguage: domain.LanguageVietnamese, TTSEnabled: false, VoiceID: "vi-Fast",
	}
	repo.steps[stepKey("saga-1", domain.StepValidateScript)] = &domain.SagaStep{SagaID: "saga-1", StepName: domain.StepValidateScript, Status: domain.SagaStepInProgress}

	narration := "một hai ba bốn năm sáu bảy tám chín mười"
	err := uc.Execute(context.Background(), StepEvent{
		SagaID: "saga-1", ProjectID: "proj-1", EventType: "script_validated",
		Payload: map[string]interface{}{
			"scenes": []interface{}{
				map[string]interface{}{"scene_index": float64(0), "narration_text": narration},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	project, _ := repo.Get(context.Background(), "proj-1")
	got := project.Scenes[0].DurationSeconds
	if plain := domain.EstimateNarrationDuration(narration, domain.LanguageVietnamese); got >= plain {
		t.Fatalf("phải dùng số đo thật (%v) thay hằng số (%v)", got, plain)
	}
}

func TestHandleStepEventUseCase_ScriptValidated_BlocksWhenARequiredBeatIsMissing(t *testing.T) {
	// CR-019 FR52.3/52.5: chặn ở dữ kiện chắc chắn (beat bắt buộc thiếu), và
	// chặn TRƯỚC TTS nên không tốn quota giọng đọc.
	uc, repo, pub, prog := newTestUseCase()
	repo.projects["proj-1"] = &domain.Project{
		ProjectID: "proj-1", Status: domain.StatusValidatingScript,
		ContentLanguage: domain.LanguageVietnamese, TTSEnabled: true,
		VideoFormatID: domain.DefaultVideoFormatID,
	}
	repo.steps[stepKey("saga-1", domain.StepValidateScript)] = &domain.SagaStep{SagaID: "saga-1", StepName: domain.StepValidateScript, Status: domain.SagaStepInProgress}

	err := uc.Execute(context.Background(), StepEvent{
		SagaID: "saga-1", ProjectID: "proj-1", EventType: "script_validated",
		Payload: map[string]interface{}{
			"scenes": []interface{}{
				map[string]interface{}{"scene_index": float64(0), "narration_text": "n0"},
			},
			// Có beat nhưng thiếu concrete/pattern/recap/cta.
			"beats": []interface{}{
				map[string]interface{}{"scene_index": float64(0), "id": "hook"},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	project, _ := repo.Get(context.Background(), "proj-1")
	if project.Status != domain.StatusFailedValidateScript {
		t.Fatalf("muốn failed_at_validate_script, có %s", project.Status)
	}
	if last := pub.last(); last != nil && last.routingKey == "tts" {
		t.Fatal("không được gửi lệnh TTS khi beat bắt buộc còn thiếu")
	}
	if got := prog.last(); got == nil || got.Status != "failed" {
		t.Fatalf("phải báo failed lên GUI: %+v", got)
	}
}

func TestHandleStepEventUseCase_ScriptValidated_DerivesChaptersFromBeats(t *testing.T) {
	// FR52.2: chapter sinh từ beat thay cho marker `# CHAPTER:` rời rạc.
	uc, repo, _, _ := newTestUseCase()
	repo.projects["proj-1"] = &domain.Project{
		ProjectID: "proj-1", Status: domain.StatusValidatingScript,
		ContentLanguage: domain.LanguageVietnamese, TTSEnabled: true,
	}
	repo.steps[stepKey("saga-1", domain.StepValidateScript)] = &domain.SagaStep{SagaID: "saga-1", StepName: domain.StepValidateScript, Status: domain.SagaStepInProgress}

	beats := []interface{}{}
	for i, id := range []string{"hook", "concrete", "pattern", "recap", "cta"} {
		beats = append(beats, map[string]interface{}{"scene_index": float64(i), "id": id})
	}
	scenes := []interface{}{}
	for i := 0; i < 5; i++ {
		scenes = append(scenes, map[string]interface{}{"scene_index": float64(i), "narration_text": "n"})
	}

	if err := uc.Execute(context.Background(), StepEvent{
		SagaID: "saga-1", ProjectID: "proj-1", EventType: "script_validated",
		Payload: map[string]interface{}{"scenes": scenes, "beats": beats},
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	project, _ := repo.Get(context.Background(), "proj-1")
	if len(project.Chapters) != 5 {
		t.Fatalf("muốn 5 chapter từ 5 beat, có %d", len(project.Chapters))
	}
	if project.Chapters[0].Title != "hook" || project.Chapters[0].SceneIndex != 0 {
		t.Fatalf("chapter đầu sai: %+v", project.Chapters[0])
	}
	// FR51.6: phiên bản format được chốt lại tại thời điểm chạy.
	if project.VideoFormatVersion == 0 {
		t.Error("phải chốt phiên bản format")
	}
}
