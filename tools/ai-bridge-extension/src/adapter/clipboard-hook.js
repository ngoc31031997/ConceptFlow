/**
 * Runs in the provider page's own world (DeepSeek, Gemini, Grok). The page's
 * Copy button puts the answer's original markdown on the clipboard; a
 * background tab is not allowed to read the clipboard, so instead this wraps
 * navigator.clipboard.writeText (and write, for a page that copies through a
 * ClipboardItem) and, only while the adapter has armed it, hands the text to
 * the adapter rather than writing it. The Creator's real clipboard is never
 * touched by a run, and every copy the Creator makes by hand goes through
 * unchanged.
 */
(function () {
  const SOURCE = "conceptflow-ai-bridge";
  const clipboard = navigator.clipboard;
  if (!clipboard) return;
  let armed = false;

  window.addEventListener("message", function (event) {
    if (event.source !== window) return;
    const data = event.data;
    if (data && data.source === SOURCE && data.kind === "arm-copy") armed = true;
  });

  function hand(text) {
    armed = false;
    window.postMessage({ source: SOURCE, kind: "copied", text: String(text) }, window.location.origin);
  }

  if (typeof clipboard.writeText === "function") {
    const original = clipboard.writeText.bind(clipboard);
    clipboard.writeText = function (text) {
      if (!armed) return original(text);
      hand(text);
      return Promise.resolve();
    };
  }

  if (typeof clipboard.write === "function") {
    const originalWrite = clipboard.write.bind(clipboard);
    clipboard.write = async function (items) {
      if (!armed) return originalWrite(items);
      const item = items && items[0];
      if (item && typeof item.getType === "function" && item.types.indexOf("text/plain") >= 0) {
        hand(await (await item.getType("text/plain")).text());
        return undefined;
      }
      // Not a text copy: let it through and keep waiting for the text one.
      return originalWrite(items);
    };
  }
})();
