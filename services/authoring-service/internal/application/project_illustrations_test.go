package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"authoring/internal/domain"
)

type fakeProjectRows struct {
	mu      sync.Mutex
	lib     *fakeIllustrationRepo
	rows    map[string][]domain.ProjectIllustration
	planned map[string]bool
	n       int
}

func (f *fakeProjectRows) ListProjectIllustrations(_ context.Context, pid string) ([]domain.ProjectIllustration, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]domain.ProjectIllustration, 0, len(f.rows[pid]))
	for _, r := range f.rows[pid] {
		if r.IllustrationID != "" {
			f.lib.mu.Lock()
			if ill, ok := f.lib.rows[r.IllustrationID]; ok {
				r.Illustration = &ill
			}
			f.lib.mu.Unlock()
		}
		out = append(out, r)
	}
	return out, nil
}
func (f *fakeProjectRows) MarkIllustrationsPlanned(_ context.Context, pid string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.planned[pid] = true
	return nil
}
func (f *fakeProjectRows) IllustrationsPlanned(_ context.Context, pid string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.planned[pid], nil
}
func (f *fakeProjectRows) ReplaceProjectIllustrations(_ context.Context, pid string, rows []domain.ProjectIllustration) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rows[pid] = nil
	for _, r := range rows {
		f.n++
		r.ID = fmt.Sprintf("r%d", f.n)
		f.rows[pid] = append(f.rows[pid], r)
	}
	return nil
}
func (f *fakeProjectRows) UpdateProjectIllustration(_ context.Context, r domain.ProjectIllustration) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i, x := range f.rows[r.ProjectID] {
		if x.ID == r.ID {
			r.Illustration = nil
			f.rows[r.ProjectID][i] = r
		}
	}
	return nil
}

type storyboardOf string

func (s storyboardOf) GetAuthoringStoryboard(context.Context, string) (string, error) {
	return string(s), nil
}

const planJSON = `Đây là danh sách:
{"reuse": ["Tooth", "Germ", "KhongCo"],
 "draw": [
   {"name": "Motorbike", "description": "xe máy đỏ nhìn ngang, có mặt", "folder_id": "phuong-tien", "shots": ["1.1", "2.3"]},
   {"name": "Candy", "description": "kẹo", "folder_id": "do-an-thuc-uong"},
   {"name": "Dentist", "description": "nha sĩ áo blouse", "folder_id": "khong-ton-tai"}
 ]}`

func stage(llm *scriptedLLM) (*ProjectIllustrationsUseCase, *fakeProjectRows, *fakeIllustrationRepo) {
	lib := newFakeIllustrationRepo()
	library := NewIllustrationsUseCase(lib, &fakeRenderer{}).WithDrawer(llm, nil, 1000)
	rows := &fakeProjectRows{lib: lib, rows: map[string][]domain.ProjectIllustration{}, planned: map[string]bool{}}
	// One at a time: the scripted model answers in order.
	uc := NewProjectIllustrationsUseCase(rows, library, storyboardOf(`{"scenes":[]}`), llm, nil, 1000).WithDrawConcurrency(1)
	return uc, rows, lib
}

func TestPlanReusesWhatTheLibraryHasAndPlansTheRest(t *testing.T) {
	llm := &scriptedLLM{replies: []string{planJSON}}
	uc, _, _ := stage(llm)
	rows, err := uc.Plan(context.Background(), "p1", "")
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]domain.ProjectIllustrationState{}
	for _, r := range rows {
		got[r.Name] = r.State
	}
	want := map[string]domain.ProjectIllustrationState{
		"Motorbike": domain.PIPlanned, "Candy": domain.PIReused, "Dentist": domain.PIPlanned,
		"Tooth": domain.PIReused, "Germ": domain.PIReused,
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("plan = %v, want %v (an unknown reuse name is dropped, a draw of an existing name becomes reuse)", got, want)
	}
	for _, r := range rows {
		if r.Name == "Dentist" && r.FolderID != "do-vat" {
			t.Errorf("unknown folder not replaced: %q", r.FolderID)
		}
	}
	if sys := llm.calls[0].System; !strings.Contains(sys, "- Tooth — Chiếc răng — co-the-suc-khoe") || !strings.Contains(sys, "- phuong-tien — Phương tiện") {
		t.Error("planner prompt lacks the library catalog or the folders")
	}
}

