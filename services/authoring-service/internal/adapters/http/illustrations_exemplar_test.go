package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"authoring/internal/application"
	"authoring/internal/domain"
)

// Delete refusals, the Hình mẫu routes and the style endpoint.
type fakeExemplars struct {
	illustrationsUseCase
	deleteErr error
	styleErr  error
	released  *domain.Illustration
}

func (f *fakeExemplars) Delete(context.Context, string) error { return f.deleteErr }
func (f *fakeExemplars) Exemplars(context.Context) ([]domain.Illustration, error) {
	return []domain.Illustration{{ID: "exemplar-Cat"}, {ID: "i7"}}, nil
}
func (f *fakeExemplars) MakeExemplar(_ context.Context, id string) (domain.Illustration, error) {
	if id == "full" {
		return domain.Illustration{}, application.ErrExemplarLimit
	}
	return domain.Illustration{ID: "i9", Name: "BusMau", Exemplar: true, SourceID: id}, nil
}
func (f *fakeExemplars) UnmakeExemplar(context.Context, string) (*domain.Illustration, error) {
	return f.released, nil
}
func (f *fakeExemplars) StyleRules(context.Context) (string, error) {
	if f.styleErr != nil {
		return "", f.styleErr
	}
	return "- [S1] LUẬT TỪ DB: dòng đang bật", nil
}

func serve(uc illustrationsUseCase, method, url string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	NewRouter(nil, nil).WithIllustrations(uc).Handler().ServeHTTP(rec, httptest.NewRequest(method, url, nil))
	return rec
}

func TestDeleteIllustrationAnswersWhyItWasRefused(t *testing.T) {
	inUse := &application.IllustrationInUseError{Projects: []application.IllustrationUser{{ProjectID: "p1", Topic: "Sâu răng"}}}
	for _, tc := range []struct {
		err  error
		want int
	}{
		{nil, http.StatusNoContent},
		{inUse, http.StatusConflict},
		{fmt.Errorf("%w (dial tcp orchestrator:8000: refused)", application.ErrIllustrationUsageUnknown), http.StatusServiceUnavailable},
		{application.ErrIllustrationReadOnly, http.StatusForbidden},
	} {
		rec := serve(&fakeExemplars{deleteErr: tc.err}, http.MethodDelete, "/v1/admin/illustrations/bus")
		if rec.Code != tc.want {
			t.Errorf("%v: got %d, want %d", tc.err, rec.Code, tc.want)
		}
	}
	down := serve(&fakeExemplars{deleteErr: fmt.Errorf("%w (dial tcp orchestrator:8000: refused)", application.ErrIllustrationUsageUnknown)},
		http.MethodDelete, "/v1/admin/illustrations/bus")
	if strings.Contains(down.Body.String(), "orchestrator:8000") {
		t.Fatalf("the orchestrator's error reached the browser: %s", down.Body)
	}
	rec := serve(&fakeExemplars{deleteErr: inUse}, http.MethodDelete, "/v1/admin/illustrations/bus")
	var body struct {
		Error    string                         `json:"error"`
		Code     string                         `json:"code"`
		Projects []application.IllustrationUser `json:"projects"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Code != "illustration_in_use" || len(body.Projects) != 1 || !strings.Contains(body.Error, "Sâu răng") {
		t.Fatalf("in-use body: %+v", body)
	}
}

func TestExemplarRoutes(t *testing.T) {
	if rec := serve(&fakeExemplars{}, http.MethodPost, "/v1/admin/illustrations/bus/exemplar"); rec.Code != http.StatusCreated ||
		!strings.Contains(rec.Body.String(), `"source_id":"bus"`) {
		t.Fatalf("make: %d %s", rec.Code, rec.Body)
	}
	if rec := serve(&fakeExemplars{}, http.MethodPost, "/v1/admin/illustrations/full/exemplar"); rec.Code != http.StatusConflict {
		t.Fatalf("make when full: %d", rec.Code)
	}
	if rec := serve(&fakeExemplars{}, http.MethodDelete, "/v1/admin/illustrations/i9/exemplar"); rec.Code != http.StatusNoContent {
		t.Fatalf("unmake a copy: %d", rec.Code)
	}
	back := &domain.Illustration{ID: "exemplar-Cat", FolderID: "dong-vat"}
	if rec := serve(&fakeExemplars{released: back}, http.MethodDelete, "/v1/admin/illustrations/exemplar-Cat/exemplar"); rec.Code != http.StatusOK ||
		!strings.Contains(rec.Body.String(), `"folder_id":"dong-vat"`) {
		t.Fatalf("unmake an original: %d %s", rec.Code, rec.Body)
	}
}

func TestIllustrationStyleListsTheExemplarsFromTheLibrary(t *testing.T) {
	rec := serve(&fakeExemplars{}, http.MethodGet, "/v1/illustration-style")
	var body struct {
		Rules        string   `json:"rules"`
		ExemplarIDs  []string `json:"exemplar_ids"`
		MaxExemplars int      `json:"max_exemplars"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK || strings.Join(body.ExemplarIDs, ",") != "exemplar-Cat,i7" || body.MaxExemplars != domain.MaxExemplars {
		t.Fatalf("style: %d %+v", rec.Code, body)
	}
	if body.Rules != "- [S1] LUẬT TỪ DB: dòng đang bật" {
		t.Fatalf("rules must be the active prompt row, got %q", body.Rules)
	}
}

func TestIllustrationStyleReportsAnUnreadablePromptLibrary(t *testing.T) {
	rec := serve(&fakeExemplars{styleErr: fmt.Errorf("illustration style rules: db down")}, http.MethodGet, "/v1/illustration-style")
	if rec.Code != http.StatusInternalServerError || !strings.Contains(rec.Body.String(), "db down") {
		t.Fatalf("got %d %s", rec.Code, rec.Body)
	}
}
