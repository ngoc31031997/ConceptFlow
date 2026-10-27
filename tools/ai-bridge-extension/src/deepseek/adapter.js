/**
 * Content script on chat.deepseek.com. In a tab the service worker opened for
 * a run it sets DeepThink and Search as the run asks, sends the prompt, waits
 * for the answer to finish and reports it back. In any other DeepSeek tab (the
 * Creator's own) the worker has no work for it and it only answers the
 * settings page's selector check.
 *
 * Every element it cannot find ends the run with `ui_changed` naming that
 * element, so a DeepSeek redesign shows up as a clear error, never as a wrong
 * answer. Each step is logged to the tab's console with the prefix
 * "ai-bridge:". Selectors come from selectors.js plus the Creator's overrides.
 */
(function () {
  const P = globalThis.AiBridgeProtocol;
  const DS = globalThis.DeepSeekSelectors;
  const A = globalThis.DeepSeekAnswer;

  const READY_TRIES = 5;
  const READY_RETRY_MS = 1000;
  const PAGE_WAIT_MS = 20 * 1000;
  const SEND_CHECK_MS = 5 * 1000;
  const TOGGLE_SETTLE_MS = 400;
  const COPY_WAIT_MS = 3000;
  const TICK_MS = 1000;
  /** The wait for the answer looks again on a page change, but not more often than this. */
  const MIN_CHECK_GAP_MS = 300;
  const HIGHLIGHT_MS = 4000;
  /** How often the wait for the answer logs what it sees. */
  const DIAGNOSE_EVERY_MS = 15 * 1000;

  /** Thrown inside the run to end it with an error code. */
  class RunError extends Error {
    constructor(code, detail) {
      super(detail || code);
      this.code = code;
      this.detail = detail || "";
    }
  }

  let cancelled = false;

  function log() {
    console.info.apply(console, ["ai-bridge:"].concat(Array.from(arguments)));
  }

  function wait(ms) {
    return new Promise(function (resolve) {
      setTimeout(resolve, ms);
    });
  }

  function report(message) {
    return chrome.runtime.sendMessage(message);
  }

  async function loadSelectors() {
    const stored = await chrome.storage.sync.get(DS.STORAGE_KEY);
    const S = DS.resolve(stored[DS.STORAGE_KEY]);
    if (S.invalid.length) log("override không dùng được, dùng mặc định:", S.invalid.join(", "));
    return S;
  }

  /** querySelectorAll that reports a broken selector as the field it came from. */
  function queryAll(selector, field) {
    try {
      return Array.from(document.querySelectorAll(selector));
    } catch (err) {
      throw new RunError("ui_changed", "selector \"" + field + "\" sai cú pháp: " + selector);
    }
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

  function findInput(S) {
    return queryAll(S.input, "input")[0] || null;
  }

  /** A mode toggle: by its own selector when set, else the smallest control whose text is one of its labels. */
  function findToggle(S, selector, labels, field) {
    if (selector) return queryAll(selector, field)[0] || null;
    const matches = queryAll(S.clickable, "clickable").filter(function (el) {
      return A.hasLabel(el.innerText, labels);
    });
    // A label span inside the control can match too: take the real button
    // around it, or else the outermost match, whose class shows the on state.
    const button = matches.map(function (el) {
      return el.closest('button, [role="button"]');
    }).find(Boolean);
    if (button) return button;
    return (
      matches.find(function (el) {
        return !matches.some(function (other) {
          return other !== el && other.contains(el);
        });
      }) || null
    );
  }

  function isOn(S, el) {
    if (el.getAttribute("aria-pressed") === "true" || el.getAttribute("aria-checked") === "true") return true;
    if (el.getAttribute("data-state") === "on" || el.getAttribute("data-state") === "checked") return true;
    return S.selectedClass.test(el.getAttribute("class") || "");
  }

  /** Puts a mode toggle in the wanted state, clicking it only when it is not there yet. */
  async function setMode(S, selector, labels, name, wanted) {
    const toggle = await waitFor(function () {
      return findToggle(S, selector, labels, name);
    }, PAGE_WAIT_MS);
    if (!toggle) throw new RunError("ui_changed", "nút " + name);
    const before = isOn(S, toggle);
    if (before !== wanted) {
      toggle.click();
      await wait(TOGGLE_SETTLE_MS);
    }
    log(name + ":", "class=\"" + (toggle.getAttribute("class") || "") + "\"", "trước", before ? "bật" : "tắt", "→ muốn", wanted ? "bật" : "tắt", "→ giờ", isOn(S, toggle) ? "bật" : "tắt");
  }

  /** Puts the prompt in the chat box the way typing would, so DeepSeek's React state sees it. */
  function typeInto(input, text) {
    input.focus();
    if (input.isContentEditable) {
      document.execCommand("selectAll", false);
      document.execCommand("insertText", false, text);
      return;
    }
    const setter = Object.getOwnPropertyDescriptor(HTMLTextAreaElement.prototype, "value").set;
    setter.call(input, text);
    input.dispatchEvent(new Event("input", { bubbles: true }));
  }

  function inputText(input) {
    return input.isContentEditable ? input.innerText : input.value;
  }

  function pressEnter(input) {
    const init = { key: "Enter", code: "Enter", keyCode: 13, which: 13, bubbles: true, cancelable: true };
    input.dispatchEvent(new KeyboardEvent("keydown", init));
    input.dispatchEvent(new KeyboardEvent("keypress", init));
    input.dispatchEvent(new KeyboardEvent("keyup", init));
  }

  /**
   * The controls matched by the clickable selector, each as the real button
   * around it: DeepSeek's buttons hold decorative children whose class also
   * contains "button" (e.g. "ds-button__background"), which are not controls.
   */
  function clickableControls(S) {
    return queryAll(S.clickable, "clickable")
      .map(function (el) {
        return el.closest('button, [role="button"]') || el;
      })
      .filter(function (el, i, all) {
        return all.indexOf(el) === i;
      });
  }

  /** The send button: its own selector when set, else the last control in the box around the chat input. */
  function findSendButton(S, input) {
    if (S.sendButton) return queryAll(S.sendButton, "sendButton")[0] || null;
    let box = input.parentElement;
    for (let depth = 0; box && depth < 6; depth += 1, box = box.parentElement) {
      const controls = clickableControls(S).filter(function (el) {
        return box.contains(el) && !el.contains(input) && !A.hasLabel(el.innerText, S.deepThinkLabels) && !A.hasLabel(el.innerText, S.searchLabels);
      });
      if (controls.length >= 2) return controls[controls.length - 1];
    }
    return null;
  }

  async function send(S, input) {
    const sent = function () {
      const current = findInput(S);
      return !current || inputText(current).trim() === "";
    };
    const button = findSendButton(S, input);
    if (button) {
      log("bấm nút gửi:", button.outerHTML.slice(0, 200));
      button.click();
      if (await waitFor(sent, SEND_CHECK_MS)) return;
    } else {
      log("không thấy nút gửi, thử Enter");
    }
    pressEnter(input);
    if (await waitFor(sent, SEND_CHECK_MS)) return;
    throw new RunError("ui_changed", button ? "gửi prompt (đã bấm nút gửi và Enter)" : "nút gửi");
  }

  /**
   * The rendered markdown blocks that are answers, not DeepThink reasoning. A
   * block counts as reasoning only when the thinking-panel ancestor it sits in
   * is local to it: an ancestor that wraps every block (a whole message or the
   * chat list whose class happens to contain "think") does not hide the answer.
   */
  function answerBlocks(S) {
    const all = queryAll(S.markdown, "markdown");
    return all.filter(function (el) {
      const panel = el.parentElement ? el.parentElement.closest(S.thinking) : null;
      if (!panel) return true;
      return all.every(function (other) {
        return panel.contains(other);
      });
    });
  }

  /** The answer to this run: the last markdown block that was not on the page before the prompt was sent. */
  function findAnswer(S, countBefore) {
    const blocks = answerBlocks(S);
    return blocks.length > countBefore ? blocks[blocks.length - 1] : null;
  }

  /** Copy, the first icon button after the answer in its own message (never the chat box's controls). */
  function findCopyButton(S, answer) {
    const input = findInput(S);
    let scope = answer.parentElement;
    for (let depth = 0; scope && depth < 6; depth += 1, scope = scope.parentElement) {
      if (input && scope.contains(input)) return null;
      const button = queryAll(S.iconButton, "iconButton").find(function (el) {
        return scope.contains(el) && !answer.contains(el) && !el.closest(S.thinking) && (answer.compareDocumentPosition(el) & Node.DOCUMENT_POSITION_FOLLOWING) !== 0;
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

  async function waitForAnswer(S, countBefore, deadline) {
    let lastText = "";
    let changedAt = Date.now();
    let loggedAt = 0;
    for (;;) {
      checkStillRunning(deadline);
      const answer = findAnswer(S, countBefore);
      const text = answer ? answer.innerText : "";
      if (text !== lastText) {
        lastText = text;
        changedAt = Date.now();
      }
      const hasActions = answer ? findCopyButton(S, answer) !== null : false;
      if (Date.now() - loggedAt >= DIAGNOSE_EVERY_MS) {
        loggedAt = Date.now();
        log(
          "đang chờ:",
          "markdown trên trang=" + queryAll(S.markdown, "markdown").length,
          "khối trả lời=" + answerBlocks(S).length + " (trước khi gửi " + countBefore + ")",
          "độ dài câu trả lời=" + text.length,
          "đứng yên " + Math.round((Date.now() - changedAt) / 1000) + "s",
          "nút Copy=" + (hasActions ? "có" : "chưa"),
          "tab " + (document.hidden ? "nền" : "đang xem"),
        );
      }
      if (answer && A.isFinished(text, changedAt, Date.now(), hasActions)) {
        log("câu trả lời xong:", text.length, "ký tự, nút Copy", hasActions ? "có" : "không");
        return answer;
      }
      await waitForPageChange(TICK_MS);
    }
  }

  /**
   * Waits until the page changes (DeepSeek typing, its icon row appearing) or
   * `maxMs` passes, so a finished answer is noticed at once instead of at the
   * next tick. Changes closer together than MIN_CHECK_GAP_MS count once.
   */
  function waitForPageChange(maxMs) {
    return new Promise(function (resolve) {
      const startedAt = Date.now();
      const observer = new MutationObserver(function () {
        if (Date.now() - startedAt >= MIN_CHECK_GAP_MS) done();
      });
      const timer = setTimeout(done, maxMs);
      function done() {
        observer.disconnect();
        clearTimeout(timer);
        resolve();
      }
      observer.observe(document.body, { childList: true, subtree: true, characterData: true });
    });
  }

  async function run(work) {
    const S = await loadSelectors();
    log("bắt đầu lượt", work.requestId, "chế độ", JSON.stringify(work.modes));
    const input = await waitFor(function () {
      return findInput(S);
    }, PAGE_WAIT_MS);
    if (!input) {
      if (S.signInPath.test(window.location.pathname)) throw new RunError("not_logged_in");
      throw new RunError("ui_changed", "ô chat");
    }
    log("ô chat:", input.outerHTML.slice(0, 200));
    report({ kind: "progress", requestId: work.requestId, stage: "sending" });
    await setMode(S, S.deepThinkButton, S.deepThinkLabels, "DeepThink", work.modes.deepThink);
    await setMode(S, S.searchButton, S.searchLabels, "Search", work.modes.search);
    const countBefore = answerBlocks(S).length;
    typeInto(input, work.prompt);
    await wait(300);
    if (inputText(input).trim() === "") throw new RunError("ui_changed", "ghi prompt vào ô chat");
    log("đã ghi prompt,", work.prompt.length, "ký tự");
    await send(S, input);
    log("đã gửi");

    report({ kind: "progress", requestId: work.requestId, stage: "answering" });
    const answer = await waitForAnswer(S, countBefore, work.deadline);

    report({ kind: "progress", requestId: work.requestId, stage: "reading" });
    const button = findCopyButton(S, answer);
    const copied = button ? await copyViaButton(button) : "";
    log(copied ? "lấy câu trả lời qua nút Copy" : "lấy câu trả lời từ trang");
    const text = copied.trim() !== "" ? copied : readFromPage(answer);
    if (text.trim() === "") throw new RunError("ui_changed", "nội dung câu trả lời");
    return text;
  }

  /** Draws an outline on matched elements for a few seconds, so the Creator sees what a selector found. */
  function highlight(elements, color) {
    elements.forEach(function (el) {
      const before = el.style.outline;
      el.style.outline = "3px solid " + color;
      setTimeout(function () {
        el.style.outline = before;
      }, HIGHLIGHT_MS);
    });
  }

  /** The settings page's check: what each selector finds on this tab, outlined in red. */
  function probe(overrides) {
    const S = DS.resolve(overrides);
    const result = {};
    function one(key, find) {
      try {
        const found = find();
        result[key] = { count: found.length, sample: found[0] ? found[0].outerHTML.slice(0, 160) : "" };
        highlight(found, "#e11d48");
      } catch (err) {
        result[key] = { error: String(err && err.message ? err.message : err) };
      }
    }
    one("input", function () {
      return queryAll(S.input, "input").slice(0, 1);
    });
    one("deepThinkButton", function () {
      const t = findToggle(S, S.deepThinkButton, S.deepThinkLabels, "deepThinkButton");
      return t ? [t] : [];
    });
    one("searchButton", function () {
      const t = findToggle(S, S.searchButton, S.searchLabels, "searchButton");
      return t ? [t] : [];
    });
    one("sendButton", function () {
      const input = findInput(S);
      const b = input ? findSendButton(S, input) : null;
      return b ? [b] : [];
    });
    one("markdown", function () {
      return answerBlocks(S);
    });
    try {
      const all = queryAll(S.markdown, "markdown");
      result.markdown.total = all.length;
      result.markdown.thinkingAncestors = all.map(function (el) {
        const panel = el.parentElement ? el.parentElement.closest(S.thinking) : null;
        return panel ? panel.getAttribute("class") || panel.tagName : "";
      });
      const last = answerBlocks(S).slice(-1)[0];
      if (last) {
        const copy = findCopyButton(S, last);
        result.iconButton = { count: copy ? 1 : 0, sample: copy ? copy.outerHTML.slice(0, 160) : "" };
        if (copy) highlight([copy], "#2563eb");
      }
    } catch (err) {
      result.markdownDetail = { error: String(err && err.message ? err.message : err) };
    }
    ["deepThinkButton", "searchButton"].forEach(function (key) {
      if (result[key] && result[key].count) {
        const el = key === "deepThinkButton" ? findToggle(S, S.deepThinkButton, S.deepThinkLabels, key) : findToggle(S, S.searchButton, S.searchLabels, key);
        result[key].on = isOn(S, el);
        result[key].className = el.getAttribute("class") || "";
      }
    });
    result.invalid = S.invalid;
    return result;
  }

  chrome.runtime.onMessage.addListener(function (message, sender, sendResponse) {
    if (!message) return false;
    if (message.kind === "cancel") cancelled = true;
    if (message.kind === "probe") {
      sendResponse(probe(message.selectors));
      return false;
    }
    return false;
  });

  askForWork().then(
    function (work) {
      if (!work) return;
      run(work).then(
        function (text) {
          log("xong,", text.length, "ký tự");
          report({ kind: "result", requestId: work.requestId, text: text });
        },
        function (err) {
          if (err instanceof RunError && err.code === "cancelled") return;
          const code = err instanceof RunError ? err.code : "ui_changed";
          const detail = err instanceof RunError ? err.detail : String(err && err.message ? err.message : err);
          console.error("ai-bridge: lỗi", code, detail);
          report({ kind: "error", requestId: work.requestId, code: code, detail: detail });
        },
      );
    },
    function (err) {
      console.error("ai-bridge: could not reach the extension worker:", err);
    },
  );
})();
