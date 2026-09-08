package application

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"orchestrator/internal/domain"
)

// StepEvent is the already-parsed representation of one orchestrator.events
// message, handed to HandleStepEventUseCase by adapters/amqp/consumer.go
// after envelope decoding.
type StepEvent struct {
	MessageID string
	SagaID    string
	ProjectID string
	EventType string
	Payload   map[string]interface{}
}

// eventStepMap maps every one of the 12 consumed event types (6 success + 6
// failure) to the SagaStep they belong to (interface-contracts.md "Saga
// Instance Tracking"). scene_rendered is intentionally absent — it is a
// progress-only event handled separately (Rule 7) and never advances the
// state machine.
var eventStepMap = map[string]domain.StepName{
	"script_parsed":       domain.StepParseScript,
	"parse_failed":        domain.StepParseScript,
	"speech_synthesized":  domain.StepSynthesizeSpeech,
	"synthesis_failed":    domain.StepSynthesizeSpeech,
	"rendering_completed": domain.StepRenderScenes,
	"rendering_failed":    domain.StepRenderScenes,
	"video_assembled":     domain.StepAssembleVideo,
	"assembly_failed":     domain.StepAssembleVideo,
	"video_published":     domain.StepPublishVideo,
	"publish_failed":      domain.StepPublishVideo,
}

var failureEvents = map[string]bool{
	"parse_failed": true, "synthesis_failed": true,
	"rendering_failed": true, "assembly_failed": true, "publish_failed": true,
}

// HandleStepEventUseCase processes one incoming Saga event: validates the
// step is currently in_progress (Rule 4), aggregates event data onto
// Project, dispatches the next command (or ends the Saga), and publishes a
// ProgressMessage (Rule 7).
type HandleStepEventUseCase struct {
	repo      domain.ProjectRepositoryPort
	publisher domain.CommandPublisherPort
	progress  domain.ProgressPublisherPort
	logger    *slog.Logger
}

// NewHandleStepEventUseCase constructs the use case with its three port
// dependencies.
func NewHandleStepEventUseCase(repo domain.ProjectRepositoryPort, publisher domain.CommandPublisherPort, progress domain.ProgressPublisherPort, logger *slog.Logger) *HandleStepEventUseCase {
	if logger == nil {
		logger = slog.Default()
	}
	return &HandleStepEventUseCase{repo: repo, publisher: publisher, progress: progress, logger: logger}
}

// Execute handles a single StepEvent. It never returns an error for a valid
// "unexpected event" case (Flow 6) — the caller (amqp consumer) always acks
// after Execute returns, regardless of outcome, per Rule 4's "vẫn ack
// message" guidance; Execute only returns an error for genuine
// infrastructure failures (repository/publisher errors) that should surface
// as a processing failure.
func (uc *HandleStepEventUseCase) Execute(ctx context.Context, event StepEvent) error {
	if event.EventType == "scene_rendered" {
		return uc.handleSceneRenderedProgress(ctx, event)
	}

	stepName, known := eventStepMap[event.EventType]
	if !known {
		uc.logger.WarnContext(ctx, "unknown event_type, ignoring", "event_type", event.EventType, "saga_id", event.SagaID)
		return nil
	}

	step, err := uc.repo.GetStep(ctx, event.SagaID, stepName)
	if err != nil {
		return err
	}
	if step.Status != domain.SagaStepInProgress {
		// Rule 4 / Flow 6: out-of-order or redelivered event for a step that
		// is no longer in_progress — skip processing, still ack (handled by
		// the caller returning nil here).
		uc.logger.WarnContext(ctx, "unexpected event for non-in-progress step, skipping",
			"event_type", event.EventType, "saga_id", event.SagaID, "step", stepName, "step_status", step.Status)
		return nil
	}

	if failureEvents[event.EventType] {
		return uc.handleFailure(ctx, event, stepName)
	}
	return uc.handleSuccess(ctx, event, stepName)
}

