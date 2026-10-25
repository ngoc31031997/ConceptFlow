package application

import (
	"context"
	"errors"
	"strings"
	"testing"

	"authoring/internal/domain"
)

func TestStateShowsTheInputTheAIGets(t *testing.T) {
	f := newBeatFixture()
	f.open(t)
	state, err := f.uc.State(context.Background(), "p1")
	if err != nil {
		t.Fatal(err)
	}
	in := state.Input
	if in.Topic != "Vì sao trong quả sung có xác ong?" || !in.BriefApproved {
		t.Fatalf("input = %+v", in)
	}
	for _, want := range []string{"Người tò mò", "Gần gũi", "Hiểu vì sao"} {
		if !strings.Contains(in.Brief, want) {
			t.Errorf("brief %q lacks %q", in.Brief, want)
		}
	}
	for _, want := range []string{"question (", "Vì sao ong chết trong quả sung?", "chain (", "payoff (", "Hai loài dựa nhau."} {
		if !strings.Contains(in.Story, want) {
			t.Errorf("story %q lacks %q", in.Story, want)
		}
	}
	if len(state.Facts) != 3 {
		t.Errorf("facts = %d, want the 3 kept ones", len(state.Facts))
	}
}

func TestStateInputIsEmptyBeforeTheStoryIsApproved(t *testing.T) {
	f := newBeatFixture()
	state, err := f.uc.State(context.Background(), "p1")
	if err != nil {
		t.Fatal(err)
	}
	if state.StoryApproved || state.Input != (BeatInput{}) {
		t.Fatalf("state = %+v", state)
	}
}

func TestStateInputSaysTheBriefIsNotApproved(t *testing.T) {
	f := newBeatFixture()
	f.open(t)
	f.briefs.briefOK = false
	state, err := f.uc.State(context.Background(), "p1")
	if err != nil {
		t.Fatal(err)
	}
	if state.Input.BriefApproved || state.Input.Brief != "" {
		t.Fatalf("input = %+v", state.Input)
	}
}

func TestBeatContextNeedsTheBrief(t *testing.T) {
	f := newBeatFixture()
	f.open(t)
	f.briefs.briefOK = false
	if _, err := f.uc.context(context.Background(), "p1"); !errors.Is(err, ErrBeatsNeedBrief) || !errors.Is(err, domain.ErrInvalidStatus) {
		t.Fatalf("err = %v", err)
	}
}

func TestApproveAndUnapproveABeat(t *testing.T) {
	f := newBeatFixture()
	f.generate(t)
	ctx := context.Background()
	if err := f.uc.Approve(ctx, "p1", "b2"); err != nil {
		t.Fatal(err)
	}
	if slot, _ := f.beat(t, "b2"); slot.Status != domain.UnitApproved {
		t.Fatalf("status = %s, want approved", slot.Status)
	}
	if err := f.uc.Unapprove(ctx, "p1", "b2"); err != nil {
		t.Fatal(err)
	}
	if slot, _ := f.beat(t, "b2"); slot.Status != domain.UnitPending {
		t.Fatalf("status = %s, want back to review", slot.Status)
	}
	if err := f.uc.Approve(ctx, "p1", "b99"); !errors.Is(err, domain.ErrUnitNotFound) {
		t.Errorf("approving a beat that is not there: err = %v", err)
	}
}

func TestBeatsWrittenFromAnotherStoryReadAsStale(t *testing.T) {
	f := newBeatFixture()
	f.generate(t)
	state, _ := f.uc.State(context.Background(), "p1")
	for _, b := range state.Beats {
		if b.Stale {
			t.Fatalf("beat %s is stale right after it was written", b.Key)
		}
	}
	f.stories.story.CoreAnswer = "Một câu trả lời khác. [F03]"
	state, _ = f.uc.State(context.Background(), "p1")
	for _, b := range state.Beats {
		if !b.Stale {
			t.Errorf("beat %s must read as stale once the story changed", b.Key)
		}
	}
}

func TestBeatSheetStaysOneStoredSheetPerChange(t *testing.T) {
	f := newBeatFixture()
	f.generate(t)
	if err := f.uc.Reorder(context.Background(), "p1", []string{"b3", "b2", "b1"}); err != nil {
		t.Fatal(err)
	}
	sheet, err := f.sheets.GetBeatSheet(context.Background(), "p1")
	if err != nil {
		t.Fatal(err)
	}
	if len(sheet.Beats) != 3 || sheet.Beats[0].Key != "b3" || !sheet.SkeletonApproved || sheet.NextSeq != 4 {
		t.Errorf("sheet = %+v, want the new order with the gate kept", sheet)
	}
}
