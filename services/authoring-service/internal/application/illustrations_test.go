package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"authoring/internal/domain"
)

type fakeIllustrationRepo struct {
	mu      sync.Mutex // the illustrations step draws several at once
	folders []domain.IllustrationFolder
	rows    map[string]domain.Illustration
	png     map[string][]byte
	pv      map[string]int
	next    int
	// The projects using a drawing, by drawing id, as the database
	// would find them (drawing list link or code naming it); linked are the
	// ones that link it in their drawing list.
	users  map[string][]IllustrationUser
	linked map[string][]string
}

func newFakeIllustrationRepo() *fakeIllustrationRepo {
	r := &fakeIllustrationRepo{folders: domain.SystemIllustrationFolders(), rows: map[string]domain.Illustration{},
		png: map[string][]byte{}, pv: map[string]int{}, users: map[string][]IllustrationUser{}, linked: map[string][]string{}}
	for _, b := range domain.BuiltinIllustrations() {
		r.rows[b.ID] = b
	}
	return r
}

func (r *fakeIllustrationRepo) ListIllustrationFolders(context.Context) ([]domain.IllustrationFolder, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.folders, nil
}
func (r *fakeIllustrationRepo) CreateIllustrationFolder(_ context.Context, f domain.IllustrationFolder) (domain.IllustrationFolder, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.folders = append(r.folders, f)
	return f, nil
}
func (r *fakeIllustrationRepo) DeleteIllustrationFolder(context.Context, string) error { return nil }
func (r *fakeIllustrationRepo) ListIllustrations(_ context.Context, f IllustrationFilter) ([]domain.Illustration, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []domain.Illustration
	for _, i := range r.rows {
		if f.FolderID == "" || i.FolderID == f.FolderID {
			out = append(out, i)
		}
	}
	return out, nil
}
func (r *fakeIllustrationRepo) GetIllustration(_ context.Context, id string) (domain.Illustration, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	i, ok := r.rows[id]
	if !ok {
		return i, ErrIllustrationNotFound
	}
	return i, nil
}
func (r *fakeIllustrationRepo) CreateIllustration(_ context.Context, i domain.Illustration) (domain.Illustration, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, e := range r.rows {
		if e.Name == i.Name {
			return i, ErrIllustrationNameTaken
		}
	}
	r.next++
	i.ID = fmt.Sprintf("i%d", r.next)
	r.rows[i.ID] = i
	return i, nil
}
func (r *fakeIllustrationRepo) UpdateIllustration(_ context.Context, i domain.Illustration) (domain.Illustration, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows[i.ID] = i
	return i, nil
}
func (r *fakeIllustrationRepo) DeleteIllustration(_ context.Context, id string, checked []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	ok := map[string]bool{}
	for _, p := range checked {
		ok[p] = true
	}
	for _, p := range r.linked[id] {
		if !ok[p] {
			return ErrIllustrationLinked
		}
	}
	delete(r.rows, id)
	return nil
}
func (r *fakeIllustrationRepo) FindIllustrationUsers(_ context.Context, id, _ string) ([]IllustrationUser, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.users[id], nil
}
func (r *fakeIllustrationRepo) CreateExemplar(_ context.Context, i domain.Illustration, limit int) (domain.Illustration, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	count := 0
	for _, e := range r.rows {
		if e.Exemplar {
			count++
			if i.SourceID != "" && e.SourceID == i.SourceID {
				return i, ErrAlreadyExemplar
			}
		}
		if e.Name == i.Name {
			return i, ErrIllustrationNameTaken
		}
	}
	if count >= limit {
		return i, ErrExemplarLimit
	}
	r.next++
	i.ID, i.FolderID, i.Exemplar = fmt.Sprintf("i%d", r.next), domain.ExemplarFolderID, true
	i.CreatedAt = fmt.Sprintf("2026-09-29T00:00:%02dZ", r.next)
	r.rows[i.ID] = i
	return i, nil
}
func (r *fakeIllustrationRepo) ReleaseExemplar(_ context.Context, id, folderID string) (domain.Illustration, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	i, ok := r.rows[id]
	if !ok || !i.Exemplar {
		return i, ErrIllustrationNotFound
	}
	i.Exemplar, i.HomeFolderID, i.FolderID = false, "", folderID
	r.rows[id] = i
	return i, nil
}
func (r *fakeIllustrationRepo) SaveIllustrationPreview(_ context.Context, id string, v int, png, _ []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.png[id], r.pv[id] = png, v
	return nil
}
func (r *fakeIllustrationRepo) GetIllustrationPreview(_ context.Context, id string) ([]byte, []byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.pv[id] != r.rows[id].Version {
		return nil, nil, nil
	}
	return r.png[id], []byte("GIF"), nil
}

type fakeRenderer struct {
	mu    sync.Mutex
	calls []string
	fail  bool
}

func (f *fakeRenderer) PreviewIllustration(_ context.Context, name, code string, _ map[string]any, _ bool) (IllustrationPreview, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, name+"|"+code)
	if f.fail || strings.Contains(code, "BROKEN") {
		line := 3
		return IllustrationPreview{OK: false, Diagnostics: []PreviewDiagnostic{{Message: "TS2304: Cannot find name", Line: &line}}}, nil
	}
	return IllustrationPreview{OK: true, PNG: []byte("PNG:" + name + code), GIF: []byte("GIF")}, nil
}

const busCode = "export function Bus() { return null; }"

