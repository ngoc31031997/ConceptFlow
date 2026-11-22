const test = require("node:test");
const assert = require("node:assert/strict");
const PR = require("../src/providers.js");
const DS = require("../src/providers/deepseek/selectors.js");
const GS = require("../src/providers/gemini/selectors.js");
const XS = require("../src/providers/grok/selectors.js");
const { makePage } = require("./helpers/page.js");

test("without overrides every DeepSeek selector is its default", () => {
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

test("Gemini's defaults are the tags and test ids of its page", () => {
  const S = GS.resolve(undefined);
  assert.equal(S.input, 'rich-textarea .ql-editor[contenteditable="true"]');
  assert.equal(S.sendButton, '[data-test-id="send-button-container"] .send-button.submit button');
  assert.equal(S.answer, "model-response");
  assert.equal(S.doneMarker, ".response-footer.complete");
  assert.equal(S.settledMarker, 'message-content [aria-busy="false"]');
  assert.equal(S.copyButton, "message-actions copy-button button");
  assert.equal(GS.STORAGE_KEY, "geminiSelectors");
});

test("Grok's defaults are the test ids of its page, and it has no settled marker", () => {
  const S = XS.resolve(undefined);
  assert.equal(S.input, '[data-testid="chat-input"] .ProseMirror[contenteditable="true"]');
  assert.equal(S.sendButton, 'button[data-testid="chat-submit"]');
  assert.equal(S.answer, '[data-testid="assistant-message"]');
  assert.equal(S.doneMarker, '[data-testid="response-footer-actions"]');
  assert.equal(S.settledMarker, "");
  assert.equal(S.copyButton, '[data-testid="response-footer-actions"] > div:first-of-type > button:first-of-type');
  assert.equal(XS.STORAGE_KEY, "grokSelectors");
});

test("an override of a plain-selector provider replaces its default; a broken one is reported and not used", () => {
  const page = makePage("<body></body>", "https://grok.com/");
  const S = XS.resolve({ copyButton: " button.copy ", answer: "div[", settledMarker: "  " }, page.doc);
  assert.equal(S.copyButton, "button.copy");
  assert.equal(S.answer, '[data-testid="assistant-message"]', "the broken selector fell back to its default");
  assert.equal(S.settledMarker, "");
  assert.deepEqual(S.invalid, ["answer"]);
  page.close();
});

test("the provider table names each provider's chat URL and host, and an unknown default falls back to DeepSeek", () => {
  assert.deepEqual(PR.IDS, ["deepseek", "gemini", "grok"]);
  assert.equal(PR.find("gemini").url, "https://gemini.google.com/app");
  assert.equal(PR.find("grok").url, "https://grok.com/");
  assert.equal(PR.find("deepseek").url, "https://chat.deepseek.com/");
  assert.equal(PR.find("chatgpt"), null);
  assert.equal(PR.readDefault("grok"), "grok");
  assert.equal(PR.readDefault("chatgpt"), "deepseek");
  assert.equal(PR.readDefault(undefined), "deepseek");
  assert.equal(PR.nameOf("gemini"), "Gemini");
});

test("the manifest has a host permission and a content script for every provider", () => {
  const manifest = require("../manifest.json");
  PR.LIST.forEach((p) => {
    assert.ok(manifest.host_permissions.includes(p.match), p.id + " host permission");
    const scripts = manifest.content_scripts.filter((c) => c.matches.includes(p.match));
    assert.ok(scripts.some((c) => c.world === "MAIN" && c.js.includes("src/adapter/clipboard-hook.js")), p.id + " clipboard hook in the page world");
    assert.ok(scripts.some((c) => c.js.includes("src/providers/" + p.id + "/adapter.js")), p.id + " adapter");
  });
  assert.equal(manifest.version, "2.0.0");
});
