package application

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"authoring/internal/domain"
)

// CR-044 — the drawings one video needs. After the storyboard, a model reads it
// against the library and lists what can be reused and what must be drawn;
// each missing drawing goes through the AI drawer; the Creator reviews them;
// the code step waits until every one is approved or skipped, then hands the
// approved drawings to the Remotion Engineer.
//
// CR-045 — planning and drawing are their own authoring step ("illustrations",
// between Visual and Code), the list has no size cap, missing drawings are
// drawn several at a time, and each drawing in flight reports its progress.

const (
	// maxCodeLibraryDrawings caps the EXTRA library drawings offered to the code
	// step on top of the video's own (which are always all handed over): it
	// bounds the prompt, not what the video may use.
	maxCodeLibraryDrawings = 40
	plannerRole            = "illustration_planner"
	// DefaultDrawConcurrency is how many drawings are drawn at once
	// (ILLUSTRATION_DRAW_CONCURRENCY).
	DefaultDrawConcurrency = 4
)

//go:embed planner_prompt_vi.txt
var plannerPromptVI string

// ErrIllustrationsPending stops the code step while the video still has
// drawings that are neither approved nor skipped.
type ErrIllustrationsPending struct{ Rows []domain.ProjectIllustration }

func (e *ErrIllustrationsPending) Error() string {
	return fmt.Sprintf("Còn %d hình minh hoạ chưa duyệt (%s). Duyệt hoặc bỏ qua chúng ở bước Hình minh hoạ, rồi chạy lại bước Code.",
		len(e.Rows), pendingNames(e.Rows))
}

// ErrIllustrationsNotPlanned stops the code step of a Remotion video whose
// drawing list was never made (CR-045): the illustrations step comes first.
var ErrIllustrationsNotPlanned = errors.New("chưa lập danh sách hình minh hoạ — hãy chạy bước Hình minh hoạ trước bước Code")

// ErrIllustrationNotDeletable refuses to delete a drawing that is not this
// video's own unapproved draft (CR-045).
var ErrIllustrationNotDeletable = errors.New("chỉ xoá được hình nháp do chính video này vẽ")

func pendingNames(rows []domain.ProjectIllustration) string {
	names := make([]string, 0, len(rows))
	for _, r := range rows {
		names = append(names, r.Name)
	}
	return strings.Join(names, ", ")
}

// AwaitingReviewMessage is the Creator-facing sentence for an illustrations
// step that finished drawing but still waits for the Creator's review.
func AwaitingReviewMessage(rows []domain.ProjectIllustration) string {
	return fmt.Sprintf("Đã vẽ xong. Còn %d hình chờ bạn duyệt hoặc bỏ qua (%s) — duyệt xong thì chạy bước Code.",
		len(rows), pendingNames(rows))
}

// ProjectIllustrationRepoPort persists the per-video list.
type ProjectIllustrationRepoPort interface {
	ListProjectIllustrations(ctx context.Context, projectID string) ([]domain.ProjectIllustration, error)
	ReplaceProjectIllustrations(ctx context.Context, projectID string, rows []domain.ProjectIllustration) error
	UpdateProjectIllustration(ctx context.Context, row domain.ProjectIllustration) error
	// CR-045: whether the list was ever planned — an empty list can mean
	// "this video needs no drawing" or "never planned".
	MarkIllustrationsPlanned(ctx context.Context, projectID string) error
	IllustrationsPlanned(ctx context.Context, projectID string) (bool, error)
}

// StoryboardReaderPort reads the saved storyboard of a project.
type StoryboardReaderPort interface {
	GetAuthoringStoryboard(ctx context.Context, projectID string) (string, error)
}

// LibraryDrawing is one approved drawing handed to the code step.
type LibraryDrawing struct {
	Name, Usage, Description, Code string
}

type ProjectIllustrationsUseCase struct {
	repo        ProjectIllustrationRepoPort
	library     *IllustrationsUseCase
	storyboards StoryboardReaderPort
	llm         LLMProviderPort
	recorder    *LLMUsageRecorder
	maxTokens   int
	concurrency int

	// drawing is the live progress of each row being drawn, by row id. In
	// memory on purpose: it only has to outlive the draw, and the row's
	// "drawing" state in the database already survives a restart.
	mu      sync.Mutex
	drawing map[string]*drawState
}

