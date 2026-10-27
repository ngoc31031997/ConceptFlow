/**
 * The one run the extension is doing, as it is kept in chrome.storage.session.
 * The service worker may be stopped and restarted while DeepSeek is still
 * answering, so every step reads the job back from storage instead of
 * holding it in memory. These helpers are pure so node tests can check them.
 */
(function (root) {
  const P = root.AiBridgeProtocol;

  /** Extra time past MAX_RUN_MS after which a job left behind (a crashed adapter) no longer blocks a new run. */
  const STALE_GRACE_MS = 60 * 1000;

  /** A new job for a run the page asked for; `providerTabId` and `providerWindowId` are filled once the window exists. */
  function newJob(requestId, prompt, modes, replyTabId, now) {
    return { requestId: requestId, prompt: prompt, modes: modes, replyTabId: replyTabId, providerTabId: null, providerWindowId: null, started: false, startedAt: now };
  }

  /** True when a stored job still blocks a new run. */
  function isActive(job, now) {
    return !!job && now - job.startedAt < P.MAX_RUN_MS + STALE_GRACE_MS;
  }

  /** True when this message from a DeepSeek tab belongs to the stored job. */
  function belongsTo(job, tabId, requestId) {
    return !!job && job.providerTabId === tabId && job.requestId === requestId;
  }

  const api = { STALE_GRACE_MS, newJob, isActive, belongsTo };
  root.AiBridgeJobs = api;
  if (typeof module !== "undefined" && module.exports) module.exports = api;
})(typeof globalThis !== "undefined" ? globalThis : self);
