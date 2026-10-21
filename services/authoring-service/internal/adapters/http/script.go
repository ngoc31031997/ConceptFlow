package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"authoring/internal/application"
	"authoring/internal/domain"
)

// Operation kind a script AI run is recorded under, so that any page can follow it.
const operationScript = "script_beat"

// scriptUseCase is the script step beyond the generic unit API.
type scriptUseCase interface {
	State(ctx context.Context, projectID string) (application.ScriptState, error)
	Generate(ctx context.Context, projectID, beatKey, note string) error
	Suggest(ctx context.Context, projectID, kind, lineID string) (string, error)
	Speak(ctx context.Context, projectID string, lineIDs []string) error
}

// voiceLinesUseCase is what the orchestrator calls to read lines.
type voiceLinesUseCase interface {
	ClaimPending(ctx context.Context, projectID string) ([]domain.VoiceLine, error)
	SaveResults(ctx context.Context, projectID string, results []domain.VoiceLineResult) error
	FailAll(ctx context.Context, projectID string, reason string) error
}

// WithScript enables the /script routes; voices enables the internal routes the orchestrator reads lines through.
func (rt *Router) WithScript(uc scriptUseCase, voices voiceLinesUseCase) *Router {
	rt.script, rt.voiceLines = uc, voices
	return rt
}

func (rt *Router) scriptRoutes(r chi.Router) {
	const base = "/v1/projects/{project_id}/script"
	r.Get(base, rt.handleGetScript)
	r.Post(base+"/generate", rt.handleGenerateScript)
	r.Post(base+"/suggest", rt.handleSuggestScript)
	r.Post(base+"/speak", rt.handleSpeakScript)
}

func (rt *Router) scriptEnabled(w http.ResponseWriter) bool {
	if rt.script == nil {
		writeError(w, http.StatusNotFound, "script is not enabled")
		return false
	}
	return true
}

func (rt *Router) writeScriptState(w http.ResponseWriter, r *http.Request) {
	state, err := rt.script.State(r.Context(), chi.URLParam(r, "project_id"))
	if err != nil {
		writeUnitError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, state)
}

func (rt *Router) handleGetScript(w http.ResponseWriter, r *http.Request) {
	if rt.scriptEnabled(w) {
		rt.writeScriptState(w, r)
	}
}

func (rt *Router) handleGenerateScript(w http.ResponseWriter, r *http.Request) {
	if !rt.scriptEnabled(w) {
		return
	}
	var body struct {
		Beat string `json:"beat"`
		Note string `json:"note"`
	}
	if r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
	}
	ctx, finish := rt.trackOperation(r, operationScript)
	err := rt.script.Generate(ctx, chi.URLParam(r, "project_id"), body.Beat, body.Note)
	finish(err)
	if err != nil {
		writeUnitRunError(w, err)
		return
	}
	rt.writeScriptState(w, r)
}

func (rt *Router) handleSuggestScript(w http.ResponseWriter, r *http.Request) {
	if !rt.scriptEnabled(w) {
		return
	}
	var body struct {
		Kind   string `json:"kind"`
		LineID string `json:"line_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	ctx, finish := rt.trackOperation(r, operationScript)
	text, err := rt.script.Suggest(ctx, chi.URLParam(r, "project_id"), body.Kind, body.LineID)
	finish(err)
	if err != nil {
		writeUnitRunError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"text": text})
}

func (rt *Router) handleSpeakScript(w http.ResponseWriter, r *http.Request) {
	if !rt.scriptEnabled(w) {
		return
	}
	var body struct {
		LineIDs []string `json:"line_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := rt.script.Speak(r.Context(), chi.URLParam(r, "project_id"), body.LineIDs); err != nil {
		if errors.Is(err, application.ErrVoiceNotConfigured) {
			writeError(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		writeUnitError(w, err)
		return
	}
	rt.writeScriptState(w, r)
}

func (rt *Router) voiceLinesEnabled(w http.ResponseWriter) bool {
	if rt.voiceLines == nil {
		writeError(w, http.StatusNotFound, "voice lines is not enabled")
		return false
	}
	return true
}

// voiceLineWire is a line the orchestrator is to have read.
type voiceLineWire struct {
	LineID   string `json:"line_id"`
	Text     string `json:"text"`
	Language string `json:"language"`
	VoiceID  string `json:"voice_id"`
	// Ref is handed back with the result, to tell which wording was read.
	Ref string `json:"ref"`
}

func (rt *Router) handleClaimVoiceLines(w http.ResponseWriter, r *http.Request) {
	if !rt.voiceLinesEnabled(w) {
		return
	}
	lines, err := rt.voiceLines.ClaimPending(r.Context(), chi.URLParam(r, "project_id"))
	if err != nil {
		writeUnitError(w, err)
		return
	}
	out := make([]voiceLineWire, 0, len(lines))
	for _, l := range lines {
		out = append(out, voiceLineWire{LineID: l.LineID, Text: l.Text, Language: l.Language, VoiceID: l.VoiceID, Ref: l.TextSHA})
	}
	writeJSON(w, http.StatusOK, map[string]any{"lines": out})
}

func (rt *Router) handleVoiceLineResults(w http.ResponseWriter, r *http.Request) {
	if !rt.voiceLinesEnabled(w) {
		return
	}
	var body struct {
		Lines []domain.VoiceLineResult `json:"lines"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := rt.voiceLines.SaveResults(r.Context(), chi.URLParam(r, "project_id"), body.Lines); err != nil {
		writeUnitError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (rt *Router) handleFailVoiceLines(w http.ResponseWriter, r *http.Request) {
	if !rt.voiceLinesEnabled(w) {
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Reason == "" {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := rt.voiceLines.FailAll(r.Context(), chi.URLParam(r, "project_id"), body.Reason); err != nil {
		writeUnitError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
