import { describe, it, expect, vi, afterEach } from "vitest";
import * as client from "../../src/api/client";

describe("api/client", () => {
  const originalFetch = global.fetch;

  afterEach(() => {
    global.fetch = originalFetch;
    vi.restoreAllMocks();
  });

  it("getPlugins unwraps the {plugins} envelope on success", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ plugins: [{ plugin_id: "p1", name: "Coding" }] }),
    }) as unknown as typeof fetch;

    const plugins = await client.getPlugins();
    expect(plugins).toEqual([{ plugin_id: "p1", name: "Coding" }]);
  });

  it("throws generic message on network error", async () => {
    global.fetch = vi.fn().mockRejectedValue(new TypeError("network down"));

    await expect(client.getPlugins()).rejects.toThrow(client.GENERIC_CONNECTION_ERROR);
  });

  it("throws backend error message on non-ok response", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: false,
      json: async () => ({ error: { message: "Plugin không tồn tại" } }),
    }) as unknown as typeof fetch;

    await expect(client.getPlugins()).rejects.toThrow("Plugin không tồn tại");
  });

  it("throws generic message when error body is unparsable", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: false,
      json: async () => {
        throw new Error("bad json");
      },
    }) as unknown as typeof fetch;

    await expect(client.getPlugins()).rejects.toThrow(client.GENERIC_CONNECTION_ERROR);
  });

  it("startRenderSaga posts input and returns saga response", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ saga_id: "s1", status: "started" }),
    }) as unknown as typeof fetch;

    const result = await client.startRenderSaga({
      project_id: "p1",
      script_content: "script",
      plugin_id: "coding",
      voice_language: "vi",
    });

    expect(result).toEqual({ saga_id: "s1", status: "started" });
    expect(global.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/v1/sagas/render"),
      expect.objectContaining({ method: "POST" }),
    );
  });

  it("getYoutubeAuthStartUrl returns a URL string without fetching", () => {
    const url = client.getYoutubeAuthStartUrl();
    expect(url).toContain("/v1/auth/youtube/start");
  });
});
