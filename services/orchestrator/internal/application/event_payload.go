package application

import "orchestrator/internal/domain"

// This file holds pure functions that decode the loosely-typed
// map[string]interface{} event payloads (as produced by encoding/json
// Unmarshal into interface{}) into typed Go values. Payloads from AMQP
// events are trusted (nfr-design-patterns.md "Security Pattern" — upstream
// services already validated before publishing), so these helpers favor
// simplicity over exhaustive validation: a missing/malformed field yields
// its zero value rather than an error.

type synthesizedSceneData struct {
	audioPath       string
	durationSeconds float64
}

// parseInitialScenes decodes the script_parsed payload's scenes array into
// domain.Scene values (business-logic-model.md Bước 1/2).
func parseInitialScenes(payload map[string]interface{}) []domain.Scene {
	raw, _ := payload["scenes"].([]interface{})
	scenes := make([]domain.Scene, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		scene := domain.Scene{
			SceneIndex:       intFromMap(m, "scene_index"),
			NarrationText:    stringFromMap(m, "narration_text"),
			IllustrationHint: stringFromMap(m, "illustration_hint"),
		}
		if v, ok := m["code_snippet"].(string); ok {
			scene.CodeSnippet = &v
		}
		if v, ok := m["code_language"].(string); ok {
			scene.CodeLanguage = &v
		}
		scenes = append(scenes, scene)
	}
	return scenes
}

// parseSynthesizedScenes decodes the speech_synthesized payload into a
// scene_index-keyed map of audio_path/duration_seconds.
func parseSynthesizedScenes(payload map[string]interface{}) map[int]synthesizedSceneData {
	raw, _ := payload["scenes"].([]interface{})
	out := make(map[int]synthesizedSceneData, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		idx := intFromMap(m, "scene_index")
		out[idx] = synthesizedSceneData{
			audioPath:       stringFromMap(m, "audio_path"),
			durationSeconds: floatFromMap(m, "duration_seconds"),
		}
	}
	return out
}

// scenesToPayload serializes accumulated Scene values into the
// []map[string]interface{} shape used by outbound command payloads.
func scenesToPayload(scenes []domain.Scene) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(scenes))
	for _, s := range sortedScenes(scenes) {
		m := map[string]interface{}{
			"scene_index":       s.SceneIndex,
			"narration_text":    s.NarrationText,
			"illustration_hint": s.IllustrationHint,
		}
		if s.CodeSnippet != nil {
			m["code_snippet"] = *s.CodeSnippet
		}
		if s.CodeLanguage != nil {
			m["code_language"] = *s.CodeLanguage
		}
		if s.Category != "" {
			m["category"] = s.Category
		}
		if s.AnimationTemplateID != "" {
			m["animation_template_id"] = s.AnimationTemplateID
		}
		if s.AudioPath != "" {
			m["audio_path"] = s.AudioPath
		}
		if s.DurationSeconds != 0 {
			m["duration_seconds"] = s.DurationSeconds
		}
		out = append(out, m)
	}
	return out
}

// scenesToPayloadForSynthesis is scenesToPayload plus per-scene "language"
// and "voice_id" keys — the TTS Service's approved interface-contracts.md
// requires synthesize_speech's scenes to each carry "language" (mirroring the
// synchronous TTSEnginePort.synthesize(text, language, ...) signature this
// command replaced), not a single top-level voice_language field. voice_id
// (CR-001) selects which bundled Piper voice reads the line; an empty value
// lets the TTS Service fall back to that language's default voice.
func scenesToPayloadForSynthesis(scenes []domain.Scene, language, voiceID string) []map[string]interface{} {
	out := scenesToPayload(scenes)
	for _, m := range out {
		m["language"] = language
		if voiceID != "" {
			m["voice_id"] = voiceID
		}
	}
	return out
}

func stringFromPayload(payload map[string]interface{}, key string) string {
	return stringFromMap(payload, key)
}

func intFromPayload(payload map[string]interface{}, key string) *int {
	if _, ok := payload[key]; !ok {
		return nil
	}
	v := intFromMap(payload, key)
	return &v
}

func stringFromMap(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func intFromMap(m map[string]interface{}, key string) int {
	switch v := m[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	default:
		return 0
	}
}

func floatFromMap(m map[string]interface{}, key string) float64 {
	switch v := m[key].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	default:
		return 0
	}
}