type drawState struct {
	progress domain.DrawProgress
	started  time.Time
}

func NewProjectIllustrationsUseCase(
	repo ProjectIllustrationRepoPort, library *IllustrationsUseCase, storyboards StoryboardReaderPort,
	llm LLMProviderPort, recorder *LLMUsageRecorder, maxTokens int,
) *ProjectIllustrationsUseCase {
	return &ProjectIllustrationsUseCase{repo: repo, library: library, storyboards: storyboards, llm: llm,
		recorder: recorder, maxTokens: maxTokens, concurrency: DefaultDrawConcurrency, drawing: map[string]*drawState{}}
}

// WithDrawConcurrency sets how many drawings are drawn at once (min 1).
func (uc *ProjectIllustrationsUseCase) WithDrawConcurrency(n int) *ProjectIllustrationsUseCase {
	if n < 1 {
		n = 1
	}
	uc.concurrency = n
	return uc
}

// List is the video's list, each row in flight carrying its live progress.
func (uc *ProjectIllustrationsUseCase) List(ctx context.Context, projectID string) ([]domain.ProjectIllustration, error) {
	rows, err := uc.repo.ListProjectIllustrations(ctx, projectID)
	if err != nil {
		return nil, err
	}
	uc.mu.Lock()
	defer uc.mu.Unlock()
	for i := range rows {
		if st := uc.drawing[rows[i].ID]; st != nil {
			p := st.progress
			p.ElapsedSeconds = int(time.Since(st.started).Seconds())
			rows[i].Progress = &p
		}
	}
	return rows, nil
}

func (uc *ProjectIllustrationsUseCase) inFlight(rowID string) bool {
	uc.mu.Lock()
	defer uc.mu.Unlock()
	return uc.drawing[rowID] != nil
}

func (uc *ProjectIllustrationsUseCase) setProgress(rowID string, p domain.DrawProgress) {
	uc.mu.Lock()
	defer uc.mu.Unlock()
	if st := uc.drawing[rowID]; st != nil {
		st.progress = p
	}
}

type plannedReply struct {
	Reuse []string `json:"reuse"`
	Draw  []struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		FolderID    string   `json:"folder_id"`
		Shots       []string `json:"shots"`
	} `json:"draw"`
}

var jsonObjectRe = regexp.MustCompile(`(?s)\{.*\}`)

func parsePlan(text string) (plannedReply, error) {
	var p plannedReply
	raw := jsonObjectRe.FindString(text)
	if raw == "" {
		return p, fmt.Errorf("câu trả lời không có JSON")
	}
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		return p, fmt.Errorf("JSON lập danh sách hình không đọc được: %w", err)
	}
	return p, nil
}

