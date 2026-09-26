import { describe, it, expect, vi, afterEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { ProjectDraftProvider } from "../../src/context/ProjectDraftContext";
import { useAuthoringMode } from "../../src/hooks/useAuthoringMode";
import * as apiClient from "../../src/api/client";

const wrapper = ({ children }: { children: ReactNode }) => <ProjectDraftProvider>{children}</ProjectDraftProvider>;

describe("useAuthoringMode.setMode", () => {
  afterEach(() => vi.restoreAllMocks());

  // project_authoring has an FK to projects and the mode picker is reachable
  // before any draft row exists: the row must be ensured BEFORE the mode PUT,
  // or the save fails and used to vanish silently.
  it("makes sure the project row exists before saving the mode", async () => {
    const order: string[] = [];
    vi.spyOn(apiClient, "getAuthoringState").mockResolvedValue({ topic: "", story: "", storyboard: "", code: "" });
    vi.spyOn(apiClient, "createProjectDraft").mockImplementation(async () => {
      order.push("draft");
      return { similarProjects: [] };
    });
    const save = vi.spyOn(apiClient, "saveAuthoringMode").mockImplementation(async () => {
      order.push("mode");
    });

    const { result } = renderHook(() => useAuthoringMode("p1"), { wrapper });
    act(() => result.current.setMode("ai"));

    await waitFor(() => expect(save).toHaveBeenCalledWith("p1", "ai"));
    expect(order).toEqual(["draft", "mode"]);
    expect(result.current.mode).toBe("ai");
  });

  it("still tries to save the mode when the draft upsert fails, and logs a failed save instead of swallowing it", async () => {
    vi.spyOn(apiClient, "getAuthoringState").mockResolvedValue({ topic: "", story: "", storyboard: "", code: "" });
    vi.spyOn(apiClient, "createProjectDraft").mockRejectedValue(new Error("raced"));
    vi.spyOn(apiClient, "saveAuthoringMode").mockRejectedValue(new Error("fk violation"));
    const logged = vi.spyOn(console, "error").mockImplementation(() => {});

    const { result } = renderHook(() => useAuthoringMode("p1"), { wrapper });
    act(() => result.current.setMode("manual"));

    await waitFor(() => expect(logged).toHaveBeenCalled());
    expect(apiClient.saveAuthoringMode).toHaveBeenCalledWith("p1", "manual");
  });
});
