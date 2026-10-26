/**
 * Content script on the ConceptFlow page: relays the page's window messages
 * to the service worker and the worker's replies back to the page. It
 * announces itself once on load as well as on every ping, because the page
 * may have pinged before this script was injected.
 */
(function () {
  const P = globalThis.AiBridgeProtocol;
  const version = chrome.runtime.getManifest().version;

  function post(message) {
    window.postMessage(message, window.location.origin);
  }

  function pong() {
    post(P.toPage("pong", { providers: P.PROVIDERS, version: version }));
  }

  function sendToWorker(message, requestId) {
    chrome.runtime.sendMessage(message).catch(function (err) {
      if (requestId) post(P.errorToPage(requestId, "bad_request", String(err && err.message ? err.message : err)));
    });
  }

  window.addEventListener("message", function (event) {
    if (event.source !== window || event.origin !== window.location.origin) return;
    const msg = P.parsePageMessage(event.data);
    if (!msg) return;
    if (msg.type === "ping") {
      pong();
      return;
    }
    if (msg.type === "cancel") {
      sendToWorker({ kind: "cancel", requestId: msg.requestId });
      return;
    }
    sendToWorker({ kind: "run", requestId: msg.requestId, provider: msg.provider, prompt: msg.prompt, modes: msg.modes }, msg.requestId);
  });

  chrome.runtime.onMessage.addListener(function (message) {
    if (message && message.kind === "to-page" && message.payload) post(message.payload);
  });

  pong();
})();
