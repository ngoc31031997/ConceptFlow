package domain

// VoiceCalibration is what a single voice has actually been measured doing
// (CR-016 FR43).
//
// The words-per-minute constants in languageProfiles are a guess that has never
// been checked against anything. Every synthesis run is a chance to check it:
// the Orchestrator knows both the text it sent and the real audio duration that
// came back, so the measurement is free.
//
// It is kept per voice, not per language (FR43.3). Two Vietnamese Azure voices
// read at visibly different speeds; averaging them together would cancel out
// exactly the thing being measured.
type VoiceCalibration struct {
	VoiceID     string  `json:"voice_id"`
	SampleCount int     `json:"sample_count"`
	TotalWords  int     `json:"total_words"`
	TotalSecond float64 `json:"total_seconds"`
}

// MinCalibrationSamples is how many synthesized projects a voice needs before
// its measured rate is trusted over the language default.
//
// Three, because one project can be unrepresentative (all short lines, or a
// stretch of dense formulas) while three of them rarely are — and waiting for
// more would mean the Creator never sees the benefit on a young channel.
const MinCalibrationSamples = 3

// WordsPerMinute reports the measured rate, and whether it should be trusted.
//
// Returns ok=false rather than a zero value when there is not enough evidence:
// the caller must fall back to the language default, and silently returning 0
// would divide by zero and stall every wait in the script.
func (c VoiceCalibration) WordsPerMinute() (float64, bool) {
	if c.SampleCount < MinCalibrationSamples || c.TotalSecond <= 0 || c.TotalWords <= 0 {
		return 0, false
	}
	return float64(c.TotalWords) / c.TotalSecond * 60.0, true
}

// EstimateNarrationDurationCalibrated is EstimateNarrationDuration using a
// voice's measured rate when there is enough of it.
//
// The floor still applies: a two-word line read by a fast voice would otherwise
// flash past before anyone could read the subtitle.
func EstimateNarrationDurationCalibrated(text string, language ContentLanguage, calibration VoiceCalibration) float64 {
	wpm, ok := calibration.WordsPerMinute()
	if !ok {
		return EstimateNarrationDuration(text, language)
	}

	words := countWords(text)
	if words == 0 {
		return minNarrationSeconds
	}
	seconds := float64(words) / wpm * 60.0
	if seconds < minNarrationSeconds {
		return minNarrationSeconds
	}
	return seconds
}