// handleSceneRenderedProgress publishes the live progress ping (Rule 7) for
// a narration segment as Rendering substitutes its `self.wait(AUTO)` and
// works through the script. There is no per-scene clip to persist anymore
// (Manim-script input mode renders the whole script into one video) — this
// is a progress-only event.
func (uc *HandleStepEventUseCase) handleSceneRenderedProgress(ctx context.Context, event StepEvent) error {
	sceneIndex := intFromPayload(event.Payload, "scene_index")
	sceneTotal := intFromPayload(event.Payload, "scene_total")

	msg := domain.ProgressMessage{
		ProjectID:  event.ProjectID,
		Step:       string(domain.StepRenderScenes),
		Status:     "in_progress",
		SceneIndex: sceneIndex,
		SceneTotal: sceneTotal,
	}
	return uc.progress.PublishProgress(ctx, msg)
}

func (uc *HandleStepEventUseCase) handleFailure(ctx context.Context, event StepEvent, stepName domain.StepName) error {
	errMsg := stringFromPayload(event.Payload, "error_message")

	step := &domain.SagaStep{SagaID: event.SagaID, StepName: stepName, Status: domain.SagaStepFailed, ErrorMessage: &errMsg}
	if err := uc.repo.UpdateStep(ctx, step); err != nil {
		return err
	}
	// No compensating rollback (Rule 8) — only the status transition happens here.
	if err := uc.repo.UpdateStatus(ctx, event.ProjectID, domain.FailedStatusForStep(stepName)); err != nil {
		return err
	}

	return uc.progress.PublishProgress(ctx, domain.ProgressMessage{
		ProjectID:    event.ProjectID,
		Step:         string(stepName),
		Status:       "failed",
		ErrorMessage: &errMsg,
	})
}

func (uc *HandleStepEventUseCase) handleSuccess(ctx context.Context, event StepEvent, stepName domain.StepName) error {
	if err := uc.repo.UpdateStep(ctx, &domain.SagaStep{SagaID: event.SagaID, StepName: stepName, Status: domain.SagaStepCompleted}); err != nil {
		return err
	}

	project, err := uc.repo.Get(ctx, event.ProjectID)
	if err != nil {
		return err
	}

	var nextErr error
	switch stepName {
	case domain.StepParseScript:
		nextErr = uc.onScriptParsed(ctx, event, project)
	case domain.StepSynthesizeSpeech:
		nextErr = uc.onSpeechSynthesized(ctx, event, project)
	case domain.StepRenderScenes:
		nextErr = uc.onRenderingCompleted(ctx, event, project)
	case domain.StepAssembleVideo:
		nextErr = uc.onVideoAssembled(ctx, event, project)
	case domain.StepPublishVideo:
		nextErr = uc.onVideoPublished(ctx, event, project)
	}
	if nextErr == errAggregationFailed {
		// Rule 1 mismatch already transitioned the project to
		// failed_at_render_scenes and published its own "failed" progress
		// message inside onSpeechSynthesized — do not also report this
		// step as "completed".
		return nil
	}
	if nextErr != nil {
		return nextErr
	}

	return uc.progress.PublishProgress(ctx, domain.ProgressMessage{
		ProjectID: event.ProjectID,
		Step:      string(stepName),
		Status:    "completed",
	})
}

// errAggregationFailed is an internal sentinel returned by
// onSpeechSynthesized when Rule 1's scene_index integrity check fails. It is
// never returned to HandleStepEventUseCase.Execute's caller — handleSuccess
// intercepts it to skip the "completed" progress message for
// synthesize_speech (a "failed" one for render_scenes was already sent).
var errAggregationFailed = fmt.Errorf("scene aggregation failed (Rule 1)")

