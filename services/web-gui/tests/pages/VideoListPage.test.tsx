import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { VideoListPage } from "../../src/pages/VideoListPage";

describe("VideoListPage", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("lists both successful and failed videos, and deletes one on confirm", async () => {
    global.fetch = vi.fn().mockImplementation((_url: string, init?: RequestInit) => {
      if (init?.method === "DELETE") {
        return Promise.resolve({ ok: true, status: 204 });
      }
      return Promise.resolve({
        ok: true,
        status: 200,
        json: async () => ({
          projects: [
            { project_id: "p1", status: "published", video_path: "/shared/p1/video/final.mp4", updated_at: "2026-01-01T00:00:00Z" },
            { project_id: "p2", status: "failed_at_render_scenes", error_message: "boom", updated_at: "2026-01-02T00:00:00Z" },
          ],
        }),
      });
    }) as unknown as typeof fetch;
    vi.spyOn(window, "confirm").mockReturnValue(true);

    render(
      <MemoryRouter>
        <VideoListPage />
      </MemoryRouter>,
    );

    await waitFor(() => expect(screen.getByTestId("video-row-p1")).toBeInTheDocument());
    expect(screen.getByTestId("video-row-p2")).toBeInTheDocument();
    expect(screen.getByText(/Thất bại/)).toBeInTheDocument();

    fireEvent.click(screen.getByTestId("delete-button-p1"));

    await waitFor(() => expect(screen.queryByTestId("video-row-p1")).not.toBeInTheDocument());
    expect(screen.getByTestId("video-row-p2")).toBeInTheDocument();
    expect(global.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/v1/projects/p1"),
      expect.objectContaining({ method: "DELETE" }),
    );
  });

  it("does not delete when the user cancels the confirmation", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        projects: [{ project_id: "p1", status: "published", updated_at: "2026-01-01T00:00:00Z" }],
      }),
    }) as unknown as typeof fetch;
    vi.spyOn(window, "confirm").mockReturnValue(false);

    render(
      <MemoryRouter>
        <VideoListPage />
      </MemoryRouter>,
    );

    await waitFor(() => expect(screen.getByTestId("video-row-p1")).toBeInTheDocument());
    fireEvent.click(screen.getByTestId("delete-button-p1"));

    expect(screen.getByTestId("video-row-p1")).toBeInTheDocument();
    expect(global.fetch).toHaveBeenCalledTimes(1);
  });
});