// Plan reads the storyboard against the library and replaces the video's list.
func (uc *ProjectIllustrationsUseCase) Plan(ctx context.Context, projectID, model string) ([]domain.ProjectIllustration, error) {
	if uc.llm == nil {
		return nil, ErrDrawerDisabled
	}
	storyboard, err := uc.storyboards.GetAuthoringStoryboard(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("load storyboard: %w", err)
	}
	if strings.TrimSpace(storyboard) == "" {
		return nil, fmt.Errorf("chưa có storyboard — hãy chạy Bước 4 — Visual trước")
	}
	all, err := uc.library.List(ctx, IllustrationFilter{})
	if err != nil {
		return nil, err
	}
	folders, err := uc.library.Folders(ctx)
	if err != nil {
		return nil, err
	}
	byName := map[string]domain.Illustration{}
	var catalog, folderText strings.Builder
	for _, i := range all {
		if !i.Builtin && i.Status != domain.IllustrationApproved {
			continue // a draft is not something to reuse yet
		}
		byName[i.Name] = i
		fmt.Fprintf(&catalog, "- %s — %s — %s — %s\n", i.Name, i.Title, i.FolderID, strings.Join(i.Tags, ", "))
	}
	folderIDs := map[string]bool{}
	for _, f := range folders {
		folderIDs[f.ID] = true
		fmt.Fprintf(&folderText, "- %s — %s — %s\n", f.ID, f.Name, f.Description)
	}
	system := strings.NewReplacer("{{catalog}}", strings.TrimSpace(catalog.String()),
		"{{folders}}", strings.TrimSpace(folderText.String())).Replace(plannerPromptVI)

	started := time.Now()
	res, err := uc.llm.Chat(ctx, ChatRequest{System: system, User: "STORYBOARD:\n" + storyboard,
		MaxTokens: uc.maxTokens, Temperature: 0.2, Model: model})
	if uc.recorder != nil {
		uc.recorder.Record(ctx, RecordFor(uc.llm.Name(), plannerRole, "illustration", projectID, res.Usage, started, err))
	}
	if err != nil {
		return nil, err
	}
	plan, err := parsePlan(res.Content)
	if err != nil {
		return nil, err
	}

	var rows []domain.ProjectIllustration
	seen := map[string]bool{}
	add := func(r domain.ProjectIllustration) {
		if r.Name == "" || seen[r.Name] {
			return
		}
		seen[r.Name] = true
		r.ProjectID, r.Position = projectID, len(rows)+1
		rows = append(rows, r)
	}
	for _, d := range plan.Draw {
		name := strings.TrimSpace(d.Name)
		if existing, ok := byName[name]; ok {
			// The model proposed drawing something the library already has.
			add(domain.ProjectIllustration{Name: name, Description: existing.Description, FolderID: existing.FolderID,
				Shots: d.Shots, State: domain.PIReused, IllustrationID: existing.ID})
			continue
		}
		folder := d.FolderID
		if !folderIDs[folder] {
			folder = "do-vat"
		}
		if domain.ValidateIllustrationName(name) != nil || strings.TrimSpace(d.Description) == "" {
			continue
		}
		add(domain.ProjectIllustration{Name: name, Description: strings.TrimSpace(d.Description), FolderID: folder,
			Shots: d.Shots, State: domain.PIPlanned})
	}
	for _, name := range plan.Reuse {
		if existing, ok := byName[strings.TrimSpace(name)]; ok {
			add(domain.ProjectIllustration{Name: existing.Name, Description: existing.Description,
				FolderID: existing.FolderID, State: domain.PIReused, IllustrationID: existing.ID})
		}
	}
	if err := uc.repo.ReplaceProjectIllustrations(ctx, projectID, rows); err != nil {
		return nil, err
	}
	if err := uc.repo.MarkIllustrationsPlanned(ctx, projectID); err != nil {
		return nil, err
	}
	return uc.List(ctx, projectID)
}

func (uc *ProjectIllustrationsUseCase) row(ctx context.Context, projectID, rowID string) (domain.ProjectIllustration, error) {
	rows, err := uc.repo.ListProjectIllustrations(ctx, projectID)
	if err != nil {
		return domain.ProjectIllustration{}, err
	}
	for _, r := range rows {
		if r.ID == rowID {
			return r, nil
		}
	}
	return domain.ProjectIllustration{}, ErrIllustrationNotFound
}

// Draw draws one planned (or failed) row with the AI drawer.
func (uc *ProjectIllustrationsUseCase) Draw(ctx context.Context, projectID, rowID, model string) (domain.ProjectIllustration, error) {
	r, err := uc.row(ctx, projectID, rowID)
	if err != nil {
		return r, err
	}
	// "drawing" in the database with nothing in flight here is a draw that a
	// restart cut short: it may be drawn again. One in flight may not.
	if r.State == domain.PIReused || (r.State == domain.PIDrawn && r.Illustration != nil) {
		return r, fmt.Errorf("hình %s đã có — dùng \"Vẽ lại bằng AI\" trên hình đó", r.Name)
	}
	uc.mu.Lock()
	if uc.drawing[rowID] != nil {
		uc.mu.Unlock()
		return r, fmt.Errorf("hình %s đang được vẽ", r.Name)
	}
	uc.drawing[rowID] = &drawState{progress: domain.DrawProgress{Phase: "waiting"}, started: time.Now()}
	uc.mu.Unlock()
	defer func() {
		uc.mu.Lock()
		delete(uc.drawing, rowID)
		uc.mu.Unlock()
	}()
	r.State, r.Error = domain.PIDrawing, ""
	if err := uc.repo.UpdateProjectIllustration(ctx, r); err != nil {
		return r, err
	}
	made, drawErr := uc.library.Draw(ctx, DrawRequest{Description: r.Description, FolderID: r.FolderID, Name: r.Name, Model: model,
		OnProgress: func(p domain.DrawProgress) { uc.setProgress(rowID, p) }})
	if drawErr != nil {
		r.State, r.Error = domain.PIFailed, drawErr.Error()
		if errors.Is(drawErr, ErrIllustrationNameTaken) {
			r.Error = "Tên " + r.Name + " đã có trong thư viện — đổi sang dùng lại hình đó hoặc bỏ qua."
		}
	} else {
		r.State, r.IllustrationID, r.Name = domain.PIDrawn, made.ID, made.Name
	}
	// Written with a fresh context: a cancelled request must not leave the row stuck at "drawing".
	if err := uc.repo.UpdateProjectIllustration(context.WithoutCancel(ctx), r); err != nil {
		return r, err
	}
	if drawErr != nil {
		return r, drawErr
	}
	return uc.row(ctx, projectID, rowID)
}

