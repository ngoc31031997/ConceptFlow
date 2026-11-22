/**
 * The four CSS selectors the adapter needs on Gemini's page (gemini.google.com/app). Defaults are taken from the page's own tags and test ids.
 * Each is looked up on the whole page, and the Creator can override any of
 * them on the extension's settings page.
 */
(function (root) {
  const kit = root.AiBridgeSelectorKit || (typeof require === "function" ? require("../../adapter/selector-kit.js") : null);

  const FIELDS = [
    { key: "input", label: "Ô chat", hint: "CSS selector của ô nhập tin nhắn", value: "rich-textarea .ql-editor[contenteditable=\"true\"]" },
    { key: "sendButton", label: "Nút gửi", hint: "CSS selector của nút gửi; để trống thì gửi bằng Enter", value: "[data-test-id=\"send-button-container\"] .send-button.submit button" },
    { key: "answer", label: "Câu trả lời", hint: "CSS selector của mỗi câu trả lời; câu cuối cùng trên trang là câu đang chờ", value: "model-response" },
    { key: "copyButton", label: "Nút Copy", hint: "CSS selector của nút Copy; dùng nút khớp đầu tiên nằm trong hoặc sau câu trả lời mới; để trống thì chờ chữ đứng yên rồi đọc từ trang", value: "message-actions copy-button button" },
  ];

  const api = kit.make("geminiSelectors", FIELDS);
  root.AiBridgeSelectors = root.AiBridgeSelectors || {};
  root.AiBridgeSelectors.gemini = api;
  if (typeof module !== "undefined" && module.exports) module.exports = api;
})(typeof globalThis !== "undefined" ? globalThis : self);
