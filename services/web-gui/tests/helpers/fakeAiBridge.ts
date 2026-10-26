/**
 * Stands in for the AI Bridge browser extension in tests: answers the page's
 * ping, records every run and cancel, and lets a test send progress, a result
 * or an error back the way the extension's content script does.
 */
export interface FakeAiBridge {
  runs: { requestId: string; provider: string; prompt: string; modes: unknown }[];
  cancels: string[];
  send: (fields: Record<string, unknown>) => void;
  remove: () => void;
}

const SOURCE = "conceptflow-ai-bridge";

export function installFakeAiBridge(options: { answersPing?: boolean } = {}): FakeAiBridge {
  const answersPing = options.answersPing ?? true;
  const fake: FakeAiBridge = {
    runs: [],
    cancels: [],
    // jsdom's postMessage leaves `source` and `origin` empty; a real browser sets both, and the page checks them.
    send: (fields) =>
      window.dispatchEvent(
        new MessageEvent("message", {
          data: { source: SOURCE, v: 1, from: "extension", ...fields },
          origin: window.location.origin,
          source: window,
        }),
      ),
    remove: () => window.removeEventListener("message", onMessage),
  };
  function onMessage(event: MessageEvent) {
    const data = event.data as Record<string, unknown> | null;
    if (!data || data.source !== SOURCE || data.from !== "page") return;
    if (data.type === "ping" && answersPing)
      fake.send({ type: "pong", providers: ["deepseek"], version: "1.0.0" });
    if (data.type === "run")
      fake.runs.push({
        requestId: String(data.request_id),
        provider: String(data.provider),
        prompt: String(data.prompt),
        modes: data.modes,
      });
    if (data.type === "cancel") fake.cancels.push(String(data.request_id));
  }
  window.addEventListener("message", onMessage);
  return fake;
}
