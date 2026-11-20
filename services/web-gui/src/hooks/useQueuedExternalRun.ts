import { useCallback, useEffect, useRef, useState } from "react";
import {
  ApiError,
  ExternalRequestDescribed,
  GENERIC_CONNECTION_ERROR,
  type ExternalAi,
  type ExternalRequest,
} from "../api/client";
import type { AiBridgeModes } from "./useAiBridge";
import { useAiQueue, type QueueItemStatus } from "./useAiQueue";

/** Most steps reason over what the project already holds, so the outside AI does not search the web. */
export const DEFAULT_AI_MODES: AiBridgeModes = { deepThink: true, search: false };

const NO_REQUEST = "Mục này không mô tả được yêu cầu để thêm vào hàng đợi.";

/** What the Creator sees for a failed request: the server's own message, else the generic one. */
export function messageOf(err: unknown): string {
  return err instanceof ApiError && err.message ? err.message : GENERIC_CONNECTION_ERROR;
}

/** The project an API path belongs to (`/v1/projects/{id}/…`), or empty; the popup names it beside the item. */
function projectOf(request: ExternalRequest): string {
  return /^\/v1\/projects\/([^/]+)/.exec(request.path)?.[1] ?? "";
}

interface UseQueuedExternalRunOptions<T> {
  /**
   * Makes the step's own AI request in the given external mode. Here it is called in "describe" mode, in which it
   * rejects with `ExternalRequestDescribed` carrying the request, and the extension makes that request itself.
   * Absent for a run that always brings its own request and prompt to `start`.
   */
  run?: (external: ExternalAi) => Promise<T>;
  /** Names the item in the extension's popup. */
  label: string;
  /** The extension ran the item and the step stored the answer; the screen loads what changed. */
  onDone: () => void | Promise<void>;
  /** The provider answered but the step cannot use the answer: the host shows it for the Creator to fix. */
  onUnreadable: (reply: string, message: string) => void;
  modes?: AiBridgeModes;
}

/** What a screen gets from useQueuedExternalRun. */
export interface QueuedExternalRun {
  /** The AI Bridge extension with the queue is installed. */
  available: boolean;
  /** An extension of the old protocol is installed and has to be updated. */
  outdated: boolean;
  /** True from the click until the item is done, failed or cancelled, including building the request. */
  busy: boolean;
  /** Where the item stands, or null when none is followed. */
  item: QueueItemStatus | null;
  /** How many items run or wait before the followed one, when it waits and the extension has reported its queue. */
  ahead: number | null;
  /** The provider an item without its own choice runs on, once the extension has said. */
  defaultProvider: string | null;
  /** Why the last run gave nothing, or null. */
  error: string | null;
  /**
   * Describes the step's request, queues it and follows the item. With `ready`, the page already holds the prompt and
   * the request that takes the answer as `{raw}`, so nothing is described. Never rejects.
   */
  start: (ready?: { prompt: string; request: ExternalRequest }) => Promise<void>;
  /** Takes the followed item out of the queue; a running one is stopped. */
  cancel: () => void;
}

/**
 * Runs one AI step of the project through the AI Bridge queue: the step describes its request, the extension
 * queues it and runs it on an outside AI (DeepSeek, Gemini or Grok, as the Creator chose), and the screen
 * follows the item. When it is done the step already holds the answer and `onDone` reloads; when it fails with
 * an answer the step could not read, `onUnreadable` hands that answer over.
 */
export function useQueuedExternalRun<T>({
  run,
  label,
  onDone,
  onUnreadable,
  modes = DEFAULT_AI_MODES,
}: UseQueuedExternalRunOptions<T>): QueuedExternalRun {
  const queue = useAiQueue();
  const { add, watch, cancel: cancelItem } = queue;
  const [preparing, setPreparing] = useState(false);
  const [itemId, setItemId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  // `start` can be awaited by a caller after other work, so it reads the latest options, not the ones it was built with.
  const latest = useRef({ run, label, onDone, onUnreadable });
  latest.current = { run, label, onDone, onUnreadable };

  const start = useCallback(async (ready?: { prompt: string; request: ExternalRequest }) => {
    const { run: runStep, label: itemLabel } = latest.current;
    setError(null);
    let request: ExternalRequest;
    if (ready) {
      request = ready.request;
    } else {
      if (!runStep) {
        setError(NO_REQUEST);
        return;
      }
      setPreparing(true);
      try {
        await runStep({ mode: "describe" });
        setError(NO_REQUEST);
        return;
      } catch (err) {
        if (!(err instanceof ExternalRequestDescribed)) {
          setError(messageOf(err));
          return;
        }
        request = err.request;
      } finally {
        setPreparing(false);
      }
    }
    const item = { label: itemLabel, project: projectOf(request), request, modes };
    const outcome = await add(ready ? { ...item, kind: "prompt", prompt: ready.prompt } : item, { watch: true });
    if (!outcome.ok) {
      setError(outcome.message);
      return;
    }
    watch(outcome.itemId);
    setItemId(outcome.itemId);
  }, [add, watch, modes]);

  const item = itemId ? (queue.items[itemId] ?? null) : null;
  const state = item?.state;
  const reply = item?.reply ?? "";
  const failure = item?.error ?? "";

  useEffect(() => {
    if (!itemId || !state) return;
    if (state === "done") {
      setItemId(null);
      void latest.current.onDone();
    } else if (state === "failed") {
      setItemId(null);
      if (reply.trim() !== "") latest.current.onUnreadable(reply, failure);
      else setError(failure || "Mục chạy không thành công.");
    } else if (state === "cancelled") {
      setItemId(null);
    }
  }, [itemId, state, reply, failure]);

  const cancel = useCallback(() => {
    if (itemId) cancelItem(itemId);
  }, [itemId, cancelItem]);

  return {
    available: queue.available,
    outdated: queue.outdated,
    busy: preparing || itemId !== null,
    item,
    ahead: state === "waiting" && queue.counts ? queue.counts.running + Math.max(queue.counts.waiting - 1, 0) : null,
    defaultProvider: queue.defaultProvider,
    error,
    start,
    cancel,
  };
}
