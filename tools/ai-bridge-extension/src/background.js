/**
 * Service worker: opens a DeepSeek tab in the background for a run, hands the
 * prompt to the DeepSeek adapter once that tab is ready, and forwards the
 * adapter's progress, answer or error to the ConceptFlow tab that asked.
 * One run at a time; the run is kept in chrome.storage.session (see jobs.js).
 *
 * It also runs the queue (queue.js, kept in chrome.storage.local): each item
 * in turn gets its prompt from the API, runs on DeepSeek, and has the API read
 * the answer, with no ConceptFlow tab needed. A failed item stops the queue.
 */
importScripts("protocol.js", "jobs.js", "queue.js", "queue-http.js");

const P = self.AiBridgeProtocol;
const J = self.AiBridgeJobs;
const Q = self.AiBridgeQueue;
const H = self.AiBridgeQueueHttp;
const JOB_KEY = "job";
const QUEUE_KEY = "queue";
const DEEPSEEK_URL = "https://chat.deepseek.com/";
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
  await broadcastQueue();
  await pump();
}

function setStage(itemId, stage) {
  return changeQueue(function (state) { return Q.setStage(state, itemId, stage); });
}

async function failItem(itemId, error, reply) {
  await changeQueue(function (state) { return Q.fail(state, itemId, error, reply); });
  await broadcastQueue();
}

async function finishItem(item) {
  await changeQueue(function (state) { return Q.finish(state, item.id); });
  await toAllPages(P.toPage("queue_item_done", { item_id: item.id, path: item.request.path, project: item.project }));
  await broadcastQueue();
  await pump();
}

/** The words for a run that ended in an error code, as the page would show them. */
function errorText(code, detail) {
  return P.errorToPage("", code, detail).message;
}

/** Starts the next waiting item when nothing else is running. */
async function pump() {
  const job = await readJob();
  if (J.isActive(job, Date.now())) return;
  let picked = null;
  await changeQueue(function (state) {
    picked = Q.next(state);
    return picked ? Q.start(state, picked.id) : state;
  });
  await broadcastQueue();
  if (picked) await runItem(picked);
}

/** Makes the item's prompt from the API as the project is now, then runs it on DeepSeek. */
async function runItem(item) {
  const prompt = await H.callExternal(fetch.bind(self), item.request, "prompt");
  if (!prompt.ok) {
    await failItem(item.id, prompt.message);
    return;
  }
  const job = J.newJob("queue-" + item.id, prompt.prompt, item.modes, null, Date.now(), item.id);
  await writeJob(job);
  await setStage(item.id, "opening");
  await broadcastQueue();
  const tab = await chrome.tabs.create({ url: DEEPSEEK_URL, active: true });
  job.providerTabId = tab.id;
  await writeJob(job);
}

/** The adapter's answer for a queue item: the API reads it, or the item fails with the answer kept. */
async function settleQueueRun(job, message) {
  const item = Q.find(await readQueue(), job.itemId);
  if (!item) return;
  if (message.kind === "progress") {
    await setStage(item.id, message.stage);
    await broadcastQueue();
    return;
  }
  await clearJob();
  if (message.kind === "error") {
    await failItem(item.id, errorText(message.code, message.detail));
    await pump();
    return;
  }
  await finishProviderTab(null, job.providerTabId);
  await setStage(item.id, "saving");
  const saved = await H.callExternal(fetch.bind(self), item.request, "paste", message.text);
  if (!saved.ok) {
    await failItem(item.id, saved.message, message.text);
    return;
  }
  await finishItem(item);
}

// ---- manual runs from the page ------------------------------------------

async function startRun(message, replyTabId) {
  const now = Date.now();
  const current = await readJob();
  const queueRunning = (await readQueue()).items.some(function (it) { return it.status === "running"; });
  if (J.isActive(current, now) || queueRunning) {
    await toPage(replyTabId, P.errorToPage(message.requestId, "busy"));
    return;
  }
  const job = J.newJob(message.requestId, message.prompt, message.modes || P.readModes(null), replyTabId, now);
  await writeJob(job);
  await toPage(replyTabId, P.toPage("progress", { request_id: job.requestId, stage: "opening" }));
  // The tab is shown, not hidden: Chrome slows a hidden tab's timers and page updates, which delays reading the answer.
  const tab = await chrome.tabs.create({ url: DEEPSEEK_URL, active: true });
  job.providerTabId = tab.id;
  await writeJob(job);
}

