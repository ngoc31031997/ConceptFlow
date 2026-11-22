/**
 * The four CSS selectors the adapter needs on Grok's page (grok.com). Defaults are taken from the page's test ids; the Copy button is the first button of the toolbar under the answer, so it does not depend on the interface language.
 * Each is looked up on the whole page, and the Creator can override any of
 * them on the extension's settings page.
 */
(function (root) {
  const kit = root.AiBridgeSelectorKit || (typeof require === "function" ? require("../../adapter/selector-kit.js") : null);

  const FIELDS = [
    { key: "input", label: "Ô chat", hint: "CSS selector của ô nhập tin nhắn", value: "[data-testid=\"chat-input\"] .ProseMirror[contenteditable=\"true\"]" },
    { key: "sendButton", label: "Nút gửi", hint: "CSS selector của nút gửi; để trống thì gửi bằng Enter", value: "button[data-testid=\"chat-submit\"]" },
    { key: "answer", label: "Câu trả lời", hint: "CSS selector của mỗi câu trả lời; câu cuối cùng trên trang là câu đang chờ", value: "[data-testid=\"assistant-message\"]" },
    { key: "copyButton", label: "Nút Copy", hint: "CSS selector của nút Copy; dùng nút khớp đầu tiên nằm trong hoặc sau câu trả lời mới; để trống thì chờ chữ đứng yên rồi đọc từ trang", value: "[data-testid=\"response-footer-actions\"] > div:first-of-type > button:first-of-type" },
  ];

  const api = kit.make("grokSelectors", FIELDS);
  root.AiBridgeSelectors = root.AiBridgeSelectors || {};
  root.AiBridgeSelectors.grok = api;
  if (typeof module !== "undefined" && module.exports) module.exports = api;
})(typeof globalThis !== "undefined" ? globalThis : self);
