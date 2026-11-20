/**
 * Content script on the ConceptFlow page: relays the page's window messages
 * to the service worker and the worker's replies back to the page. It
 * announces itself once on load as well as on every ping, because the page
 * may have pinged before this script was injected.
 */
(function () {
  const P = globalThis.AiBridgeProtocol;
  const PR = globalThis.AiBridgeProviders;
  const version = chrome.runtime.getManifest().version;

  function post(message) {
    window.postMessage(message, window.location.origin);
  }

  async function pong() {
    let stored = {};
    try {
      stored = await chrome.storage.local.get(PR.DEFAULT_KEY);
    } catch (err) {
      console.warn("ai-bridge: could not read the default provider:", err);
    }
    post(P.toPage("pong", { providers: PR.IDS, default_provider: PR.readDefault(stored[PR.DEFAULT_KEY]), version: version }));
    chrome.runtime.sendMessage({ kind: "queue_get" }).then(function (counts) {
      if (counts) post(P.toPage("queue_state", counts));
    }).catch(function (err) {
      console.warn("ai-bridge: could not read the queue:", err);
    });
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
      void pong();
      return;
    }
    if (msg.type === "cancel") {
      sendToWorker({ kind: "page_cancel", itemId: msg.itemId });
      return;
    }
    sendToWorker({ kind: "queue_add", requestId: msg.requestId, item: msg.item }, msg.requestId);
  });

  chrome.runtime.onMessage.addListener(function (message) {
    if (message && message.kind === "to-page" && message.payload) post(message.payload);
  });

  void pong();
})();