// onScriptParsed stores the initial scene set (one per "# NARRATION: ..."
// marker) and the Manim scene class name, then dispatches synthesize_speech
// directly. classify_scenes is a no-op pass-through in the Manim-script
// input mode — there is no per-scene template/category to classify since
// Rendering now executes the Creator's own script rather than choosing a
// pre-built template, so it is marked completed without ever being
// dispatched to Content Plugin.
func (uc *HandleStepEventUseCase) onScriptParsed(ctx context.Context, event StepEvent, project *domain.Project) error {
	scenes := parseInitialScenes(event.Payload)
	project.Scenes = scenes
	project.Chapters = parseChapters(event.Payload)
	project.ManimSceneClassName = stringFromPayload(event.Payload, "scene_class_name")
	if err := uc.repo.Save(ctx, project); err != nil {
		return err
	}

	if err := uc.repo.UpdateStep(ctx, &domain.SagaStep{SagaID: event.SagaID, StepName: domain.StepClassifyScenes, Status: domain.SagaStepCompleted}); err != nil {
		return err
	}
	if err := uc.progress.PublishProgress(ctx, domain.ProgressMessage{
		ProjectID: event.ProjectID, Step: string(domain.StepClassifyScenes), Status: "completed",
	}); err != nil {
		return err
	}

	if !project.TTSEnabled {
		return uc.skipSynthesizeSpeech(ctx, event.SagaID, event.ProjectID, project)
	}
	return uc.startSynthesizeSpeech(ctx, event.SagaID, event.ProjectID, project)
}

// skipSynthesizeSpeech is the TTS-disabled branch (CR-001 FR4.6): no audio is
// synthesized, so the step is closed as completed without ever being
// dispatched, each scene's duration is estimated from its narration text, and
// render_scenes is dispatched directly. Rendering and Video Assembly stay
// unaware of the toggle — they only ever see a populated DurationSeconds.
func (uc *HandleStepEventUseCase) skipSynthesizeSpeech(ctx context.Context, sagaID, projectID string, project *domain.Project) error {
	for i := range project.Scenes {
		project.Scenes[i].DurationSeconds = domain.EstimateNarrationDuration(
			project.Scenes[i].NarrationText, project.ContentLanguage,
		)
		project.Scenes[i].AudioPath = ""
	}
	if err := uc.repo.Save(ctx, project); err != nil {
		return err
	}

	if err := uc.repo.UpdateStep(ctx, &domain.SagaStep{SagaID: sagaID, StepName: domain.StepSynthesizeSpeech, Status: domain.SagaStepCompleted}); err != nil {
		return err
	}
	if err := uc.progress.PublishProgress(ctx, domain.ProgressMessage{
		ProjectID: projectID, Step: string(domain.StepSynthesizeSpeech), Status: "completed",
	}); err != nil {
		return err
	}

	return uc.startRenderScenes(ctx, sagaID, projectID, project)
}

// startSynthesizeSpeech dispatches synthesize_speech (Bước 3) — shared by
// onScriptParsed since classify_scenes no longer runs as a separate
// dispatched step.
func (uc *HandleStepEventUseCase) startSynthesizeSpeech(ctx context.Context, sagaID, projectID string, project *domain.Project) error {
	if err := uc.repo.UpdateStep(ctx, &domain.SagaStep{SagaID: sagaID, StepName: domain.StepSynthesizeSpeech, Status: domain.SagaStepInProgress}); err != nil {
		return err
	}
	payload := map[string]interface{}{
		"scenes": scenesToPayloadForSynthesis(project.Scenes, string(project.ContentLanguage), project.VoiceID),
	}
	if err := uc.dispatch(ctx, sagaID, projectID, "tts", string(domain.StepSynthesizeSpeech), payload); err != nil {
		return err
	}
	return uc.repo.UpdateStatus(ctx, projectID, domain.StatusSynthesizingSpeech)
}

