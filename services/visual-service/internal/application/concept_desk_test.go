package application

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"visual/internal/domain"
)

var ctx = context.Background()

func TestState_NeedsALockedScript(t *testing.T) {
	r := newRig(t)
	r.auth.err = domain.ErrScriptNotLocked
	if _, err := r.desk.State(ctx, project); !errors.Is(err, domain.ErrScriptNotLocked) {
		t.Fatalf("%v", err)
	}
}

func TestGenerate_WritesTheEmptyPartsInOrderEachReadingTheOnesBefore(t *testing.T) {
	r := newRig(t)
	r.queueAll()
	st := mustState(t)(r.desk.Generate(ctx, project, "", false))
	if len(r.llm.calls) != 8 {
		t.Fatalf("%d calls", len(r.llm.calls))
	}
	for i, spec := range domain.ConceptUnits {
		if r.llm.calls[i].Role != domain.PromptRoleOf(spec.Key) || r.llm.calls[i].Model != "ollama" || r.llm.calls[i].Step != ConceptStep {
			t.Errorf("call %d %+v", i, r.llm.calls[i])
		}
	}
	if !strings.Contains(r.llm.calls[2].System, `"style_name":"Phẳng sáng"`) || !strings.Contains(r.llm.calls[0].System, "R01 · Sáng: nền tối\n  - hình dạng: tròn") {
		t.Error("a later part reads the parts before it, and the picked descriptions")
	}
	for _, u := range st.Units {
		if !u.Written || u.Status != domain.UnitPending || len(u.Problems) > 0 || u.Stale {
			t.Errorf("%s %+v", u.Key, u)
		}
	}
	var f domain.Feasibility
	_ = json.Unmarshal(unitView(st, domain.UnitFeasibility).Content, &f)
	if f.Scope.Beats != 2 || f.Scope.Characters != 1 || f.Scope.EstShots != 8 {
		t.Errorf("the machine counts the scope: %+v", f.Scope)
	}
	r.llm.calls = nil
	if _, err := r.desk.Generate(ctx, project, "", false); err != nil || len(r.llm.calls) != 0 {
		t.Errorf("nothing left to write: %d calls, %v", len(r.llm.calls), err)
	}
}

func TestGenerateUnit_KeepsTheCreatorsFieldsAndStoresNothingOnABadReply(t *testing.T) {
	r := newRig(t)
	r.queueAll()
	mustState(t)(r.desk.GenerateUnit(ctx, project, domain.UnitTypography, ""))
	mustState(t)(r.desk.Edit(ctx, project, domain.UnitTypography, map[string]json.RawMessage{"captions": json.RawMessage(`"của tôi"`)}))
	r.llm.replies[domain.PromptRoleOf(domain.UnitTypography)] = []string{strings.Replace(goodReplies[domain.UnitTypography], `"captions":"c"`, `"captions":"AI viết"`, 1)}
	st := mustState(t)(r.desk.GenerateUnit(ctx, project, domain.UnitTypography, "ngắn hơn"))
	u := unitView(st, domain.UnitTypography)
	var typo domain.Typography
	_ = json.Unmarshal(u.Content, &typo)
	if typo.Captions != "của tôi" || u.RedoCount != 1 || u.CreatorFields[0] != "captions" {
		t.Errorf("creator field kept: %+v %+v", typo, u)
	}
	last := r.llm.calls[len(r.llm.calls)-1].System
	if !strings.Contains(last, "ngắn hơn") || !strings.Contains(last, "captions") {
		t.Error("the prompt carries the note and the Creator's fields")
	}

	before := r.store.units["visual_concept/typography"].Content
	r.llm.replies[domain.PromptRoleOf(domain.UnitTypography)] = []string{`{"font":3}`}
	if _, err := r.desk.GenerateUnit(ctx, project, domain.UnitTypography, ""); !errors.Is(err, domain.ErrUnreadableReply) {
		t.Fatalf("%v", err)
	}
	if string(r.store.units["visual_concept/typography"].Content) != string(before) {
		t.Error("an unreadable reply stores nothing")
	}
	if _, err := r.desk.GenerateUnit(ctx, project, "palette", ""); !errors.Is(err, domain.ErrUnknownUnit) {
		t.Error(err)
	}
}