func TestPreparePlansDrawsAndHoldsTheCodeStepUntilReviewed(t *testing.T) {
	llm := &scriptedLLM{replies: []string{planJSON, reply("Motorbike", motoCode)}}
	uc, rows, lib := stage(llm)
	ctx := context.Background()
	var reports []StageReport
	pending, err := uc.Prepare(ctx, "p1", "", func(r StageReport) { reports = append(reports, r) })
	if err != nil {
		t.Fatal(err)
	}
	names := []string{}
	for _, p := range pending {
		names = append(names, p.Name+":"+string(p.State))
	}
	// Motorbike drawn but still a draft; Dentist's drawing came back renamed to
	// Motorbike (the scripted model keeps answering that), so it failed.
	if strings.Join(names, ",") != "Motorbike:drawn,Dentist:failed" {
		t.Fatalf("pending = %v", names)
	}
	var moto string
	for _, r := range rows.rows["p1"] {
		if r.Name == "Motorbike" {
			moto = r.IllustrationID
		}
		if r.Name == "Dentist" {
			uc.SetSkipped(ctx, "p1", r.ID, true)
		}
	}
	if reports[0].Phase != "plan" {
		t.Errorf("first report must be the planning phase: %+v", reports)
	}
	last := reports[len(reports)-1]
	if last.Phase != "draw" || last.Total != 2 || last.Done != 2 || last.Failed != 1 || last.Reused != 3 || last.Planned != 5 {
		t.Errorf("final report = %+v, want 2 drawn of 2 (1 failed), 3 reused, 5 listed", last)
	}
	calls := len(llm.calls)
	if pending, _, _ = uc.Gate(ctx, "p1"); len(pending) != 1 || len(llm.calls) != calls {
		t.Fatalf("the gate must not re-plan or redraw: %d pending, %d new calls", len(pending), len(llm.calls)-calls)
	}
	ill := lib.rows[moto]
	ill.Status = domain.IllustrationApproved
	lib.rows[moto] = ill
	if pending, _, _ = uc.Gate(ctx, "p1"); len(pending) != 0 {
		t.Fatalf("approved + skipped must open the gate, still pending: %+v", pending)
	}
	drawings, _ := uc.ForCode(ctx, "p1")
	if len(drawings) == 0 || drawings[0].Name != "Motorbike" {
		t.Fatalf("the video's own drawing must come first: %+v", drawings)
	}
	for _, d := range drawings {
		if d.Name == "Tooth" || d.Code == "" {
			t.Errorf("kit built-ins have no code to hand over: %+v", d)
		}
	}
}

func TestForCodeNeverHandsOverADraft(t *testing.T) {
	uc, _, lib := stage(&scriptedLLM{replies: []string{"{}"}})
	lib.rows["d"] = domain.Illustration{ID: "d", Name: "Draft", Code: "export function Draft() {}", Status: domain.IllustrationDraft}
	drawings, _ := uc.ForCode(context.Background(), "p1")
	for _, d := range drawings {
		if d.Name == "Draft" {
			t.Fatal("a draft reached the code step")
		}
	}
}

