package domain

import "fmt"

// TargetLength is the length a video aims for: the seconds the approved brief
// sets, and the range a tenth either side of it that the beat sheet, the
// script and the lock step check the total against.
type TargetLength struct {
	Seconds int
	Lo, Hi  int
}

// NewTargetLength returns the target length for a brief that asks for seconds.
func NewTargetLength(seconds int) TargetLength {
	return TargetLength{Seconds: seconds, Lo: (seconds*90 + 50) / 100, Hi: (seconds*110 + 50) / 100}
}

// Label writes the target as prompts and screens say it: "4:30 phút (khoảng
// 4:03–4:57)". It is empty when there is no target.
func (t TargetLength) Label(language string) string {
	if t.Seconds <= 0 {
		return ""
	}
	unit, about := "phút", "khoảng"
	if language == "en" {
		unit, about = "minutes", "about"
	}
	return fmt.Sprintf("%s %s (%s %s–%s)", FormatDuration(t.Seconds), unit, about, FormatDuration(t.Lo), FormatDuration(t.Hi))
}
