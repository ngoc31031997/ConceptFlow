/**
 * The two requests the worker makes around a run on DeepSeek, for one queue
 * item (docs/contracts/external-ai-v1.md): the item's request with
 * `X-External-AI: prompt`, which answers with the prompt built from the project
 * as it is now, and the same request with `X-External-AI: paste` and the
 * DeepSeek answer as `external_reply`, which has the step read and store it.
 * `fetchImpl` is injected so node tests can fake the gateway.
 */
(function (root) {
  const HEADER = "X-External-AI";

  const NO_PROMPT = "Mục này không trả prompt để chạy trên DeepSeek.";
  const NO_CONNECTION = "Không kết nối được máy chủ ConceptFlow. Kiểm tra các dịch vụ đang chạy rồi thử lại.";

  /** The words the server gave for a failed request, else a generic line with the status. */
  async function errorMessage(response) {
    try {
      const body = await response.json();
      if (typeof body.error === "string" && body.error) return body.error;
      if (body.error && typeof body.error.message === "string") return body.error.message;
      if (typeof body.error_message === "string" && body.error_message) return body.error_message;
    } catch (err) {
      // The body is not JSON; the status line below says what failed.
    }
    return "Máy chủ trả lỗi " + response.status + ".";
  }

  /**
   * Makes the item's request in an external-AI mode. Resolves to `{ok: true, prompt}` for
   * "prompt" and `{ok: true}` for "paste", or `{ok: false, message}` with the server's own words.
   */
  async function callExternal(fetchImpl, request, mode, reply) {
    let body = request.body;
    if (mode === "paste") {
      let fields;
      try {
        fields = request.body ? JSON.parse(request.body) : {};
      } catch (err) {
        return { ok: false, message: "Nội dung yêu cầu đã lưu không đọc được." };
      }
      body = JSON.stringify(Object.assign({}, fields, { external_reply: reply }));
    }
    let response;
    try {
      response = await fetchImpl(request.gateway + request.path, {
        method: request.method,
        headers: { "Content-Type": "application/json", [HEADER]: mode },
        body: body,
      });
    } catch (err) {
      return { ok: false, message: NO_CONNECTION };
    }
    if (!response.ok) return { ok: false, message: await errorMessage(response) };
    if (mode !== "prompt") return { ok: true };
    let prompt;
    try {
      prompt = (await response.json()).external_prompt;
    } catch (err) {
      prompt = "";
    }
    if (typeof prompt !== "string" || prompt.trim() === "") return { ok: false, message: NO_PROMPT };
    return { ok: true, prompt: prompt };
  }

  const api = { HEADER, NO_PROMPT, NO_CONNECTION, callExternal };
  root.AiBridgeQueueHttp = api;
  if (typeof module !== "undefined" && module.exports) module.exports = api;
})(typeof globalThis !== "undefined" ? globalThis : self);
