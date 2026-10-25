package application

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"authoring/internal/domain"
)

const ideaReplyJSON = `{"question":"AI câu hỏi","why":"AI vì sao","known":"AI đã biết","unknown":"AI chưa biết"}`

type ideaFixture struct {
	units   *briefUnits
	store   *briefStore
	project *briefProject
	llm     *briefLLM
	uc      *IdeaUseCase
}

func newIdeaFixture() *ideaFixture {
	f := &ideaFixture{
		units: newBriefUnits(),
		store: &briefStore{topic: "Vì sao trong quả sung có xác ong?", duration: domain.Duration5to7},
		project: &briefProject{
			project: domain.Project{ContentLanguage: domain.LanguageVietnamese, VideoOutputMode: domain.ModeLongOnly},
			status:  domain.StatusDraft,
		},
		llm: &briefLLM{reply: "```json\n" + ideaReplyJSON + "\n```"},
	}
	f.uc = NewIdeaUseCase(f.units, f.store, f.project, f.project, briefPrompts{}, f.llm, 4000)
	return f
}

func ideaOf(t *testing.T, u domain.ReviewUnit) domain.Idea {
	t.Helper()
	var i domain.Idea
	if err := json.Unmarshal(u.Content, &i); err != nil {
		t.Fatalf("stored idea is not JSON: %v", err)
	}
	return i
}

func TestGenerateWritesTheIdeaForReview(t *testing.T) {
	f := newIdeaFixture()
	unit, err := f.uc.Generate(context.Background(), "p1", "")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if unit.Status != domain.UnitPending || unit.RedoCount != 0 {
		t.Errorf("a first idea waits for review with no redo counted, got status %q redo %d", unit.Status, unit.RedoCount)
	}
	if i := ideaOf(t, unit); i.Question != "AI câu hỏi" || i.Unknown != "AI chưa biết" {
		t.Errorf("idea stored wrong: %+v", i)
	}
	for _, want := range []string{"Vì sao trong quả sung có xác ong?", "tiếng Việt"} {
		if !strings.Contains(f.llm.systems[0], want) {
			t.Errorf("the idea prompt lacks %q", want)
		}
	}
}

func TestIdeaRunsOnTheBriefsModelSlot(t *testing.T) {
	f := newIdeaFixture()
	f.store.models = domain.AuthoringStepModels{Brief: "ollama", Story: "zai-org/glm-5.3-flash"}
	if _, err := f.uc.Generate(context.Background(), "p1", ""); err != nil {
		t.Fatal(err)
	}
	if f.llm.models[0] != "ollama" {
		t.Errorf("the idea must run on the model picked for the brief step, got %q", f.llm.models[0])
	}
}

func TestIdeaRedoKeepsTheCreatorsFieldsAndCountsTheRedo(t *testing.T) {
	f := newIdeaFixture()
	ctx := context.Background()
	if _, err := f.uc.Generate(ctx, "p1", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := f.uc.Edit(ctx, "p1", domain.IdeaUnitKey, map[string]string{"why": "Creator vì sao"}); err != nil {
		t.Fatal(err)
	}
	unit, err := f.uc.Redo(ctx, "p1", domain.IdeaUnitKey, "nhấn mạnh ngộ nhận")
	if err != nil {
		t.Fatalf("redo: %v", err)
	}
	if i := ideaOf(t, unit); i.Why != "Creator vì sao" || i.Question != "AI câu hỏi" {
		t.Errorf("redo must keep the Creator's field and take the AI's others, got %+v", i)
	}
	if unit.RedoCount != 1 {
		t.Errorf("one redo must be counted, got %d", unit.RedoCount)
	}
	if !strings.Contains(f.llm.systems[1], "nhấn mạnh ngộ nhận") || !strings.Contains(f.llm.systems[1], "- why: Creator vì sao") {
		t.Error("the redo prompt must carry the Creator's note and the field they wrote")
	}
}

func TestIdeaGenerateStoresNothingWhenTheReplyMissesAField(t *testing.T) {
	f := newIdeaFixture()
	f.llm.reply = `{"question":"Q"}`
	if _, err := f.uc.Generate(context.Background(), "p1", ""); !errors.Is(err, domain.ErrIdeaReplyUnusable) {
		t.Fatalf("an incomplete reply must fail the run, got %v", err)
	}
	if f.units.saves != 0 {
		t.Error("an unusable reply must not be stored")
	}
}

func TestIdeaGenerateWithoutAModelSaysSo(t *testing.T) {
	f := newIdeaFixture()
	uc := NewIdeaUseCase(f.units, f.store, f.project, f.project, briefPrompts{}, nil, 4000)
	if _, err := uc.Generate(context.Background(), "p1", ""); !errors.Is(err, ErrLLMNotConfigured) {
		t.Fatalf("no provider must be reported, got %v", err)
	}
}

func TestIdeaEditRejectsAnUnknownField(t *testing.T) {
	f := newIdeaFixture()
	if _, err := f.uc.Edit(context.Background(), "p1", domain.IdeaUnitKey, map[string]string{"tone": "x"}); !errors.Is(err, domain.ErrInvalidUnitEdit) {
		t.Fatalf("a brief field is not an idea field, got %v", err)
	}
}

func TestIdeaCannotBeApprovedWithAnEmptyField(t *testing.T) {
	f := newIdeaFixture()
	ctx := context.Background()
	units := NewReviewUnitsUseCase(f.units, f.project).Register(domain.UnitKindIdea, f.uc)
	if _, err := f.uc.Edit(ctx, "p1", domain.IdeaUnitKey, map[string]string{"question": "Câu của tôi"}); err != nil {
		t.Fatal(err)
	}
	if _, err := units.Approve(ctx, "p1", domain.UnitKindIdea, domain.IdeaUnitKey); !errors.Is(err, ErrIdeaIncomplete) {
		t.Fatalf("an idea with empty fields cannot be approved, got %v", err)
	}
}

func TestIdeaImpactNamesTheBriefWhenOneExists(t *testing.T) {
	f := newIdeaFixture()
	ctx := context.Background()
	items, err := f.uc.Impact(ctx, "p1", domain.IdeaUnitKey)
	if err != nil || len(items) != 0 {
		t.Fatalf("nothing depends on the idea before the brief is written, got %v, %v", items, err)
	}
	f.units.units["p1|brief|main"] = domain.ReviewUnit{ProjectID: "p1", Kind: domain.UnitKindBrief, Key: domain.BriefUnitKey, Content: []byte("{}")}
	items, err = f.uc.Impact(ctx, "p1", domain.IdeaUnitKey)
	if err != nil || len(items) != 1 || !strings.Contains(items[0], "Creative brief") {
		t.Fatalf("a written brief is named in the impact, got %v, %v", items, err)
	}
}
