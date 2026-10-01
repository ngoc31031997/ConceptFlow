package domain

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
)

const primitivesTSX = "../../../rendering/remotion_project/src/conceptflow-mini/primitives.tsx"

func TestFramesMatchTheTypeScriptKit(t *testing.T) {
	raw, err := os.ReadFile(primitivesTSX)
	if err != nil {
		t.Skipf("rendering is not in this tree: %v", err)
	}
	for name, f := range map[string]Frame{"landscape": LandscapeFrame, "portrait": PortraitFrame} {
		re := regexp.MustCompile(name + `: \{width: (\d+), height: (\d+), safe: \{left: (\d+), top: (\d+), right: (\d+), bottom: (\d+)\}\}`)
		m := re.FindStringSubmatch(string(raw))
		if m == nil {
			t.Fatalf("primitives.tsx has no %s frame", name)
		}
		want := fmt.Sprintf("%d %d %d %d %d %d", f.Width, f.Height, f.Safe.Left, f.Safe.Top, f.Safe.Right, f.Safe.Bottom)
		if got := strings.Join(m[1:], " "); got != want {
			t.Errorf("%s: primitives.tsx has %s, Go has %s", name, got, want)
		}
	}
}

func TestAShortIsBuiltOnThePortraitFrame(t *testing.T) {
	if FrameFor(ModeShortOnly) != PortraitFrame || FrameFor(ModeLongOnly) != LandscapeFrame || FrameFor("") != LandscapeFrame {
		t.Fatal("short must be portrait, everything else landscape")
	}
	long := FramePromptVars(LandscapeFrame, "vi")
	if long["frame"] != "1920×1080 (ngang)" || long["safe_area"] != "(96, 96) đến (1824, 984)" || long["frame_rules"] != "" {
		t.Errorf("landscape vars = %v", long)
	}
	short := FramePromptVars(PortraitFrame, "vi")
	if short["frame_width"] != "1080" || short["frame_height"] != "1920" || !strings.Contains(short["frame_rules"], "KHUNG DỌC") {
		t.Errorf("portrait vars = %v", short)
	}
}

func TestBurnedSubtitlesSitAtTheBottomOfTheRealFrame(t *testing.T) {
	style := SubtitleStyle{FontSize: "medium", Position: "bottom"}
	if got := SubtitleZoneIn(SubtitleModeBurnIn, style, PortraitFrame, "vi"); !strings.Contains(got, "dải y từ 1680 đến 1920 px") {
		t.Errorf("portrait zone = %q", got)
	}
	if SubtitleZoneIn(SubtitleModeBurnIn, style, LandscapeFrame, "vi") != SubtitleZoneFor(SubtitleModeBurnIn, style, "vi") {
		t.Error("SubtitleZoneFor must stay the landscape zone")
	}
}