// SetSkipped lets the Creator go on without one drawing, or take it back.
func (uc *ProjectIllustrationsUseCase) SetSkipped(ctx context.Context, projectID, rowID string, skipped bool) (domain.ProjectIllustration, error) {
	r, err := uc.row(ctx, projectID, rowID)
	if err != nil {
		return r, err
	}
	// Taking a skip back: back to the drawing it had, or to "planned".
	// Ready() treats drawn and reused alike, so the label is only for display.
	switch {
	case skipped:
		r.State = domain.PISkipped
	case r.IllustrationID == "":
		r.State = domain.PIPlanned
	case r.Illustration != nil && r.Illustration.Builtin:
		r.State = domain.PIReused
	default:
		r.State = domain.PIDrawn
	}
	if err := uc.repo.UpdateProjectIllustration(ctx, r); err != nil {
		return r, err
	}
	return uc.row(ctx, projectID, rowID)
}

// StageReport is the illustrations step's progress, sent as it moves: planning,
// then drawing with counts over the whole list.
type StageReport struct {
	Phase   string // "plan" | "draw"
	Total   int    // rows that needed drawing when the draw began
	Done    int    // of those, finished (drawn or failed)
	Failed  int
	Reused  int // rows served by the library, never drawn
	Planned int // rows in the list
}

// DrawMissing draws every planned or failed row of the video, several at once.
// A drawing that fails stays "failed" with its reason; the Creator retries or
// skips it. Only a cancelled context stops the batch.
func (uc *ProjectIllustrationsUseCase) DrawMissing(
	ctx context.Context, projectID, model string, report func(StageReport),
) error {
	rows, err := uc.repo.ListProjectIllustrations(ctx, projectID)
	if err != nil {
		return err
	}
	base := StageReport{Phase: "draw", Planned: len(rows)}
	var todo []domain.ProjectIllustration
	for _, r := range rows {
		switch r.State {
		case domain.PIPlanned, domain.PIFailed:
			todo = append(todo, r)
		case domain.PIDrawing:
			if !uc.inFlight(r.ID) {
				todo = append(todo, r) // cut short by a restart
			}
		case domain.PIReused:
			base.Reused++
		}
	}
	base.Total = len(todo)
	if report == nil {
		report = func(StageReport) {}
	}
	report(base)
	if len(todo) == 0 {
		return nil
	}

	var mu sync.Mutex
	sem := make(chan struct{}, uc.concurrency)
	var wg sync.WaitGroup
	for _, r := range todo {
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			wg.Wait()
			return ctx.Err()
		}
		wg.Add(1)
		go func(r domain.ProjectIllustration) {
			defer wg.Done()
			defer func() { <-sem }()
			_, drawErr := uc.Draw(ctx, projectID, r.ID, model)
			mu.Lock()
			base.Done++
			if drawErr != nil {
				base.Failed++
			}
			snapshot := base
			mu.Unlock()
			report(snapshot)
		}(r)
	}
	wg.Wait()
	return ctx.Err()
}

