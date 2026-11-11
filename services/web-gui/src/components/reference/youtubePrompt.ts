/** Where the YouTube link goes in the prompt template. */
export const YOUTUBE_URL_PLACEHOLDER = "{{url}}";

/** The template with every placeholder replaced by the link. */
export function fillYoutubeUrl(template: string, url: string): string {
  return template.split(YOUTUBE_URL_PLACEHOLDER).join(url);
}

/** True when `value` is an http or https address with a host. */
export function isWebUrl(value: string): boolean {
  try {
    const u = new URL(value.trim());
    return (u.protocol === "http:" || u.protocol === "https:") && u.host !== "";
  } catch {
    return false;
  }
}