// onSpeechSynthesized merges audio_path/duration_seconds by scene_index,
// enforces Rule 1's scene_index set integrity, and — if it holds —
// dispatches render_scenes (Bước 4).
func (uc *HandleStepEventUseCase) onSpeechSynthesized(ctx context.Context, event StepEvent, project *domain.Project) error {
	incoming := parseSynthesizedScenes(event.Payload)

	if mismatch := validateSceneIndexSets(project.Scenes, incoming); mismatch != "" {
		errMsg := fmt.Sprintf("scene_index set mismatch when aggregating render_scenes payload: %s", mismatch)
		if err := uc.repo.UpdateStep(ctx, &domain.SagaStep{
			SagaID: event.SagaID, StepName: domain.StepRenderScenes, Status: domain.SagaStepFailed, ErrorMessage: &errMsg,
		}); err != nil {
			return err
		}
		if err := uc.repo.UpdateStatus(ctx, event.ProjectID, domain.StatusFailedRenderScenes); err != nil {
			return err
		}
		if err := uc.progress.PublishProgress(ctx, domain.ProgressMessage{
			ProjectID:    event.ProjectID,
			Step:         string(domain.StepRenderScenes),
			Status:       "failed",
			ErrorMessage: &errMsg,
		}); err != nil {
			return err
		}
		return errAggregationFailed
	}

	for idx, data := range incoming {
		for i := range project.Scenes {
			if project.Scenes[i].SceneIndex == idx {
				project.Scenes[i].AudioPath = data.audioPath
				project.Scenes[i].DurationSeconds = data.durationSeconds
			}
		}
	}
	if err := uc.repo.Save(ctx, project); err != nil {
		return err
	}

	return uc.startRenderScenes(ctx, event.SagaID, event.ProjectID, project)
}

// startRenderScenes dispatches render_scenes (Bước 4) — shared by both the
// TTS-enabled path (after speech_synthesized merges real audio durations) and
// the TTS-disabled path (after estimated durations are filled in).
func (uc *HandleStepEventUseCase) startRenderScenes(ctx context.Context, sagaID, projectID string, project *domain.Project) error {
	if err := uc.repo.UpdateStep(ctx, &domain.SagaStep{SagaID: sagaID, StepName: domain.StepRenderScenes, Status: domain.SagaStepInProgress}); err != nil {
		return err
	}
	quality := project.RenderQuality
	if !quality.IsValid() {
		// Covers projects created before CR-004 added the field.
		quality = domain.DefaultRenderQuality
	}
	payload := map[string]interface{}{
		"scenes":           scenesToPayload(project.Scenes),
		"script_content":   project.ScriptContent,
		"scene_class_name": project.ManimSceneClassName,
		"render_quality":   string(quality),
	}
	if err := uc.dispatch(ctx, sagaID, projectID, "rendering", string(domain.StepRenderScenes), payload); err != nil {
		return err
	}
	return uc.repo.UpdateStatus(ctx, projectID, domain.StatusRendering)
}

