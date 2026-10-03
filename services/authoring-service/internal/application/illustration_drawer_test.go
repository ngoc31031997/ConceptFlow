package application

import (
	"context"
	"errors"
	"strings"
	"testing"

	"authoring/internal/domain"
)

type scriptedLLM struct {
	replies []string
	calls   []ChatRequest
}

func (s *scriptedLLM) Name() string { return "fake" }
func (s *scriptedLLM) Chat(_ context.Context, req ChatRequest) (ChatResult, error) {
	s.calls = append(s.calls, req)
	r := s.replies[0]
	if len(s.replies) > 1 {
		s.replies = s.replies[1:]
	}
	return ChatResult{Content: r}, nil
}

func reply(name, code string) string {
	return "TÊN: " + name + "\nTÊN HIỂN THỊ: Xe máy\nTHẺ: xe, xe máy, Xe\nMÔ TẢ: Xe máy đỏ nhìn ngang.\nCÁCH GỌI: <" + name + " color /> — 300×200\n```tsx\n" + code + "\n```\n"
}

const motoCode = "export function Motorbike({color = '#E8453C', ...fig}) { return null; }"

func TestDrawRetriesWithTheRenderersLineNumberedErrorsThenStoresADraft(t *testing.T) {
	repo, rend := newFakeIllustrationRepo(), &fakeRenderer{}
	llm := &scriptedLLM{replies: []string{reply("Motorbike", "BROKEN"), reply("Motorbike", motoCode)}}
	uc := NewIllustrationsUseCase(repo, rend).WithDrawer(llm, nil, 1000).WithStylePrompts(systemStyle{})
	out, err := uc.Draw(context.Background(), DrawRequest{Description: "xe máy màu đỏ", FolderID: "phuong-tien"})
	if err != nil {
		t.Fatal(err)
	}
	if len(llm.calls) != 2 || !strings.Contains(llm.calls[1].User, "dòng 3") || !strings.Contains(llm.calls[1].User, "BROKEN") {
		t.Fatalf("second turn must carry the failed code and its errors: %d calls, %q", len(llm.calls), llm.calls[len(llm.calls)-1].User)
	}
	if out.Name != "Motorbike" || out.Title != "Xe máy" || out.Status != domain.IllustrationDraft || !out.HasPreview {
		t.Fatalf("stored row: %+v", out)
	}
	if strings.Join(out.Tags, "|") != "xe|xe máy" {
		t.Fatalf("tags: %v", out.Tags)
	}
}

func TestDrawPromptCarriesRulesHelpersExemplarsAndTheCreatorsApprovedDrawings(t *testing.T) {
	repo := newFakeIllustrationRepo()
	repo.rows["mine"] = domain.Illustration{ID: "mine", Name: "Scooter", Title: "Xe tay ga", FolderID: "phuong-tien",
		Code: "export function Scooter() {}", Status: domain.IllustrationApproved}
	repo.rows["other"] = domain.Illustration{ID: "other", Name: "Dog", FolderID: "dong-vat",
		Code: "export function Dog() {}", Status: domain.IllustrationApproved}
	// The Hình mẫu are library rows the Creator picked.
	for n, name := range []string{"SchoolBus", "Cat", "Microscope"} {
		id := "exemplar-" + name
		repo.rows[id] = domain.Illustration{ID: id, Name: name, FolderID: domain.ExemplarFolderID, Exemplar: true,
			Code: "export function " + name + "() {}", Status: domain.IllustrationApproved,
			CreatedAt: "2026-09-2" + string(rune('1'+n)) + "T00:00:00Z"}
	}
	llm := &scriptedLLM{replies: []string{reply("Motorbike", motoCode)}}
	uc := NewIllustrationsUseCase(repo, &fakeRenderer{}).WithDrawer(llm, nil, 1000).WithStylePrompts(systemStyle{})
	if _, err := uc.Draw(context.Background(), DrawRequest{Description: "xe máy", FolderID: "phuong-tien"}); err != nil {
		t.Fatal(err)
	}
	sys := llm.calls[0].System
	for _, want := range []string{"[S1]", "[S19]", "<Figure", "useBlink", "export function SchoolBus", "export function Cat",
		"export function Microscope", "export function Scooter", "TÊN HIỂN THỊ:"} {
		if !strings.Contains(sys, want) {
			t.Errorf("drawer prompt lacks %q", want)
		}
	}
	if strings.Contains(sys, "export function Dog") || strings.Contains(sys, "{{") {
		t.Error("drawer prompt holds another folder's drawing or an unfilled placeholder")
	}
	if a, b := strings.Index(sys, "SchoolBus"), strings.Index(sys, "Microscope"); a > b {
		t.Error("Hình mẫu must come oldest first")
	}
}