func TestExternalAI_PromptModeStoresNothingAndPasteStoresTheReply(t *testing.T) {
	r := newRig(t)
	run := NewExternalAIRun(ExternalAIPrompt, "")
	_, err := r.desk.GenerateUnit(WithExternalAI(ctx, run), project, domain.UnitVisualLanguage, "")
	if !errors.Is(err, ErrExternalAIPromptBuilt) {
		t.Fatal(err)
	}
	if p, ok := run.Prompt(); !ok || !strings.Contains(p, "ROLE visual_concept_visual_language") {
		t.Errorf("prompt %q", p)
	}
	if len(r.store.units) != 0 {
		t.Error("prompt mode stores nothing")
	}
	paste := NewExternalAIRun(ExternalAIPaste, goodReplies[domain.UnitVisualLanguage])
	st := mustState(t)(r.desk.GenerateUnit(WithExternalAI(ctx, paste), project, domain.UnitVisualLanguage, ""))
	if !unitView(st, domain.UnitVisualLanguage).Written {
		t.Error("paste stores the part")
	}
}

func TestStale_FollowsOnlyTheSourcesAPartReads(t *testing.T) {
	r := newRig(t)
	r.queueAll()
	mustState(t)(r.desk.Generate(ctx, project, "", false))
	r.auth.v = lockedScript(t, 3, "C đã sửa")
	st := mustState(t)(r.desk.State(ctx, project))
	for _, u := range st.Units {
		readsScript := hasLine(u.Reads, "Kịch bản")
		if u.Stale != readsScript {
			t.Errorf("%s stale=%v reads script=%v", u.Key, u.Stale, readsScript)
		}
	}
	r.auth.v = lockedScript(t, 2, "C")
	st = mustState(t)(r.desk.SetRefs(ctx, project, []string{"R01", "R02"}))
	for _, u := range st.Units {
		readsRefs := u.Key == domain.UnitCreativeDirection || u.Key == domain.UnitVisualLanguage
		if u.Stale != readsRefs {
			t.Errorf("changing the picks: %s stale=%v, only parts 1 and 3 read references", u.Key, u.Stale)
		}
	}
	edited := r.store.library[0]
	edited.Intent = "đổi"
	r.store.library[0] = edited
	r.store.picks = []string{"R01"}
	st = mustState(t)(r.desk.State(ctx, project))
	if !unitView(st, domain.UnitVisualLanguage).Stale || unitView(st, domain.UnitColorSystem).Stale {
		t.Error("editing a picked description makes only the parts that read references stale")
	}
	if _, err := r.desk.SetRefs(ctx, project, []string{"R77"}); !errors.Is(err, domain.ErrUnknownReference) {
		t.Error(err)
	}
}

func TestEdit_ChecksTheSchema(t *testing.T) {
	r := newRig(t)
	if _, err := r.desk.Edit(ctx, project, domain.UnitVisualLanguage, map[string]json.RawMessage{"colours": json.RawMessage(`[]`)}); !errors.Is(err, domain.ErrInvalidEdit) {
		t.Error(err)
	}
	if _, err := r.desk.Edit(ctx, project, domain.UnitVisualLanguage, map[string]json.RawMessage{"shape_rules": json.RawMessage(`"x"`)}); !errors.Is(err, domain.ErrInvalidEdit) {
		t.Error(err)
	}
	st := mustState(t)(r.desk.Edit(ctx, project, domain.UnitVisualLanguage, map[string]json.RawMessage{"shape_rules": json.RawMessage(`["tròn"]`)}))
	u := unitView(st, domain.UnitVisualLanguage)
	if !u.Written || len(u.Problems) == 0 || u.Stale {
		t.Errorf("a hand-written part starts empty and shows what is missing: %+v", u)
	}
	if _, err := r.desk.Approve(ctx, project, domain.UnitVisualLanguage); !errors.Is(err, domain.ErrUnitInvalid) {
		t.Error("a part that fails its checks cannot be approved")
	}
}

func approveEverything(t *testing.T, r *rig) *ConceptState {
	t.Helper()
	return mustState(t)(r.desk.ApproveAll(ctx, project))
}