// onRenderingCompleted stores the single rendered video path (rendering_completed
// now carries one "video_path" for the whole script, not per-scene clips —
// the Manim-script input mode renders one script into one video) and
// dispatches assemble_video with that video plus the ordered narration
// audio_paths from step 3.
func (uc *HandleStepEventUseCase) onRenderingCompleted(ctx context.Context, event StepEvent, project *domain.Project) error {
	videoPath := stringFromPayload(event.Payload, "video_path")

	// CR-002 FR10.5: the offsets must line up one-to-one with the scenes, in
	// order. A short or mismatched list would silently put every later
	// narration on the wrong offset, which is precisely the desynchronisation
	// this CR removes — so fail the saga loudly instead of assembling a video
	// whose audio drifts away from its animation.
	waitOffsets := floatSliceFromPayload(event.Payload, "wait_offsets")
	if len(waitOffsets) != len(project.Scenes) {
		errMsg := fmt.Sprintf(
			"rendering_completed carried %d wait_offsets but the project has %d scenes — "+
				"each `self.wait(AUTO)` must run exactly once, so it cannot sit inside a loop or a conditional",
			len(waitOffsets), len(project.Scenes),
		)
		if err := uc.repo.UpdateStep(ctx, &domain.SagaStep{
			SagaID: event.SagaID, StepName: domain.StepRenderScenes, Status: domain.SagaStepFailed, ErrorMessage: &errMsg,
		}); err != nil {
			return err
		}
		if err := uc.repo.UpdateStatus(ctx, event.ProjectID, domain.StatusFailedRenderScenes); err != nil {
			return err
		}
		if err := uc.progress.PublishProgress(ctx, domain.ProgressMessage{
			ProjectID:    event.ProjectID,
			Step:         string(domain.StepRenderScenes),
			Status:       "failed",
			ErrorMessage: &errMsg,
		}); err != nil {
			return err
		}
		return errAggregationFailed
	}

	project.RenderedVideoPath = &videoPath
	project.WaitOffsets = waitOffsets
	project.RenderedVideoSeconds = floatFromPayload(event.Payload, "video_duration_seconds")
	if err := uc.repo.Save(ctx, project); err != nil {
		return err
	}

	if err := uc.repo.UpdateStep(ctx, &domain.SagaStep{SagaID: event.SagaID, StepName: domain.StepAssembleVideo, Status: domain.SagaStepInProgress}); err != nil {
		return err
	}
	payload := assembleVideoPayload(project)
	if err := uc.dispatch(ctx, event.SagaID, event.ProjectID, "video_assembly", string(domain.StepAssembleVideo), payload); err != nil {
		return err
	}
	return uc.repo.UpdateStatus(ctx, event.ProjectID, domain.StatusAssemblingVideo)
}

// onVideoAssembled stores video_path and ends the Render Saga implicitly
// (no command/event for this transition — business-logic-model.md).
func (uc *HandleStepEventUseCase) onVideoAssembled(ctx context.Context, event StepEvent, project *domain.Project) error {
	videoPath := stringFromPayload(event.Payload, "video_path")
	project.VideoPath = &videoPath
	if err := uc.repo.Save(ctx, project); err != nil {
		return err
	}
	return uc.repo.UpdateStatus(ctx, event.ProjectID, domain.StatusReadyToPublish)
}

// onVideoPublished stores youtube_video_url and ends the Publish Saga.
func (uc *HandleStepEventUseCase) onVideoPublished(ctx context.Context, event StepEvent, project *domain.Project) error {
	url := stringFromPayload(event.Payload, "youtube_video_url")
	project.YoutubeVideoURL = &url
	if err := uc.repo.Save(ctx, project); err != nil {
		return err
	}
	return uc.repo.UpdateStatus(ctx, event.ProjectID, domain.StatusPublished)
}

