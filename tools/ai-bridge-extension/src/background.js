/**
 * Service worker: runs the AI queue (queue.js, kept in chrome.storage.local).
 * Each item in turn gets its prompt from the API (or carries it), opens a
 * chat tab of its provider (DeepSeek, Gemini or Grok), hands the prompt to
 * that provider's adapter once the tab is ready, and has the API read the
 * answer the adapter brings back, with no ConceptFlow tab needed. One run at
 * a time; the run is kept in chrome.storage.session (see jobs.js). A failed
 * item stops the queue. Every change to an item is announced to all open
 * ConceptFlow tabs as `queue_item`, so a screen can follow the item it queued.
 */
importScripts("providers.js", "protocol.js", "jobs.js", "queue.js", "queue-http.js");

const P = self.AiBridgeProtocol;
const PR = self.AiBridgeProviders;
const J = self.AiBridgeJobs;
const Q = self.AiBridgeQueue;
const H = self.AiBridgeQueueHttp;
const JOB_KEY = "job";
const QUEUE_KEY = "queue";
const CONCEPTFLOW_URLS = ["http://localhost:3000/*", "http://127.0.0.1:3000/*"];

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

/** The provider an item without its own choice runs on. */
async function readDefaultProvider() {
  const stored = await chrome.storage.local.get(PR.DEFAULT_KEY);
  return PR.readDefault(stored[PR.DEFAULT_KEY]);
}

function toPage(tabId, payload) {
  return chrome.tabs.sendMessage(tabId, { kind: "to-page", payload: payload }).catch(function (err) {
    console.warn("ai-bridge: ConceptFlow tab " + tabId + " did not take the message:", err);
  });
}

/** Sends a message to every open ConceptFlow tab. */
async function toAllPages(payload) {
  const tabs = await chrome.tabs.query({ url: CONCEPTFLOW_URLS });
  await Promise.all(tabs.map(function (tab) { return toPage(tab.id, payload); }));
}

// ---- queue state -------------------------------------------------------

let queueChain = Promise.resolve();

/**
 * Reads the queue, applies `change` to it and stores the result. Changes run
 * one after another, so two events arriving together do not overwrite each other.
 */
function changeQueue(change) {
  const run = queueChain.then(async function () {
    const stored = await chrome.storage.local.get(QUEUE_KEY);
    const before = Q.readState(stored[QUEUE_KEY]);
    const after = change(before);
    if (after !== before) await chrome.storage.local.set({ [QUEUE_KEY]: after });
    return after;
  });
  queueChain = run.catch(function (err) {
    console.error("ai-bridge: could not change the queue:", err);
  });
  return run;
}

async function readQueue() {
  const stored = await chrome.storage.local.get(QUEUE_KEY);
  return Q.readState(stored[QUEUE_KEY]);
}

async function broadcastQueue() {
  await toAllPages(P.toPage("queue_state", Q.counts(await readQueue())));
}

/** Tells every ConceptFlow tab where an item stands now. */
async function announce(itemId) {
  const item = Q.find(await readQueue(), itemId);
  if (item) await toAllPages(P.toPage("queue_item", Q.describe(item, await readDefaultProvider())));
}

/** Tells every ConceptFlow tab that an item left the queue without finishing. */
async function announceRemoved(item) {
  const gone = Object.assign({}, item, { status: "failed", stage: "" });
  await toAllPages(P.toPage("queue_item", Object.assign(Q.describe(gone, await readDefaultProvider()), { status: "cancelled", error: "", reply: "" })));
}

function newItemId() {
  return "q" + Date.now().toString(36) + Math.random().toString(36).slice(2, 8);
}

async function addToQueue(message, replyTabId) {
  let result = { item: null, added: false };
  await changeQueue(function (state) {
    result = Q.add(state, message.item, newItemId(), Date.now());
    return result.state;
  });
  if (!result.item) {
    await toPage(replyTabId, P.errorToPage(message.requestId, "queue_bad_item"));
    return;
  }
  await toPage(replyTabId, P.toPage("queue_added", { request_id: message.requestId, item_id: result.item.id, added: result.added }));
  await announce(result.item.id);
  await broadcastQueue();
  await pump();
}

async function setStage(itemId, stage) {
  await changeQueue(function (state) { return Q.setStage(state, itemId, stage); });
  await announce(itemId);
}

async function failItem(itemId, error, reply) {
  await changeQueue(function (state) { return Q.fail(state, itemId, error, reply); });
  await announce(itemId);
  await broadcastQueue();
}

