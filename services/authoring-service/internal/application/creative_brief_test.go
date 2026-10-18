package application

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"sync"
	"testing"

	"authoring/internal/domain"
)

// briefUnits is an in-memory ReviewUnitPort, safe for the background source
// checks that write to it while a test reads.
type briefUnits struct {
	mu    sync.Mutex
	units map[string]domain.ReviewUnit
	saves int
}

func newBriefUnits() *briefUnits { return &briefUnits{units: map[string]domain.ReviewUnit{}} }

func (b *briefUnits) GetUnit(_ context.Context, projectID, kind, key string) (domain.ReviewUnit, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.get(projectID, kind, key)
}

// get reads a unit; the caller holds the lock.
func (b *briefUnits) get(projectID, kind, key string) (domain.ReviewUnit, error) {
	u, ok := b.units[projectID+"|"+kind+"|"+key]
	if !ok {
		return domain.ReviewUnit{}, domain.ErrUnitNotFound
	}
	u.Stale = u.InputRef != "" && u.InputSHA != b.sha(projectID, u.InputRef)
	return u, nil
}

// sha hashes the content of the unit named "kind:key", as the store does.
func (b *briefUnits) sha(projectID, ref string) string {
	u, ok := b.units[projectID+"|"+strings.Replace(ref, ":", "|", 1)]
	if !ok {
		return ""
	}
	sum := md5.Sum(u.Content)
	return hex.EncodeToString(sum[:])
}

func (b *briefUnits) UnitSHA(_ context.Context, projectID, kind, key string) (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.sha(projectID, kind+":"+key), nil
}

func (b *briefUnits) ListUnits(_ context.Context, projectID, kind string) ([]domain.ReviewUnit, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	var out []domain.ReviewUnit
	for k := range b.units {
		parts := strings.SplitN(k, "|", 3)
		if parts[0] == projectID && parts[1] == kind {
			u, _ := b.get(projectID, kind, parts[2])
			out = append(out, u)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if len(out[i].Key) != len(out[j].Key) {
			return len(out[i].Key) < len(out[j].Key)
		}
		return out[i].Key < out[j].Key
	})
	return out, nil
}

func (b *briefUnits) DeleteUnit(_ context.Context, projectID, kind, key string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.units, projectID+"|"+kind+"|"+key)
	return nil
}

func (b *briefUnits) SaveUnit(_ context.Context, u domain.ReviewUnit) (domain.ReviewUnit, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.saves++
	if u.CreatorFields == nil {
		u.CreatorFields = []string{}
	}
	b.units[u.ProjectID+"|"+u.Kind+"|"+u.Key] = u
	return u, nil
}

// briefStore is an in-memory BriefStorePort.
type briefStore struct {
	topic    string
	duration domain.TargetDuration
	link     StoryBriefLink
	models   domain.AuthoringStepModels
}

func (s *briefStore) GetAuthoringTopic(context.Context, string) (string, error) { return s.topic, nil }
func (s *briefStore) GetAuthoringModels(context.Context, string) (domain.AuthoringStepModels, error) {
	return s.models, nil
}
func (s *briefStore) GetTargetDuration(context.Context, string) (domain.TargetDuration, error) {
	return s.duration, nil
}
func (s *briefStore) SaveTargetDuration(_ context.Context, _ string, d domain.TargetDuration) error {
	s.duration = d
	return nil
}
func (s *briefStore) StoryBriefLink(context.Context, string) (StoryBriefLink, error) {
	return s.link, nil
}

// briefProject answers both ProjectReaderPort and AuthoringLockPort.
type briefProject struct {
	project domain.Project
	status  domain.ProjectStatus
}

func (p *briefProject) Get(context.Context, string) (*domain.Project, error) {
	project := p.project
	return &project, nil
}
func (p *briefProject) GetStatus(context.Context, string) (domain.ProjectStatus, error) {
	return p.status, nil
}

// briefPrompts serves the shipped prompts.
type briefPrompts struct{}

func (briefPrompts) GetActive(_ context.Context, role domain.PromptRole) (domain.Prompt, error) {
	for _, p := range domain.SystemPrompts() {
		if p.Role == role {
			return p, nil
		}
	}
	return domain.Prompt{}, errors.New("no prompt")
}

// briefLLM replies with a fixed text and keeps the prompts it was sent.
type briefLLM struct {
	reply   string
	err     error
	systems []string
	models  []string
}

