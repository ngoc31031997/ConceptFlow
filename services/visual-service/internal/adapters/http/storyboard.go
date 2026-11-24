package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"visual/internal/application"
)

// storyboardDesk backs the routes of step 11 "Storyboard".
type storyboardDesk interface {
	State(ctx context.Context, projectID string) (*application.StoryboardScreen, error)
	SetModel(ctx context.Context, projectID, model string) (*application.StoryboardScreen, error)
	GenerateAssets(ctx context.Context, projectID, note string) (*application.StoryboardScreen, error)
	GenerateAssetDetail(ctx context.Context, projectID, assetID, note string) (*application.StoryboardScreen, error)
	GenerateAllAssetDetails(ctx context.Context, projectID, note string) (*application.StoryboardScreen, error)
	AddAsset(ctx context.Context, projectID string, fields map[string]json.RawMessage) (*application.StoryboardScreen, error)
	EditAsset(ctx context.Context, projectID, assetID string, fields map[string]json.RawMessage) (*application.StoryboardScreen, error)
	DeleteAsset(ctx context.Context, projectID, assetID string) (*application.StoryboardScreen, error)
	DrawAsset(ctx context.Context, projectID, assetID string) (*application.StoryboardScreen, error)
	RedrawAsset(ctx context.Context, projectID, assetID, note string) (*application.StoryboardScreen, error)
	FixAsset(ctx context.Context, projectID, assetID string, fix application.AssetDrawFix, redraw bool) (*application.StoryboardScreen, error)
	ApproveAsset(ctx context.Context, projectID, assetID, projectName string) (*application.StoryboardScreen, error)
	UnapproveAsset(ctx context.Context, projectID, assetID string) (*application.StoryboardScreen, error)
	ApproveAssets(ctx context.Context, projectID string) (*application.StoryboardScreen, error)
	UnapproveAssets(ctx context.Context, projectID string) (*application.StoryboardScreen, error)
	Generate(ctx context.Context, projectID, note string) (*application.StoryboardScreen, error)
	GenerateUnit(ctx context.Context, projectID, unitID, note string) (*application.StoryboardScreen, error)
	AddShot(ctx context.Context, projectID, unitID string, fields map[string]json.RawMessage) (*application.StoryboardScreen, error)
	EditShot(ctx context.Context, projectID, unitID, shotID string, fields map[string]json.RawMessage) (*application.StoryboardScreen, error)
	DeleteShot(ctx context.Context, projectID, unitID, shotID string) (*application.StoryboardScreen, error)
	DrawFrame(ctx context.Context, projectID, unitID, shotID string) (*application.StoryboardScreen, error)
	ApproveShot(ctx context.Context, projectID, unitID, shotID string) (*application.StoryboardScreen, error)
	UnapproveShot(ctx context.Context, projectID, unitID, shotID string) (*application.StoryboardScreen, error)
	ApproveAll(ctx context.Context, projectID string) (*application.StoryboardScreen, error)
	Review(ctx context.Context, projectID, note string) (*application.StoryboardScreen, error)
	ApplyFinding(ctx context.Context, projectID, findingID string) (*application.StoryboardScreen, error)
	AcceptFinding(ctx context.Context, projectID, findingID, why string) (*application.StoryboardScreen, error)
	Lock(ctx context.Context, projectID, note string) (*application.StoryboardScreen, error)
	Version(ctx context.Context, projectID string, version int) (application.StoryboardVersionRecord, error)
	LatestDocument(ctx context.Context, projectID string) (json.RawMessage, error)
	FrameImage(ctx context.Context, projectID, shotID string) ([]byte, error)
	AssetPreview(ctx context.Context, projectID, assetID string) ([]byte, error)
}

// WithStoryboard adds the routes of step 11 to the router.
func (rt *Router) WithStoryboard(desk storyboardDesk) *Router {
	rt.storyboard = desk
	return rt
}

