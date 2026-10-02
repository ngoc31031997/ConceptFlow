package domain

import "testing"

func TestVideoOutputModeAcceptsOnlyLongAndShort(t *testing.T) {
	for _, m := range []VideoOutputMode{ModeLongOnly, ModeShortOnly} {
		if !m.IsValid() {
			t.Errorf("%q should be a valid output mode", m)
		}
	}
	for _, m := range []VideoOutputMode{"both", "", "vertical"} {
		if m.IsValid() {
			t.Errorf("%q should not be a valid output mode", m)
		}
	}
}
