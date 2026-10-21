package postgres

import (
	"context"
	"testing"
	"time"

	"authoring/internal/domain"
)

func voiceLine(id, text, voice string) domain.VoiceLine {
	return domain.VoiceLine{ProjectID: "vl-test", LineID: id, Text: text, TextSHA: domain.LineTextSHA(text), VoiceID: voice, Language: "vi"}
}

func TestVoiceLinesAreClaimedOnceAndAnsweredOnlyForTheWordingSent(t *testing.T) {
	r, ctx := researchRepo(t, "vl-test")
	if err := r.RequestVoiceLine(ctx, voiceLine("b1.1", "Câu một.", "v")); err != nil {
		t.Fatal(err)
	}
	if err := r.RequestVoiceLine(ctx, voiceLine("b1.2", "Câu hai.", "v")); err != nil {
		t.Fatal(err)
	}
	claimed, err := r.ClaimVoiceLines(ctx, "vl-test", time.Minute)
	if err != nil || len(claimed) != 2 {
		t.Fatalf("claimed %d lines, err %v; want both waiting lines", len(claimed), err)
	}
	if again, _ := r.ClaimVoiceLines(ctx, "vl-test", time.Minute); len(again) != 0 {
		t.Errorf("a second claim returned %d lines; lines already sent must not be sent twice", len(again))
	}
	if retry, _ := r.ClaimVoiceLines(ctx, "vl-test", 0); len(retry) != 2 {
		t.Errorf("lines sent long ago without an answer must be claimable again, got %d", len(retry))
	}

	// The Creator edits b1.2 while it is being read: the old answer must be dropped.
	if err := r.RequestVoiceLine(ctx, voiceLine("b1.2", "Câu hai đã sửa.", "v")); err != nil {
		t.Fatal(err)
	}
	err = r.SaveVoiceResults(ctx, "vl-test", []domain.VoiceLineResult{
		{LineID: "b1.1", Ref: domain.LineTextSHA("Câu một."), AudioPath: "/shared/vl-test/audio/b1-1_a.wav", DurationSeconds: 3.5, CacheKey: "a", CacheHit: true},
		{LineID: "b1.2", Ref: domain.LineTextSHA("Câu hai."), AudioPath: "/shared/vl-test/audio/b1-2_old.wav", DurationSeconds: 2, CacheKey: "old"},
	})
	if err != nil {
		t.Fatal(err)
	}
	rows, err := r.ListVoiceLines(ctx, "vl-test")
	if err != nil || len(rows) != 2 {
		t.Fatalf("rows %d err %v", len(rows), err)
	}
	if rows[0].Status != domain.VoiceLineReady || rows[0].DurationSeconds != 3.5 || !rows[0].CacheHit || rows[0].AudioPath == "" {
		t.Errorf("b1.1 = %+v, want ready with its file", rows[0])
	}
	if rows[1].Status != domain.VoiceLinePending || rows[1].AudioPath != "" || rows[1].Text != "Câu hai đã sửa." {
		t.Errorf("b1.2 = %+v: the answer for older wording must not attach to the edited line", rows[1])
	}
}

func TestVoiceLineFailuresCarryTheirReason(t *testing.T) {
	r, ctx := researchRepo(t, "vl-test")
	if err := r.RequestVoiceLine(ctx, voiceLine("b1.1", "Câu.", "")); err != nil {
		t.Fatal(err)
	}
	err := r.SaveVoiceResults(ctx, "vl-test", []domain.VoiceLineResult{{LineID: "b1.1", Ref: domain.LineTextSHA("Câu."), Error: "engine hỏng"}})
	if err != nil {
		t.Fatal(err)
	}
	rows, _ := r.ListVoiceLines(ctx, "vl-test")
	if rows[0].Status != domain.VoiceLineFailed || rows[0].Error != "engine hỏng" {
		t.Errorf("row = %+v", rows[0])
	}
	if err := r.RequestVoiceLine(ctx, voiceLine("b1.1", "Câu.", "")); err != nil {
		t.Fatal(err)
	}
	if rows, _ = r.ListVoiceLines(ctx, "vl-test"); rows[0].Status != domain.VoiceLinePending || rows[0].Error != "" {
		t.Errorf("asking again must clear the failure: %+v", rows[0])
	}
	if err := r.FailVoiceLines(ctx, "vl-test", []string{"b1.1"}, "không gửi được"); err != nil {
		t.Fatal(err)
	}
	if rows, _ = r.ListVoiceLines(ctx, "vl-test"); rows[0].Status != domain.VoiceLineFailed || rows[0].Error != "không gửi được" {
		t.Errorf("row = %+v", rows[0])
	}
}

func TestDeletingAProjectsAuthoringRemovesItsVoiceLines(t *testing.T) {
	r, ctx := researchRepo(t, "vl-test")
	if err := r.RequestVoiceLine(ctx, voiceLine("b1.1", "Câu.", "")); err != nil {
		t.Fatal(err)
	}
	if err := r.DeleteAuthoring(ctx, "vl-test"); err != nil {
		t.Fatal(err)
	}
	if rows, err := r.ListVoiceLines(context.Background(), "vl-test"); err != nil || len(rows) != 0 {
		t.Errorf("rows %d err %v, want none after the project is deleted", len(rows), err)
	}
}
