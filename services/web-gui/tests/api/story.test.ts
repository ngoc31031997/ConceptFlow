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

const storyWire = {
  central_question: "Vì sao?",
  core_answer: "Vì thế.",
  narrative_arc: [
    { beat: "Hook", text: "Mở", facts: null },
    { beat: "Payoff", text: "Kết", facts: ["F01"] },
  ],
  creator_fields: ["core_answer"],
  status: "pending",
  updated_at: "2026-10-24T10:00:00Z",
};

const stateWire = {
  story: storyWire,
  facts: [{ id: "F01", claim_vi: "Sung là chùm hoa", type: "fact", must: true, cited: true, link: "none" }],
  brief: { audience: "Học sinh", duration_seconds: 300, tone: "Ấm", learning_objectives: null, avoid: ["Thuật ngữ"] },
  brief_approved: true,
  research_resolved: true,
  research_skipped: false,
  brief_stale: true,
  research_stale: false,
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
    expect(state.status).toBe("pending");
    expect(state.story).toMatchObject({ centralQuestion: "Vì sao?", coreAnswer: "Vì thế.", creatorFields: ["core_answer"] });
    expect(state.story?.narrativeArc).toEqual([
      { beat: "Hook", text: "Mở", facts: [] },
      { beat: "Payoff", text: "Kết", facts: ["F01"] },
    ]);
    expect(state.facts[0]).toMatchObject({ claimVi: "Sung là chùm hoa", must: true, cited: true, link: "none" });
    expect(state.brief).toMatchObject({ audience: "Học sinh", durationSeconds: 300, learningObjectives: [], avoid: ["Thuật ngữ"] });
    expect(state.briefStale).toBe(true);
    expect(state.researchStale).toBe(false);
  });

  it("reads a project with no story and no approved brief", async () => {
    mockFetch(200, { ...stateWire, story: null, facts: null, brief: null, brief_approved: false, research_resolved: false });
    const state = await client.getStory("p1");
    expect(state).toMatchObject({ story: null, facts: [], brief: null, status: "pending", briefApproved: false });
  });

  it("starts the AI run with the note and the operation id", async () => {
    const fn = mockFetch(200, storyWire);
    const story = await client.generateStory("p1", "ngắn hơn", { id: "op-1", signal: new AbortController().signal });
    expect(story.narrativeArc).toHaveLength(2);
    expect(lastCall(fn)).toMatchObject({ path: "story/generate", method: "POST", body: { note: "ngắn hơn" } });
    expect(lastCall(fn).headers?.["X-Operation-Id"]).toBe("op-1");
  });

  it("sends only the fields of a hand edit", async () => {
    const fn = mockFetch(200, storyWire);
    await client.editStory("p1", { coreAnswer: "Đáp án" });
    expect(lastCall(fn)).toMatchObject({ path: "story", method: "PUT", body: { core_answer: "Đáp án" } });
    await client.editStory("p1", { narrativeArc: [{ beat: "A", text: "a", facts: ["F01"] }] });
    expect(lastCall(fn).body).toEqual({ narrative_arc: [{ beat: "A", text: "a", facts: ["F01"] }] });
  });

  it("approves and takes the approval back", async () => {
    let fn = mockFetch(200, { ...storyWire, status: "approved" });
    expect((await client.approveStory("p1")).status).toBe("approved");
    expect(lastCall(fn)).toMatchObject({ path: "story/approve", method: "POST" });
    fn = mockFetch(200, storyWire);
    expect((await client.unapproveStory("p1")).status).toBe("pending");
    expect(lastCall(fn)).toMatchObject({ path: "story/unapprove", method: "POST" });
  });

  it("names the fields of the story", () => {
    expect(client.STORY_FIELD_KEYS).toEqual(["central_question", "core_answer", "narrative_arc"]);
  });

  it("raises the server's words when the step is not open", async () => {
    mockFetch(409, { error: "duyệt brief ở bước Creative brief trước" });
    await expect(client.generateStory("p1", "")).rejects.toThrow(/duyệt brief/);
  });
});
