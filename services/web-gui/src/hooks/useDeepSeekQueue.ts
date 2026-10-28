import { useCallback, useEffect, useRef, useState } from "react";
import type { ExternalRequest } from "../api/client";
import { newRequestId, post, readExtensionMessage, type AiBridgeModes } from "./useAiBridge";

/** An add the extension has not acknowledged within this time failed to reach it. */
const ACK_TIMEOUT_MS = 5000;

const NO_ACK =
  "Extension không phản hồi. Tải lại trang hoặc kiểm tra extension đã bật trong chrome://extensions.";
const UNKNOWN_ERROR = "Extension báo lỗi nhưng không nói lý do.";

/** How many items of the extension's queue are in each state. */
export interface QueueCounts {
  waiting: number;
  running: number;
  done: number;
  failed: number;
  paused: boolean;
}

/** One request to run on DeepSeek later: what the Creator sees in the popup, and the request itself. */
export interface QueueItem {
  label: string;
  /** Names the project in the popup, since the queue is shared by every project. */
  project: string;
  request: ExternalRequest;
  modes: AiBridgeModes;
}

/** Whether the extension queued the item, or already held that request. */
export type QueueAddOutcome = { ok: true; added: boolean } | { ok: false; message: string };

/** What the screen gets from useDeepSeekQueue. */
export interface DeepSeekQueue {
  /** The installed extension has the queue (its popup lists the items). */
  available: boolean;
  /** Null until the extension has reported its queue. */
  counts: QueueCounts | null;
  /** Adds one item; resolves when the extension answers, never rejects. */
  add: (item: QueueItem) => Promise<QueueAddOutcome>;
}

interface PendingAdd {
  resolve: (outcome: QueueAddOutcome) => void;
  ackTimer: ReturnType<typeof setTimeout>;
}

/**
 * Talks to the queue of the ConceptFlow AI Bridge extension (docs/contracts/ai-bridge-v1.md,
 * "Hàng đợi"): adds items that the extension runs on DeepSeek one after another, even with
 * this tab closed, and follows how many are waiting, running or failed. `onItemDone` is
 * called with the API path of every item that finished, so the screen can load what the
 * run stored. An extension without the queue never reports it, so `available` stays false.
 */
export function useDeepSeekQueue(onItemDone?: (path: string) => void): DeepSeekQueue {
  const [counts, setCounts] = useState<QueueCounts | null>(null);
  const pending = useRef(new Map<string, PendingAdd>());
  const done = useRef(onItemDone);
  done.current = onItemDone;

  useEffect(() => {
    const adds = pending.current;
    function onMessage(event: MessageEvent) {
      const msg = readExtensionMessage(event);
      if (!msg) return;
      if (msg.type === "queue_state") {
        setCounts({
          waiting: msg.waiting ?? 0,
          running: msg.running ?? 0,
          done: msg.done ?? 0,
          failed: msg.failed ?? 0,
          paused: msg.paused === true,
        });
        return;
      }
      if (msg.type === "queue_item_done") {
        if (msg.path) done.current?.(msg.path);
        return;
      }
      const add = msg.request_id ? adds.get(msg.request_id) : undefined;
      if (!add || !msg.request_id) return;
      if (msg.type !== "queue_added" && msg.type !== "error") return;
      clearTimeout(add.ackTimer);
      adds.delete(msg.request_id);
      if (msg.type === "queue_added") add.resolve({ ok: true, added: msg.added !== false });
      else add.resolve({ ok: false, message: msg.message || UNKNOWN_ERROR });
    }
    window.addEventListener("message", onMessage);
    // The extension answers a ping with its queue state when it has a queue.
    post({ type: "ping" });
    return () => {
      window.removeEventListener("message", onMessage);
      adds.forEach((add) => {
        clearTimeout(add.ackTimer);
        add.resolve({ ok: false, message: NO_ACK });
      });
      adds.clear();
    };
  }, []);

  const add = useCallback(
    (item: QueueItem) =>
      new Promise<QueueAddOutcome>((resolve) => {
        const requestId = newRequestId();
        const ackTimer = setTimeout(() => {
          pending.current.delete(requestId);
          resolve({ ok: false, message: NO_ACK });
        }, ACK_TIMEOUT_MS);
        pending.current.set(requestId, { resolve, ackTimer });
        post({ type: "queue_add", request_id: requestId, item });
      }),
    [],
  );

  return { available: counts !== null, counts, add };
}
