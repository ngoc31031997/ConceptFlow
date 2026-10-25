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
  return { path: url.replace(GATEWAY, ""), method: init?.method ?? "GET", body: init?.body ? JSON.parse(String(init.body)) : undefined };
}

const stateWire = {
  status: "pending",
  story_approved: true,
  beats: [],
  facts: [],
  input: { topic: "Đề tài", brief: "Khán giả: a", brief_approved: true, story: "question (Câu hỏi trung tâm):\nQ" },
  checks: [],
  total_seconds: 0,
  target_label: "4–5 phút",
  phase: "empty",
  unwritten: 0,
  rebalanced: 0,
  script_approved: 0,
};

describe("beat sheet api", () => {
  const originalFetch = global.fetch;
  afterEach(() => {
    global.fetch = originalFetch;
  });

  it("reads the input the AI gets with the state", async () => {
    mockFetch(200, stateWire);
    const state = await client.getBeats("p1");
    expect(state.input).toEqual({ topic: "Đề tài", brief: "Khán giả: a", briefApproved: true, story: "question (Câu hỏi trung tâm):\nQ" });
  });

  it("reads an input the server left out as empty", async () => {
    mockFetch(200, { ...stateWire, input: null });
    expect((await client.getBeats("p1")).input).toEqual({ topic: "", brief: "", briefApproved: false, story: "" });
  });

  it("edits, redoes, approves and unapproves one beat on its own route", async () => {
    const fetchMock = mockFetch(200, stateWire);
    await client.editBeat("p1", "b2", { dur: "25" });
    expect(lastCall(fetchMock)).toMatchObject({ path: "beats/b2", method: "PUT", body: { fields: { dur: "25" } } });
    await client.redoBeat("p1", "b2", "ngắn hơn");
    expect(lastCall(fetchMock)).toMatchObject({ path: "beats/b2/redo", method: "POST", body: { note: "ngắn hơn" } });
    await client.approveBeat("p1", "b2");
    expect(lastCall(fetchMock)).toMatchObject({ path: "beats/b2/approve", method: "POST" });
    await client.unapproveBeat("p1", "b2");
    expect(lastCall(fetchMock)).toMatchObject({ path: "beats/b2/unapprove", method: "POST" });
  });

  it("lists what a change to a beat sends back", async () => {
    const fetchMock = mockFetch(200, { items: ["Viết kịch bản: lời thoại của beat này (2 câu) cần xem lại"] });
    expect(await client.getBeatImpact("p1", { kind: "beat", key: "b2" })).toHaveLength(1);
    expect(lastCall(fetchMock)).toMatchObject({ path: "beats/b2/impact", method: "GET" });
  });
});
