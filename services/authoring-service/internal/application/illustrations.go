package application

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"authoring/internal/domain"
)

// The illustration library: folders of drawings, each with a PNG
// still and a short GIF rendered by the rendering service, reviewed and
// approved by the Creator before the Remotion Engineer may use it.

var (
	ErrIllustrationNotFound  = errors.New("illustration not found")
	ErrIllustrationReadOnly  = errors.New("hình có sẵn của hệ thống và Hình mẫu chỉ đọc: không sửa, vẽ lại hay xoá được")
	ErrIllustrationNameTaken = errors.New("illustration name is already used")
	ErrFolderNotFound        = errors.New("illustration folder not found")
	ErrFolderTaken           = errors.New("illustration folder id is already used")
	ErrFolderNotEmpty        = errors.New("illustration folder still holds drawings")
	ErrFolderReadOnly        = errors.New("system folders cannot be deleted")
	// ErrIllustrationInvalid wraps a drawing whose code the renderer refused;
	// the diagnostics travel with it so the editor can point at the line.
	ErrIllustrationInvalid = errors.New("illustration code did not pass the check")

	// Deleting a drawing a video may still render, and the Hình mẫu folder.
	ErrIllustrationInUse        = errors.New("illustration is used by a project that has not reached its result")
	ErrIllustrationUsageUnknown = errors.New("không kiểm tra được hình có đang được dự án nào dùng không — thử lại sau")
	// ErrIllustrationLinked is the repository's answer when a project it was
	// not told about links the drawing: the check is run again.
	ErrIllustrationLinked = errors.New("illustration is linked to a project that was not checked")
	ErrExemplarFolder     = errors.New("thư mục Hình mẫu chỉ nhận hình qua nút \"Đặt làm mẫu\"")
	ErrExemplarLimit      = fmt.Errorf("thư mục Hình mẫu đã đủ %d hình — bỏ bớt một Hình mẫu trước", domain.MaxExemplars)
	ErrAlreadyExemplar    = errors.New("hình này đã có bản Hình mẫu")
	ErrNotExemplarSource  = errors.New("chỉ hình đã duyệt, không phải hình có sẵn của hệ thống, mới đặt làm mẫu được")
	ErrNotExemplar        = errors.New("hình này không phải Hình mẫu")
)

// IllustrationUser is a project whose drawing list or code uses a drawing.
type IllustrationUser struct {
	ProjectID string `json:"id"`
	Topic     string `json:"topic"`
}

// IllustrationInUseError names the projects that still need a drawing.
type IllustrationInUseError struct{ Projects []IllustrationUser }

func (e *IllustrationInUseError) Error() string {
	names := make([]string, 0, len(e.Projects))
	for _, p := range e.Projects {
		if p.Topic != "" {
			names = append(names, "\""+p.Topic+"\"")
		} else {
			names = append(names, p.ProjectID)
		}
	}
	return "Đang được dùng trong dự án " + strings.Join(names, ", ") +
		" — xoá được sau khi dự án tới bước Kết quả."
}

func (e *IllustrationInUseError) Unwrap() error { return ErrIllustrationInUse }

// ProjectStatusPort reads a project's status from the orchestrator. A project
// that no longer exists answers domain.ErrProjectNotFound.
type ProjectStatusPort interface {
	GetStatus(ctx context.Context, projectID string) (domain.ProjectStatus, error)
}

// IllustrationFilter narrows a library listing. Empty fields match everything.
type IllustrationFilter struct {
	FolderID string
	Query    string // matched against name, title, tags and description
	Status   domain.IllustrationStatus
}

