package application

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"authoring/internal/domain"
)

// storyArchetypes is an in-memory Kiểu video table.
type storyArchetypes struct {
	list    []domain.VideoArchetype
	created []domain.VideoArchetype
}

func (a *storyArchetypes) List(context.Context) ([]domain.VideoArchetype, error) {
	return append([]domain.VideoArchetype{}, a.list...), nil
}
func (a *storyArchetypes) Create(_ context.Context, v domain.VideoArchetype) (domain.VideoArchetype, error) {
	v.ID = "row-" + v.Code
	a.created = append(a.created, v)
	a.list = append(a.list, v)
	return v, nil
}

type storyFixture struct {
	units      *briefUnits
	store      *researchStore
	project    *briefProject
	llm        *briefLLM
	archetypes *storyArchetypes
	uc         *StoryDeskUseCase
}

const storyReplyFull = `{"question":"Vì sao ong chết trong quả sung?","promise":"Bạn sẽ biết vì sao.","chain":[{"text":"Sung là chùm hoa kín.","facts":["F01"]},{"text":"Ong chui vào và mất cánh.","facts":["F02","F01"]}],"payoff":"Ong là cái giá của một quan hệ.","payoff_facts":["F02"],"logline":"Một con ong hy sinh vì quả sung.","summary":"Mở từ đĩa sung, đi vào lòng quả.","used_facts":["F03"],"facts":[]}`

func newStoryFixture() *storyFixture {
	units := newBriefUnits()
	f := &storyFixture{
		units:      units,
		store:      &researchStore{briefUnits: units, topic: "Vì sao trong quả sung có xác ong?", duration: domain.Duration5to7},
		project:    &briefProject{project: domain.Project{ContentLanguage: domain.LanguageVietnamese, VideoOutputMode: domain.ModeLongOnly}, status: domain.StatusDraft},
		llm:        &briefLLM{reply: storyReplyFull},
		archetypes: &storyArchetypes{list: []domain.VideoArchetype{{ID: "system-F", Code: "HÀNH-TRÌNH", Name: "Câu chuyện khám phá", WhenToUse: "khám phá", Playbook: "kể theo mốc"}}},
	}
	f.uc = NewStoryDeskUseCase(f.store, f.project, f.project, briefPrompts{}, f.archetypes, f.llm, nil, 4000)
	return f
}