/** The adapter asks for its work when its page has loaded; it gets the prompt once. */
async function handOutWork(tabId) {
  const job = await readJob();
  if (!job || job.providerTabId !== tabId || job.started) return null;
  job.started = true;
  await writeJob(job);
  return { requestId: job.requestId, prompt: job.prompt, modes: job.modes, deadline: job.startedAt + P.MAX_RUN_MS };
}

/**
 * Takes the Creator back to the ConceptFlow tab (when there is one) and closes the
 * DeepSeek tab of a run whose answer was delivered. The job is cleared first, so
 * closing the tab is not reported as an error.
 */
async function finishProviderTab(replyTabId, providerTabId) {
  if (replyTabId !== null) {
    try {
      await chrome.tabs.update(replyTabId, { active: true });
    } catch (err) {
      console.warn("ai-bridge: could not show the ConceptFlow tab " + replyTabId + ":", err);
    }
  }
  try {
    await chrome.tabs.remove(providerTabId);
  } catch (err) {
    console.warn("ai-bridge: could not close the DeepSeek tab " + providerTabId + ":", err);
  }
}

async function forwardFromAdapter(message, tabId) {
  const job = await readJob();
  if (!J.belongsTo(job, tabId, message.requestId)) return;
  if (job.itemId) {
    await settleQueueRun(job, message);
    return;
  }
  if (message.kind === "progress") {
    await toPage(job.replyTabId, P.toPage("progress", { request_id: job.requestId, stage: message.stage }));
    return;
  }
  await clearJob();
  if (message.kind === "result") {
    await toPage(job.replyTabId, P.toPage("result", { request_id: job.requestId, text: message.text }));
    await finishProviderTab(job.replyTabId, job.providerTabId);
  } else {
    await toPage(job.replyTabId, P.errorToPage(job.requestId, message.code, message.detail));
  }
  await pump();
}

async function cancelRun(requestId) {
  const job = await readJob();
  if (!job || job.requestId !== requestId) return;
  await clearJob();
  if (job.providerTabId !== null) {
    chrome.tabs.sendMessage(job.providerTabId, { kind: "cancel", requestId: requestId }).catch(function (err) {
      console.warn("ai-bridge: DeepSeek tab did not take the cancel:", err);
    });
  }
  await pump();
}

// ---- commands from the popup ---------------------------------------------

/** Stops the run of a queue item; the item fails as cancelled, so the queue waits for the Creator. */
async function cancelItem(itemId) {
  const job = await readJob();
  if (job && job.itemId === itemId) {
    await clearJob();
    if (job.providerTabId !== null) {
      chrome.tabs.sendMessage(job.providerTabId, { kind: "cancel", requestId: job.requestId }).catch(function (err) {
        console.warn("ai-bridge: DeepSeek tab did not take the cancel:", err);
      });
      await finishProviderTab(null, job.providerTabId);
    }
  }
  await failItem(itemId, "Đã huỷ.");
}

async function queueCommand(message) {
  if (message.kind === "queue_cancel") {
    await cancelItem(message.id);
    return;
  }
  await changeQueue(function (state) {
    if (message.kind === "queue_pause") return Q.setPaused(state, true);
    if (message.kind === "queue_resume") return Q.setPaused(state, false);
    if (message.kind === "queue_retry") return Q.retry(state, message.id);
    if (message.kind === "queue_drop") return Q.drop(state, message.id);
    if (message.kind === "queue_clear") return Q.clear(state);
    return state;
  });
  await broadcastQueue();
  await pump();
}

const QUEUE_COMMANDS = ["queue_pause", "queue_resume", "queue_retry", "queue_drop", "queue_clear", "queue_cancel"];

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
  if (message.kind === "run") work = startRun(message, tabId);
  else if (message.kind === "cancel") work = cancelRun(message.requestId);
  else if (message.kind === "queue_add") work = addToQueue(message, tabId);
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
      if (job.itemId) {
        await failItem(job.itemId, errorText("tab_closed"));
      } else {
        await toPage(job.replyTabId, P.errorToPage(job.requestId, "tab_closed"));
        await pump();
      }
    } else if (job.replyTabId === tabId) {
      await clearJob();
      await pump();
    }
  }).catch(function (err) {
    console.error("ai-bridge: could not settle the run of closed tab " + tabId + ":", err);
  });
});

// The toolbar icon opens the popup (queue and selector settings); nothing else to do on click.

/**
 * When the worker starts: an item that was running but whose DeepSeek run is gone
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
