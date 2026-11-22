const test = require("node:test");
const assert = require("node:assert/strict");
const A = require("../src/adapter/answer.js");

test("code blocks are fenced again so the JSON block can be found", () => {
  const text = A.joinBlocks([
    { kind: "text", text: " Kết quả: " },
    { kind: "code", lang: "json", code: '{"a":1}\n\n' },
    { kind: "text", text: "" },
    { kind: "code", lang: "", code: "  " },
  ]);
  assert.equal(text, 'Kết quả:\n\n```json\n{"a":1}\n```');
});
