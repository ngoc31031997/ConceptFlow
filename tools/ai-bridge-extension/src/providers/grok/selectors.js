/**
 * Everything the adapter knows about Grok's page (grok.com). Each entry has a
 * default taken from the page's own test ids, and the Creator can override any
 * of them on the extension's settings page. The Copy button is found by its
 * English label "Copy response", the only selector that depends on the
 * language of Grok's interface.
 */
(function (root) {
  const kit = root.AiBridgeSelectorKit || (typeof require === "function" ? require("../../adapter/selector-kit.js") : null);

  const FIELDS = [
    { key: "input", label: "Ô chat", hint: "CSS selector của ô nhập tin nhắn", value: '[data-testid="chat-input"] .ProseMirror[contenteditable="true"]' },
    { key: "sendButton", label: "Nút gửi", hint: "CSS selector của nút gửi", value: 'button[data-testid="chat-submit"]' },
    { key: "answer", label: "Lượt trả lời", hint: "CSS selector của mỗi lượt trả lời; lượt cuối cùng là câu trả lời đang chờ", value: '[data-testid="assistant-message"]' },
    { key: "doneMarker", label: "Dấu hiệu đã xong", hint: "CSS selector, tìm trong lượt trả lời cuối; có là câu trả lời đã chạy xong", value: '[data-testid="response-footer-actions"]' },
    { key: "settledMarker", label: "Dấu hiệu nội dung đã ổn định", hint: "CSS selector tìm trong lượt trả lời cuối, cũng phải có; để trống nếu không dùng", value: "" },
    { key: "copyButton", label: "Nút Copy", hint: "CSS selector của nút Copy, tìm trong lượt trả lời cuối. Nhãn \"Copy response\" là tiếng Anh: đổi nếu giao diện Grok không để tiếng Anh", value: '[data-testid="response-footer-actions"] button[aria-label="Copy response"]' },
  ];

  const api = kit.make("grokSelectors", FIELDS);
  root.AiBridgeSelectors = root.AiBridgeSelectors || {};
  root.AiBridgeSelectors.grok = api;
  if (typeof module !== "undefined" && module.exports) module.exports = api;
})(typeof globalThis !== "undefined" ? globalThis : self);
