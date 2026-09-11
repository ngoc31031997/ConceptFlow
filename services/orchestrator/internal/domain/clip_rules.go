package domain

import (
	"fmt"
	"os"
	"strconv"
)

// Preset length thresholds for CR-007's vertical clips (FR19.5/C2b). Mirrors
// video-assembly/domain/clip_rules.py's CLIP_PRESET_* constants exactly —
// both services validate independently (D4: GUI catches it early, video-
// assembly re-checks against the real post-intro offset), so they must agree
// on the same env var names and defaults or a request the GUI accepted could
// still be rejected downstream for a different reason.
const (
	defaultClipPresetShortMaxSeconds = 60.0
	defaultClipPresetLongMinSeconds  = 60.0
	defaultClipPresetLongMaxSeconds  = 180.0

	ClipPresetShort = "short"
	ClipPresetLong  = "long"
)

// clipFloatEnvOrDefault reads a float env var, falling back to def when the
// variable is unset or does not parse — same posture as qc_rules.py's
// _as_float: a mistyped threshold must not crash validation, only leave it
// at its shipped default.
func clipFloatEnvOrDefault(key string, def float64) float64 {
	raw := os.Getenv(key)
	if raw == "" {
		return def
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return def
	}
	return v
}

// ValidateClipDuration reports whether durationSeconds is within bounds for
// preset (FR19.6), reading thresholds from CLIP_PRESET_SHORT_MAX_SECONDS /
// CLIP_PRESET_LONG_MIN_SECONDS / CLIP_PRESET_LONG_MAX_SECONDS (C2b — third
// parties change these over time, so they must be config, not a constant).
//
// Returns nil when durationSeconds fits preset, or an error with a specific,
// actionable message otherwise — FR19.6 explicitly rules out silently
// truncating a segment that does not fit.
func ValidateClipDuration(durationSeconds float64, preset string) error {
	shortMax := clipFloatEnvOrDefault("CLIP_PRESET_SHORT_MAX_SECONDS", defaultClipPresetShortMaxSeconds)
	longMin := clipFloatEnvOrDefault("CLIP_PRESET_LONG_MIN_SECONDS", defaultClipPresetLongMinSeconds)
	longMax := clipFloatEnvOrDefault("CLIP_PRESET_LONG_MAX_SECONDS", defaultClipPresetLongMaxSeconds)

	switch preset {
	case ClipPresetShort:
		if durationSeconds <= 0 {
			return fmt.Errorf("duration %.1fs is not valid for preset \"short\"", durationSeconds)
		}
		if durationSeconds > shortMax {
			return fmt.Errorf("duration %.1fs is too long for preset \"short\" (max %.0fs)", durationSeconds, shortMax)
		}
		return nil
	case ClipPresetLong:
		if durationSeconds < longMin || durationSeconds > longMax {
			return fmt.Errorf("duration %.1fs is out of range for preset \"long\" (%.0f-%.0fs)", durationSeconds, longMin, longMax)
		}
		return nil
	default:
		return fmt.Errorf("unknown clip preset %q", preset)
	}
}
