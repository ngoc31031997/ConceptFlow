package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"visual/internal/application"
	"visual/internal/domain"
)

// conceptDesk backs the routes of step 8 "Ngôn ngữ hình ảnh".
type conceptDesk interface {
	State(ctx context.Context, projectID string) (*application.ConceptState, error)
	SetRefs(ctx context.Context, projectID string, refIDs []string) (*application.ConceptState, error)
	SetModel(ctx context.Context, projectID, model string) (*application.ConceptState, error)
	Generate(ctx context.Context, projectID, note string, rewriteAll bool) (*application.ConceptState, error)
	GenerateUnit(ctx context.Context, projectID, key, note string) (*application.ConceptState, error)
	Edit(ctx context.Context, projectID, key string, fields map[string]json.RawMessage) (*application.ConceptState, error)
	Approve(ctx context.Context, projectID, key string) (*application.ConceptState, error)
	Unapprove(ctx context.Context, projectID, key string) (*application.ConceptState, error)
	ApproveAll(ctx context.Context, projectID string) (*application.ConceptState, error)
	Review(ctx context.Context, projectID, note string) (*application.ConceptState, error)
	ApplyFinding(ctx context.Context, projectID, findingID string) (*application.ConceptState, error)
	AcceptFinding(ctx context.Context, projectID, findingID, why string) (*application.ConceptState, error)
	Lock(ctx context.Context, projectID, note string) (*application.ConceptState, error)
	Version(ctx context.Context, projectID string, version int) (application.VersionRecord, error)
	LatestDocument(ctx context.Context, projectID, step string) (json.RawMessage, error)
}

// WithConcept adds the routes of step 8 to the router.
func (rt *Router) WithConcept(desk conceptDesk) *Router {
	rt.concept = desk
	return rt
}

func (rt *Router) mountConcept(r chi.Router) {
	r.Get("/internal/v1/visual/{project_id}/concept", rt.handleLatestConcept)
	r.Route("/v1/projects/{project_id}/visual-concept", func(r chi.Router) {
		r.Get("/", rt.conceptState)
		r.Put("/refs", rt.conceptRefs)
		r.Put("/model", rt.conceptModel)
		r.Post("/generate", rt.conceptGenerate)
		r.With(externalAI).Post("/units/{key}/generate", rt.conceptGenerateUnit)
		r.Put("/units/{key}", rt.conceptEdit)
		r.Post("/units/{key}/approve", rt.conceptUnitAction(func(ctx context.Context, p, k string) (*application.ConceptState, error) {
			return rt.concept.Approve(ctx, p, k)
		}))
		r.Post("/units/{key}/unapprove", rt.conceptUnitAction(func(ctx context.Context, p, k string) (*application.ConceptState, error) {
			return rt.concept.Unapprove(ctx, p, k)
		}))
		r.Post("/approve-all", rt.conceptApproveAll)
		r.With(externalAI).Post("/review", rt.conceptReview)
		r.Post("/findings/{finding}/apply", rt.conceptApply)
		r.Post("/findings/{finding}/accept", rt.conceptAccept)
		r.Post("/lock", rt.conceptLock)
		r.Get("/versions/{version}", rt.conceptVersion)
	})
}

func pid(r *http.Request) string { return chi.URLParam(r, "project_id") }

// decodeBody reads an optional JSON body into v; an empty body leaves v as is.
func decodeBody(r *http.Request, v any) error {
	raw, err := io.ReadAll(io.LimitReader(r.Body, 4<<20))
	if err != nil {
		return err
	}
	if len(raw) == 0 {
		return nil
	}
	return json.Unmarshal(raw, v)
}

// writeConcept answers with the screen, or maps err to its status.
func writeConcept(w http.ResponseWriter, st any, err error) {
	if err != nil {
		writeConceptError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, st)
}

type codedError struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}

func writeConceptError(w http.ResponseWriter, err error) {
	status, code := http.StatusInternalServerError, ""
	switch {
	case errors.Is(err, application.ErrExternalAIPromptBuilt):
		// The external-AI middleware answers with the prompt.
		status = http.StatusOK
	case errors.Is(err, domain.ErrScriptNotLocked):
		status, code = http.StatusConflict, "script_not_locked"
	case errors.Is(err, domain.ErrConceptNotLocked):
		status, code = http.StatusConflict, "concept_not_locked"
	case errors.Is(err, domain.ErrBeatStale):
		status, code = http.StatusConflict, "beat_stale"
	case errors.Is(err, domain.ErrUnknownUnit), errors.Is(err, domain.ErrUnknownFinding), errors.Is(err, domain.ErrNoVersion),
		errors.Is(err, domain.ErrUnknownPlanUnit), errors.Is(err, domain.ErrUnknownQuestion):
		status = http.StatusNotFound
	case errors.Is(err, domain.ErrInvalidEdit), errors.Is(err, domain.ErrFindingNotApplicable), errors.Is(err, domain.ErrUnknownReference),
		errors.Is(err, application.ErrUnknownModel), errors.Is(err, domain.ErrUnknownStep), errors.Is(err, application.ErrNoProjectID),
		errors.Is(err, application.ErrExternalAISingleTurn), errors.Is(err, domain.ErrNotSplittable), errors.Is(err, domain.ErrNotMergeable):
		status = http.StatusBadRequest
	case errors.Is(err, domain.ErrUnitRunning), errors.Is(err, domain.ErrUnitInvalid), errors.Is(err, domain.ErrUnitNotWritten),
		errors.Is(err, domain.ErrLockNotReady), errors.Is(err, domain.ErrNothingToLock), errors.Is(err, domain.ErrVersionExists):
		status = http.StatusConflict
	case errors.Is(err, domain.ErrUnreadableReply), errors.Is(err, application.ErrLLMFailed), errors.Is(err, application.ErrPromptNotFound),
		errors.Is(err, application.ErrAuthoringFailed), errors.Is(err, application.ErrRenderingFailed):
		status = http.StatusBadGateway
	}
	msg := err.Error()
	if status == http.StatusInternalServerError {
		msg = "internal error: " + msg
	}
	writeJSON(w, status, codedError{Error: msg, Code: code})
}

