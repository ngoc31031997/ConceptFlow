/**
 * The message format between the ConceptFlow page and this extension
 * (docs/contracts/ai-bridge-v1.md), plus the limits every part of the
 * extension shares. Loaded as a plain script so content scripts, the service
 * worker (importScripts) and node tests can all read it.
 */
(function (root) {
  const SOURCE = "conceptflow-ai-bridge";
  const VERSION = 1;
  const PROVIDERS = ["deepseek"];
  const STAGES = ["opening", "sending", "answering", "reading"];

  /** Longest a run may take before it ends with `timeout`; Search + DeepThink can think for many minutes. */
  const MAX_RUN_MS = 15 * 60 * 1000;

  /** Vietnamese text the page shows for each error code. */
  const ERROR_TEXT = {
    not_logged_in: "Chưa đăng nhập DeepSeek. Mở chat.deepseek.com, đăng nhập rồi chạy lại.",
    ui_changed: "Giao diện DeepSeek đã đổi, extension không tìm thấy phần cần dùng. Dán tay và sửa selector theo README của extension.",
    timeout: "DeepSeek chưa trả lời xong sau 15 phút. Xem tab DeepSeek; nếu đã có câu trả lời thì chép và dán tay.",
    tab_closed: "Tab DeepSeek đã bị đóng trước khi có câu trả lời.",
    busy: "Đang có một lượt chạy DeepSeek khác. Chờ lượt đó xong hoặc huỷ nó.",
    bad_request: "Yêu cầu gửi cho extension không hợp lệ.",
    queue_bad_item: "Mục không hợp lệ nên không thêm được vào hàng đợi.",
  };

  /** Builds a message the extension posts to the page. */
  function toPage(type, fields) {
    return Object.assign({ source: SOURCE, v: VERSION, from: "extension", type: type }, fields || {});
  }

  /** Builds an error message for the page from an error code (and an optional detail, e.g. which element was missing). */
  function errorToPage(requestId, code, detail) {
    const base = ERROR_TEXT[code] || ERROR_TEXT.bad_request;
    return toPage("error", { request_id: requestId, code: code, message: detail ? base + " (" + detail + ")" : base });
  }

  /** Which DeepSeek modes a run turns on; a run that does not say turns both on. */
  function readModes(raw) {
    const modes = raw && typeof raw === "object" ? raw : {};
    return { deepThink: modes.deep_think !== false, search: modes.search !== false };
  }

  /**
   * Reads a message the page posted and returns it when it is a valid request
   * for this extension, or null for anything else (other scripts' messages,
   * the extension's own replies, a run without a prompt).
   */
  function parsePageMessage(data) {
    if (!data || typeof data !== "object") return null;
    if (data.source !== SOURCE || data.v !== VERSION || data.from !== "page") return null;
    if (data.type === "ping") return { type: "ping" };
    if (typeof data.request_id !== "string" || data.request_id === "") return null;
    if (data.type === "cancel") return { type: "cancel", requestId: data.request_id };
    if (data.type === "queue_add") {
      if (!data.item || typeof data.item !== "object") return null;
      return { type: "queue_add", requestId: data.request_id, item: data.item };
    }
    if (data.type !== "run") return null;
    if (PROVIDERS.indexOf(data.provider) < 0) return null;
    if (typeof data.prompt !== "string" || data.prompt.trim() === "") return null;
    return { type: "run", requestId: data.request_id, provider: data.provider, prompt: data.prompt, modes: readModes(data.modes) };
  }

  const api = { SOURCE, VERSION, PROVIDERS, STAGES, MAX_RUN_MS, ERROR_TEXT, toPage, errorToPage, readModes, parsePageMessage };
  root.AiBridgeProtocol = api;
  if (typeof module !== "undefined" && module.exports) module.exports = api;
})(typeof globalThis !== "undefined" ? globalThis : self);