// dispatch builds the outbound command envelope and publishes it via the
// injected CommandPublisherPort (Outbox-backed — module-structure.md).
func (uc *HandleStepEventUseCase) dispatch(ctx context.Context, sagaID, projectID, routingKey, eventType string, payload map[string]interface{}) error {
	envelope := domain.Envelope{
		MessageID: newUUID(),
		SagaID:    sagaID,
		ProjectID: projectID,
		EventType: eventType,
		Payload:   payload,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	return uc.publisher.PublishCommand(ctx, routingKey, envelope)
}

// assembleVideoPayload builds the assemble_video command payload: the single
// rendered video_path (from rendering_completed) plus the narration segments,
// each carrying the offset in that video where it belongs (CR-002), plus the
// static background_music_path (Rule 3).
//
// The offsets come from Rendering, not from adding up durations here. A Manim
// video is animation time plus narration time, so narration i starts after all
// the animation before it too — the old audio_segments payload, which Video
// Assembly laid end to end, drifted by the accumulated animation time (61.6s
// by the end of a 3.6-minute reference video).
//
// When narration is disabled (CR-001) narration_segments comes back empty and
// the video is assembled silent (or with background music only); subtitle_cues
// are sent whenever the Creator enabled subtitles, timed by the same offsets.
func assembleVideoPayload(project *domain.Project) map[string]interface{} {
	scenes := sortedScenes(project.Scenes)
	offsets := project.WaitOffsets

	narrationSegments := make([]map[string]interface{}, 0, len(scenes))
	for i, s := range scenes {
		if s.AudioPath == "" {
			continue
		}
		startTime := 0.0
		if i < len(offsets) {
			startTime = offsets[i]
		}
		narrationSegments = append(narrationSegments, map[string]interface{}{
			"audio_path": s.AudioPath,
			"start_time": startTime,
		})
	}
	var videoPath string
	if project.RenderedVideoPath != nil {
		videoPath = *project.RenderedVideoPath
	}
	payload := map[string]interface{}{
		"video_path":             videoPath,
		"narration_segments":     narrationSegments,
		"video_duration_seconds": project.RenderedVideoSeconds,
	}
	if project.BackgroundMusicPath != nil {
		payload["background_music_path"] = *project.BackgroundMusicPath
		volume := project.BackgroundMusicVolume
		if volume <= 0 || volume > 1 {
			volume = domain.DefaultBackgroundMusicVolume
		}
		payload["background_music_volume"] = volume
	}
	if project.SubtitlesEnabled {
		style := domain.DefaultSubtitleStyle()
		if project.SubtitleStyle != nil {
			style = *project.SubtitleStyle
		}
		payload["subtitle_style"] = style
		payload["subtitle_cues"] = subtitleCues(scenes, offsets)
	}
	return payload
}

// subtitleCues shows each narration line from the offset Rendering measured
// for it, for its own duration — real audio length when TTS ran, estimated
// reading time otherwise.
//
// The offsets are essential here for the same reason as the audio: cueing
// subtitles off a running total of durations would drift them away from the
// picture exactly as far as the narration used to drift.
func subtitleCues(scenes []domain.Scene, offsets []float64) []map[string]interface{} {
	cues := make([]map[string]interface{}, 0, len(scenes))
	for i, s := range scenes {
		start := 0.0
		if i < len(offsets) {
			start = offsets[i]
		}
		cues = append(cues, map[string]interface{}{
			"scene_index": s.SceneIndex,
			"text":        s.NarrationText,
			"start_time":  start,
			"end_time":    start + s.DurationSeconds,
		})
	}
	return cues
}

// validateSceneIndexSets enforces Rule 1: the scene_index set already
// accumulated on Project (from script_parsed + scenes_classified) must
// exactly match the incoming speech_synthesized set. Returns a non-empty
// description on mismatch, empty string when they match.
func validateSceneIndexSets(existing []domain.Scene, incoming map[int]synthesizedSceneData) string {
	existingSet := make(map[int]bool, len(existing))
	for _, s := range existing {
		existingSet[s.SceneIndex] = true
	}
	incomingSet := make(map[int]bool, len(incoming))
	for idx := range incoming {
		incomingSet[idx] = true
	}
	if len(existingSet) != len(incomingSet) {
		return fmt.Sprintf("expected %d scenes (script_parsed/scenes_classified), got %d from speech_synthesized",
			len(existingSet), len(incomingSet))
	}
	var missing, extra []int
	for idx := range existingSet {
		if !incomingSet[idx] {
			missing = append(missing, idx)
		}
	}
	for idx := range incomingSet {
		if !existingSet[idx] {
			extra = append(extra, idx)
		}
	}
	if len(missing) > 0 || len(extra) > 0 {
		sort.Ints(missing)
		sort.Ints(extra)
		return fmt.Sprintf("missing scene_index %v, unexpected scene_index %v", missing, extra)
	}
	return ""
}

func sortedScenes(scenes []domain.Scene) []domain.Scene {
	out := make([]domain.Scene, len(scenes))
	copy(out, scenes)
	sort.Slice(out, func(i, j int) bool { return out[i].SceneIndex < out[j].SceneIndex })
	return out
}
