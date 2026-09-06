import { describe, it, expect, vi, afterEach } from "vitest";
import * as client from "../../src/api/client";

describe("api/client", () => {
  const originalFetch = global.fetch;

  afterEach(() => {
    global.fetch = originalFetch;
    vi.restoreAllMocks();
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
