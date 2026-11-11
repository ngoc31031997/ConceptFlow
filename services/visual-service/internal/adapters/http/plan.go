package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"visual/internal/application"
)

// planDesk backs the routes of step 9 "Kế hoạch hình".
type planDesk interface {
	State(ctx context.Context, projectID string) (*application.PlanState, error)
	SetModel(ctx context.Context, projectID, model string) (*application.PlanState, error)
	Generate(ctx context.Context, projectID, note string) (*application.PlanState, error)
	GenerateBeat(ctx context.Context, projectID, beatKey, note string) (*application.PlanState, error)
	Review(ctx context.Context, projectID, note string) (*application.PlanState, error)
	Edit(ctx context.Context, projectID, unitID string, fields map[string]json.RawMessage) (*application.PlanState, error)
	SetAnswer(ctx context.Context, projectID, unitID, questionID, answer string) (*application.PlanState, error)
	Split(ctx context.Context, projectID, unitID, atLine string) (*application.PlanState, error)
	Merge(ctx context.Context, projectID, unitID, withID string) (*application.PlanState, error)
	Approve(ctx context.Context, projectID, unitID string) (*application.PlanState, error)
	Unapprove(ctx context.Context, projectID, unitID string) (*application.PlanState, error)
	ApproveAll(ctx context.Context, projectID string) (*application.PlanState, error)
	AutoFix(ctx context.Context, projectID string) (*application.PlanState, error)
	ApplyFinding(ctx context.Context, projectID, findingID string) (*application.PlanState, error)
	AcceptFinding(ctx context.Context, projectID, findingID, why string) (*application.PlanState, error)
	Lock(ctx context.Context, projectID, note string) (*application.PlanState, error)
	Version(ctx context.Context, projectID string, version int) (application.PlanVersionRecord, error)
	LatestDocument(ctx context.Context, projectID string) (json.RawMessage, error)
}

// WithPlan adds the routes of step 9 to the router.
func (rt *Router) WithPlan(desk planDesk) *Router {
	rt.plan = desk
	return rt
}

func (rt *Router) mountPlan(r chi.Router) {
	r.Get("/internal/v1/visual/{project_id}/plan", rt.handleLatestPlan)
	r.Route("/v1/projects/{project_id}/visual-plan", func(r chi.Router) {
		r.Get("/", rt.planState)
		r.Put("/model", rt.planModel)
		r.Post("/generate", rt.planGenerate)
		r.With(externalAI).Post("/beats/{beat}/generate", rt.planGenerateBeat)
		r.With(externalAI).Post("/review", rt.planReview)
		r.Put("/units/{unit}", rt.planEdit)
		r.Put("/units/{unit}/questions/{question}", rt.planAnswer)
		r.Post("/units/{unit}/split", rt.planSplit)
		r.Post("/units/{unit}/merge", rt.planMerge)
		r.Post("/units/{unit}/approve", rt.planUnitAction(func(ctx context.Context, p, u string) (*application.PlanState, error) {
			return rt.plan.Approve(ctx, p, u)
		}))
		r.Post("/units/{unit}/unapprove", rt.planUnitAction(func(ctx context.Context, p, u string) (*application.PlanState, error) {
			return rt.plan.Unapprove(ctx, p, u)
		}))
		r.Post("/approve-all", rt.planApproveAll)
		r.Post("/auto-fix", rt.planAutoFix)
		r.Post("/findings/{finding}/apply", rt.planApply)
		r.Post("/findings/{finding}/accept", rt.planAccept)
		r.Post("/lock", rt.planLock)
		r.Get("/versions/{version}", rt.planVersion)
	})
}

func (rt *Router) planState(w http.ResponseWriter, r *http.Request) {
	st, err := rt.plan.State(r.Context(), pid(r))
	writeConcept(w, st, err)
}

func (rt *Router) planModel(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Model string `json:"model"`
	}
	if err := decodeBody(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	st, err := rt.plan.SetModel(r.Context(), pid(r), in.Model)
	writeConcept(w, st, err)
}

