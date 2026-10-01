import type { VideoFormat } from "../types";

/**
 * A format that runs at most this long is for the vertical short; anything
 * longer is for a long video. The server (domain.ShortFormatMaxSeconds) uses
 * the same line when step 2 is confirmed.
 */
export const SHORT_FORMAT_MAX_SECONDS = 90;
/** The built-in format a new short starts with. */
export const SHORT_FORMAT_ID = "vertical_short_60s";
/** The built-in format a long video falls back to when it leaves the short. */
export const DEFAULT_LONG_FORMAT_ID = "case_study_essay_8min";

/** Whether `format` is a format for the vertical short. */
export function isShortFormat(format: VideoFormat): boolean {
  return format.max_seconds > 0 && format.max_seconds <= SHORT_FORMAT_MAX_SECONDS;
}

/** The formats that fit `mode`: short formats for a short, the others for a long video. */
export function formatsFor(mode: string, formats: VideoFormat[]): VideoFormat[] {
  const short = mode === "short";
  return formats.filter((f) => isShortFormat(f) === short);
}
