/**
 * Gemini (gemini.google.com/app) as the shared adapter core drives it. It does
 * not touch Gemini's mode or model pickers: the run uses whatever the Creator
 * has chosen there. An answer is complete when its turn shows the "complete"
 * response footer and its content is no longer busy (selectors.js).
 */
(function (root) {
  const core = root.AiBridgeAdapterCore || (typeof require === "function" ? require("../../adapter/core.js") : null);
  const selectors = (root.AiBridgeSelectors && root.AiBridgeSelectors.gemini) || require("./selectors.js");
  const M = core.markers;

  const def = {
    id: "gemini",
    name: "Gemini",
    selectors: selectors,
    turns: M.turns,
    isDone: M.isDone,
    findCopy: M.findCopy,
    probe: M.probe,
  };

  root.AiBridgeProviderAdapters = root.AiBridgeProviderAdapters || {};
  root.AiBridgeProviderAdapters.gemini = def;
  if (typeof module !== "undefined" && module.exports) module.exports = def;
  else core.start(def);
})(typeof globalThis !== "undefined" ? globalThis : self);
