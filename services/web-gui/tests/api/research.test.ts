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
  return { path: url.replace(GATEWAY, ""), method: init?.method ?? "GET", body: init?.body ? JSON.parse(String(init.body)) : undefined, headers: init?.headers as Record<string, string> | undefined };
}

const factWire = {
  id: "F01",
  question: "Q1",
  claim_vi: "Quả sung là chùm hoa",
  type: "fact",
  source_title: "Syconium",
  publisher: "Wikipedia",
  year: "2025",
  url: "https://en.wikipedia.org/wiki/Syconium",
  quote: "The syconium is a fleshy receptacle.",
  confidence: "high",
  note: "",
  must: true,
  keep: true,
  link: { state: "bad", why: "không có đoạn trích", nearest: "câu gần nhất", checked_at: "2026-10-18T10:00:00Z" },
  mine: null,
};

describe("research api", () => {
  const originalFetch = global.fetch;
  afterEach(() => {
    global.fetch = originalFetch;
    vi.restoreAllMocks();
  });

  it("reads the whole screen, filling lists the server left null", async () => {
    mockFetch(200, {
      gate: { status: "pending", skipped: false, resolved: false },
      questions: [{ id: "Q1", text: "Một?", mine: true, stale: false }],
      facts: [factWire],
      pastes: [{ n: 1, at: "2026-10-18T10:00:00Z", added: ["F01"], updated: null, rejected: null }],
      coverage: [{ question: "Q1", sources: 1, bad_facts: null, weak: true }],
      brief_approved: true,
      questions_stale: true,
      idea: { central_question: "Vì sao?", curiosity_points: null, scope: "Phạm vi" },
      brief: { audience: "Khán giả", duration_seconds: 270, tone: "Tò mò", learning_objectives: ["Một"], avoid: null },
    });

    const state = await client.getResearch("p1");
    expect(state.idea).toEqual({ centralQuestion: "Vì sao?", curiosityPoints: [], scope: "Phạm vi" });
    expect(state.brief).toEqual({ audience: "Khán giả", durationSeconds: 270, tone: "Tò mò", learningObjectives: ["Một"], avoid: [] });

    expect(state.briefApproved).toBe(true);
    expect(state.questionsStale).toBe(true);
    expect(state.questions[0]).toEqual({ id: "Q1", text: "Một?", mine: true, stale: false });
    expect(state.facts[0]).toMatchObject({
      id: "F01",
      claimVi: "Quả sung là chùm hoa",
      sourceTitle: "Syconium",
      mine: [],
      link: { state: "bad", why: "không có đoạn trích", nearest: "câu gần nhất", checkedAt: "2026-10-18T10:00:00Z" },
    });
    expect(state.pastes[0]).toEqual({ n: 1, at: "2026-10-18T10:00:00Z", added: ["F01"], updated: [], rejected: [] });
    expect(state.coverage[0]).toEqual({ question: "Q1", sources: 1, badFacts: [], weak: true });
  });

  it("treats a missing link detail as empty text", async () => {
    mockFetch(200, {
      gate: { status: "pending", skipped: false, resolved: false },
      questions: null,
      facts: [{ ...factWire, link: { state: "run" } }],
      pastes: null,
      coverage: null,
      brief_approved: false,
      questions_stale: false,
      idea: null,
      brief: null,
    });
    const state = await client.getResearch("p1");
    expect(state.questions).toEqual([]);
    expect(state.facts[0].link).toEqual({ state: "run", why: "", nearest: "", checkedAt: "" });
  });

  it("reports a request the Creator aborted as a cancellation, not a connection failure", async () => {
    const controller = new AbortController();
    global.fetch = vi.fn().mockImplementation(() => {
      controller.abort();
      return Promise.reject(new DOMException("aborted", "AbortError"));
    }) as unknown as typeof fetch;
    const err = await client.generateResearchQuestions("p1", { replaceEdited: false }, { id: "op-1", signal: controller.signal }).catch((e: unknown) => e);
    expect(client.isRequestCancelled(err)).toBe(true);
  });

  it("generates questions with the replace flag, the note and the operation id", async () => {
    const fn = mockFetch(200, { questions: [{ id: "Q7", text: "Mới?", mine: false, stale: false }] });
    const questions = await client.generateResearchQuestions("p1", { replaceEdited: true, note: "cơ chế" }, { id: "op-9", signal: new AbortController().signal });
    expect(questions).toHaveLength(1);
    const call = lastCall(fn);
    expect(call.path).toBe("research/questions/generate");
    expect(call.method).toBe("POST");
    expect(call.body).toEqual({ replace_edited: true, note: "cơ chế" });
    expect(call.headers?.["X-Operation-Id"]).toBe("op-9");
  });

  it("adds, edits and deletes a question", async () => {
    let fn = mockFetch(201, { id: "Q2", text: "Mới?", mine: true, stale: false });
    expect(await client.addResearchQuestion("p1", "Mới?")).toMatchObject({ id: "Q2", mine: true });
    expect(lastCall(fn)).toMatchObject({ path: "research/questions", method: "POST", body: { text: "Mới?" } });

    fn = mockFetch(200, { project_id: "p1", kind: "research_question", key: "Q2", status: "pending", content: {}, creator_fields: ["text"], redo_count: 0, updated_at: "" });
    await client.editResearchQuestion("p1", "Q2", "Sửa rồi?");
    expect(lastCall(fn)).toMatchObject({ path: "units/research_question/Q2", method: "PUT", body: { fields: { text: "Sửa rồi?" } } });

    fn = mockFetch(204);
    await client.deleteResearchQuestion("p1", "Q2");
    expect(lastCall(fn)).toMatchObject({ path: "research/questions/Q2", method: "DELETE" });
  });

  it("reads the two prompts", async () => {
    let fn = mockFetch(200, { prompt: "PROMPT" });
    expect(await client.getResearchPrompt("p1")).toBe("PROMPT");
    expect(lastCall(fn).path).toBe("research/prompt");
    fn = mockFetch(200, { prompt: "FOLLOW" });
    expect(await client.getFollowUpPrompt("p1")).toBe("FOLLOW");
    expect(lastCall(fn).path).toBe("research/follow-up-prompt");
  });

  it("pastes an answer and reads what was added, updated and left out", async () => {
    const fn = mockFetch(200, {
      added: ["F01", "F02"],
      updated: null,
      rejected: [{ row: 3, reason: "thiếu type" }],
      log: { n: 1, at: "2026-10-18T10:00:00Z", added: ["F01", "F02"], updated: null, rejected: [{ row: 3, reason: "thiếu type" }] },
    });
    const result = await client.pasteResearch("p1", "câu trả lời");
    expect(lastCall(fn)).toMatchObject({ path: "research/paste", method: "POST", body: { raw: "câu trả lời" } });
    expect(result).toEqual({
      added: ["F01", "F02"],
      updated: [],
      rejected: [{ row: 3, reason: "thiếu type" }],
      log: { n: 1, at: "2026-10-18T10:00:00Z", added: ["F01", "F02"], updated: [], rejected: [{ row: 3, reason: "thiếu type" }] },
    });
  });

  it("adds a fact by hand with snake_case names", async () => {
    const fn = mockFetch(201, factWire);
    const fact = await client.addFact("p1", { question: "Q1", claimVi: "Tay", type: "hypothesis", url: "https://a.org/" });
    expect(lastCall(fn)).toMatchObject({
      path: "research/facts",
      method: "POST",
      body: { question: "Q1", claim_vi: "Tay", type: "hypothesis", source_title: "", publisher: "", year: "", url: "https://a.org/", quote: "" },
    });
    expect(fact.id).toBe("F01");
  });

  it("changes a fact's flags, edits its text and asks what it sends back", async () => {
    let fn = mockFetch(200, factWire);
    await client.setFactFlags("p1", "F01", { type: "misconception", must: true });
    expect(lastCall(fn)).toMatchObject({ path: "research/facts/F01", method: "PUT", body: { type: "misconception", must: true } });

    fn = mockFetch(200, { project_id: "p1", kind: "fact", key: "F01", status: "pending", content: {}, creator_fields: [], redo_count: 0, updated_at: "" });
    await client.editFact("p1", "F01", { claim_vi: "Đổi", url: "https://b.org/" });
    expect(lastCall(fn)).toMatchObject({ path: "units/fact/F01", method: "PUT", body: { fields: { claim_vi: "Đổi", url: "https://b.org/" } } });

    fn = mockFetch(200, { items: ["Beat B2 dùng F01"] });
    expect(await client.getFactImpact("p1", "F01")).toEqual(["Beat B2 dùng F01"]);
    expect(lastCall(fn).path).toBe("units/fact/F01/impact");
  });

  it("verifies facts, skips the step and approves the table", async () => {
    let fn = mockFetch(202);
    await client.verifyFacts("p1", ["F01", "F02"]);
    expect(lastCall(fn)).toMatchObject({ path: "research/verify", method: "POST", body: { fact_ids: ["F01", "F02"] } });

    fn = mockFetch(200, { status: "pending", skipped: true, resolved: true });
    expect(await client.setResearchSkipped("p1", true)).toEqual({ status: "pending", skipped: true, resolved: true });
    expect(lastCall(fn)).toMatchObject({ path: "research/skip", method: "PUT", body: { skipped: true } });

    fn = mockFetch(200, { project_id: "p1", kind: "research", key: "main", status: "approved", content: {}, creator_fields: [], redo_count: 0, updated_at: "" });
    await client.approveResearch("p1");
    expect(lastCall(fn)).toMatchObject({ path: "units/research/main/approve", method: "POST" });
    await client.unapproveResearch("p1");
    expect(lastCall(fn)).toMatchObject({ path: "units/research/main/unapprove", method: "POST" });
  });

  it("raises the server's message when a request is refused", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 400,
      json: async () => ({ error: "không tìm thấy khối JSON conceptflow.research.v1 trong đoạn đã dán" }),
    }) as unknown as typeof fetch;
    await expect(client.pasteResearch("p1", "x")).rejects.toMatchObject({
      status: 400,
      message: expect.stringContaining("không tìm thấy khối JSON"),
    });
  });

  it("carries the research model with the other step models", async () => {
    const fn = mockFetch(204);
    await client.saveAuthoringModels("p1", { brief: "b", research: "ollama", story: "", storyboard: "", code: "" });
    expect(lastCall(fn).body).toMatchObject({ brief: "b", research: "ollama" });
  });
});