func TestDrawPromptWithoutExemplarsSaysSoAndKeepsAtMostTwoOwnDrawings(t *testing.T) {
	repo := newFakeIllustrationRepo()
	for _, name := range []string{"Scooter", "Truck", "Van"} {
		repo.rows[name] = domain.Illustration{ID: name, Name: name, FolderID: "phuong-tien",
			Code: "export function " + name + "() {}", Status: domain.IllustrationApproved}
	}
	llm := &scriptedLLM{replies: []string{reply("Motorbike", motoCode)}}
	uc := NewIllustrationsUseCase(repo, &fakeRenderer{}).WithDrawer(llm, nil, 1000).WithStylePrompts(systemStyle{})
	if _, err := uc.Draw(context.Background(), DrawRequest{Description: "xe máy", FolderID: "phuong-tien"}); err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(llm.calls[0].System, "hình Creator đã duyệt, cùng thư mục"); n != maxOwnReferences {
		t.Fatalf("want %d own references, got %d", maxOwnReferences, n)
	}

	empty := newFakeIllustrationRepo()
	llm = &scriptedLLM{replies: []string{reply("Motorbike", motoCode)}}
	uc = NewIllustrationsUseCase(empty, &fakeRenderer{}).WithDrawer(llm, nil, 1000).WithStylePrompts(systemStyle{})
	if _, err := uc.Draw(context.Background(), DrawRequest{Description: "xe máy", FolderID: "phuong-tien"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(llm.calls[0].System, "Chưa có hình tham chiếu") {
		t.Fatal("an empty reference section must say so")
	}
}

func TestDrawAndRedrawRefuseTheExemplarFolderAndExemplars(t *testing.T) {
	repo := newFakeIllustrationRepo()
	repo.rows["ex"] = domain.Illustration{ID: "ex", Name: "BusMau", FolderID: domain.ExemplarFolderID, Exemplar: true,
		Code: busCode, Status: domain.IllustrationApproved}
	llm := &scriptedLLM{replies: []string{reply("Motorbike", motoCode)}}
	uc := NewIllustrationsUseCase(repo, &fakeRenderer{}).WithDrawer(llm, nil, 1000).WithStylePrompts(systemStyle{})
	if _, err := uc.Draw(context.Background(), DrawRequest{Description: "xe", FolderID: domain.ExemplarFolderID}); !errors.Is(err, ErrExemplarFolder) {
		t.Fatalf("draw into Hình mẫu: %v", err)
	}
	if _, err := uc.Redraw(context.Background(), "ex", "", ""); !errors.Is(err, ErrIllustrationReadOnly) {
		t.Fatalf("redraw an exemplar: %v", err)
	}
	if len(llm.calls) != 0 {
		t.Fatal("the model must not be called")
	}
}

func TestDrawGivesUpAfterThreeAttemptsAndSavesNothing(t *testing.T) {
	repo := newFakeIllustrationRepo()
	llm := &scriptedLLM{replies: []string{reply("Motorbike", "BROKEN")}}
	uc := NewIllustrationsUseCase(repo, &fakeRenderer{}).WithDrawer(llm, nil, 1000).WithStylePrompts(systemStyle{})
	_, err := uc.Draw(context.Background(), DrawRequest{Description: "xe", FolderID: "phuong-tien"})
	if err == nil || len(llm.calls) != maxDrawAttempts || !errors.Is(err, ErrIllustrationInvalid) {
		t.Fatalf("want failure after %d calls, got %d, %v", maxDrawAttempts, len(llm.calls), err)
	}
	for _, r := range repo.rows {
		if r.Name == "Motorbike" {
			t.Fatal("a drawing that never passed was saved")
		}
	}
	if _, err := NewIllustrationsUseCase(repo, &fakeRenderer{}).Draw(context.Background(), DrawRequest{Description: "x", FolderID: "phuong-tien"}); !errors.Is(err, ErrDrawerDisabled) {
		t.Fatalf("no drawer configured: %v", err)
	}
}

func TestRedrawKeepsTheNameAndMakesANewDraftVersion(t *testing.T) {
	repo, rend := newFakeIllustrationRepo(), &fakeRenderer{}
	uc := NewIllustrationsUseCase(repo, rend)
	ctx := context.Background()
	made, _ := uc.Create(ctx, domain.Illustration{Name: "Motorbike", FolderID: "phuong-tien", Code: motoCode})
	uc.SetStatus(ctx, made.ID, domain.IllustrationApproved)

	llm := &scriptedLLM{replies: []string{reply("Motorbike", motoCode+"\n// bánh to hơn")}}
	uc.WithDrawer(llm, nil, 1000).WithStylePrompts(systemStyle{})
	out, err := uc.Redraw(ctx, made.ID, "bánh to hơn", "")
	if err != nil {
		t.Fatal(err)
	}
	if out.Version != 2 || out.Status != domain.IllustrationDraft || !strings.Contains(llm.calls[0].User, "bánh to hơn") {
		t.Fatalf("redraw: %+v", out)
	}
	llm.replies = []string{reply("Scooter", strings.Replace(motoCode, "Motorbike", "Scooter", 1))}
	if _, err := uc.Redraw(ctx, made.ID, "", ""); err == nil {
		t.Fatal("a redraw that renamed the component was accepted")
	}
	if _, err := uc.Redraw(ctx, "builtin-Tooth", "", ""); !errors.Is(err, ErrIllustrationReadOnly) {
		t.Fatalf("built-in redraw: %v", err)
	}
}

// systemStyle is a prompt library whose active illustration_style row is the shipped one.
type systemStyle struct{}

func (systemStyle) GetActive(_ context.Context, role domain.PromptRole) (domain.Prompt, error) {
	return domain.Prompt{Role: role, TemplateText: domain.IllustrationStyleGuide()}, nil
}

// creatorStyle is a prompt library where the Creator activated their own rules, or one that cannot be read.
type creatorStyle struct {
	text string
	err  error
}

func (c creatorStyle) GetActive(_ context.Context, role domain.PromptRole) (domain.Prompt, error) {
	if role != domain.RoleIllustrationStyle {
		return domain.Prompt{}, errors.New("asked for the wrong role " + string(role))
	}
	return domain.Prompt{Role: role, TemplateText: c.text}, c.err
}

func TestTheIllustratorIsHeldToTheActiveStyleRowForFiguresAndBackdrops(t *testing.T) {
	ctx := context.Background()
	uc := NewIllustrationsUseCase(newFakeIllustrationRepo(), &fakeRenderer{}).WithDrawer(&scriptedLLM{}, nil, 1000).
		WithStylePrompts(creatorStyle{text: "  - [S1] LUẬT CỦA CREATOR: chỉ khối tròn  "})
	for _, kind := range []domain.IllustrationKind{domain.IllustrationFigure, domain.IllustrationBackdrop} {
		system, err := uc.drawerSystem(ctx, "phuong-tien", "", kind)
		if err != nil {
			t.Fatalf("%s: %v", kind, err)
		}
		if !strings.Contains(system, "- [S1] LUẬT CỦA CREATOR: chỉ khối tròn") {
			t.Errorf("%s system prompt does not carry the active style row", kind)
		}
		if strings.Contains(system, domain.IllustrationStyleGuide()) {
			t.Errorf("%s system prompt still carries the shipped rules over the Creator's", kind)
		}
	}
}

func TestDrawingStopsWhenTheStyleRulesCannotBeRead(t *testing.T) {
	ctx := context.Background()
	failing := NewIllustrationsUseCase(newFakeIllustrationRepo(), &fakeRenderer{}).WithDrawer(&scriptedLLM{}, nil, 1000).
		WithStylePrompts(creatorStyle{err: errors.New("db down")})
	if _, err := failing.drawerSystem(ctx, "phuong-tien", "", domain.IllustrationFigure); err == nil || !strings.Contains(err.Error(), "db down") {
		t.Fatalf("want the read error, got %v", err)
	}
	unwired := NewIllustrationsUseCase(newFakeIllustrationRepo(), &fakeRenderer{})
	if _, err := unwired.StyleRules(ctx); err == nil {
		t.Fatal("no prompt library must be an error, not the shipped text")
	}
}