func (l *briefLLM) Name() string { return "fake" }
func (l *briefLLM) Chat(_ context.Context, req ChatRequest) (ChatResult, error) {
	l.systems = append(l.systems, req.System)
	l.models = append(l.models, req.Model)
	return ChatResult{Content: l.reply}, l.err
}

const briefReplyJSON = `{"question":"AI câu hỏi","audience":"AI khán giả","know":"AI biết","understand":"AI hiểu","feel":"AI cảm thấy","tone":"AI giọng","out_of_scope":"AI ngoài phạm vi"}`

type briefFixture struct {
	units   *briefUnits
	store   *briefStore
	project *briefProject
	llm     *briefLLM
	uc      *CreativeBriefUseCase
}

func newBriefFixture() *briefFixture {
	f := &briefFixture{
		units: newBriefUnits(),
		store: &briefStore{topic: "Vì sao trong quả sung có xác ong?", duration: domain.Duration5to7},
		project: &briefProject{
			project: domain.Project{ContentLanguage: domain.LanguageVietnamese, VideoOutputMode: domain.ModeLongOnly},
			status:  domain.StatusDraft,
		},
		llm: &briefLLM{reply: "```json\n" + briefReplyJSON + "\n```"},
	}
	f.uc = NewCreativeBriefUseCase(f.units, f.store, f.project, f.project, briefPrompts{}, f.llm, nil, 4000)
	return f
}

func briefOf(t *testing.T, u domain.ReviewUnit) domain.CreativeBrief {
	t.Helper()
	var b domain.CreativeBrief
	if err := json.Unmarshal(u.Content, &b); err != nil {
		t.Fatalf("stored brief is not JSON: %v", err)
	}
	return b
}

func TestGenerateWritesTheBriefForReview(t *testing.T) {
	f := newBriefFixture()
	unit, err := f.uc.Generate(context.Background(), "p1", "")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if unit.Status != domain.UnitPending || unit.RedoCount != 0 {
		t.Errorf("a first brief waits for review with no redo counted, got status %q redo %d", unit.Status, unit.RedoCount)
	}
	if b := briefOf(t, unit); b.Question != "AI câu hỏi" || b.OutOfScope != "AI ngoài phạm vi" {
		t.Errorf("brief stored wrong: %+v", b)
	}
	system := f.llm.systems[0]
	for _, want := range []string{"Vì sao trong quả sung có xác ong?", "5–7 phút", "Video dài 16:9", "tiếng Việt"} {
		if !strings.Contains(system, want) {
			t.Errorf("the brief prompt lacks %q", want)
		}
	}
}

func TestGenerateRunsOnTheBriefsModel(t *testing.T) {
	f := newBriefFixture()
	f.store.models = domain.AuthoringStepModels{Brief: "ollama", Story: "zai-org/glm-5.3-flash"}
	if _, err := f.uc.Generate(context.Background(), "p1", ""); err != nil {
		t.Fatal(err)
	}
	if f.llm.models[0] != "ollama" {
		t.Errorf("the brief must run on the model picked for it, got %q", f.llm.models[0])
	}
}

