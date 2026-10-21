package application

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"authoring/internal/domain"
)

const beatsReplyFull = `{"beats":[` +
	`{"key":"","fn":"Hook","title":"Quả sung","goal":"Gây tò mò","content":"Mở từ đĩa sung","facts":["F01"],"dur":18,"next":"Zoom vào"},` +
	`{"key":"","fn":"Giải thích cơ chế","title":"Cửa hẹp","goal":"Ong mất cánh","content":"Ong chui qua cửa","facts":["F02"],"dur":200,"next":"Vào trong"},` +
	`{"key":"","fn":"Payoff","title":"Ai nuôi ai","goal":"Trả lời","content":"Hai loài dựa nhau","facts":["F03"],"dur":50,"next":""}]}`

type beatFixture struct {
	units *briefUnits
	store *researchStore
	llm   *briefLLM
	uc    *BeatDeskUseCase
}

func newBeatFixture() *beatFixture {
	units := newBriefUnits()
	project := &briefProject{project: domain.Project{ContentLanguage: domain.LanguageVietnamese, VideoOutputMode: domain.ModeLongOnly}, status: domain.StatusDraft}
	f := &beatFixture{
		units: units,
		store: &researchStore{briefUnits: units, topic: "Vì sao trong quả sung có xác ong?", duration: domain.Duration4to5},
		llm:   &briefLLM{reply: beatsReplyFull},
	}
	f.uc = NewBeatDeskUseCase(f.store, project, project, briefPrompts{}, f.llm, nil, 4000)
	return f
}

func (f *beatFixture) save(t *testing.T, kind, key string, status domain.UnitStatus, content any, creatorFields ...string) {
	t.Helper()
	raw, err := json.Marshal(content)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.units.SaveUnit(context.Background(), domain.ReviewUnit{
		ProjectID: "p1", Kind: kind, Key: key, Status: status, Content: raw, CreatorFields: creatorFields,
	}); err != nil {
		t.Fatal(err)
	}
}

// open approves the brief and the story and seeds three facts, F01 a must.
func (f *beatFixture) open(t *testing.T) {
	t.Helper()
	f.save(t, domain.UnitKindBrief, domain.BriefUnitKey, domain.UnitApproved, domain.CreativeBrief{Question: "Vì sao ong chết?", Audience: "a", Know: "k", Understand: "u", Feel: "f", Tone: "t", OutOfScope: "o"})
	f.save(t, domain.UnitKindStory, domain.StoryUnitKey, domain.UnitApproved, domain.StoryGate{UsedFacts: []string{}})
	f.save(t, domain.UnitKindStoryField, "question", domain.UnitPending, domain.StoryField{Text: "Vì sao ong chết trong quả sung?", Facts: []string{}})
	f.save(t, domain.UnitKindFact, "F01", domain.UnitPending, domain.Fact{ClaimVI: "Sung là chùm hoa kín.", Type: domain.FactTypeFact, Keep: true, Must: true})
	f.save(t, domain.UnitKindFact, "F02", domain.UnitPending, domain.Fact{ClaimVI: "Ong mẹ chết trong quả.", Type: domain.FactTypeFact, Keep: true})
	f.save(t, domain.UnitKindFact, "F03", domain.UnitPending, domain.Fact{ClaimVI: "Ficin phân giải xác ong.", Type: domain.FactTypeHypothesis, Keep: true})
	f.save(t, domain.UnitKindFact, "F04", domain.UnitPending, domain.Fact{ClaimVI: "Đã bỏ.", Type: domain.FactTypeFact, Keep: false})
}

func (f *beatFixture) beat(t *testing.T, key string) (domain.ReviewUnit, domain.Beat) {
	t.Helper()
	u, err := f.units.GetUnit(context.Background(), "p1", domain.UnitKindBeat, key)
	if err != nil {
		t.Fatalf("beat %s: %v", key, err)
	}
	b, err := readBeat(u)
	if err != nil {
		t.Fatal(err)
	}
	return u, b
}

func (f *beatFixture) generate(t *testing.T) BeatState {
	t.Helper()
	f.open(t)
	if err := f.uc.Generate(context.Background(), "p1", ""); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	state, err := f.uc.State(context.Background(), "p1")
	if err != nil {
		t.Fatal(err)
	}
	return state
}