func (rt *Router) planGenerate(w http.ResponseWriter, r *http.Request) {
	var in noteBody
	if err := decodeBody(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	st, err := rt.plan.Generate(r.Context(), pid(r), in.Note)
	writeConcept(w, st, err)
}

func (rt *Router) planGenerateBeat(w http.ResponseWriter, r *http.Request) {
	var in noteBody
	if err := decodeBody(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	st, err := rt.plan.GenerateBeat(r.Context(), pid(r), chi.URLParam(r, "beat"), in.Note)
	writeConcept(w, st, err)
}

func (rt *Router) planReview(w http.ResponseWriter, r *http.Request) {
	var in noteBody
	if err := decodeBody(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	st, err := rt.plan.Review(r.Context(), pid(r), in.Note)
	writeConcept(w, st, err)
}

func (rt *Router) planEdit(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Fields map[string]json.RawMessage `json:"fields"`
	}
	if err := decodeBody(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	st, err := rt.plan.Edit(r.Context(), pid(r), chi.URLParam(r, "unit"), in.Fields)
	writeConcept(w, st, err)
}

func (rt *Router) planAnswer(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Answer string `json:"answer"`
	}
	if err := decodeBody(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	st, err := rt.plan.SetAnswer(r.Context(), pid(r), chi.URLParam(r, "unit"), chi.URLParam(r, "question"), in.Answer)
	writeConcept(w, st, err)
}

func (rt *Router) planSplit(w http.ResponseWriter, r *http.Request) {
	var in struct {
		AtLine string `json:"at_line"`
	}
	if err := decodeBody(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	st, err := rt.plan.Split(r.Context(), pid(r), chi.URLParam(r, "unit"), in.AtLine)
	writeConcept(w, st, err)
}

func (rt *Router) planMerge(w http.ResponseWriter, r *http.Request) {
	var in struct {
		With string `json:"with"`
	}
	if err := decodeBody(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	st, err := rt.plan.Merge(r.Context(), pid(r), chi.URLParam(r, "unit"), in.With)
	writeConcept(w, st, err)
}

func (rt *Router) planUnitAction(fn func(ctx context.Context, projectID, unitID string) (*application.PlanState, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		st, err := fn(r.Context(), pid(r), chi.URLParam(r, "unit"))
		writeConcept(w, st, err)
	}
}

func (rt *Router) planApproveAll(w http.ResponseWriter, r *http.Request) {
	st, err := rt.plan.ApproveAll(r.Context(), pid(r))
	writeConcept(w, st, err)
}

func (rt *Router) planAutoFix(w http.ResponseWriter, r *http.Request) {
	st, err := rt.plan.AutoFix(r.Context(), pid(r))
	writeConcept(w, st, err)
}

func (rt *Router) planApply(w http.ResponseWriter, r *http.Request) {
	st, err := rt.plan.ApplyFinding(r.Context(), pid(r), chi.URLParam(r, "finding"))
	writeConcept(w, st, err)
}

func (rt *Router) planAccept(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Why string `json:"why"`
	}
	if err := decodeBody(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	st, err := rt.plan.AcceptFinding(r.Context(), pid(r), chi.URLParam(r, "finding"), in.Why)
	writeConcept(w, st, err)
}

func (rt *Router) planLock(w http.ResponseWriter, r *http.Request) {
	var in noteBody
	if err := decodeBody(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	st, err := rt.plan.Lock(r.Context(), pid(r), in.Note)
	writeConcept(w, st, err)
}

func (rt *Router) planVersion(w http.ResponseWriter, r *http.Request) {
	n, err := strconv.Atoi(chi.URLParam(r, "version"))
	if err != nil || n < 1 {
		writeError(w, http.StatusBadRequest, "số phiên bản không hợp lệ")
		return
	}
	v, err := rt.plan.Version(r.Context(), pid(r), n)
	writeConcept(w, v, err)
}

// handleLatestPlan serves the latest locked visual plan to a later step.
func (rt *Router) handleLatestPlan(w http.ResponseWriter, r *http.Request) {
	doc, err := rt.plan.LatestDocument(r.Context(), pid(r))
	if err != nil {
		writeConceptError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(doc)
}
