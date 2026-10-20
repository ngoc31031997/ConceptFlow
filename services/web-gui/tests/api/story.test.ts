import { describe, it, expect, vi, afterEach } from "vitest";
import * as client from "../../src/api/client";

const GATEWAY = /^.*\/v1\/projects\/p1\//;

function mockFetch(status: number, body?: unknown) {
  const fn = vi.fn().mockResolvedValue({
    ok: status >= 200 && status < 300,
    status,
    json: async () => body,
  });
  global.fetch = fn as unknown as typeof fetch;
  return fn;
}

function lastCall(fn: ReturnType<typeof vi.fn>) {
  const [url, init] = fn.mock.calls[fn.mock.calls.length - 1] as [string, RequestInit | undefined];
  return {
    path: url.replace(GATEWAY, ""),
    method: init?.method ?? "GET",
    body: init?.body ? JSON.parse(String(init.body)) : undefined,
    headers: init?.headers as Record<string, string> | undefined,
  };
}

const stateWire = {
  status: "pending",
  approach_key: "A2",
  fields: [
    { key: "question", label: "Câu hỏi mở đầu", text: "Vì sao?", facts: null, mine: true, stale: false },
    { key: "chain", label: "Chuỗi nhân quả", text: "Bước một [F01]", facts: ["F01"], mine: false, stale: true },
  ],
  approaches: [
    { key: "A2", code: "VÒNG-ĐỜI", name: "Vòng đời", angle: "theo ong", why: "F04", refs: null, open: "Câu chuyện...", is_new: true, archetype_id: "", selected: true },
  ],
  facts: [{ id: "F01", claim_vi: "Sung là chùm hoa", type: "fact", must: true, used: true, cited: true, link: "none" }],
  brief_approved: true,
  research_resolved: true,
  research_skipped: true,
};

describe("story api", () => {
  const originalFetch = global.fetch;
  afterEach(() => {
    global.fetch = originalFetch;
  });

  it("reads the story state with empty lists where the server sent null", async () => {
    const fn = mockFetch(200, stateWire);
    const state = await client.getStory("p1");
    expect(lastCall(fn)).toMatchObject({ path: "story", method: "GET" });
    expect(state.approachKey).toBe("A2");
    expect(state.fields[0]).toMatchObject({ key: "question", mine: true, facts: [] });
    expect(state.fields[1]).toMatchObject({ stale: true, facts: ["F01"] });
    expect(state.approaches[0]).toMatchObject({ isNew: true, archetypeId: "", refs: [], selected: true });
    expect(state.facts[0]).toMatchObject({ claimVi: "Sung là chùm hoa", must: true, link: "none" });
    expect(state.researchSkipped).toBe(true);
  });

  it("starts the AI runs with the note and the operation id", async () => {
    const fn = mockFetch(200, { approaches: [stateWire.approaches[0]] });
    const approaches = await client.generateStoryApproaches("p1", "thử cách khác", { id: "op-1", signal: new AbortController().signal });
    expect(approaches[0].code).toBe("VÒNG-ĐỜI");
    expect(lastCall(fn)).toMatchObject({ path: "story/approaches/generate", method: "POST", body: { note: "thử cách khác" } });
    expect(lastCall(fn).headers?.["X-Operation-Id"]).toBe("op-1");

    const gen = mockFetch(200, stateWire);
    const state = await client.generateStory("p1", "", { id: "op-2", signal: new AbortController().signal });
    expect(state.fields).toHaveLength(2);
    expect(lastCall(gen)).toMatchObject({ path: "story/generate", method: "POST", body: { note: "" } });
    expect(lastCall(gen).headers?.["X-Operation-Id"]).toBe("op-2");
  });

  it("chooses an approach, saves a new kind and turns a fact on or off", async () => {
    let fn = mockFetch(204);
    await client.selectStoryApproach("p1", "A3");
    expect(lastCall(fn)).toMatchObject({ path: "story/approach", method: "PUT", body: { key: "A3" } });

    fn = mockFetch(200, { ...stateWire.approaches[0], archetype_id: "row-1" });
    const saved = await client.saveApproachAsArchetype("p1", "A2");
    expect(lastCall(fn)).toMatchObject({ path: "story/approaches/A2/archetype", method: "POST" });
    expect(saved.archetypeId).toBe("row-1");

    fn = mockFetch(204);
    await client.setStoryFactUse("p1", "F02", false);
    expect(lastCall(fn)).toMatchObject({ path: "story/facts/F02", method: "PUT", body: { use: false } });
  });

  it("addresses the story fields and the gate as review units", () => {
    expect(client.STORY_UNIT).toMatchObject({ kind: "story", key: "main" });
    expect(client.storyFieldUnit("payoff")).toMatchObject({ kind: "story_field", key: "payoff" });
    expect(client.STORY_FIELD_KEYS).toEqual(["question", "promise", "chain", "payoff", "logline", "summary"]);
  });

  it("raises the server's words when the step is not open", async () => {
    mockFetch(409, { error: "duyệt brief ở bước Ý tưởng trước" });
    await expect(client.generateStory("p1", "")).rejects.toThrow(/duyệt brief/);
  });
});
