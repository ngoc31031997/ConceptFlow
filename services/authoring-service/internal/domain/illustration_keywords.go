package domain

import (
	"fmt"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// illustrationKeywordGroup ties one component of the flat illustration kit
// (prompts/illustration_kit_vi.txt) to the Vietnamese words that name it.
//
// Spoken words trigger the check when they appear in a shot's narration, and
// count as the thing being shown when they appear in its visual. Shown words
// only count as shown: they are too common in speech to trigger on ("người",
// "bàn", "cây"), but a visual that says "một người mặc áo blouse" does show the
// doctor the narration talks about.
type illustrationKeywordGroup struct {
	Component string
	Spoken    []string
	Shown     []string
}

// illustrationKeywords is declared beside the kit rather than parsed out of it:
// the kit describes props in prose for the code model, not as a word list.
// illustration_keywords_test.go fails when a component is added to or removed
// from the kit without this table (or illustrationKitUnkeyed) following.
//
// Spoken words are kept narrow on purpose. A missed shot costs nothing — the
// Creator sees no warning, as before this check existed — while a warning that
// fires on ordinary speech teaches the Creator to ignore the whole list.
var illustrationKeywords = []illustrationKeywordGroup{
	{Component: "Person",
		Spoken: []string{"em bé", "cậu bé", "cô bé", "bé trai", "bé gái", "bác sĩ", "nha sĩ", "nhà khoa học", "học sinh", "cô giáo", "thầy giáo", "ông cụ", "bà cụ"},
		Shown:  []string{"người", "nhân vật", "bé", "cậu", "ông", "bà", "bố", "mẹ", "trẻ"}},
	{Component: "Tooth", Spoken: []string{"răng"}},
	{Component: "Germ", Spoken: []string{"vi khuẩn", "vi trùng", "mầm bệnh"}},
	{Component: "OpenMouth", Spoken: []string{"khoang miệng", "há miệng", "trong miệng"}, Shown: []string{"miệng"}},
	{Component: "Drop", Spoken: []string{"axit", "axít", "giọt", "nước bọt", "mồ hôi", "nước mắt"}},
	{Component: "Shield", Spoken: []string{"khiên", "lá chắn"}},
	{Component: "Heart", Spoken: []string{"tim"}},
	{Component: "Candy", Spoken: []string{"kẹo"}},
	{Component: "Lollipop", Spoken: []string{"kẹo mút"}},
	{Component: "Soda", Spoken: []string{"nước ngọt", "nước có ga", "soda"}},
	{Component: "Donut", Spoken: []string{"donut", "bánh vòng"}},
	{Component: "Apple", Spoken: []string{"quả táo", "trái táo"}, Shown: []string{"táo"}},
	{Component: "Toothbrush", Spoken: []string{"bàn chải"}},
	{Component: "Toothpaste", Spoken: []string{"kem đánh răng"}},
	{Component: "Clock", Spoken: []string{"đồng hồ"}},
	{Component: "Table", Spoken: []string{"cái bàn", "bàn ăn", "bàn học"}, Shown: []string{"bàn"}},
	{Component: "Chair", Spoken: []string{"ghế"}},
	{Component: "Window", Spoken: []string{"cửa sổ"}},
	{Component: "Plant", Spoken: []string{"chậu cây", "cây cảnh"}, Shown: []string{"cây"}},
	{Component: "House", Spoken: []string{"ngôi nhà", "căn nhà"}, Shown: []string{"nhà"}},
	{Component: "Tree", Spoken: []string{"cái cây", "cây xanh", "cây cổ thụ"}, Shown: []string{"cây"}},
	{Component: "Sun", Spoken: []string{"mặt trời"}},
	{Component: "Cloud", Spoken: []string{"mây"}},
	{Component: "Lightbulb", Spoken: []string{"bóng đèn"}, Shown: []string{"đèn"}},
	{Component: "Coin", Spoken: []string{"đồng xu", "tiền xu"}, Shown: []string{"xu"}},
	// Not bare "sách": "danh sách" (a list) is everyday speech.
	{Component: "Book", Spoken: []string{"quyển sách", "cuốn sách", "trang sách"}, Shown: []string{"sách"}},
	{Component: "Phone", Spoken: []string{"điện thoại"}},
	{Component: "Magnifier", Spoken: []string{"kính lúp"}},
	{Component: "Airplane", Spoken: []string{"máy bay"}},
}

// illustrationKitUnkeyed lists the kit components the check deliberately has
// no words for, with the reason. Together with illustrationKeywords it must
// cover the kit exactly.
var illustrationKitUnkeyed = map[string]string{
	"Backdrop": "nền của cảnh, không phải vật lời thoại nói tới",
	"Panel":    "mảng màu bố cục",
	"Bubble":   "bong bóng thoại chứa nhãn, không phải vật",
	"Mark":     "dấu đúng/sai là ký hiệu, lời thoại không gọi tên nó",
	"Sparkle":  "tia lấp lánh trang trí",
	// Places are the director's scene setting, not a thing the narration names.
	"MeadowBackdrop":     "nền của cảnh",
	"RoomBackdrop":       "nền của cảnh",
	"StreetBackdrop":     "nền của cảnh",
	"InsideBodyBackdrop": "nền của cảnh",
	"UnderwaterBackdrop": "nền của cảnh",
	"SpaceBackdrop":      "nền của cảnh",
}

// CheckIllustratedNarration flags shots whose narration names a thing the
// illustration kit can draw while the shot's visual never mentions it — the
// "mute the video and look" rule (visual director rule 19) caught cheaply
// after step 1b. Warnings only; it is a word match, not an
// understanding of the shot.
//
// One warning per shot and component. When two missing keywords overlap
// ("kẹo mút" also contains "kẹo"), only the longer is reported.
func CheckIllustratedNarration(scenes []StoryboardScene) []string {
	var warnings []string
	for _, scene := range scenes {
		for _, shot := range scene.Shots {
			narration := keywordTokens(shot.Narration)
			visual := keywordTokens(shot.Visual)
			var missing [][]string
			for _, group := range illustrationKeywords {
				said := longestMatch(narration, group.Spoken)
				if said == nil {
					continue
				}
				if longestMatch(visual, group.Spoken) != nil || longestMatch(visual, group.Shown) != nil {
					continue
				}
				missing = append(missing, said)
			}
			for i, kw := range missing {
				if coveredByLonger(kw, missing, i) {
					continue
				}
				id := strings.TrimSpace(shot.ID)
				if id == "" {
					id = "?"
				}
				warnings = append(warnings, fmt.Sprintf(
					"Shot %s: lời thoại nhắc '%s' nhưng HÌNH không có", id, strings.Join(kw, " ")))
			}
		}
	}
	return warnings
}

// keywordTokens lower-cases, NFC-normalises and splits text into words, so
// "răng" matches "Răng," but not the inside of "trăng", and a decomposed
// "vi khuẩn" matches a precomposed one.
func keywordTokens(text string) []string {
	text = strings.ToLower(norm.NFC.String(text))
	return strings.FieldsFunc(text, func(r rune) bool {
		return !(unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsMark(r))
	})
}

// longestMatch returns the tokens of the longest keyword found in words, or
// nil when none is.
func longestMatch(words []string, keywords []string) []string {
	var best []string
	for _, kw := range keywords {
		tokens := keywordTokens(kw)
		if len(tokens) > len(best) && containsRun(words, tokens) {
			best = tokens
		}
	}
	return best
}

// containsRun reports whether needle occurs in haystack as consecutive words.
func containsRun(haystack, needle []string) bool {
	if len(needle) == 0 {
		return false
	}
outer:
	for i := 0; i+len(needle) <= len(haystack); i++ {
		for j, w := range needle {
			if haystack[i+j] != w {
				continue outer
			}
		}
		return true
	}
	return false
}

func coveredByLonger(kw []string, all [][]string, self int) bool {
	for i, other := range all {
		if i != self && len(other) > len(kw) && containsRun(other, kw) {
			return true
		}
	}
	return false
}