// IllustrationRepoPort is the library's persistence.
type IllustrationRepoPort interface {
	ListIllustrationFolders(ctx context.Context) ([]domain.IllustrationFolder, error)
	CreateIllustrationFolder(ctx context.Context, f domain.IllustrationFolder) (domain.IllustrationFolder, error)
	DeleteIllustrationFolder(ctx context.Context, id string) error
	ListIllustrations(ctx context.Context, f IllustrationFilter) ([]domain.Illustration, error)
	GetIllustration(ctx context.Context, id string) (domain.Illustration, error)
	CreateIllustration(ctx context.Context, i domain.Illustration) (domain.Illustration, error)
	UpdateIllustration(ctx context.Context, i domain.Illustration) (domain.Illustration, error)
	// DeleteIllustration deletes a non-kit row unless a project outside
	// checked links it (ErrIllustrationLinked). A missing row is not an error.
	DeleteIllustration(ctx context.Context, id string, checked []string) error
	// FindIllustrationUsers lists the projects whose drawing list links the
	// drawing or whose code names its component.
	FindIllustrationUsers(ctx context.Context, id, name string) ([]IllustrationUser, error)
	// CreateExemplar stores an exemplar row, refusing (ErrExemplarLimit,
	// ErrAlreadyExemplar) under a lock so two calls cannot pass the limit.
	CreateExemplar(ctx context.Context, i domain.Illustration, limit int) (domain.Illustration, error)
	// ReleaseExemplar turns an original exemplar back into an ordinary
	// approved drawing filed in folderID.
	ReleaseExemplar(ctx context.Context, id, folderID string) (domain.Illustration, error)
	SaveIllustrationPreview(ctx context.Context, id string, version int, png, gif []byte) error
	// GetIllustrationPreview returns the stored preview, or empty slices when
	// none has been rendered for the current version.
	GetIllustrationPreview(ctx context.Context, id string) (png, gif []byte, err error)
}

// PreviewDiagnostic is one problem the renderer found in a drawing's code.
type PreviewDiagnostic = domain.CodeFinding

// IllustrationPreview is the renderer's answer for one drawing. Warnings are
// style findings that do not block saving.
type IllustrationPreview struct {
	OK          bool
	Diagnostics []PreviewDiagnostic
	Warnings    []PreviewDiagnostic
	PNG, GIF    []byte
}

// IllustrationRendererPort renders a drawing (rendering's /v1/illustrations/preview).
// code "" renders a built-in kit component by name.
type IllustrationRendererPort interface {
	PreviewIllustration(ctx context.Context, name, code string, props map[string]any, gif bool, kind domain.IllustrationKind) (IllustrationPreview, error)
}

// InvalidIllustrationError carries the renderer's diagnostics.
type InvalidIllustrationError struct{ Diagnostics []PreviewDiagnostic }

func (e *InvalidIllustrationError) Error() string {
	msgs := make([]string, 0, len(e.Diagnostics))
	for _, d := range e.Diagnostics {
		if d.Line != nil {
			msgs = append(msgs, fmt.Sprintf("dòng %d: %s", *d.Line, d.Message))
		} else {
			msgs = append(msgs, d.Message)
		}
	}
	return "code hình minh hoạ chưa qua kiểm tra: " + strings.Join(msgs, "; ")
}

func (e *InvalidIllustrationError) Unwrap() error { return ErrIllustrationInvalid }

// IllustrationsUseCase manages the library.
// StylePromptPort reads the active row of a prompt role: here, the
// illustration style rules the Creator may have rewritten in the prompt library.
type StylePromptPort interface {
	GetActive(ctx context.Context, role domain.PromptRole) (domain.Prompt, error)
}

type IllustrationsUseCase struct {
	repo     IllustrationRepoPort
	renderer IllustrationRendererPort
	drawer   *illustrationDrawer // nil = AI drawing disabled
	projects ProjectStatusPort   // nil = a drawing any project uses cannot be deleted
	styles   StylePromptPort     // nil = the style rules cannot be read
}

func NewIllustrationsUseCase(repo IllustrationRepoPort, renderer IllustrationRendererPort) *IllustrationsUseCase {
	return &IllustrationsUseCase{repo: repo, renderer: renderer}
}

// WithProjectStatus lets Delete tell a project still working from one that is done.
func (uc *IllustrationsUseCase) WithProjectStatus(p ProjectStatusPort) *IllustrationsUseCase {
	uc.projects = p
	return uc
}

