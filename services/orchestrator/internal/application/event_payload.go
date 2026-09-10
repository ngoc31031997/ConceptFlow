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
// parseChapters reads the optional `# CHAPTER:` markers (CR-006 FR15.1).
// Absent or malformed entries yield no chapters rather than an error: a video
// without chapters is fine, a saga that fails over a description detail is not.
func parseChapters(payload map[string]interface{}) []domain.Chapter {
	raw, _ := payload["chapters"].([]interface{})
	chapters := make([]domain.Chapter, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		title := stringFromMap(m, "title")
		if title == "" {
			continue
		}
		chapters = append(chapters, domain.Chapter{
			SceneIndex: intFromMap(m, "scene_index"),
			Title:      title,
		})
	}
	return chapters
}

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

// floatFromPayload reads a JSON number; absent or malformed yields 0.
func floatFromPayload(payload map[string]interface{}, key string) float64 {
	switch v := payload[key].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	default:
		return 0
	}
}

// floatSliceFromPayload reads a JSON array of numbers. A nil result is
// indistinguishable from an empty array here on purpose: the caller checks the
// length against its own scene count either way.
func floatSliceFromPayload(payload map[string]interface{}, key string) []float64 {
	raw, ok := payload[key].([]interface{})
	if !ok {
		return nil
	}
	out := make([]float64, 0, len(raw))
	for _, item := range raw {
		switch v := item.(type) {
		case float64:
			out = append(out, v)
		case int:
			out = append(out, float64(v))
		default:
			// A non-numeric entry means the payload is not what we think it is;
			// returning a short slice makes the caller's length check fail,
			// which is the outcome we want.
			return out
		}
	}
	return out
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

// parseBeats reads the beat markers the dry pass observed (CR-019 FR52.1).
//
// Each carries the narration index it opens on, not a timestamp — the real
// timestamp is only known after the render pass measures it (CR-002), and
// storing a guess here is what CR-006 FR15 deliberately avoided for chapters.
func parseBeats(payload map[string]interface{}) []domain.BeatOccurrence {
	raw, ok := payload["beats"].([]interface{})
	if !ok {
		return nil
	}
	beats := make([]domain.BeatOccurrence, 0, len(raw))
	for _, item := range raw {
		entry, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		id, _ := entry["id"].(string)
		if id == "" {
			continue
		}
		index, _ := entry["scene_index"].(float64)
		beats = append(beats, domain.BeatOccurrence{SceneIndex: int(index), ID: id})
	}
	return beats
}

// chaptersFromBeats turns the observed beats into YouTube chapters
// (CR-019 FR52.2).
//
// Beats replace the old `# CHAPTER:` markers rather than living beside them:
// two mechanisms describing the same structure drift apart, and the beat is
// already where the Creator decided what the section is.
//
// The title is the beat id as written. Making it prettier would mean guessing
// what the section is about, which is exactly what CR-006 refused to let a
// model do.
func chaptersFromBeats(beats []domain.BeatOccurrence) []domain.Chapter {
	if len(beats) == 0 {
		return nil
	}
	chapters := make([]domain.Chapter, 0, len(beats))
	for _, beat := range beats {
		chapters = append(chapters, domain.Chapter{SceneIndex: beat.SceneIndex, Title: beat.ID})
	}
	return chapters
}
