/**
 * The part of the content script every provider shares. In a tab the service
 * worker opened for a queue item it waits for the chat box, lets the provider
 * adapter prepare the page (DeepSeek turns its modes on), types the prompt,
 * sends it, waits until the provider says the new answer is complete, presses
 * the Copy button under it and reports the text back. In any other tab of the
 * provider (the Creator's own) the worker has no work for it and it only
 * answers the settings page's selector check.
 *
 * What differs per provider is described by an adapter definition (see
 * src/providers/<id>/adapter.js): its selectors, `turns` (the answer turns on
 * the page), `isDone` (is this answer complete), `findCopy`, and optionally
 * `prepare`, `findSendButton`, `readFromPage` and `probe`.
 *
 * Every element it cannot find ends the run with `ui_changed` naming that
 * element, so a redesign shows up as a clear error, never as a wrong answer.
 * Each step is logged to the tab's console with the prefix "ai-bridge:".
 * `create` takes its environment (document, window, report, selector
 * overrides) as arguments so node tests can run it on a fake page.
 */
(function (root) {
  const P = root.AiBridgeProtocol || (typeof require === "function" ? require("../protocol.js") : null);
  const A = root.AiBridgeAnswer || (typeof require === "function" ? require("./answer.js") : null);

  const DEFAULT_TIMING = {
    READY_TRIES: 5,
    READY_RETRY_MS: 1000,
    /** How long the chat box may take to appear before the run says the Creator is not signed in. */
    INPUT_WAIT_MS: 30 * 1000,
    PAGE_WAIT_MS: 20 * 1000,
    SEND_ENABLE_MS: 3 * 1000,
    SEND_CHECK_MS: 5 * 1000,
    TOGGLE_SETTLE_MS: 400,
    TYPE_SETTLE_MS: 300,
    COPY_WAIT_MS: 3000,
    ARM_SETTLE_MS: 50,
    TICK_MS: 1000,
    /** The wait for the answer looks again on a page change, but not more often than this. */
    MIN_CHECK_GAP_MS: 300,
    HIGHLIGHT_MS: 4000,
    /** How often the wait for the answer logs what it sees. */
    DIAGNOSE_EVERY_MS: 15 * 1000,
  };

  /** Thrown inside the run to end it with an error code. */
  class RunError extends Error {
    constructor(code, detail) {
      super(detail || code);
      this.code = code;
      this.detail = detail || "";
    }
  }

  /** The text an element shows; innerText when the page has layout, else textContent. */
  function textOf(el) {
    return typeof el.innerText === "string" ? el.innerText : el.textContent || "";
  }

  /**
   * Builds the runner for one provider. `env` is `{document, window, report(message),
   * readOverrides(): Promise<object>, log(...args), timing}`.
   */
  function create(def, env) {
    const doc = env.document;
    const win = env.window;
    const T = Object.assign({}, DEFAULT_TIMING, env.timing || {});
    const report = env.report;
    const log = env.log || function () {};
    let cancelled = false;

    function wait(ms) {
      return new Promise(function (resolve) {
        setTimeout(resolve, ms);
      });
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

    /** querySelectorAll that reports a broken selector as the field it came from. */
    function queryAll(selector, field, scope) {
      try {
        return Array.from((scope || doc).querySelectorAll(selector));
      } catch (err) {
        throw new RunError("ui_changed", "selector \"" + field + "\" sai cú pháp: " + selector);
      }
    }

    const ctx = { document: doc, window: win, A: A, timing: T, log: log, wait: wait, waitFor: waitFor, queryAll: queryAll, textOf: textOf, RunError: RunError };

    async function loadSelectors() {
      const S = def.selectors.resolve(await env.readOverrides(), doc);
      if (S.invalid.length) log("override không dùng được, dùng mặc định:", S.invalid.join(", "));
      return S;
    }

    async function askForWork() {
      for (let i = 0; i < T.READY_TRIES; i += 1) {
        const work = await report({ kind: "ready", provider: def.id });
        if (work) return work;
        await wait(T.READY_RETRY_MS);
      }
      return null;
    }

    function findInput(S) {
      return queryAll(S.input, "input")[0] || null;
    }

    function isEditable(el) {
      if (el.isContentEditable === true) return true;
      const attr = el.getAttribute("contenteditable");
      return attr !== null && attr !== "false";
    }

    /** Puts the prompt in the chat box the way typing would, so the page's own state sees it. */
    function typeInto(input, text) {
      input.focus();
      if (isEditable(input)) {
        doc.execCommand("selectAll", false);
        doc.execCommand("insertText", false, text);
        return;
      }
      const setter = Object.getOwnPropertyDescriptor(win.HTMLTextAreaElement.prototype, "value").set;
      setter.call(input, text);
      input.dispatchEvent(new win.Event("input", { bubbles: true }));
    }

    function inputText(input) {
      return isEditable(input) ? textOf(input) : input.value;
    }

    function pressEnter(input) {
      const init = { key: "Enter", code: "Enter", keyCode: 13, which: 13, bubbles: true, cancelable: true };
      input.dispatchEvent(new win.KeyboardEvent("keydown", init));
      input.dispatchEvent(new win.KeyboardEvent("keypress", init));
      input.dispatchEvent(new win.KeyboardEvent("keyup", init));
    }

    /** The send button: the adapter's own rule when it has one, else the sendButton selector. */
    function findSendButton(S, input) {
      if (def.findSendButton) return def.findSendButton(S, input, ctx);
      return S.sendButton ? queryAll(S.sendButton, "sendButton")[0] || null : null;
    }

    function isEnabled(button) {
      return !button.disabled && button.getAttribute("aria-disabled") !== "true";
    }

    async function send(S, input) {
      const sent = function () {
        const current = findInput(S);
        return !current || inputText(current).trim() === "";
      };
      const button = await waitFor(function () {
        const found = findSendButton(S, input);
        return found && isEnabled(found) ? found : null;
      }, T.SEND_ENABLE_MS);
      if (button) {
        log("bấm nút gửi:", button.outerHTML.slice(0, 200));
        button.click();
        if (await waitFor(sent, T.SEND_CHECK_MS)) return;
      } else {
        log("không thấy nút gửi (hoặc đang bị khoá), thử Enter");
      }
      pressEnter(input);
      if (await waitFor(sent, T.SEND_CHECK_MS)) return;
      throw new RunError("ui_changed", button ? "gửi prompt (đã bấm nút gửi và Enter)" : "nút gửi");
    }

    /** The answer to this run: the last turn when there are more turns than before the prompt was sent, else null. */
    function findAnswer(S, countBefore) {
      const turns = def.turns(S, ctx);
      return turns.length > countBefore ? turns[turns.length - 1] : null;
    }

    function copyViaButton(button) {
      return new Promise(function (resolve) {
        function onMessage(event) {
          const data = event.data;
          if (event.source !== win || !data || data.source !== P.SOURCE || data.kind !== "copied") return;
          win.removeEventListener("message", onMessage);
          clearTimeout(timer);
          resolve(data.text);
        }
        const timer = setTimeout(function () {
          win.removeEventListener("message", onMessage);
          resolve("");
        }, T.COPY_WAIT_MS);
        win.addEventListener("message", onMessage);
        win.postMessage({ source: P.SOURCE, kind: "arm-copy" }, win.location.origin);
        // The hook in the page's own world arms itself on this message, which arrives after the current task: click only then.
        setTimeout(function () {
          button.click();
        }, T.ARM_SETTLE_MS);
      });
    }

    function checkStillRunning(deadline) {
      if (cancelled) throw new RunError("cancelled");
      if (Date.now() >= deadline) throw new RunError("timeout");
    }

    /**
     * Waits until the page changes (the answer typing, its action row appearing)
     * or `maxMs` passes, so a finished answer is noticed at once instead of at
     * the next tick. Changes closer together than MIN_CHECK_GAP_MS count once.
     */
    function waitForPageChange(maxMs) {
      return new Promise(function (resolve) {
        const startedAt = Date.now();
        const observer = new win.MutationObserver(function () {
          if (Date.now() - startedAt >= T.MIN_CHECK_GAP_MS) done();
        });
        const timer = setTimeout(done, maxMs);
        function done() {
          observer.disconnect();
          clearTimeout(timer);
          resolve();
        }
        observer.observe(doc.body, { childList: true, subtree: true, characterData: true });
      });
    }

    async function waitForAnswer(S, countBefore, deadline) {
      let lastText = "";
      let changedAt = Date.now();
      let loggedAt = 0;
      for (;;) {
        checkStillRunning(deadline);
        const turn = findAnswer(S, countBefore);
        const text = turn ? textOf(turn) : "";
        if (text !== lastText) {
          lastText = text;
          changedAt = Date.now();
        }
        if (Date.now() - loggedAt >= T.DIAGNOSE_EVERY_MS) {
          loggedAt = Date.now();
          log(
            "đang chờ:",
            "lượt trả lời=" + def.turns(S, ctx).length + " (trước khi gửi " + countBefore + ")",
            "độ dài câu trả lời=" + text.length,
            "đứng yên " + Math.round((Date.now() - changedAt) / 1000) + "s",
            "tab " + (doc.hidden ? "nền" : "đang xem"),
          );
        }
        if (turn && def.isDone(S, turn, { text: text, changedAt: changedAt, now: Date.now() }, ctx)) {
          log("câu trả lời xong:", text.length, "ký tự");
          return turn;
        }
        await waitForPageChange(T.TICK_MS);
      }
    }

    async function run(work) {
      const S = await loadSelectors();
      log("bắt đầu lượt", work.requestId, "chế độ", JSON.stringify(work.modes));
      const input = await waitFor(function () {
        return findInput(S);
      }, T.INPUT_WAIT_MS);
      if (!input) throw new RunError("not_logged_in");
      log("ô chat:", input.outerHTML.slice(0, 200));
      report({ kind: "progress", requestId: work.requestId, stage: "sending" });
      if (def.prepare) await def.prepare(S, work, ctx);
      const countBefore = def.turns(S, ctx).length;
      typeInto(input, work.prompt);
      await wait(T.TYPE_SETTLE_MS);
      if (inputText(input).trim() === "") throw new RunError("ui_changed", "ghi prompt vào ô chat");
      log("đã ghi prompt,", work.prompt.length, "ký tự");
      await send(S, input);
      log("đã gửi");

      report({ kind: "progress", requestId: work.requestId, stage: "answering" });
      const turn = await waitForAnswer(S, countBefore, work.deadline);

      report({ kind: "progress", requestId: work.requestId, stage: "reading" });
      const button = def.findCopy(S, turn, ctx);
      const copied = button ? await copyViaButton(button) : "";
      let text = copied;
      if (text.trim() === "" && def.readFromPage) text = def.readFromPage(turn, ctx);
      log(copied.trim() !== "" ? "lấy câu trả lời qua nút Copy" : "lấy câu trả lời từ trang");
      if (text.trim() === "") throw new RunError("ui_changed", button ? "nội dung câu trả lời (nút Copy không đưa văn bản)" : "nút Copy");
      return text;
    }

    /** Draws an outline on matched elements for a few seconds, so the Creator sees what a selector found. */
    function highlight(elements, color) {
      elements.forEach(function (el) {
        const before = el.style.outline;
        el.style.outline = "3px solid " + color;
        setTimeout(function () {
          el.style.outline = before;
        }, T.HIGHLIGHT_MS);
      });
    }

    /** The settings page's check: what each selector finds on this tab, outlined in red. */
    function probe(overrides) {
      const S = def.selectors.resolve(overrides, doc);
      const result = {};
      function one(key, find, color) {
        try {
          const found = find();
          result[key] = { count: found.length, sample: found[0] ? found[0].outerHTML.slice(0, 160) : "" };
          highlight(found, color || "#e11d48");
        } catch (err) {
          result[key] = { error: String(err && err.message ? err.message : err) };
        }
      }
      def.probe(S, ctx, one, result);
      result.invalid = S.invalid;
      return result;
    }

    function cancel() {
      cancelled = true;
    }

    return { run, probe, cancel, askForWork, typeInto, findInput };
  }

  // ---- helpers for providers described by plain selectors (Gemini, Grok) ----

  const markers = {
    /** Every answer turn on the page, oldest first. */
    turns: function (S, ctx) {
      return ctx.queryAll(S.answer, "answer");
    },
    /**
     * Complete when the last turn holds the done marker (and the settled marker when
     * one is set) and has text. The caller only offers a turn that is newer than the prompt.
     */
    isDone: function (S, turn, progress, ctx) {
      if (progress.text.trim() === "") return false;
      if (ctx.queryAll(S.doneMarker, "doneMarker", turn).length === 0) return false;
      if (S.settledMarker && ctx.queryAll(S.settledMarker, "settledMarker", turn).length === 0) return false;
      return true;
    },
    /** The Copy button inside the turn. */
    findCopy: function (S, turn, ctx) {
      return ctx.queryAll(S.copyButton, "copyButton", turn)[0] || null;
    },
    /** The settings page's check for a provider described by plain selectors. */
    probe: function (S, ctx, one) {
      one("input", function () {
        return ctx.queryAll(S.input, "input").slice(0, 1);
      });
      one("sendButton", function () {
        return ctx.queryAll(S.sendButton, "sendButton").slice(0, 1);
      });
      one("answer", function () {
        return ctx.queryAll(S.answer, "answer").slice(-1);
      });
      const last = ctx.queryAll(S.answer, "answer").slice(-1)[0];
      ["doneMarker", "settledMarker", "copyButton"].forEach(function (key) {
        if (!S[key]) return;
        one(key, function () {
          return last ? ctx.queryAll(S[key], key, last).slice(0, 1) : [];
        }, key === "copyButton" ? "#2563eb" : "#e11d48");
      });
    },
  };

  /**
   * Wires an adapter definition to its page: answers the worker's cancel and the
   * settings page's probe, and runs the work the worker hands out.
   */
  function start(def) {
    const core = create(def, {
      document: document,
      window: window,
      report: function (message) {
        return chrome.runtime.sendMessage(message);
      },
      readOverrides: async function () {
        const stored = await chrome.storage.sync.get(def.selectors.STORAGE_KEY);
        return stored[def.selectors.STORAGE_KEY];
      },
      log: function () {
        console.info.apply(console, ["ai-bridge:"].concat(Array.from(arguments)));
      },
    });

    chrome.runtime.onMessage.addListener(function (message, sender, sendResponse) {
      if (!message) return false;
      if (message.kind === "cancel") core.cancel();
      if (message.kind === "probe") sendResponse(core.probe(message.selectors));
      return false;
    });

    core.askForWork().then(
      function (work) {
        if (!work) return;
        core.run(work).then(
          function (text) {
            console.info("ai-bridge: xong,", text.length, "ký tự");
            chrome.runtime.sendMessage({ kind: "result", requestId: work.requestId, text: text });
          },
          function (err) {
            if (err instanceof RunError && err.code === "cancelled") return;
            const code = err instanceof RunError ? err.code : "ui_changed";
            const detail = err instanceof RunError ? err.detail : String(err && err.message ? err.message : err);
            console.error("ai-bridge: lỗi", code, detail);
            chrome.runtime.sendMessage({ kind: "error", requestId: work.requestId, code: code, detail: detail });
          },
        );
      },
      function (err) {
        console.error("ai-bridge: could not reach the extension worker:", err);
      },
    );
  }

  const api = { DEFAULT_TIMING, RunError, textOf, create, markers, start };
  root.AiBridgeAdapterCore = api;
  if (typeof module !== "undefined" && module.exports) module.exports = api;
})(typeof globalThis !== "undefined" ? globalThis : self);
