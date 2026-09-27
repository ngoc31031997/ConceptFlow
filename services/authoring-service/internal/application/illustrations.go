package application

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"authoring/internal/domain"
)

// CR-044 — the illustration library: folders of drawings, each with a PNG
// still and a short GIF rendered by the rendering service, reviewed and
// approved by the Creator before the Remotion Engineer may use it.

var (
	ErrIllustrationNotFound  = errors.New("illustration not found")
	ErrIllustrationReadOnly  = errors.New("built-in illustrations are read-only")
	ErrIllustrationNameTaken = errors.New("illustration name is already used")
	ErrFolderNotFound        = errors.New("illustration folder not found")
	ErrFolderTaken           = errors.New("illustration folder id is already used")
	ErrFolderNotEmpty        = errors.New("illustration folder still holds drawings")
	ErrFolderReadOnly        = errors.New("system folders cannot be deleted")
	// ErrIllustrationInvalid wraps a drawing whose code the renderer refused;
	// the diagnostics travel with it so the editor can point at the line.
	ErrIllustrationInvalid = errors.New("illustration code did not pass the check")
)

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
	DeleteIllustration(ctx context.Context, id string) error
	SaveIllustrationPreview(ctx context.Context, id string, version int, png, gif []byte) error
	// GetIllustrationPreview returns the stored preview, or empty slices when
	// none has been rendered for the current version.
	GetIllustrationPreview(ctx context.Context, id string) (png, gif []byte, err error)
}

// PreviewDiagnostic is one problem the renderer found in a drawing's code.
type PreviewDiagnostic = domain.CodeFinding

// IllustrationPreview is the renderer's answer for one drawing. Warnings are
// style findings that do not block saving (CR-044, agreed with the Creator).
type IllustrationPreview struct {
	OK          bool
	Diagnostics []PreviewDiagnostic
	Warnings    []PreviewDiagnostic
	PNG, GIF    []byte
}

// IllustrationRendererPort renders a drawing (rendering's /v1/illustrations/preview).
// code "" renders a built-in kit component by name.
type IllustrationRendererPort interface {
	PreviewIllustration(ctx context.Context, name, code string, props map[string]any, gif bool) (IllustrationPreview, error)
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
type IllustrationsUseCase struct {
	repo     IllustrationRepoPort
	renderer IllustrationRendererPort
	drawer   *illustrationDrawer // nil = AI drawing disabled
}

func NewIllustrationsUseCase(repo IllustrationRepoPort, renderer IllustrationRendererPort) *IllustrationsUseCase {
	return &IllustrationsUseCase{repo: repo, renderer: renderer}
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

// render checks and previews code; a refused drawing is an
// InvalidIllustrationError, a renderer that cannot run is a plain error.
func (uc *IllustrationsUseCase) render(ctx context.Context, name, code string) (IllustrationPreview, error) {
	out, err := uc.renderer.PreviewIllustration(ctx, name, code, nil, true)
	if err != nil {
		return out, fmt.Errorf("dựng xem trước: %w", err)
	}
	if !out.OK {
		return out, &InvalidIllustrationError{Diagnostics: out.Diagnostics}
	}
	return out, nil
}

// Try renders code without saving anything — the editor's "Xem trước" button.
func (uc *IllustrationsUseCase) Try(ctx context.Context, name, code string) (IllustrationPreview, error) {
	if err := domain.ValidateIllustrationName(strings.TrimSpace(name)); err != nil {
		return IllustrationPreview{}, err
	}
	return uc.render(ctx, strings.TrimSpace(name), code)
}

// Create stores a new library drawing as a draft. The code must pass the
// renderer's check first: the library never holds a drawing that cannot render.
func (uc *IllustrationsUseCase) Create(ctx context.Context, i domain.Illustration) (domain.Illustration, error) {
	i, err := normalizeIllustration(i)
	if err != nil {
		return i, err
	}
	if err := uc.folderExists(ctx, i.FolderID); err != nil {
		return i, err
	}
	preview, err := uc.render(ctx, i.Name, i.Code)
	if err != nil {
		return i, err
	}
	i.Builtin, i.Exemplar, i.Status, i.Version = false, false, domain.IllustrationDraft, 1
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
	if existing.Builtin {
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
	if err := uc.folderExists(ctx, i.FolderID); err != nil {
		return i, err
	}
	i.ID, i.Builtin = id, false
	i.Status, i.Version, i.Warnings = existing.Status, existing.Version, existing.Warnings
	codeChanged := i.Code != existing.Code || i.Name != existing.Name
	var preview IllustrationPreview
	if codeChanged {
		if preview, err = uc.render(ctx, i.Name, i.Code); err != nil {
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
	if existing.Builtin {
		return existing, ErrIllustrationReadOnly
	}
	if status != domain.IllustrationDraft && status != domain.IllustrationApproved {
		return existing, fmt.Errorf("trạng thái %q không hợp lệ", status)
	}
	existing.Status = status
	return uc.repo.UpdateIllustration(ctx, existing)
}

func (uc *IllustrationsUseCase) Delete(ctx context.Context, id string) error {
	existing, err := uc.repo.GetIllustration(ctx, id)
	if err != nil {
		return err
	}
	if existing.Builtin {
		return ErrIllustrationReadOnly
	}
	return uc.repo.DeleteIllustration(ctx, id)
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
	out, err := uc.render(ctx, existing.Name, existing.Code)
	if err != nil {
		return nil, nil, err
	}
	if err := uc.repo.SaveIllustrationPreview(ctx, id, existing.Version, out.PNG, out.GIF); err != nil {
		return nil, nil, err
	}
	return out.PNG, out.GIF, nil
}
