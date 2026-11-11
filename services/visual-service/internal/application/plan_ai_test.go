package application

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"visual/internal/domain"
)

// seedPrompts serves the prompts the service ships, so a variable the code
// forgets to fill shows up as a leftover {{name}}.
type seedPrompts struct{}

func (seedPrompts) ActiveTemplate(_ context.Context, role string) (string, error) {
	for _, p := range domain.SystemPrompts() {
		if p.Role == role {
			return p.TemplateText, nil
		}
	}
	return "", ErrPromptNotFound
}

type planModels struct{ fakeModels }

func (planModels) StepDefaultModel(_ context.Context, step string) (string, error) {
	if step != PlanStep {
		return "", ErrLLMFailed
	}
	return "ollama", nil
}

type aiFixture struct {
	*planFixture
	llm *fakeLLM
}

func newAIFixture(t *testing.T) *aiFixture {
	t.Helper()
	f := &aiFixture{planFixture: newPlanFixture(t), llm: &fakeLLM{replies: map[string][]string{}}}
	f.desk.WithAI(f.llm, seedPrompts{}, planModels{}, 1000)
	return f
}

func unitJSON(summary string, refs ...string) string {
	quoted := make([]string, len(refs))
	for i, r := range refs {
		quoted[i] = `"` + r + `"`
	}
	return `{"script_refs":[` + strings.Join(quoted, ",") + `],"narration_summary":"` + summary + `","communication_goal":"g",
	"essential_visual_info":[{"item":"i","priority":"must"}],"on_screen_text":[],
	"visual_approach":{"method":"diagram","description":"d","concept_rule_refs":["color_system.palette.ink"]},
	"narration_relationship":"contrast","duration":{"planned_s":0,"extra_reason":""},"risk_notes":[],"open_questions":[]}`
}

func beatReply(units ...string) string { return `{"units":[` + strings.Join(units, ",") + `]}` }

func (f *aiFixture) queue(role string, replies ...string) { f.llm.replies[role] = replies }

func (f *aiFixture) beatUnits(key string) []domain.PlanUnit {
	f.t.Helper()
	plan, err := domain.DecodeBeatPlan(f.store.units[domain.UnitKindPlan+"/"+key].Content)
	if err != nil {
		f.t.Fatal(err)
	}
	return plan.Units
}

func TestPlanGenerateBeat_WritesTheBeatFromItsLinesAndStoresAfterTheAnswer(t *testing.T) {
	f := newAIFixture(t)
	f.queue(domain.PromptRolePlanBeat, beatReply(unitJSON("một", "b1.1"), unitJSON("hai ba", "b1.2", "b1.3")))
	st, err := f.desk.GenerateBeat(context.Background(), project, "b1", "ngắn gọn")
	if err != nil {
		t.Fatal(err)
	}
	if len(f.llm.calls) != 1 {
		t.Fatalf("one model turn, got %d", len(f.llm.calls))
	}
	call := f.llm.calls[0]
	if call.Role != domain.PromptRolePlanBeat || call.Step != PlanStep || call.Model != "ollama" || !call.ExpectJSON || call.ProjectID != project {
		t.Errorf("the turn is of the beat role on the step's default model: %+v", call)
	}
	if strings.Contains(call.System, "{{") {
		t.Errorf("a variable of the prompt was not filled:\n%s", call.System)
	}
	for _, want := range []string{"b1.2 Có thể B [F02] · tối thiểu 30.00 giây", "ngắn gọn", "color_system.palette.ink", "F02 [giả thuyết] B", "(không có)", "b2 · B2 · Payoff"} {
		if !strings.Contains(call.System, want) {
			t.Errorf("the prompt lacks %q", want)
		}
	}
	units := f.beatUnits("b1")
	if len(units) != 2 || units[0].UnitID != "VP-001" || units[1].UnitID != "VP-002" || units[1].NarrationSummary != "hai ba" || units[0].Status != domain.UnitPending {
		t.Fatalf("units over the lines a unit already had keep its id: %+v", units)
	}
	if f.store.seq != 4 {
		t.Errorf("no unit number is used when every unit keeps its id")
	}
	if st.UnitsTotal != 3 || len(st.Beats[0].Units) != 2 || st.Beats[0].Stale {
		t.Errorf("the screen shows the new units and a fresh beat: %+v", st.Beats[0])
	}
	if got, want := f.store.units[domain.UnitKindPlan+"/b1"].InputHashes, f.input().HashesFor("b1"); !reflect.DeepEqual(got, want) {
		t.Errorf("the beat is stored with the hashes of its sources now")
	}
	if f.store.units[domain.UnitKindPlan+"/b1"].RedoCount != 1 {
		t.Errorf("a rewrite counts as a redo")
	}
}

