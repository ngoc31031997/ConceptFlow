/**
 * Everything the adapter knows about DeepSeek's page. Each entry has a default
 * here; the Creator can override any of them on the extension's settings page
 * (copied from DevTools, F12) without editing code. Hashed class names change
 * on every DeepSeek release, so the defaults lean on ids, roles, the `ds-`
 * design-system classes and visible labels.
 *
 * All values are strings, as the settings page stores them; `resolve` turns
 * them into what the adapter uses (label lists, a RegExp) and falls back to
 * the default for an empty or invalid value.
 */
(function (root) {
  /** chrome.storage.sync key holding the Creator's overrides. */
  const STORAGE_KEY = "deepseekSelectors";

  const FIELDS = [
    { key: "input", label: "Ô chat", hint: "CSS selector của ô nhập tin nhắn", value: "textarea#chat-input, textarea" },
    {
      key: "sendButton",
      label: "Nút gửi",
      hint: "CSS selector của nút mũi tên gửi. Để trống: nút bấm cuối cùng trong khung chat",
      value: "",
    },
    {
      key: "deepThinkButton",
      label: "Nút DeepThink",
      hint: "CSS selector. Để trống: tìm theo chữ ở ô dưới",
      value: "",
    },
    { key: "deepThinkLabels", label: "Chữ trên nút DeepThink", hint: "Ngăn cách bằng dấu phẩy", value: "DeepThink, 深度思考" },
    { key: "searchButton", label: "Nút Search", hint: "CSS selector. Để trống: tìm theo chữ ở ô dưới", value: "" },
    { key: "searchLabels", label: "Chữ trên nút Search", hint: "Ngăn cách bằng dấu phẩy", value: "Search, 联网搜索" },
    {
      key: "selectedClass",
      label: "Class khi nút đang bật",
      hint: "Biểu thức chính quy so với class của nút DeepThink/Search khi đang bật",
      value: "selected|active|checked",
    },
    { key: "markdown", label: "Khối câu trả lời", hint: "CSS selector của khối markdown câu trả lời", value: ".ds-markdown" },
    { key: "thinking", label: "Khung suy nghĩ DeepThink", hint: "CSS selector của khung bao phần suy nghĩ (bị bỏ khỏi câu trả lời)", value: '[class*="think"]' },
    {
      key: "iconButton",
      label: "Nút dưới câu trả lời",
      hint: "CSS selector các nút icon dưới câu trả lời đã xong; nút đầu tiên là Copy",
      value: '.ds-icon-button, [class*="icon-button"]',
    },
  ];

  /** Controls searched when a toggle is found by its label. */
  const CLICKABLE = 'button, [role="button"], [class*="button"]';
  /** URL path of the sign-in page DeepSeek redirects to. */
  const SIGN_IN_PATH = /sign_in|login/i;

  function defaults() {
    const out = {};
    FIELDS.forEach(function (f) {
      out[f.key] = f.value;
    });
    return out;
  }

  function labels(text) {
    return String(text)
      .split(",")
      .map(function (s) {
        return s.trim();
      })
      .filter(Boolean);
  }

  /**
   * The selectors the adapter uses: each override that is a non-blank string
   * replaces its default. `invalid` lists the keys whose override was not
   * usable (a broken regular expression) and fell back to the default.
   */
  function resolve(overrides) {
    const raw = defaults();
    const given = overrides && typeof overrides === "object" ? overrides : {};
    FIELDS.forEach(function (f) {
      const v = given[f.key];
      if (typeof v === "string" && v.trim() !== "") raw[f.key] = v.trim();
    });
    const invalid = [];
    let selected;
    try {
      selected = new RegExp(raw.selectedClass, "i");
    } catch (err) {
      invalid.push("selectedClass");
      selected = new RegExp(defaults().selectedClass, "i");
    }
    return {
      input: raw.input,
      sendButton: raw.sendButton,
      deepThinkButton: raw.deepThinkButton,
      deepThinkLabels: labels(raw.deepThinkLabels),
      searchButton: raw.searchButton,
      searchLabels: labels(raw.searchLabels),
      selectedClass: selected,
      markdown: raw.markdown,
      thinking: raw.thinking,
      iconButton: raw.iconButton,
      clickable: CLICKABLE,
      signInPath: SIGN_IN_PATH,
      invalid: invalid,
    };
  }

  const api = { STORAGE_KEY, FIELDS, defaults, resolve };
  root.DeepSeekSelectors = api;
  if (typeof module !== "undefined" && module.exports) module.exports = api;
})(typeof globalThis !== "undefined" ? globalThis : self);
