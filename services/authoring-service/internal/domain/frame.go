package domain

import "fmt"

// SafeArea is the rectangle anything meaningful stays inside, in px from the
// frame's top-left corner.
type SafeArea struct {
	Left, Top, Right, Bottom int
}

// Frame is the canvas a video is built on: its size and its safe area.
type Frame struct {
	Width, Height int
	Safe          SafeArea
}

// LandscapeFrame is the long-form 16:9 frame; PortraitFrame is the vertical
// Shorts frame, whose safe area keeps clear of the Shorts overlay (title and
// channel at the bottom, the like/comment column on the right). The same
// numbers live in rendering's remotion_project/src/conceptflow-mini/primitives.tsx
// (FRAMES), rendering/domain/frame.py and llm-service/app/frame.py; frame_test.go
// compares against the TypeScript file. See ADR-0031.
var (
	LandscapeFrame = Frame{Width: 1920, Height: 1080, Safe: SafeArea{Left: 96, Top: 96, Right: 1824, Bottom: 984}}
	PortraitFrame  = Frame{Width: 1080, Height: 1920, Safe: SafeArea{Left: 72, Top: 200, Right: 940, Bottom: 1560}}
)

// FrameFor is the frame a project is built on: a short is produced vertically
// from the start, everything else in 16:9.
func FrameFor(mode VideoOutputMode) Frame {
	if mode == ModeShortOnly {
		return PortraitFrame
	}
	return LandscapeFrame
}

// Portrait reports whether f is taller than wide.
func (f Frame) Portrait() bool { return f.Height > f.Width }

// FramePromptVars are the prompt variables that describe the frame:
// {{frame}}, {{frame_width}}, {{frame_height}}, {{safe_area}} and
// {{frame_rules}} (empty for the landscape frame, the vertical-composition
// rules for the portrait one).
func FramePromptVars(f Frame, language string) map[string]string {
	vars := map[string]string{
		"frame_width":  fmt.Sprint(f.Width),
		"frame_height": fmt.Sprint(f.Height),
		"safe_area":    fmt.Sprintf("(%d, %d) đến (%d, %d)", f.Safe.Left, f.Safe.Top, f.Safe.Right, f.Safe.Bottom),
		"frame":        fmt.Sprintf("%d×%d (ngang)", f.Width, f.Height),
		"frame_rules":  "",
	}
	if language != "vi" {
		vars["safe_area"] = fmt.Sprintf("(%d, %d) to (%d, %d)", f.Safe.Left, f.Safe.Top, f.Safe.Right, f.Safe.Bottom)
		vars["frame"] = fmt.Sprintf("%d×%d (landscape)", f.Width, f.Height)
	}
	if f.Portrait() {
		vars["frame"] = fmt.Sprintf("%d×%d (dọc, cho Shorts/TikTok)", f.Width, f.Height)
		vars["frame_rules"] = portraitRulesVI
		if language != "vi" {
			vars["frame"] = fmt.Sprintf("%d×%d (portrait, for Shorts/TikTok)", f.Width, f.Height)
			vars["frame_rules"] = portraitRulesEN
		}
	}
	return vars
}

const portraitRulesVI = `## KHUNG DỌC (SHORT 9:16)

Video này là một short dọc 1080×1920, xem trên điện thoại, dài 30–60 giây.
- Bố cục theo chiều dọc: chồng các lớp và các vật từ trên xuống (trời ở trên, nhân vật ở giữa, tiền cảnh ở dưới), không xếp ngang hàng nhiều vật. Nhân vật chính rộng ít nhất 60% bề ngang khung.
- Vùng an toàn dọc chừa chỗ cho giao diện Shorts: 200 px trên cùng, 360 px dưới cùng (tiêu đề, tên kênh), 140 px mép phải (cột nút thích/bình luận). Không đặt chữ hay vật quan trọng ở đó.
- Shot đầu tiên chuyển động ngay từ frame 0 và đặt câu hỏi gây tò mò trong 2 giây đầu.
- Chữ trên hình là TỪ KHOÁ: tối đa 3 từ, chữ to, bật lên trên một mảng màu, đúng lúc lời thoại gọi tên khái niệm đó. Không có phụ đề.
- Nhịp nhanh hơn video dài: khoảng mỗi 2–4 giây một thay đổi hình.`

const portraitRulesEN = `## PORTRAIT FRAME (SHORT 9:16)

This video is a 1080×1920 vertical short, watched on a phone, 30–60 seconds long.
- Compose vertically: stack layers and objects from top to bottom (sky above, the character in the middle, foreground below); do not line many objects up side by side. The main character spans at least 60% of the frame's width.
- The portrait safe area keeps clear of the Shorts overlay: the top 200 px, the bottom 360 px (title, channel name), the right 140 px (like/comment buttons). No text or important object there.
- The first shot moves from frame 0 and raises a question within the first 2 seconds.
- On-screen text is a KEYWORD: at most 3 words, large, popping in on a coloured block exactly when the narration names that concept. No subtitles.
- Faster pace than a long video: a meaningful visual change about every 2–4 seconds.`