// WithStylePrompts gives the use case the prompt library the style rules are read from.
func (uc *IllustrationsUseCase) WithStylePrompts(p StylePromptPort) *IllustrationsUseCase {
	uc.styles = p
	return uc
}

// StyleRules is the illustration style text in force: the active row of the
// illustration_style prompt role. It is what the AI illustrator is held to
// and what the library shows; reading it failing is an error, never a quiet
// fall back to other text.
func (uc *IllustrationsUseCase) StyleRules(ctx context.Context) (string, error) {
	if uc.styles == nil {
		return "", errors.New("illustration style rules: no prompt library configured")
	}
	p, err := uc.styles.GetActive(ctx, domain.RoleIllustrationStyle)
	if err != nil {
		return "", fmt.Errorf("illustration style rules: %w", err)
	}
	return strings.TrimSpace(p.TemplateText), nil
}

func (uc *IllustrationsUseCase) Folders(ctx context.Context) ([]domain.IllustrationFolder, error) {
	return uc.repo.ListIllustrationFolders(ctx)
}

func (uc *IllustrationsUseCase) CreateFolder(ctx context.Context, f domain.IllustrationFolder) (domain.IllustrationFolder, error) {
	f.ID = strings.TrimSpace(f.ID)
	f.Name = strings.TrimSpace(f.Name)
	if err := domain.ValidateFolderID(f.ID); err != nil {
		return f, err
	}
	if f.Name == "" {
		return f, fmt.Errorf("tên thư mục là bắt buộc")
	}
	f.IsSystem = false
	return uc.repo.CreateIllustrationFolder(ctx, f)
}

// DeleteFolder removes an empty Creator folder.
func (uc *IllustrationsUseCase) DeleteFolder(ctx context.Context, id string) error {
	folders, err := uc.repo.ListIllustrationFolders(ctx)
	if err != nil {
		return err
	}
	for _, f := range folders {
		if f.ID != id {
			continue
		}
		if f.IsSystem {
			return ErrFolderReadOnly
		}
		inside, err := uc.repo.ListIllustrations(ctx, IllustrationFilter{FolderID: id})
		if err != nil {
			return err
		}
		if len(inside) > 0 {
			return ErrFolderNotEmpty
		}
		return uc.repo.DeleteIllustrationFolder(ctx, id)
	}
	return ErrFolderNotFound
}

func (uc *IllustrationsUseCase) List(ctx context.Context, f IllustrationFilter) ([]domain.Illustration, error) {
	return uc.repo.ListIllustrations(ctx, f)
}

func (uc *IllustrationsUseCase) Get(ctx context.Context, id string) (domain.Illustration, error) {
	return uc.repo.GetIllustration(ctx, id)
}

// writableFolder is a folder a drawing may be saved into: it exists and is
// not the Hình mẫu folder, which only "Đặt làm mẫu" fills.
func (uc *IllustrationsUseCase) writableFolder(ctx context.Context, id string) error {
	if id == domain.ExemplarFolderID {
		return ErrExemplarFolder
	}
	return uc.folderExists(ctx, id)
}

func (uc *IllustrationsUseCase) folderExists(ctx context.Context, id string) error {
	folders, err := uc.repo.ListIllustrationFolders(ctx)
	if err != nil {
		return err
	}
	for _, f := range folders {
		if f.ID == id {
			return nil
		}
	}
	return ErrFolderNotFound
}

func normalizeIllustration(i domain.Illustration) (domain.Illustration, error) {
	i.Name = strings.TrimSpace(i.Name)
	i.Title = strings.TrimSpace(i.Title)
	i.Description = strings.TrimSpace(i.Description)
	i.Usage = strings.TrimSpace(i.Usage)
	i.Tags = domain.NormalizeTags(i.Tags)
	if err := domain.ValidateIllustrationName(i.Name); err != nil {
		return i, err
	}
	if i.Title == "" {
		i.Title = i.Name
	}
	if strings.TrimSpace(i.Code) == "" {
		return i, fmt.Errorf("code là bắt buộc")
	}
	return i, nil
}

