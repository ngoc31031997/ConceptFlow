package domain

import (
	_ "embed"
	"fmt"
	"math"
	"strings"
)

// CR-027 FR77: these prompt variables used to be filled in by the browser.
// scriptPrompts.ts held the constants and substituted them into the template
// it had fetched, which left the server unable to render a prompt at all —
// exactly what FR78's generate endpoints have to do.
//
// Moving them here is not tidying up. Two copies would let the
// copy-the-prompt-out flow and the run-it-here flow drift apart on the same
// role, and nothing in either output would say so.
//
// The constant blocks are EMBEDDED as the exact files the shipping
// TypeScript produced, rather than retyped into Go string literals. A block
// of Vietnamese prose containing backticks and quotes is precisely the kind
// of thing that loses a character in translation, and embedding removes that
// risk by construction instead of asking a test to catch it afterwards.

//go:embed prompts/channel_identity_vi.txt
var channelIdentityVI string

//go:embed prompts/channel_identity_en.txt
var channelIdentityEN string

//go:embed prompts/narration_rule_vi.txt
var narrationRuleVI string

//go:embed prompts/narration_rule_en.txt
var narrationRuleEN string

// wordsPerMinute is the fallback speaking rate per content language, used
// when the project's voice has no measured calibration of its own (CR-016).
var wordsPerMinute = map[string]float64{
	"vi": 140,
	"en": 150,
}

// ChannelIdentity returns the block {{channel_identity}} expands to: what
// THIS channel is, in place of the older "make it like 3Blue1Brown".
//
// Falls back to English for an unknown language rather than returning an
// empty section — a prompt missing its identity block still looks valid and
// produces a generic video, which is the worst way to fail here.
func ChannelIdentity(language string) string {
	if language == "vi" {
		return channelIdentityVI
	}
	return channelIdentityEN
}

// NarrationLanguageRule returns {{narration_language_rule}} — which language
// the spoken lines must be written in (CR-008 FR21.3).
func NarrationLanguageRule(language string) string {
	if language == "vi" {
		return narrationRuleVI
	}
	return narrationRuleEN
}

// BuildStoryBeatSheetSection renders {{format_beats}}: the beats of the
// chosen format with their budgets converted from seconds into words, at the
// speaking rate of the voice this project will actually use.
//
// Words rather than seconds because the Story Architect step writes prose,
// and a budget it cannot apply while writing is not a budget (CR-016/CR-019).
//
// calibratedWPM is the measured rate for the project's voice, or 0 to fall
// back to the language default.
//
// Unlike the constants above this is real logic — ordering, rounding, the
// required/optional wording — so it is a port, and prompt_vars_test.go holds
// it to golden files generated from the TypeScript it replaces.
func BuildStoryBeatSheetSection(format VideoFormat, language string, calibratedWPM float64) string {
	wpm := calibratedWPM
	if wpm == 0 {
		var ok bool
		if wpm, ok = wordsPerMinute[language]; !ok {
			wpm = wordsPerMinute["en"]
		}
	}
	// JavaScript's Math.round is half-up; Go's math.Round is
	// half-away-from-zero. Budgets are always positive, so the two agree —
	// the golden files are what prove it rather than this comment.
	words := func(seconds float64) int {
		return int(math.Round((seconds * wpm) / 60))
	}

	rows := make([]string, 0, len(format.Beats))
	total := 0
	for _, beat := range format.Beats {
		repeat := ""
		if beat.MaxRepeat > 1 {
			repeat = fmt.Sprintf(" (lặp tối đa %d lần)", beat.MaxRepeat)
		}
		required := "tuỳ chọn"
		if beat.Required {
			required = "BẮT BUỘC"
			total += words(beat.MinSeconds)
		}
		rows = append(rows, fmt.Sprintf("   - id `%s` — %s%s: khoảng %d–%d từ lời thoại",
			beat.ID, required, repeat, words(beat.MinSeconds), words(beat.MaxSeconds)))
	}

	return fmt.Sprintf(beatSheetTemplate, format.Name, strings.Join(rows, "\n"), total)
}

// beatSheetTemplate is kept beside the function rather than inlined so the
// backticks inside it (which name `concrete` and `pattern` as code) do not
// have to fight Go's raw-string syntax.
const beatSheetTemplate = "## CẤU TRÚC BẮT BUỘC — format \"%s\"\n" +
	"\n" +
	"Dàn ý phải gồm ĐÚNG các beat dưới đây, theo ĐÚNG thứ tự này, và mỗi beat phải\n" +
	"được đặt tên bằng ĐÚNG id trong danh sách (đừng tự đặt tên beat mới):\n" +
	"\n" +
	"%s\n" +
	"\n" +
	"Quy tắc cứng:\n" +
	"   - Thiếu một beat BẮT BUỘC thì dàn ý bị trả lại — các bước sau không dựng được.\n" +
	"   - `concrete` phải đứng TRƯỚC `pattern`: cho người xem thấy một ví dụ chạy\n" +
	"     thật rồi mới rút ra quy luật. Đây là bản sắc của kênh, không phải sở thích\n" +
	"     trình bày.\n" +
	"   - Ngân sách từ là để canh nhịp — lệch trong khoảng ±15%% thì không sao; thứ tự\n" +
	"     và các beat bắt buộc thì không được lệch.\n" +
	"   - Cuối mỗi beat, ghi số từ thực tế của lời thoại nháp bạn vừa viết, để tự\n" +
	"     kiểm. Tổng các beat bắt buộc rơi vào khoảng %d từ trở lên.\n"
