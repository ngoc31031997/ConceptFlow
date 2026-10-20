/** One line of a chain: the step's text without its trailing "[F01, F02]" and the facts it cites. */
export function splitChainLine(line: string): { text: string; ids: string[] } {
  const match = /\s*\[([^[\]]*)\]\s*$/.exec(line);
  if (!match) return { text: line, ids: [] };
  const ids = match[1].split(",").map((s) => s.trim());
  if (!ids.every((id) => /^F\d+$/.test(id))) return { text: line, ids: [] };
  return { text: line.slice(0, match.index), ids };
}
