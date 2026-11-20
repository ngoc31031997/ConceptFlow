/**
 * The outside AI sites the extension can drive: where a new chat opens and
 * which hosts its adapter runs on. Loaded as a plain script so content
 * scripts, the service worker, the popup, the settings page and node tests
 * read the same table. The default provider (the one an item without its own
 * choice runs on) is kept in chrome.storage.local under DEFAULT_KEY.
 */
(function (root) {
  /** chrome.storage.local key holding the Creator's default provider id. */
  const DEFAULT_KEY = "defaultProvider";

  const LIST = [
    { id: "deepseek", name: "DeepSeek", url: "https://chat.deepseek.com/", host: "chat.deepseek.com", match: "https://chat.deepseek.com/*", hasModes: true },
    { id: "gemini", name: "Gemini", url: "https://gemini.google.com/app", host: "gemini.google.com", match: "https://gemini.google.com/*", hasModes: false },
    { id: "grok", name: "Grok", url: "https://grok.com/", host: "grok.com", match: "https://grok.com/*", hasModes: false },
  ];

  /** Used when nothing valid is stored. */
  const FALLBACK_ID = "deepseek";

  const IDS = LIST.map(function (p) {
    return p.id;
  });

  function isId(id) {
    return typeof id === "string" && IDS.indexOf(id) >= 0;
  }

  /** The provider with this id, or null. */
  function find(id) {
    return (
      LIST.find(function (p) {
        return p.id === id;
      }) || null
    );
  }

  /** The id to use as default: the stored one when it is a known provider, else the fallback. */
  function readDefault(stored) {
    return isId(stored) ? stored : FALLBACK_ID;
  }

  /** Display name of a provider id; an unknown id is returned as is. */
  function nameOf(id) {
    const p = find(id);
    return p ? p.name : String(id);
  }

  const api = { DEFAULT_KEY, LIST, IDS, FALLBACK_ID, isId, find, readDefault, nameOf };
  root.AiBridgeProviders = api;
  if (typeof module !== "undefined" && module.exports) module.exports = api;
})(typeof globalThis !== "undefined" ? globalThis : self);
