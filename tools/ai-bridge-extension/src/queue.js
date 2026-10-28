/**
 * The DeepSeek queue as it is kept in chrome.storage.local: items that run one
 * after another, each a plain HTTP request of ConceptFlow's API that the
 * service worker makes in the "prompt" and "paste" external-AI modes around a
 * run on DeepSeek. Every function takes a state and returns a new one, so the
 * service worker can read the state, change it and write it back after being
 * stopped and restarted, and node tests can check the rules.
 */
(function (root) {
  const STATUSES = ["waiting", "running", "done", "failed"];

  /** Statuses whose request still has work to do, so the same request is not added twice. */
  const OPEN = ["waiting", "running", "failed"];

  function emptyState() {
    return { items: [], paused: false };
  }

  /** Reads the state stored earlier; anything that is not a queue gives an empty one. */
  function readState(raw) {
    if (!raw || typeof raw !== "object" || !Array.isArray(raw.items)) return emptyState();
    return { items: raw.items.filter(isItem), paused: raw.paused === true };
  }

  function isItem(item) {
    return !!item && typeof item.id === "string" && STATUSES.indexOf(item.status) >= 0 && !!item.request;
  }

  /**
   * Checks a request the page wants queued. Only a POST to the API (`/v1/...`)
   * of an http(s) gateway is accepted, since the worker will make it as is.
   */
  function readRequest(raw) {
    if (!raw || typeof raw !== "object") return null;
    if (typeof raw.gateway !== "string" || !/^https?:\/\/[^/\s]+$/.test(raw.gateway)) return null;
    if (raw.method !== "POST") return null;
    if (typeof raw.path !== "string" || !raw.path.startsWith("/v1/") || raw.path.includes("..")) return null;
    if (raw.body !== undefined && typeof raw.body !== "string") return null;
    return { gateway: raw.gateway, method: raw.method, path: raw.path, body: raw.body };
  }

  function sameRequest(a, b) {
    return a.gateway === b.gateway && a.method === b.method && a.path === b.path && (a.body || "") === (b.body || "");
  }

  /**
   * Adds an item the page described, as `{label, project, request, modes}`.
   * Returns `{state, item, added}`; a request that is already waiting, running
   * or failed is not added again and `item` is the one that holds it. `item`
   * is null when the description is not valid.
   */
  function add(state, raw, id, now) {
    const request = raw ? readRequest(raw.request) : null;
    if (!request || typeof raw.label !== "string" || raw.label.trim() === "") return { state: state, item: null, added: false };
    const same = state.items.find(function (it) {
      return OPEN.indexOf(it.status) >= 0 && sameRequest(it.request, request);
    });
    if (same) return { state: state, item: same, added: false };
    const item = {
      id: id,
      label: raw.label.trim(),
      project: typeof raw.project === "string" ? raw.project : "",
      request: request,
      modes: { deepThink: !raw.modes || raw.modes.deepThink !== false, search: !!(raw.modes && raw.modes.search === true) },
      status: "waiting",
      stage: "",
      error: "",
      reply: "",
      addedAt: now,
    };
    return { state: Object.assign({}, state, { items: state.items.concat([item]) }), item: item, added: true };
  }

  function update(state, id, change) {
    return Object.assign({}, state, {
      items: state.items.map(function (it) {
        return it.id === id ? Object.assign({}, it, change) : it;
      }),
    });
  }

  function find(state, id) {
    return state.items.find(function (it) {
      return it.id === id;
    }) || null;
  }

  /**
   * The item to run now, or null: the queue is paused, an item is running, an
   * item failed (the queue stops there until the Creator retries or drops it),
   * or nothing waits.
   */
  function next(state) {
    if (state.paused) return null;
    if (state.items.some(function (it) { return it.status === "running" || it.status === "failed"; })) return null;
    return state.items.find(function (it) { return it.status === "waiting"; }) || null;
  }

  function start(state, id) {
    return update(state, id, { status: "running", stage: "preparing", error: "", reply: "" });
  }

  function setStage(state, id, stage) {
    return update(state, id, { stage: stage });
  }

  function finish(state, id) {
    return update(state, id, { status: "done", stage: "", error: "", reply: "" });
  }

  /** `reply` is DeepSeek's answer when the server could not use it, kept so the Creator can fix it by hand. */
  function fail(state, id, error, reply) {
    return update(state, id, { status: "failed", stage: "", error: error, reply: reply || "" });
  }

  /** A failed item goes back to waiting; its request is made again from the start. */
  function retry(state, id) {
    const item = find(state, id);
    return item && item.status === "failed" ? update(state, id, { status: "waiting", stage: "", error: "", reply: "" }) : state;
  }

  /** Removes an item that is not running. */
  function drop(state, id) {
    const item = find(state, id);
    if (!item || item.status === "running") return state;
    return Object.assign({}, state, { items: state.items.filter(function (it) { return it.id !== id; }) });
  }

  /** Removes every item that is not running. */
  function clear(state) {
    return Object.assign({}, state, { items: state.items.filter(function (it) { return it.status === "running"; }) });
  }

  function setPaused(state, paused) {
    return Object.assign({}, state, { paused: paused });
  }

  /**
   * After the worker restarts: a running item whose DeepSeek run is gone
   * (`keepId` is the item of the run still stored, or null) goes back to
   * waiting, so it is made again from the start.
   */
  function recover(state, keepId) {
    return Object.assign({}, state, {
      items: state.items.map(function (it) {
        return it.status === "running" && it.id !== keepId ? Object.assign({}, it, { status: "waiting", stage: "" }) : it;
      }),
    });
  }

  /** The numbers the page shows on its queue button. */
  function counts(state) {
    const n = { waiting: 0, running: 0, done: 0, failed: 0 };
    state.items.forEach(function (it) {
      n[it.status] += 1;
    });
    return { waiting: n.waiting, running: n.running, done: n.done, failed: n.failed, paused: state.paused };
  }

  const api = { STATUSES, emptyState, readState, readRequest, add, find, next, start, setStage, finish, fail, retry, drop, clear, setPaused, recover, counts };
  root.AiBridgeQueue = api;
  if (typeof module !== "undefined" && module.exports) module.exports = api;
})(typeof globalThis !== "undefined" ? globalThis : self);
