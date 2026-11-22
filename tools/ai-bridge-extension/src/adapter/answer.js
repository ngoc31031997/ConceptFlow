/**
 * Pure rules for reading a provider answer: how long it must stay unchanged
 * to count as finished, and how the page's rendered blocks are turned back
 * into text that keeps the code fences (ConceptFlow looks for the ```json block). No DOM access here,
 * so node tests can check every rule.
 */
(function (root) {
  /** The answer must not change for this long, with its Copy button shown, to count as finished. */
  const STABLE_MS = 5 * 1000;
  /** Without a Copy button to wait for, the answer must not change for this long instead. */
  const STABLE_WITHOUT_ACTIONS_MS = 20 * 1000;

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

  const api = { STABLE_MS, STABLE_WITHOUT_ACTIONS_MS, fence, joinBlocks };
  root.AiBridgeAnswer = api;
  if (typeof module !== "undefined" && module.exports) module.exports = api;
})(typeof globalThis !== "undefined" ? globalThis : self);
