package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"authoring/internal/application"
	"authoring/internal/domain"
)

type fakeVisualLanguages struct {
	err     error
	updated application.VisualLanguageInput
}

func (f *fakeVisualLanguages) List(context.Context) ([]domain.VisualLanguage, error) {
	return []domain.VisualLanguage{{ID: "phatsang", Name: "Khoa học phát sáng", KitKey: "phatsang", IsActive: true}}, f.err
}
func (f *fakeVisualLanguages) Get(_ context.Context, id string) (domain.VisualLanguage, error) {
	return domain.VisualLanguage{ID: id, Name: "x"}, f.err
}
func (f *fakeVisualLanguages) Activate(context.Context, string) error { return f.err }
func (f *fakeVisualLanguages) Copy(_ context.Context, id string) (domain.VisualLanguage, error) {
	return domain.VisualLanguage{ID: id + "-copy"}, f.err
}
func (f *fakeVisualLanguages) Update(_ context.Context, id string, in application.VisualLanguageInput) (domain.VisualLanguage, error) {
	f.updated = in
	return domain.VisualLanguage{ID: id, Name: in.Name, Version: 2}, f.err
}
func (f *fakeVisualLanguages) Delete(context.Context, string) error { return f.err }
func (f *fakeVisualLanguages) Pinned(_ context.Context, projectID string) (domain.PinnedLanguage, error) {
	return domain.PinnedLanguage{ID: "phatsang", Version: 1, KitKey: "phatsang"}, f.err
}
func (f *fakeVisualLanguages) Samples(context.Context, string) ([]domain.LanguageSample, error) {
	return []domain.LanguageSample{{KitKey: "phatsang", Scene: "fig", Frame: "wide", File: "a.png"}}, f.err
}
func (f *fakeVisualLanguages) RebuildSamples(context.Context, string) ([]domain.LanguageSample, error) {
	return nil, f.err
}

func callLanguages(t *testing.T, f *fakeVisualLanguages, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	router := NewRouter(nil, nil).WithVisualLanguages(f)
	rec := httptest.NewRecorder()
	router.Handler().ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)))
	return rec
}

func TestVisualLanguageRoutesServeTheStore(t *testing.T) {
	f := &fakeVisualLanguages{}
	for _, c := range []struct {
		method, path, body string
		want               int
	}{
		{"GET", "/v1/visual-languages", "", 200},
		{"GET", "/v1/visual-languages/phatsang", "", 200},
		{"GET", "/v1/visual-languages/phatsang/samples", "", 200},
		{"GET", "/v1/projects/p1/visual-language", "", 200},
		{"POST", "/v1/admin/visual-languages/phatsang/activate", "", 204},
		{"POST", "/v1/admin/visual-languages/phatsang/copy", "", 201},
		{"DELETE", "/v1/admin/visual-languages/mine", "", 204},
		{"POST", "/v1/admin/visual-languages/phatsang/samples", "", 200},
	} {
		if rec := callLanguages(t, f, c.method, c.path, c.body); rec.Code != c.want {
			t.Errorf("%s %s: %d %s, want %d", c.method, c.path, rec.Code, rec.Body.String(), c.want)
		}
	}
}

func TestUpdatingALanguagePassesItsRulesThrough(t *testing.T) {
	f := &fakeVisualLanguages{}
	body := `{"name":"Của tôi","note":"n","description":"d","rules":{"illustration_rules":"a","backdrop_rules":"b","scene_rules":"c"}}`
	rec := callLanguages(t, f, "PUT", "/v1/admin/visual-languages/mine", body)
	if rec.Code != 200 {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
	want := application.VisualLanguageInput{Name: "Của tôi", Note: "n", Description: "d",
		Rules: domain.VisualLanguageRules{Illustration: "a", Backdrop: "b", Scene: "c"}}
	if f.updated != want {
		t.Fatalf("update input %+v, want %+v", f.updated, want)
	}
	if rec := callLanguages(t, f, "PUT", "/v1/admin/visual-languages/mine", `{`); rec.Code != 400 {
		t.Fatalf("a broken body: %d", rec.Code)
	}
}

func TestVisualLanguageErrorsMapToTheRightStatus(t *testing.T) {
	for _, c := range []struct {
		err  error
		want int
	}{
		{domain.ErrVisualLanguageNotFound, http.StatusNotFound},
		{domain.ErrProjectNotPinned, http.StatusNotFound},
		{domain.ErrVisualLanguageReadOnly, http.StatusForbidden},
		{domain.ErrVisualLanguageActive, http.StatusConflict},
		{domain.ErrVisualLanguageInUse, http.StatusConflict},
		{domain.ErrVisualLanguageNameTaken, http.StatusConflict},
		{domain.ErrInvalidVisualLanguage, http.StatusBadRequest},
		{domain.ErrUnknownKitKey, http.StatusBadRequest},
	} {
		rec := callLanguages(t, &fakeVisualLanguages{err: c.err}, "DELETE", "/v1/admin/visual-languages/x", "")
		if rec.Code != c.want {
			t.Errorf("%v -> %d, want %d", c.err, rec.Code, c.want)
		}
	}
}

func TestTheProjectsPinnedLanguageIsServedWithItsKit(t *testing.T) {
	rec := callLanguages(t, &fakeVisualLanguages{}, "GET", "/v1/projects/p1/visual-language", "")
	var out domain.PinnedLanguage
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil || out.KitKey != "phatsang" || out.Version != 1 {
		t.Fatalf("%s (%v)", rec.Body.String(), err)
	}
}

func TestVisualLanguageRoutesAreOffUntilWired(t *testing.T) {
	rec := httptest.NewRecorder()
	NewRouter(nil, nil).Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/v1/visual-languages", nil))
	if rec.Code != 404 {
		t.Fatalf("%d", rec.Code)
	}
}
