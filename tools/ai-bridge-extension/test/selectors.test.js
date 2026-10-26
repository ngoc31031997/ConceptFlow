const test = require("node:test");
const assert = require("node:assert/strict");
const DS = require("../src/deepseek/selectors.js");

test("without overrides every selector is its default", () => {
  const S = DS.resolve(undefined);
  assert.equal(S.input, "textarea#chat-input, textarea");
  assert.deepEqual(S.deepThinkLabels, ["DeepThink", "深度思考"]);
  assert.equal(S.sendButton, "");
  assert.ok(S.selectedClass.test("ds-toggle-button--selected"));
  assert.deepEqual(S.invalid, []);
});

test("a non-blank override replaces its default and blank ones are ignored", () => {
  const S = DS.resolve({ input: "  #chat-input  ", sendButton: "div._abc", searchLabels: "Tìm kiếm, Search", markdown: "   " });
  assert.equal(S.input, "#chat-input");
  assert.equal(S.sendButton, "div._abc");
  assert.deepEqual(S.searchLabels, ["Tìm kiếm", "Search"]);
  assert.equal(S.markdown, ".ds-markdown");
});

test("a broken regular expression falls back to the default and is reported", () => {
  const S = DS.resolve({ selectedClass: "selected(" });
  assert.deepEqual(S.invalid, ["selectedClass"]);
  assert.ok(S.selectedClass.test("x--active"));
});