// CR-045: the planner's list is kept whole — no cap on how many drawings a video gets.
func TestPlanKeepsEveryDrawingTheStoryboardNeeds(t *testing.T) {
	var draw []string
	for i := 0; i < 15; i++ {
		draw = append(draw, fmt.Sprintf(`{"name": "Thing%d", "description": "vật %d", "folder_id": "do-vat"}`, i, i))
	}
	uc, rows, _ := stage(&scriptedLLM{replies: []string{`{"reuse": [], "draw": [` + strings.Join(draw, ",") + `]}`}})
	got, err := uc.Plan(context.Background(), "p1", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 15 {
		t.Fatalf("plan kept %d of 15 drawings", len(got))
	}
	if !rows.planned["p1"] {
		t.Error("planning must mark the list as planned")
	}
}

// gateLLM answers each drawing with a drawing of the requested name, holding
// every call until `hold` calls are in flight at once (or a timeout passes).
type gateLLM struct {
	mu        sync.Mutex
	inFlight  int
	maxFlight int
	hold      int
	release   chan struct{}
	once      sync.Once
}

func (g *gateLLM) Name() string { return "fake" }
func (g *gateLLM) Chat(ctx context.Context, req ChatRequest) (ChatResult, error) {
	g.mu.Lock()
	g.inFlight++
	if g.inFlight > g.maxFlight {
		g.maxFlight = g.inFlight
	}
	if g.inFlight >= g.hold {
		g.once.Do(func() { close(g.release) })
	}
	g.mu.Unlock()
	if req.OnProgress != nil {
		req.OnProgress(ChatProgress{ContentChars: 120})
	}
	select {
	case <-g.release:
	case <-time.After(2 * time.Second):
	case <-ctx.Done():
	}
	g.mu.Lock()
	g.inFlight--
	g.mu.Unlock()
	name := strings.TrimSpace(req.User[strings.LastIndex(req.User, ":")+1:])
	return ChatResult{Content: reply(name, "export function "+name+"({color = '#E8453C', ...fig}) { return null; }")}, nil
}

func plannedRows(rows *fakeProjectRows, pid string, n int) {
	for i := 0; i < n; i++ {
		rows.n++
		rows.rows[pid] = append(rows.rows[pid], domain.ProjectIllustration{
			ID: fmt.Sprintf("r%d", rows.n), ProjectID: pid, Position: i + 1, Name: fmt.Sprintf("Thing%d", i),
			Description: "vật", FolderID: "do-vat", State: domain.PIPlanned,
		})
	}
	rows.planned[pid] = true
}

func TestDrawMissingDrawsSeveralAtOnce(t *testing.T) {
	llm := &gateLLM{hold: 3, release: make(chan struct{})}
	uc, rows, _ := stage(&scriptedLLM{replies: []string{"{}"}})
	uc.library.drawer.llm = llm
	uc.WithDrawConcurrency(3)
	plannedRows(rows, "p1", 7)

	var mu sync.Mutex
	var last StageReport
	if err := uc.DrawMissing(context.Background(), "p1", "", func(r StageReport) {
		mu.Lock()
		if r.Done >= last.Done {
			last = r
		}
		mu.Unlock()
	}); err != nil {
		t.Fatal(err)
	}
	if llm.maxFlight != 3 {
		t.Errorf("drew %d at once, want 3", llm.maxFlight)
	}
	if last.Total != 7 || last.Done != 7 || last.Failed != 0 {
		t.Errorf("final report %+v", last)
	}
	for _, r := range rows.rows["p1"] {
		if r.State != domain.PIDrawn {
			t.Errorf("%s left %s", r.Name, r.State)
		}
	}
}

func TestListShowsTheLiveProgressOfADrawingInFlight(t *testing.T) {
	llm := &gateLLM{hold: 99, release: make(chan struct{})}
	uc, rows, _ := stage(&scriptedLLM{replies: []string{"{}"}})
	uc.library.drawer.llm = llm
	plannedRows(rows, "p1", 1)

	done := make(chan error)
	go func() {
		_, err := uc.Draw(context.Background(), "p1", "r1", "")
		done <- err
	}()
	var seen *domain.DrawProgress
	for i := 0; i < 200 && seen == nil; i++ {
		list, _ := uc.List(context.Background(), "p1")
		if p := list[0].Progress; p != nil && p.Phase == "writing" {
			seen = p
		}
		time.Sleep(5 * time.Millisecond)
	}
	if seen == nil || seen.Attempt != 1 || seen.MaxAttempts != maxDrawAttempts || seen.ContentChars != 120 {
		t.Fatalf("progress in flight = %+v", seen)
	}
	if _, err := uc.Draw(context.Background(), "p1", "r1", ""); err == nil {
		t.Error("a second draw of a row in flight must be refused")
	}
	llm.once.Do(func() { close(llm.release) })
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if list, _ := uc.List(context.Background(), "p1"); list[0].Progress != nil {
		t.Error("a finished drawing still reports progress")
	}
}

func TestDrawMissingRedrawsARowARestartLeftDrawing(t *testing.T) {
	uc, rows, _ := stage(&scriptedLLM{replies: []string{reply("Thing0", "export function Thing0({color = '#fff', ...fig}) { return null; }")}})
	plannedRows(rows, "p1", 1)
	rows.rows["p1"][0].State = domain.PIDrawing
	if err := uc.DrawMissing(context.Background(), "p1", "", nil); err != nil {
		t.Fatal(err)
	}
	if st := rows.rows["p1"][0].State; st != domain.PIDrawn {
		t.Fatalf("stale drawing row = %s, want drawn", st)
	}
}

func TestGateTellsNeverPlannedFromNothingToDraw(t *testing.T) {
	uc, _, _ := stage(&scriptedLLM{replies: []string{`{"reuse": [], "draw": []}`}})
	ctx := context.Background()
	if _, planned, _ := uc.Gate(ctx, "p1"); planned {
		t.Fatal("a video never planned reads as planned")
	}
	if _, err := uc.Plan(ctx, "p1", ""); err != nil {
		t.Fatal(err)
	}
	if pending, planned, _ := uc.Gate(ctx, "p1"); !planned || len(pending) != 0 {
		t.Fatalf("an empty plan must open the gate: planned=%v pending=%d", planned, len(pending))
	}
}

func TestDeleteDrawingRemovesTheDraftFromTheLibraryAndSkipsTheRow(t *testing.T) {
	uc, rows, lib := stage(&scriptedLLM{replies: []string{reply("Thing0", "export function Thing0({color = '#fff', ...fig}) { return null; }")}})
	ctx := context.Background()
	plannedRows(rows, "p1", 1)
	drawn, err := uc.Draw(ctx, "p1", "r1", "")
	if err != nil {
		t.Fatal(err)
	}
	illID := drawn.IllustrationID
	row, err := uc.DeleteDrawing(ctx, "p1", "r1")
	if err != nil {
		t.Fatal(err)
	}
	if _, still := lib.rows[illID]; still {
		t.Error("the draft is still in the library")
	}
	if row.State != domain.PISkipped || row.IllustrationID != "" || !row.Ready() {
		t.Fatalf("row after delete = %+v", row)
	}
	if back, _ := uc.SetSkipped(ctx, "p1", "r1", false); back.State != domain.PIPlanned {
		t.Errorf("taking the skip back must plan the drawing again, got %s", back.State)
	}
}

func TestDeleteDrawingKeepsApprovedAndLibraryDrawings(t *testing.T) {
	uc, rows, lib := stage(&scriptedLLM{replies: []string{"{}"}})
	ctx := context.Background()
	lib.rows["ok"] = domain.Illustration{ID: "ok", Name: "Approved", Code: "x", Status: domain.IllustrationApproved}
	rows.rows["p1"] = []domain.ProjectIllustration{
		{ID: "a", ProjectID: "p1", Name: "Approved", State: domain.PIDrawn, IllustrationID: "ok"},
		{ID: "b", ProjectID: "p1", Name: "Tooth", State: domain.PIReused, IllustrationID: domain.BuiltinIllustrations()[0].ID},
	}
	for _, id := range []string{"a", "b"} {
		if _, err := uc.DeleteDrawing(ctx, "p1", id); !errors.Is(err, ErrIllustrationNotDeletable) {
			t.Errorf("row %s: err = %v, want ErrIllustrationNotDeletable", id, err)
		}
	}
	if _, ok := lib.rows["ok"]; !ok {
		t.Error("an approved drawing was deleted")
	}
}

// CR-052: a Hình mẫu copy only teaches the AI drawer. Videos are offered its
// source; an original exemplar stays usable as before.
func TestExemplarCopiesAreNeitherPlannedNorHandedToTheCode(t *testing.T) {
	llm := &scriptedLLM{replies: []string{`{"reuse": [], "draw": []}`}}
	uc, _, lib := stage(llm)
	lib.rows["bus"] = domain.Illustration{ID: "bus", Name: "Bus", FolderID: "phuong-tien", Code: busCode, Status: domain.IllustrationApproved}
	lib.rows["cp"] = domain.Illustration{ID: "cp", Name: "BusMau", FolderID: domain.ExemplarFolderID, Exemplar: true, SourceID: "bus",
		Code: "export function BusMau() {}", Status: domain.IllustrationApproved}
	lib.rows["cat"] = domain.Illustration{ID: "cat", Name: "Cat", FolderID: domain.ExemplarFolderID, Exemplar: true, HomeFolderID: "dong-vat",
		Code: "export function Cat() {}", Status: domain.IllustrationApproved}
	ctx := context.Background()
	if _, err := uc.Plan(ctx, "p1", ""); err != nil {
		t.Fatal(err)
	}
	sys := llm.calls[0].System
	if strings.Contains(sys, "BusMau") || !strings.Contains(sys, "- Bus —") || !strings.Contains(sys, "- Cat —") {
		t.Fatalf("planner catalog: %s", sys)
	}
	if strings.Contains(sys, "- "+domain.ExemplarFolderID+" —") {
		t.Fatal("the planner must not be offered the Hình mẫu folder")
	}
	drawings, err := uc.ForCode(ctx, "p1")
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, d := range drawings {
		names[d.Name] = true
	}
	if names["BusMau"] || !names["Bus"] || !names["Cat"] {
		t.Fatalf("code step drawings: %v", names)
	}
}

// CR-052: deleting this video's own draft is not blocked by this video, but
// is by another video that has not reached its result.
func TestDeleteDrawingIgnoresItsOwnProjectButNotOthers(t *testing.T) {
	uc, rows, lib := stage(&scriptedLLM{replies: []string{reply("Thing0", "export function Thing0({color = '#fff', ...fig}) { return null; }")}})
	uc.library.WithProjectStatus(fakeProjectStatus{status: map[string]domain.ProjectStatus{"p1": domain.StatusDraft, "p2": domain.StatusDraft}})
	ctx := context.Background()
	plannedRows(rows, "p1", 1)
	drawn, err := uc.Draw(ctx, "p1", "r1", "")
	if err != nil {
		t.Fatal(err)
	}
	lib.users[drawn.IllustrationID] = []IllustrationUser{{ProjectID: "p1"}, {ProjectID: "p2"}}
	if _, err := uc.DeleteDrawing(ctx, "p1", "r1"); !errors.Is(err, ErrIllustrationInUse) {
		t.Fatalf("another draft project uses it: %v", err)
	}
	lib.users[drawn.IllustrationID] = []IllustrationUser{{ProjectID: "p1"}}
	lib.linked[drawn.IllustrationID] = []string{"p1"}
	if _, err := uc.DeleteDrawing(ctx, "p1", "r1"); err != nil {
		t.Fatalf("its own project must not block it: %v", err)
	}
}