// Prepare is the illustrations step (CR-045): plan the list when the video has
// none, draw what is missing, and return the rows still waiting for the
// Creator. Re-running it never re-plans an existing list — "Lập lại danh sách"
// does that — it only draws what is still planned or failed.
func (uc *ProjectIllustrationsUseCase) Prepare(
	ctx context.Context, projectID, model string, report func(StageReport),
) ([]domain.ProjectIllustration, error) {
	if report == nil {
		report = func(StageReport) {}
	}
	rows, err := uc.repo.ListProjectIllustrations(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		report(StageReport{Phase: "plan"})
		if _, err = uc.Plan(ctx, projectID, model); err != nil {
			return nil, fmt.Errorf("lập danh sách hình minh hoạ: %w", err)
		}
	}
	if err := uc.DrawMissing(ctx, projectID, model, report); err != nil {
		return nil, err
	}
	pending, _, err := uc.Gate(ctx, projectID)
	return pending, err
}

// Gate is the code step's check: the rows that still hold it back, and whether
// the list was planned at all. A list made before CR-045 has rows but no mark.
func (uc *ProjectIllustrationsUseCase) Gate(ctx context.Context, projectID string) ([]domain.ProjectIllustration, bool, error) {
	rows, err := uc.repo.ListProjectIllustrations(ctx, projectID)
	if err != nil {
		return nil, false, err
	}
	planned := len(rows) > 0
	if !planned {
		if planned, err = uc.repo.IllustrationsPlanned(ctx, projectID); err != nil {
			return nil, false, err
		}
	}
	var pending []domain.ProjectIllustration
	for _, r := range rows {
		if !r.Ready() {
			pending = append(pending, r)
		}
	}
	return pending, planned, nil
}

// DeleteDrawing removes the row's drawing from the library — for an AI drawing
// the Creator does not want, so it does not litter the library — and skips the
// row. Only this video's own unapproved draft can go: an approved or reused
// drawing may be serving other videos. "Dùng lại" on the row then plans it
// again, to be drawn afresh.
func (uc *ProjectIllustrationsUseCase) DeleteDrawing(ctx context.Context, projectID, rowID string) (domain.ProjectIllustration, error) {
	r, err := uc.row(ctx, projectID, rowID)
	if err != nil {
		return r, err
	}
	ill := r.Illustration
	if r.State != domain.PIDrawn || ill == nil || ill.Builtin || ill.Status != domain.IllustrationDraft {
		return r, ErrIllustrationNotDeletable
	}
	if err := uc.library.Delete(ctx, ill.ID); err != nil && !errors.Is(err, ErrIllustrationNotFound) {
		return r, err
	}
	r.State, r.IllustrationID, r.Error, r.Illustration = domain.PISkipped, "", "", nil
	if err := uc.repo.UpdateProjectIllustration(ctx, r); err != nil {
		return r, err
	}
	return uc.row(ctx, projectID, rowID)
}

// ForCode lists the approved drawings with code the Remotion Engineer may use:
// this video's own first (all of them), then up to maxCodeLibraryDrawings more
// from the rest of the library (exemplars included).
// Kit built-ins are not listed: they are always imported.
func (uc *ProjectIllustrationsUseCase) ForCode(ctx context.Context, projectID string) ([]LibraryDrawing, error) {
	rows, err := uc.repo.ListProjectIllustrations(ctx, projectID)
	if err != nil {
		return nil, err
	}
	all, err := uc.library.List(ctx, IllustrationFilter{})
	if err != nil {
		return nil, err
	}
	var out []LibraryDrawing
	seen := map[string]bool{}
	extra := 0
	take := func(i domain.Illustration, own bool) {
		if seen[i.Name] || strings.TrimSpace(i.Code) == "" {
			return
		}
		if !i.Builtin && i.Status != domain.IllustrationApproved {
			return
		}
		if !own {
			if extra >= maxCodeLibraryDrawings {
				return
			}
			extra++
		}
		seen[i.Name] = true
		out = append(out, LibraryDrawing{Name: i.Name, Usage: i.Usage, Description: i.Description, Code: i.Code})
	}
	for _, r := range rows {
		if r.State != domain.PISkipped && r.Illustration != nil {
			take(*r.Illustration, true)
		}
	}
	for _, i := range all {
		take(i, false)
	}
	return out, nil
}
