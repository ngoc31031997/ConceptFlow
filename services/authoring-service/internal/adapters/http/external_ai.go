package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"authoring/internal/application"
)

const (
	// externalAIHeader marks a request that runs its AI step through an AI
	// outside the system: "prompt" to get the prompt, "paste" to hand back the reply.
	externalAIHeader = "X-External-AI"
	// maxExternalReplyBytes caps the body of a paste request.
	maxExternalReplyBytes = 4 << 20
)

// externalAIResponse is the answer to a "prompt" request: the prompt to paste
// into the other AI.
type externalAIResponse struct {
	Prompt string `json:"external_prompt"`
}

// externalAI lets any step run through an AI outside the system without the
// step knowing: it puts the request's mode in the context, where the provider
// reads it. The handler runs as it always does.
//
// In "prompt" mode the handler's own answer is held back; when the flow built
// a prompt the answer is that prompt, otherwise the held answer (an error such
// as a missing project) goes out unchanged. In "paste" mode the body's
// "external_reply" is the model's reply and the rest of the body is the step's
// own request.
func externalAI(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mode := application.ExternalAIMode(r.Header.Get(externalAIHeader))
		switch mode {
		case "":
			next.ServeHTTP(w, r)
		case application.ExternalAIPrompt:
			servePrompt(next, w, r)
		case application.ExternalAIPaste:
			servePaste(next, w, r)
		default:
			writeError(w, http.StatusBadRequest, "X-External-AI phải là prompt hoặc paste")
		}
	})
}

func servePrompt(next http.Handler, w http.ResponseWriter, r *http.Request) {
	run := application.NewExternalAIRun(application.ExternalAIPrompt, "")
	held := &heldResponse{header: http.Header{}, status: http.StatusOK}
	next.ServeHTTP(held, r.WithContext(application.WithExternalAI(r.Context(), run)))
	if prompt, ok := run.Prompt(); ok {
		writeJSON(w, http.StatusOK, externalAIResponse{Prompt: prompt})
		return
	}
	held.release(w)
}

func servePaste(next http.Handler, w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxExternalReplyBytes))
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeError(w, http.StatusRequestEntityTooLarge, "kết quả dán quá dài")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	var in struct {
		Reply string `json:"external_reply"`
	}
	if err := json.Unmarshal(body, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	reply := strings.TrimSpace(in.Reply)
	if reply == "" {
		writeError(w, http.StatusBadRequest, "Dán kết quả của AI ngoài trước khi đọc.")
		return
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	run := application.NewExternalAIRun(application.ExternalAIPaste, reply)
	next.ServeHTTP(w, r.WithContext(application.WithExternalAI(r.Context(), run)))
}

// heldResponse keeps a handler's answer until the caller decides to send it.
type heldResponse struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func (h *heldResponse) Header() http.Header         { return h.header }
func (h *heldResponse) WriteHeader(status int)      { h.status = status }
func (h *heldResponse) Write(b []byte) (int, error) { return h.body.Write(b) }

// release sends the held answer to w.
func (h *heldResponse) release(w http.ResponseWriter) {
	for k, v := range h.header {
		w.Header()[k] = v
	}
	w.WriteHeader(h.status)
	_, _ = w.Write(h.body.Bytes())
}
