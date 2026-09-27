package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"authoring/internal/application"
	"authoring/internal/domain"
)

type fakeProjectIllustrations struct {
	projectIllustrationsUseCase
	err     error
	deleted string
}

func (f *fakeProjectIllustrations) DeleteDrawing(_ context.Context, _, rowID string) (domain.ProjectIllustration, error) {
	f.deleted = rowID
	return domain.ProjectIllustration{ID: rowID, State: domain.PISkipped}, f.err
}

// CR-045: deleting a video's AI draft from the library.
func TestDeleteProjectIllustrationDrawing(t *testing.T) {
	uc := &fakeProjectIllustrations{}
	h := NewRouter(nil, nil).WithProjectIllustrations(uc).Handler()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/v1/projects/p1/illustrations/r7/drawing", nil))
	if rec.Code != http.StatusOK || uc.deleted != "r7" || !strings.Contains(rec.Body.String(), `"state":"skipped"`) {
		t.Fatalf("got %d %s (deleted %q)", rec.Code, rec.Body.String(), uc.deleted)
	}

	uc.err = application.ErrIllustrationNotDeletable
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/v1/projects/p1/illustrations/r7/drawing", nil))
	if rec.Code != http.StatusConflict {
		t.Fatalf("an approved drawing must be refused with 409, got %d", rec.Code)
	}
}

func TestNeverPlannedDrawingsAreAConflictNotAnAIFailure(t *testing.T) {
	status, msg := DescribeGenerateError(application.ErrIllustrationsNotPlanned)
	if status != http.StatusConflict || !strings.Contains(msg, "Hình minh hoạ") {
		t.Fatalf("got %d %q", status, msg)
	}
}
