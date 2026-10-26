import { afterEach, describe, expect, it, vi } from "vitest";
import * as client from "../../src/api/client";

function mockFetch(status: number, body?: unknown) {
  const fn = vi.fn().mockResolvedValue({
    ok: status >= 200 && status < 300,
    status,
    json: async () => body,
  });
  global.fetch = fn as unknown as typeof fetch;
  return fn;
}

function lastRequest(fn: ReturnType<typeof vi.fn>) {
  const [, init] = fn.mock.calls[fn.mock.calls.length - 1] as [string, RequestInit];
  return {
    headers: init.headers as Record<string, string>,
    body: init.body ? JSON.parse(String(init.body)) : undefined,
    signal: init.signal,
  };
}

afterEach(() => vi.restoreAllMocks());

describe("AI calls through an AI outside the system", () => {
  it("asks for the prompt with the mode header and rejects with it", async () => {
    const fetchFn = mockFetch(200, { external_prompt: "Viết câu chuyện\n\nLàm đi" });
    const error = await client
      .generateStory("p1", "ngắn hơn", { external: { mode: "prompt" } })
      .catch((e: unknown) => e);
    expect(error).toBeInstanceOf(client.ExternalPromptReady);
    expect((error as client.ExternalPromptReady).prompt).toBe("Viết câu chuyện\n\nLàm đi");
    const req = lastRequest(fetchFn);
    expect(req.headers["X-External-AI"]).toBe("prompt");
    expect(req.headers["X-Operation-Id"]).toBeUndefined();
    expect(req.body).toEqual({ note: "ngắn hơn" });
  });

  it("sends the pasted reply next to the step's own fields", async () => {
    const fetchFn = mockFetch(200, {
      central_question: "Vì sao?",
      core_answer: "Vì thế.",
      narrative_arc: [],
      creator_fields: [],
      status: "pending",
      updated_at: "2026-10-24T10:00:00Z",
    });
    await client.generateStory("p1", "", { external: { mode: "paste", reply: '{"a":1}' } });
    const req = lastRequest(fetchFn);
    expect(req.headers["X-External-AI"]).toBe("paste");
    expect(req.body).toEqual({ note: "", external_reply: '{"a":1}' });
  });

  it("carries the reply on a call that has no fields of its own", async () => {
    const fetchFn = mockFetch(200, { review: null });
    await client
      .reviewProjectIdea("p1", { external: { mode: "paste", reply: "kết quả" } })
      .catch(() => undefined);
    expect(lastRequest(fetchFn).body).toEqual({ external_reply: "kết quả" });
  });

  it("fails clearly when the step answers without a prompt", async () => {
    mockFetch(200, {});
    await expect(client.generateStory("p1", "", { external: { mode: "prompt" } })).rejects.toThrow(
      /không có prompt/,
    );
  });

  it("leaves a followed call as it was", async () => {
    const fetchFn = mockFetch(200, {
      central_question: "",
      core_answer: "",
      narrative_arc: [],
      creator_fields: [],
      status: "pending",
      updated_at: "",
    });
    const controller = new AbortController();
    await client.generateStory("p1", "ghi chú", { id: "op1", signal: controller.signal });
    const req = lastRequest(fetchFn);
    expect(req.headers["X-Operation-Id"]).toBe("op1");
    expect(req.headers["X-External-AI"]).toBeUndefined();
    expect(req.body).toEqual({ note: "ghi chú" });
    expect(req.signal).toBe(controller.signal);
  });
});

describe("replaceInScript", () => {
  it("posts the word, the replacement, where to write it and whether to apply", async () => {
    const fetchFn = mockFetch(200, { matches: null, skipped_approved: [{ beat: "b3", lines: 2 }], applied: false });
    const result = await client.replaceInScript("p1", { find: "TRNG", replace: "TI A EN GI", field: "spoken", matchCase: true, apply: false });
    const req = lastRequest(fetchFn);
    expect(req.body).toEqual({ find: "TRNG", replace: "TI A EN GI", field: "spoken", match_case: true, apply: false });
    expect(result).toEqual({ matches: [], skippedApproved: [{ beat: "b3", lines: 2 }], applied: false });
  });
});
