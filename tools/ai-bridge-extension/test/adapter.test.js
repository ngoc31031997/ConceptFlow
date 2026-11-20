const test = require("node:test");
const assert = require("node:assert/strict");
const P = require("../src/protocol.js");
const A = require("../src/adapter/answer.js");
const core = require("../src/adapter/core.js");
const gemini = require("../src/providers/gemini/adapter.js");
const grok = require("../src/providers/grok/adapter.js");
const deepseek = require("../src/providers/deepseek/adapter.js");
const { makePage, copyOnClick, FAST } = require("./helpers/page.js");

const ANSWER = 'Kết quả:\n\n```json\n{"ok": true}\n```';

const GEMINI_HTML = `<body>
<main id="chat">
  <model-response id="old"><message-content><div class="markdown" aria-busy="false">câu trả lời cũ</div></message-content>
    <div class="response-footer complete"><message-actions><copy-button><button id="old-copy">copy</button></copy-button></message-actions></div></model-response>
</main>
<rich-textarea><div class="ql-editor" contenteditable="true"></div></rich-textarea>
<div data-test-id="send-button-container"><span class="send-button submit"><button id="send">gửi</button></span></div>
</body>`;

const GROK_HTML = `<body>
<main id="chat">
  <div data-testid="assistant-message"><p>câu trả lời cũ</p>
    <div data-testid="response-footer-actions"><button aria-label="Copy response">copy</button></div></div>
</main>
<div data-testid="chat-input"><div class="ProseMirror" contenteditable="true"></div></div>
<button data-testid="chat-submit" id="send">gửi</button>
</body>`;

/** Puts a new Gemini answer in the chat; busy first, complete with its footer after `finishAfterMs`. */
function geminiAnswers(page, finishAfterMs, copyText) {
  const turn = page.doc.createElement("model-response");
  turn.innerHTML = '<message-content><div class="markdown" aria-busy="true">đang viết</div></message-content>';
  page.doc.getElementById("chat").append(turn);
  setTimeout(() => {
    turn.querySelector(".markdown").setAttribute("aria-busy", "false");
    turn.querySelector(".markdown").textContent = "xong";
    turn.insertAdjacentHTML("beforeend", '<div class="response-footer complete"><message-actions><copy-button><button>copy</button></copy-button></message-actions></div>');
    copyOnClick(page, turn.querySelector("copy-button button"), copyText, P.SOURCE);
  }, finishAfterMs);
}

function grokAnswers(page, finishAfterMs, copyText) {
  const turn = page.doc.createElement("div");
  turn.setAttribute("data-testid", "assistant-message");
  turn.innerHTML = "<p>đang viết</p>";
  page.doc.getElementById("chat").append(turn);
  setTimeout(() => {
    turn.querySelector("p").textContent = "xong";
    turn.insertAdjacentHTML("beforeend", '<div data-testid="response-footer-actions"><button aria-label="Copy response">copy</button></div>');
    copyOnClick(page, turn.querySelector("button"), copyText, P.SOURCE);
  }, finishAfterMs);
}

function runnerFor(def, page, overrides) {
  const reports = [];
  const runner = core.create(def, {
    document: page.doc,
    window: page.win,
    report: (message) => {
      reports.push(message);
    },
    readOverrides: async () => overrides || {},
    timing: FAST,
  });
  return { runner, reports };
}

const work = (prompt) => ({ requestId: "queue-i1", prompt: prompt, modes: { deepThink: false, search: false }, deadline: Date.now() + 10000 });

test("a Gemini run types the prompt, sends it, waits for the new answer to complete and copies it", async () => {
  const page = makePage(GEMINI_HTML, "https://gemini.google.com/app");
  const sentPrompts = [];
  page.doc.getElementById("send").addEventListener("click", () => {
    sentPrompts.push(page.doc.querySelector(".ql-editor").textContent);
    page.doc.querySelector(".ql-editor").textContent = "";
    geminiAnswers(page, 150, ANSWER);
  });
  const { runner, reports } = runnerFor(gemini, page);
  const text = await runner.run(work("viết cho tôi"));
  assert.equal(text, ANSWER);
  assert.deepEqual(sentPrompts, ["viết cho tôi"]);
  assert.deepEqual(reports.map((r) => r.stage), ["sending", "answering", "reading"]);
  page.close();
});

test("a Gemini answer is not taken from the turn that was there before the prompt was sent", async () => {
  const page = makePage(GEMINI_HTML, "https://gemini.google.com/app");
  page.doc.getElementById("send").addEventListener("click", () => {
    page.doc.querySelector(".ql-editor").textContent = "";
    // The new turn only shows up after a while; the old complete turn must not count as the answer.
    setTimeout(() => geminiAnswers(page, 100, ANSWER), 200);
  });
  const oldCopy = copyOnClick(page, page.doc.getElementById("old-copy"), "CŨ", P.SOURCE);
  const { runner } = runnerFor(gemini, page);
  assert.equal(await runner.run(work("p")), ANSWER);
  assert.equal(oldCopy.count, 0, "the old turn's Copy button was never pressed");
  page.close();
});

