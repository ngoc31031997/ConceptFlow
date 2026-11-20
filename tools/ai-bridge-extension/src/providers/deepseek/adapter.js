/**
 * DeepSeek (chat.deepseek.com) as the shared adapter core drives it. Before
 * typing it sets DeepThink and Search as the item asks. An answer is a
 * rendered markdown block that was not on the page before the prompt was
 * sent (DeepThink's reasoning panel is left out); it is complete once it
 * stops changing with the icon row under it shown. Selectors come from
 * selectors.js plus the Creator's overrides.
 */
(function (root) {
  const core = root.AiBridgeAdapterCore || (typeof require === "function" ? require("../../adapter/core.js") : null);
  const A = root.AiBridgeAnswer || (typeof require === "function" ? require("../../adapter/answer.js") : null);
  const selectors = (root.AiBridgeSelectors && root.AiBridgeSelectors.deepseek) || require("./selectors.js");

  /** A mode toggle: by its own selector when set, else the smallest control whose text is one of its labels. */
  function findToggle(S, selector, labels, field, ctx) {
    if (selector) return ctx.queryAll(selector, field)[0] || null;
    const matches = ctx.queryAll(S.clickable, "clickable").filter(function (el) {
      return A.hasLabel(ctx.textOf(el), labels);
    });
    // A label span inside the control can match too: take the real button
    // around it, or else the outermost match, whose class shows the on state.
    const button = matches
      .map(function (el) {
        return el.closest('button, [role="button"]');
      })
      .find(Boolean);
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
  async function setMode(S, selector, labels, name, wanted, ctx) {
    const toggle = await ctx.waitFor(function () {
      return findToggle(S, selector, labels, name, ctx);
    }, ctx.timing.PAGE_WAIT_MS);
    if (!toggle) throw new ctx.RunError("ui_changed", "nút " + name);
    const before = isOn(S, toggle);
    if (before !== wanted) {
      toggle.click();
      await ctx.wait(ctx.timing.TOGGLE_SETTLE_MS);
    }
    ctx.log(name + ":", "class=\"" + (toggle.getAttribute("class") || "") + "\"", "trước", before ? "bật" : "tắt", "→ muốn", wanted ? "bật" : "tắt", "→ giờ", isOn(S, toggle) ? "bật" : "tắt");
  }

  /**
   * The controls matched by the clickable selector, each as the real button
   * around it: DeepSeek's buttons hold decorative children whose class also
   * contains "button" (e.g. "ds-button__background"), which are not controls.
   */
  function clickableControls(S, ctx) {
    return ctx
      .queryAll(S.clickable, "clickable")
      .map(function (el) {
        return el.closest('button, [role="button"]') || el;
      })
      .filter(function (el, i, all) {
        return all.indexOf(el) === i;
      });
  }

  /** The send button: its own selector when set, else the last control in the box around the chat input. */
  function findSendButton(S, input, ctx) {
    if (S.sendButton) return ctx.queryAll(S.sendButton, "sendButton")[0] || null;
    let box = input.parentElement;
    for (let depth = 0; box && depth < 6; depth += 1, box = box.parentElement) {
      const controls = clickableControls(S, ctx).filter(function (el) {
        return box.contains(el) && !el.contains(input) && !A.hasLabel(ctx.textOf(el), S.deepThinkLabels) && !A.hasLabel(ctx.textOf(el), S.searchLabels);
      });
      if (controls.length >= 2) return controls[controls.length - 1];
    }
    return null;
  }

  /**
   * The rendered markdown blocks that are answers, not DeepThink reasoning. A
   * block counts as reasoning only when the thinking-panel ancestor it sits in
   * is local to it: an ancestor that wraps every block (a whole message or the
   * chat list whose class happens to contain "think") does not hide the answer.
   */
  function turns(S, ctx) {
    const all = ctx.queryAll(S.markdown, "markdown");
    return all.filter(function (el) {
      const panel = el.parentElement ? el.parentElement.closest(S.thinking) : null;
      if (!panel) return true;
      return all.every(function (other) {
        return panel.contains(other);
      });
    });
  }

  /** Copy, the first icon button after the answer in its own message (never the chat box's controls). */
  function findCopy(S, answer, ctx) {
    const input = ctx.queryAll(S.input, "input")[0] || null;
    const FOLLOWING = 4; // Node.DOCUMENT_POSITION_FOLLOWING
    let scope = answer.parentElement;
    for (let depth = 0; scope && depth < 6; depth += 1, scope = scope.parentElement) {
      if (input && scope.contains(input)) return null;
      const button = ctx.queryAll(S.iconButton, "iconButton").find(function (el) {
        return scope.contains(el) && !answer.contains(el) && !el.closest(S.thinking) && (answer.compareDocumentPosition(el) & FOLLOWING) !== 0;
      });
      if (button) return button;
    }
    return null;
  }

  /** The answer read from the page, with code blocks fenced again; used when the Copy button gives nothing. */
  function readFromPage(answer, ctx) {
    const blocks = Array.from(answer.children).map(function (child) {
      const pre = child.matches("pre") ? child : child.querySelector("pre");
      if (!pre) return { kind: "text", text: ctx.textOf(child) };
      const label = child.querySelector('[class*="banner"] span, [class*="lang"]');
      return { kind: "code", lang: label ? label.textContent.trim() : "", code: pre.textContent };
    });
    return A.joinBlocks(blocks);
  }

  const def = {
    id: "deepseek",
    name: "DeepSeek",
    selectors: selectors,
    findSendButton: findSendButton,
    turns: turns,
    findCopy: findCopy,
    readFromPage: readFromPage,

    async prepare(S, work, ctx) {
      await setMode(S, S.deepThinkButton, S.deepThinkLabels, "DeepThink", work.modes.deepThink, ctx);
      await setMode(S, S.searchButton, S.searchLabels, "Search", work.modes.search, ctx);
    },

    /** Done once the answer stays the same with the icon row (Copy, Regenerate...) under it shown. */
    isDone(S, answer, progress, ctx) {
      return A.isFinished(progress.text, progress.changedAt, progress.now, findCopy(S, answer, ctx) !== null);
    },

    probe(S, ctx, one, result) {
      one("input", function () {
        return ctx.queryAll(S.input, "input").slice(0, 1);
      });
      one("deepThinkButton", function () {
        const t = findToggle(S, S.deepThinkButton, S.deepThinkLabels, "deepThinkButton", ctx);
        return t ? [t] : [];
      });
      one("searchButton", function () {
        const t = findToggle(S, S.searchButton, S.searchLabels, "searchButton", ctx);
        return t ? [t] : [];
      });
      one("sendButton", function () {
        const input = ctx.queryAll(S.input, "input")[0];
        const b = input ? findSendButton(S, input, ctx) : null;
        return b ? [b] : [];
      });
      one("markdown", function () {
        return turns(S, ctx);
      });
      try {
        const all = ctx.queryAll(S.markdown, "markdown");
        result.markdown.total = all.length;
        result.markdown.thinkingAncestors = all.map(function (el) {
          const panel = el.parentElement ? el.parentElement.closest(S.thinking) : null;
          return panel ? panel.getAttribute("class") || panel.tagName : "";
        });
        const last = turns(S, ctx).slice(-1)[0];
        if (last) {
          const copy = findCopy(S, last, ctx);
          result.iconButton = { count: copy ? 1 : 0, sample: copy ? copy.outerHTML.slice(0, 160) : "" };
          if (copy) {
            const before = copy.style.outline;
            copy.style.outline = "3px solid #2563eb";
            setTimeout(function () {
              copy.style.outline = before;
            }, ctx.timing.HIGHLIGHT_MS);
          }
        }
      } catch (err) {
        result.markdownDetail = { error: String(err && err.message ? err.message : err) };
      }
      [
        ["deepThinkButton", S.deepThinkButton, S.deepThinkLabels],
        ["searchButton", S.searchButton, S.searchLabels],
      ].forEach(function (entry) {
        if (!result[entry[0]] || !result[entry[0]].count) return;
        const el = findToggle(S, entry[1], entry[2], entry[0], ctx);
        result[entry[0]].on = isOn(S, el);
        result[entry[0]].className = el.getAttribute("class") || "";
      });
    },
  };

  root.AiBridgeProviderAdapters = root.AiBridgeProviderAdapters || {};
  root.AiBridgeProviderAdapters.deepseek = def;
  if (typeof module !== "undefined" && module.exports) module.exports = def;
  else core.start(def);
})(typeof globalThis !== "undefined" ? globalThis : self);
