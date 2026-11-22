/**
 * The four CSS selectors the adapter needs on DeepSeek's page (chat.deepseek.com). Hashed class names change on every DeepSeek release, so the defaults lean on ids and the `ds-` design-system classes; the send and Copy buttons have no stable selector, so they default to blank: the prompt is sent with Enter and the answer is read from the page once its text has stopped changing.
 * Each is looked up on the whole page, and the Creator can override any of
 * them on the extension's settings page.
 */
(function (root) {
  const kit = root.AiBridgeSelectorKit || (typeof require === "function" ? require("../../adapter/selector-kit.js") : null);

  const FIELDS = [
    { key: "input", label: "Ô chat", hint: "CSS selector của ô nhập tin nhắn", value: "textarea#chat-input, textarea" },
    { key: "sendButton", label: "Nút gửi", hint: "CSS selector của nút gửi; để trống thì gửi bằng Enter", value: "" },
    { key: "answer", label: "Câu trả lời", hint: "CSS selector của mỗi câu trả lời; câu cuối cùng trên trang là câu đang chờ", value: ".ds-markdown" },
    { key: "copyButton", label: "Nút Copy", hint: "CSS selector của nút Copy; dùng nút khớp đầu tiên nằm trong hoặc sau câu trả lời mới; để trống thì chờ chữ đứng yên rồi đọc từ trang", value: "" },
  ];

  const api = kit.make("deepseekSelectors", FIELDS);
  root.AiBridgeSelectors = root.AiBridgeSelectors || {};
  root.AiBridgeSelectors.deepseek = api;
  if (typeof module !== "undefined" && module.exports) module.exports = api;
})(typeof globalThis !== "undefined" ? globalThis : self);