async function finishItem(item) {
  await changeQueue(function (state) { return Q.finish(state, item.id); });
  await announce(item.id);
  await broadcastQueue();
  await pump();
}

/** The words for a run that ended in an error code, as the page would show them, naming the provider the run used. */
function errorText(code, detail, providerId) {
  return P.errorMessage(code, detail, PR.find(providerId));
}

/** Starts the next waiting item when nothing else is running. */
async function pump() {
  const job = await readJob();
  if (J.isActive(job, Date.now())) return;
  const defaultId = await readDefaultProvider();
  let picked = null;
  let providerId = "";
  await changeQueue(function (state) {
    picked = Q.next(state);
    if (!picked) return state;
    providerId = Q.providerOf(picked, defaultId);
    return Q.start(state, picked.id, providerId);
  });
  if (!picked) return;
  await announce(picked.id);
  await broadcastQueue();
  await runItem(picked, providerId);
}

/** Gets the item's prompt (from the API as the project is now, or its own), then runs it on its provider. */
async function runItem(item, providerId) {
  let prompt = item.prompt;
  if (item.kind !== "prompt") {
    const made = await H.callExternal(fetch.bind(self), item.request, "prompt");
    if (!made.ok) {
      await failItem(item.id, made.message);
      return;
    }
    prompt = made.prompt;
  }
  const job = J.newJob(item.id, providerId, prompt, item.modes, Date.now());
  await writeJob(job);
  await setStage(item.id, "opening");
  await broadcastQueue();
  // The tab is shown, not hidden: Chrome slows a hidden tab's timers and page updates, which delays reading the answer.
  const tab = await chrome.tabs.create({ url: PR.find(providerId).url, active: true });
  job.providerTabId = tab.id;
  await writeJob(job);
}

/** The adapter's answer for a queue item: the API reads it, or the item fails with the answer kept. */
async function settleQueueRun(job, message) {
  const item = Q.find(await readQueue(), job.itemId);
  if (!item) return;
  if (message.kind === "progress") {
    await setStage(item.id, message.stage);
    return;
  }
  await clearJob();
  if (message.kind === "error") {
    await failItem(item.id, errorText(message.code, message.detail, job.provider));
    await pump();
    return;
  }
  await closeProviderTab(job.providerTabId, job.provider);
  await setStage(item.id, "saving");
  const saved = await H.callExternal(fetch.bind(self), item.request, item.kind === "prompt" ? "raw" : "paste", message.text);
  if (!saved.ok) {
    await failItem(item.id, saved.message, message.text);
    return;
  }
  await finishItem(item);
}

/** The adapter asks for its work when its page has loaded; it gets the prompt once. */
async function handOutWork(tabId) {
  const job = await readJob();
  if (!job || job.providerTabId !== tabId || job.started) return null;
  job.started = true;
  await writeJob(job);
  return { requestId: job.requestId, prompt: job.prompt, modes: job.modes, provider: job.provider, deadline: job.startedAt + P.MAX_RUN_MS };
}

/** Closes the provider tab of a run whose answer was delivered or that was stopped; the job is cleared first, so this is not reported as an error. */
async function closeProviderTab(tabId, providerId) {
  try {
    await chrome.tabs.remove(tabId);
  } catch (err) {
    console.warn("ai-bridge: could not close the " + PR.nameOf(providerId) + " tab " + tabId + ":", err);
  }
}

async function forwardFromAdapter(message, tabId) {
  const job = await readJob();
  if (!J.belongsTo(job, tabId, message.requestId)) return;
  await settleQueueRun(job, message);
}

/** Stops the run of a queue item, if it is the stored one: the adapter is told to cancel and its tab closes. */
async function stopRun(itemId) {
  const job = await readJob();
  if (!job || job.itemId !== itemId) return;
  await clearJob();
  if (job.providerTabId === null) return;
  chrome.tabs.sendMessage(job.providerTabId, { kind: "cancel", requestId: job.requestId }).catch(function (err) {
    console.warn("ai-bridge: " + PR.nameOf(job.provider) + " tab did not take the cancel:", err);
  });
  await closeProviderTab(job.providerTabId, job.provider);
}

// ---- commands from the popup and the page ---------------------------------

/** Stops the run of a queue item; the item fails as cancelled, so the queue waits for the Creator. */
async function cancelItem(itemId) {
  await stopRun(itemId);
  await failItem(itemId, "Đã huỷ.");
}