func (f *storyFixture) save(t *testing.T, kind, key string, status domain.UnitStatus, content any, creatorFields ...string) {
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

// open approves the brief and the research step and seeds three facts.
func (f *storyFixture) open(t *testing.T) {
	t.Helper()
	f.save(t, domain.UnitKindIdea, domain.IdeaUnitKey, domain.UnitApproved, domain.Idea{Question: "Vì sao ong chết?", Why: "w", Known: "k", Unknown: "u"})
	f.save(t, domain.UnitKindBrief, domain.BriefUnitKey, domain.UnitApproved, domain.CreativeBrief{
		Audience: "a", Know: "k", Understand: "u", Feel: "f", Tone: "t", OutOfScope: "o",
	})
	f.save(t, domain.UnitKindResearch, domain.ResearchUnitKey, domain.UnitApproved, domain.ResearchGate{})
	f.save(t, domain.UnitKindFact, "F01", domain.UnitPending, domain.Fact{ClaimVI: "Sung là chùm hoa kín.", Type: domain.FactTypeFact, Keep: true, Must: true, Link: domain.FactLink{State: domain.LinkOK}})
	f.save(t, domain.UnitKindFact, "F02", domain.UnitPending, domain.Fact{ClaimVI: "Ong mẹ chết trong quả.", Type: domain.FactTypeFact, Keep: true, Link: domain.FactLink{State: domain.LinkOK}})
	f.save(t, domain.UnitKindFact, "F03", domain.UnitPending, domain.Fact{ClaimVI: "Ficin phân giải xác ong.", Type: domain.FactTypeHypothesis, Keep: true, Link: domain.FactLink{State: domain.LinkOK}})
	f.save(t, domain.UnitKindFact, "F04", domain.UnitPending, domain.Fact{ClaimVI: "Đã bỏ.", Type: domain.FactTypeFact, Keep: false})
}

func (f *storyFixture) field(t *testing.T, key string) (domain.ReviewUnit, domain.StoryField) {
	t.Helper()
	u, err := f.units.GetUnit(context.Background(), "p1", domain.UnitKindStoryField, key)
	if err != nil {
		t.Fatalf("field %s: %v", key, err)
	}
	var sf domain.StoryField
	if err := json.Unmarshal(u.Content, &sf); err != nil {
		t.Fatal(err)
	}
	return u, sf
}

func (f *storyFixture) gateStatus(t *testing.T) domain.UnitStatus {
	t.Helper()
	u, err := f.units.GetUnit(context.Background(), "p1", domain.UnitKindStory, domain.StoryUnitKey)
	if err != nil {
		t.Fatalf("gate: %v", err)
	}
	return u.Status
}

func TestGenerateFrameWritesSixFieldsAndGoesToReview(t *testing.T) {
	f := newStoryFixture()
	f.open(t)
	if err := f.uc.GenerateFrame(context.Background(), "p1", ""); err != nil {
		t.Fatalf("GenerateFrame: %v", err)
	}
	state, err := f.uc.State(context.Background(), "p1")
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Fields) != 6 || state.Status != domain.UnitPending {
		t.Fatalf("state = %+v, want six fields and a story waiting for review", state)
	}
	chain := state.Fields[2]
	if chain.Key != "chain" || chain.Text != "Sung là chùm hoa kín. [F01]\nOng chui vào và mất cánh. [F02, F01]" || strings.Join(chain.Facts, ",") != "F01,F02" {
		t.Errorf("chain = %+v", chain)
	}
	if chain.Mine || chain.Stale {
		t.Errorf("an AI field is not the Creator's and not stale: %+v", chain)
	}
	used := map[string]StoryFactView{}
	for _, v := range state.Facts {
		used[v.ID] = v
	}
	if len(used) != 3 {
		t.Fatalf("facts = %+v, want the three kept facts only", state.Facts)
	}
	if !used["F01"].Used || !used["F01"].Must || !used["F02"].Cited || !used["F03"].Used || used["F03"].Cited {
		t.Errorf("fact use wrong: %+v", used)
	}
	if !strings.Contains(f.llm.systems[0], "F01 [fact · bắt buộc nói]") || strings.Contains(f.llm.systems[0], "F04") {
		t.Errorf("the prompt must list the kept facts with their tags, got %q", f.llm.systems[0])
	}
}

func TestGenerateFrameKeepsFieldsTheCreatorWrote(t *testing.T) {
	f := newStoryFixture()
	f.open(t)
	f.save(t, domain.UnitKindStoryField, "logline", domain.UnitPending, domain.StoryField{Text: "Logline của tôi", Facts: []string{}}, domain.TextFieldName)
	if err := f.uc.GenerateFrame(context.Background(), "p1", ""); err != nil {
		t.Fatalf("GenerateFrame: %v", err)
	}
	u, sf := f.field(t, "logline")
	if sf.Text != "Logline của tôi" || !u.HasCreatorField(domain.TextFieldName) {
		t.Errorf("logline = %q mine=%v, want the Creator's text kept", sf.Text, u.CreatorFields)
	}
	if !strings.Contains(f.llm.systems[0], "- logline: Logline của tôi") {
		t.Errorf("the prompt must carry the Creator's field, got %q", f.llm.systems[0])
	}
}

func TestGenerateFrameRefusesWhenTheStepIsNotOpen(t *testing.T) {
	f := newStoryFixture()
	if err := f.uc.GenerateFrame(context.Background(), "p1", ""); !errors.Is(err, ErrStoryNeedsBrief) {
		t.Errorf("err = %v, want the brief to be required first", err)
	}
	f.save(t, domain.UnitKindIdea, domain.IdeaUnitKey, domain.UnitApproved, domain.Idea{Question: "Vì sao ong chết?", Why: "w", Known: "k", Unknown: "u"})
	f.save(t, domain.UnitKindBrief, domain.BriefUnitKey, domain.UnitApproved, domain.CreativeBrief{Audience: "q"})
	if err := f.uc.GenerateFrame(context.Background(), "p1", ""); !errors.Is(err, ErrStoryNeedsResearch) {
		t.Errorf("err = %v, want the research step to be required next", err)
	}
	if len(f.llm.systems) != 0 {
		t.Error("the model must not be called while the step is not open")
	}
}

