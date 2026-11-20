/**
 * Grok (grok.com) as the shared adapter core drives it. It does not touch
 * Grok's mode or model pickers: the run uses whatever the Creator has chosen
 * there. An answer is complete when its turn shows the response footer
 * actions (selectors.js).
 */
(function (root) {
  const core = root.AiBridgeAdapterCore || (typeof require === "function" ? require("../../adapter/core.js") : null);
  const selectors = (root.AiBridgeSelectors && root.AiBridgeSelectors.grok) || require("./selectors.js");
  const M = core.markers;

  const def = {
    id: "grok",
    name: "Grok",
    selectors: selectors,
    turns: M.turns,
    isDone: M.isDone,
    findCopy: M.findCopy,
    probe: M.probe,
  };

  root.AiBridgeProviderAdapters = root.AiBridgeProviderAdapters || {};
  root.AiBridgeProviderAdapters.grok = def;
  if (typeof module !== "undefined" && module.exports) module.exports = def;
  else core.start(def);
})(typeof globalThis !== "undefined" ? globalThis : self);