func (rt *Router) mountStoryboard(r chi.Router) {
	sb := rt.storyboard
	r.Get("/internal/v1/visual/{project_id}/storyboard", rt.handleLatestStoryboard)
	r.Route("/v1/projects/{project_id}/storyboard", func(r chi.Router) {
		r.Get("/", sbProject(sb.State))
		r.Put("/model", rt.sbModel)
		r.With(externalAI).Post("/assets/generate", sbNote(sb.GenerateAssets))
		r.Post("/assets/detail", sbNote(sb.GenerateAllAssetDetails))
		r.With(externalAI).Post("/assets/{aid}/detail", sbGenerate(sb.GenerateAssetDetail, "aid"))
		r.Post("/assets", rt.sbAddAsset)
		r.Post("/assets/approve", sbProject(sb.ApproveAssets))
		r.Post("/assets/unapprove", sbProject(sb.UnapproveAssets))
		r.Put("/assets/{aid}", sbFields(sb.EditAsset, "aid"))
		r.Delete("/assets/{aid}", sbParam(sb.DeleteAsset, "aid"))
		r.With(externalAI).Post("/assets/{aid}/draw", rt.sbDrawAsset)
		r.With(externalAI).Post("/assets/{aid}/redraw", rt.sbRedrawAsset)
		r.With(externalAI).Post("/assets/{aid}/fix", rt.sbFixAsset)
		r.Post("/assets/{aid}/approve", rt.sbApproveAsset)
		r.Get("/assets/{aid}/preview", rt.sbAssetPreview)
		r.Post("/assets/{aid}/unapprove", sbParam(sb.UnapproveAsset, "aid"))
		r.Post("/generate", sbNote(sb.Generate))
		r.With(externalAI).Post("/units/{vp}/generate", sbGenerate(sb.GenerateUnit, "vp"))
		r.Post("/units/{vp}/shots", rt.sbAddShot)
		r.Put("/units/{vp}/shots/{sid}", rt.sbEditShot)
		r.Delete("/units/{vp}/shots/{sid}", sbShot(sb.DeleteShot))
		r.Post("/units/{vp}/shots/{sid}/frame", sbShot(sb.DrawFrame))
		r.Post("/units/{vp}/shots/{sid}/approve", sbShot(sb.ApproveShot))
		r.Post("/units/{vp}/shots/{sid}/unapprove", sbShot(sb.UnapproveShot))
		r.Post("/approve-all", sbProject(sb.ApproveAll))
		r.With(externalAI).Post("/review", sbNote(sb.Review))
		r.Post("/findings/{finding}/apply", sbParam(sb.ApplyFinding, "finding"))
		r.Post("/findings/{finding}/accept", rt.sbAccept)
		r.Post("/lock", rt.sbLock)
		r.Get("/versions/{version}", rt.sbVersion)
		r.Get("/frames/{shot}", rt.sbFrame)
	})
}

type sbCall = func(ctx context.Context, projectID string) (*application.StoryboardScreen, error)

// sbProject serves a call that takes only the project.
func sbProject(fn sbCall) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		st, err := fn(r.Context(), pid(r))
		writeConcept(w, st, err)
	}
}

// sbParam serves a call that takes the project and one path parameter.
func sbParam(fn func(ctx context.Context, projectID, id string) (*application.StoryboardScreen, error), param string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		st, err := fn(r.Context(), pid(r), chi.URLParam(r, param))
		writeConcept(w, st, err)
	}
}

// sbShot serves a call on one shot, named by the unit and shot parameters.
func sbShot(fn func(ctx context.Context, projectID, unitID, shotID string) (*application.StoryboardScreen, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		st, err := fn(r.Context(), pid(r), chi.URLParam(r, "vp"), chi.URLParam(r, "sid"))
		writeConcept(w, st, err)
	}
}

// sbNote serves a call that takes the project and the Creator's note.
func sbNote(fn func(ctx context.Context, projectID, note string) (*application.StoryboardScreen, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in noteBody
		if err := decodeBody(r, &in); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		st, err := fn(r.Context(), pid(r), in.Note)
		writeConcept(w, st, err)
	}
}

// sbGenerate serves a call on one asset or unit, named by a path parameter, with
// the Creator's note.
func sbGenerate(fn func(ctx context.Context, projectID, id, note string) (*application.StoryboardScreen, error), param string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in noteBody
		if err := decodeBody(r, &in); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		st, err := fn(r.Context(), pid(r), chi.URLParam(r, param), in.Note)
		writeConcept(w, st, err)
	}
}

// sbFields serves an edit of the asset named by a path parameter.
func sbFields(fn func(ctx context.Context, projectID, id string, fields map[string]json.RawMessage) (*application.StoryboardScreen, error), param string) http.HandlerFunc {
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

func (rt *Router) sbModel(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Model string `json:"model"`
	}
	if err := decodeBody(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	st, err := rt.storyboard.SetModel(r.Context(), pid(r), in.Model)
	writeConcept(w, st, err)
}

// sbApproveAsset approves an asset; the body is optional and may name the
// project the library description is written with.
func (rt *Router) sbApproveAsset(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ProjectName string `json:"project_name"`
	}
	if err := decodeBody(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	st, err := rt.storyboard.ApproveAsset(r.Context(), pid(r), chi.URLParam(r, "aid"), in.ProjectName)
	writeConcept(w, st, err)
}

// writeAssetDraw answers a drawing of an asset with the screen. Drawn code
// rendering refused answers 422 {"error", "detail": {kind, message, line,
// snippet}}, the structured error of rendering passed on unchanged; any other
// error is mapped like the other step routes.
func writeAssetDraw(w http.ResponseWriter, st *application.StoryboardScreen, err error) {
	var refused *application.AssetDrawCheckError
	if errors.As(err, &refused) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": refused.Detail.Message, "detail": refused.Detail})
		return
	}
	writeConcept(w, st, err)
}

// sbDrawAsset draws an asset for the first time; the body is not read.
func (rt *Router) sbDrawAsset(w http.ResponseWriter, r *http.Request) {
	st, err := rt.storyboard.DrawAsset(r.Context(), pid(r), chi.URLParam(r, "aid"))
	writeAssetDraw(w, st, err)
}

