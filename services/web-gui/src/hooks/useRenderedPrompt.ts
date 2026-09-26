import { useEffect, useState } from "react";
import { renderPrompt, type PromptRenderInput } from "../api/client";

export interface RenderedPromptState {
  /** The server-rendered text; null until the first render answers. */
  prompt: string | null;
  /** The last render failed (server unreachable, unknown role...). */
  failed: boolean;
  /**
   * `prompt` was rendered for an earlier input and the new one is still in
   * flight. Keep showing it (no flicker while typing) but do not let it be
   * copied: it no longer matches what the Creator has typed.
   */
  stale: boolean;
}

interface Answer {
  key: string | null;
  prompt: string | null;
  failed: boolean;
}

/**
 * The prompt for one library role, rendered by authoring-service (CR-040
 * FR113): the browser assembles no prompt text, so what the Creator copies is
 * what the server would send to the model.
 *
 * Re-renders — debounced, so typing a topic is one request, not one per key —
 * whenever the input changes. A `null` input renders nothing (not ready yet).
 */
export function useRenderedPrompt(input: PromptRenderInput | null, debounceMs = 250): RenderedPromptState {
  const [answer, setAnswer] = useState<Answer>({ key: null, prompt: null, failed: false });
  const key = input ? JSON.stringify(input) : null;

  useEffect(() => {
    if (key === null) {
      setAnswer({ key: null, prompt: null, failed: false });
      return;
    }
    let cancelled = false;
    const timer = window.setTimeout(() => {
      renderPrompt(JSON.parse(key) as PromptRenderInput)
        .then((r) => {
          if (!cancelled) setAnswer({ key, prompt: r.prompt, failed: false });
        })
        .catch(() => {
          if (!cancelled) setAnswer((a) => ({ key, prompt: a.prompt, failed: true }));
        });
    }, debounceMs);
    return () => {
      cancelled = true;
      window.clearTimeout(timer);
    };
  }, [key, debounceMs]);

  return { prompt: answer.prompt, failed: answer.failed, stale: key !== null && answer.key !== key };
}
