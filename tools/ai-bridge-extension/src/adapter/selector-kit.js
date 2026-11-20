/**
 * Builds the selector set of a provider whose fields are all plain CSS
 * selectors (each with a default the Creator can override on the settings
 * page, copied from DevTools). `resolve` gives each field the Creator's
 * non-blank override or its default, and `invalid` lists the fields whose
 * value is not a usable selector so the adapter can say so.
 */
(function (root) {
  /** `fields` is a list of `{key, label, hint, value}`; `storageKey` is the chrome.storage.sync key of the overrides. */
  function make(storageKey, fields) {
    function defaults() {
      const out = {};
      fields.forEach(function (f) {
        out[f.key] = f.value;
      });
      return out;
    }

    function resolve(overrides, doc) {
      const out = defaults();
      const given = overrides && typeof overrides === "object" ? overrides : {};
      fields.forEach(function (f) {
        const v = given[f.key];
        if (typeof v === "string" && v.trim() !== "") out[f.key] = v.trim();
      });
      out.invalid = fields
        .filter(function (f) {
          return !valid(out[f.key], doc);
        })
        .map(function (f) {
          return f.key;
        });
      out.invalid.forEach(function (key) {
        out[key] = defaults()[key];
      });
      return out;
    }

    function valid(selector, doc) {
      if (selector === "") return true;
      const d = doc || root.document;
      if (!d) return true;
      try {
        d.createDocumentFragment().querySelector(selector);
        return true;
      } catch (err) {
        return false;
      }
    }

    return { STORAGE_KEY: storageKey, FIELDS: fields, defaults: defaults, resolve: resolve };
  }

  const api = { make };
  root.AiBridgeSelectorKit = api;
  if (typeof module !== "undefined" && module.exports) module.exports = api;
})(typeof globalThis !== "undefined" ? globalThis : self);
