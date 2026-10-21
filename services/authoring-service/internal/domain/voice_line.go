package domain

import "time"

// VoiceLineStatus is where the reading of one line stands.
type VoiceLineStatus string

const (
	// VoiceLinePending waits to be sent to the voice.
	VoiceLinePending VoiceLineStatus = "pending"
	// VoiceLineRunning has been sent; the file is not back yet.
	VoiceLineRunning VoiceLineStatus = "running"
	// VoiceLineReady has a voice file.
	VoiceLineReady VoiceLineStatus = "ready"
	// VoiceLineFailed could not be read; Error says why.
	VoiceLineFailed VoiceLineStatus = "failed"
)

// VoiceLine is the reading of one script line by one voice. It records the
// file the voice service wrote, never the audio itself.
type VoiceLine struct {
	ProjectID string `json:"project_id"`
	LineID    string `json:"line_id"`
	// Text is the wording sent to the voice; TextSHA is its hash, which tells
	// whether the line still has the wording the file was read from.
	Text    string `json:"text"`
	TextSHA string `json:"text_sha"`
	// VoiceID is the voice asked for; "" is the voice service's default.
	VoiceID  string          `json:"voice_id"`
	Language string          `json:"language"`
	Status   VoiceLineStatus `json:"status"`
	// AudioPath is the file under /shared; DurationSeconds its length.
	AudioPath       string  `json:"audio_path"`
	DurationSeconds float64 `json:"duration_seconds"`
	CacheKey        string  `json:"cache_key"`
	// CacheHit is true when the voice service found the file already made.
	CacheHit    bool      `json:"cache_hit"`
	Error       string    `json:"error"`
	RequestedAt time.Time `json:"requested_at"`
}

// VoiceLineResult is what the voice service reports for one line.
type VoiceLineResult struct {
	LineID string `json:"line_id"`
	// Ref is the TextSHA the line was sent with, echoed back.
	Ref             string  `json:"ref"`
	AudioPath       string  `json:"audio_path"`
	DurationSeconds float64 `json:"duration_seconds"`
	CacheKey        string  `json:"cache_key"`
	CacheHit        bool    `json:"cache_hit"`
	// Error is set when the line could not be read.
	Error string `json:"error"`
}
