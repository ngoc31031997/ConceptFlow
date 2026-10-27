/**
 * Service worker: opens a DeepSeek window of its own (unfocused) for a run, hands
 * the prompt to the DeepSeek adapter once its tab is ready, forwards the
 * adapter's progress, answer or error to the ConceptFlow tab that asked, and
 * closes that window when the run ends or is cancelled.
 * One run at a time; the run is kept in chrome.storage.session (see jobs.js).
 */
importScripts("protocol.js", "jobs.js");

const P = self.AiBridgeProtocol;
const J = self.AiBridgeJobs;
const JOB_KEY = "job";
const DEEPSEEK_URL = "https://chat.deepseek.com/";

async function readJob() {
  const stored = await chrome.storage.session.get(JOB_KEY);
  return stored[JOB_KEY] || null;
}

async function writeJob(job) {
  await chrome.storage.session.set({ [JOB_KEY]: job });
}

async function clearJob() {
  await chrome.storage.session.remove(JOB_KEY);
}

function toPage(tabId, payload) {
  return chrome.tabs.sendMessage(tabId, { kind: "to-page", payload: payload }).catch(function (err) {
    console.warn("ai-bridge: ConceptFlow tab " + tabId + " did not take the message:", err);
  });
}

async function startRun(message, replyTabId) {
  const now = Date.now();
  const current = await readJob();
  if (J.isActive(current, now)) {
    await toPage(replyTabId, P.errorToPage(message.requestId, "busy"));
    return;
  }
  const job = J.newJob(message.requestId, message.prompt, message.modes || P.readModes(null), replyTabId, now);
  await writeJob(job);
  await toPage(replyTabId, P.toPage("progress", { request_id: job.requestId, stage: "opening" }));
  const win = await chrome.windows.create({ url: DEEPSEEK_URL, focused: false });
  job.providerWindowId = win.id;
  job.providerTabId = win.tabs[0].id;
  await writeJob(job);
}

/** Closes the run's DeepSeek window; it may already be gone (closed by hand). */
function closeProviderWindow(job) {
  if (!job || job.providerWindowId === null || job.providerWindowId === undefined) return;
  chrome.windows.remove(job.providerWindowId).catch(function (err) {
    console.warn("ai-bridge: DeepSeek window " + job.providerWindowId + " was not closed:", err);
  });
}

/** The adapter asks for its work when its page has loaded; it gets the prompt once. */
async function handOutWork(tabId) {
  const job = await readJob();
  if (!job || job.providerTabId !== tabId || job.started) return null;
  job.started = true;
  await writeJob(job);
  return { requestId: job.requestId, prompt: job.prompt, modes: job.modes, deadline: job.startedAt + P.MAX_RUN_MS };
}

async function forwardFromAdapter(message, tabId) {
  const job = await readJob();
  if (!J.belongsTo(job, tabId, message.requestId)) return;
  if (message.kind === "progress") {
    await toPage(job.replyTabId, P.toPage("progress", { request_id: job.requestId, stage: message.stage }));
    return;
  }
  await clearJob();
  closeProviderWindow(job);
  if (message.kind === "result") {
    await toPage(job.replyTabId, P.toPage("result", { request_id: job.requestId, text: message.text }));
  } else {
    await toPage(job.replyTabId, P.errorToPage(job.requestId, message.code, message.detail));
  }
}

async function cancelRun(requestId) {
  const job = await readJob();
  if (!job || job.requestId !== requestId) return;
  await clearJob();
  closeProviderWindow(job);
}

chrome.runtime.onMessage.addListener(function (message, sender, sendResponse) {
  const tabId = sender.tab ? sender.tab.id : null;
  if (!message || tabId === null) return false;
  if (message.kind === "ready") {
    handOutWork(tabId).then(sendResponse, function (err) {
      console.error("ai-bridge: could not hand out the run:", err);
      sendResponse(null);
    });
    return true;
  }
  let work;
  if (message.kind === "run") work = startRun(message, tabId);
  else if (message.kind === "cancel") work = cancelRun(message.requestId);
  else if (message.kind === "progress" || message.kind === "result" || message.kind === "error") work = forwardFromAdapter(message, tabId);
  else return false;
  work.catch(function (err) {
    console.error("ai-bridge: " + message.kind + " failed:", err);
    if (message.kind === "run") toPage(tabId, P.errorToPage(message.requestId, "bad_request", String(err && err.message ? err.message : err)));
  });
  return false;
});

chrome.tabs.onRemoved.addListener(function (tabId) {
  readJob().then(async function (job) {
    if (!job) return;
    if (job.providerTabId === tabId) {
      await clearJob();
      await toPage(job.replyTabId, P.errorToPage(job.requestId, "tab_closed"));
    } else if (job.replyTabId === tabId) {
      await clearJob();
      closeProviderWindow(job);
    }
  }).catch(function (err) {
    console.error("ai-bridge: could not settle the run of closed tab " + tabId + ":", err);
  });
});

// The toolbar icon opens the selector settings page.
chrome.action.onClicked.addListener(function () {
  chrome.runtime.openOptionsPage();
});
