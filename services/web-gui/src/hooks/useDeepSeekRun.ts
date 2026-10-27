import { useCallback, useRef, useState } from "react";
import {
  ApiError,
  ExternalPromptReady,
  GENERIC_CONNECTION_ERROR,
  type ExternalAi,
} from "../api/client";
import { useAiBridge, type AiBridgeModes, type AiBridgeStage } from "./useAiBridge";

/** Most steps reason over what the project already holds, so DeepSeek does not search the web. */
export const DEFAULT_DEEPSEEK_MODES: AiBridgeModes = { deepThink: true, search: false };

const NO_PROMPT = "Mục này không trả prompt để chạy trên DeepSeek.";

/** What the Creator sees for a failed request: the server's own message, else the generic one. */
export function messageOf(err: unknown): string {
  return err instanceof ApiError && err.message ? err.message : GENERIC_CONNECTION_ERROR;
}

interface UseDeepSeekRunOptions<T> {
  /**
   * Makes the step's own AI request in the given external mode. In "prompt" mode it rejects with
   * `ExternalPromptReady`; in "paste" mode it resolves once the step has stored the result.
   */
  run: (external: ExternalAi) => Promise<T>;
  /** DeepSeek's answer was read and the step ran on it; the screen loads what changed. */
  onDone: (result: T) => void | Promise<void>;
  /** DeepSeek answered but the step cannot use the answer: the host shows it for the Creator to fix. */
  onUnreadable: (reply: string, message: string) => void;
  modes?: AiBridgeModes;
}

/** What a screen gets from useDeepSeekRun. */
export interface DeepSeekRun {
  /** The AI Bridge extension is installed and drives DeepSeek. */
  available: boolean;
  /** The stage of the run in progress, or null when nothing runs. */
  stage: AiBridgeStage | null;
  /** True from the click until the step has read the answer, including the prompt and read requests. */
  busy: boolean;
  modes: AiBridgeModes;
  /** Why the last run gave nothing, or null. */
  error: string | null;
  /** Builds the step's prompt, runs it on DeepSeek and has the step read the answer. Never rejects. */
  start: () => Promise<void>;
  /** Stops waiting for the run in progress. */
  cancel: () => void;
}

/**
 * Runs one AI step of the project on DeepSeek through the AI Bridge extension,
 * with the same two requests as a copy-and-paste by hand: the step's prompt,
 * then the step on DeepSeek's answer.
 */
export function useDeepSeekRun<T>({
  run,
  onDone,
  onUnreadable,
  modes = DEFAULT_DEEPSEEK_MODES,
}: UseDeepSeekRunOptions<T>): DeepSeekRun {
  const bridge = useAiBridge();
  const runOnBridge = bridge.run;
  const [preparing, setPreparing] = useState(false);
  const [reading, setReading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  // `start` can be awaited by a caller after other work, so it reads the latest options, not the ones it was built with.
  const latest = useRef({ run, onDone, onUnreadable });
  latest.current = { run, onDone, onUnreadable };

  const start = useCallback(async () => {
    const { run: runStep, onDone: finish, onUnreadable: unreadable } = latest.current;
    setError(null);
    setPreparing(true);
    let prompt: string;
    try {
      await runStep({ mode: "prompt" });
      setError(NO_PROMPT);
      return;
    } catch (err) {
      if (!(err instanceof ExternalPromptReady)) {
        setError(messageOf(err));
        return;
      }
      prompt = err.prompt;
    } finally {
      setPreparing(false);
    }
    const outcome = await runOnBridge("deepseek", prompt, modes);
    if (!outcome.ok) {
      if (!outcome.cancelled) setError(outcome.message);
      return;
    }
    setReading(true);
    try {
      let result: T;
      try {
        result = await runStep({ mode: "paste", reply: outcome.text });
      } catch (err) {
        unreadable(outcome.text, messageOf(err));
        return;
      }
      await finish(result);
    } finally {
      setReading(false);
    }
  }, [runOnBridge, modes]);

  return {
    available: bridge.providers.includes("deepseek"),
    stage: bridge.stage,
    busy: preparing || reading || bridge.stage !== null,
    modes,
    error,
    start,
    cancel: bridge.cancel,
  };
}