/** The page cancels the item it queued: its run stops and the item leaves the queue. */
async function cancelFromPage(itemId) {
  const item = Q.find(await readQueue(), itemId);
  if (!item) return;
  if (item.status === "running") {
    await stopRun(itemId);
    await changeQueue(function (state) { return Q.fail(state, itemId, "Đã huỷ."); });
  }
  await changeQueue(function (state) { return Q.drop(state, itemId); });
  await announceRemoved(item);
  await broadcastQueue();
  await pump();
}

async function queueCommand(message) {
  if (message.kind === "queue_cancel") {
    await cancelItem(message.id);
    return;
  }
  let before = [];
  const after = await changeQueue(function (state) {
    before = state.items;
    if (message.kind === "queue_pause") return Q.setPaused(state, true);
    if (message.kind === "queue_resume") return Q.setPaused(state, false);
    if (message.kind === "queue_retry") return Q.retry(state, message.id);
    if (message.kind === "queue_drop") return Q.drop(state, message.id);
    if (message.kind === "queue_clear") return Q.clear(state);
    if (message.kind === "queue_set_provider") return Q.setProvider(state, message.id, message.provider === "" ? null : message.provider);
    return state;
  });
  const kept = after.items.map(function (it) { return it.id; });
  await Promise.all(before.filter(function (it) { return kept.indexOf(it.id) < 0; }).map(announceRemoved));
  if (message.kind === "queue_retry" || message.kind === "queue_set_provider") await announce(message.id);
  await broadcastQueue();
  await pump();
}

const QUEUE_COMMANDS = ["queue_pause", "queue_resume", "queue_retry", "queue_drop", "queue_clear", "queue_cancel", "queue_set_provider"];

chrome.runtime.onMessage.addListener(function (message, sender, sendResponse) {
  if (!message) return false;
  if (message.kind === "queue_get") {
    readQueue().then(function (state) { sendResponse(Q.counts(state)); }, function (err) {
      console.error("ai-bridge: could not read the queue:", err);
      sendResponse(null);
    });
    return true;
  }
  if (QUEUE_COMMANDS.indexOf(message.kind) >= 0) {
    queueCommand(message).catch(function (err) {
      console.error("ai-bridge: " + message.kind + " failed:", err);
    });
    return false;
  }
  const tabId = sender.tab ? sender.tab.id : null;
  if (tabId === null) return false;
  if (message.kind === "ready") {
    handOutWork(tabId).then(sendResponse, function (err) {
      console.error("ai-bridge: could not hand out the run:", err);
      sendResponse(null);
    });
    return true;
  }
  let work;
  if (message.kind === "page_cancel") work = cancelFromPage(message.itemId);
  else if (message.kind === "queue_add") work = addToQueue(message, tabId);
  else if (message.kind === "progress" || message.kind === "result" || message.kind === "error") work = forwardFromAdapter(message, tabId);
  else return false;
  work.catch(function (err) {
    console.error("ai-bridge: " + message.kind + " failed:", err);
    if (message.kind === "queue_add") toPage(tabId, P.errorToPage(message.requestId, "bad_request", String(err && err.message ? err.message : err)));
  });
  return false;
});

chrome.tabs.onRemoved.addListener(function (tabId) {
  readJob().then(async function (job) {
    if (!job || job.providerTabId !== tabId) return;
    await clearJob();
    await failItem(job.itemId, errorText("tab_closed", "", job.provider));
  }).catch(function (err) {
    console.error("ai-bridge: could not settle the run of closed tab " + tabId + ":", err);
  });
});

// A new default provider shows on the waiting items that have not chosen one.
chrome.storage.onChanged.addListener(function (changes, area) {
  if (area !== "local" || !changes[PR.DEFAULT_KEY]) return;
  readQueue().then(function (state) {
    return Promise.all(state.items.filter(function (it) { return it.status === "waiting" && it.provider === null; }).map(function (it) { return announce(it.id); }));
  }).catch(function (err) {
    console.error("ai-bridge: could not announce the new default provider:", err);
  });
});

// The toolbar icon opens the popup (queue and settings); nothing else to do on click.

/**
 * When the worker starts: an item that was running but whose provider run is gone
 * goes back to waiting, then the queue carries on.
 */
async function boot() {
  const job = await readJob();
  const keep = J.isActive(job, Date.now()) && job.itemId ? job.itemId : null;
  await changeQueue(function (state) { return Q.recover(state, keep); });
  await pump();
}

boot().catch(function (err) {
  console.error("ai-bridge: could not start the queue:", err);
});