func TestGenerateFrameStoresNothingFromAnUnusableReply(t *testing.T) {
	f := newStoryFixture()
	f.open(t)
	f.llm.reply = strings.Replace(storyReplyFull, `"payoff_facts":["F02"]`, `"payoff_facts":["F77"]`, 1)
	if err := f.uc.GenerateFrame(context.Background(), "p1", ""); !errors.Is(err, domain.ErrStoryReplyUnusable) {
		t.Fatalf("err = %v, want ErrStoryReplyUnusable", err)
	}
	units, _ := f.units.ListUnits(context.Background(), "p1", domain.UnitKindStoryField)
	if len(units) != 0 {
		t.Errorf("%d fields stored from an unusable reply", len(units))
	}
}

func TestGenerateFrameListsFactsWhenResearchWasSkipped(t *testing.T) {
	f := newStoryFixture()
	f.save(t, domain.UnitKindIdea, domain.IdeaUnitKey, domain.UnitApproved, domain.Idea{Question: "Vì sao ong chết?", Why: "w", Known: "k", Unknown: "u"})
	f.save(t, domain.UnitKindBrief, domain.BriefUnitKey, domain.UnitApproved, domain.CreativeBrief{Audience: "q"})
	f.save(t, domain.UnitKindResearch, domain.ResearchUnitKey, domain.UnitPending, domain.ResearchGate{Skipped: true})
	f.llm.reply = `{"question":"q","promise":"p","chain":[{"text":"s","facts":["F01"]}],"payoff":"x","payoff_facts":["F01"],"logline":"l","summary":"u","used_facts":[],"facts":[{"id":"F01","claim_vi":"Sung là chùm hoa","type":"fact","note":""}]}`
	if err := f.uc.GenerateFrame(context.Background(), "p1", ""); err != nil {
		t.Fatalf("GenerateFrame: %v", err)
	}
	state, _ := f.uc.State(context.Background(), "p1")
	if len(state.Facts) != 1 || state.Facts[0].ID != "F01" || state.Facts[0].Link != domain.LinkNone || !state.ResearchSkipped {
		t.Errorf("facts = %+v, want the listed fact without a source", state.Facts)
	}
	if !strings.Contains(f.llm.systems[0], "tự liệt kê các dữ kiện") || !strings.Contains(f.llm.systems[0], `"facts":[{"id":"F01"`) {
		t.Error("the prompt must tell the model to list the facts itself, and its sample answer must list some")
	}
}

func TestGenerateFrameDoesNotOverwriteAFactTheCreatorHas(t *testing.T) {
	f := newStoryFixture()
	f.save(t, domain.UnitKindIdea, domain.IdeaUnitKey, domain.UnitApproved, domain.Idea{Question: "Vì sao ong chết?", Why: "w", Known: "k", Unknown: "u"})
	f.save(t, domain.UnitKindBrief, domain.BriefUnitKey, domain.UnitApproved, domain.CreativeBrief{Audience: "q"})
	f.save(t, domain.UnitKindResearch, domain.ResearchUnitKey, domain.UnitPending, domain.ResearchGate{Skipped: true})
	f.save(t, domain.UnitKindFact, "F01", domain.UnitPending, domain.Fact{ClaimVI: "Đã bỏ", Type: domain.FactTypeFact, Keep: false})
	f.llm.reply = `{"question":"q","promise":"p","chain":[{"text":"s","facts":["F01"]}],"payoff":"x","payoff_facts":[],"logline":"l","summary":"u","used_facts":[],"facts":[{"id":"F01","claim_vi":"Khác","type":"fact","note":""}]}`
	if err := f.uc.GenerateFrame(context.Background(), "p1", ""); !errors.Is(err, domain.ErrStoryReplyUnusable) {
		t.Errorf("err = %v, want a listed code that is already taken refused", err)
	}
}

