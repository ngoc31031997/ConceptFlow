/**
 * Everything the adapter knows about DeepSeek's page. When DeepSeek changes
 * its interface, this is the one file to update (README, "Khi DeepSeek đổi
 * giao diện"). Hashed class names change on every DeepSeek release, so the
 * entries lean on ids, roles, the `ds-` design-system classes and visible
 * labels instead.
 */
(function (root) {
  const DEEPSEEK_SELECTORS = {
    /** The chat box; its presence is also how the adapter knows the Creator is signed in. */
    input: "textarea#chat-input, textarea",
    /** Rendered markdown blocks; the answer is the last one outside the thinking panel. */
    markdown: ".ds-markdown",
    /** Ancestor of the DeepThink reasoning text, which is left out of the answer. */
    thinking: '[class*="think"]',
    /** Clickable controls: the mode toggles under the chat box and the icon row under an answer. */
    clickable: 'button, [role="button"]',
    /** Icon-only buttons under a finished answer, Copy first. */
    iconButton: '.ds-icon-button, [class*="icon-button"]',
    /** A toggle counts as on when it carries one of these, or aria-pressed="true". */
    selectedClass: /selected|active|checked/i,
    /** Visible labels of the two mode toggles, in DeepSeek's English and Chinese interface. */
    deepThinkLabels: ["DeepThink", "深度思考"],
    searchLabels: ["Search", "联网搜索"],
    /** URL path of the sign-in page DeepSeek redirects to. */
    signInPath: /sign_in|login/i,
  };

  root.DEEPSEEK_SELECTORS = DEEPSEEK_SELECTORS;
  if (typeof module !== "undefined" && module.exports) module.exports = DEEPSEEK_SELECTORS;
})(typeof globalThis !== "undefined" ? globalThis : self);
