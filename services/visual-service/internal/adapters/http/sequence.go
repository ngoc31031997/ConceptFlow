package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"visual/internal/application"
)

// sequenceDesk backs the routes of step 10 "Chuỗi hình".
type sequenceDesk interface {
	State(ctx context.Context, projectID string) (*application.SequenceState, error)
	Edit(ctx context.Context, projectID, unitID string, fields map[string]json.RawMessage) (*application.SequenceState, error)
	Approve(ctx context.Context, projectID, unitID string) (*application.SequenceState, error)
	Unapprove(ctx context.Context, projectID, unitID string) (*application.SequenceState, error)
	ApproveAll(ctx context.Context, projectID string) (*application.SequenceState, error)
	ConfirmBeat(ctx context.Context, projectID, beatKey string) (*application.SequenceState, error)
	AddMotif(ctx context.Context, projectID string, fields map[string]json.RawMessage) (*application.SequenceState, error)
	EditMotif(ctx context.Context, projectID, motifID string, fields map[string]json.RawMessage) (*application.SequenceState, error)
	DeleteMotif(ctx context.Context, projectID, motifID string) (*application.SequenceState, error)
	ApproveMotifs(ctx context.Context, projectID string) (*application.SequenceState, error)
	UnapproveMotifs(ctx context.Context, projectID string) (*application.SequenceState, error)
	ApplyFinding(ctx context.Context, projectID, findingID string) (*application.SequenceState, error)
	AcceptFinding(ctx context.Context, projectID, findingID, why string) (*application.SequenceState, error)
	Lock(ctx context.Context, projectID, note string) (*application.SequenceState, error)
	Version(ctx context.Context, projectID string, version int) (application.SequenceVersionRecord, error)
	LatestDocument(ctx context.Context, projectID string) (json.RawMessage, error)
}

// WithSequence adds the routes of step 10 to the router.
func (rt *Router) WithSequence(desk sequenceDesk) *Router {
	rt.sequence = desk
	return rt
}

// fieldsBody is the body of an edit: the fields to write, as JSON values.
type fieldsBody struct {
	Fields map[string]json.RawMessage `json:"fields"`
}

func (rt *Router) mountSequence(r chi.Router) {
	r.Get("/internal/v1/visual/{project_id}/sequence", rt.handleLatestSequence)
	r.Route("/v1/projects/{project_id}/visual-sequence", func(r chi.Router) {
		r.Get("/", rt.seqProject(rt.sequence.State))
		r.Put("/units/{unit}", rt.seqFields("unit", rt.sequence.Edit))
		r.Post("/units/{unit}/approve", rt.seqParam("unit", rt.sequence.Approve))
		r.Post("/units/{unit}/unapprove", rt.seqParam("unit", rt.sequence.Unapprove))
		r.Post("/approve-all", rt.seqProject(rt.sequence.ApproveAll))
		r.Post("/beats/{beat}/confirm", rt.seqParam("beat", rt.sequence.ConfirmBeat))
		r.Post("/motifs", rt.seqAddMotif)
		r.Put("/motifs/{motif}", rt.seqFields("motif", rt.sequence.EditMotif))
		r.Delete("/motifs/{motif}", rt.seqParam("motif", rt.sequence.DeleteMotif))
		r.Post("/motifs/approve", rt.seqProject(rt.sequence.ApproveMotifs))
		r.Post("/motifs/unapprove", rt.seqProject(rt.sequence.UnapproveMotifs))
		r.Post("/findings/{finding}/apply", rt.seqParam("finding", rt.sequence.ApplyFinding))
		r.Post("/findings/{finding}/accept", rt.seqAccept)
		r.Post("/lock", rt.seqLock)
		r.Get("/versions/{version}", rt.seqVersion)
	})
}

// seqProject serves a call that takes only the project.
func (rt *Router) seqProject(fn func(ctx context.Context, projectID string) (*application.SequenceState, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		st, err := fn(r.Context(), pid(r))
		writeConcept(w, st, err)
	}
}

// seqParam serves a call that takes the project and one path parameter.
func (rt *Router) seqParam(param string, fn func(ctx context.Context, projectID, id string) (*application.SequenceState, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		st, err := fn(r.Context(), pid(r), chi.URLParam(r, param))
		writeConcept(w, st, err)
	}
}

// seqFields serves an edit of the item or motif named by a path parameter.
func (rt *Router) seqFields(param string, fn func(ctx context.Context, projectID, id string, fields map[string]json.RawMessage) (*application.SequenceState, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in fieldsBody
		if err := decodeBody(r, &in); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		st, err := fn(r.Context(), pid(r), chi.URLParam(r, param), in.Fields)
		writeConcept(w, st, err)
	}
}

func (rt *Router) seqAddMotif(w http.ResponseWriter, r *http.Request) {
	var in fieldsBody
	if err := decodeBody(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	st, err := rt.sequence.AddMotif(r.Context(), pid(r), in.Fields)
	writeConcept(w, st, err)
}

func (rt *Router) seqAccept(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Why string `json:"why"`
	}
	if err := decodeBody(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	st, err := rt.sequence.AcceptFinding(r.Context(), pid(r), chi.URLParam(r, "finding"), in.Why)
	writeConcept(w, st, err)
}

func (rt *Router) seqLock(w http.ResponseWriter, r *http.Request) {
	var in noteBody
	if err := decodeBody(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	st, err := rt.sequence.Lock(r.Context(), pid(r), in.Note)
	writeConcept(w, st, err)
}

func (rt *Router) seqVersion(w http.ResponseWriter, r *http.Request) {
	n, err := strconv.Atoi(chi.URLParam(r, "version"))
	if err != nil || n < 1 {
		writeError(w, http.StatusBadRequest, "số phiên bản không hợp lệ")
		return
	}
	v, err := rt.sequence.Version(r.Context(), pid(r), n)
	writeConcept(w, v, err)
}

// handleLatestSequence serves the latest locked visual sequence to a later step.
func (rt *Router) handleLatestSequence(w http.ResponseWriter, r *http.Request) {
	doc, err := rt.sequence.LatestDocument(r.Context(), pid(r))
	if err != nil {
		writeConceptError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(doc)
}
