const test = require("node:test");
const assert = require("node:assert/strict");
const A = require("../src/deepseek/answer.js");

test("an answer is finished once it stays the same with its icon row shown", () => {
  assert.equal(A.isFinished("xong", 0, A.STABLE_MS - 1, true), false);
  assert.equal(A.isFinished("xong", 0, A.STABLE_MS, true), true);
});

test("without an icon row the answer has to stay the same much longer", () => {
  assert.equal(A.isFinished("xong", 0, A.STABLE_MS, false), false);
  assert.equal(A.isFinished("xong", 0, A.STABLE_WITHOUT_ACTIONS_MS, false), true);
});

test("an empty answer is never finished", () => {
  assert.equal(A.isFinished("  ", 0, 10 * A.STABLE_WITHOUT_ACTIONS_MS, true), false);
});

test("toggle labels match with or without the model name after them", () => {
  assert.equal(A.hasLabel(" DeepThink (R1) ", ["DeepThink"]), true);
  assert.equal(A.hasLabel("search", ["Search"]), true);
  assert.equal(A.hasLabel("联网搜索", ["Search", "联网搜索"]), true);
  assert.equal(A.hasLabel("Research notes", ["Search"]), false);
});

test("code blocks are fenced again so the JSON block can be found", () => {
  const text = A.joinBlocks([
    { kind: "text", text: " Kết quả: " },
    { kind: "code", lang: "json", code: '{"a":1}\n\n' },
    { kind: "text", text: "" },
    { kind: "code", lang: "", code: "  " },
  ]);
  assert.equal(text, 'Kết quả:\n\n```json\n{"a":1}\n```');
});