func TestRedoKeepsTheCreatorsFieldsAndCountsTheRedo(t *testing.T) {
	f := newBriefFixture()
	ctx := context.Background()
	if _, err := f.uc.Generate(ctx, "p1", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := f.uc.Edit(ctx, "p1", domain.BriefUnitKey, map[string]string{"tone": "Creator giọng"}); err != nil {
		t.Fatal(err)
	}
	unit, err := f.uc.Redo(ctx, "p1", domain.BriefUnitKey, "nói về ong đực nhiều hơn")
	if err != nil {
		t.Fatalf("redo: %v", err)
	}
	if b := briefOf(t, unit); b.Tone != "Creator giọng" || b.Question != "AI câu hỏi" {
		t.Errorf("redo must keep the Creator's tone and take the AI's other fields, got %+v", b)
	}
	if unit.RedoCount != 1 {
		t.Errorf("one redo must be counted, got %d", unit.RedoCount)
	}
	redoPrompt := f.llm.systems[1]
	if !strings.Contains(redoPrompt, "nói về ong đực nhiều hơn") || !strings.Contains(redoPrompt, "- tone: Creator giọng") {
		t.Error("the redo prompt must carry the Creator's note and the field they wrote")
	}
}

func TestGenerateStoresNothingWhenTheReplyMissesAField(t *testing.T) {
	f := newBriefFixture()
	f.llm.reply = `{"question":"Q"}`
	if _, err := f.uc.Generate(context.Background(), "p1", ""); !errors.Is(err, domain.ErrBriefReplyUnusable) {
		t.Fatalf("an incomplete reply must fail the run, got %v", err)
	}
	if f.units.saves != 0 {
		t.Error("an unusable reply must not be stored")
	}
}

func TestGenerateWithoutAModelSaysSo(t *testing.T) {
	f := newBriefFixture()
	uc := NewCreativeBriefUseCase(f.units, f.store, f.project, f.project, briefPrompts{}, nil, nil, 4000)
	if _, err := uc.Generate(context.Background(), "p1", ""); !errors.Is(err, ErrLLMNotConfigured) {
		t.Fatalf("no provider must be reported, got %v", err)
	}
}

func TestEditBeforeAnyBriefCreatesOneTheCreatorWrote(t *testing.T) {
	f := newBriefFixture()
	unit, err := f.uc.Edit(context.Background(), "p1", domain.BriefUnitKey, map[string]string{"question": "  Câu của tôi  "})
	if err != nil {
		t.Fatalf("edit: %v", err)
	}
	if b := briefOf(t, unit); b.Question != "Câu của tôi" {
		t.Errorf("edited field stored wrong: %q", b.Question)
	}
	if !unit.HasCreatorField("question") || len(unit.CreatorFields) != 1 {
		t.Errorf("only the edited field becomes the Creator's, got %v", unit.CreatorFields)
	}
}

func TestEmptyingAFieldHandsItBackToTheAI(t *testing.T) {
	f := newBriefFixture()
	ctx := context.Background()
	if _, err := f.uc.Edit(ctx, "p1", domain.BriefUnitKey, map[string]string{"tone": "Creator giọng"}); err != nil {
		t.Fatal(err)
	}
	unit, err := f.uc.Edit(ctx, "p1", domain.BriefUnitKey, map[string]string{"tone": "  "})
	if err != nil {
		t.Fatal(err)
	}
	if unit.HasCreatorField("tone") {
		t.Fatalf("an emptied field must stop being the Creator's, got %v", unit.CreatorFields)
	}
	redone, err := f.uc.Redo(ctx, "p1", domain.BriefUnitKey, "")
	if err != nil {
		t.Fatal(err)
	}
	if b := briefOf(t, redone); b.Tone != "AI giọng" {
		t.Errorf("the redo must write the emptied field, got tone %q", b.Tone)
	}
}

func TestEditRejectsAnUnknownField(t *testing.T) {
	f := newBriefFixture()
	if _, err := f.uc.Edit(context.Background(), "p1", domain.BriefUnitKey, map[string]string{"logline": "x"}); !errors.Is(err, domain.ErrInvalidUnitEdit) {
		t.Fatalf("an unknown field must be refused, got %v", err)
	}
}

func TestApprovedBriefStaysApprovedUnlessAFieldIsEmptied(t *testing.T) {
	f := newBriefFixture()
	ctx := context.Background()
	units := NewReviewUnitsUseCase(f.units, f.project).Register(domain.UnitKindBrief, f.uc)
	if _, err := f.uc.Generate(ctx, "p1", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := units.Approve(ctx, "p1", domain.UnitKindBrief, domain.BriefUnitKey); err != nil {
		t.Fatalf("approve: %v", err)
	}
	unit, err := units.Edit(ctx, "p1", domain.UnitKindBrief, domain.BriefUnitKey, map[string]string{"feel": "Thán phục"})
	if err != nil || unit.Status != domain.UnitApproved {
		t.Fatalf("the Creator's own edit keeps the brief approved, got %q, %v", unit.Status, err)
	}
	unit, err = units.Edit(ctx, "p1", domain.UnitKindBrief, domain.BriefUnitKey, map[string]string{"feel": ""})
	if err != nil || unit.Status != domain.UnitPending {
		t.Fatalf("emptying a field sends the brief back to review, got %q, %v", unit.Status, err)
	}
	if _, err := units.Approve(ctx, "p1", domain.UnitKindBrief, domain.BriefUnitKey); !errors.Is(err, ErrBriefIncomplete) {
		t.Fatalf("a brief with an empty field cannot be approved, got %v", err)
	}
}

func TestUnitsRefuseChangesOnceTheProjectIsRendering(t *testing.T) {
	f := newBriefFixture()
	f.project.status = domain.StatusRendering
	units := NewReviewUnitsUseCase(f.units, f.project).Register(domain.UnitKindBrief, f.uc)
	_, err := units.Edit(context.Background(), "p1", domain.UnitKindBrief, domain.BriefUnitKey, map[string]string{"tone": "x"})
	if !errors.Is(err, domain.ErrInvalidStatus) {
		t.Fatalf("a rendering project's brief is locked, got %v", err)
	}
}

func TestUnitsRejectAnUnregisteredKind(t *testing.T) {
	f := newBriefFixture()
	units := NewReviewUnitsUseCase(f.units, f.project).Register(domain.UnitKindBrief, f.uc)
	if _, err := units.Get(context.Background(), "p1", "beat", "1"); !errors.Is(err, domain.ErrUnknownUnitKind) {
		t.Fatalf("an unregistered kind must be unknown, got %v", err)
	}
}

func TestTargetDurationIsChosenBeforeTheBrief(t *testing.T) {
	f := newBriefFixture()
	ctx := context.Background()
	if err := f.uc.SetTargetDuration(ctx, "p1", "9-12"); !errors.Is(err, domain.ErrInvalidWizardInput) {
		t.Fatalf("a length not offered must be refused, got %v", err)
	}
	if err := f.uc.SetTargetDuration(ctx, "p1", domain.Duration7to10); err != nil || f.store.duration != domain.Duration7to10 {
		t.Fatalf("a valid length before the brief must be saved, got %v", err)
	}
	if _, err := f.uc.Generate(ctx, "p1", ""); err != nil {
		t.Fatal(err)
	}
	if err := f.uc.SetTargetDuration(ctx, "p1", domain.Duration3to4); !errors.Is(err, domain.ErrInvalidStatus) {
		t.Fatalf("the length cannot change once a brief exists, got %v", err)
	}
}

func TestImpactNamesTheStoryOnlyWhenOneWasWritten(t *testing.T) {
	f := newBriefFixture()
	ctx := context.Background()
	items, err := f.uc.Impact(ctx, "p1", domain.BriefUnitKey)
	if err != nil || len(items) != 0 {
		t.Fatalf("nothing depends on the brief before the story is written, got %v, %v", items, err)
	}
	f.store.link = StoryBriefLink{HasStory: true, FromCurrentBrief: true}
	items, err = f.uc.Impact(ctx, "p1", domain.BriefUnitKey)
	if err != nil || len(items) != 1 || !strings.Contains(items[0], "Viết kịch bản") {
		t.Fatalf("a written story is named in the impact, got %v, %v", items, err)
	}
}

func TestApprovedBriefSectionIsOnlyTheApprovedBrief(t *testing.T) {
	f := newBriefFixture()
	ctx := context.Background()
	project := &f.project.project
	if _, err := f.uc.Generate(ctx, "p1", ""); err != nil {
		t.Fatal(err)
	}
	section, err := f.uc.ApprovedBriefSection(ctx, "p1", project)
	if err != nil || section != creativeBriefPlaceholder {
		t.Fatalf("a brief waiting for review does not reach the story prompt, got %q, %v", section, err)
	}
	units := NewReviewUnitsUseCase(f.units, f.project).Register(domain.UnitKindBrief, f.uc)
	if _, err := units.Approve(ctx, "p1", domain.UnitKindBrief, domain.BriefUnitKey); err != nil {
		t.Fatal(err)
	}
	section, err = f.uc.ApprovedBriefSection(ctx, "p1", project)
	if err != nil || !strings.Contains(section, "AI câu hỏi") || !strings.Contains(section, "5–7 phút") {
		t.Fatalf("the approved brief and its length reach the story prompt, got %q, %v", section, err)
	}
}

func TestStateReportsAStoryWrittenFromAnotherBrief(t *testing.T) {
	f := newBriefFixture()
	f.store.link = StoryBriefLink{HasStory: true, FromCurrentBrief: false}
	state, err := f.uc.State(context.Background(), "p1")
	if err != nil {
		t.Fatal(err)
	}
	if state.Unit != nil || !state.StoryStale || !state.StoryPromptReadsBrief {
		t.Errorf("want no unit, a stale story and the shipped story prompt reading the brief, got %+v", state)
	}
}

func TestStateGivesTheDefaultLengthWhenNoneWasPicked(t *testing.T) {
	f := newBriefFixture()
	f.store.duration = ""
	state, err := f.uc.State(context.Background(), "p1")
	if err != nil {
		t.Fatal(err)
	}
	if state.TargetDuration != domain.DefaultTargetDuration {
		t.Errorf("an unpicked length must read as the default %q, got %q", domain.DefaultTargetDuration, state.TargetDuration)
	}
}
