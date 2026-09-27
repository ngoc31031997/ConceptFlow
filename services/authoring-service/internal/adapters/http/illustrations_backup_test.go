package http

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"authoring/internal/application"
)

// fakeLibraryBackup answers only the backup calls; the other methods of the
// interface are never reached by these tests.
type fakeLibraryBackup struct {
	illustrationsUseCase
	got     []byte
	gotOpts application.ImportOptions
	err     error
}

func (f *fakeLibraryBackup) Export(context.Context, time.Time) (application.LibraryExport, error) {
	return application.LibraryExport{Zip: []byte("PK-zip"), Folders: 1, Illustrations: 1}, f.err
}

func (f *fakeLibraryBackup) Import(_ context.Context, file []byte, opts application.ImportOptions) (application.LibraryImportReport, error) {
	f.got, f.gotOpts = file, opts
	return application.LibraryImportReport{Items: []application.ImportItem{{Name: "Ball", Result: application.ImportCreated}}}, f.err
}

func TestExportIllustrationsDownloadsTheZip(t *testing.T) {
	h := NewRouter(nil, nil).WithIllustrations(&fakeLibraryBackup{}).Handler()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/admin/illustrations/export", nil))
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "application/zip" || rec.Body.String() != "PK-zip" {
		t.Fatalf("got %d %q %q", rec.Code, rec.Header().Get("Content-Type"), rec.Body.String())
	}
	if cd := rec.Header().Get("Content-Disposition"); !strings.Contains(cd, `filename="conceptflow-thu-vien-hinh-`) {
		t.Fatalf("no download name: %q", cd)
	}
}

func TestImportIllustrationsPassesTheBodyAndConflictMode(t *testing.T) {
	uc := &fakeLibraryBackup{}
	h := NewRouter(nil, nil).WithIllustrations(uc).Handler()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v1/admin/illustrations/import?on_conflict=replace", strings.NewReader("PK-zip")))
	body, _ := io.ReadAll(rec.Body)
	if rec.Code != http.StatusOK || string(uc.got) != "PK-zip" || !uc.gotOpts.Replace || !strings.Contains(string(body), `"result":"created"`) {
		t.Fatalf("got %d %s opts=%+v", rec.Code, body, uc.gotOpts)
	}

	for _, tc := range []struct {
		url, body string
		err       error
		want      int
	}{
		{"/v1/admin/illustrations/import?on_conflict=merge", "PK", nil, http.StatusBadRequest},
		{"/v1/admin/illustrations/import", "", nil, http.StatusBadRequest},
		{"/v1/admin/illustrations/import", "PK", fmt.Errorf("%w: thiếu manifest.json", application.ErrBackupInvalid), http.StatusBadRequest},
	} {
		uc := &fakeLibraryBackup{err: tc.err}
		rec := httptest.NewRecorder()
		NewRouter(nil, nil).WithIllustrations(uc).Handler().
			ServeHTTP(rec, httptest.NewRequest(http.MethodPost, tc.url, strings.NewReader(tc.body)))
		if rec.Code != tc.want {
			t.Errorf("%s %q: got %d, want %d", tc.url, tc.body, rec.Code, tc.want)
		}
		if tc.body == "" && uc.got != nil {
			t.Error("empty body reached the use case")
		}
	}
}