// render checks and previews code as a figure or a backdrop (kind); a
// refused drawing is an InvalidIllustrationError, a renderer that cannot run
// is a plain error.
func (uc *IllustrationsUseCase) render(ctx context.Context, name, code string, kind domain.IllustrationKind) (IllustrationPreview, error) {
	out, err := uc.renderer.PreviewIllustration(ctx, name, code, nil, true, domain.KindOrFigure(kind))
	if err != nil {
		return out, fmt.Errorf("dựng xem trước: %w", err)
	}
	if !out.OK {
		return out, &InvalidIllustrationError{Diagnostics: out.Diagnostics}
	}
	return out, nil
}

// Try renders code without saving anything — the editor's "Xem trước" button.
func (uc *IllustrationsUseCase) Try(ctx context.Context, name, code string, kind domain.IllustrationKind) (IllustrationPreview, error) {
	if err := domain.ValidateIllustrationName(strings.TrimSpace(name)); err != nil {
		return IllustrationPreview{}, err
	}
	return uc.render(ctx, strings.TrimSpace(name), code, kind)
}

// Create stores a new library drawing as a draft. The code must pass the
// renderer's check first: the library never holds a drawing that cannot render.
func (uc *IllustrationsUseCase) Create(ctx context.Context, i domain.Illustration) (domain.Illustration, error) {
	i, err := normalizeIllustration(i)
	if err != nil {
		return i, err
	}
	if err := uc.writableFolder(ctx, i.FolderID); err != nil {
		return i, err
	}
	preview, err := uc.render(ctx, i.Name, i.Code, i.Kind)
	if err != nil {
		return i, err
	}
	i.Builtin, i.Exemplar, i.Status, i.Version = false, false, domain.IllustrationDraft, 1
	i.SourceID, i.HomeFolderID = "", ""
	i.Warnings = preview.Warnings
	saved, err := uc.repo.CreateIllustration(ctx, i)
	if err != nil {
		return saved, err
	}
	if err := uc.repo.SaveIllustrationPreview(ctx, saved.ID, saved.Version, preview.PNG, preview.GIF); err != nil {
		return saved, err
	}
	saved.HasPreview = true
	return saved, nil
}

// Update edits a library drawing. A code change goes back to draft and gets a
// new version, since what the Creator approved is no longer what would render.
func (uc *IllustrationsUseCase) Update(ctx context.Context, id string, i domain.Illustration) (domain.Illustration, error) {
	existing, err := uc.repo.GetIllustration(ctx, id)
	if err != nil {
		return existing, err
	}
	if existing.ReadOnly() {
		return existing, ErrIllustrationReadOnly
	}
	if strings.TrimSpace(i.Code) == "" {
		i.Code = existing.Code
	}
	if i.FolderID == "" {
		i.FolderID = existing.FolderID
	}
	if i.Name == "" {
		i.Name = existing.Name
	}
	if i, err = normalizeIllustration(i); err != nil {
		return i, err
	}
	if err := uc.writableFolder(ctx, i.FolderID); err != nil {
		return i, err
	}
	i.ID, i.Builtin, i.Exemplar = id, false, false
	i.Status, i.Version, i.Warnings = existing.Status, existing.Version, existing.Warnings
	codeChanged := i.Code != existing.Code || i.Name != existing.Name
	var preview IllustrationPreview
	if codeChanged {
		if preview, err = uc.render(ctx, i.Name, i.Code, i.Kind); err != nil {
			return i, err
		}
		i.Status, i.Version, i.Warnings = domain.IllustrationDraft, existing.Version+1, preview.Warnings
	}
	saved, err := uc.repo.UpdateIllustration(ctx, i)
	if err != nil {
		return saved, err
	}
	if codeChanged {
		if err := uc.repo.SaveIllustrationPreview(ctx, id, saved.Version, preview.PNG, preview.GIF); err != nil {
			return saved, err
		}
		saved.HasPreview = true
	}
	return saved, nil
}

