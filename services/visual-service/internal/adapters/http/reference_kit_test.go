package http

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"visual/internal/application"
	"visual/internal/domain"
)

func (f *fakeLibrary) KitOf(_ context.Context, id string) (application.ReferenceKitResult, error) {
	f.id = id
	return application.ReferenceKitResult{RefID: id, Kit: domain.ReferenceKit{Ink: "#112233"}}, f.err
}

func (f *fakeLibrary) KitOfProject(_ context.Context, projectID string) (application.ReferenceKitResult, error) {
	f.id = projectID
	return application.ReferenceKitResult{RefID: "R01", Kit: domain.ReferenceKit{Ink: "#112233"}}, f.err
}

func TestReferenceStyles_KitRoutes(t *testing.T) {
	f := &fakeLibrary{}
	rt := libraryRouter(f)
	rec := doBody(rt, http.MethodGet, "/internal/v1/reference-styles/R05/kit", "")
	if rec.Code != http.StatusOK || f.id != "R05" || !strings.Contains(rec.Body.String(), `"ref_id":"R05"`) || !strings.Contains(rec.Body.String(), `"ink":"#112233"`) {
		t.Errorf("style kit: %d %s", rec.Code, rec.Body.String())
	}
	rec = doBody(rt, http.MethodGet, "/internal/v1/visual/p1/kit", "")
	if rec.Code != http.StatusOK || f.id != "p1" || !strings.Contains(rec.Body.String(), `"ref_id":"R01"`) {
		t.Errorf("project kit: %d %s", rec.Code, rec.Body.String())
	}
}

func TestReferenceStyles_KitErrorStatuses(t *testing.T) {
	cases := []struct {
		err  error
		want int
	}{
		{application.ErrNoReferencePicked, http.StatusNotFound},
		{application.ErrReferenceStyleNotFound, http.StatusNotFound},
		{application.ErrReferenceKitMissing, http.StatusConflict},
	}
	for _, c := range cases {
		rec := doBody(libraryRouter(&fakeLibrary{err: c.err}), http.MethodGet, "/internal/v1/visual/p1/kit", "")
		if rec.Code != c.want || !strings.Contains(rec.Body.String(), `"error"`) {
			t.Errorf("%v: status %d body %s, want %d", c.err, rec.Code, rec.Body.String(), c.want)
		}
	}
}
