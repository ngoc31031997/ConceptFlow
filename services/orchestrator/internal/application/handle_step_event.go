package application

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
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
	"script_validated":    domain.StepValidateScript,
	"validation_failed":   domain.StepValidateScript,
	"speech_synthesized":  domain.StepSynthesizeSpeech,
	"synthesis_failed":    domain.StepSynthesizeSpeech,
	"rendering_completed": domain.StepRenderScenes,
	"rendering_failed":    domain.StepRenderScenes,
	"video_assembled":     domain.StepAssembleVideo,
	"assembly_failed":     domain.StepAssembleVideo,
	// CR-021 FR61.4: qc_completed is the ONLY event qc_video ever produces.
	// There is deliberately no qc_failed counterpart — a QC that could not run
	// reports status="not_scored" and the saga carries on, so a broken measuring
	// tool can never hold a finished video hostage.
	"qc_completed": domain.StepQCVideo,
	// CR-007 D1: clips_generated is the ONLY event generate_clips ever
	// produces — same posture as qc_completed. A clip that failed to cut is
	// reported inside its own entry (status="error"), never as a step-level
	// failure, so there is no generate_clips_failed counterpart here.
	"clips_generated": domain.StepGenerateClips,
	"video_published": domain.StepPublishVideo,
	"publish_failed":  domain.StepPublishVideo,
}

var failureEvents = map[string]bool{
	"parse_failed": true, "validation_failed": true, "synthesis_failed": true,
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
	// qcReports stores the automated QC report (CR-021 D6). Optional/nil-checked
	// at its call sites for the same reason channelAssets is: callers and tests
	// written before CR-021 keep working, and a project simply ends up with no
	// report — which the publish gate already has to treat as "nothing to
	// enforce" anyway (FR61.4).
	qcReports domain.QCReportPort
	// channelAssets resolves the active intro/outro asset (CR-023 D1/D2).
	// Optional (nil-checked at the one call site) so existing callers/tests
	// built before CR-023 keep compiling and behaving exactly as before —
	// intro/outro simply stay unattached without one.
	channelAssets domain.ChannelAssetPort
	logger        *slog.Logger
}

// NewHandleStepEventUseCase constructs the use case with its port
// dependencies. channelAssets may be nil (CR-023's intro/outro lookup is then
// skipped entirely, same as if both toggles were off).
func NewHandleStepEventUseCase(repo domain.ProjectRepositoryPort, publisher domain.CommandPublisherPort, progress domain.ProgressPublisherPort, channelAssets domain.ChannelAssetPort, logger *slog.Logger) *HandleStepEventUseCase {
	if logger == nil {
		logger = slog.Default()
	}
	return &HandleStepEventUseCase{repo: repo, publisher: publisher, progress: progress, channelAssets: channelAssets, logger: logger}
}

