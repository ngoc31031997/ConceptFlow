/**
 * The content script every provider runs. In a tab the service worker opened
 * for a queue item it waits for the chat box, types the prompt, sends it, waits
 * until the new answer is complete, presses its Copy button and reports the
 * text back. In any other tab of the provider (the Creator's own) the worker
 * has no work for it and it only answers the settings page's selector check.
 *
 * A provider is described only by four CSS selectors (see
 * src/providers/<id>/selectors.js and the settings page): the chat box, the
 * send button, an answer and the Copy button. Each is looked up on the whole
 * page. The answer to a run is the last answer once there are more answers
 * than before the prompt was sent; its Copy button is the first Copy match
 * inside that answer or after where it starts, so a button of an earlier
 * answer never counts. The answer is complete when its Copy button is there
 * and its text has stopped changing; with no Copy selector set, when its text
 * has stopped changing for longer, and it is then read from the page.
 *
 * Every element it cannot find ends the run with `ui_changed` naming that
 * element and its selector, so a redesign shows up as a clear error, never as
 * a wrong answer. Each step is logged to the tab's console with the prefix
 * "ai-bridge:". `create` takes its environment (document, window, report,
 * selector overrides) as arguments so node tests can run it on a fake page.
 */
(function (root) {
  const P = root.AiBridgeProtocol || (typeof require === "function" ? require("../protocol.js") : null);
  const A = root.AiBridgeAnswer || (typeof require === "function" ? require("./answer.js") : null);

  const DEFAULT_TIMING = {
    READY_TRIES: 5,
    READY_RETRY_MS: 1000,
    /** How long the chat box may take to appear before the run says the Creator is not signed in. */
    INPUT_WAIT_MS: 30 * 1000,
    SEND_ENABLE_MS: 3 * 1000,
    SEND_CHECK_MS: 5 * 1000,
    TYPE_SETTLE_MS: 300,
    COPY_WAIT_MS: 3000,
    ARM_SETTLE_MS: 50,
    TICK_MS: 1000,
    /** The wait for the answer looks again on a page change, but not more often than this. */
    MIN_CHECK_GAP_MS: 300,
    HIGHLIGHT_MS: 4000,
    /** How often the wait for the answer logs what it sees. */
    DIAGNOSE_EVERY_MS: 15 * 1000,
    /** How long the answer must stay unchanged with its Copy button shown to count as complete. */
    STABLE_MS: A.STABLE_MS,
    /** How long it must stay unchanged when no Copy selector is set. */
    STABLE_WITHOUT_COPY_MS: A.STABLE_WITHOUT_ACTIONS_MS,
    /**
     * How long an answer that has text may stay unchanged without its Copy
     * button before the run ends with `ui_changed` naming the Copy selector,
     * instead of waiting for the full run timeout.
     */
    STALLED_MS: 2 * 60 * 1000,
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

    /** The send button; with no selector set the prompt is sent with Enter. */
    function findSendButton(S) {
      return S.sendButton ? queryAll(S.sendButton, "sendButton")[0] || null : null;
    }

    function answers(S) {
      return queryAll(S.answer, "answer");
    }

    /** The Copy button of `answer`: the first Copy match inside it or after where it starts. */
    function findCopy(S, answer) {
      if (!S.copyButton) return null;
      const FOLLOWING = 4; // Node.DOCUMENT_POSITION_FOLLOWING
      return (
        queryAll(S.copyButton, "copyButton").find(function (el) {
          return answer.contains(el) || (answer.compareDocumentPosition(el) & FOLLOWING) !== 0;
        }) || null
      );
    }

    /** The answer read from the page, with code blocks fenced again; used when no Copy selector is set. */
    function readFromPage(answer) {
      const blocks = Array.from(answer.children).map(function (child) {
        const pre = child.matches("pre") ? child : child.querySelector("pre");
        if (!pre) return { kind: "text", text: textOf(child) };
        const label = child.querySelector('[class*="banner"] span, [class*="lang"]');
        return { kind: "code", lang: label ? label.textContent.trim() : "", code: pre.textContent };
      });
      return blocks.length ? A.joinBlocks(blocks) : textOf(answer).trim();
    }

    /** Why the new answer has no Copy button yet, naming the selector and what the page has. */
    function copyMissing(S) {
      const total = queryAll(S.copyButton, "copyButton").length;
      return (
        "không thấy \"Nút Copy\" (" + S.copyButton + ") ở câu trả lời mới (" + S.answer + ")" +
        (total > 0 ? "; trên trang có " + total + " nút khớp nhưng đều nằm trước câu trả lời mới" : "; trên trang không có nút nào khớp")
      );
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
        const found = findSendButton(S);
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
      throw new RunError("ui_changed", button ? "gửi prompt (đã bấm nút gửi và Enter)" : "nút gửi (" + (S.sendButton || "để trống, đã thử Enter") + ")");
    }

    /** The answer to this run: the last answer when there are more answers than before the prompt was sent, else null. */
    function findAnswer(S, countBefore) {
      const all = answers(S);
      return all.length > countBefore ? all[all.length - 1] : null;
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
        const answer = findAnswer(S, countBefore);
        const text = answer ? textOf(answer) : "";
        if (text !== lastText) {
          lastText = text;
          changedAt = Date.now();
        }
        const copy = answer ? findCopy(S, answer) : null;
        const quietMs = Date.now() - changedAt;
        const hasText = text.trim() !== "";
        if (hasText && (S.copyButton ? copy !== null && quietMs >= T.STABLE_MS : quietMs >= T.STABLE_WITHOUT_COPY_MS)) {
          log("câu trả lời xong:", text.length, "ký tự");
          return { answer: answer, copy: copy };
        }
        if (Date.now() - loggedAt >= T.DIAGNOSE_EVERY_MS) {
          loggedAt = Date.now();
          log(
            "đang chờ:",
            "câu trả lời=" + answers(S).length + " (trước khi gửi " + countBefore + ")",
            "độ dài=" + text.length,
            "đứng yên " + Math.round(quietMs / 1000) + "s",
            S.copyButton ? "nút Copy " + (copy ? "đã có" : "chưa có") : "không đặt nút Copy",
            "tab " + (doc.hidden ? "nền" : "đang xem"),
          );
        }
        if (S.copyButton && !copy && hasText && quietMs >= T.STALLED_MS) throw new RunError("ui_changed", copyMissing(S));
        await waitForPageChange(T.TICK_MS);
      }
    }

    async function run(work) {
      const S = await loadSelectors();
      log("bắt đầu lượt", work.requestId);
      const input = await waitFor(function () {
        return findInput(S);
      }, T.INPUT_WAIT_MS);
      if (!input) throw new RunError("not_logged_in");
      log("ô chat:", input.outerHTML.slice(0, 200));
      report({ kind: "progress", requestId: work.requestId, stage: "sending" });
      const countBefore = answers(S).length;
      typeInto(input, work.prompt);
      await wait(T.TYPE_SETTLE_MS);
      if (inputText(input).trim() === "") throw new RunError("ui_changed", "ghi prompt vào ô chat");
      log("đã ghi prompt,", work.prompt.length, "ký tự");
      await send(S, input);
      log("đã gửi");

      report({ kind: "progress", requestId: work.requestId, stage: "answering" });
      const done = await waitForAnswer(S, countBefore, work.deadline);

      report({ kind: "progress", requestId: work.requestId, stage: "reading" });
      if (!done.copy) {
        log("lấy câu trả lời từ trang (không đặt nút Copy)");
        return readFromPage(done.answer);
      }
      const text = await copyViaButton(done.copy);
      if (text.trim() === "") throw new RunError("ui_changed", "nội dung câu trả lời (nút Copy " + S.copyButton + " không đưa văn bản)");
      log("lấy câu trả lời qua nút Copy");
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

    /**
     * The settings page's check: what each selector finds on this tab, outlined
     * in red (the Copy button of the last answer in blue).
     */
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
      one("input", function () {
        return queryAll(S.input, "input").slice(0, 1);
      });
      if (S.sendButton) {
        one("sendButton", function () {
          return queryAll(S.sendButton, "sendButton").slice(0, 1);
        });
      }
      one("answer", function () {
        return answers(S).slice(-1);
      });
      if (S.copyButton) {
        one("copyButton", function () {
          const last = answers(S).slice(-1)[0];
          const copy = last ? findCopy(S, last) : null;
          return copy ? [copy] : [];
        }, "#2563eb");
      }
      result.invalid = S.invalid;
      return result;
    }

    function cancel() {
      cancelled = true;
    }

    return { run, probe, cancel, askForWork, typeInto, findInput };
  }

  /**
   * Wires a provider (`{id, name, selectors}`) to its page: answers the worker's cancel and the
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

  const api = { DEFAULT_TIMING, RunError, textOf, create, start };
  root.AiBridgeAdapterCore = api;
  if (typeof module !== "undefined" && module.exports) module.exports = api;
})(typeof globalThis !== "undefined" ? globalThis : self);
