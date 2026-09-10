package domain

import "testing"

func occurrences(pairs ...interface{}) []BeatOccurrence {
	out := make([]BeatOccurrence, 0, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		out = append(out, BeatOccurrence{SceneIndex: pairs[i].(int), ID: pairs[i+1].(string)})
	}
	return out
}

func TestValidateBeats_AcceptsAWellFormedScript(t *testing.T) {
	issues := FormatVisualFirst7Min.ValidateBeats(occurrences(
		0, "hook", 1, "concrete", 4, "pattern", 8, "variation", 12, "recap", 15, "cta",
	))
	if len(issues) != 0 {
		t.Fatalf("không nên có vấn đề gì: %+v", issues)
	}
}

func TestValidateBeats_BlocksOnMissingRequiredBeat(t *testing.T) {
	// Beat bắt buộc thiếu là một dữ kiện chắc chắn, không phải ước lượng — nên
	// chặn được (FR52.5).
	issues := FormatVisualFirst7Min.ValidateBeats(occurrences(
		0, "hook", 1, "pattern", 5, "recap", 8, "cta",
	))
	blocking := BlockingBeatIssues(issues)
	if len(blocking) != 1 || blocking[0].BeatID != "concrete" {
		t.Fatalf("phải chặn vì thiếu `concrete`: %+v", issues)
	}
}

func TestValidateBeats_WarnsOnUnknownBeat(t *testing.T) {
	issues := FormatVisualFirst7Min.ValidateBeats(occurrences(
		0, "hook", 1, "concrete", 4, "pattern", 8, "lan-man", 12, "recap", 15, "cta",
	))
	if len(BlockingBeatIssues(issues)) != 0 {
		t.Fatal("beat lạ chỉ nên cảnh báo")
	}
	if len(issues) == 0 {
		t.Fatal("beat lạ phải được nêu ra")
	}
}

func TestValidateBeats_WarnsWhenABeatRepeatsTooOften(t *testing.T) {
	issues := FormatVisualFirst7Min.ValidateBeats(occurrences(
		0, "hook", 1, "concrete", 4, "pattern",
		6, "variation", 8, "variation", 10, "variation",
		12, "recap", 15, "cta",
	))
	found := false
	for _, issue := range issues {
		if issue.BeatID == "variation" {
			found = true
			if issue.Blocking {
				t.Fatal("lặp quá số lần chỉ nên cảnh báo")
			}
		}
	}
	if !found {
		t.Fatalf("phải nêu variation lặp quá 2 lần: %+v", issues)
	}
}

func TestValidateBeats_WarnsWhenPatternComesBeforeConcrete(t *testing.T) {
	// Đây là ràng buộc mang toàn bộ ý nghĩa của format: ví dụ cụ thể phải chạy
	// TRƯỚC định nghĩa. Không có nó, "ví dụ trực quan thay vì text đơn điệu"
	// quay lại thành lời khuyên trong prompt.
	issues := FormatVisualFirst7Min.ValidateBeats(occurrences(
		0, "hook", 1, "pattern", 5, "concrete", 9, "recap", 12, "cta",
	))
	found := false
	for _, issue := range issues {
		if issue.BeatID == "concrete" {
			found = true
		}
	}
	if !found {
		t.Fatalf("phải nêu sai thứ tự: %+v", issues)
	}
}

func TestMeasureBeatBudgets_AttributesDurationsToTheOpenBeat(t *testing.T) {
	format := VideoFormat{
		ID: "t", Name: "t", Beats: []FormatBeat{
			{ID: "hook", MinSeconds: 5, MaxSeconds: 15, MaxRepeat: 1},
			{ID: "body", MinSeconds: 20, MaxSeconds: 60, MaxRepeat: 1},
		},
	}
	budgets := format.MeasureBeatBudgets(
		occurrences(0, "hook", 2, "body"),
		[]float64{4, 4, 10, 10, 10},
	)
	if len(budgets) != 2 {
		t.Fatalf("muốn 2 beat, có %d", len(budgets))
	}
	if budgets[0].Seconds != 8 || budgets[0].Status != "ok" {
		t.Errorf("hook: %+v", budgets[0])
	}
	if budgets[1].Seconds != 30 || budgets[1].Status != "ok" {
		t.Errorf("body: %+v", budgets[1])
	}
}

func TestMeasureBeatBudgets_FlagsShortAndLong(t *testing.T) {
	format := VideoFormat{ID: "t", Beats: []FormatBeat{{ID: "b", MinSeconds: 20, MaxSeconds: 30, MaxRepeat: 1}}}

	short := format.MeasureBeatBudgets(occurrences(0, "b"), []float64{5})
	if short[0].Status != "short" {
		t.Errorf("muốn short, có %q", short[0].Status)
	}
	long := format.MeasureBeatBudgets(occurrences(0, "b"), []float64{100})
	if long[0].Status != "long" {
		t.Errorf("muốn long, có %q", long[0].Status)
	}
}

func TestMeasureBeatBudgets_ScalesBudgetForRepeatableBeats(t *testing.T) {
	// Hai đoạn `variation` hợp lệ không được đọc thành vượt ngân sách.
	budgets := FormatVisualFirst7Min.MeasureBeatBudgets(
		occurrences(0, "variation", 1, "variation"),
		[]float64{85, 85},
	)
	if budgets[0].Status != "ok" {
		t.Fatalf("hai lần variation trong hạn phải là ok: %+v", budgets[0])
	}
}

func TestValidateBeats_ScriptWithNoBeatsIsWarnedNotBlocked(t *testing.T) {
	// Chặn mọi script chưa khai báo beat sẽ phá hỏng cả những video đơn giản
	// nhất và mọi script viết trước khi beat tồn tại — cách nhanh nhất để
	// Creator ghét cơ chế này thay vì dùng nó.
	issues := FormatVisualFirst7Min.ValidateBeats(nil)
	if len(BlockingBeatIssues(issues)) != 0 {
		t.Fatalf("không được chặn: %+v", issues)
	}
	if len(issues) != 1 {
		t.Fatalf("phải có đúng một cảnh báo: %+v", issues)
	}
}

func TestValidateBeats_OptingInIsAllOrNothing(t *testing.T) {
	// Khai báo một beat là đã chọn dùng format, nên lỗ hổng ở trên không dùng
	// được để né một beat bắt buộc phiền phức.
	issues := FormatVisualFirst7Min.ValidateBeats(occurrences(0, "hook"))
	if len(BlockingBeatIssues(issues)) == 0 {
		t.Fatal("khai báo beat rồi thì thiếu beat bắt buộc phải bị chặn")
	}
}
