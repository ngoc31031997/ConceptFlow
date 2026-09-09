package domain

import (
	"math"
	"testing"
)

func TestVoiceCalibration_NotTrustedBelowThreshold(t *testing.T) {
	// Một project có thể không đại diện (toàn câu ngắn, hoặc một đoạn dày công
	// thức); ba project thì hiếm khi vậy.
	c := VoiceCalibration{VoiceID: "v1", SampleCount: MinCalibrationSamples - 1, TotalWords: 900, TotalSecond: 360}
	if _, ok := c.WordsPerMinute(); ok {
		t.Fatal("chưa đủ mẫu mà đã tin số đo")
	}
}

func TestVoiceCalibration_ComputesMeasuredRate(t *testing.T) {
	c := VoiceCalibration{VoiceID: "v1", SampleCount: 3, TotalWords: 900, TotalSecond: 360}
	got, ok := c.WordsPerMinute()
	if !ok {
		t.Fatal("đủ mẫu mà không tin số đo")
	}
	if math.Abs(got-150) > 1e-9 {
		t.Fatalf("WPM = %v, muốn 150", got)
	}
}

func TestVoiceCalibration_RejectsDegenerateTotals(t *testing.T) {
	// Trả về ok=false chứ không phải 0: một WPM bằng 0 sẽ chia cho 0 và làm
	// đứng mọi khoảng chờ trong script.
	for _, c := range []VoiceCalibration{
		{SampleCount: 5, TotalWords: 100, TotalSecond: 0},
		{SampleCount: 5, TotalWords: 0, TotalSecond: 100},
	} {
		if _, ok := c.WordsPerMinute(); ok {
			t.Fatalf("%+v không nên được tin", c)
		}
	}
}

func TestEstimateNarrationDurationCalibrated_FallsBackWithoutEvidence(t *testing.T) {
	text := "một hai ba bốn năm sáu bảy tám chín mười"
	got := EstimateNarrationDurationCalibrated(text, LanguageVietnamese, VoiceCalibration{})
	want := EstimateNarrationDuration(text, LanguageVietnamese)
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("không có số đo thì phải rơi về hằng số: %v != %v", got, want)
	}
}

func TestEstimateNarrationDurationCalibrated_UsesMeasuredRate(t *testing.T) {
	text := "một hai ba bốn năm sáu bảy tám chín mười"
	fast := VoiceCalibration{SampleCount: 3, TotalWords: 1000, TotalSecond: 200} // 300 wpm
	got := EstimateNarrationDurationCalibrated(text, LanguageVietnamese, fast)
	if got >= EstimateNarrationDuration(text, LanguageVietnamese) {
		t.Fatal("giọng đọc nhanh hơn phải cho thời lượng ngắn hơn")
	}
}

func TestEstimateNarrationDurationCalibrated_KeepsTheFloor(t *testing.T) {
	// Câu hai từ đọc bằng giọng nhanh sẽ vụt qua trước khi kịp đọc phụ đề.
	fast := VoiceCalibration{SampleCount: 3, TotalWords: 1000, TotalSecond: 100}
	if got := EstimateNarrationDurationCalibrated("hai từ", LanguageVietnamese, fast); got != minNarrationSeconds {
		t.Fatalf("got %v, muốn sàn %v", got, minNarrationSeconds)
	}
}