func TestGenerateWritesBeatsInOrderWithStableKeys(t *testing.T) {
	f := newBeatFixture()
	state := f.generate(t)
	if len(state.Beats) != 3 || state.Beats[0].Key != "b1" || state.Beats[2].Key != "b3" || state.Beats[1].Title != "Cửa hẹp" {
		t.Fatalf("beats = %+v, want three beats keyed b1..b3 in order", state.Beats)
	}
	if state.Status != domain.UnitPending || state.TotalSeconds != 268 || state.TargetLabel != "4–5 phút" {
		t.Errorf("status=%s total=%d target=%q", state.Status, state.TotalSeconds, state.TargetLabel)
	}
	if len(state.Facts) != 3 {
		t.Errorf("facts = %+v, want the three kept facts only", state.Facts)
	}
	if !strings.Contains(f.llm.systems[0], "F01 [fact · bắt buộc nói]") || !strings.Contains(f.llm.systems[0], "question (Câu hỏi mở đầu)") {
		t.Errorf("the prompt must carry the facts and the story, got %q", f.llm.systems[0])
	}
}

func TestGenerateRefusesBeforeTheStoryIsApproved(t *testing.T) {
	f := newBeatFixture()
	f.open(t)
	f.save(t, domain.UnitKindStory, domain.StoryUnitKey, domain.UnitPending, domain.StoryGate{})
	if err := f.uc.Generate(context.Background(), "p1", ""); !errors.Is(err, ErrBeatsNeedStory) {
		t.Errorf("err = %v, want the story to be required first", err)
	}
	if len(f.llm.systems) != 0 {
		t.Error("the model must not be called while the step is not open")
	}
}

func TestGenerateStoresNothingFromAnUnusableReply(t *testing.T) {
	f := newBeatFixture()
	f.open(t)
	f.llm.reply = strings.Replace(beatsReplyFull, `"F02"`, `"F77"`, 1)
	if err := f.uc.Generate(context.Background(), "p1", ""); !errors.Is(err, domain.ErrBeatReplyUnusable) {
		t.Fatalf("err = %v, want ErrBeatReplyUnusable", err)
	}
	units, _ := f.units.ListUnits(context.Background(), "p1", domain.UnitKindBeat)
	if len(units) != 0 {
		t.Errorf("%d beats stored from an unusable reply", len(units))
	}
}

func TestGenerateKeepsWhatTheCreatorWrote(t *testing.T) {
	f := newBeatFixture()
	f.generate(t)
	if _, err := f.uc.BeatUnitKind().Edit(context.Background(), "p1", "b2", map[string]string{"goal": "Mục tiêu của tôi"}); err != nil {
		t.Fatal(err)
	}
	f.llm.reply = strings.Replace(strings.Replace(beatsReplyFull, `"key":"","fn":"Giải`, `"key":"b2","fn":"Giải`, 1), `"goal":"Ong mất cánh"`, `"goal":"AI viết lại"`, 1)
	if err := f.uc.Generate(context.Background(), "p1", ""); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	u, b := f.beat(t, "b2")
	if b.Goal != "Mục tiêu của tôi" || !u.HasCreatorField(domain.BeatFieldGoal) {
		t.Errorf("goal = %q mine=%v, want the Creator's goal kept", b.Goal, u.CreatorFields)
	}
	state, _ := f.uc.State(context.Background(), "p1")
	if len(state.Beats) != 3 || state.Beats[0].Key != "b4" || state.Beats[1].Key != "b2" {
		t.Errorf("beats = %+v, want the kept beat b2 between two new keys b4 and b5 (keys are never reused)", state.Beats)
	}
	if _, err := f.units.GetUnit(context.Background(), "p1", domain.UnitKindBeat, "b1"); !errors.Is(err, domain.ErrUnitNotFound) {
		t.Error("a replaced beat must be removed")
	}
}

func TestGenerateMustKeepEveryBeatTheCreatorWrote(t *testing.T) {
	f := newBeatFixture()
	f.generate(t)
	if _, err := f.uc.BeatUnitKind().Edit(context.Background(), "p1", "b2", map[string]string{"title": "Tên của tôi"}); err != nil {
		t.Fatal(err)
	}
	if err := f.uc.Generate(context.Background(), "p1", ""); !errors.Is(err, domain.ErrBeatReplyUnusable) {
		t.Errorf("err = %v, want a reply that drops the Creator's beat to be refused", err)
	}
	if _, b := f.beat(t, "b2"); b.Title != "Tên của tôi" {
		t.Error("the Creator's beat must be untouched after the refused run")
	}
}

func TestEditMarksTheFieldAsTheCreatorsAndSendsTheBeatBackToReview(t *testing.T) {
	f := newBeatFixture()
	f.generate(t)
	kind := f.uc.BeatUnitKind()
	f.save(t, domain.UnitKindBeat, "b1", domain.UnitApproved, domain.Beat{Fn: "Hook", Title: "t", Goal: "g", Content: "c", Facts: []string{}, Dur: 18})
	unit, err := kind.Edit(context.Background(), "p1", "b1", map[string]string{"dur": "23", "facts": "f1, F02", "fn": "Đặt vấn đề"})
	if err != nil {
		t.Fatalf("Edit: %v", err)
	}
	b, _ := readBeat(unit)
	if b.Dur != 23 || strings.Join(b.Facts, ",") != "F01,F02" || b.Fn != "Đặt vấn đề" || unit.Status != domain.UnitPending {
		t.Errorf("beat = %+v status=%s", b, unit.Status)
	}
	if !unit.HasCreatorField("dur") || !unit.HasCreatorField("facts") || !unit.HasCreatorField("fn") || unit.HasCreatorField("goal") {
		t.Errorf("mine = %v, want exactly the fields edited", unit.CreatorFields)
	}
}

