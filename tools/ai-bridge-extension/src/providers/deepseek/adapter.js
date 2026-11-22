/**
 * DeepSeek (chat.deepseek.com) as the shared adapter core drives it, from the four
 * selectors in selectors.js plus the Creator's overrides. It does not touch
 * DeepSeek's mode or model pickers: the run uses whatever the Creator has
 * chosen in the page.
 */
(function (root) {
  const core = root.AiBridgeAdapterCore || (typeof require === "function" ? require("../../adapter/core.js") : null);
  const selectors = (root.AiBridgeSelectors && root.AiBridgeSelectors.deepseek) || require("./selectors.js");

  const def = { id: "deepseek", name: "DeepSeek", selectors: selectors };

  root.AiBridgeProviderAdapters = root.AiBridgeProviderAdapters || {};
  root.AiBridgeProviderAdapters.deepseek = def;
  if (typeof module !== "undefined" && module.exports) module.exports = def;
  else core.start(def);
})(typeof globalThis !== "undefined" ? globalThis : self);