// sbRedrawAsset draws an asset again from its code with {note}.
func (rt *Router) sbRedrawAsset(w http.ResponseWriter, r *http.Request) {
	var in noteBody
	if err := decodeBody(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	st, err := rt.storyboard.RedrawAsset(r.Context(), pid(r), chi.URLParam(r, "aid"), in.Note)
	writeAssetDraw(w, st, err)
}

// sbFixAsset sends the code the AI drew for an asset back with rendering's
// error: {code, error: {kind, message, line, snippet}, redraw?}. redraw keeps
// the asset's current code as the context, like a redraw.
func (rt *Router) sbFixAsset(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Code   string                      `json:"code"`
		Error  *application.AssetDrawError `json:"error"`
		Redraw bool                        `json:"redraw"`
	}
	if err := decodeBody(r, &in); err != nil || in.Error == nil {
		writeError(w, http.StatusBadRequest, "thân cần {code, error: {kind, message, line, snippet}, redraw?}")
		return
	}
	st, err := rt.storyboard.FixAsset(r.Context(), pid(r), chi.URLParam(r, "aid"), application.AssetDrawFix{Code: in.Code, Error: *in.Error}, in.Redraw)
	writeAssetDraw(w, st, err)
}

// sbAssetPreview serves the PNG of an asset's drawing, read through rendering.
func (rt *Router) sbAssetPreview(w http.ResponseWriter, r *http.Request) {
	png, err := rt.storyboard.AssetPreview(r.Context(), pid(r), chi.URLParam(r, "aid"))
	if err != nil {
		writeConceptError(w, err)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "private, no-cache")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(png)
}

func (rt *Router) sbAddAsset(w http.ResponseWriter, r *http.Request) {
	var in fieldsBody
	if err := decodeBody(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	st, err := rt.storyboard.AddAsset(r.Context(), pid(r), in.Fields)
	writeConcept(w, st, err)
}

func (rt *Router) sbAddShot(w http.ResponseWriter, r *http.Request) {
	var in fieldsBody
	if err := decodeBody(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	st, err := rt.storyboard.AddShot(r.Context(), pid(r), chi.URLParam(r, "vp"), in.Fields)
	writeConcept(w, st, err)
}

func (rt *Router) sbEditShot(w http.ResponseWriter, r *http.Request) {
	var in fieldsBody
	if err := decodeBody(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	st, err := rt.storyboard.EditShot(r.Context(), pid(r), chi.URLParam(r, "vp"), chi.URLParam(r, "sid"), in.Fields)
	writeConcept(w, st, err)
}

func (rt *Router) sbAccept(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Why string `json:"why"`
	}
	if err := decodeBody(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	st, err := rt.storyboard.AcceptFinding(r.Context(), pid(r), chi.URLParam(r, "finding"), in.Why)
	writeConcept(w, st, err)
}

func (rt *Router) sbLock(w http.ResponseWriter, r *http.Request) {
	var in noteBody
	if err := decodeBody(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	st, err := rt.storyboard.Lock(r.Context(), pid(r), in.Note)
	writeConcept(w, st, err)
}

func (rt *Router) sbVersion(w http.ResponseWriter, r *http.Request) {
	n, err := strconv.Atoi(chi.URLParam(r, "version"))
	if err != nil || n < 1 {
		writeError(w, http.StatusBadRequest, "số phiên bản không hợp lệ")
		return
	}
	v, err := rt.storyboard.Version(r.Context(), pid(r), n)
	writeConcept(w, v, err)
}

// sbFrame serves the PNG of a shot's last drawn frame, read through rendering.
func (rt *Router) sbFrame(w http.ResponseWriter, r *http.Request) {
	png, err := rt.storyboard.FrameImage(r.Context(), pid(r), chi.URLParam(r, "shot"))
	if err != nil {
		writeConceptError(w, err)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "private, no-cache")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(png)
}

// handleLatestStoryboard serves the latest locked storyboard to a later step.
func (rt *Router) handleLatestStoryboard(w http.ResponseWriter, r *http.Request) {
	doc, err := rt.storyboard.LatestDocument(r.Context(), pid(r))
	if err != nil {
		writeConceptError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(doc)
}

// writeRenderingError passes a refusal of rendering on with its status and
// reason; a 5xx other than 503 is a bad gateway. The keys rendering adds to its
// answer (the assets that are not ready, the frames that are missing) are kept.
func writeRenderingError(w http.ResponseWriter, err error, re *application.RenderingError) {
	status := re.Status
	if status >= 500 && status != http.StatusServiceUnavailable {
		status = http.StatusBadGateway
	}
	body := map[string]any{"error": err.Error()}
	for k, v := range re.Extra {
		body[k] = v
	}
	switch {
	case re.Extra["assets"] != nil:
		body["code"] = "assets_not_ready"
	case re.Extra["missing_frames"] != nil:
		body["code"] = "missing_frames"
	}
	writeJSON(w, status, body)
}
