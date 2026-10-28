const test = require("node:test");
const assert = require("node:assert/strict");
const Q = require("../src/queue.js");
const H = require("../src/queue-http.js");

const request = (key) => ({ gateway: "http://localhost:8080", method: "POST", path: "/v1/projects/p1/beats/" + key + "/redo", body: JSON.stringify({ note: "" }) });
const described = (key) => ({ label: "B" + key, project: "Dự án 1", request: request(key), modes: { deepThink: true, search: false } });

function queueOf(...keys) {
  let state = Q.emptyState();
  keys.forEach((key, i) => {
    state = Q.add(state, described(key), "i" + i, 1000 + i).state;
  });
  return state;
}

test("an item is added once per request, even when the page asks again", () => {
  const first = Q.add(Q.emptyState(), described("b1"), "i1", 1);
  assert.equal(first.added, true);
  assert.equal(first.item.status, "waiting");
  const again = Q.add(first.state, described("b1"), "i2", 2);
  assert.equal(again.added, false);
  assert.equal(again.item.id, "i1");
  assert.equal(again.state.items.length, 1);
});

test("a request that already finished can be queued again", () => {
  let state = queueOf("b1");
  state = Q.finish(Q.start(state, "i0"), "i0");
  const again = Q.add(state, described("b1"), "i9", 9);
  assert.equal(again.added, true);
  assert.equal(again.state.items.length, 2);
});

test("only a POST to the API of an http gateway is accepted", () => {
  const bad = (change) => Q.add(Q.emptyState(), Object.assign(described("b1"), { request: Object.assign(request("b1"), change) }), "i1", 1);
  assert.equal(bad({ method: "DELETE" }).item, null);
  assert.equal(bad({ path: "/admin/x" }).item, null);
  assert.equal(bad({ path: "/v1/../secret" }).item, null);
  assert.equal(bad({ gateway: "file:///etc" }).item, null);
  assert.equal(bad({ gateway: "http://localhost:8080/extra" }).item, null);
  assert.equal(bad({ body: 5 }).item, null);
  assert.equal(Q.add(Q.emptyState(), Object.assign(described("b1"), { label: " " }), "i1", 1).item, null, "an item without a name");
  assert.equal(Q.add(Q.emptyState(), null, "i1", 1).item, null);
});

test("DeepThink is on and search is off unless the page asks otherwise", () => {
  const { item } = Q.add(Q.emptyState(), Object.assign(described("b1"), { modes: undefined }), "i1", 1);
  assert.deepEqual(item.modes, { deepThink: true, search: false });
  const withSearch = Q.add(Q.emptyState(), Object.assign(described("b1"), { modes: { deepThink: false, search: true } }), "i1", 1);
  assert.deepEqual(withSearch.item.modes, { deepThink: false, search: true });
});

test("items run one at a time, in the order they were added", () => {
  let state = queueOf("b1", "b2", "b3");
  assert.equal(Q.next(state).id, "i0");
  state = Q.start(state, "i0");
  assert.equal(Q.next(state), null, "one is running");
  state = Q.finish(state, "i0");
  assert.equal(Q.next(state).id, "i1");
});

test("a failed item stops the queue until it is retried or dropped", () => {
  let state = queueOf("b1", "b2");
  state = Q.fail(Q.start(state, "i0"), "i0", "câu trả lời không dùng được", "```json {}```");
  assert.equal(Q.next(state), null);
  const failed = Q.find(state, "i0");
  assert.equal(failed.error, "câu trả lời không dùng được");
  assert.equal(failed.reply, "```json {}```");
  assert.equal(Q.next(Q.retry(state, "i0")).id, "i0");
  assert.equal(Q.next(Q.drop(state, "i0")).id, "i1");
  assert.equal(Q.find(Q.drop(state, "i0"), "i1").status, "waiting", "the items after it are kept");
});

test("a paused queue starts nothing", () => {
  const state = Q.setPaused(queueOf("b1"), true);
  assert.equal(Q.next(state), null);
  assert.equal(Q.next(Q.setPaused(state, false)).id, "i0");
});

test("a running item cannot be dropped or cleared", () => {
  let state = Q.start(queueOf("b1", "b2"), "i0");
  assert.equal(Q.drop(state, "i0"), state);
  state = Q.clear(state);
  assert.deepEqual(state.items.map((it) => it.id), ["i0"]);
});

test("retry touches only a failed item", () => {
  const state = queueOf("b1");
  assert.equal(Q.retry(state, "i0"), state);
});

