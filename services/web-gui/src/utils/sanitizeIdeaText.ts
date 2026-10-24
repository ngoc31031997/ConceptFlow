/**
 * Cleans text typed on the idea step before it reaches the AI prompts. The AI
 * answers in JSON and tends to copy a straight double quote or a backslash from
 * its input straight into a JSON string, which then cannot be read. So a
 * straight double quote becomes a typographic one (“ after a space, bracket or
 * the start, ” anywhere else), a backslash becomes a slash, and control
 * characters other than a new line and a tab are dropped. Everything else,
 * Vietnamese letters and single quotes included, is kept as typed.
 */
export function sanitizeIdeaText(text: string): string {
  let out = "";
  for (const ch of text.replace(/\r\n?/g, "\n")) {
    if (ch === '"') {
      const prev = out.length > 0 ? out[out.length - 1] : "";
      out += prev === "" || /[\s([{“‘—–-]/.test(prev) ? "“" : "”";
    } else if (ch === "\\") {
      out += "/";
    } else if (ch === "\n" || ch === "\t" || ch >= " ") {
      out += ch;
    }
  }
  return out;
}
