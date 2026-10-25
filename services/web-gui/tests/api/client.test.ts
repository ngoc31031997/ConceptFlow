import { describe, it, expect, vi, afterEach } from "vitest";
import * as client from "../../src/api/client";

describe("api/client", () => {
  const originalFetch = global.fetch;

  afterEach(() => {
    global.fetch = originalFetch;
    vi.restoreAllMocks();
  });

  describe("review units", () => {
    const unitWire = {
      project_id: "p1",
      kind: "research_question",
      key: "Q1",
      status: "pending",
      content: { text: "Câu hỏi?" },
      creator_fields: null,
      redo_count: 2,
      updated_at: "2026-10-18T00:00:00Z",
    };
    const ref: client.UnitRef<unknown> = { kind: "research_question", key: "Q1", parse: (c) => c };
    const ok = (body: unknown) =>
      vi.fn().mockResolvedValue({ ok: true, status: 200, json: async () => body }) as unknown as typeof fetch;

    it("sends the operation id with an AI redo, so the screen can follow it", async () => {
      global.fetch = ok(unitWire);

      await client.redoUnit("p1", ref, "ngắn hơn", { id: "op-2", signal: new AbortController().signal });

      const calls = (global.fetch as unknown as { mock: { calls: [string, RequestInit][] } }).mock.calls;
      expect(calls[0][0]).toMatch(/\/v1\/projects\/p1\/units\/research_question\/Q1\/redo$/);
      expect((calls[0][1].headers as Record<string, string>)["X-Operation-Id"]).toBe("op-2");
      expect(JSON.parse(calls[0][1].body as string)).toEqual({ note: "ngắn hơn" });
    });

    it("sends a hand edit as the changed fields and reads the impact list", async () => {
      global.fetch = ok(unitWire);
      await client.editUnit("p1", ref, { text: "Sửa rồi?" });
      const edit = (global.fetch as unknown as { mock: { calls: [string, RequestInit][] } }).mock.calls[0];
      expect(edit[1].method).toBe("PUT");
      expect(JSON.parse(edit[1].body as string)).toEqual({ fields: { text: "Sửa rồi?" } });

      global.fetch = ok({ items: ["Dàn ý"] });
      expect(await client.getUnitImpact("p1", ref)).toEqual(["Dàn ý"]);
      global.fetch = ok({ items: null });
      expect(await client.getUnitImpact("p1", ref)).toEqual([]);
    });
  });

  it("throws generic message on network error", async () => {
    global.fetch = vi.fn().mockRejectedValue(new TypeError("network down"));

    await expect(client.listProjects()).rejects.toThrow(client.GENERIC_CONNECTION_ERROR);
  });

  it("throws backend error message on non-ok response", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: false,
      json: async () => ({ error: { message: "Project không tồn tại" } }),
    }) as unknown as typeof fetch;

    await expect(client.listProjects()).rejects.toThrow("Project không tồn tại");
  });

  it("throws generic message when error body is unparsable", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: false,
      json: async () => {
        throw new Error("bad json");
      },
    }) as unknown as typeof fetch;

    await expect(client.listProjects()).rejects.toThrow(client.GENERIC_CONNECTION_ERROR);
  });

  it("startRenderSaga posts input and returns saga response", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ saga_id: "s1", status: "started" }),
    }) as unknown as typeof fetch;

    const result = await client.startRenderSaga({
      project_id: "p1",
      script_content: "script",
      voice_language: "vi",
      tts_enabled: true,
      subtitle_mode: "off",
    });

    expect(result).toEqual({ saga_id: "s1", status: "started" });
    expect(global.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/v1/sagas/render"),
      expect.objectContaining({ method: "POST" }),
    );
  });

  it("listProjects unwraps the {projects} envelope on success", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ projects: [{ project_id: "p1", status: "published", updated_at: "2026-01-01T00:00:00Z" }] }),
    }) as unknown as typeof fetch;

    const projects = await client.listProjects();
    expect(projects).toEqual([{ project_id: "p1", status: "published", updated_at: "2026-01-01T00:00:00Z" }]);
  });

  it("listProjectsPage asks for one filtered page and returns it as is", async () => {
    const page = {
      projects: [{ project_id: "p1", status: "published", updated_at: "2026-01-01T00:00:00Z" }],
      total: 21, page: 2, page_size: 20,
      counts: { all: 30, running: 1, waiting: 5, problem: 3, done: 21 },
    };
    global.fetch = vi.fn().mockResolvedValue({ ok: true, json: async () => page }) as unknown as typeof fetch;

    await expect(client.listProjectsPage({ page: 2, pageSize: 20, filter: "done", steps: [10, 13] })).resolves.toEqual(page);
    expect(global.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/v1/projects?page=2&page_size=20&filter=done&steps=10%2C13"),
      undefined,
    );
  });

  it("listProjectsPage leaves steps out when no step is picked", async () => {
    global.fetch = vi.fn().mockResolvedValue({ ok: true, json: async () => ({}) }) as unknown as typeof fetch;

    await client.listProjectsPage({ page: 1, pageSize: 50, filter: "all", steps: [] });
    const url = (global.fetch as unknown as { mock: { calls: [string][] } }).mock.calls[0][0];
    expect(url).toMatch(/\/v1\/projects\?page=1&page_size=50&filter=all$/);
  });

  it("deleteProject sends a DELETE request and resolves on 204", async () => {
    global.fetch = vi.fn().mockResolvedValue({ ok: true, status: 204 }) as unknown as typeof fetch;

    await expect(client.deleteProject("p1")).resolves.toBeUndefined();
    expect(global.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/v1/projects/p1"),
      expect.objectContaining({ method: "DELETE" }),
    );
  });

  it("getYoutubeAuthStartUrl returns a URL string without fetching", () => {
    const url = client.getYoutubeAuthStartUrl("project-1");
    expect(url).toContain("/v1/auth/youtube/start");
    expect(url).toContain("state=project-1");
  });
});
