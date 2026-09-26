package domain

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// TestEstimateNarrationDuration_MatchesSharedVectors khoá bản Go với bản
// TypeScript trong web-gui (CR-016 FR42.2).
//
// Cùng một con số được dùng ở hai nơi: ở GUI để Creator thấy trước lúc soạn, và
// ở đây để định thời lượng thật khi tắt TTS (CR-001). Nếu hai bản trôi khỏi
// nhau thì con số hiển thị lúc soạn khác con số hệ thống thực sự dùng — loại
// sai lệch không ai truy ra được từ triệu chứng.
func TestEstimateNarrationDuration_MatchesSharedVectors(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "tests", "fixtures", "narration-duration-vectors.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("không đọc được bộ vector dùng chung: %v", err)
	}

	var fixture struct {
		WordsPerMinute      map[string]float64 `json:"words_per_minute"`
		MinNarrationSeconds float64            `json:"min_narration_seconds"`
		Vectors             []struct {
			Text            string  `json:"text"`
			Language        string  `json:"language"`
			ExpectedSeconds float64 `json:"expected_seconds"`
			Why             string  `json:"why"`
		} `json:"vectors"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("bộ vector hỏng: %v", err)
	}
	if len(fixture.Vectors) == 0 {
		t.Fatal("bộ vector rỗng — test này sẽ không khoá được gì")
	}

	if got := ProfileFor(LanguageVietnamese).WordsPerMinute; got != fixture.WordsPerMinute["vi"] {
		t.Errorf("WPM tiếng Việt: Go có %v, bộ vector nói %v", got, fixture.WordsPerMinute["vi"])
	}
	if got := ProfileFor(LanguageEnglish).WordsPerMinute; got != fixture.WordsPerMinute["en"] {
		t.Errorf("WPM tiếng Anh: Go có %v, bộ vector nói %v", got, fixture.WordsPerMinute["en"])
	}

	for _, v := range fixture.Vectors {
		got := EstimateNarrationDuration(v.Text, ContentLanguage(v.Language))
		if math.Abs(got-v.ExpectedSeconds) > 1e-6 {
			t.Errorf("%s: EstimateNarrationDuration(%q, %s) = %v, muốn %v",
				v.Why, v.Text, v.Language, got, v.ExpectedSeconds)
		}
	}
}