// SetStatus approves a draft, or sends an approved drawing back to draft.
func (uc *IllustrationsUseCase) SetStatus(ctx context.Context, id string, status domain.IllustrationStatus) (domain.Illustration, error) {
	existing, err := uc.repo.GetIllustration(ctx, id)
	if err != nil {
		return existing, err
	}
	if existing.ReadOnly() {
		return existing, ErrIllustrationReadOnly
	}
	if status != domain.IllustrationDraft && status != domain.IllustrationApproved {
		return existing, fmt.Errorf("trạng thái %q không hợp lệ", status)
	}
	existing.Status = status
	return uc.repo.UpdateIllustration(ctx, existing)
}

// Delete removes a drawing from the library, unless a video that has not
// reached its result screen still uses it.
func (uc *IllustrationsUseCase) Delete(ctx context.Context, id string) error {
	return uc.deleteDrawing(ctx, id, "")
}

// deleteDrawing is Delete for the project ignore, which removes its own draft
// and so must not be stopped by its own use of it.
func (uc *IllustrationsUseCase) deleteDrawing(ctx context.Context, id, ignore string) error {
	existing, err := uc.repo.GetIllustration(ctx, id)
	if err != nil {
		return err
	}
	if existing.ReadOnly() {
		return ErrIllustrationReadOnly
	}
	return uc.removeUnused(ctx, existing, ignore)
}

// removeUnused deletes ill once no project still working uses it. The check
// reads the orchestrator, so it cannot share a transaction with the delete:
// the delete itself refuses when a project it was not checked against links
// the drawing meanwhile, and the check then runs again.
func (uc *IllustrationsUseCase) removeUnused(ctx context.Context, ill domain.Illustration, ignore string) error {
	for attempt := 0; attempt < 2; attempt++ {
		checked, err := uc.checkUnused(ctx, ill, ignore)
		if err != nil {
			return err
		}
		if err := uc.repo.DeleteIllustration(ctx, ill.ID, checked); !errors.Is(err, ErrIllustrationLinked) {
			return err
		}
	}
	return fmt.Errorf("%w: danh sách dự án dùng hình này vừa thay đổi", ErrIllustrationUsageUnknown)
}

// checkUnused returns the projects that use ill but are done with it (plus
// ignore), or IllustrationInUseError naming the ones that are not.
func (uc *IllustrationsUseCase) checkUnused(ctx context.Context, ill domain.Illustration, ignore string) ([]string, error) {
	users, err := uc.repo.FindIllustrationUsers(ctx, ill.ID, ill.Name)
	if err != nil {
		return nil, err
	}
	checked := []string{}
	if ignore != "" {
		checked = append(checked, ignore)
	}
	var busy []IllustrationUser
	for _, u := range users {
		if u.ProjectID == ignore {
			continue
		}
		if uc.projects == nil {
			return nil, ErrIllustrationUsageUnknown
		}
		status, err := uc.projects.GetStatus(ctx, u.ProjectID)
		switch {
		case errors.Is(err, domain.ErrProjectNotFound):
		case err != nil:
			return nil, fmt.Errorf("%w (%v)", ErrIllustrationUsageUnknown, err)
		case !status.DoneWithLibrary():
			busy = append(busy, u)
			continue
		}
		checked = append(checked, u.ProjectID)
	}
	if len(busy) > 0 {
		return nil, &IllustrationInUseError{Projects: busy}
	}
	return checked, nil
}

// Exemplars are the Hình mẫu, oldest first: the order the AI drawer reads them.
func (uc *IllustrationsUseCase) Exemplars(ctx context.Context) ([]domain.Illustration, error) {
	rows, err := uc.repo.ListIllustrations(ctx, IllustrationFilter{FolderID: domain.ExemplarFolderID})
	if err != nil {
		return nil, err
	}
	out := []domain.Illustration{}
	for _, i := range rows {
		if i.Exemplar {
			out = append(out, i)
		}
	}
	sort.SliceStable(out, func(a, b int) bool {
		if out[a].CreatedAt != out[b].CreatedAt {
			return out[a].CreatedAt < out[b].CreatedAt
		}
		return out[a].Name < out[b].Name
	})
	return out, nil
}

