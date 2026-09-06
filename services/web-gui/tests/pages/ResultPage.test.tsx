import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { ResultPage } from "../../src/pages/ResultPage";

function renderResultPage() {
  return render(
    <MemoryRouter initialEntries={["/projects/p1/result"]}>
      <Routes>
        <Route path="/projects/:id/result" element={<ResultPage />} />
        <Route path="/videos" element={<div data-testid="video-list-page-stub" />} />
      </Routes>
    </MemoryRouter>,
  );
}

function mockFetch(overrides: { onDelete?: () => { ok: boolean; status: number } } = {}) {
  return vi.fn().mockImplementation((url: string, init?: RequestInit) => {
    if (init?.method === "DELETE") {
      return Promise.resolve(overrides.onDelete ? overrides.onDelete() : { ok: true, status: 204 });
    }
    if (url.includes("/v1/auth/youtube/status")) {
      return Promise.resolve({ ok: true, status: 200, json: async () => ({ connected: false }) });
    }
    return Promise.resolve({
      ok: true,
      status: 200,
      json: async () => ({ project_id: "p1", status: "ready_to_publish", scenes: [], video_path: "/shared/p1/video/final.mp4" }),
    });
  }) as unknown as typeof fetch;
}

describe("ResultPage delete button", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("deletes the project and navigates to the video list on confirm", async () => {
    global.fetch = mockFetch();
    vi.spyOn(window, "confirm").mockReturnValue(true);

    renderResultPage();

    await waitFor(() => expect(screen.getByTestId("result-delete-button")).toBeInTheDocument());
    fireEvent.click(screen.getByTestId("result-delete-button"));

    await waitFor(() => expect(screen.getByTestId("video-list-page-stub")).toBeInTheDocument());
    expect(global.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/v1/projects/p1"),
      expect.objectContaining({ method: "DELETE" }),
    );
  });

  it("does not delete or navigate when the user cancels", async () => {
    global.fetch = mockFetch();
    vi.spyOn(window, "confirm").mockReturnValue(false);

    renderResultPage();

    await waitFor(() => expect(screen.getByTestId("result-delete-button")).toBeInTheDocument());
    fireEvent.click(screen.getByTestId("result-delete-button"));

    expect(screen.getByTestId("result-page")).toBeInTheDocument();
    expect(global.fetch).not.toHaveBeenCalledWith(
      expect.stringContaining("/v1/projects/p1"),
      expect.objectContaining({ method: "DELETE" }),
    );
  });
});