test("a Grok run waits for the footer actions of the new turn and copies through its Copy button", async () => {
  const page = makePage(GROK_HTML, "https://grok.com/");
  page.doc.getElementById("send").addEventListener("click", () => {
    page.doc.querySelector(".ProseMirror").textContent = "";
    grokAnswers(page, 150, ANSWER);
  });
  const { runner, reports } = runnerFor(grok, page);
  assert.equal(await runner.run(work("p")), ANSWER);
  assert.deepEqual(reports.map((r) => r.stage), ["sending", "answering", "reading"]);
  page.close();
});

test("a send button that stays locked falls back to Enter", async () => {
  const page = makePage(GROK_HTML, "https://grok.com/");
  page.doc.getElementById("send").disabled = true;
  page.doc.querySelector(".ProseMirror").addEventListener("keydown", (event) => {
    if (event.key !== "Enter") return;
    page.doc.querySelector(".ProseMirror").textContent = "";
    grokAnswers(page, 50, "bằng Enter");
  });
  const { runner } = runnerFor(grok, page);
  assert.equal(await runner.run(work("p")), "bằng Enter");
  page.close();
});

test("without a chat box the run says the Creator is not signed in, naming the provider", async () => {
  const page = makePage("<body><p>Sign in</p></body>", "https://gemini.google.com/app");
  const { runner } = runnerFor(gemini, page);
  await assert.rejects(runner.run(work("p")), (err) => {
    assert.equal(err.code, "not_logged_in");
    assert.match(P.errorMessage(err.code, err.detail, { name: gemini.name, host: "gemini.google.com" }), /Chưa đăng nhập Gemini/);
    return true;
  });
  page.close();
});

test("a Copy button that hands over nothing ends the run with ui_changed, not a wrong answer", async () => {
  const page = makePage(GROK_HTML, "https://grok.com/");
  page.doc.getElementById("send").addEventListener("click", () => {
    page.doc.querySelector(".ProseMirror").textContent = "";
    grokAnswers(page, 50, "");
  });
  const { runner } = runnerFor(grok, page);
  await assert.rejects(runner.run(work("p")), (err) => err.code === "ui_changed" && /nội dung câu trả lời/.test(err.detail));
  page.close();
});

test("a cancelled run stops while it waits for the answer", async () => {
  const page = makePage(GEMINI_HTML, "https://gemini.google.com/app");
  page.doc.getElementById("send").addEventListener("click", () => {
    page.doc.querySelector(".ql-editor").textContent = "";
    const turn = page.doc.createElement("model-response");
    turn.innerHTML = '<message-content><div aria-busy="true">đang viết</div></message-content>';
    page.doc.getElementById("chat").append(turn);
  });
  const { runner } = runnerFor(gemini, page);
  setTimeout(() => runner.cancel(), 300);
  await assert.rejects(runner.run(work("p")), (err) => err.code === "cancelled");
  page.close();
});

test("a run that passes its deadline ends with timeout", async () => {
  const page = makePage(GROK_HTML, "https://grok.com/");
  page.doc.getElementById("send").addEventListener("click", () => {
    page.doc.querySelector(".ProseMirror").textContent = "";
    const turn = page.doc.createElement("div");
    turn.setAttribute("data-testid", "assistant-message");
    turn.textContent = "đang viết";
    page.doc.getElementById("chat").append(turn);
  });
  const { runner } = runnerFor(grok, page);
  await assert.rejects(runner.run(Object.assign(work("p"), { deadline: Date.now() + 300 })), (err) => err.code === "timeout");
  page.close();
});

test("Gemini is done only with the footer complete and its content no longer busy", () => {
  const page = makePage(GEMINI_HTML, "https://gemini.google.com/app");
  const S = gemini.selectors.resolve({}, page.doc);
  const ctx = { queryAll: (selector, field, scope) => Array.from((scope || page.doc).querySelectorAll(selector)) };
  const progress = { text: "có chữ", changedAt: 0, now: 0 };
  const turn = (html) => {
    const el = page.doc.createElement("model-response");
    el.innerHTML = html;
    return el;
  };
  const footer = '<div class="response-footer complete"></div>';
  assert.equal(gemini.isDone(S, turn('<message-content><div aria-busy="false">x</div></message-content>' + footer), progress, ctx), true);
  assert.equal(gemini.isDone(S, turn('<message-content><div aria-busy="true">x</div></message-content>' + footer), progress, ctx), false, "still busy");
  assert.equal(gemini.isDone(S, turn('<message-content><div aria-busy="false">x</div></message-content><div class="response-footer"></div>'), progress, ctx), false, "footer not complete");
  assert.equal(gemini.isDone(S, turn('<message-content><div aria-busy="false">x</div></message-content>' + footer), { text: " ", changedAt: 0, now: 0 }, ctx), false, "no text");
  page.close();
});