func TestCreateRendersFirstAndStoresADraftWithItsPreview(t *testing.T) {
	repo, rend := newFakeIllustrationRepo(), &fakeRenderer{}
	uc := NewIllustrationsUseCase(repo, rend)
	out, err := uc.Create(context.Background(), domain.Illustration{Name: "Bus", FolderID: "phuong-tien", Code: busCode, Tags: []string{" Xe ", "xe"}})
	if err != nil {
		t.Fatal(err)
	}
	if out.Status != domain.IllustrationDraft || out.Version != 1 || !out.HasPreview || out.Title != "Bus" {
		t.Fatalf("unexpected row %+v", out)
	}
	if len(out.Tags) != 1 || out.Tags[0] != "xe" {
		t.Fatalf("tags not normalised: %v", out.Tags)
	}
	if string(repo.png[out.ID]) != "PNG:Bus"+busCode {
		t.Fatal("preview not stored")
	}
}

func TestCreateRefusesCodeTheRendererRejectsWithLineNumbers(t *testing.T) {
	repo := newFakeIllustrationRepo()
	uc := NewIllustrationsUseCase(repo, &fakeRenderer{})
	_, err := uc.Create(context.Background(), domain.Illustration{Name: "Bus", FolderID: "phuong-tien", Code: "BROKEN"})
	var invalid *InvalidIllustrationError
	if !errors.As(err, &invalid) || *invalid.Diagnostics[0].Line != 3 || !strings.Contains(err.Error(), "dòng 3") {
		t.Fatalf("want diagnostics with line, got %v", err)
	}
	if len(repo.rows) != len(domain.BuiltinIllustrations()) {
		t.Fatal("a drawing that does not render was saved")
	}
	if _, err := uc.Create(context.Background(), domain.Illustration{Name: "Bus", FolderID: "khong-co", Code: busCode}); !errors.Is(err, ErrFolderNotFound) {
		t.Fatalf("unknown folder accepted: %v", err)
	}
	if _, err := uc.Create(context.Background(), domain.Illustration{Name: "school_bus", FolderID: "phuong-tien", Code: busCode}); err == nil {
		t.Fatal("non-PascalCase name accepted")
	}
}

func TestEditingCodeSendsAnApprovedDrawingBackToDraftWithANewVersion(t *testing.T) {
	repo, rend := newFakeIllustrationRepo(), &fakeRenderer{}
	uc := NewIllustrationsUseCase(repo, rend)
	ctx := context.Background()
	made, _ := uc.Create(ctx, domain.Illustration{Name: "Bus", FolderID: "phuong-tien", Code: busCode})
	if _, err := uc.SetStatus(ctx, made.ID, domain.IllustrationApproved); err != nil {
		t.Fatal(err)
	}
	// Moving it to another folder keeps it approved and renders nothing.
	calls := len(rend.calls)
	moved, _ := uc.Update(ctx, made.ID, domain.Illustration{FolderID: "do-vat"})
	if moved.Status != domain.IllustrationApproved || moved.Version != 1 || len(rend.calls) != calls {
		t.Fatalf("metadata edit changed review state: %+v", moved)
	}
	edited, err := uc.Update(ctx, made.ID, domain.Illustration{Code: busCode + "\n// bánh to hơn"})
	if err != nil {
		t.Fatal(err)
	}
	if edited.Status != domain.IllustrationDraft || edited.Version != 2 || repo.pv[made.ID] != 2 {
		t.Fatalf("code edit must go back to draft as v2: %+v", edited)
	}
}

func TestBuiltinsAreReadOnlyAndRenderedLazilyWithoutCode(t *testing.T) {
	repo, rend := newFakeIllustrationRepo(), &fakeRenderer{}
	uc := NewIllustrationsUseCase(repo, rend)
	ctx := context.Background()
	if _, err := uc.Update(ctx, "builtin-Tooth", domain.Illustration{Code: busCode}); !errors.Is(err, ErrIllustrationReadOnly) {
		t.Fatalf("built-in edited: %v", err)
	}
	if err := uc.Delete(ctx, "builtin-Tooth"); !errors.Is(err, ErrIllustrationReadOnly) {
		t.Fatalf("built-in deleted: %v", err)
	}
	png, _, err := uc.Preview(ctx, "builtin-Tooth")
	if err != nil || string(png) != "PNG:Tooth" || rend.calls[0] != "Tooth|" {
		t.Fatalf("built-in preview must render the kit component by name: %q %v %v", png, rend.calls, err)
	}
	uc.Preview(ctx, "builtin-Tooth") // now cached
	if len(rend.calls) != 1 {
		t.Fatal("stored preview rendered again")
	}
}

func TestSystemFoldersCannotBeDeletedAndFullFoldersStay(t *testing.T) {
	repo := newFakeIllustrationRepo()
	uc := NewIllustrationsUseCase(repo, &fakeRenderer{})
	ctx := context.Background()
	if err := uc.DeleteFolder(ctx, "con-nguoi"); !errors.Is(err, ErrFolderReadOnly) {
		t.Fatalf("system folder: %v", err)
	}
	if _, err := uc.CreateFolder(ctx, domain.IllustrationFolder{ID: "đồ chơi", Name: "Đồ chơi"}); err == nil {
		t.Fatal("non-slug folder id accepted")
	}
	uc.CreateFolder(ctx, domain.IllustrationFolder{ID: "do-choi", Name: "Đồ chơi"})
	uc.Create(ctx, domain.Illustration{Name: "Ball", FolderID: "do-choi", Code: busCode})
	if err := uc.DeleteFolder(ctx, "do-choi"); !errors.Is(err, ErrFolderNotEmpty) {
		t.Fatalf("non-empty folder: %v", err)
	}
}
