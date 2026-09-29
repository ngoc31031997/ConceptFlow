package application

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"authoring/internal/domain"
)

var backupNow = time.Date(2026, 9, 27, 8, 30, 0, 0, time.UTC)

// libraryWithDrawings is a library holding a Creator folder and two drawings,
// one of them approved.
func libraryWithDrawings(t *testing.T) (*IllustrationsUseCase, *fakeIllustrationRepo) {
	t.Helper()
	repo := newFakeIllustrationRepo()
	uc := NewIllustrationsUseCase(repo, &fakeRenderer{})
	ctx := context.Background()
	if _, err := uc.CreateFolder(ctx, domain.IllustrationFolder{ID: "do-choi", Name: "Đồ chơi", Description: "bóng, búp bê"}); err != nil {
		t.Fatal(err)
	}
	ball, err := uc.Create(ctx, domain.Illustration{Name: "Ball", Title: "Quả bóng", FolderID: "do-choi", Code: "export function Ball() { return null; }", Tags: []string{"bóng"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := uc.SetStatus(ctx, ball.ID, domain.IllustrationApproved); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.Create(ctx, domain.Illustration{Name: "Bus", FolderID: "phuong-tien", Code: busCode}); err != nil {
		t.Fatal(err)
	}
	return uc, repo
}

func zipFiles(t *testing.T, file []byte) map[string][]byte {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(file), int64(len(file)))
	if err != nil {
		t.Fatal(err)
	}
	out := map[string][]byte{}
	for _, f := range zr.File {
		rc, _ := f.Open()
		out[f.Name], _ = io.ReadAll(rc)
		rc.Close()
	}
	return out
}

func makeZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, _ := zw.Create(name)
		w.Write([]byte(body))
	}
	zw.Close()
	return buf.Bytes()
}

func byName(repo *fakeIllustrationRepo, name string) (domain.Illustration, bool) {
	for _, r := range repo.rows {
		if r.Name == name {
			return r, true
		}
	}
	return domain.Illustration{}, false
}

func TestExportWritesCreatorDrawingsWithCodeAndImagesByFolder(t *testing.T) {
	uc, _ := libraryWithDrawings(t)
	out, err := uc.Export(context.Background(), backupNow)
	if err != nil {
		t.Fatal(err)
	}
	if out.Illustrations != 2 || out.Folders != len(domain.SystemIllustrationFolders())+1 {
		t.Fatalf("counts: %+v", out)
	}
	files := zipFiles(t, out.Zip)
	for _, want := range []string{"manifest.json", "hinh/do-choi/Ball.tsx", "hinh/do-choi/Ball.png", "hinh/do-choi/Ball.gif", "hinh/phuong-tien/Bus.tsx"} {
		if _, ok := files[want]; !ok {
			t.Fatalf("%s missing from the backup: %v", want, files)
		}
	}
	if string(files["hinh/phuong-tien/Bus.tsx"]) != busCode {
		t.Fatal("code not written verbatim")
	}
	for name := range files {
		if strings.Contains(name, "Tooth") {
			t.Fatal("built-ins ship with the service and must not be in the backup")
		}
	}
	var m backupManifest
	if err := json.Unmarshal(files["manifest.json"], &m); err != nil {
		t.Fatal(err)
	}
	if m.Format != LibraryBackupFormat || m.FormatVersion != 1 || m.ExportedAt != "2026-09-27T08:30:00Z" {
		t.Fatalf("header: %+v", m)
	}
}

