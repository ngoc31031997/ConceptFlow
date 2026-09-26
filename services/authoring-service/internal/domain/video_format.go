package domain

import "fmt"

// VideoFormat is a channel's repeatable shape for a video (CR-019 FR51).
//
// Before this, the only statement of structure anywhere in the system was one
// sentence inside the prompt string the Creator copies to an external AI:
// "mở đầu gây chú ý → giải thích khái niệm cốt lõi → ví dụ minh họa → tổng kết".
// Nothing checked it, so every video invented its own shape.
//
// Making it data rather than code is the point of FR51.4/FR51.5: which beats a
// topic needs varies a lot, so a bộ beat hardcoded in the source would be wrong
// the first time a subject did not fit. Formats are rows, cloned and edited by
// the Creator, and versioned so a project rendered last month still reports the
// structure it was actually built against (FR51.6).
type VideoFormat struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version int    `json:"version"`

	// Bounds for the finished video, in seconds.
	MinSeconds float64 `json:"min_seconds"`
	MaxSeconds float64 `json:"max_seconds"`

	Beats []FormatBeat `json:"beats"`
}

// FormatBeat is one section of the shape.
type FormatBeat struct {
	ID   string `json:"id"`
	Role string `json:"role"`

	MinSeconds float64 `json:"min_seconds"`
	MaxSeconds float64 `json:"max_seconds"`

	// Required beats are the only ones whose absence blocks a render (FR52.5).
	// Everything else is advisory, because a budget checked against an estimate
	// carries the estimate's error with it.
	Required bool `json:"required"`

	// MaxRepeat > 1 allows the beat to appear several times — `variation` is
	// meant to, since two or three worked examples is the shape of the format.
	MaxRepeat int `json:"max_repeat"`
}

// BeatOccurrence is one `self.beat(...)` call the dry pass observed, tied to
// the narration line that opens it.
type BeatOccurrence struct {
	SceneIndex int    `json:"scene_index"`
	ID         string `json:"id"`
}

// BeatIssue is one problem found comparing a script against its format.
type BeatIssue struct {
	BeatID   string `json:"beat_id"`
	Message  string `json:"message"`
	Blocking bool   `json:"blocking"`
}

// ValidateBeats checks a script's observed beats against the format (FR52.3).
//
// Three classes of problem, and only the first blocks:
//
//   - a required beat is missing — a fact, not an estimate, so it is safe to
//     block on;
//   - a beat the format does not define, or one repeated more than allowed;
//   - beats out of the format's order.
//
// Duration budgets are deliberately NOT checked here: they are checked against
// an estimate that carries ±15% error, so they are warnings raised elsewhere
// (FR52.5). Blocking a render on a number that soft would teach the Creator to
// ignore the whole mechanism.
func (f VideoFormat) ValidateBeats(observed []BeatOccurrence) []BeatIssue {
	issues := make([]BeatIssue, 0)

	// A script that declares no beats at all has not opted into the format, so
	// it gets a warning rather than a wall.
	//
	// This is a deliberate hole, and a narrow one. Blocking every beatless
	// script would break the simplest videos and every script written before
	// beats existed, which is the fastest way to make the Creator resent the
	// mechanism instead of using it. Declare a single beat and the full check
	// applies — opting in is all or nothing, so the hole cannot be used to
	// dodge one inconvenient required beat.
	if len(observed) == 0 {
		return append(issues, BeatIssue{
			Message: "script chưa khai báo beat nào (self.beat(...)), nên cấu trúc video " +
				"không được kiểm. Thêm beat để hệ thống kiểm giúp bạn.",
			Blocking: false,
		})
	}

	defined := make(map[string]FormatBeat, len(f.Beats))
	order := make(map[string]int, len(f.Beats))
	for i, beat := range f.Beats {
		defined[beat.ID] = beat
		order[beat.ID] = i
	}

	counts := make(map[string]int, len(observed))
	for _, occurrence := range observed {
		counts[occurrence.ID]++
	}

	for _, beat := range f.Beats {
		if beat.Required && counts[beat.ID] == 0 {
			issues = append(issues, BeatIssue{
				BeatID:   beat.ID,
				Message:  fmt.Sprintf("thiếu beat bắt buộc %q (%s)", beat.ID, beat.Role),
				Blocking: true,
			})
		}
	}

	for _, occurrence := range observed {
		beat, known := defined[occurrence.ID]
		if !known {
			issues = append(issues, BeatIssue{
				BeatID:   occurrence.ID,
				Message:  fmt.Sprintf("beat %q không có trong format %q", occurrence.ID, f.Name),
				Blocking: false,
			})
			continue
		}
		allowed := beat.MaxRepeat
		if allowed < 1 {
			allowed = 1
		}
		if counts[occurrence.ID] > allowed {
			// Reported once per beat, not once per extra occurrence.
			counts[occurrence.ID] = 0
			issues = append(issues, BeatIssue{
				BeatID:   occurrence.ID,
				Message:  fmt.Sprintf("beat %q lặp quá số lần cho phép (%d)", occurrence.ID, allowed),
				Blocking: false,
			})
		}
	}

	previous := -1
	for _, occurrence := range observed {
		position, known := order[occurrence.ID]
		if !known {
			continue
		}
		if position < previous {
			issues = append(issues, BeatIssue{
				BeatID: occurrence.ID,
				Message: fmt.Sprintf(
					"beat %q xuất hiện sau một beat lẽ ra phải đứng sau nó — format %q định thứ tự khác",
					occurrence.ID, f.Name),
				Blocking: false,
			})
			break
		}
		previous = position
	}

	return issues
}

// BlockingBeatIssues filters to the ones that must stop a render.
func BlockingBeatIssues(issues []BeatIssue) []BeatIssue {
	blocking := make([]BeatIssue, 0, len(issues))
	for _, issue := range issues {
		if issue.Blocking {
			blocking = append(blocking, issue)
		}
	}
	return blocking
}