test("after a restart a running item with no run left starts again from the beginning", () => {
  const state = Q.setStage(Q.start(queueOf("b1", "b2"), "i0"), "i0", "answering");
  assert.equal(Q.find(Q.recover(state, null), "i0").status, "waiting");
  assert.equal(Q.find(Q.recover(state, "i0"), "i0").status, "running", "its DeepSeek run is still stored");
});

test("counts say how many items are in each state", () => {
  let state = queueOf("b1", "b2", "b3", "b4");
  state = Q.finish(Q.start(state, "i0"), "i0");
  state = Q.fail(Q.start(state, "i1"), "i1", "lỗi");
  state = Q.start(state, "i2");
  assert.deepEqual(Q.counts(state), { waiting: 1, running: 1, done: 1, failed: 1, paused: false });
});

test("a state read from storage drops what is not a queue item", () => {
  assert.deepEqual(Q.readState(undefined), Q.emptyState());
  assert.deepEqual(Q.readState({ items: "x" }), Q.emptyState());
  const state = Q.readState({ items: [{ id: "i1", status: "waiting", request: request("b1") }, { id: 5 }, null], paused: true });
  assert.equal(state.items.length, 1);
  assert.equal(state.paused, true);
});

function fakeFetch(answer) {
  const calls = [];
  const impl = async (url, init) => {
    calls.push({ url, init });
    return typeof answer === "function" ? answer(url, init) : answer;
  };
  impl.calls = calls;
  return impl;
}
const reply = (status, body) => ({ ok: status < 400, status, json: async () => body });

test("the prompt request sends the item's own body with the prompt mode", async () => {
  const fetchImpl = fakeFetch(reply(200, { external_prompt: "viết beat" }));
  const out = await H.callExternal(fetchImpl, request("b1"), "prompt");
  assert.deepEqual(out, { ok: true, prompt: "viết beat" });
  assert.equal(fetchImpl.calls[0].url, "http://localhost:8080/v1/projects/p1/beats/b1/redo");
  assert.equal(fetchImpl.calls[0].init.headers["X-External-AI"], "prompt");
  assert.equal(fetchImpl.calls[0].init.body, JSON.stringify({ note: "" }));
});

test("the paste request adds the DeepSeek answer to the item's body", async () => {
  const fetchImpl = fakeFetch(reply(200, {}));
  const out = await H.callExternal(fetchImpl, request("b1"), "paste", "câu trả lời");
  assert.deepEqual(out, { ok: true });
  assert.equal(fetchImpl.calls[0].init.headers["X-External-AI"], "paste");
  assert.deepEqual(JSON.parse(fetchImpl.calls[0].init.body), { note: "", external_reply: "câu trả lời" });
});

test("a paste for an item without a body sends only the answer", async () => {
  const fetchImpl = fakeFetch(reply(200, {}));
  await H.callExternal(fetchImpl, Object.assign(request("b1"), { body: undefined }), "paste", "x");
  assert.deepEqual(JSON.parse(fetchImpl.calls[0].init.body), { external_reply: "x" });
});

test("the server's own words come back when it refuses", async () => {
  const out = await H.callExternal(fakeFetch(reply(422, { error: "AI trả thiếu ô goal" })), request("b1"), "paste", "x");
  assert.deepEqual(out, { ok: false, message: "AI trả thiếu ô goal" });
  const nested = await H.callExternal(fakeFetch(reply(409, { error: { message: "bộ khung chưa duyệt" } })), request("b1"), "prompt");
  assert.deepEqual(nested, { ok: false, message: "bộ khung chưa duyệt" });
});

test("a reply that is not JSON is reported with its status", async () => {
  const bare = { ok: false, status: 502, json: async () => { throw new Error("not json"); } };
  const out = await H.callExternal(fakeFetch(bare), request("b1"), "prompt");
  assert.deepEqual(out, { ok: false, message: "Máy chủ trả lỗi 502." });
});

test("a gateway that cannot be reached is said plainly", async () => {
  const out = await H.callExternal(async () => { throw new TypeError("Failed to fetch"); }, request("b1"), "prompt");
  assert.equal(out.ok, false);
  assert.equal(out.message, H.NO_CONNECTION);
});

test("a prompt answer without a prompt is an error, not an empty run", async () => {
  const out = await H.callExternal(fakeFetch(reply(200, {})), request("b1"), "prompt");
  assert.deepEqual(out, { ok: false, message: H.NO_PROMPT });
});
