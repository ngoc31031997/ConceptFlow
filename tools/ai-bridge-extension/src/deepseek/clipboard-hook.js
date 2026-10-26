/**
 * Runs in DeepSeek's own page world. DeepSeek's Copy button puts the answer's
 * original markdown on the clipboard; a background tab is not allowed to read
 * the clipboard, so instead this wraps navigator.clipboard.writeText and, only
 * while the adapter has armed it, hands the text to the adapter rather than
 * writing it. The Creator's real clipboard is never touched by a run, and
 * every copy the Creator makes by hand goes through unchanged.
 */
(function () {
  const SOURCE = "conceptflow-ai-bridge";
  const clipboard = navigator.clipboard;
  if (!clipboard || typeof clipboard.writeText !== "function") return;
  const original = clipboard.writeText.bind(clipboard);
  let armed = false;

  window.addEventListener("message", function (event) {
    if (event.source !== window) return;
    const data = event.data;
    if (data && data.source === SOURCE && data.kind === "arm-copy") armed = true;
  });

  clipboard.writeText = function (text) {
    if (!armed) return original(text);
    armed = false;
    window.postMessage({ source: SOURCE, kind: "copied", text: String(text) }, window.location.origin);
    return Promise.resolve();
  };
})();
