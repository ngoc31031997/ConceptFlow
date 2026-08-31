package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"orchestrator/internal/application"
	"orchestrator/internal/domain"
)

type fakeStartRenderSaga struct {
	out *application.StartRenderSagaOutput
	err error
}

func (f *fakeStartRenderSaga) Execute(_ context.Context, _ application.StartRenderSagaInput) (*application.StartRenderSagaOutput, error) {
	return f.out, f.err
}

type fakeStartPublishSaga struct {
	out *application.StartPublishSagaOutput
	err error
}

func (f *fakeStartPublishSaga) Execute(_ context.Context, _ application.StartPublishSagaInput) (*application.StartPublishSagaOutput, error) {
	return f.out, f.err
}

type fakeRetryStep struct {
	out *application.RetryStepOutput
	err error
}

func (f *fakeRetryStep) Execute(_ context.Context, _ string) (*application.RetryStepOutput, error) {
	return f.out, f.err
}

type fakeProjectReader struct {
	project *domain.Project
	err     error
}

func (f *fakeProjectReader) Get(_ context.Context, _ string) (*domain.Project, error) {
	return f.project, f.err
}

func TestHandleStartRenderSaga_Created(t *testing.T) {
	router := NewRouter(
		&fakeStartRenderSaga{out: &application.StartRenderSagaOutput{SagaID: "saga-1", Status: domain.StatusParsingScript}},
		&fakeStartPublishSaga{}, &fakeRetryStep{}, &fakeProjectReader{},
	)

	body, _ := json.Marshal(map[string]interface{}{
		"project_id": "p1", "script_content": "s", "plugin_id": "plugin", "voice_language": "en",
	})
	req := httptest.NewRequest("POST", "/v1/sagas/render", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.Handler().ServeHTTP(rec, req)

	if rec.Code != 201 {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp sagaStartedResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.SagaID != "saga-1" || resp.Status != string(domain.StatusParsingScript) {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestHandleStartRenderSaga_InvalidBody(t *testing.T) {
	router := NewRouter(&fakeStartRenderSaga{}, &fakeStartPublishSaga{}, &fakeRetryStep{}, &fakeProjectReader{})

	req := httptest.NewRequest("POST", "/v1/sagas/render", bytes.NewReader([]byte(`{"project_id":""}`)))
	rec := httptest.NewRecorder()
	router.Handler().ServeHTTP(rec, req)

	if rec.Code != 400 {
		t.Fatalf("expected 400 for missing required fields, got %d", rec.Code)
	}
}

func TestHandleStartPublishSaga_Conflict(t *testing.T) {
	router := NewRouter(
		&fakeStartRenderSaga{}, &fakeStartPublishSaga{err: domain.ErrInvalidStatus}, &fakeRetryStep{}, &fakeProjectReader{},
	)

	body, _ := json.Marshal(map[string]interface{}{"project_id": "p1", "youtube_title": "t", "visibility": "public"})
	req := httptest.NewRequest("POST", "/v1/sagas/publish", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.Handler().ServeHTTP(rec, req)

	if rec.Code != 409 {
		t.Fatalf("expected 409, got %d", rec.Code)
	}
}

func TestHandleGetProject_NotFound(t *testing.T) {
	router := NewRouter(
		&fakeStartRenderSaga{}, &fakeStartPublishSaga{}, &fakeRetryStep{},
		&fakeProjectReader{err: domain.ErrProjectNotFound},
	)

	req := httptest.NewRequest("GET", "/v1/projects/unknown", nil)
	rec := httptest.NewRecorder()
	router.Handler().ServeHTTP(rec, req)

	if rec.Code != 404 {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandleGetProject_OK(t *testing.T) {
	router := NewRouter(
		&fakeStartRenderSaga{}, &fakeStartPublishSaga{}, &fakeRetryStep{},
		&fakeProjectReader{project: &domain.Project{ProjectID: "p1", Status: domain.StatusDraft}},
	)

	req := httptest.NewRequest("GET", "/v1/projects/p1", nil)
	rec := httptest.NewRecorder()
	router.Handler().ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestHandleRetry_OK(t *testing.T) {
	router := NewRouter(
		&fakeStartRenderSaga{}, &fakeStartPublishSaga{},
		&fakeRetryStep{out: &application.RetryStepOutput{SagaID: "saga-1", Status: domain.StatusRendering}},
		&fakeProjectReader{},
	)

	req := httptest.NewRequest("POST", "/v1/projects/p1/retry", nil)
	rec := httptest.NewRecorder()
	router.Handler().ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleHealth(t *testing.T) {
	router := NewRouter(&fakeStartRenderSaga{}, &fakeStartPublishSaga{}, &fakeRetryStep{}, &fakeProjectReader{})

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()
	router.Handler().ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