func TestLock_ConditionsThenV1ThenV2(t *testing.T) {
	r := newRig(t)
	r.queueAll()
	mustState(t)(r.desk.Generate(ctx, project, "", false))
	if _, err := r.desk.Lock(ctx, project, ""); !errors.Is(err, domain.ErrLockNotReady) {
		t.Fatalf("not approved: %v", err)
	}
	st := approveEverything(t, r)
	if st.ApprovedCount != 8 || !st.CanLock {
		t.Fatalf("ready: %+v %v", st.Blockers, st.ApprovedCount)
	}

	r.llm.replies[domain.PromptRoleReview] = []string{`{"findings":[{"group":"accuracy","unit_key":"typography","field":"captions","text":"mơ hồ","detail":"d","fix":"Chú thích chỉ cho số liệu","same_as":""},{"group":"feasibility","unit_key":"feasibility","field":"","text":"nặng","detail":"d","fix":"","same_as":""}],"notes":["ổn"]}`}
	st = mustState(t)(r.desk.Review(ctx, project, ""))
	if st.Review.OpenCount != 2 || st.CanLock || st.Criteria[2].OK || st.Criteria[3].OK {
		t.Fatalf("open findings block: %+v", st.Review)
	}
	if _, err := r.desk.ApplyFinding(ctx, project, "f2"); !errors.Is(err, domain.ErrFindingNotApplicable) {
		t.Error("a finding without a field cannot be applied")
	}
	st = mustState(t)(r.desk.ApplyFinding(ctx, project, "f1"))
	u := unitView(st, domain.UnitTypography)
	if u.Status != domain.UnitPending || !strings.Contains(string(u.Content), "Chú thích chỉ cho số liệu") || u.CreatorFields[0] != "captions" {
		t.Errorf("apply: %+v", u)
	}
	st = mustState(t)(r.desk.AcceptFinding(ctx, project, "f2", ""))
	if st.Review.Findings[1].Why != "không ghi lý do" {
		t.Error("an empty reason is stored as no reason")
	}
	st = approveEverything(t, r)
	if !st.CanLock {
		t.Fatalf("blockers %v", st.Blockers)
	}
	st = mustState(t)(r.desk.Lock(ctx, project, ""))
	if st.LockedVersion != 1 || st.Versions[0].Note != "Chốt lần đầu" || st.Versions[0].ScriptVersion != 2 || !st.Criteria[4].OK || st.CanLock {
		t.Fatalf("v1 %+v", st.Versions)
	}
	if _, err := r.desk.Lock(ctx, project, ""); !errors.Is(err, domain.ErrNothingToLock) {
		t.Errorf("nothing changed: %v", err)
	}
	doc, err := r.desk.LatestDocument(ctx, project, "")
	if err != nil || !strings.Contains(string(doc), `"document_id":"visual-concept-v1"`) || !strings.Contains(string(doc), `"accuracy_notes":[{"fact":"F01"`) {
		t.Errorf("document %s %v", doc, err)
	}
	if _, err := r.desk.LatestDocument(ctx, project, "storyboard"); !errors.Is(err, domain.ErrUnknownStep) {
		t.Error(err)
	}

	mustState(t)(r.desk.Edit(ctx, project, domain.UnitVisualLanguage, map[string]json.RawMessage{"examples": json.RawMessage(`["mới"]`)}))
	st = approveEverything(t, r)
	if len(st.Draft) != 1 || !strings.Contains(st.Draft[0], "Phần 3") {
		t.Fatalf("draft %v", st.Draft)
	}
	st = mustState(t)(r.desk.Lock(ctx, project, "đổi ví dụ"))
	if st.LockedVersion != 2 || st.Versions[1].Note != "đổi ví dụ" {
		t.Fatalf("v2 %+v", st.Versions)
	}
	v1, _ := r.desk.Version(ctx, project, 1)
	if strings.Contains(string(v1.Document), "mới") {
		t.Error("v1 stays as it was locked")
	}
}

func TestMachineCheckAcceptance(t *testing.T) {
	r := newRig(t)
	r.queueAll()
	r.llm.replies[domain.PromptRoleOf(domain.UnitDoNotUse)] = []string{`{"items":[{"text":"viền","why":"","facts":[],"source":"style"}]}`}
	mustState(t)(r.desk.Generate(ctx, project, "", false))
	st := approveEverything(t, r)
	if st.CanLock {
		t.Fatal("the accuracy check fails without accuracy rows")
	}
	if _, err := r.desk.AcceptFinding(ctx, project, domain.CheckTerms, "x"); !errors.Is(err, domain.ErrFindingNotApplicable) {
		t.Error("a passing check cannot be accepted")
	}
	st = mustState(t)(r.desk.AcceptFinding(ctx, project, domain.CheckAccuracy, "đã nói trong lời thoại"))
	if !st.CanLock {
		t.Fatalf("accepted check: %v", st.Blockers)
	}
}

func TestSetModel(t *testing.T) {
	r := newRig(t)
	st := mustState(t)(r.desk.SetModel(ctx, project, "deepseek-chat"))
	if st.Model != "deepseek-chat" || st.DefaultModel != "ollama" || len(st.Models) != 2 {
		t.Errorf("%+v", st.Model)
	}
	if _, err := r.desk.SetModel(ctx, project, "gpt-x"); !errors.Is(err, ErrUnknownModel) {
		t.Error(err)
	}
	r.queueAll()
	mustState(t)(r.desk.GenerateUnit(ctx, project, domain.UnitVisualLanguage, ""))
	if r.llm.calls[0].Model != "deepseek-chat" {
		t.Error("the video's model is used")
	}
}
