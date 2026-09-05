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
	"script_parsed":         domain.StepParseScript,
	"parse_failed":          domain.StepParseScript,
	"scenes_classified":     domain.StepClassifyScenes,
	"classification_failed": domain.StepClassifyScenes,
	"speech_synthesized":    domain.StepSynthesizeSpeech,
	"synthesis_failed":      domain.StepSynthesizeSpeech,
	"rendering_completed":   domain.StepRenderScenes,
	"rendering_failed":      domain.StepRenderScenes,
	"video_assembled":       domain.StepAssembleVideo,
	"assembly_failed":       domain.StepAssembleVideo,
	"video_published":       domain.StepPublishVideo,
	"publish_failed":        domain.StepPublishVideo,
}

var failureEvents = map[string]bool{
	"parse_failed": true, "classification_failed": true, "synthesis_failed": true,
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

// handleSceneRenderedProgress does double duty: it publishes the live
// progress ping (Rule 7) AND is the only place a scene's ClipPath is ever
// stored — Rendering Service's approved interface-contracts.md carries the
// rendered clip's path as "animation_path" on each per-scene scene_rendered
// event, not in the batch-level rendering_completed event (which only
// carries "scene_count"). onRenderingCompleted (below) relies on ClipPath
// already being populated by the time it runs.
func (uc *HandleStepEventUseCase) handleSceneRenderedProgress(ctx context.Context, event StepEvent) error {
	sceneIndex := intFromPayload(event.Payload, "scene_index")
	sceneTotal := intFromPayload(event.Payload, "scene_total")
	clipPath := stringFromPayload(event.Payload, "animation_path")

	if clipPath != "" && sceneIndex != nil {
		project, err := uc.repo.Get(ctx, event.ProjectID)
		if err != nil {
			return err
		}
		for i := range project.Scenes {
			if project.Scenes[i].SceneIndex == *sceneIndex {
				project.Scenes[i].ClipPath = clipPath
			}
		}
		if err := uc.repo.Save(ctx, project); err != nil {
			return err
		}
	}

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
	case domain.StepClassifyScenes:
		nextErr = uc.onScenesClassified(ctx, event, project)
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

// onScriptParsed stores the initial scene set and dispatches classify_scenes
// (business-logic-model.md Bước 2).
func (uc *HandleStepEventUseCase) onScriptParsed(ctx context.Context, event StepEvent, project *domain.Project) error {
	scenes := parseInitialScenes(event.Payload)
	project.Scenes = scenes
	if err := uc.repo.Save(ctx, project); err != nil {
		return err
	}

	nextSagaID := event.SagaID
	if err := uc.repo.UpdateStep(ctx, &domain.SagaStep{SagaID: nextSagaID, StepName: domain.StepClassifyScenes, Status: domain.SagaStepInProgress}); err != nil {
		return err
	}
	payload := map[string]interface{}{
		"plugin_id": project.PluginID,
		"scenes":    scenesToPayloadForClassification(scenes, project.CategoryHint),
	}
	if err := uc.dispatch(ctx, event.SagaID, event.ProjectID, "content_plugin", string(domain.StepClassifyScenes), payload); err != nil {
		return err
	}
	return uc.repo.UpdateStatus(ctx, event.ProjectID, domain.StatusClassifyingScenes)
}

// onScenesClassified merges category/animation_template_id by scene_index
// and dispatches synthesize_speech (Bước 3).
func (uc *HandleStepEventUseCase) onScenesClassified(ctx context.Context, event StepEvent, project *domain.Project) error {
	incoming := parseClassifiedScenes(event.Payload)
	for idx, data := range incoming {
		for i := range project.Scenes {
			if project.Scenes[i].SceneIndex == idx {
				project.Scenes[i].Category = data.category
				project.Scenes[i].AnimationTemplateID = data.templateID
			}
		}
	}
	if err := uc.repo.Save(ctx, project); err != nil {
		return err
	}

	if err := uc.repo.UpdateStep(ctx, &domain.SagaStep{SagaID: event.SagaID, StepName: domain.StepSynthesizeSpeech, Status: domain.SagaStepInProgress}); err != nil {
		return err
	}
	payload := map[string]interface{}{
		"scenes": scenesToPayloadForSynthesis(project.Scenes, string(project.VoiceLanguage)),
	}
	if err := uc.dispatch(ctx, event.SagaID, event.ProjectID, "tts", string(domain.StepSynthesizeSpeech), payload); err != nil {
		return err
	}
	return uc.repo.UpdateStatus(ctx, event.ProjectID, domain.StatusSynthesizingSpeech)
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

	if err := uc.repo.UpdateStep(ctx, &domain.SagaStep{SagaID: event.SagaID, StepName: domain.StepRenderScenes, Status: domain.SagaStepInProgress}); err != nil {
		return err
	}
	payload := map[string]interface{}{"scenes": scenesToPayload(project.Scenes)}
	if err := uc.dispatch(ctx, event.SagaID, event.ProjectID, "rendering", string(domain.StepRenderScenes), payload); err != nil {
		return err
	}
	return uc.repo.UpdateStatus(ctx, event.ProjectID, domain.StatusRendering)
}

// onRenderingCompleted dispatches assemble_video, reusing ClipPath (already
// merged per-scene by handleSceneRenderedProgress as each scene finished —
// rendering_completed itself carries only "scene_count", no per-scene data)
// and audio_path stored from step 3 — never re-reading it from this event
// (Rule 2, Rendering Service does not return audio_path).
func (uc *HandleStepEventUseCase) onRenderingCompleted(ctx context.Context, event StepEvent, project *domain.Project) error {
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

// assembleVideoPayload builds the assemble_video command payload: clip_path
// (from rendering_completed) + audio_path (stored from step 3, Rule 2) per
// scene, plus the static background_music_path (Rule 3).
func assembleVideoPayload(project *domain.Project) map[string]interface{} {
	scenes := make([]map[string]interface{}, 0, len(project.Scenes))
	for _, s := range sortedScenes(project.Scenes) {
		scenes = append(scenes, map[string]interface{}{
			"scene_index": s.SceneIndex,
			"clip_path":   s.ClipPath,
			"audio_path":  s.AudioPath,
		})
	}
	payload := map[string]interface{}{"scenes": scenes}
	if project.BackgroundMusicPath != nil {
		payload["background_music_path"] = *project.BackgroundMusicPath
	}
	return payload
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