func (rt *Router) conceptState(w http.ResponseWriter, r *http.Request) {
	st, err := rt.concept.State(r.Context(), pid(r))
	writeConcept(w, st, err)
}

func (rt *Router) conceptRefs(w http.ResponseWriter, r *http.Request) {
	var in struct {
		RefIDs []string `json:"ref_ids"`
	}
	if err := decodeBody(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	st, err := rt.concept.SetRefs(r.Context(), pid(r), in.RefIDs)
	writeConcept(w, st, err)
}

func (rt *Router) conceptModel(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Model string `json:"model"`
	}
	if err := decodeBody(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	st, err := rt.concept.SetModel(r.Context(), pid(r), in.Model)
	writeConcept(w, st, err)
}

type noteBody struct {
	Note       string `json:"note"`
	RewriteAll bool   `json:"rewrite_all"`
}

func (rt *Router) conceptGenerate(w http.ResponseWriter, r *http.Request) {
	var in noteBody
	if err := decodeBody(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	st, err := rt.concept.Generate(r.Context(), pid(r), in.Note, in.RewriteAll)
	writeConcept(w, st, err)
}

func (rt *Router) conceptGenerateUnit(w http.ResponseWriter, r *http.Request) {
	var in noteBody
	if err := decodeBody(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	st, err := rt.concept.GenerateUnit(r.Context(), pid(r), chi.URLParam(r, "key"), in.Note)
	writeConcept(w, st, err)
}

func (rt *Router) conceptEdit(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Fields map[string]json.RawMessage `json:"fields"`
	}
	if err := decodeBody(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	st, err := rt.concept.Edit(r.Context(), pid(r), chi.URLParam(r, "key"), in.Fields)
	writeConcept(w, st, err)
}

func (rt *Router) conceptUnitAction(fn func(ctx context.Context, projectID, key string) (*application.ConceptState, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		st, err := fn(r.Context(), pid(r), chi.URLParam(r, "key"))
		writeConcept(w, st, err)
	}
}

func (rt *Router) conceptApproveAll(w http.ResponseWriter, r *http.Request) {
	st, err := rt.concept.ApproveAll(r.Context(), pid(r))
	writeConcept(w, st, err)
}

func (rt *Router) conceptReview(w http.ResponseWriter, r *http.Request) {
	var in noteBody
	if err := decodeBody(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	st, err := rt.concept.Review(r.Context(), pid(r), in.Note)
	writeConcept(w, st, err)
}

func (rt *Router) conceptApply(w http.ResponseWriter, r *http.Request) {
	st, err := rt.concept.ApplyFinding(r.Context(), pid(r), chi.URLParam(r, "finding"))
	writeConcept(w, st, err)
}

func (rt *Router) conceptAccept(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Why string `json:"why"`
	}
	if err := decodeBody(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	st, err := rt.concept.AcceptFinding(r.Context(), pid(r), chi.URLParam(r, "finding"), in.Why)
	writeConcept(w, st, err)
}

func (rt *Router) conceptLock(w http.ResponseWriter, r *http.Request) {
	var in noteBody
	if err := decodeBody(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	st, err := rt.concept.Lock(r.Context(), pid(r), in.Note)
	writeConcept(w, st, err)
}

func (rt *Router) conceptVersion(w http.ResponseWriter, r *http.Request) {
	n, err := strconv.Atoi(chi.URLParam(r, "version"))
	if err != nil || n < 1 {
		writeError(w, http.StatusBadRequest, "số phiên bản không hợp lệ")
		return
	}
	v, err := rt.concept.Version(r.Context(), pid(r), n)
	writeConcept(w, v, err)
}

// handleLatestConcept serves the latest locked visual concept to a later step.
func (rt *Router) handleLatestConcept(w http.ResponseWriter, r *http.Request) {
	doc, err := rt.concept.LatestDocument(r.Context(), pid(r), r.URL.Query().Get("step"))
	if err != nil {
		writeConceptError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(doc)
}
