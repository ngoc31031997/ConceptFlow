package application

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"authoring/internal/domain"
)

// CR-044 — the drawings one video needs. After the storyboard, a model reads it
// against the library and lists what can be reused and what must be drawn;
// each missing drawing goes through the AI drawer; the Creator reviews them;
// the code step waits until every one is approved or skipped, then hands the
// approved drawings to the Remotion Engineer.

const (
	maxPlannedDrawings = 10
	maxCodeDrawings    = 40
	plannerRole        = "illustration_planner"
)

//go:embed planner_prompt_vi.txt
var plannerPromptVI string

// ErrIllustrationsPending stops the code step while the video still has
// drawings that are neither approved nor skipped.
type ErrIllustrationsPending struct{ Rows []domain.ProjectIllustration }

func (e *ErrIllustrationsPending) Error() string {
	names := make([]string, 0, len(e.Rows))
	for _, r := range e.Rows {
		names = append(names, r.Name)
	}
	return fmt.Sprintf("Còn %d hình minh hoạ chưa duyệt (%s). Duyệt hoặc bỏ qua chúng ở mục \"Hình minh hoạ của video\" trong tab Code, rồi chạy lại bước Code.",
		len(e.Rows), strings.Join(names, ", "))
}

// ProjectIllustrationRepoPort persists the per-video list.
type ProjectIllustrationRepoPort interface {
	ListProjectIllustrations(ctx context.Context, projectID string) ([]domain.ProjectIllustration, error)
	ReplaceProjectIllustrations(ctx context.Context, projectID string, rows []domain.ProjectIllustration) error
	UpdateProjectIllustration(ctx context.Context, row domain.ProjectIllustration) error
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
}

func NewProjectIllustrationsUseCase(
	repo ProjectIllustrationRepoPort, library *IllustrationsUseCase, storyboards StoryboardReaderPort,
	llm LLMProviderPort, recorder *LLMUsageRecorder, maxTokens int,
) *ProjectIllustrationsUseCase {
	return &ProjectIllustrationsUseCase{repo: repo, library: library, storyboards: storyboards, llm: llm,
		recorder: recorder, maxTokens: maxTokens}
}

func (uc *ProjectIllustrationsUseCase) List(ctx context.Context, projectID string) ([]domain.ProjectIllustration, error) {
	return uc.repo.ListProjectIllustrations(ctx, projectID)
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
		return nil, fmt.Errorf("chưa có storyboard — hãy chạy bước Visual trước")
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
		if len(rows) >= maxPlannedDrawings {
			break
		}
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
	return uc.repo.ListProjectIllustrations(ctx, projectID)
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
	if r.State == domain.PIDrawing {
		return r, fmt.Errorf("hình %s đang được vẽ", r.Name)
	}
	if r.State == domain.PIReused || (r.State == domain.PIDrawn && r.Illustration != nil) {
		return r, fmt.Errorf("hình %s đã có — dùng \"Vẽ lại bằng AI\" trên hình đó", r.Name)
	}
	r.State, r.Error = domain.PIDrawing, ""
	if err := uc.repo.UpdateProjectIllustration(ctx, r); err != nil {
		return r, err
	}
	made, drawErr := uc.library.Draw(ctx, DrawRequest{Description: r.Description, FolderID: r.FolderID, Name: r.Name, Model: model})
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

// Ensure runs at the head of the code step: plan if the video has no list yet,
// draw what is planned, and return the rows that still hold the step back.
func (uc *ProjectIllustrationsUseCase) Ensure(ctx context.Context, projectID, model string) ([]domain.ProjectIllustration, error) {
	rows, err := uc.repo.ListProjectIllustrations(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		if rows, err = uc.Plan(ctx, projectID, model); err != nil {
			return nil, fmt.Errorf("lập danh sách hình minh hoạ: %w", err)
		}
	}
	for _, r := range rows {
		if r.State == domain.PIPlanned {
			if _, err := uc.Draw(ctx, projectID, r.ID, model); err != nil && ctx.Err() != nil {
				return nil, ctx.Err()
			} // a drawing that fails stays "failed"; the Creator retries or skips it
		}
	}
	if rows, err = uc.repo.ListProjectIllustrations(ctx, projectID); err != nil {
		return nil, err
	}
	var pending []domain.ProjectIllustration
	for _, r := range rows {
		if !r.Ready() {
			pending = append(pending, r)
		}
	}
	return pending, nil
}

// ForCode lists the approved drawings with code the Remotion Engineer may use:
// this video's own first, then the rest of the library (exemplars included).
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
	take := func(i domain.Illustration) {
		if len(out) >= maxCodeDrawings || seen[i.Name] || strings.TrimSpace(i.Code) == "" {
			return
		}
		if !i.Builtin && i.Status != domain.IllustrationApproved {
			return
		}
		seen[i.Name] = true
		out = append(out, LibraryDrawing{Name: i.Name, Usage: i.Usage, Description: i.Description, Code: i.Code})
	}
	for _, r := range rows {
		if r.State != domain.PISkipped && r.Illustration != nil {
			take(*r.Illustration)
		}
	}
	for _, i := range all {
		take(i)
	}
	return out, nil
}
