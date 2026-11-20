/**
 * Everything the adapter knows about Gemini's page (gemini.google.com/app).
 * Each entry has a default taken from the page's own tags and test ids, and
 * the Creator can override any of them on the extension's settings page.
 */
(function (root) {
  const kit = root.AiBridgeSelectorKit || (typeof require === "function" ? require("../../adapter/selector-kit.js") : null);

  const FIELDS = [
    { key: "input", label: "Ô chat", hint: "CSS selector của ô nhập tin nhắn", value: 'rich-textarea .ql-editor[contenteditable="true"]' },
    { key: "sendButton", label: "Nút gửi", hint: "CSS selector của nút gửi", value: '[data-test-id="send-button-container"] .send-button.submit button' },
    { key: "answer", label: "Lượt trả lời", hint: "CSS selector của mỗi lượt trả lời; lượt cuối cùng là câu trả lời đang chờ", value: "model-response" },
    { key: "doneMarker", label: "Dấu hiệu đã xong", hint: "CSS selector, tìm trong lượt trả lời cuối; có là câu trả lời đã chạy xong", value: ".response-footer.complete" },
    { key: "settledMarker", label: "Dấu hiệu nội dung đã ổn định", hint: "CSS selector tìm trong lượt trả lời cuối, cũng phải có; để trống nếu không dùng", value: 'message-content [aria-busy="false"]' },
    { key: "copyButton", label: "Nút Copy", hint: "CSS selector của nút Copy, tìm trong lượt trả lời cuối", value: "message-actions copy-button button" },
  ];

  const api = kit.make("geminiSelectors", FIELDS);
  root.AiBridgeSelectors = root.AiBridgeSelectors || {};
  root.AiBridgeSelectors.gemini = api;
  if (typeof module !== "undefined" && module.exports) module.exports = api;
})(typeof globalThis !== "undefined" ? globalThis : self);
