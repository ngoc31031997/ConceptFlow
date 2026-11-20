/**
 * Stands in for the AI Bridge browser extension (protocol v2, docs/contracts/ai-bridge-v2.md) in tests:
 * answers the page's ping, keeps the items the page queues, records cancels, and lets a test move an
 * item along (waiting, running on a provider, done, failed) the way the extension's worker does.
 */
export interface FakeQueuedItem {
  requestId: string;
  itemId: string;
  item: Record<string, unknown>;
}

export interface FakeAiBridge {
  /** Items the page added to the queue, in order. */
  queued: FakeQueuedItem[];
  /** Ids of the items the page asked to take out of the queue. */
  cancels: string[];
  /** Sends a raw extension message (protocol v2 unless `v` says otherwise). */
  send: (fields: Record<string, unknown>) => void;
  /** Announces where an item stands, as the extension does after every change. */
  announce: (itemId: string, fields: { status: string; stage?: string; provider?: string; error?: string; reply?: string }) => void;
  remove: () => void;
}

const SOURCE = "conceptflow-ai-bridge";

export interface FakeAiBridgeOptions {
  answersPing?: boolean;
  /** Reports a queue state with the pong, as an extension with the queue does. */
  queue?: boolean;
  /** Answers every add with this error text. */
  refuseQueue?: string;
  /** Behaves as the old extension: it answers only a v1 ping, with a v1 pong. */
  legacy?: boolean;
  defaultProvider?: string;
}

export function installFakeAiBridge(options: FakeAiBridgeOptions = {}): FakeAiBridge {
  const answersPing = options.answersPing ?? true;
  const fake: FakeAiBridge = {
    queued: [],
    cancels: [],
    // jsdom's postMessage leaves `source` and `origin` empty; a real browser sets both, and the page checks them.
    send: (fields) =>
      window.dispatchEvent(
        new MessageEvent("message", {
          data: { source: SOURCE, v: 2, from: "extension", ...fields },
          origin: window.location.origin,
          source: window,
        }),
      ),
    announce: (itemId, fields) => {
      const queued = fake.queued.find((q) => q.itemId === itemId);
      const request = (queued?.item.request ?? {}) as { path?: string };
      fake.send({
        type: "queue_item",
        item_id: itemId,
        path: request.path ?? "",
        project: String(queued?.item.project ?? ""),
        stage: "",
        provider: options.defaultProvider ?? "deepseek",
        error: "",
        reply: "",
        ...fields,
      });
    },
    remove: () => window.removeEventListener("message", onMessage),
  };
  function onMessage(event: MessageEvent) {
    const data = event.data as Record<string, unknown> | null;
    if (!data || data.source !== SOURCE || data.from !== "page") return;
    if (options.legacy) {
      if (data.type === "ping" && data.v === 1) fake.send({ v: 1, type: "pong", providers: ["deepseek"], version: "1.4.0" });
      return;
    }
    if (data.v !== 2) return;
    if (data.type === "ping" && answersPing) {
      fake.send({
        type: "pong",
        providers: ["deepseek", "gemini", "grok"],
        default_provider: options.defaultProvider ?? "deepseek",
        version: "2.0.0",
      });
      if (options.queue) fake.send({ type: "queue_state", waiting: 0, running: 0, done: 0, failed: 0, paused: false });
    }
    if (data.type === "queue_add") {
      const itemId = `q${fake.queued.length + 1}`;
      fake.queued.push({ requestId: String(data.request_id), itemId, item: data.item as Record<string, unknown> });
      if (options.refuseQueue) {
        fake.send({ type: "error", request_id: data.request_id, code: "queue_bad_item", message: options.refuseQueue });
        return;
      }
      fake.send({ type: "queue_added", request_id: data.request_id, item_id: itemId, added: true });
    }
    if (data.type === "cancel") fake.cancels.push(String(data.item_id));
  }
  window.addEventListener("message", onMessage);
  return fake;
}
