/**
 * The message format between the ConceptFlow page and this extension
 * (docs/contracts/ai-bridge-v2.md), plus the limits every part of the
 * extension shares. Loaded as a plain script so content scripts, the service
 * worker (importScripts) and node tests can all read it.
 */
(function (root) {
  const SOURCE = "conceptflow-ai-bridge";
  const VERSION = 2;
  const STAGES = ["preparing", "opening", "sending", "answering", "reading", "saving"];

  /** Longest a run may take before it ends with `timeout`; Search + DeepThink can think for many minutes. */
  const MAX_RUN_MS = 15 * 60 * 1000;

  /** Vietnamese text the page shows for each error code; each names the provider the item ran on. */
  const ERROR_TEXT = {
    not_logged_in: function (name, host) {
      return "Chưa đăng nhập " + name + " hoặc giao diện " + name + " đã đổi (không thấy ô chat). Mở " + host + ", đăng nhập rồi chạy lại; nếu đã đăng nhập, sửa selector ô chat ở Cài đặt của extension.";
    },
    ui_changed: function (name) {
      return "Giao diện " + name + " đã đổi, extension không tìm thấy phần cần dùng. Dán tay và sửa selector theo README của extension.";
    },
    timeout: function (name) {
      return name + " chưa trả lời xong sau 15 phút. Xem tab " + name + "; nếu đã có câu trả lời thì chép và dán tay.";
    },
    tab_closed: function (name) {
      return "Tab " + name + " đã bị đóng trước khi có câu trả lời.";
    },
    bad_request: function () {
      return "Yêu cầu gửi cho extension không hợp lệ.";
    },
    queue_bad_item: function () {
      return "Mục không hợp lệ nên không thêm được vào hàng đợi.";
    },
  };

  /** Builds a message the extension posts to the page. */
  function toPage(type, fields) {
    return Object.assign({ source: SOURCE, v: VERSION, from: "extension", type: type }, fields || {});
  }

  /**
   * The words for an error code. `provider` is the provider's table entry
   * (name, host) for the codes that name one; without it the name is generic.
   */
  function errorMessage(code, detail, provider) {
    const make = ERROR_TEXT[code] || ERROR_TEXT.bad_request;
    const base = make(provider ? provider.name : "AI ngoài", provider ? provider.host : "trang AI");
    return detail ? base + " (" + detail + ")" : base;
  }

  /** Builds an error message for the page from an error code (and an optional detail, e.g. which element was missing). */
  function errorToPage(requestId, code, detail, provider) {
    return toPage("error", { request_id: requestId, code: code, message: errorMessage(code, detail, provider) });
  }

  /** Which DeepSeek modes a run turns on; an item that does not say gets DeepThink without Search. */
  function readModes(raw) {
    const modes = raw && typeof raw === "object" ? raw : {};
    return { deepThink: modes.deepThink !== false, search: modes.search === true };
  }

  /**
   * Reads a message the page posted and returns it when it is a valid request
   * for this extension, or null for anything else (other scripts' messages,
   * the extension's own replies, another protocol version).
   */
  function parsePageMessage(data) {
    if (!data || typeof data !== "object") return null;
    if (data.source !== SOURCE || data.v !== VERSION || data.from !== "page") return null;
    if (data.type === "ping") return { type: "ping" };
    if (data.type === "cancel") {
      if (typeof data.item_id !== "string" || data.item_id === "") return null;
      return { type: "cancel", itemId: data.item_id };
    }
    if (typeof data.request_id !== "string" || data.request_id === "") return null;
    if (data.type === "queue_add") {
      if (!data.item || typeof data.item !== "object") return null;
      return { type: "queue_add", requestId: data.request_id, item: data.item };
    }
    return null;
  }

  const api = { SOURCE, VERSION, STAGES, MAX_RUN_MS, ERROR_TEXT, toPage, errorMessage, errorToPage, readModes, parsePageMessage };
  root.AiBridgeProtocol = api;
  if (typeof module !== "undefined" && module.exports) module.exports = api;
})(typeof globalThis !== "undefined" ? globalThis : self);
