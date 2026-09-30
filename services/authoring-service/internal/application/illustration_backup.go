package application

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"time"

	"authoring/internal/domain"
)

// Back up the illustration library to one ZIP file and restore it,
// so the Creator's drawings survive a lost database.
//
// The ZIP holds manifest.json (every folder, and every Creator drawing with its
// folder, tags, status and version) and, per drawing, its TSX and its stored
// PNG still and GIF under hinh/<folder>/<Name>.*, so the backup can also be
// browsed by hand. The kit is left out: it ships with the service and is seeded
// again on every start. The Hình mẫu are kept, marked as such.
//
// Restoring files every drawing into the folder it was exported from,
// creating Creator folders that are missing, and keeps its review status. Like
// Create, every drawing goes through the renderer first — the library never
// holds a drawing that cannot render with today's kit — and the stored preview
// is the one rendered now, not the one in the file.

const (
	LibraryBackupFormat        = "conceptflow-illustration-library"
	LibraryBackupFormatVersion = 1
	backupManifestPath         = "manifest.json"
	maxBackupManifestBytes     = 8 << 20
	maxBackupCodeBytes         = 1 << 20
)

// ErrBackupInvalid is a file that is not a library backup this service can read.
var ErrBackupInvalid = errors.New("illustration library backup is not valid")

type backupManifest struct {
	Format        string               `json:"format"`
	FormatVersion int                  `json:"format_version"`
	ExportedAt    string               `json:"exported_at"`
	Folders       []backupFolder       `json:"folders"`
	Illustrations []backupIllustration `json:"illustrations"`
}

type backupFolder struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Position    int    `json:"position"`
	IsSystem    bool   `json:"is_system"`
}