// WithQCReports attaches the QC report store (CR-021 D6).
//
// A setter rather than another constructor parameter: NewHandleStepEventUseCase
// already takes five, and every existing caller and test would have to be
// edited to pass a nil for a dependency only one branch of the switch uses.
func (uc *HandleStepEventUseCase) WithQCReports(qcReports domain.QCReportPort) *HandleStepEventUseCase {
	uc.qcReports = qcReports
	return uc
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

	// CR-023 correction: channel_asset_rendered/channel_asset_normalized are
	// not saga-step events for any project — intro/outro belong to the
	// channel, not a project (see rendering/producer.py's
	// channel_asset_rendered_envelope docstring). They update Orchestrator's
	// own channel_asset_pointers projection instead of eventStepMap.
	if event.EventType == "channel_asset_rendered" || event.EventType == "channel_asset_normalized" {
		return uc.handleChannelAssetProjection(ctx, event)
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

// handleChannelAssetProjection upserts channel_asset_pointers from a
// channel_asset_normalized event (video-assembly's authoritative record of a
// newly-registered intro/outro, carrying asset_id + render_quality + version).
//
// channel_asset_rendered (rendering's raw output, video_path only, no
// asset_id/render_quality yet — see rendering/producer.py) is intentionally a
// no-op here: video-assembly is the one that ingests it, transcodes per
// RENDER_QUALITY and assigns an asset_id, and it re-announces the result as
// channel_asset_normalized — the only event shaped enough for this
// projection. Still explicitly matched (rather than falling into "unknown
// event_type") so it is not logged as a warning on every render.
func (uc *HandleStepEventUseCase) handleChannelAssetProjection(ctx context.Context, event StepEvent) error {
	if uc.channelAssets == nil || event.EventType != "channel_asset_normalized" {
		return nil
	}
	kind := stringFromPayload(event.Payload, "kind")
	quality := domain.RenderQuality(stringFromPayload(event.Payload, "render_quality"))
	assetID := stringFromPayload(event.Payload, "asset_id")
	version := 0
	if v := intFromPayload(event.Payload, "version"); v != nil {
		version = *v
	}
	if kind == "" || assetID == "" {
		uc.logger.WarnContext(ctx, "channel_asset_normalized missing kind/asset_id, ignoring", "saga_id", event.SagaID)
		return nil
	}
	return uc.channelAssets.UpsertChannelAssetPointer(ctx, kind, quality, assetID, version)
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
	case domain.StepValidateScript:
		nextErr = uc.onScriptValidated(ctx, event, project)
	case domain.StepSynthesizeSpeech:
		nextErr = uc.onSpeechSynthesized(ctx, event, project)
	case domain.StepRenderScenes:
		nextErr = uc.onRenderingCompleted(ctx, event, project)
	case domain.StepAssembleVideo:
		nextErr = uc.onVideoAssembled(ctx, event, project)
	case domain.StepQCVideo:
		nextErr = uc.onQCCompleted(ctx, event, project)
	case domain.StepGenerateClips:
		nextErr = uc.onClipsGenerated(ctx, event, project)
	case domain.StepPublishVideo:
		nextErr = uc.onVideoPublished(ctx, event, project)
	}
	if nextErr == errAwaitingReview {
		return nil
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

// errAwaitingReview is the same trick for CR-024's pause: validate_script did
// finish, but the saga is now parked at the review gate, so the trailing
// "completed" progress message must not overwrite the "awaiting_review" one
// the handler already sent. Reporting the step completed here would tell the
// GUI the pipeline is moving when it is waiting for a human.
var errAwaitingReview = fmt.Errorf("saga is parked awaiting review")

// onScriptParsed dispatches the validation pass (CR-020 FR56.1).
//
// Before CR-018 this step also carried the narration lines, which Script
// Processing had scraped out of the source with a regex over `# NARRATION:`
// comments. It cannot: narration now lives inside `self.narrate(...)` calls
// that may sit in a loop, a branch or a helper, so the only way to know what a
// script says is to run it. `script_parsed` therefore carries just the scene
// class name, and validate_script — a dry pass in Rendering — produces the rest.
//
// Putting the gate here, before synthesize_speech, is the point: a script that
// fails costs no TTS quota.
func (uc *HandleStepEventUseCase) onScriptParsed(ctx context.Context, event StepEvent, project *domain.Project) error {
	project.ManimSceneClassName = stringFromPayload(event.Payload, "scene_class_name")
	if err := uc.repo.Save(ctx, project); err != nil {
		return err
	}

	if err := uc.repo.UpdateStep(ctx, &domain.SagaStep{SagaID: event.SagaID, StepName: domain.StepValidateScript, Status: domain.SagaStepInProgress}); err != nil {
		return err
	}
	if err := uc.repo.UpdateStatus(ctx, event.ProjectID, domain.StatusValidatingScript); err != nil {
		return err
	}

	payload := map[string]interface{}{
		"script_content":   project.ScriptContent,
		"scene_class_name": project.ManimSceneClassName,
		"render_quality":   string(project.RenderQuality),
	}
	return uc.dispatch(ctx, event.SagaID, event.ProjectID, "rendering", string(domain.StepValidateScript), payload)
}

// onScriptValidated stores what the dry pass learned and moves on to speech.
//
// The scenes it stores come from *running* the script, so their order is the
// order the viewer will hear them — including narration produced inside loops,
// which no amount of reading the source text could have counted correctly.
func (uc *HandleStepEventUseCase) onScriptValidated(ctx context.Context, event StepEvent, project *domain.Project) error {
	project.Scenes = parseInitialScenes(event.Payload)
	beats := parseBeats(event.Payload)
	// Bug report (2026-09-12): the dry pass already knows whether the script
	// called `with self.clip(...)` at all — store it now (render_scenes will
	// overwrite with the real numbers later) so the warning below can fire
	// before TTS runs, not just after a Creator reaches generate_clips and
	// finds an empty ClipsPanel.
	project.ClipMarks = mapSliceFromPayload(event.Payload, "clip_marks")

	// CR-019 FR52.2: chapter sinh ra từ beat, không còn từ marker `# CHAPTER:`
	// rời rạc. Hai cơ chế song song sẽ trôi khỏi nhau, và beat vốn đã là chỗ
	// Creator quyết định cấu trúc.
	project.Chapters = chaptersFromBeats(beats)

	// Chốt phiên bản format tại thời điểm chạy (FR51.6), để sửa format sau này
	// không làm sai lệch cấu trúc của video đã dựng.
	format, err := uc.repo.GetVideoFormat(ctx, project.VideoFormatID, project.VideoFormatVersion)
	if err == nil {
		project.VideoFormatID = format.ID
		project.VideoFormatVersion = format.Version
	}
	if err := uc.repo.Save(ctx, project); err != nil {
		return err
	}

	// FR52.3/FR52.5: thiếu beat bắt buộc là dữ kiện chắc chắn nên chặn được;
	// mọi thứ còn lại (beat lạ, lặp quá, sai thứ tự) chỉ cảnh báo, vì chặn
	// render vì một con số mềm sẽ dạy Creator bỏ qua cả cơ chế.
	issues := format.ValidateBeats(beats)
	if len(domain.BlockingBeatIssues(issues)) > 0 {
		return uc.failValidationForBeats(ctx, event, domain.BlockingBeatIssues(issues))
	}

	// Mọi thứ còn lại đi kèm dàn ý để Creator duyệt cùng một lúc (CR-024 FR68.3):
	// duyệt nội dung và duyệt cảnh báo tách làm hai lần nhìn thì lần thứ hai sẽ
	// bị bỏ qua.
	project.Beats = beats
	project.ValidationWarnings = append(
		warningsFromPayload(event.Payload), beatIssueMessages(issues)...,
	)
	// Bug report (2026-09-12): overlap warnings from the dry pass's geometry
	// check (an unpositioned Text landing on an existing visual) ride the same
	// non-blocking ValidationWarnings slice as lint warnings and beat issues —
	// one list, one place the Creator looks, per CR-024 FR68.3.
	project.ValidationWarnings = append(
		project.ValidationWarnings, layoutWarningsFromPayload(event.Payload)...,
	)
	// Bug report: video_output_mode short/both promises a clip, but a clip
	// only ever comes from `with self.clip(...)` in the script — nothing else
	// produces one. Without this, the Creator only learned that after TTS,
	// render and QC had already run, from an empty ClipsPanel with no link
	// back to "the script never marked anything."
	if project.VideoOutputMode.WantsClips() && len(project.ClipMarks) == 0 {
		project.ValidationWarnings = append(project.ValidationWarnings,
			"Đã chọn tạo bản Shorts/TikTok, nhưng script này không có đoạn nào đánh dấu "+
				`with self.clip("tên"): — sẽ không có clip nào được tạo. Quay lại sửa script nếu muốn có clip.`,
		)
	}
	if err := uc.repo.Save(ctx, project); err != nil {
		return err
	}

	// CR-024 FR69.1 — điểm dừng, đặt ở đúng ranh giới giữa phần rẻ và phần đắt:
	// lượt dry vừa xong nên đã có đủ dữ liệu để dựng dàn ý, mà TTS thì chưa chạy
	// nên chưa tốn gì.
	if project.ReviewEnabled {
		if err := uc.repo.UpdateStatus(ctx, event.ProjectID, domain.StatusAwaitingReview); err != nil {
			return err
		}
		if err := uc.progress.PublishProgress(ctx, domain.ProgressMessage{
			ProjectID: event.ProjectID,
			Step:      string(domain.StepValidateScript),
			// "awaiting_review" chứ không phải "completed": một Saga đang dừng
			// mà giao diện trông như đang chạy là cách chắc chắn để Creator ngồi
			// đợi vô ích (FR69.5).
			Status: "awaiting_review",
		}); err != nil {
			return err
		}
		return errAwaitingReview
	}

	if !project.TTSEnabled {
		return uc.skipSynthesizeSpeech(ctx, event.SagaID, event.ProjectID, project)
	}
	return uc.startSynthesizeSpeech(ctx, event.SagaID, event.ProjectID, project)
}

// failValidationForBeats stops the saga at validate_script when the script is
// missing a beat the format requires.
func (uc *HandleStepEventUseCase) failValidationForBeats(ctx context.Context, event StepEvent, issues []domain.BeatIssue) error {
	parts := make([]string, 0, len(issues))
	for _, issue := range issues {
		parts = append(parts, issue.Message)
	}
	errMsg := strings.Join(parts, "; ")

	if err := uc.repo.UpdateStep(ctx, &domain.SagaStep{
		SagaID: event.SagaID, StepName: domain.StepValidateScript,
		Status: domain.SagaStepFailed, ErrorMessage: &errMsg,
	}); err != nil {
		return err
	}
	if err := uc.repo.UpdateStatus(ctx, event.ProjectID, domain.StatusFailedValidateScript); err != nil {
		return err
	}
	if err := uc.progress.PublishProgress(ctx, domain.ProgressMessage{
		ProjectID:    event.ProjectID,
		Step:         string(domain.StepValidateScript),
		Status:       "failed",
		ErrorMessage: &errMsg,
	}); err != nil {
		return err
	}
	return errAggregationFailed
}

// ResumeAfterReview restarts the saga once the Creator has approved the outline
// (CR-024 FR69.2).
//
// Exported because the approve use case needs exactly the branch this type
// already owns — with narration on, dispatch to TTS; with it off, estimate the
// durations and go straight to rendering. Duplicating that choice in a second
// place is how the two quietly stop agreeing.
func (uc *HandleStepEventUseCase) ResumeAfterReview(ctx context.Context, project *domain.Project) error {
	if !project.TTSEnabled {
		return uc.skipSynthesizeSpeech(ctx, project.SagaID, project.ProjectID, project)
	}
	return uc.startSynthesizeSpeech(ctx, project.SagaID, project.ProjectID, project)
}

// recordVoiceCalibration folds this project's totals into its voice's running
// average (CR-016 FR43).
func (uc *HandleStepEventUseCase) recordVoiceCalibration(ctx context.Context, project *domain.Project) error {
	words := 0
	seconds := 0.0
	for _, scene := range project.Scenes {
		words += len(strings.Fields(scene.NarrationText))
		seconds += scene.DurationSeconds
	}
	return uc.repo.RecordVoiceSamples(ctx, project.VoiceID, words, seconds)
}

// skipSynthesizeSpeech is the TTS-disabled branch (CR-001 FR4.6): no audio is
// synthesized, so the step is closed as completed without ever being
// dispatched, each scene's duration is estimated from its narration text, and
// render_scenes is dispatched directly. Rendering and Video Assembly stay
// unaware of the toggle — they only ever see a populated DurationSeconds.
func (uc *HandleStepEventUseCase) skipSynthesizeSpeech(ctx context.Context, sagaID, projectID string, project *domain.Project) error {
	// Use whatever this voice has actually been measured at, when there is
	// enough of it (CR-016 FR43.2). Falls back to the language constant on its
	// own, so an unmeasured voice behaves exactly as before.
	calibration, err := uc.repo.GetVoiceCalibration(ctx, project.VoiceID)
	if err != nil {
		calibration = domain.VoiceCalibration{VoiceID: project.VoiceID}
	}
	for i := range project.Scenes {
		project.Scenes[i].DurationSeconds = domain.EstimateNarrationDurationCalibrated(
			project.Scenes[i].NarrationText, project.ContentLanguage, calibration,
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

	// CR-016 FR43.1: this is the only moment where both halves of the
	// measurement exist together — the text we sent, and how long it really
	// took to read. Recording it costs nothing and is what eventually replaces
	// the guessed words-per-minute constants with something measured.
	//
	// Best-effort: a calibration row that fails to write is a slightly worse
	// estimate next time, not a reason to fail a project whose audio is done.
	if err := uc.recordVoiceCalibration(ctx, project); err != nil {
		uc.logger.Warn("could not record voice calibration",
			"project_id", project.ProjectID, "voice_id", project.VoiceID, "error", err)
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
	// CR-021 FR58: on-screen geometry per scene, needed by qc_video to score
	// the render. Never assigned before this fix — every project fell back to
	// QC's "not_scored" path regardless of QC_ENFORCE.
	project.LayoutMarks = mapSliceFromPayload(event.Payload, "layout_marks")
	// CR-007 FR19.2: the `with self.clip(...)` selections Rendering measured
	// on this real render pass, stored verbatim exactly like LayoutMarks —
	// Orchestrator never interprets a field inside, only carries it forward to
	// generate_clips's request-merging (buildClipRequests).
	project.ClipMarks = mapSliceFromPayload(event.Payload, "clip_marks")

	// CR-023 D2: resolve the channel intro/outro before publishing
	// assemble_video, and persist the resolved id on Project rather than
	// re-resolving on retry (RetryStepUseCase's Rule 5 rebuilds purely from
	// Project). A toggle off or no active asset both simply leave the field
	// nil — this never fails the saga.
	uc.resolveChannelAssets(ctx, project)

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

// resolveChannelAssets looks up the active intro/outro asset for each toggle
// the project has enabled (CR-023 FR67.1/67.2 default true) and sets
// IntroAssetID/OutroAssetID on project. Best-effort: a lookup failure or a
// missing asset is logged and leaves the field nil, never fails the saga —
// video-assembly is asked to attach an asset if one is there, not required to
// have one.
func (uc *HandleStepEventUseCase) resolveChannelAssets(ctx context.Context, project *domain.Project) {
	if uc.channelAssets == nil {
		return
	}
	quality := project.RenderQuality
	if !quality.IsValid() {
		quality = domain.DefaultRenderQuality
	}
	if project.IntroEnabled {
		if id, err := uc.channelAssets.LatestChannelAsset(ctx, "intro", quality); err != nil {
			uc.logger.Warn("could not resolve channel intro asset, proceeding without one",
				"project_id", project.ProjectID, "error", err)
		} else if id != "" {
			project.IntroAssetID = &id
		}
	}
	if project.OutroEnabled {
		if id, err := uc.channelAssets.LatestChannelAsset(ctx, "outro", quality); err != nil {
			uc.logger.Warn("could not resolve channel outro asset, proceeding without one",
				"project_id", project.ProjectID, "error", err)
		} else if id != "" {
			project.OutroAssetID = &id
		}
	}
}

// onVideoAssembled stores video_path and ends the Render Saga implicitly
// (no command/event for this transition — business-logic-model.md).
func (uc *HandleStepEventUseCase) onVideoAssembled(ctx context.Context, event StepEvent, project *domain.Project) error {
	videoPath := stringFromPayload(event.Payload, "video_path")
	project.VideoPath = &videoPath
	// CR-015 FR38.4: present only when subtitle_mode produced a caption
	// track. Absent (not null) for every other case — see producer.py's
	// video_assembled_envelope — so an empty string here would wrongly
	// overwrite a caption_path a previous, idempotent call already stored.
	if captionPath := stringFromPayload(event.Payload, "caption_path"); captionPath != "" {
		project.CaptionPath = &captionPath
	}
	project.IntroDurationSeconds = floatFromPayload(event.Payload, "intro_duration_seconds")
	if err := uc.repo.Save(ctx, project); err != nil {
		return err
	}

	// CR-021 D2: assembling the video no longer ends the Render Saga —
	// qc_video does. Setting ready_to_publish here instead would put the
	// publish button in front of the Creator before anything had looked at the
	// file, which is the whole gap this CR closes.
	if err := uc.repo.UpdateStep(ctx, &domain.SagaStep{SagaID: event.SagaID, StepName: domain.StepQCVideo, Status: domain.SagaStepInProgress}); err != nil {
		return err
	}
	// Same routing key as assemble_video: D1 puts the QC worker inside
	// video-assembly (it already has ffmpeg/ffprobe and the file itself), so
	// both commands ride the one video_assembly.commands queue and are told
	// apart by event_type — exactly how rendering already runs three commands.
	if err := uc.dispatch(ctx, event.SagaID, event.ProjectID, "video_assembly", string(domain.StepQCVideo), qcVideoPayload(project)); err != nil {
		return err
	}
	return uc.repo.UpdateStatus(ctx, event.ProjectID, domain.StatusRunningQC)
}

// onQCCompleted stores the QC report and dispatches generate_clips (CR-007
// D1) — QC no longer ends the Render Saga itself.
//
// It moves on unconditionally. Whatever the verdict — passed, has_findings, or
// not_scored — the project proceeds to generate_clips (FR61.4): QC decides
// what the Creator is *told*, never whether the pipeline finishes. The one
// place a blocking finding can actually stop anything is the Publish Saga,
// and only with QC_ENFORCE on (D5).
func (uc *HandleStepEventUseCase) onQCCompleted(ctx context.Context, event StepEvent, project *domain.Project) error {
	status := domain.QCStatus(stringFromPayload(event.Payload, "status"))
	switch status {
	case domain.QCStatusPassed, domain.QCStatusHasFindings, domain.QCStatusNotScored:
	default:
		// An unrecognised verdict is a not_scored, not a failure: this service
		// must not be the reason a video cannot ship because the QC worker
		// grew a fourth status.
		uc.logger.WarnContext(ctx, "unrecognised qc status, recording as not_scored",
			"status", string(status), "project_id", event.ProjectID)
		status = domain.QCStatusNotScored
	}

	report := domain.QCReport{
		ProjectID: event.ProjectID,
		Status:    status,
		Findings:  parseQCFindings(event.Payload),
	}
	if reason := stringFromPayload(event.Payload, "reason"); reason != "" {
		report.Reason = &reason
	}

	if uc.qcReports != nil {
		// Best-effort, for the same reason the whole step is: a report that
		// fails to persist is a Creator missing some advice, not grounds to
		// strand a finished video short of ready_to_publish.
		if err := uc.qcReports.SaveQCReport(ctx, report); err != nil {
			uc.logger.Warn("could not store qc report", "project_id", event.ProjectID, "error", err)
		}
	}

	if err := uc.repo.Save(ctx, project); err != nil {
		return err
	}

	// CR-007 follow-up: a project that only wants its long-form video (the
	// default, and every project created before this field existed — Go's
	// zero value for VideoOutputMode is "") has no clip requests to act on
	// anyway, so dispatching generate_clips would only be a round-trip that
	// comes back empty. Skip it and finish exactly like onClipsGenerated does.
	if !project.VideoOutputMode.WantsClips() {
		return uc.repo.UpdateStatus(ctx, event.ProjectID, domain.StatusReadyToPublish)
	}

	if err := uc.repo.UpdateStep(ctx, &domain.SagaStep{SagaID: event.SagaID, StepName: domain.StepGenerateClips, Status: domain.SagaStepInProgress}); err != nil {
		return err
	}
	// Same routing key/queue as assemble_video and qc_video (D1) — video-
	// assembly already has ffmpeg and the assembled file itself.
	if err := uc.dispatch(ctx, event.SagaID, event.ProjectID, "video_assembly", string(domain.StepGenerateClips), generateClipsPayload(project)); err != nil {
		return err
	}
	return uc.repo.UpdateStatus(ctx, event.ProjectID, domain.StatusGeneratingClips)
}

// onClipsGenerated stores the outcome of generate_clips and ends the Render
// Saga.
//
// It ends it unconditionally, exactly like onQCCompleted did before this CR:
// a clip that failed to cut (status="error" on that one entry of "clips") is
// surfaced to the Creator, never a reason to strand a finished, QC'd video
// short of ready_to_publish (D1 — a vertical clip is a derivative product).
func (uc *HandleStepEventUseCase) onClipsGenerated(ctx context.Context, event StepEvent, project *domain.Project) error {
	project.Clips = parseClipResults(event.Payload)
	if err := uc.repo.Save(ctx, project); err != nil {
		return err
	}
	return uc.repo.UpdateStatus(ctx, event.ProjectID, domain.StatusReadyToPublish)
}

// generateClipsPayload builds the generate_clips command from data already on
// Project (Rule 5 — a retry rebuilds it without re-running an earlier step):
// the assembled video, its subtitle cues (dịch về mốc 0 của từng clip là việc
// của video-assembly — D6, vì chỉ nó biết offset thật sau khi ghép intro), and
// the merged request list (D3).
//
// intro_duration_seconds comes from Project.IntroDurationSeconds, which
// onVideoAssembled stored from video-assembly's own measurement (the only
// place that number exists — see the field's doc comment on domain.Project).
// 0.0 when the project has no intro.
func generateClipsPayload(project *domain.Project) map[string]interface{} {
	scenes := sortedScenes(project.Scenes)
	return map[string]interface{}{
		"video_path":             derefString(project.VideoPath),
		"intro_duration_seconds": project.IntroDurationSeconds,
		"subtitle_cues":          subtitleCues(scenes, project.WaitOffsets),
		"requests":               buildClipRequests(project.ClipMarks, project.ClipRequests),
	}
}

// qcVideoPayload builds the qc_video command from data already on Project
// (Rule 5 — a retry rebuilds it without re-running an earlier step).
//
// It sends the *assembled* video_path, not RenderedVideoPath: the loudness,
// clipping and container checks of FR60 only mean anything against the file
// that will actually be uploaded, music bed, intro sting and all.
func qcVideoPayload(project *domain.Project) map[string]interface{} {
	scenes := sortedScenes(project.Scenes)

	quality := project.RenderQuality
	if !quality.IsValid() {
		quality = domain.DefaultRenderQuality
	}

	layoutMarks := project.LayoutMarks
	if layoutMarks == nil {
		layoutMarks = []map[string]interface{}{}
	}

	payload := map[string]interface{}{
		"video_path":         derefString(project.VideoPath),
		"render_quality":     string(quality),
		"narration_segments": narrationSegments(scenes, project.WaitOffsets),
		"layout_marks":       layoutMarks,
	}
	// Subtitle cues are sent whenever they exist, regardless of delivery mode:
	// FR60.4 checks whether two cues overlap in time, which is wrong in a
	// burn-in render exactly as much as in a sidecar .srt.
	payload["subtitle_cues"] = subtitleCues(scenes, project.WaitOffsets)
	return payload
}

// onVideoPublished stores youtube_video_url and ends the Publish Saga.
func (uc *HandleStepEventUseCase) onVideoPublished(ctx context.Context, event StepEvent, project *domain.Project) error {
	url := stringFromPayload(event.Payload, "youtube_video_url")
	project.YoutubeVideoURL = &url
	// CR-015 FR39.4: absent when no caption was requested, otherwise
	// "uploaded" | "skipped_no_scope" | "failed" — surfaced on the project
	// so a silently skipped or failed caption is not invisible.
	if captionStatus := stringFromPayload(event.Payload, "caption_status"); captionStatus != "" {
		project.CaptionStatus = &captionStatus
	}
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

	var videoPath string
	if project.RenderedVideoPath != nil {
		videoPath = *project.RenderedVideoPath
	}
	payload := map[string]interface{}{
		"video_path":             videoPath,
		"narration_segments":     narrationSegments(scenes, offsets),
		"video_duration_seconds": project.RenderedVideoSeconds,
	}
	// CR-023 D2: optional, nullable — nil (omitted) when the toggle is off or
	// no active channel_assets row was found, in which case video-assembly
	// assembles without an intro/outro exactly as it did before this CR.
	if project.IntroAssetID != nil {
		payload["intro_asset_id"] = *project.IntroAssetID
	}
	if project.OutroAssetID != nil {
		payload["outro_asset_id"] = *project.OutroAssetID
	}
	if project.BackgroundMusicPath != nil {
		payload["background_music_path"] = *project.BackgroundMusicPath
		volume := project.BackgroundMusicVolume
		if volume <= 0 || volume > 1 {
			volume = domain.DefaultBackgroundMusicVolume
		}
		payload["background_music_volume"] = volume
	}
	// SubtitleMode is normally already resolved by the time a Project
	// reaches here (start_render_saga.go at creation, project_repository.go's
	// Get for a pre-CR-015 row) — this fallback exists only so an invalid or
	// zero-value mode (a Project built by hand, e.g. in a test) degrades to
	// the one behaviour SubtitlesEnabled ever meant, rather than treating ""
	// as if it needed cues.
	mode := project.SubtitleMode
	if !mode.IsValid() {
		mode = domain.SubtitleModeFromLegacy(project.SubtitlesEnabled)
	}
	if mode.NeedsCues() {
		style := domain.DefaultSubtitleStyle()
		if project.SubtitleStyle != nil {
			style = *project.SubtitleStyle
		}
		payload["subtitle_style"] = style
		payload["subtitle_cues"] = subtitleCues(scenes, offsets)
		// Sent explicitly rather than relying on Video Assembly's own
		// default, so the wire contract does not depend on a default that
		// a future change there could alter out from under it.
		payload["subtitle_mode"] = string(mode)
	}
	return payload
}

// narrationSegments pairs each scene's audio file with the offset Rendering
// measured for it. Shared by assemble_video and qc_video (CR-021 FR60.2) so
// the overlap check runs against precisely the placement assembly used —
// a second, independently-built list is a second chance to disagree.
func narrationSegments(scenes []domain.Scene, offsets []float64) []map[string]interface{} {
	segments := make([]map[string]interface{}, 0, len(scenes))
	for i, s := range scenes {
		if s.AudioPath == "" {
			continue
		}
		startTime := 0.0
		if i < len(offsets) {
			startTime = offsets[i]
		}
		segments = append(segments, map[string]interface{}{
			"audio_path": s.AudioPath,
			"start_time": startTime,
		})
	}
	return segments
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
