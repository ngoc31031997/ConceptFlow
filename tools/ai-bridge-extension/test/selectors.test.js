const test = require("node:test");
const assert = require("node:assert/strict");
const PR = require("../src/providers.js");
const DS = require("../src/providers/deepseek/selectors.js");
const GS = require("../src/providers/gemini/selectors.js");
const XS = require("../src/providers/grok/selectors.js");
const { makePage } = require("./helpers/page.js");

test("every provider has exactly the four basic selectors, all looked up on the whole page", () => {
  [DS, GS, XS].forEach((kit) => {
    assert.deepEqual(kit.FIELDS.map((f) => f.key), ["input", "sendButton", "answer", "copyButton"]);
  });
});

test("DeepSeek's defaults send with Enter and read the answer from the page", () => {
  const S = DS.resolve(undefined);
  assert.equal(S.input, "textarea#chat-input, textarea");
  assert.equal(S.sendButton, "");
  assert.equal(S.answer, ".ds-markdown");
  assert.equal(S.copyButton, "");
  assert.deepEqual(S.invalid, []);
  assert.equal(DS.STORAGE_KEY, "deepseekSelectors");
});

test("Gemini's defaults are the tags and test ids of its page", () => {
  const S = GS.resolve(undefined);
  assert.equal(S.input, 'rich-textarea .ql-editor[contenteditable="true"]');
  assert.equal(S.sendButton, '[data-test-id="send-button-container"] .send-button.submit button');
  assert.equal(S.answer, "model-response");
  assert.equal(S.copyButton, "message-actions copy-button button");
  assert.equal(GS.STORAGE_KEY, "geminiSelectors");
});

test("Grok's Copy default is the first button of the toolbar under the answer", () => {
  const S = XS.resolve(undefined);
  assert.equal(S.input, '[data-testid="chat-input"] .ProseMirror[contenteditable="true"]');
  assert.equal(S.sendButton, 'button[data-testid="chat-submit"]');
  assert.equal(S.answer, '[data-testid="assistant-message"]');
  assert.equal(S.copyButton, '[data-testid="response-footer-actions"] > div:first-of-type > button:first-of-type');
  assert.equal(XS.STORAGE_KEY, "grokSelectors");
});

test("a non-blank override replaces its default, a blank one is ignored, and a broken one is reported and not used", () => {
  const page = makePage("<body></body>", "https://grok.com/");
  const S = XS.resolve({ copyButton: " button.copy ", answer: "div[", input: "  ", doneMarker: "gone" }, page.doc);
  assert.equal(S.copyButton, "button.copy");
  assert.equal(S.answer, '[data-testid="assistant-message"]', "the broken selector fell back to its default");
  assert.equal(S.input, '[data-testid="chat-input"] .ProseMirror[contenteditable="true"]');
  assert.equal(S.doneMarker, undefined, "a stored override of a field that no longer exists is ignored");
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
