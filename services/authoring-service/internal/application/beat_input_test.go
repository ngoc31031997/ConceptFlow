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
