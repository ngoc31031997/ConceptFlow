import { useEffect, useState } from "react";

/** Tag every message of the page ↔ extension protocol carries (docs/contracts/ai-bridge-v2.md). */
const BRIDGE_SOURCE = "conceptflow-ai-bridge";
const BRIDGE_VERSION = 2;
/** The protocol of the extension before the queue became the only way to run; its pong marks an extension to update. */
const LEGACY_VERSION = 1;

/** The outside AI sites the extension can drive. */
export type AiBridgeProvider = "deepseek" | "gemini" | "grok";

/** Display names of the providers, for text that names the one an item runs on. */
export const PROVIDER_NAMES: Record<AiBridgeProvider, string> = {
  deepseek: "DeepSeek",
  gemini: "Gemini",
  grok: "Grok",
};

/** The name to show for a provider id the extension reported; an unknown id is shown as is. */
export function providerName(id: string): string {
  return id in PROVIDER_NAMES ? PROVIDER_NAMES[id as AiBridgeProvider] : id;
}

/** Which DeepSeek modes an item turns on; the other providers ignore them. */
export interface AiBridgeModes {
  deepThink: boolean;
  search: boolean;
}

/** What the screen gets from useAiBridge. */
export interface AiBridge {
  /** True once the extension has answered on this page. */
  available: boolean;
  /** True when an extension answered with the old protocol: it has to be updated before it can run anything. */
  outdated: boolean;
  /** Providers the installed extension supports. */
  providers: AiBridgeProvider[];
  /** The provider an item without its own choice runs on. */
  defaultProvider: AiBridgeProvider | null;
}

export interface ExtensionMessage {
  type: "pong" | "error" | "queue_added" | "queue_state" | "queue_item";
  request_id?: string;
  item_id?: string;
  added?: boolean;
  path?: string;
  project?: string;
  status?: string;
  stage?: string;
  provider?: string;
  error?: string;
  reply?: string;
  waiting?: number;
  running?: number;
  done?: number;
  failed?: number;
  paused?: boolean;
  providers?: AiBridgeProvider[];
  default_provider?: AiBridgeProvider;
  message?: string;
}

/** The message of an extension on this page, or null for anything else (other scripts, other origins, another protocol version). */
export function readExtensionMessage(event: MessageEvent): ExtensionMessage | null {
  const data = ownMessage(event, BRIDGE_VERSION);
  return data as unknown as ExtensionMessage | null;
}

function ownMessage(event: MessageEvent, version: number): Record<string, unknown> | null {
  if (event.source !== window || event.origin !== window.location.origin) return null;
  const data = event.data as Record<string, unknown> | null;
  if (!data || data.source !== BRIDGE_SOURCE || data.v !== version || data.from !== "extension") return null;
  return data;
}

export function post(fields: Record<string, unknown>) {
  window.postMessage(
    { source: BRIDGE_SOURCE, v: BRIDGE_VERSION, from: "page", ...fields },
    window.location.origin,
  );
}

/**
 * Pings in both protocol versions, so an extension that still speaks the old one
 * (which ignores the new ping) answers and can be told to update.
 */
export function pingExtension() {
  post({ type: "ping" });
  window.postMessage(
    { source: BRIDGE_SOURCE, v: LEGACY_VERSION, from: "page", type: "ping" },
    window.location.origin,
  );
}

/** True when this message is the pong of an extension with the old protocol. */
export function isLegacyPong(event: MessageEvent): boolean {
  const data = ownMessage(event, LEGACY_VERSION);
  return data !== null && data.type === "pong";
}

export function newRequestId(): string {
  return typeof crypto !== "undefined" && "randomUUID" in crypto
    ? crypto.randomUUID()
    : `${Date.now()}-${Math.random()}`;
}

/**
 * Detects the ConceptFlow AI Bridge browser extension (tools/ai-bridge-extension),
 * which runs prompts on the outside AI websites the Creator is signed in to. It
 * pings on mount and is `available` once the extension answers; without the
 * extension nothing changes on the page. Running is done through the queue
 * (`useAiQueue`).
 */
export function useAiBridge(): AiBridge {
  const [providers, setProviders] = useState<AiBridgeProvider[]>([]);
  const [defaultProvider, setDefaultProvider] = useState<AiBridgeProvider | null>(null);
  const [outdated, setOutdated] = useState(false);

  useEffect(() => {
    function onMessage(event: MessageEvent) {
      if (isLegacyPong(event)) {
        setOutdated(true);
        return;
      }
      const msg = readExtensionMessage(event);
      if (!msg || msg.type !== "pong") return;
      setProviders(Array.isArray(msg.providers) ? msg.providers : []);
      setDefaultProvider(msg.default_provider ?? null);
      setOutdated(false);
    }
    window.addEventListener("message", onMessage);
    pingExtension();
    return () => window.removeEventListener("message", onMessage);
  }, []);

  return { available: providers.length > 0, outdated, providers, defaultProvider };
}