type backupIllustration struct {
	Name        string   `json:"name"`
	Title       string   `json:"title"`
	FolderID    string   `json:"folder_id"`
	Tags        []string `json:"tags"`
	Description string   `json:"description"`
	Usage       string   `json:"usage"`
	Status      string   `json:"status"`
	Version     int      `json:"version"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
	// A Hình mẫu, the name of the drawing it copies (if any), and the
	// folder an original exemplar goes back to.
	Exemplar     bool   `json:"exemplar,omitempty"`
	SourceName   string `json:"source_name,omitempty"`
	HomeFolderID string `json:"home_folder_id,omitempty"`
	// Paths inside the ZIP. The images are absent when no preview was stored.
	CodeFile string `json:"code_file"`
	PNGFile  string `json:"png_file,omitempty"`
	GIFFile  string `json:"gif_file,omitempty"`
}

// LibraryExport is a finished backup file.
type LibraryExport struct {
	Zip           []byte
	Folders       int
	Illustrations int
}

// Export writes every folder and every Creator drawing, with its stored
// preview images, to a ZIP. It reads the database only; nothing is rendered.
func (uc *IllustrationsUseCase) Export(ctx context.Context, now time.Time) (LibraryExport, error) {
	folders, err := uc.repo.ListIllustrationFolders(ctx)
	if err != nil {
		return LibraryExport{}, err
	}
	rows, err := uc.repo.ListIllustrations(ctx, IllustrationFilter{})
	if err != nil {
		return LibraryExport{}, err
	}
	manifest := backupManifest{
		Format: LibraryBackupFormat, FormatVersion: LibraryBackupFormatVersion,
		ExportedAt: now.UTC().Format(time.RFC3339),
		Folders:    make([]backupFolder, 0, len(folders)), Illustrations: []backupIllustration{},
	}
	for _, f := range folders {
		manifest.Folders = append(manifest.Folders, backupFolder{
			ID: f.ID, Name: f.Name, Description: f.Description, Position: f.Position, IsSystem: f.IsSystem,
		})
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	put := func(path string, body []byte, method uint16) error {
		w, err := zw.CreateHeader(&zip.FileHeader{Name: path, Method: method, Modified: now})
		if err != nil {
			return err
		}
		_, err = w.Write(body)
		return err
	}
	nameOf := map[string]string{}
	for _, i := range rows {
		nameOf[i.ID] = i.Name
	}
	for _, i := range rows {
		if i.Builtin {
			continue
		}
		// Folder ids are slugs and names PascalCase, so the path is safe as is.
		base := "hinh/" + i.FolderID + "/" + i.Name
		entry := backupIllustration{
			Name: i.Name, Title: i.Title, FolderID: i.FolderID, Tags: i.Tags, Description: i.Description,
			Usage: i.Usage, Status: string(i.Status), Version: i.Version, CreatedAt: i.CreatedAt, UpdatedAt: i.UpdatedAt,
			CodeFile: base + ".tsx",
			Exemplar: i.Exemplar, SourceName: nameOf[i.SourceID], HomeFolderID: i.HomeFolderID,
		}
		if err := put(entry.CodeFile, []byte(i.Code), zip.Deflate); err != nil {
			return LibraryExport{}, err
		}
		png, gif, err := uc.repo.GetIllustrationPreview(ctx, i.ID)
		if err != nil {
			return LibraryExport{}, err
		}
		// PNG and GIF are compressed already; storing them saves the CPU.
		if len(png) > 0 {
			entry.PNGFile = base + ".png"
			if err := put(entry.PNGFile, png, zip.Store); err != nil {
				return LibraryExport{}, err
			}
		}
		if len(gif) > 0 {
			entry.GIFFile = base + ".gif"
			if err := put(entry.GIFFile, gif, zip.Store); err != nil {
				return LibraryExport{}, err
			}
		}
		manifest.Illustrations = append(manifest.Illustrations, entry)
	}
	body, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return LibraryExport{}, err
	}
	if err := put(backupManifestPath, body, zip.Deflate); err != nil {
		return LibraryExport{}, err
	}
	if err := zw.Close(); err != nil {
		return LibraryExport{}, err
	}
	return LibraryExport{Zip: buf.Bytes(), Folders: len(manifest.Folders), Illustrations: len(manifest.Illustrations)}, nil
}

// ImportResult is what happened to one drawing of a backup.
type ImportResult string

const (
	ImportCreated  ImportResult = "created"  // added to the library
	ImportReplaced ImportResult = "replaced" // overwrote the Creator drawing of the same name
	ImportSkipped  ImportResult = "skipped"  // left alone, see Reason
	ImportFailed   ImportResult = "failed"   // not imported, see Reason and Diagnostics
)

// ImportItem reports one drawing of the backup.
type ImportItem struct {
	Name        string              `json:"name"`
	FolderID    string              `json:"folder_id"`
	Result      ImportResult        `json:"result"`
	Reason      string              `json:"reason,omitempty"`
	Diagnostics []PreviewDiagnostic `json:"diagnostics,omitempty"`
}

// LibraryImportReport is the outcome of one restore. Aborted is set when the
// renderer or the database failed (not a drawing): the drawings from that
// point on were not tried (NotProcessed), and importing the same file again
// resumes, since the ones already in the library are skipped.
type LibraryImportReport struct {
	ExportedAt     string       `json:"exported_at"`
	FoldersCreated []string     `json:"folders_created"`
	FolderErrors   []string     `json:"folder_errors"`
	Items          []ImportItem `json:"items"`
	Aborted        string       `json:"aborted,omitempty"`
	NotProcessed   int          `json:"not_processed"`
}

// ImportOptions: Replace overwrites a Creator drawing that has the same name;
// otherwise it is kept and the backup's copy skipped.
type ImportOptions struct {
	Replace bool
}

func invalidBackup(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrBackupInvalid, fmt.Sprintf(format, args...))
}

func readZipEntry(f *zip.File, limit int64) ([]byte, error) {
	if f.UncompressedSize64 > uint64(limit) {
		return nil, invalidBackup("%s lớn quá (%d byte)", f.Name, f.UncompressedSize64)
	}
	rc, err := f.Open()
	if err != nil {
		return nil, invalidBackup("không mở được %s: %v", f.Name, err)
	}
	defer rc.Close()
	body, err := io.ReadAll(io.LimitReader(rc, limit+1))
	if err != nil {
		return nil, invalidBackup("không đọc được %s: %v", f.Name, err)
	}
	if int64(len(body)) > limit {
		return nil, invalidBackup("%s lớn quá", f.Name)
	}
	return body, nil
}

// Import restores a backup made by Export.
func (uc *IllustrationsUseCase) Import(ctx context.Context, file []byte, opts ImportOptions) (LibraryImportReport, error) {
	zr, err := zip.NewReader(bytes.NewReader(file), int64(len(file)))
	if err != nil {
		return LibraryImportReport{}, invalidBackup("không phải file ZIP (%v)", err)
	}
	entries := map[string]*zip.File{}
	for _, f := range zr.File {
		entries[f.Name] = f
	}
	mf, ok := entries[backupManifestPath]
	if !ok {
		return LibraryImportReport{}, invalidBackup("thiếu %s — đây không phải file sao lưu thư viện hình", backupManifestPath)
	}
	raw, err := readZipEntry(mf, maxBackupManifestBytes)
	if err != nil {
		return LibraryImportReport{}, err
	}
	var manifest backupManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return LibraryImportReport{}, invalidBackup("%s hỏng: %v", backupManifestPath, err)
	}
	if manifest.Format != LibraryBackupFormat {
		return LibraryImportReport{}, invalidBackup("định dạng %q không phải sao lưu thư viện hình", manifest.Format)
	}
	if manifest.FormatVersion < 1 || manifest.FormatVersion > LibraryBackupFormatVersion {
		return LibraryImportReport{}, invalidBackup("phiên bản định dạng %d chưa được hỗ trợ (bản này đọc tới %d)",
			manifest.FormatVersion, LibraryBackupFormatVersion)
	}

	report := LibraryImportReport{ExportedAt: manifest.ExportedAt, FoldersCreated: []string{}, FolderErrors: []string{}, Items: []ImportItem{}}
	if err := uc.importFolders(ctx, manifest.Folders, &report); err != nil {
		return report, err
	}

	rows, err := uc.repo.ListIllustrations(ctx, IllustrationFilter{})
	if err != nil {
		return report, err
	}
	byName := map[string]domain.Illustration{}
	for _, r := range rows {
		byName[r.Name] = r
	}
	seen := map[string]bool{}
	// Hình mẫu last, so a copy finds the drawing it was made from.
	sort.SliceStable(manifest.Illustrations, func(a, b int) bool {
		return !manifest.Illustrations[a].Exemplar && manifest.Illustrations[b].Exemplar
	})
	for n, b := range manifest.Illustrations {
		item, abort := uc.importIllustration(ctx, b, entries, byName, seen, opts)
		if abort != nil {
			report.Aborted = abort.Error()
			report.NotProcessed = len(manifest.Illustrations) - n
			return report, nil
		}
		report.Items = append(report.Items, item)
	}
	return report, nil
}

// importFolders creates, in their exported order, the folders the library
// lacks. A folder that exists keeps its current name. A folder that cannot be
// created is reported; its drawings then fail with "folder not found".
func (uc *IllustrationsUseCase) importFolders(ctx context.Context, folders []backupFolder, report *LibraryImportReport) error {
	existing, err := uc.repo.ListIllustrationFolders(ctx)
	if err != nil {
		return err
	}
	have := map[string]bool{}
	for _, f := range existing {
		have[f.ID] = true
	}
	ordered := append([]backupFolder(nil), folders...)
	sort.SliceStable(ordered, func(a, b int) bool { return ordered[a].Position < ordered[b].Position })
	for _, f := range ordered {
		if have[f.ID] {
			continue
		}
		made, err := uc.CreateFolder(ctx, domain.IllustrationFolder{ID: f.ID, Name: f.Name, Description: f.Description})
		if err != nil {
			report.FolderErrors = append(report.FolderErrors, fmt.Sprintf("%s: %v", f.ID, err))
			continue
		}
		have[made.ID] = true
		report.FoldersCreated = append(report.FoldersCreated, made.ID)
	}
	return nil
}

// importIllustration restores one drawing. The returned error is not about
// this drawing but about the renderer or the database: the import stops there.
func (uc *IllustrationsUseCase) importIllustration(
	ctx context.Context, b backupIllustration, entries map[string]*zip.File,
	byName map[string]domain.Illustration, seen map[string]bool, opts ImportOptions,
) (ImportItem, error) {
	item := ImportItem{Name: b.Name, FolderID: b.FolderID}
	fail := func(reason string) (ImportItem, error) {
		item.Result, item.Reason = ImportFailed, reason
		return item, nil
	}
	if seen[b.Name] {
		return fail("tên này xuất hiện hai lần trong file sao lưu")
	}
	seen[b.Name] = true

	codeFile, ok := entries[b.CodeFile]
	if b.CodeFile == "" || !ok {
		return fail(fmt.Sprintf("thiếu file code %q trong file sao lưu", b.CodeFile))
	}
	code, err := readZipEntry(codeFile, maxBackupCodeBytes)
	if err != nil {
		return fail(err.Error())
	}
	ill, err := normalizeIllustration(domain.Illustration{
		Name: b.Name, Title: b.Title, FolderID: b.FolderID, Tags: b.Tags,
		Description: b.Description, Usage: b.Usage, Code: string(code),
	})
	if err != nil {
		return fail(err.Error())
	}
	status := domain.IllustrationStatus(b.Status)
	if status != domain.IllustrationApproved {
		status = domain.IllustrationDraft
	}

	existing, taken := byName[ill.Name]
	switch {
	case taken && existing.Builtin:
		item.Result, item.Reason = ImportSkipped, "trùng tên một hình có sẵn của hệ thống"
		return item, nil
	case taken && existing.Exemplar:
		item.Result, item.Reason = ImportSkipped, "trùng tên một Hình mẫu (chỉ đọc)"
		return item, nil
	case taken && b.Exemplar:
		// A Hình mẫu from the file never overwrites one of the Creator's drawings.
		item.Result, item.Reason = ImportSkipped, "đã có hình cùng tên trong thư viện"
		return item, nil
	case taken && !opts.Replace:
		item.Result, item.Reason = ImportSkipped, "đã có hình cùng tên trong thư viện"
		return item, nil
	}
	if !b.Exemplar && ill.FolderID == domain.ExemplarFolderID {
		return fail(ErrExemplarFolder.Error())
	}
	if b.Exemplar && b.HomeFolderID == domain.ExemplarFolderID {
		return fail("thư mục cũ của Hình mẫu không thể là chính thư mục Hình mẫu")
	}
	if b.Exemplar && b.HomeFolderID != "" {
		if err := uc.folderExists(ctx, b.HomeFolderID); err != nil {
			return fail(fmt.Sprintf("thư mục cũ %q của Hình mẫu không có trong thư viện", b.HomeFolderID))
		}
	}
	if err := uc.folderExists(ctx, ill.FolderID); err != nil {
		if errors.Is(err, ErrFolderNotFound) {
			return fail(fmt.Sprintf("thư mục %q không có trong thư viện", ill.FolderID))
		}
		return item, err
	}

	preview, err := uc.render(ctx, ill.Name, ill.Code)
	var invalid *InvalidIllustrationError
	if errors.As(err, &invalid) {
		item.Diagnostics = invalid.Diagnostics
		return fail(invalid.Error())
	}
	if err != nil {
		return item, fmt.Errorf("dừng ở hình %s: %w", ill.Name, err)
	}
	ill.Builtin, ill.Exemplar, ill.Status, ill.Warnings = false, false, status, preview.Warnings
	if b.Exemplar {
		return uc.importExemplar(ctx, ill, b, byName, preview, item)
	}

	var saved domain.Illustration
	if taken {
		// A new version, so a browser never shows the old preview from its cache.
		ill.ID, ill.Version = existing.ID, existing.Version+1
		saved, err = uc.repo.UpdateIllustration(ctx, ill)
		item.Result = ImportReplaced
	} else {
		ill.Version = max(b.Version, 1)
		saved, err = uc.repo.CreateIllustration(ctx, ill)
		item.Result = ImportCreated
	}
	if errors.Is(err, ErrIllustrationNameTaken) {
		return fail("đã có hình cùng tên trong thư viện")
	}
	if err != nil {
		return item, err
	}
	if err := uc.repo.SaveIllustrationPreview(ctx, saved.ID, saved.Version, preview.PNG, preview.GIF); err != nil {
		return item, err
	}
	byName[saved.Name] = saved
	return item, nil
}

// importExemplar restores a Hình mẫu while the folder has room. When it is
// full, an original exemplar comes back as an ordinary drawing in its folder;
// a copy is skipped, since its source is a drawing of the library already.
func (uc *IllustrationsUseCase) importExemplar(
	ctx context.Context, ill domain.Illustration, b backupIllustration,
	byName map[string]domain.Illustration, preview IllustrationPreview, item ImportItem,
) (ImportItem, error) {
	ex := ill
	ex.FolderID, ex.Exemplar, ex.Status, ex.Version = domain.ExemplarFolderID, true, domain.IllustrationApproved, max(b.Version, 1)
	ex.HomeFolderID = b.HomeFolderID
	if src, ok := byName[b.SourceName]; ok && b.SourceName != "" && !src.ReadOnly() {
		ex.SourceID = src.ID
	}
	saved, err := uc.repo.CreateExemplar(ctx, ex, domain.MaxExemplars)
	item.Result = ImportCreated
	switch {
	case errors.Is(err, ErrExemplarLimit) && ex.HomeFolderID != "":
		plain := ill
		plain.FolderID, plain.Status, plain.Version = ex.HomeFolderID, domain.IllustrationApproved, ex.Version
		saved, err = uc.repo.CreateIllustration(ctx, plain)
		item.FolderID = plain.FolderID
		item.Reason = fmt.Sprintf("thư mục Hình mẫu đã đủ %d hình — nhập thành hình thường", domain.MaxExemplars)
	case errors.Is(err, ErrExemplarLimit):
		item.Result = ImportSkipped
		item.Reason = fmt.Sprintf("thư mục Hình mẫu đã đủ %d hình; đây là bản sao của %s nên không nhập", domain.MaxExemplars, b.SourceName)
		return item, nil
	case errors.Is(err, ErrAlreadyExemplar):
		item.Result, item.Reason = ImportSkipped, "hình gốc "+b.SourceName+" đã có bản Hình mẫu"
		return item, nil
	}
	if errors.Is(err, ErrIllustrationNameTaken) {
		item.Result, item.Reason = ImportFailed, "đã có hình cùng tên trong thư viện"
		return item, nil
	}
	if err != nil {
		return item, err
	}
	if err := uc.repo.SaveIllustrationPreview(ctx, saved.ID, saved.Version, preview.PNG, preview.GIF); err != nil {
		return item, err
	}
	byName[saved.Name] = saved
	return item, nil
}