func TestPlanGenerateBeat_AUnitOverNewLinesTakesANewNumber(t *testing.T) {
	f := newAIFixture(t)
	f.queue(domain.PromptRolePlanBeat, beatReply(unitJSON("a", "b1.1"), unitJSON("b", "b1.2"), unitJSON("c", "b1.3")))
	if _, err := f.desk.GenerateBeat(context.Background(), project, "b1", ""); err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, u := range f.beatUnits("b1") {
		ids = append(ids, u.UnitID)
	}
	if strings.Join(ids, ",") != "VP-001,VP-004,VP-005" || f.store.seq != 6 {
		t.Errorf("the unit over b1.1 keeps VP-001, the two new ones take the next numbers: %v seq %d", ids, f.store.seq)
	}
}

func TestPlanGenerateBeat_AReplyThatDoesNotFitStoresNothing(t *testing.T) {
	for name, reply := range map[string]string{
		"a line left out": beatReply(unitJSON("một", "b1.1"), unitJSON("hai", "b1.2")),
		"a strange value": strings.Replace(beatReply(unitJSON("a", "b1.1", "b1.2", "b1.3")), `"diagram"`, `"hologram"`, 1),
		"not json":        `{"units": nope`,
	} {
		f := newAIFixture(t)
		before := fmt.Sprint(f.store.units)
		f.queue(domain.PromptRolePlanBeat, reply)
		if _, err := f.desk.GenerateBeat(context.Background(), project, "b1", ""); !errors.Is(err, domain.ErrUnreadableReply) {
			t.Errorf("%s: want an unreadable reply, got %v", name, err)
		}
		if fmt.Sprint(f.store.units) != before || f.store.seq != 4 {
			t.Errorf("%s: nothing may be stored, no unit number used", name)
		}
	}
	f := newAIFixture(t)
	if _, err := f.desk.GenerateBeat(context.Background(), project, "b1", ""); !errors.Is(err, ErrLLMFailed) {
		t.Errorf("a model failure is passed on, got %v", err)
	}
}

func TestPlanGenerateBeat_KeepsTheUnitsTheCreatorEdited(t *testing.T) {
	f := newAIFixture(t)
	row := f.store.units[domain.UnitKindPlan+"/b1"]
	plan, _ := domain.DecodeBeatPlan(row.Content)
	plan.Units[1].NarrationSummary = "lời của Creator"
	raw, _ := plan.Encode()
	row.Content, row.CreatorFields = raw, []string{"VP-002.narration_summary"}
	f.store.units[domain.UnitKindPlan+"/b1"] = row

	f.queue(domain.PromptRolePlanBeat, beatReply(unitJSON("mới", "b1.1"), unitJSON("lời của AI", "b1.2", "b1.3")))
	if _, err := f.desk.GenerateBeat(context.Background(), project, "b1", ""); err != nil {
		t.Fatal(err)
	}
	units := f.beatUnits("b1")
	if units[1].UnitID != "VP-002" || units[1].NarrationSummary != "lời của Creator" || units[1].CommunicationGoal != "g" {
		t.Errorf("the edited unit keeps its id and the field, the rest is the AI's: %+v", units[1])
	}
	if units[0].UnitID != "VP-001" || units[0].NarrationSummary != "mới" {
		t.Errorf("the unit over the same line is rewritten under its id: %+v", units[0])
	}
	if got := f.store.units[domain.UnitKindPlan+"/b1"].CreatorFields; !reflect.DeepEqual(got, []string{"VP-002.narration_summary"}) {
		t.Errorf("the Creator's field stays recorded: %v", got)
	}
	if !strings.Contains(f.llm.calls[0].System, "VP-002.narration_summary") {
		t.Errorf("the prompt tells the AI which fields are the Creator's")
	}

	f.queue(domain.PromptRolePlanBeat, beatReply(unitJSON("gộp", "b1.1", "b1.2", "b1.3")))
	before := fmt.Sprint(f.store.units)
	if _, err := f.desk.GenerateBeat(context.Background(), project, "b1", ""); !errors.Is(err, domain.ErrUnreadableReply) || !strings.Contains(err.Error(), "VP-002") {
		t.Errorf("a reply that drops the edited unit is refused by name, got %v", err)
	}
	if fmt.Sprint(f.store.units) != before {
		t.Errorf("the refused reply stored something")
	}
}

