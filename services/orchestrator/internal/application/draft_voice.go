package application

import (
	"context"
	"fmt"
	"time"

	"orchestrator/internal/domain"
)

// Event and command types of the draft voice: reading script lines outside a
// render saga.
const (
	// CommandSynthesizeLines asks tts to read lines; it is the EventType of the
	// command envelope and of the event_type inside its payload.
	CommandSynthesizeLines = "synthesize_lines"
	// EventLinesSynthesized is tts's answer with a file for every line.
	EventLinesSynthesized = "lines_synthesized"
	// EventLinesFailed is tts's answer when the lines could not be read.
	EventLinesFailed = "lines_synthesis_failed"
	// draftVoiceRoutingKey is the tts command queue's binding.
	draftVoiceRoutingKey = "tts"
)

// IsDraftVoiceEvent reports whether eventType is an answer to a synthesize_lines command.
func IsDraftVoiceEvent(eventType string) bool {
	return eventType == EventLinesSynthesized || eventType == EventLinesFailed
}

// VoiceLine is a script line waiting to be read, as authoring-service lists it.
type VoiceLine struct {
	LineID   string `json:"line_id"`
	Text     string `json:"text"`
	Language string `json:"language"`
	VoiceID  string `json:"voice_id"`
	// Ref is handed back with the result so authoring-service can tell which
	// wording of the line was read.
	Ref string `json:"ref"`
}

// VoiceLineResult is tts's answer for one line.
type VoiceLineResult struct {
	LineID          string  `json:"line_id"`
	Ref             string  `json:"ref"`
	AudioPath       string  `json:"audio_path"`
	DurationSeconds float64 `json:"duration_seconds"`
	CacheKey        string  `json:"cache_key"`
	CacheHit        bool    `json:"cache_hit"`
	Error           string  `json:"error"`
}

// DraftVoiceAuthoringPort is what the draft voice needs from authoring-service,
// which owns the script lines and the record of their reading.
type DraftVoiceAuthoringPort interface {
	// ClaimVoiceLines returns the lines waiting to be read and marks them as sent.
	ClaimVoiceLines(ctx context.Context, projectID string) ([]VoiceLine, error)
	// SaveVoiceResults records what tts answered.
	SaveVoiceResults(ctx context.Context, projectID string, results []VoiceLineResult) error
	// FailVoiceLines marks the lines sent and still unanswered as failed.
	FailVoiceLines(ctx context.Context, projectID, reason string) error
}

// DraftVoiceUseCase reads script lines aloud outside a render saga: it takes
// the lines waiting in authoring-service, sends them to tts in one command and
// hands tts's answer back. See ADR-0040.
type DraftVoiceUseCase struct {
	authoring DraftVoiceAuthoringPort
	commands  domain.CommandPublisherPort
	now       func() time.Time
}

// NewDraftVoiceUseCase wires the use case. commands is the Outbox-backed publisher.
func NewDraftVoiceUseCase(authoring DraftVoiceAuthoringPort, commands domain.CommandPublisherPort) *DraftVoiceUseCase {
	return &DraftVoiceUseCase{authoring: authoring, commands: commands, now: time.Now}
}

// Dispatch sends every waiting line of the project to tts. Nothing waiting is
// not an error. When the command cannot be queued the lines are marked failed
// so the Creator sees it, and the error is returned.
func (uc *DraftVoiceUseCase) Dispatch(ctx context.Context, projectID string) error {
	lines, err := uc.authoring.ClaimVoiceLines(ctx, projectID)
	if err != nil {
		return fmt.Errorf("claim voice lines: %w", err)
	}
	if len(lines) == 0 {
		return nil
	}
	payload := make([]map[string]interface{}, 0, len(lines))
	for _, l := range lines {
		payload = append(payload, map[string]interface{}{
			"line_id": l.LineID, "text": l.Text, "language": l.Language, "voice_id": l.VoiceID, "ref": l.Ref,
		})
	}
	id := newUUID()
	err = uc.commands.PublishCommand(ctx, draftVoiceRoutingKey, domain.Envelope{
		MessageID: id,
		SagaID:    id,
		ProjectID: projectID,
		EventType: CommandSynthesizeLines,
		Payload:   map[string]interface{}{"event_type": CommandSynthesizeLines, "lines": payload},
		Timestamp: uc.now().UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		reason := "không xếp được lệnh đọc giọng: " + err.Error()
		if failErr := uc.authoring.FailVoiceLines(ctx, projectID, reason); failErr != nil {
			return fmt.Errorf("publish synthesize_lines: %w (and marking the lines failed: %v)", err, failErr)
		}
		return fmt.Errorf("publish synthesize_lines: %w", err)
	}
	return nil
}

// HandleEvent hands tts's answer to authoring-service: a file for every line,
// or the reason the lines could not be read.
func (uc *DraftVoiceUseCase) HandleEvent(ctx context.Context, projectID, eventType string, payload map[string]interface{}) error {
	if eventType == EventLinesFailed {
		reason, _ := payload["error_message"].(string)
		if reason == "" {
			reason = "dịch vụ giọng không đọc được các câu"
		}
		return uc.authoring.FailVoiceLines(ctx, projectID, reason)
	}
	raw, ok := payload["lines"].([]interface{})
	if !ok {
		return fmt.Errorf("event %s has no lines", eventType)
	}
	results := make([]VoiceLineResult, 0, len(raw))
	for i, item := range raw {
		m, ok := item.(map[string]interface{})
		if !ok {
			return fmt.Errorf("event %s: line %d is not an object", eventType, i)
		}
		r := VoiceLineResult{}
		r.LineID, _ = m["line_id"].(string)
		r.Ref, _ = m["ref"].(string)
		r.AudioPath, _ = m["audio_path"].(string)
		r.CacheKey, _ = m["cache_key"].(string)
		r.CacheHit, _ = m["cache_hit"].(bool)
		r.Error, _ = m["error"].(string)
		r.DurationSeconds, _ = m["duration_seconds"].(float64)
		results = append(results, r)
	}
	return uc.authoring.SaveVoiceResults(ctx, projectID, results)
}

// FailUndelivered marks the lines failed when the command was dead-lettered:
// no tts worker picked it up.
func (uc *DraftVoiceUseCase) FailUndelivered(ctx context.Context, projectID, reason string) error {
	return uc.authoring.FailVoiceLines(ctx, projectID, reason)
}
