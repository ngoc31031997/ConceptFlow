import { useCallback, useEffect, useRef, useState } from "react";

/** Tag every message of the page ↔ extension protocol carries (docs/contracts/ai-bridge-v1.md). */
const BRIDGE_SOURCE = "conceptflow-ai-bridge";
const BRIDGE_VERSION = 1;
/** A run the extension has not acknowledged with a first progress message within this time failed to reach it. */
const ACK_TIMEOUT_MS = 5000;

/** The outside AI sites the extension can drive. */
export type AiBridgeProvider = "deepseek";

/** Where a run stands, as the extension reports it. */
export type AiBridgeStage = "opening" | "sending" | "answering" | "reading";

/** Which DeepSeek modes a run turns on. */
export interface AiBridgeModes {
  deepThink: boolean;
  search: boolean;
}

/** How a run ended: the answer, or the reason it has none. */
export type AiBridgeOutcome =
  { ok: true; text: string } | { ok: false; cancelled: boolean; message: string };

/** What the screen gets from useAiBridge. */
export interface AiBridge {
  /** True once the extension has answered on this page. */
  available: boolean;
  /** Providers the installed extension supports. */
  providers: AiBridgeProvider[];
  /** The stage of the run in progress, or null when nothing runs. */
  stage: AiBridgeStage | null;
  /** Sends the prompt to the provider and resolves with its answer or the error; never rejects. */
  run: (
    provider: AiBridgeProvider,
    prompt: string,
    modes: AiBridgeModes,
  ) => Promise<AiBridgeOutcome>;
  /** Stops waiting for the run in progress, which then resolves as cancelled. */
  cancel: () => void;
}

export interface ExtensionMessage {
  type: "pong" | "progress" | "result" | "error" | "queue_added" | "queue_state" | "queue_item_done";
  request_id?: string;
  item_id?: string;
  added?: boolean;
  path?: string;
  waiting?: number;
  running?: number;
  done?: number;
  failed?: number;
  paused?: boolean;
  providers?: AiBridgeProvider[];
  stage?: AiBridgeStage;
  text?: string;
  message?: string;
}

interface Pending {
  requestId: string;
  resolve: (outcome: AiBridgeOutcome) => void;
  ackTimer: ReturnType<typeof setTimeout>;
}

const NO_ACK =
  "Extension không phản hồi. Tải lại trang hoặc kiểm tra extension đã bật trong chrome://extensions.";
const UNKNOWN_ERROR = "Extension báo lỗi nhưng không nói lý do.";

export function readExtensionMessage(event: MessageEvent): ExtensionMessage | null {
  if (event.source !== window || event.origin !== window.location.origin) return null;
  const data = event.data as Record<string, unknown> | null;
  if (
    !data ||
    data.source !== BRIDGE_SOURCE ||
    data.v !== BRIDGE_VERSION ||
    data.from !== "extension"
  )
    return null;
  return data as unknown as ExtensionMessage;
}

export function post(fields: Record<string, unknown>) {
  window.postMessage(
    { source: BRIDGE_SOURCE, v: BRIDGE_VERSION, from: "page", ...fields },
    window.location.origin,
  );
}

export function newRequestId(): string {
  return typeof crypto !== "undefined" && "randomUUID" in crypto
    ? crypto.randomUUID()
    : `${Date.now()}-${Math.random()}`;
}

/**
 * Talks to the ConceptFlow AI Bridge browser extension (tools/ai-bridge-extension),
 * which runs a prompt on an outside AI website the Creator is signed in to and
 * brings the answer back. It pings on mount and is `available` once the
 * extension answers; without the extension nothing changes on the page.
 * One run at a time; messages for any other request id are ignored.
 */
export function useAiBridge(): AiBridge {
  const [providers, setProviders] = useState<AiBridgeProvider[]>([]);
  const [stage, setStage] = useState<AiBridgeStage | null>(null);
  const pending = useRef<Pending | null>(null);

  const finish = useCallback((outcome: AiBridgeOutcome) => {
    const current = pending.current;
    if (!current) return;
    clearTimeout(current.ackTimer);
    pending.current = null;
    setStage(null);
    current.resolve(outcome);
  }, []);

  useEffect(() => {
    function onMessage(event: MessageEvent) {
      const msg = readExtensionMessage(event);
      if (!msg) return;
      if (msg.type === "pong") {
        setProviders(Array.isArray(msg.providers) ? msg.providers : []);
        return;
      }
      const current = pending.current;
      if (!current || msg.request_id !== current.requestId) return;
      clearTimeout(current.ackTimer);
      if (msg.type === "progress" && msg.stage) setStage(msg.stage);
      else if (msg.type === "result") finish({ ok: true, text: msg.text ?? "" });
      else if (msg.type === "error")
        finish({ ok: false, cancelled: false, message: msg.message || UNKNOWN_ERROR });
    }
    window.addEventListener("message", onMessage);
    post({ type: "ping" });
    return () => {
      window.removeEventListener("message", onMessage);
      const current = pending.current;
      if (current) {
        clearTimeout(current.ackTimer);
        post({ type: "cancel", request_id: current.requestId });
        pending.current = null;
      }
    };
  }, [finish]);

  const run = useCallback(
    (provider: AiBridgeProvider, prompt: string, modes: AiBridgeModes) =>
      new Promise<AiBridgeOutcome>((resolve) => {
        if (pending.current) {
          resolve({ ok: false, cancelled: false, message: "Đang có một lượt chạy khác." });
          return;
        }
        const requestId = newRequestId();
        const ackTimer = setTimeout(
          () => finish({ ok: false, cancelled: false, message: NO_ACK }),
          ACK_TIMEOUT_MS,
        );
        pending.current = { requestId, resolve, ackTimer };
        setStage("opening");
        post({
          type: "run",
          request_id: requestId,
          provider,
          prompt,
          modes: { deep_think: modes.deepThink, search: modes.search },
        });
      }),
    [finish],
  );

  const cancel = useCallback(() => {
    const current = pending.current;
    if (!current) return;
    post({ type: "cancel", request_id: current.requestId });
    finish({ ok: false, cancelled: true, message: "" });
  }, [finish]);

  return { available: providers.length > 0, providers, stage, run, cancel };
}
