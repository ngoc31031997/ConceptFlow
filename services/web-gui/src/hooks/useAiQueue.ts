import { useCallback, useEffect, useRef, useState } from "react";
import type { ExternalRequest } from "../api/client";
import {
  isLegacyPong,
  newRequestId,
  pingExtension,
  post,
  readExtensionMessage,
  type AiBridgeModes,
  type AiBridgeProvider,
} from "./useAiBridge";

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

/**
 * One request to run on an outside AI later: what the Creator sees in the popup, and the request itself.
 * `kind` "prompt" is an item whose prompt the page already holds (`prompt`): `request` then takes the
 * answer as `{raw}` instead of being made in the external-AI modes.
 */
export interface QueueItem {
  label: string;
  /** Names the project in the popup, since the queue is shared by every project. */
  project: string;
  request: ExternalRequest;
  modes: AiBridgeModes;
  /** The provider to run on; absent or null runs on the extension's default when the item's turn comes. */
  provider?: AiBridgeProvider | null;
  kind?: "request" | "prompt";
  prompt?: string;
}

/** Where a queued item stands, as the extension reports it. */
export type QueueItemState = "waiting" | "running" | "done" | "failed" | "cancelled";

/** What the extension last said about one item. */
export interface QueueItemStatus {
  itemId: string;
  /** API path of the item's request, to tell which data a finished run changed. */
  path: string;
  state: QueueItemState;
  /** The step of a running item ("opening", "answering"…); empty otherwise. */
  stage: string;
  /** The provider the item runs or ran on, or would run on while it waits. */
  provider: string;
  /** Why the item failed, else empty. */
  error: string;
  /** The provider's answer when the server could not use it, else empty. */
  reply: string;
}

/** Whether the extension queued the item, or already held that request. */
export type QueueAddOutcome =
  { ok: true; added: boolean; itemId: string } | { ok: false; message: string };

/** What the screen gets from useAiQueue. */
export interface AiQueue {
  /** The installed extension has the queue (its popup lists the items). */
  available: boolean;
  /** True when an extension answered with the old protocol and has to be updated. */
  outdated: boolean;
  /** Null until the extension has reported its queue. */
  counts: QueueCounts | null;
  /** The provider an item without its own choice runs on, once the extension has said. */
  defaultProvider: AiBridgeProvider | null;
  /**
   * Adds one item; resolves when the extension answers, never rejects. With `watch`, the item is
   * followed from the moment it is queued, so `items[itemId]` holds its status.
   */
  add: (item: QueueItem, options?: { watch?: boolean }) => Promise<QueueAddOutcome>;
  /** Follows an item that was queued elsewhere: from now on `items[itemId]` holds its status. */
  watch: (itemId: string) => void;
  /** The last status of every followed item. */
  items: Record<string, QueueItemStatus>;
  /** Takes an item out of the queue; a running one is stopped. */
  cancel: (itemId: string) => void;
}

interface PendingAdd {
  resolve: (outcome: QueueAddOutcome) => void;
  ackTimer: ReturnType<typeof setTimeout>;
  watch: boolean;
}

/**
 * Talks to the queue of the ConceptFlow AI Bridge extension (docs/contracts/ai-bridge-v2.md,
 * "Hàng đợi"): adds items that the extension runs on DeepSeek, Gemini or Grok one after another,
 * even with this tab closed, follows how many are waiting, running or failed, and follows single
 * items (`watch`). `onItemDone` is called with the API path of every item that finished, so the
 * screen can load what the run stored. An extension without the queue never reports it, so
 * `available` stays false.
 */
export function useAiQueue(onItemDone?: (path: string) => void): AiQueue {
  const [counts, setCounts] = useState<QueueCounts | null>(null);
  const [outdated, setOutdated] = useState(false);
  const [defaultProvider, setDefaultProvider] = useState<AiBridgeProvider | null>(null);
  const [items, setItems] = useState<Record<string, QueueItemStatus>>({});
  const pending = useRef(new Map<string, PendingAdd>());
  const watched = useRef(new Set<string>());
  const finished = useRef(new Set<string>());
  const done = useRef(onItemDone);
  done.current = onItemDone;

  useEffect(() => {
    const adds = pending.current;
    function onMessage(event: MessageEvent) {
      if (isLegacyPong(event)) {
        setOutdated(true);
        return;
      }
      const msg = readExtensionMessage(event);
      if (!msg) return;
      if (msg.type === "pong") {
        setOutdated(false);
        setDefaultProvider(msg.default_provider ?? null);
        return;
      }
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
      if (msg.type === "queue_item") {
        const itemId = msg.item_id;
        if (!itemId) return;
        const status: QueueItemStatus = {
          itemId,
          path: msg.path ?? "",
          state: (msg.status as QueueItemState | undefined) ?? "waiting",
          stage: msg.stage ?? "",
          provider: msg.provider ?? "",
          error: msg.error ?? "",
          reply: msg.reply ?? "",
        };
        if (watched.current.has(itemId)) setItems((before) => ({ ...before, [itemId]: status }));
        if (status.state === "done" && !finished.current.has(itemId)) {
          finished.current.add(itemId);
          if (status.path) done.current?.(status.path);
        }
        return;
      }
      const add = msg.request_id ? adds.get(msg.request_id) : undefined;
      if (!add || !msg.request_id) return;
      if (msg.type !== "queue_added" && msg.type !== "error") return;
      clearTimeout(add.ackTimer);
      adds.delete(msg.request_id);
      if (msg.type === "queue_added" && msg.item_id) {
        // Followed from this very message, so no update of the item can arrive before it is watched.
        if (add.watch) watched.current.add(msg.item_id);
        add.resolve({ ok: true, added: msg.added !== false, itemId: msg.item_id });
      } else {
        add.resolve({ ok: false, message: msg.message || UNKNOWN_ERROR });
      }
    }
    window.addEventListener("message", onMessage);
    // The extension answers a ping with its queue state when it has a queue.
    pingExtension();
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
    (item: QueueItem, options?: { watch?: boolean }) =>
      new Promise<QueueAddOutcome>((resolve) => {
        const requestId = newRequestId();
        const ackTimer = setTimeout(() => {
          pending.current.delete(requestId);
          resolve({ ok: false, message: NO_ACK });
        }, ACK_TIMEOUT_MS);
        pending.current.set(requestId, { resolve, ackTimer, watch: options?.watch === true });
        post({ type: "queue_add", request_id: requestId, item });
      }),
    [],
  );

  const watch = useCallback((itemId: string) => {
    watched.current.add(itemId);
  }, []);

  const cancel = useCallback((itemId: string) => {
    post({ type: "cancel", item_id: itemId });
  }, []);

  return { available: counts !== null, outdated, counts, defaultProvider, add, watch, items, cancel };
}
