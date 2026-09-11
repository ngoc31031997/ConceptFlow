package domain

import "testing"

func TestValidateClipDuration_ShortPreset(t *testing.T) {
	cases := []struct {
		name     string
		duration float64
		wantErr  bool
	}{
		{"well within short", 30, false},
		{"at the boundary", 60, false},
		{"just over the boundary", 60.1, true},
		{"too long (75s from the LLD example)", 75, true},
		{"zero is not a valid clip", 0, true},
		{"negative is not a valid clip", -5, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ValidateClipDuration(c.duration, ClipPresetShort)
			if (err != nil) != c.wantErr {
				t.Fatalf("ValidateClipDuration(%v, short) error = %v, wantErr %v", c.duration, err, c.wantErr)
			}
		})
	}
}

func TestValidateClipDuration_LongPreset(t *testing.T) {
	cases := []struct {
		name     string
		duration float64
		wantErr  bool
	}{
		{"75s from the LLD example fits long", 75, false},
		{"at the lower boundary", 60, false},
		{"at the upper boundary", 180, false},
		{"just under the lower boundary", 59.9, true},
		{"just over the upper boundary", 180.1, true},
		{"far too short", 10, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ValidateClipDuration(c.duration, ClipPresetLong)
			if (err != nil) != c.wantErr {
				t.Fatalf("ValidateClipDuration(%v, long) error = %v, wantErr %v", c.duration, err, c.wantErr)
			}
		})
	}
}

// TestValidateClipDuration_75sBothPresets locks FR19.6/19.7's own example:
// a 75s segment is rejected for "short" with a clear reason and accepted for
// "long" — one preset's rejection never disqualifies the other.
func TestValidateClipDuration_75sBothPresets(t *testing.T) {
	if err := ValidateClipDuration(75, ClipPresetShort); err == nil {
		t.Fatalf("expected 75s to be rejected for preset short")
	}
	if err := ValidateClipDuration(75, ClipPresetLong); err != nil {
		t.Fatalf("expected 75s to be accepted for preset long, got %v", err)
	}
}

func TestValidateClipDuration_UnknownPreset(t *testing.T) {
	if err := ValidateClipDuration(30, "vertical"); err == nil {
		t.Fatalf("expected an error for an unrecognised preset")
	}
}

func TestValidateClipDuration_ThresholdsFromEnv(t *testing.T) {
	t.Setenv("CLIP_PRESET_SHORT_MAX_SECONDS", "30")
	if err := ValidateClipDuration(45, ClipPresetShort); err == nil {
		t.Fatalf("expected 45s to be rejected once CLIP_PRESET_SHORT_MAX_SECONDS=30")
	}
	if err := ValidateClipDuration(20, ClipPresetShort); err != nil {
		t.Fatalf("expected 20s to still fit under the lowered threshold, got %v", err)
	}
}