func TestEditRefusesWhatABeatCannotHold(t *testing.T) {
	f := newBeatFixture()
	f.generate(t)
	kind := f.uc.BeatUnitKind()
	bad := []map[string]string{
		{"dur": "2"}, {"dur": "abc"}, {"fn": "Mở bài"}, {"facts": "F77"}, {"facts": "xyz"}, {"title": " "}, {"colour": "red"}, {},
	}
	for _, fields := range bad {
		if _, err := kind.Edit(context.Background(), "p1", "b1", fields); !errors.Is(err, domain.ErrInvalidUnitEdit) {
			t.Errorf("Edit(%v) err = %v, want ErrInvalidUnitEdit", fields, err)
		}
	}
	if _, b := f.beat(t, "b1"); b.Dur != 18 {
		t.Error("a refused edit must change nothing")
	}
}

func TestEmptyGoalCannotBeApproved(t *testing.T) {
	f := newBeatFixture()
	f.generate(t)
	kind := f.uc.BeatUnitKind()
	if _, err := kind.Edit(context.Background(), "p1", "b1", map[string]string{"goal": ""}); err != nil {
		t.Fatal(err)
	}
	u, _ := f.units.GetUnit(context.Background(), "p1", domain.UnitKindBeat, "b1")
	if err := kind.CheckApprovable(context.Background(), u); !errors.Is(err, ErrBeatNoGoal) {
		t.Errorf("err = %v, want ErrBeatNoGoal", err)
	}
}

func TestRedoWritesOnlyThatBeatAndReleasesItsCreatorFields(t *testing.T) {
	f := newBeatFixture()
	f.generate(t)
	kind := f.uc.BeatUnitKind()
	if _, err := kind.Edit(context.Background(), "p1", "b2", map[string]string{"title": "Của tôi"}); err != nil {
		t.Fatal(err)
	}
	f.llm.reply = `{"beats":[{"key":"b2","fn":"Cao trào","title":"Viết lại","goal":"G","content":"C","facts":["F02"],"dur":40,"next":"N"}]}`
	unit, err := kind.Redo(context.Background(), "p1", "b2", "ngắn hơn")
	if err != nil {
		t.Fatalf("Redo: %v", err)
	}
	b, _ := readBeat(unit)
	if b.Title != "Viết lại" || unit.RedoCount != 1 || len(unit.CreatorFields) != 0 || unit.Status != domain.UnitPending {
		t.Errorf("beat = %+v redo=%d mine=%v", b, unit.RedoCount, unit.CreatorFields)
	}
	if _, other := f.beat(t, "b1"); other.Title != "Quả sung" {
		t.Error("redoing one beat must not touch the others")
	}
	if !strings.Contains(f.llm.systems[len(f.llm.systems)-1], "ngắn hơn") {
		t.Error("the prompt must carry the Creator's note")
	}
}

func TestAddPutsTheNewBeatAfterTheGivenOneAndNeverReusesAKey(t *testing.T) {
	f := newBeatFixture()
	f.generate(t)
	f.llm.reply = `{"beats":[{"key":"","fn":"Cao trào","title":"Mới","goal":"G","content":"C","facts":[],"dur":20,"next":"N"}]}`
	key, err := f.uc.Add(context.Background(), "p1", "b1", false, "")
	if err != nil || key != "b4" {
		t.Fatalf("Add = %q, %v, want b4", key, err)
	}
	if err := f.uc.Delete(context.Background(), "p1", "b4"); err != nil {
		t.Fatal(err)
	}
	key, err = f.uc.Add(context.Background(), "p1", "b1", true, "")
	if err != nil || key != "b5" {
		t.Fatalf("Add blank = %q, %v, want b5 (b4 is not reused)", key, err)
	}
	state, _ := f.uc.State(context.Background(), "p1")
	order := []string{}
	for _, b := range state.Beats {
		order = append(order, b.Key)
	}
	if strings.Join(order, ",") != "b1,b5,b2,b3" {
		t.Errorf("order = %v, want b1,b5,b2,b3", order)
	}
}

