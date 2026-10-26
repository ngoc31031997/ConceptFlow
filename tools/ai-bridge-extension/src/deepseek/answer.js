/**
 * Pure rules for reading a DeepSeek answer: when it counts as finished, and
 * how the page's rendered blocks are turned back into text that keeps the
 * code fences (ConceptFlow looks for the ```json block). No DOM access here,
 * so node tests can check every rule.
 */
(function (root) {
  /** The answer must not change for this long, with its icon row shown, to count as finished. */
  const STABLE_MS = 5 * 1000;
  /** Without an icon row (DeepSeek changed it) the answer must not change for this long instead. */
  const STABLE_WITHOUT_ACTIONS_MS = 60 * 1000;

  /**
   * Whether the answer is finished. `text` is the answer read so far,
   * `changedAt` when it last changed, `hasActions` whether the icon row under
   * it (Copy, Regenerate…) is there, which DeepSeek only shows once done.
   */
  function isFinished(text, changedAt, now, hasActions) {
    if (!text || text.trim() === "") return false;
    const quiet = now - changedAt;
    return hasActions ? quiet >= STABLE_MS : quiet >= STABLE_WITHOUT_ACTIONS_MS;
  }

  /** True when a control's visible text is one of the labels, ignoring case and spacing. */
  function hasLabel(text, labels) {
    const t = String(text || "").replace(/\s+/g, " ").trim().toLowerCase();
    return labels.some(function (label) {
      return t === label.toLowerCase() || t.startsWith(label.toLowerCase() + " ") || t.startsWith(label.toLowerCase() + "(");
    });
  }

  /** A code block written back as a markdown fence. */
  function fence(lang, code) {
    return "```" + (lang || "") + "\n" + String(code).replace(/\n+$/, "") + "\n```";
  }

  /**
   * Joins the answer's blocks, in page order, into one text: `{kind:"text", text}`
   * as it reads, `{kind:"code", lang, code}` as a fence. Empty blocks are dropped.
   */
  function joinBlocks(blocks) {
    return blocks
      .map(function (b) {
        if (b.kind === "code") return b.code.trim() === "" ? "" : fence(b.lang, b.code);
        return String(b.text || "").trim();
      })
      .filter(function (s) {
        return s !== "";
      })
      .join("\n\n");
  }

  const api = { STABLE_MS, STABLE_WITHOUT_ACTIONS_MS, isFinished, hasLabel, fence, joinBlocks };
  root.DeepSeekAnswer = api;
  if (typeof module !== "undefined" && module.exports) module.exports = api;
})(typeof globalThis !== "undefined" ? globalThis : self);
