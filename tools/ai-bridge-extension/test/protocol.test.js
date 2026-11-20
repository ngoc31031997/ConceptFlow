const test = require("node:test");
const assert = require("node:assert/strict");
const P = require("../src/protocol.js");
const J = require("../src/jobs.js");

const page = (fields) => Object.assign({ source: P.SOURCE, v: P.VERSION, from: "page" }, fields);

test("the protocol is version 2", () => {
  assert.equal(P.VERSION, 2);
});

test("ping and a cancel of a queued item are accepted", () => {
  assert.deepEqual(P.parsePageMessage(page({ type: "ping" })), { type: "ping" });
  assert.deepEqual(P.parsePageMessage(page({ type: "cancel", item_id: "q1" })), { type: "cancel", itemId: "q1" });
  assert.equal(P.parsePageMessage(page({ type: "cancel" })), null, "a cancel without an item");
});

test("a queue_add carries the item the page described", () => {
  const item = { label: "B3", request: { gateway: "http://localhost:8080", method: "POST", path: "/v1/projects/p1/beats/b3/redo" } };
  assert.deepEqual(P.parsePageMessage(page({ type: "queue_add", request_id: "r1", item })), { type: "queue_add", requestId: "r1", item });
  assert.equal(P.parsePageMessage(page({ type: "queue_add", request_id: "r1" })), null, "no item");
  assert.equal(P.parsePageMessage(page({ type: "queue_add", item })), null, "no request id");
});

test("messages that are not page requests for this extension are ignored", () => {
  assert.equal(P.parsePageMessage(null), null);
  assert.equal(P.parsePageMessage({ type: "ping" }), null, "a message without the source tag");
  assert.equal(P.parsePageMessage(page({ type: "ping", v: 1 })), null, "the old protocol version");
  assert.equal(P.parsePageMessage(P.toPage("pong", {})), null, "the extension's own reply");
  assert.equal(P.parsePageMessage(page({ type: "run", request_id: "r1", provider: "deepseek", prompt: "x" })), null, "a direct run no longer exists");
});

test("a message to the page carries the version, the source and the sender", () => {
  const msg = P.toPage("queue_item", { item_id: "q1" });
  assert.deepEqual(msg, { source: P.SOURCE, v: 2, from: "extension", type: "queue_item", item_id: "q1" });
});

test("an item that does not say turns DeepThink on and Search off", () => {
  assert.deepEqual(P.readModes(null), { deepThink: true, search: false });
  assert.deepEqual(P.readModes({ deepThink: false, search: true }), { deepThink: false, search: true });
});

test("an error for the page carries its Vietnamese text, the provider's name and the detail", () => {
  const gemini = { name: "Gemini", host: "gemini.google.com" };
  const msg = P.errorToPage("r1", "ui_changed", "nút Copy", gemini);
  assert.equal(msg.from, "extension");
  assert.equal(msg.code, "ui_changed");
  assert.match(msg.message, /Giao diện Gemini đã đổi.*\(nút Copy\)$/);
  assert.match(P.errorMessage("not_logged_in", "", gemini), /Chưa đăng nhập Gemini.*gemini\.google\.com/);
  assert.match(P.errorMessage("timeout", "", { name: "Grok", host: "grok.com" }), /^Grok chưa trả lời xong/);
  assert.match(P.errorMessage("tab_closed", "", { name: "DeepSeek", host: "chat.deepseek.com" }), /^Tab DeepSeek đã bị đóng/);
  assert.match(P.errorMessage("timeout", ""), /^AI ngoài chưa trả lời xong/, "without a provider the name is generic");
});

test("a stored job blocks a new run until it is older than the run limit plus grace", () => {
  const job = J.newJob("i1", "gemini", "p", P.readModes(null), 1000);
  assert.equal(J.isActive(job, 1000 + P.MAX_RUN_MS), true);
  assert.equal(J.isActive(job, 1000 + P.MAX_RUN_MS + J.STALE_GRACE_MS), false);
  assert.equal(J.isActive(null, 0), false);
});

test("adapter messages only count for the job's own tab and request", () => {
  const job = Object.assign(J.newJob("i1", "grok", "p", P.readModes(null), 0), { providerTabId: 9 });
  assert.equal(J.belongsTo(job, 9, "queue-i1"), true);
  assert.equal(J.belongsTo(job, 8, "queue-i1"), false);
  assert.equal(J.belongsTo(job, 9, "queue-i2"), false);
});

test("a job names its queue item and the provider it runs on", () => {
  const job = J.newJob("i1", "gemini", "p", P.readModes(null), 1000);
  assert.equal(job.itemId, "i1");
  assert.equal(job.provider, "gemini");
  assert.equal(job.requestId, "queue-i1");
});