func TestAddBlankWorksWithoutALanguageModel(t *testing.T) {
	f := newBeatFixture()
	f.open(t)
	f.uc.provider = nil
	if _, err := f.uc.Add(context.Background(), "p1", "", false, ""); !errors.Is(err, ErrLLMNotConfigured) {
		t.Errorf("err = %v, want ErrLLMNotConfigured for an AI beat", err)
	}
	key, err := f.uc.Add(context.Background(), "p1", "", true, "")
	if err != nil || key != "b1" {
		t.Fatalf("Add blank = %q, %v, want b1", key, err)
	}
}

func TestAddNeedsAnAnchorOnceTheSheetHasBeats(t *testing.T) {
	f := newBeatFixture()
	f.generate(t)
	if _, err := f.uc.Add(context.Background(), "p1", "", true, ""); !errors.Is(err, domain.ErrInvalidUnitEdit) {
		t.Errorf("err = %v, want the anchor to be required", err)
	}
	if _, err := f.uc.Add(context.Background(), "p1", "b99", true, ""); !errors.Is(err, domain.ErrUnitNotFound) {
		t.Errorf("err = %v, want ErrUnitNotFound", err)
	}
}

func TestReorderChangesOnlyTheOrder(t *testing.T) {
	f := newBeatFixture()
	f.generate(t)
	f.save(t, domain.UnitKindBeat, "b1", domain.UnitApproved, domain.Beat{Fn: "Hook", Title: "t", Goal: "g", Content: "c", Facts: []string{}, Dur: 18})
	if err := f.uc.Reorder(context.Background(), "p1", []string{"b2", "b1", "b3"}); err != nil {
		t.Fatalf("Reorder: %v", err)
	}
	state, _ := f.uc.State(context.Background(), "p1")
	if state.Beats[0].Key != "b2" || state.Beats[1].Key != "b1" || state.Beats[1].Status != domain.UnitApproved {
		t.Errorf("beats = %+v, want the new order with b1 still approved", state.Beats)
	}
	for _, order := range [][]string{{"b1", "b2"}, {"b1", "b1", "b2"}, {"b1", "b2", "b9"}} {
		if err := f.uc.Reorder(context.Background(), "p1", order); !errors.Is(err, domain.ErrInvalidUnitEdit) {
			t.Errorf("Reorder(%v) err = %v, want ErrInvalidUnitEdit", order, err)
		}
	}
}

func TestDeleteNamesTheMustFactNoOtherBeatHolds(t *testing.T) {
	f := newBeatFixture()
	state := f.generate(t)
	if len(state.Beats[0].OnlyMust) != 1 || state.Beats[0].OnlyMust[0] != "F01" || len(state.Beats[1].OnlyMust) != 0 {
		t.Errorf("only_must = %v / %v, want F01 on b1 only", state.Beats[0].OnlyMust, state.Beats[1].OnlyMust)
	}
	if err := f.uc.Delete(context.Background(), "p1", "b1"); err != nil {
		t.Fatal(err)
	}
	state, _ = f.uc.State(context.Background(), "p1")
	if len(state.Beats) != 2 || state.Checks[2].OK {
		t.Errorf("beats=%d checks=%+v, want the must-fact check to fail after the delete", len(state.Beats), state.Checks)
	}
	if err := f.uc.Delete(context.Background(), "p1", "b1"); !errors.Is(err, domain.ErrUnitNotFound) {
		t.Errorf("err = %v, want ErrUnitNotFound for a beat already gone", err)
	}
}

func TestStateIsApprovedOnlyWhenEveryBeatIs(t *testing.T) {
	f := newBeatFixture()
	f.generate(t)
	for _, key := range []string{"b1", "b2"} {
		u, b := f.beat(t, key)
		u.Status = domain.UnitApproved
		if _, err := f.uc.saveBeat(context.Background(), "p1", key, b, u); err != nil {
			t.Fatal(err)
		}
	}
	if state, _ := f.uc.State(context.Background(), "p1"); state.Status != domain.UnitPending {
		t.Error("two of three approved must still wait for review")
	}
	u, b := f.beat(t, "b3")
	u.Status = domain.UnitApproved
	if _, err := f.uc.saveBeat(context.Background(), "p1", "b3", b, u); err != nil {
		t.Fatal(err)
	}
	if state, _ := f.uc.State(context.Background(), "p1"); state.Status != domain.UnitApproved {
		t.Error("every beat approved must approve the sheet")
	}
}

func TestFactDependentsNameTheBeatsThatUseAFact(t *testing.T) {
	f := newBeatFixture()
	f.generate(t)
	items, err := f.uc.FactDependents()(context.Background(), "p1", "F02")
	if err != nil || len(items) != 1 || !strings.Contains(items[0], "B2") {
		t.Errorf("items = %v, err = %v, want B2 named", items, err)
	}
	if items, _ := f.uc.FactDependents()(context.Background(), "p1", "F04"); len(items) != 0 {
		t.Errorf("items = %v, want none for an unused fact", items)
	}
}
