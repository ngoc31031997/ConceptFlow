package application

import (
	"context"
	"errors"
	"fmt"
	"path"
	"time"

	"authoring/internal/domain"
)

// voiceRetryAfter is how long a line sent to the voice may stay without an
// answer before it is sent again.
const voiceRetryAfter = 2 * time.Minute

// ErrVoiceNotConfigured is returned when listening is asked for but the voice
// dispatcher is not wired.
var ErrVoiceNotConfigured = errors.New("chưa cấu hình đường đọc giọng")

// VoiceLinePort stores the reading of each line.
type VoiceLinePort interface {
	// ListVoiceLines returns every voice line of a project.
	ListVoiceLines(ctx context.Context, projectID string) ([]domain.VoiceLine, error)
	// RequestVoiceLine sets a line to pending with the given wording and voice,
	// replacing the earlier reading of that line.
	RequestVoiceLine(ctx context.Context, line domain.VoiceLine) error
	// ClaimVoiceLines returns the lines waiting to be read, plus those sent more than
	// retryAfter ago without an answer, and marks them running.
	ClaimVoiceLines(ctx context.Context, projectID string, retryAfter time.Duration) ([]domain.VoiceLine, error)
	// SaveVoiceResults records what the voice service reported. A result whose Ref
	// is no longer the line's TextSHA belongs to older wording and is dropped.
	SaveVoiceResults(ctx context.Context, projectID string, results []domain.VoiceLineResult) error
	// FailVoiceLines marks the lines failed with the reason.
	FailVoiceLines(ctx context.Context, projectID string, lineIDs []string, reason string) error
}

// VoiceDispatcherPort asks the orchestrator to have the waiting lines read.
type VoiceDispatcherPort interface {
	DispatchDraftVoice(ctx context.Context, projectID string) error
}

// VoiceLinesUseCase is the reading of script lines: which are waiting, and
// what the voice service answered. The orchestrator calls Claim and Save; the
// script step calls Request through ScriptDeskUseCase.Speak.
type VoiceLinesUseCase struct {
	store VoiceLinePort
}

// NewVoiceLinesUseCase wires the use case.
func NewVoiceLinesUseCase(store VoiceLinePort) *VoiceLinesUseCase {
	return &VoiceLinesUseCase{store: store}
}

// ClaimPending returns the lines to read now and marks them running.
func (uc *VoiceLinesUseCase) ClaimPending(ctx context.Context, projectID string) ([]domain.VoiceLine, error) {
	return uc.store.ClaimVoiceLines(ctx, projectID, voiceRetryAfter)
}

// SaveVoiceResults records the voice service's answer for lines. A line with an
// error is marked failed; one without must carry a file.
func (uc *VoiceLinesUseCase) SaveResults(ctx context.Context, projectID string, results []domain.VoiceLineResult) error {
	for _, r := range results {
		if r.LineID == "" || r.Ref == "" {
			return fmt.Errorf("%w: kết quả đọc thiếu mã câu hoặc ref", domain.ErrInvalidUnitEdit)
		}
		if r.Error == "" && r.AudioPath == "" {
			return fmt.Errorf("%w: câu %s không có file giọng và không có lỗi", domain.ErrInvalidUnitEdit, r.LineID)
		}
	}
	return uc.store.SaveVoiceResults(ctx, projectID, results)
}

// FailAll marks lines failed, for when the command to read them never reached the voice.
func (uc *VoiceLinesUseCase) FailAll(ctx context.Context, projectID string, reason string) error {
	lines, err := uc.store.ListVoiceLines(ctx, projectID)
	if err != nil {
		return err
	}
	var ids []string
	for _, l := range lines {
		if l.Status == domain.VoiceLineRunning {
			ids = append(ids, l.LineID)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	return uc.store.FailVoiceLines(ctx, projectID, ids, reason)
}

// audioFileName is the file name of a voice file, which is what the browser
// asks api-gateway for.
func audioFileName(l domain.VoiceLine) string { return path.Base(l.AudioPath) }

// audioIsCurrent reports whether the voice file of l was read from the
// wording and voice the line has now.
func audioIsCurrent(l domain.VoiceLine, text, voiceID string) bool {
	return l.TextSHA == domain.LineTextSHA(text) && l.VoiceID == voiceID
}