func TestPlanGenerate_WritesEmptyBeatsInOrderEachReadingTheLast(t *testing.T) {
	f := newAIFixture(t)
	delete(f.store.units, domain.UnitKindPlan+"/b1")
	delete(f.store.units, domain.UnitKindPlan+"/b2")
	f.queue(domain.PromptRolePlanBeat,
		beatReply(unitJSON("hình mở đầu", "b1.1", "b1.2", "b1.3")),
		beatReply(unitJSON("hình kết", "b2.1")))
	st, err := f.desk.Generate(context.Background(), project, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(f.llm.calls) != 2 || !strings.Contains(f.llm.calls[0].System, "b1.1 A là thật") || !strings.Contains(f.llm.calls[1].System, "b2.1 Không phải C") {
		t.Fatalf("one turn per beat, in story order: %d", len(f.llm.calls))
	}
	if !strings.Contains(f.llm.calls[1].System, "hình mở đầu") {
		t.Errorf("the second beat reads the units of the first")
	}
	if st.UnitsTotal != 2 || !st.Coverage.Complete() {
		t.Errorf("both beats are written and cover the script: %+v", st.Coverage)
	}

	// Only beats with no unit are written; a failure keeps what came before it.
	g := newAIFixture(t)
	delete(g.store.units, domain.UnitKindPlan+"/b1")
	delete(g.store.units, domain.UnitKindPlan+"/b2")
	g.queue(domain.PromptRolePlanBeat, beatReply(unitJSON("một", "b1.1", "b1.2", "b1.3")))
	if _, err := g.desk.Generate(context.Background(), project, ""); !errors.Is(err, ErrLLMFailed) || !strings.Contains(err.Error(), "beat 2") {
		t.Errorf("the failing beat is named: %v", err)
	}
	if len(g.beatUnits("b1")) != 1 || g.store.units[domain.UnitKindPlan+"/b2"].Key != "" {
		t.Errorf("beat 1 stays written, beat 2 is not")
	}
	h := newAIFixture(t)
	if _, err := h.desk.Generate(context.Background(), project, ""); err != nil || len(h.llm.calls) != 0 {
		t.Errorf("a plan with every beat written asks nothing: %v, %d calls", err, len(h.llm.calls))
	}
}

func TestPlanAI_RefusesWhatItCannotRunNow(t *testing.T) {
	ctx := context.Background()
	f := newPlanFixture(t)
	if _, err := f.desk.GenerateBeat(ctx, project, "b1", ""); !errors.Is(err, ErrLLMNotConfigured) {
		t.Errorf("a desk with no AI: %v", err)
	}
	if _, err := f.desk.Review(ctx, project, ""); !errors.Is(err, ErrLLMNotConfigured) {
		t.Errorf("a desk with no AI: %v", err)
	}
	g := newAIFixture(t)
	if _, err := g.desk.GenerateBeat(ctx, project, "b9", ""); !errors.Is(err, domain.ErrUnknownPlanBeat) {
		t.Errorf("a beat the script lacks: %v", err)
	}
	finish := g.desk.running.start(project, "b1")
	if _, err := g.desk.GenerateBeat(ctx, project, "b1", ""); !errors.Is(err, domain.ErrUnitRunning) {
		t.Errorf("a beat the AI is writing: %v", err)
	}
	finish()
	if len(g.llm.calls) != 0 {
		t.Errorf("nothing was asked")
	}
}

func TestPlanAI_AnAIOutsideTheSystemGetsThePromptAndGivesBackTheReply(t *testing.T) {
	f := newAIFixture(t)
	llm := f.llm
	f.desk.WithAI(NewExternalAIProvider(llm, llm), seedPrompts{}, planModels{}, 1000)

	run := NewExternalAIRun(ExternalAIPrompt, "")
	before := fmt.Sprint(f.store.units)
	_, err := f.desk.GenerateBeat(WithExternalAI(context.Background(), run), project, "b1", "")
	prompt, ok := run.Prompt()
	if !errors.Is(err, ErrExternalAIPromptBuilt) || !ok || !strings.Contains(prompt, "b1.2 Có thể B") || len(llm.calls) != 0 || fmt.Sprint(f.store.units) != before {
		t.Fatalf("prompt mode builds the prompt and stops: %v %v %d", err, ok, len(llm.calls))
	}

	run = NewExternalAIRun(ExternalAIPaste, beatReply(unitJSON("dán", "b1.1", "b1.2", "b1.3")))
	if _, err := f.desk.GenerateBeat(WithExternalAI(context.Background(), run), project, "b1", ""); err != nil {
		t.Fatal(err)
	}
	if units := f.beatUnits("b1"); len(units) != 1 || units[0].NarrationSummary != "dán" {
		t.Errorf("paste mode stores the pasted reply once it fits: %+v", units)
	}
}

func reviewReply(items ...string) string {
	return `{"findings":[` + strings.Join(items, ",") + `],"notes":["ổn"]}`
}

func finding(group, unit, field, text, sameAs string) string {
	return fmt.Sprintf(`{"group":%q,"unit_key":%q,"field":%q,"text":%q,"detail":"d","fix":"sửa","same_as":%q}`, group, unit, field, text, sameAs)
}

func TestPlanReview_FoldsEachRoundIntoTheLast(t *testing.T) {
	ctx := context.Background()
	f := newAIFixture(t)
	f.queue(domain.PromptRolePlanReview,
		reviewReply(finding("clarity", "VP-001", "communication_goal", "mục tiêu mơ hồ", ""), finding("accuracy", "VP-002", "", "dễ hiểu lầm", "")),
		reviewReply(finding("clarity", "VP-001", "communication_goal", "mục tiêu vẫn mơ hồ", "f1"), finding("feasibility", "VP-003", "visual_approach", "khó dựng", "")))
	st, err := f.desk.Review(ctx, project, "kỹ nhé")
	if err != nil {
		t.Fatal(err)
	}
	call := f.llm.calls[0]
	if call.Role != domain.PromptRolePlanReview || call.Step != PlanStep || strings.Contains(call.System, "{{") {
		t.Errorf("the review turn is of the review role with every variable filled: %+v\n%s", call.Role, call.System)
	}
	for _, want := range []string{"VP-002 · tối thiểu 60.0 giây", "b1.2 Có thể B", "(lần rà soát đầu)", "kỹ nhé", "Mỗi câu lời thoại thuộc đúng một unit: phủ 4/4 câu", "Giả thuyết và ngộ nhận đều có ghi chú rủi ro"} {
		if !strings.Contains(call.System, want) {
			t.Errorf("the review prompt lacks %q", want)
		}
	}
	if st.Review.Rounds != 1 || len(st.Review.Findings) != 2 || st.Review.OpenCount != 2 || st.Review.Findings[0].GroupLabel != "Rõ ràng" {
		t.Fatalf("first round: %+v", st.Review)
	}

	st, err = f.desk.Review(ctx, project, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(f.llm.calls[1].System, "f1 · clarity · VP-001: mục tiêu mơ hồ") || !strings.Contains(f.llm.calls[1].System, "f2 · accuracy · VP-002") {
		t.Errorf("the second round is told the findings still standing")
	}
	byID := map[string]string{}
	for _, fd := range st.Review.Findings {
		byID[fd.ID] = fd.State + ":" + fd.Text
	}
	if st.Review.Rounds != 2 || byID["f1"] != "open:mục tiêu vẫn mơ hồ" || !strings.HasPrefix(byID["f2"], "fixed:") || !strings.HasPrefix(byID["f3"], "open:") {
		t.Errorf("a finding reported again stays, one not reported is fixed, a new one is added: %v", byID)
	}
}

func TestPlanReview_RefusesAReplyThatDoesNotFit(t *testing.T) {
	for name, reply := range map[string]string{
		"a unit not in the plan":         reviewReply(finding("clarity", "VP-099", "", "x", "")),
		"a field nobody can write":       reviewReply(finding("clarity", "VP-001", "unit_id", "x", "")),
		"a group the review has not":     reviewReply(finding("flow", "VP-001", "", "x", "")),
		"a finding with no text":         reviewReply(finding("clarity", "VP-001", "", " ", "")),
		"a repeat of an unknown finding": reviewReply(finding("clarity", "VP-001", "", "x", "f7")),
	} {
		f := newAIFixture(t)
		f.queue(domain.PromptRolePlanReview, reply)
		if _, err := f.desk.Review(context.Background(), project, ""); !errors.Is(err, domain.ErrUnreadableReply) {
			t.Errorf("%s: want an unreadable reply, got %v", name, err)
		}
		if _, ok := f.store.units[domain.UnitKindPlanReview+"/main"]; ok {
			t.Errorf("%s: nothing may be stored", name)
		}
	}
	f := newAIFixture(t)
	for _, k := range []string{"b1", "b2"} {
		delete(f.store.units, domain.UnitKindPlan+"/"+k)
	}
	if _, err := f.desk.Review(context.Background(), project, ""); !errors.Is(err, domain.ErrUnitNotWritten) || len(f.llm.calls) != 0 {
		t.Errorf("a plan with no unit has nothing to review: %v", err)
	}
}

func TestPlanReview_ItsFindingsApplyToTheUnitTheyName(t *testing.T) {
	ctx := context.Background()
	f := newAIFixture(t)
	f.queue(domain.PromptRolePlanReview, reviewReply(finding("clarity", "VP-003", "communication_goal", "mục tiêu mơ hồ", "")))
	if _, err := f.desk.Review(ctx, project, ""); err != nil {
		t.Fatal(err)
	}
	st := f.must(f.desk.ApplyFinding(ctx, project, "f1"))
	if u := f.unit(st, "VP-003"); u.CommunicationGoal != "sửa" || strings.Join(u.CreatorFields, ",") != "communication_goal" || u.Status != domain.UnitPending {
		t.Errorf("applying writes the fix, makes the field the Creator's and puts the unit back to review: %+v", u)
	}
}

func TestPlanSetModel(t *testing.T) {
	ctx := context.Background()
	f := newAIFixture(t)
	st := f.must(f.desk.State(ctx, project))
	if st.Model != "" || st.DefaultModel != "ollama" || len(st.Models) != 2 {
		t.Errorf("a video with no model of its own shows the step default: %+v", st)
	}
	if _, err := f.desk.SetModel(ctx, project, "nope"); !errors.Is(err, ErrUnknownModel) {
		t.Errorf("a model outside the catalog: %v", err)
	}
	if st = f.must(f.desk.SetModel(ctx, project, "deepseek-chat")); st.Model != "deepseek-chat" {
		t.Errorf("the model is saved: %+v", st)
	}
	f.queue(domain.PromptRolePlanBeat, beatReply(unitJSON("một", "b1.1", "b1.2", "b1.3")))
	if _, err := f.desk.GenerateBeat(ctx, project, "b1", ""); err != nil {
		t.Fatal(err)
	}
	if f.llm.calls[0].Model != "deepseek-chat" {
		t.Errorf("the turn runs on the video's model, got %q", f.llm.calls[0].Model)
	}
	if st = f.must(f.desk.SetModel(ctx, project, " ")); st.Model != "" {
		t.Errorf("a blank model goes back to the step default: %+v", st)
	}
	if _, err := newPlanFixture(t).desk.SetModel(ctx, project, ""); !errors.Is(err, ErrLLMNotConfigured) {
		t.Errorf("no AI wired: %v", err)
	}
}
