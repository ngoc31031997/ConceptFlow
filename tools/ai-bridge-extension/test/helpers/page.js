/**
 * Builds fake provider pages for the adapter tests with jsdom, which the
 * extension itself does not use: it is taken from services/web-gui's
 * dependencies, so run `npm ci` there once before the extension tests.
 */
const path = require("node:path");

function loadJsdom() {
  const guiDir = path.resolve(__dirname, "../../../../services/web-gui");
  try {
    return require(require.resolve("jsdom", { paths: [guiDir] }));
  } catch (err) {
    throw new Error("jsdom not found: run `npm ci` in services/web-gui first (" + err.message + ")");
  }
}

/** Timings short enough for a test; the waits keep their order. */
const FAST = {
  INPUT_WAIT_MS: 400,
  SEND_ENABLE_MS: 300,
  SEND_CHECK_MS: 300,
  TYPE_SETTLE_MS: 5,
  TOGGLE_SETTLE_MS: 5,
  PAGE_WAIT_MS: 400,
  TICK_MS: 40,
  MIN_CHECK_GAP_MS: 5,
  COPY_WAIT_MS: 400,
  DIAGNOSE_EVERY_MS: 60 * 60 * 1000,
};

/**
 * A page for `html` at `url`. `execCommand` is not in jsdom, so a contenteditable box is filled by
 * the stand-in installed here: `insertText` puts the text in the box that was focused last through `typeInto`.
 */
function makePage(html, url) {
  const { JSDOM } = loadJsdom();
  const dom = new JSDOM(html, { url: url, pretendToBeVisual: true });
  const win = dom.window;
  const doc = win.document;
  let target = null;
  const focus = win.HTMLElement.prototype.focus;
  win.HTMLElement.prototype.focus = function () {
    target = this;
    return focus.call(this);
  };
  doc.execCommand = function (command, _ui, text) {
    if (command === "insertText" && target) target.textContent = text;
    return true;
  };
  return { win, doc, close: () => win.close() };
}

/**
 * Plays the clipboard hook: once the adapter arms it, the next click on `button` hands `text` to the
 * adapter instead of writing the clipboard. Returns how many times the button was pressed.
 */
function copyOnClick(page, button, text, source) {
  let armed = false;
  const presses = { count: 0 };
  page.win.addEventListener("message", (event) => {
    if (event.data && event.data.kind === "arm-copy") armed = true;
  });
  button.addEventListener("click", () => {
    presses.count += 1;
    if (!armed) return;
    armed = false;
    page.win.dispatchEvent(
      new page.win.MessageEvent("message", {
        data: { source: source, kind: "copied", text: text },
        origin: page.win.location.origin,
        source: page.win,
      }),
    );
  });
  return presses;
}

module.exports = { makePage, copyOnClick, FAST };