func TestEditFieldMarksItMineAndReopensTheStory(t *testing.T) {
	f := newStoryFixture()
	f.open(t)
	if err := f.uc.GenerateFrame(context.Background(), "p1", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := NewReviewUnitsUseCase(f.store, f.project).Register(domain.UnitKindStory, f.uc.GateUnitKind()).Approve(context.Background(), "p1", domain.UnitKindStory, domain.StoryUnitKey); err != nil {
		t.Fatalf("approve: %v", err)
	}
	kind := f.uc.FieldUnitKind()
	u, err := kind.Edit(context.Background(), "p1", "payoff", map[string]string{"text": "Kết của tôi [F03]"})
	if err != nil {
		t.Fatalf("Edit: %v", err)
	}
	if !u.HasCreatorField(domain.TextFieldName) {
		t.Error("an edited field must become the Creator's")
	}
	if _, sf := f.field(t, "payoff"); sf.Text != "Kết của tôi [F03]" || strings.Join(sf.Facts, ",") != "F03" {
		t.Errorf("payoff = %+v", sf)
	}
	if f.gateStatus(t) != domain.UnitPending {
		t.Error("editing a field must send an approved story back to review")
	}
}

func TestEditFieldRefusesWhatCannotBeStored(t *testing.T) {
	f := newStoryFixture()
	f.open(t)
	kind := f.uc.FieldUnitKind()
	if _, err := kind.Edit(context.Background(), "p1", "payoff", map[string]string{"text": "Kết [F09]"}); !errors.Is(err, domain.ErrInvalidUnitEdit) {
		t.Errorf("err = %v, want a citation of an unknown fact refused", err)
	}
	if _, err := kind.Edit(context.Background(), "p1", "payoff", map[string]string{"text": "Kết [F04]"}); !errors.Is(err, domain.ErrInvalidUnitEdit) {
		t.Errorf("err = %v, want a citation of a dropped fact refused", err)
	}
	if _, err := kind.Edit(context.Background(), "p1", "nothing", map[string]string{"text": "x"}); !errors.Is(err, domain.ErrUnitNotFound) {
		t.Errorf("err = %v, want an unknown field refused", err)
	}
	if _, err := kind.Edit(context.Background(), "p1", "logline", map[string]string{"other": "x"}); !errors.Is(err, domain.ErrInvalidUnitEdit) {
		t.Errorf("err = %v, want an unknown form field refused", err)
	}
}

func TestEmptyingAStoryFieldHandsItBackToTheAI(t *testing.T) {
	f := newStoryFixture()
	f.open(t)
	kind := f.uc.FieldUnitKind()
	if _, err := kind.Edit(context.Background(), "p1", "logline", map[string]string{"text": "Của tôi"}); err != nil {
		t.Fatal(err)
	}
	u, err := kind.Edit(context.Background(), "p1", "logline", map[string]string{"text": "  "})
	if err != nil {
		t.Fatal(err)
	}
	if u.HasCreatorField(domain.TextFieldName) {
		t.Error("an emptied field goes back to the AI")
	}
}

func TestRedoWritesOnlyThatFieldAndReleasesIt(t *testing.T) {
	f := newStoryFixture()
	f.open(t)
	if err := f.uc.GenerateFrame(context.Background(), "p1", ""); err != nil {
		t.Fatal(err)
	}
	kind := f.uc.FieldUnitKind()
	if _, err := kind.Edit(context.Background(), "p1", "logline", map[string]string{"text": "Của tôi"}); err != nil {
		t.Fatal(err)
	}
	_, beforePromise := f.field(t, "promise")
	f.llm.reply = `{"logline":"Logline mới của AI."}`
	u, err := kind.Redo(context.Background(), "p1", "logline", "ngắn hơn")
	if err != nil {
		t.Fatalf("Redo: %v", err)
	}
	if _, sf := f.field(t, "logline"); sf.Text != "Logline mới của AI." || u.HasCreatorField(domain.TextFieldName) {
		t.Errorf("logline = %q mine=%v, want the AI's text and no longer the Creator's", sf.Text, u.CreatorFields)
	}
	if _, after := f.field(t, "promise"); after.Text != beforePromise.Text {
		t.Error("redoing one field must leave the others as they were")
	}
	last := f.llm.systems[len(f.llm.systems)-1]
	if !strings.Contains(last, "ngắn hơn") || !strings.Contains(last, "logline") {
		t.Errorf("the prompt must carry the note and the field asked for, got %q", last)
	}
}

func TestGateApprovalNeedsAllSixFields(t *testing.T) {
	f := newStoryFixture()
	f.open(t)
	units := NewReviewUnitsUseCase(f.store, f.project).Register(domain.UnitKindStory, f.uc.GateUnitKind())
	if _, err := units.Approve(context.Background(), "p1", domain.UnitKindStory, domain.StoryUnitKey); !errors.Is(err, ErrStoryIncomplete) {
		t.Fatalf("err = %v, want an empty story refused", err)
	}
	if err := f.uc.GenerateFrame(context.Background(), "p1", ""); err != nil {
		t.Fatal(err)
	}
	approved, err := units.Approve(context.Background(), "p1", domain.UnitKindStory, domain.StoryUnitKey)
	if err != nil || approved.Status != domain.UnitApproved {
		t.Fatalf("approve = %+v %v, want the full story approved", approved, err)
	}
	if _, err := f.uc.GateUnitKind().Redo(context.Background(), "p1", domain.StoryUnitKey, "x"); err == nil {
		t.Error("the gate cannot be redone by the AI")
	}
}

func TestSetFactUseTurnsFactsOnAndOffAndReopens(t *testing.T) {
	f := newStoryFixture()
	f.open(t)
	if err := f.uc.GenerateFrame(context.Background(), "p1", ""); err != nil {
		t.Fatal(err)
	}
	units := NewReviewUnitsUseCase(f.store, f.project).Register(domain.UnitKindStory, f.uc.GateUnitKind())
	if _, err := units.Approve(context.Background(), "p1", domain.UnitKindStory, domain.StoryUnitKey); err != nil {
		t.Fatal(err)
	}
	if err := f.uc.SetFactUse(context.Background(), "p1", "F01", false); !errors.Is(err, domain.ErrInvalidUnitEdit) {
		t.Errorf("err = %v, want a must-say fact kept in use", err)
	}
	if err := f.uc.SetFactUse(context.Background(), "p1", "F04", true); !errors.Is(err, domain.ErrUnitNotFound) {
		t.Errorf("err = %v, want a dropped fact not found", err)
	}
	if err := f.uc.SetFactUse(context.Background(), "p1", "F03", false); err != nil {
		t.Fatal(err)
	}
	state, _ := f.uc.State(context.Background(), "p1")
	for _, v := range state.Facts {
		if v.ID == "F03" && v.Used {
			t.Error("F03 was turned off and nothing cites it, so it is not used")
		}
	}
	if state.Status != domain.UnitPending {
		t.Error("changing the facts used must send an approved story back to review")
	}
}

func TestFactDependentsNameTheFieldsThatCiteIt(t *testing.T) {
	f := newStoryFixture()
	f.open(t)
	if err := f.uc.GenerateFrame(context.Background(), "p1", ""); err != nil {
		t.Fatal(err)
	}
	items, err := f.uc.FactDependents()(context.Background(), "p1", "F02")
	if err != nil || len(items) != 2 || !strings.Contains(items[0], "chuỗi nhân quả") {
		t.Errorf("items = %v %v, want the chain and the payoff to cite F02", items, err)
	}
	items, _ = f.uc.FactDependents()(context.Background(), "p1", "F03")
	if len(items) != 1 || !strings.Contains(items[0], "dùng F03") {
		t.Errorf("items = %v, want F03 reported as used", items)
	}
}

const approachesReply = `{"approaches":[` +
	`{"code":"HÀNH-TRÌNH","name":"Hành trình","angle":"đi vào quả","why":"F01 cho cú lật","refs":["F01"],"open":"Quả sung..."},` +
	`{"code":"VÒNG-ĐỜI","name":"Vòng đời","angle":"theo ong mẹ","why":"F02","refs":["F02"],"open":"Câu chuyện...","when_to_use":"chu trình khép kín","playbook":"theo một cá thể"}]}`

func TestGenerateApproachesKeepsTheChosenOne(t *testing.T) {
	f := newStoryFixture()
	f.open(t)
	f.llm.reply = approachesReply
	views, err := f.uc.GenerateApproaches(context.Background(), "p1", "")
	if err != nil || len(views) != 2 || views[0].Key != "A1" || views[1].Key != "A2" || !views[1].IsNew {
		t.Fatalf("views = %+v %v, want A1 from the table and A2 a new kind", views, err)
	}
	if err := f.uc.SelectApproach(context.Background(), "p1", "A2"); err != nil {
		t.Fatal(err)
	}
	views, err = f.uc.GenerateApproaches(context.Background(), "p1", "thử cách khác")
	if err != nil {
		t.Fatal(err)
	}
	keys := []string{}
	for _, v := range views {
		keys = append(keys, v.Key)
		if v.Key == "A2" && !v.Selected {
			t.Error("the chosen approach must stay chosen")
		}
	}
	if strings.Join(keys, ",") != "A2,A3,A4" {
		t.Errorf("keys = %v, want the chosen A2 kept and two new ones", keys)
	}
	if !strings.Contains(f.llm.systems[1], "thử cách khác") || !strings.Contains(f.llm.systems[0], "HÀNH-TRÌNH: Câu chuyện khám phá") {
		t.Error("the prompt must carry the note and the Kiểu video table")
	}
}

func TestSelectApproachSendsAnApprovedStoryBackToReview(t *testing.T) {
	f := newStoryFixture()
	f.open(t)
	f.llm.reply = approachesReply
	if _, err := f.uc.GenerateApproaches(context.Background(), "p1", ""); err != nil {
		t.Fatal(err)
	}
	f.llm.reply = storyReplyFull
	if err := f.uc.GenerateFrame(context.Background(), "p1", ""); err != nil {
		t.Fatal(err)
	}
	units := NewReviewUnitsUseCase(f.store, f.project).Register(domain.UnitKindStory, f.uc.GateUnitKind())
	if _, err := units.Approve(context.Background(), "p1", domain.UnitKindStory, domain.StoryUnitKey); err != nil {
		t.Fatal(err)
	}
	if err := f.uc.SelectApproach(context.Background(), "p1", "A9"); !errors.Is(err, domain.ErrUnitNotFound) {
		t.Errorf("err = %v, want an unknown approach refused", err)
	}
	if err := f.uc.SelectApproach(context.Background(), "p1", "A1"); err != nil {
		t.Fatal(err)
	}
	if f.gateStatus(t) != domain.UnitPending {
		t.Error("changing the approach must send an approved story back to review")
	}
	if err := f.uc.GenerateFrame(context.Background(), "p1", ""); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(f.llm.systems[len(f.llm.systems)-1], "Cách kể của kiểu này:\nkể theo mốc") {
		t.Error("the frame prompt must carry the playbook of the chosen kind")
	}
}

func TestSaveApproachAsArchetypeOnlyForNewKinds(t *testing.T) {
	f := newStoryFixture()
	f.open(t)
	f.llm.reply = approachesReply
	if _, err := f.uc.GenerateApproaches(context.Background(), "p1", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := f.uc.SaveApproachAsArchetype(context.Background(), "p1", "A1"); !errors.Is(err, domain.ErrInvalidUnitEdit) {
		t.Errorf("err = %v, want a kind already in the table refused", err)
	}
	view, err := f.uc.SaveApproachAsArchetype(context.Background(), "p1", "A2")
	if err != nil || view.ArchetypeID != "row-VÒNG-ĐỜI" {
		t.Fatalf("view = %+v %v, want the new kind saved", view, err)
	}
	if len(f.archetypes.created) != 1 || f.archetypes.created[0].Playbook != "theo một cá thể" {
		t.Errorf("created = %+v, want the playbook the AI drafted", f.archetypes.created)
	}
	if _, err := f.uc.SaveApproachAsArchetype(context.Background(), "p1", "A2"); !errors.Is(err, domain.ErrInvalidUnitEdit) {
		t.Errorf("err = %v, want saving twice refused", err)
	}
}

func TestBriefImpactNamesTheStoryFrameWhenOneExists(t *testing.T) {
	f := newBriefFixture()
	ctx := context.Background()
	items, _ := f.uc.Impact(ctx, "p1", domain.BriefUnitKey)
	for _, it := range items {
		if strings.Contains(it, "Khung câu chuyện") {
			t.Fatalf("items = %v, want no story frame before one is written", items)
		}
	}
	raw, _ := json.Marshal(domain.StoryField{Text: "x"})
	if _, err := f.units.SaveUnit(ctx, domain.ReviewUnit{ProjectID: "p1", Kind: domain.UnitKindStoryField, Key: "question", Content: raw}); err != nil {
		t.Fatal(err)
	}
	items, err := f.uc.Impact(ctx, "p1", domain.BriefUnitKey)
	if err != nil || len(items) == 0 || !strings.Contains(strings.Join(items, "|"), "Khung câu chuyện") {
		t.Errorf("items = %v %v, want the story frame named", items, err)
	}
}

func TestStoryAIRunsUseTheModelChosenForTheStep(t *testing.T) {
	f := newStoryFixture()
	f.open(t)
	f.store.models = domain.AuthoringStepModels{StoryFrame: "ollama", Research: "other"}
	f.llm.reply = approachesReply
	if _, err := f.uc.GenerateApproaches(context.Background(), "p1", ""); err != nil {
		t.Fatal(err)
	}
	f.llm.reply = storyReplyFull
	if err := f.uc.GenerateFrame(context.Background(), "p1", ""); err != nil {
		t.Fatal(err)
	}
	f.llm.reply = `{"logline":"Mới."}`
	if _, err := f.uc.FieldUnitKind().Redo(context.Background(), "p1", "logline", "ngắn"); err != nil {
		t.Fatal(err)
	}
	if len(f.llm.models) != 3 {
		t.Fatalf("models = %v, want three calls", f.llm.models)
	}
	for _, m := range f.llm.models {
		if m != "ollama" {
			t.Errorf("model used = %q, want the one chosen for the story step", m)
		}
	}
}

func TestApproachesPromptTellsTheModelNotToCiteFactsWhenThereAreNone(t *testing.T) {
	f := newStoryFixture()
	f.save(t, domain.UnitKindIdea, domain.IdeaUnitKey, domain.UnitApproved, domain.Idea{Question: "Vì sao ong chết?", Why: "w", Known: "k", Unknown: "u"})
	f.save(t, domain.UnitKindBrief, domain.BriefUnitKey, domain.UnitApproved, domain.CreativeBrief{Audience: "q"})
	f.save(t, domain.UnitKindResearch, domain.ResearchUnitKey, domain.UnitPending, domain.ResearchGate{Skipped: true})
	f.llm.reply = `{"approaches":[{"code":"HÀNH-TRÌNH","name":"Hành trình","angle":"đi vào quả","why":"không có dữ kiện","refs":[],"open":"Quả sung..."}]}`
	if _, err := f.uc.GenerateApproaches(context.Background(), "p1", ""); err != nil {
		t.Fatalf("GenerateApproaches: %v", err)
	}
	if strings.Contains(f.llm.systems[0], `"refs":["F01"]`) || !strings.Contains(f.llm.systems[0], `"refs":[]`) {
		t.Error("the sample answer must not show a fact code when there is no fact to cite")
	}
	if !strings.Contains(f.llm.systems[0], "tuyệt đối không bịa mã dữ kiện") {
		t.Errorf("the prompt must forbid citing facts that do not exist, got %q", f.llm.systems[0])
	}
}