// MakeExemplar copies an approved drawing into the Hình mẫu folder under a
// name of its own (<Name>Mau), rendered like any drawing before it is kept.
func (uc *IllustrationsUseCase) MakeExemplar(ctx context.Context, id string) (domain.Illustration, error) {
	src, err := uc.repo.GetIllustration(ctx, id)
	if err != nil {
		return src, err
	}
	switch {
	case src.Exemplar:
		return src, ErrAlreadyExemplar
	case src.Builtin, src.Status != domain.IllustrationApproved, strings.TrimSpace(src.Code) == "":
		return src, ErrNotExemplarSource
	}
	all, err := uc.repo.ListIllustrations(ctx, IllustrationFilter{})
	if err != nil {
		return src, err
	}
	taken, exemplars := map[string]bool{}, 0
	for _, i := range all {
		taken[i.Name] = true
		if i.Exemplar {
			exemplars++
			if i.SourceID == src.ID {
				return src, ErrAlreadyExemplar
			}
		}
	}
	if exemplars >= domain.MaxExemplars {
		return src, ErrExemplarLimit
	}
	name := domain.ExemplarCopyName(src.Name, func(n string) bool { return taken[n] })
	code, usage, err := domain.RenameExport(src.Code, src.Usage, name)
	if err != nil {
		return src, err
	}
	preview, err := uc.render(ctx, name, code, src.Kind)
	if err != nil {
		return src, err
	}
	saved, err := uc.repo.CreateExemplar(ctx, domain.Illustration{
		Name: name, Title: src.Title, FolderID: domain.ExemplarFolderID, Tags: src.Tags,
		Description: src.Description, Usage: usage, Code: code, Exemplar: true, SourceID: src.ID, Kind: src.Kind,
		Status: domain.IllustrationApproved, Version: 1, Warnings: preview.Warnings,
	}, domain.MaxExemplars)
	if err != nil {
		return saved, err
	}
	if err := uc.repo.SaveIllustrationPreview(ctx, saved.ID, saved.Version, preview.PNG, preview.GIF); err != nil {
		return saved, err
	}
	saved.HasPreview = true
	return saved, nil
}

// UnmakeExemplar takes a drawing out of the Hình mẫu. A copy is deleted (its
// source stays in the library), unless a video still working uses it; one of
// the original exemplars goes back to its folder as an ordinary drawing, and
// is returned.
func (uc *IllustrationsUseCase) UnmakeExemplar(ctx context.Context, id string) (*domain.Illustration, error) {
	ill, err := uc.repo.GetIllustration(ctx, id)
	if err != nil {
		return nil, err
	}
	if !ill.Exemplar {
		return nil, ErrNotExemplar
	}
	if ill.HomeFolderID == "" {
		return nil, uc.removeUnused(ctx, ill, "")
	}
	if err := uc.folderExists(ctx, ill.HomeFolderID); err != nil {
		return nil, fmt.Errorf("thư mục cũ %q của hình mẫu không còn: %w", ill.HomeFolderID, err)
	}
	out, err := uc.repo.ReleaseExemplar(ctx, id, ill.HomeFolderID)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Preview returns the stored PNG and GIF, rendering them first when the
// current version has none (built-ins are rendered on first view).
func (uc *IllustrationsUseCase) Preview(ctx context.Context, id string) (png, gif []byte, err error) {
	png, gif, err = uc.repo.GetIllustrationPreview(ctx, id)
	if err != nil || len(png) > 0 {
		return png, gif, err
	}
	return uc.Rerender(ctx, id)
}

// Rerender renders the preview of the current version again and stores it.
func (uc *IllustrationsUseCase) Rerender(ctx context.Context, id string) (png, gif []byte, err error) {
	existing, err := uc.repo.GetIllustration(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	// Kit built-ins have no code (it ships in the image); exemplars carry theirs.
	out, err := uc.render(ctx, existing.Name, existing.Code, existing.Kind)
	if err != nil {
		return nil, nil, err
	}
	if err := uc.repo.SaveIllustrationPreview(ctx, id, existing.Version, out.PNG, out.GIF); err != nil {
		return nil, nil, err
	}
	return out.PNG, out.GIF, nil
}