test("Grok is done only when the turn has its footer actions", () => {
  const page = makePage(GROK_HTML, "https://grok.com/");
  const S = grok.selectors.resolve({}, page.doc);
  const ctx = { queryAll: (selector, field, scope) => Array.from((scope || page.doc).querySelectorAll(selector)) };
  const progress = { text: "có chữ", changedAt: 0, now: 0 };
  const turn = (html) => {
    const el = page.doc.createElement("div");
    el.innerHTML = html;
    return el;
  };
  assert.equal(grok.isDone(S, turn('<p>x</p><div data-testid="response-footer-actions"></div>'), progress, ctx), true);
  assert.equal(grok.isDone(S, turn("<p>x</p>"), progress, ctx), false);
  page.close();
});

test("the settings check outlines what a provider's selectors find on the page", () => {
  const page = makePage(GROK_HTML, "https://grok.com/");
  const { runner } = runnerFor(grok, page);
  const probe = runner.probe({});
  assert.equal(probe.input.count, 1);
  assert.equal(probe.sendButton.count, 1);
  assert.equal(probe.answer.count, 1);
  assert.equal(probe.doneMarker.count, 1);
  assert.equal(probe.copyButton.count, 1);
  assert.equal(probe.settledMarker, undefined, "an unused selector is not checked");
  assert.deepEqual(probe.invalid, []);
  assert.equal(runner.probe({ sendButton: "div[" }).sendButton.error === undefined, true, "a broken override falls back to the default");
  assert.deepEqual(runner.probe({ sendButton: "div[" }).invalid, ["sendButton"]);
  page.close();
});

const DEEPSEEK_HTML = `<body>
<div class="chat">
  <div class="msg"><div class="think-panel"><div class="ds-markdown" id="reason">suy nghĩ</div></div>
    <div class="ds-markdown" id="answer">trả lời</div>
    <div class="ds-icon-button" id="copy"></div></div>
</div>
<div class="box"><textarea id="chat-input"></textarea>
  <div role="button" id="dt"><span>DeepThink</span></div><div role="button" id="se"><span>Search</span></div></div>
</body>`;

test("DeepSeek's answer turns leave out the DeepThink reasoning and the Copy button is the icon after the answer", () => {
  const page = makePage(DEEPSEEK_HTML, "https://chat.deepseek.com/");
  const S = deepseek.selectors.resolve({});
  const ctx = { queryAll: (selector, field, scope) => Array.from((scope || page.doc).querySelectorAll(selector)), textOf: core.textOf };
  const turns = deepseek.turns(S, ctx);
  assert.deepEqual(turns.map((t) => t.id), ["answer"]);
  assert.equal(deepseek.findCopy(S, turns[0], ctx).id, "copy");
  assert.equal(deepseek.isDone(S, turns[0], { text: "trả lời", changedAt: 0, now: A.STABLE_MS }, ctx), true);
  assert.equal(deepseek.isDone(S, turns[0], { text: "trả lời", changedAt: 0, now: A.STABLE_MS - 1 }, ctx), false);
  page.close();
});

test("DeepSeek's modes are set from the item and only clicked when they differ", async () => {
  const page = makePage(DEEPSEEK_HTML, "https://chat.deepseek.com/");
  const clicks = [];
  ["dt", "se"].forEach((id) => {
    const el = page.doc.getElementById(id);
    el.addEventListener("click", () => {
      clicks.push(id);
      el.classList.toggle("selected");
    });
  });
  page.doc.getElementById("se").classList.add("selected");
  const S = deepseek.selectors.resolve({});
  const ctx = {
    queryAll: (selector, field, scope) => Array.from((scope || page.doc).querySelectorAll(selector)),
    textOf: core.textOf,
    waitFor: async (find) => find(),
    wait: async () => {},
    log: () => {},
    timing: FAST,
    RunError: core.RunError,
  };
  await deepseek.prepare(S, { modes: { deepThink: true, search: false } }, ctx);
  assert.deepEqual(clicks, ["dt", "se"], "DeepThink was off and Search was on, so both were clicked");
  await deepseek.prepare(S, { modes: { deepThink: true, search: false } }, ctx);
  assert.deepEqual(clicks, ["dt", "se"], "already in the wanted state, so nothing was clicked again");
  page.close();
});