func TestImportRestoresALostLibraryIntoTheSameFolders(t *testing.T) {
	src, _ := libraryWithDrawings(t)
	backup, err := src.Export(context.Background(), backupNow)
	if err != nil {
		t.Fatal(err)
	}

	// A fresh database: only the seeded folders and built-ins.
	repo, rend := newFakeIllustrationRepo(), &fakeRenderer{}
	uc := NewIllustrationsUseCase(repo, rend)
	report, err := uc.Import(context.Background(), backup.Zip, ImportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if report.Aborted != "" || len(report.Items) != 2 || len(report.FoldersCreated) != 1 || report.FoldersCreated[0] != "do-choi" {
		t.Fatalf("report: %+v", report)
	}
	ball, ok := byName(repo, "Ball")
	if !ok || ball.FolderID != "do-choi" || ball.Status != domain.IllustrationApproved || ball.Title != "Quả bóng" || ball.Tags[0] != "bóng" {
		t.Fatalf("Ball not restored as exported: %+v", ball)
	}
	bus, _ := byName(repo, "Bus")
	if bus.FolderID != "phuong-tien" || bus.Status != domain.IllustrationDraft || bus.Code != busCode {
		t.Fatalf("Bus not restored as exported: %+v", bus)
	}
	// Every drawing went through the renderer, and its fresh preview is stored.
	if len(rend.calls) != 2 || repo.pv[ball.ID] != ball.Version || len(repo.png[ball.ID]) == 0 {
		t.Fatalf("restored drawing not rendered and stored: calls=%v", rend.calls)
	}
	for _, f := range repo.folders {
		if f.ID == "do-choi" && (f.Name != "Đồ chơi" || f.Description != "bóng, búp bê") {
			t.Fatalf("folder not restored: %+v", f)
		}
	}
}

func TestImportSkipsExistingNamesUnlessAskedToReplace(t *testing.T) {
	uc, repo := libraryWithDrawings(t)
	ctx := context.Background()
	backup, _ := uc.Export(ctx, backupNow)
	before, _ := byName(repo, "Bus")
	if _, err := uc.Update(ctx, before.ID, domain.Illustration{Code: busCode + "\n// sửa sau khi sao lưu"}); err != nil {
		t.Fatal(err)
	}

	report, err := uc.Import(ctx, backup.Zip, ImportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range report.Items {
		if it.Result != ImportSkipped {
			t.Fatalf("existing drawing touched without replace: %+v", it)
		}
	}
	if len(report.FoldersCreated) != 0 {
		t.Fatal("existing folder created again")
	}

	report, err = uc.Import(ctx, backup.Zip, ImportOptions{Replace: true})
	if err != nil {
		t.Fatal(err)
	}
	bus, _ := byName(repo, "Bus")
	var busResult ImportResult
	for _, it := range report.Items {
		if it.Name == "Bus" {
			busResult = it.Result
		}
	}
	if busResult != ImportReplaced || bus.ID != before.ID || bus.Code != busCode || bus.Version != 3 {
		t.Fatalf("replace must restore the backup's code as a new version of the same row: %+v %+v", report.Items, bus)
	}
}

func TestImportNeverOverwritesABuiltinAndReportsBrokenDrawings(t *testing.T) {
	manifest := `{"format":"conceptflow-illustration-library","format_version":1,"folders":[],"illustrations":[
		{"name":"Tooth","folder_id":"co-the-suc-khoe","status":"approved","version":1,"code_file":"a.tsx"},
		{"name":"Broken","folder_id":"do-vat","status":"approved","version":1,"code_file":"b.tsx"},
		{"name":"Lost","folder_id":"khong-co","version":1,"code_file":"c.tsx"},
		{"name":"NoCode","folder_id":"do-vat","version":1,"code_file":"missing.tsx"},
		{"name":"Kite","folder_id":"do-vat","status":"weird","version":4,"code_file":"d.tsx"}]}`
	file := makeZip(t, map[string]string{"manifest.json": manifest,
		"a.tsx": "export function Tooth() {}", "b.tsx": "BROKEN", "c.tsx": "export function Lost() {}", "d.tsx": "export function Kite() {}"})
	repo := newFakeIllustrationRepo()
	uc := NewIllustrationsUseCase(repo, &fakeRenderer{})
	report, err := uc.Import(context.Background(), file, ImportOptions{Replace: true})
	if err != nil {
		t.Fatal(err)
	}
	want := []ImportResult{ImportSkipped, ImportFailed, ImportFailed, ImportFailed, ImportCreated}
	for n, it := range report.Items {
		if it.Result != want[n] {
			t.Fatalf("item %d %s: got %s (%s), want %s", n, it.Name, it.Result, it.Reason, want[n])
		}
	}
	if len(report.Items[1].Diagnostics) == 0 || !strings.Contains(report.Items[1].Reason, "dòng 3") {
		t.Fatalf("broken drawing must carry the renderer's diagnostics: %+v", report.Items[1])
	}
	if tooth := repo.rows["builtin-Tooth"]; tooth.Code != "" {
		t.Fatal("built-in overwritten")
	}
	kite, _ := byName(repo, "Kite")
	if kite.Status != domain.IllustrationDraft || kite.Version != 4 {
		t.Fatalf("unknown status must come back as a draft, version kept: %+v", kite)
	}
}

type unreachableRenderer struct{ calls int }

func (u *unreachableRenderer) PreviewIllustration(context.Context, string, string, map[string]any, bool) (IllustrationPreview, error) {
	u.calls++
	return IllustrationPreview{}, errors.New("rendering unreachable")
}

func TestImportStopsWhenTheRendererIsDownAndSaysWhatWasLeft(t *testing.T) {
	src, _ := libraryWithDrawings(t)
	backup, _ := src.Export(context.Background(), backupNow)
	repo, rend := newFakeIllustrationRepo(), &unreachableRenderer{}
	report, err := NewIllustrationsUseCase(repo, rend).Import(context.Background(), backup.Zip, ImportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(report.Aborted, "rendering unreachable") || report.NotProcessed != 2 || rend.calls != 1 {
		t.Fatalf("must stop at the first renderer failure: %+v calls=%d", report, rend.calls)
	}
	if _, ok := byName(repo, "Ball"); ok {
		t.Fatal("a drawing that was not rendered was saved")
	}
}

func TestImportRefusesFilesThatAreNotALibraryBackup(t *testing.T) {
	uc := NewIllustrationsUseCase(newFakeIllustrationRepo(), &fakeRenderer{})
	cases := map[string][]byte{
		"not a zip":       []byte("hello"),
		"no manifest":     makeZip(t, map[string]string{"x.txt": "x"}),
		"broken manifest": makeZip(t, map[string]string{"manifest.json": "{"}),
		"other format":    makeZip(t, map[string]string{"manifest.json": `{"format":"something","format_version":1}`}),
		"newer format":    makeZip(t, map[string]string{"manifest.json": `{"format":"conceptflow-illustration-library","format_version":9}`}),
	}
	for name, file := range cases {
		if _, err := uc.Import(context.Background(), file, ImportOptions{}); !errors.Is(err, ErrBackupInvalid) {
			t.Errorf("%s: want ErrBackupInvalid, got %v", name, err)
		}
	}
}

// CR-052: the Hình mẫu are Creator data now, so the backup keeps them.
func TestBackupKeepsTheExemplarsAndRestoresThemAsExemplars(t *testing.T) {
	src, repo := libraryWithDrawings(t)
	ctx := context.Background()
	repo.rows["cat"] = domain.Illustration{ID: "cat", Name: "Cat", Title: "Con mèo", FolderID: domain.ExemplarFolderID, Exemplar: true,
		HomeFolderID: "dong-vat", Code: "export function Cat() {}", Status: domain.IllustrationApproved, Version: 2}
	ball, _ := byName(repo, "Ball")
	cp, err := src.MakeExemplar(ctx, ball.ID)
	if err != nil {
		t.Fatal(err)
	}
	backup, err := src.Export(ctx, backupNow)
	if err != nil {
		t.Fatal(err)
	}
	var m backupManifest
	if err := json.Unmarshal(zipFiles(t, backup.Zip)["manifest.json"], &m); err != nil {
		t.Fatal(err)
	}
	marked := map[string]backupIllustration{}
	for _, i := range m.Illustrations {
		marked[i.Name] = i
	}
	if !marked["Cat"].Exemplar || marked["Cat"].HomeFolderID != "dong-vat" || !marked[cp.Name].Exemplar || marked[cp.Name].SourceName != "Ball" {
		t.Fatalf("manifest: %+v", m.Illustrations)
	}

	fresh := newFakeIllustrationRepo()
	uc := NewIllustrationsUseCase(fresh, &fakeRenderer{})
	report, err := uc.Import(ctx, backup.Zip, ImportOptions{})
	if err != nil || report.Aborted != "" {
		t.Fatalf("import: %+v %v", report, err)
	}
	cat, _ := byName(fresh, "Cat")
	restoredCopy, _ := byName(fresh, cp.Name)
	restoredBall, _ := byName(fresh, "Ball")
	if !cat.Exemplar || cat.HomeFolderID != "dong-vat" || cat.FolderID != domain.ExemplarFolderID {
		t.Fatalf("original exemplar: %+v", cat)
	}
	if !restoredCopy.Exemplar || restoredCopy.SourceID != restoredBall.ID {
		t.Fatalf("copy must find its source again: %+v (Ball %s)", restoredCopy, restoredBall.ID)
	}
}

func TestImportIntoAFullExemplarFolderKeepsOriginalsAsDrawingsAndSkipsCopies(t *testing.T) {
	src, repo := libraryWithDrawings(t)
	ctx := context.Background()
	repo.rows["cat"] = domain.Illustration{ID: "cat", Name: "Cat", FolderID: domain.ExemplarFolderID, Exemplar: true,
		HomeFolderID: "dong-vat", Code: "export function Cat() {}", Status: domain.IllustrationApproved}
	ball, _ := byName(repo, "Ball")
	cp, err := src.MakeExemplar(ctx, ball.ID)
	if err != nil {
		t.Fatal(err)
	}
	backup, err := src.Export(ctx, backupNow)
	if err != nil {
		t.Fatal(err)
	}

	full := newFakeIllustrationRepo()
	for n := 0; n < domain.MaxExemplars; n++ {
		id := string(rune('a' + n))
		full.rows[id] = domain.Illustration{ID: id, Name: "Full" + id, FolderID: domain.ExemplarFolderID, Exemplar: true}
	}
	report, err := NewIllustrationsUseCase(full, &fakeRenderer{}).Import(ctx, backup.Zip, ImportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	results := map[string]ImportItem{}
	for _, it := range report.Items {
		results[it.Name] = it
	}
	if r := results["Cat"]; r.Result != ImportCreated || r.FolderID != "dong-vat" || !strings.Contains(r.Reason, "hình thường") {
		t.Fatalf("original exemplar: %+v", r)
	}
	if cat, _ := byName(full, "Cat"); cat.Exemplar || cat.FolderID != "dong-vat" {
		t.Fatalf("Cat must come back as an ordinary drawing: %+v", cat)
	}
	if r := results[cp.Name]; r.Result != ImportSkipped || !strings.Contains(r.Reason, "Ball") {
		t.Fatalf("copy: %+v", r)
	}
}

func TestImportRefusesAnExemplarWhoseHomeIsTheExemplarFolder(t *testing.T) {
	manifest := `{"format":"conceptflow-illustration-library","format_version":1,"folders":[],"illustrations":[
		{"name":"Loop","title":"x","folder_id":"hinh-mau","status":"approved","version":1,"code_file":"hinh/hinh-mau/Loop.tsx",
		 "exemplar":true,"home_folder_id":"hinh-mau"}]}`
	file := makeZip(t, map[string]string{"manifest.json": manifest, "hinh/hinh-mau/Loop.tsx": "export function Loop() {}"})
	repo := newFakeIllustrationRepo()
	report, err := NewIllustrationsUseCase(repo, &fakeRenderer{}).Import(context.Background(), file, ImportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Items) != 1 || report.Items[0].Result != ImportFailed {
		t.Fatalf("report: %+v", report.Items)
	}
}

func TestImportRefusesAnOrdinaryDrawingFiledInTheExemplarFolder(t *testing.T) {
	manifest := `{"format":"conceptflow-illustration-library","format_version":1,"folders":[],"illustrations":[
		{"name":"Sneaky","title":"x","folder_id":"hinh-mau","status":"approved","version":1,"code_file":"hinh/hinh-mau/Sneaky.tsx"}]}`
	file := makeZip(t, map[string]string{"manifest.json": manifest, "hinh/hinh-mau/Sneaky.tsx": "export function Sneaky() {}"})
	repo := newFakeIllustrationRepo()
	report, err := NewIllustrationsUseCase(repo, &fakeRenderer{}).Import(context.Background(), file, ImportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Items) != 1 || report.Items[0].Result != ImportFailed {
		t.Fatalf("report: %+v", report.Items)
	}
	if _, ok := byName(repo, "Sneaky"); ok {
		t.Fatal("an ordinary drawing got into the Hình mẫu folder")
	}
}
