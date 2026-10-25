package domain

import "testing"

func TestTargetLengthIsATenthEitherSideOfTheBriefSeconds(t *testing.T) {
	got := NewTargetLength(270)
	if got.Seconds != 270 || got.Lo != 243 || got.Hi != 297 {
		t.Errorf("NewTargetLength(270) = %+v, want 270 in 243–297", got)
	}
	if label := got.Label("vi"); label != "4:30 phút (khoảng 4:03–4:57)" {
		t.Errorf("vi label = %q", label)
	}
	if label := got.Label("en"); label != "4:30 minutes (about 4:03–4:57)" {
		t.Errorf("en label = %q", label)
	}
	if label := (TargetLength{}).Label("vi"); label != "" {
		t.Errorf("no target must have no label, got %q", label)
	}
}
