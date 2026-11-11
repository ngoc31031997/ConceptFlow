package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"visual/internal/application"
	"visual/internal/domain"
)

type fakeLibrary struct {
	input domain.ReferenceStyleInput
	id    string
	err   error
}

func (f *fakeLibrary) List(context.Context) ([]domain.ReferenceStyle, error) {
	return []domain.ReferenceStyle{{ID: "R01", ReferenceStyleInput: domain.ReferenceStyleInput{StyleName: "n", Tags: []string{}}}}, f.err
}

func (f *fakeLibrary) Create(_ context.Context, in domain.ReferenceStyleInput) (domain.ReferenceStyle, error) {
	f.input = in
	return domain.ReferenceStyle{ID: "R02", ReferenceStyleInput: domain.ReferenceStyleInput{StyleName: in.StyleName}}, f.err
}

func (f *fakeLibrary) Update(_ context.Context, id string, in domain.ReferenceStyleInput) (domain.ReferenceStyle, error) {
	f.id, f.input = id, in
	return domain.ReferenceStyle{ID: id, ReferenceStyleInput: domain.ReferenceStyleInput{StyleName: in.StyleName}}, f.err
}

func (f *fakeLibrary) Delete(_ context.Context, id string) error {
	f.id = id
	return f.err
}

func (f *fakeLibrary) Validate(in domain.ReferenceStyleInput) (domain.ReferenceStyleInput, error) {
	f.input = in
	return in, f.err
}

func (f *fakeLibrary) UsedBy(_ context.Context, id string) ([]string, error) {
	f.id = id
	return []string{"p1", "p2"}, f.err
}

func doBody(rt *Router, method, path, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	rt.Handler().ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)))
	return rec
}

func libraryRouter(f *fakeLibrary) *Router {
	return NewRouter(&fakeDelete{}).WithReferenceLibrary(f)
}

func TestReferenceStyles_Routes(t *testing.T) {
	f := &fakeLibrary{}
	rt := libraryRouter(f)

	if rec := doBody(rt, http.MethodGet, "/v1/reference-styles", ""); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"styles":[{"id":"R01"`) {
		t.Errorf("list: %d %s", rec.Code, rec.Body.String())
	}
	if rec := doBody(rt, http.MethodPost, "/v1/reference-styles", `{"style_name":"a","intent":"d","tags":["x"]}`); rec.Code != http.StatusCreated || f.input.StyleName != "a" || len(f.input.Tags) != 1 {
		t.Errorf("create: %d input %+v", rec.Code, f.input)
	}
	if rec := doBody(rt, http.MethodPut, "/v1/reference-styles/R01", `{"style_name":"b","intent":"d"}`); rec.Code != http.StatusOK || f.id != "R01" || f.input.StyleName != "b" {
		t.Errorf("update: %d id %q input %+v", rec.Code, f.id, f.input)
	}
	if rec := doBody(rt, http.MethodDelete, "/v1/reference-styles/R03", ""); rec.Code != http.StatusNoContent || f.id != "R03" {
		t.Errorf("delete: %d id %q", rec.Code, f.id)
	}
	rec := doBody(rt, http.MethodGet, "/v1/reference-styles/R04/usage", "")
	if rec.Code != http.StatusOK || f.id != "R04" || strings.TrimSpace(rec.Body.String()) != `{"projects":[{"project_id":"p1"},{"project_id":"p2"}]}` {
		t.Errorf("usage: %d %s", rec.Code, rec.Body.String())
	}
}

func TestReferenceStyles_ErrorStatuses(t *testing.T) {
	cases := []struct {
		err  error
		want int
	}{
		{fmt.Errorf("%w: tên", domain.ErrInvalidReferenceStyle), http.StatusBadRequest},
		{application.ErrReferenceStyleNotFound, http.StatusNotFound},
		{errors.New("db down"), http.StatusInternalServerError},
	}
	for _, c := range cases {
		rt := libraryRouter(&fakeLibrary{err: c.err})
		rec := doBody(rt, http.MethodPut, "/v1/reference-styles/R01", `{"style_name":"n","intent":"d"}`)
		if rec.Code != c.want || !strings.Contains(rec.Body.String(), `"error"`) {
			t.Errorf("%v: status %d body %s, want %d", c.err, rec.Code, rec.Body.String(), c.want)
		}
	}
	if rec := doBody(libraryRouter(&fakeLibrary{}), http.MethodPost, "/v1/reference-styles", `{`); rec.Code != http.StatusBadRequest {
		t.Errorf("broken JSON: status %d, want 400", rec.Code)
	}
}

func TestReferenceStyles_ImportReturnsTheCleanedStyleWithoutStoring(t *testing.T) {
	f := &fakeLibrary{}
	rec := doBody(libraryRouter(f), http.MethodPost, "/v1/reference-styles/import",
		`{"style_name":"a","intent":"s","drawing":{"shape":"tròn"},"source":{"kind":"video"}}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"style":{"style_name":"a"`) || !strings.Contains(rec.Body.String(), `"shape":"tròn"`) {
		t.Errorf("import: %d %s", rec.Code, rec.Body.String())
	}
	if f.id != "" {
		t.Errorf("import must not touch a stored style, id %q", f.id)
	}
}

func TestReferenceStyles_ImportErrors(t *testing.T) {
	bad := fmt.Errorf("%w: drawing.shape: dài quá 300 ký tự", domain.ErrInvalidReferenceStyle)
	rec := doBody(libraryRouter(&fakeLibrary{err: bad}), http.MethodPost, "/v1/reference-styles/import", `{"style_name":"a","intent":"s"}`)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "drawing.shape: dài quá 300 ký tự") {
		t.Errorf("rule broken: %d %s", rec.Code, rec.Body.String())
	}
	rec = doBody(libraryRouter(&fakeLibrary{}), http.MethodPost, "/v1/reference-styles/import", `{"error":"không xem được link"}`)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "không xem được link") {
		t.Errorf("reported error: %d %s", rec.Code, rec.Body.String())
	}
	if rec := doBody(libraryRouter(&fakeLibrary{}), http.MethodPost, "/v1/reference-styles/import", `không phải JSON`); rec.Code != http.StatusBadRequest {
		t.Errorf("not JSON: %d", rec.Code)
	}
}
