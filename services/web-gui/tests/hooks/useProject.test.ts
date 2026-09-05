import { describe, it, expect, vi, afterEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { useProject } from "../../src/hooks/useProject";

describe("useProject", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("fetches project on mount", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ project_id: "p1", status: "ready_to_publish", scenes: [] }),
    }) as unknown as typeof fetch;

    const { result } = renderHook(() => useProject("p1"));

    await waitFor(() => expect(result.current.project?.project_id).toBe("p1"));
  });

  it("sets error message on failure", async () => {
    global.fetch = vi.fn().mockRejectedValue(new TypeError("down"));

    const { result } = renderHook(() => useProject("p1"));

    await waitFor(() => expect(result.current.error).toBeTruthy());
  });
});
