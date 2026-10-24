/**
 * Converts a video length between seconds and the clock form the brief screen
 * shows ("4:30", "1:02:05"). The server stores whole seconds.
 */

/** Seconds as "m:ss" under an hour and "h:mm:ss" from an hour. */
export function formatClock(seconds: number): string {
  const total = Math.max(0, Math.floor(seconds));
  const h = Math.floor(total / 3600);
  const m = Math.floor((total % 3600) / 60);
  const s = total % 60;
  const ss = String(s).padStart(2, "0");
  return h > 0 ? `${h}:${String(m).padStart(2, "0")}:${ss}` : `${m}:${ss}`;
}

/**
 * Reads "m:ss" or "h:mm:ss" back to seconds. A bare number is read as seconds.
 * Returns null for anything else, and for minutes or seconds of 60 or more in
 * a clock form, so a typo is shown as invalid rather than guessed.
 */
export function parseClock(text: string): number | null {
  const value = text.trim();
  if (/^\d+$/.test(value)) return Number(value);
  const parts = value.split(":");
  if (parts.length < 2 || parts.length > 3 || !parts.every((p) => /^\d+$/.test(p))) return null;
  const nums = parts.map(Number);
  const [h, m, s] = parts.length === 3 ? nums : [0, nums[0], nums[1]];
  if (m >= 60 || s >= 60) return null;
  return h * 3600 + m * 60 + s;
}
