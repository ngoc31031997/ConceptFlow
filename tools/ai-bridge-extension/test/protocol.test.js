const test = require("node:test");
const assert = require("node:assert/strict");
const P = require("../src/protocol.js");
const J = require("../src/jobs.js");

const page = (fields) => Object.assign({ source: P.SOURCE, v: P.VERSION, from: "page" }, fields);

test("a run with a known provider and a prompt is accepted", () => {
  assert.deepEqual(P.parsePageMessage(page({ type: "run", request_id: "r1", provider: "deepseek", prompt: "hỏi" })), {
    type: "run",
    requestId: "r1",
    provider: "deepseek",
    prompt: "hỏi",
    modes: { deepThink: true, search: true },
  });
});

test("a run can turn DeepSeek's search off and keeps DeepThink on unless told", () => {
  const msg = P.parsePageMessage(page({ type: "run", request_id: "r1", provider: "deepseek", prompt: "hỏi", modes: { search: false } }));
  assert.deepEqual(msg.modes, { deepThink: true, search: false });
  assert.deepEqual(P.readModes({ deep_think: false, search: false }), { deepThink: false, search: false });
});

test("messages that are not page requests for this extension are ignored", () => {
  assert.equal(P.parsePageMessage(null), null);
  assert.equal(P.parsePageMessage({ type: "ping" }), null, "a message without the source tag");
  assert.equal(P.parsePageMessage(page({ type: "ping", v: 2 })), null, "another protocol version");
  assert.equal(P.parsePageMessage(P.toPage("pong", {})), null, "the extension's own reply");
  assert.equal(P.parsePageMessage(page({ type: "run", request_id: "r1", provider: "chatgpt", prompt: "x" })), null, "an unknown provider");
  assert.equal(P.parsePageMessage(page({ type: "run", request_id: "r1", provider: "deepseek", prompt: "  " })), null, "a blank prompt");
  assert.equal(P.parsePageMessage(page({ type: "run", provider: "deepseek", prompt: "x" })), null, "a run without request id");
});

test("ping and cancel are accepted", () => {
  assert.deepEqual(P.parsePageMessage(page({ type: "ping" })), { type: "ping" });
  assert.deepEqual(P.parsePageMessage(page({ type: "cancel", request_id: "r1" })), { type: "cancel", requestId: "r1" });
});

test("an error for the page carries its Vietnamese text and the detail", () => {
  const msg = P.errorToPage("r1", "ui_changed", "nút Search");
  assert.equal(msg.from, "extension");
  assert.equal(msg.code, "ui_changed");
  assert.match(msg.message, /Giao diện DeepSeek đã đổi.*\(nút Search\)$/);
});

test("a stored job blocks a new run until it is older than the run limit plus grace", () => {
  const job = J.newJob("r1", "p", P.readModes(null), 7, 1000);
  assert.equal(J.isActive(job, 1000 + P.MAX_RUN_MS), true);
  assert.equal(J.isActive(job, 1000 + P.MAX_RUN_MS + J.STALE_GRACE_MS), false);
  assert.equal(J.isActive(null, 0), false);
});

test("adapter messages only count for the job's own tab and request", () => {
  const job = Object.assign(J.newJob("r1", "p", P.readModes(null), 7, 0), { providerTabId: 9 });
  assert.equal(J.belongsTo(job, 9, "r1"), true);
  assert.equal(J.belongsTo(job, 8, "r1"), false);
  assert.equal(J.belongsTo(job, 9, "r2"), false);
});

test("a queue_add carries the item the page described", () => {
  const item = { label: "B3", request: { gateway: "http://localhost:8080", method: "POST", path: "/v1/projects/p1/beats/b3/redo" } };
  assert.deepEqual(P.parsePageMessage(page({ type: "queue_add", request_id: "r1", item })), { type: "queue_add", requestId: "r1", item });
  assert.equal(P.parsePageMessage(page({ type: "queue_add", request_id: "r1" })), null, "no item");
  assert.equal(P.parsePageMessage(page({ type: "queue_add", item })), null, "no request id");
});

test("a job for a queue item names its item and has no ConceptFlow tab to answer", () => {
  const job = J.newJob("queue-i1", "p", P.readModes(null), null, 1000, "i1");
  assert.equal(job.itemId, "i1");
  assert.equal(job.replyTabId, null);
  assert.equal(J.newJob("r1", "p", P.readModes(null), 7, 1000).itemId, null, "a run from the page has no item");
});
