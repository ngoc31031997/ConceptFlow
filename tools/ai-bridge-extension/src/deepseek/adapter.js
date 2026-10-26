/**
 * Content script on chat.deepseek.com. In a tab the service worker opened for
 * a run it sets DeepThink and Search as the run asks, sends the prompt, waits for the
 * answer to finish and reports it back. In any other DeepSeek tab (the
 * Creator's own) the worker has no work for it and it does nothing.
 *
 * Every element it cannot find ends the run with `ui_changed` naming that
 * element, so a DeepSeek redesign shows up as a clear error, never as a wrong
 * answer. Selectors live in selectors.js.
 */
(function () {
  const P = globalThis.AiBridgeProtocol;
  const S = globalThis.DEEPSEEK_SELECTORS;
  const A = globalThis.DeepSeekAnswer;

  const READY_TRIES = 5;
  const READY_RETRY_MS = 1000;
  const INPUT_WAIT_MS = 20 * 1000;
  const SEND_CHECK_MS = 10 * 1000;
  const COPY_WAIT_MS = 3000;
  const TICK_MS = 1000;

  /** Thrown inside the run to end it with an error code. */
  class RunError extends Error {
    constructor(code, detail) {
      super(detail || code);
      this.code = code;
      this.detail = detail || "";
    }
  }

  let cancelled = false;

  function wait(ms) {
    return new Promise(function (resolve) {
      setTimeout(resolve, ms);
    });
  }

  function report(message) {
    return chrome.runtime.sendMessage(message);
  }

  async function askForWork() {
    for (let i = 0; i < READY_TRIES; i += 1) {
      const work = await report({ kind: "ready" });
      if (work) return work;
      await wait(READY_RETRY_MS);
    }
    return null;
  }

  async function waitFor(find, timeoutMs) {
    const until = Date.now() + timeoutMs;
    for (;;) {
      const found = find();
      if (found) return found;
      if (Date.now() >= until) return null;
      await wait(250);
    }
  }

  function findInput() {
    return document.querySelector(S.input);
  }

  function findToggle(labels) {
    const all = Array.from(document.querySelectorAll(S.clickable));
    return all.find(function (el) {
      return A.hasLabel(el.innerText, labels);
    }) || null;
  }

  function isOn(el) {
    if (el.getAttribute("aria-pressed") === "true") return true;
    return S.selectedClass.test(el.getAttribute("class") || "");
  }

  /** Puts a mode toggle in the wanted state, clicking it only when it is not there yet. */
  function setMode(labels, name, wanted) {
    const toggle = findToggle(labels);
    if (!toggle) throw new RunError("ui_changed", "nút " + name);
    if (isOn(toggle) !== wanted) toggle.click();
  }

  /** Sets the chat box the way a typing user would, so DeepSeek's React state sees the prompt. */
  function typeInto(input, text) {
    const setter = Object.getOwnPropertyDescriptor(HTMLTextAreaElement.prototype, "value").set;
    setter.call(input, text);
    input.dispatchEvent(new Event("input", { bubbles: true }));
  }

  function pressEnter(input) {
    const init = { key: "Enter", code: "Enter", keyCode: 13, which: 13, bubbles: true, cancelable: true };
    input.dispatchEvent(new KeyboardEvent("keydown", init));
    input.dispatchEvent(new KeyboardEvent("keyup", init));
  }

  function answerBlocks() {
    return Array.from(document.querySelectorAll(S.markdown)).filter(function (el) {
      return !el.closest(S.thinking);
    });
  }

  /** The answer to this run: the last markdown block that was not on the page before the prompt was sent. */
  function findAnswer(countBefore) {
    const blocks = answerBlocks();
    return blocks.length > countBefore ? blocks[blocks.length - 1] : null;
  }

  /** Copy, the first icon button after the answer in its own message (never the chat box's controls). */
  function findCopyButton(answer) {
    const input = findInput();
    let scope = answer.parentElement;
    for (let depth = 0; scope && depth < 6; depth += 1, scope = scope.parentElement) {
      if (input && scope.contains(input)) return null;
      const button = Array.from(scope.querySelectorAll(S.iconButton)).find(function (el) {
        return !answer.contains(el) && !el.closest(S.thinking) && (answer.compareDocumentPosition(el) & Node.DOCUMENT_POSITION_FOLLOWING) !== 0;
      });
      if (button) return button;
    }
    return null;
  }

  /** The answer read from the page, with code blocks fenced again; used when the Copy button gives nothing. */
  function readFromPage(answer) {
    const blocks = Array.from(answer.children).map(function (child) {
      const pre = child.matches("pre") ? child : child.querySelector("pre");
      if (!pre) return { kind: "text", text: child.innerText };
      const label = child.querySelector('[class*="banner"] span, [class*="lang"]');
      return { kind: "code", lang: label ? label.textContent.trim() : "", code: pre.textContent };
    });
    return A.joinBlocks(blocks);
  }

  function copyViaButton(button) {
    return new Promise(function (resolve) {
      function onMessage(event) {
        const data = event.data;
        if (event.source !== window || !data || data.source !== P.SOURCE || data.kind !== "copied") return;
        window.removeEventListener("message", onMessage);
        clearTimeout(timer);
        resolve(data.text);
      }
      const timer = setTimeout(function () {
        window.removeEventListener("message", onMessage);
        resolve("");
      }, COPY_WAIT_MS);
      window.addEventListener("message", onMessage);
      window.postMessage({ source: P.SOURCE, kind: "arm-copy" }, window.location.origin);
      button.click();
    });
  }

  function checkStillRunning(deadline) {
    if (cancelled) throw new RunError("cancelled");
    if (Date.now() >= deadline) throw new RunError("timeout");
  }

  async function waitForAnswer(countBefore, deadline) {
    let lastText = "";
    let changedAt = Date.now();
    for (;;) {
      checkStillRunning(deadline);
      const answer = findAnswer(countBefore);
      const text = answer ? answer.innerText : "";
      if (text !== lastText) {
        lastText = text;
        changedAt = Date.now();
      }
      if (answer && A.isFinished(text, changedAt, Date.now(), findCopyButton(answer) !== null)) return answer;
      await wait(TICK_MS);
    }
  }

  async function run(work) {
    const input = await waitFor(findInput, INPUT_WAIT_MS);
    if (!input) {
      if (S.signInPath.test(window.location.pathname)) throw new RunError("not_logged_in");
      throw new RunError("ui_changed", "ô chat");
    }
    report({ kind: "progress", requestId: work.requestId, stage: "sending" });
    setMode(S.deepThinkLabels, "DeepThink", work.modes.deepThink);
    setMode(S.searchLabels, "Search", work.modes.search);
    const countBefore = answerBlocks().length;
    typeInto(input, work.prompt);
    await wait(300);
    pressEnter(input);
    const sent = await waitFor(function () {
      return findInput() && findInput().value === "";
    }, SEND_CHECK_MS);
    if (!sent) throw new RunError("ui_changed", "gửi prompt");

    report({ kind: "progress", requestId: work.requestId, stage: "answering" });
    const answer = await waitForAnswer(countBefore, work.deadline);

    report({ kind: "progress", requestId: work.requestId, stage: "reading" });
    const button = findCopyButton(answer);
    const copied = button ? await copyViaButton(button) : "";
    const text = copied.trim() !== "" ? copied : readFromPage(answer);
    if (text.trim() === "") throw new RunError("ui_changed", "nội dung câu trả lời");
    return text;
  }

  chrome.runtime.onMessage.addListener(function (message) {
    if (message && message.kind === "cancel") cancelled = true;
  });

  askForWork().then(function (work) {
    if (!work) return;
    run(work).then(
      function (text) {
        report({ kind: "result", requestId: work.requestId, text: text });
      },
      function (err) {
        if (err instanceof RunError && err.code === "cancelled") return;
        const code = err instanceof RunError ? err.code : "ui_changed";
        const detail = err instanceof RunError ? err.detail : String(err && err.message ? err.message : err);
        report({ kind: "error", requestId: work.requestId, code: code, detail: detail });
      },
    );
  }, function (err) {
    console.error("ai-bridge: could not reach the extension worker:", err);
  });
})();
