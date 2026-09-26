import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { VideoListPage } from "../../src/pages/VideoListPage";
import { ThemeProvider } from "../../src/context/ThemeContext";

describe("VideoListPage", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("lists both successful and failed videos, and deletes one on confirm", async () => {
    global.fetch = vi.fn().mockImplementation((url: string, init?: RequestInit) => {
      if (init?.method === "DELETE") {
        return Promise.resolve({ ok: true, status: 204 });
      }
      // FR116.2: the row goes once the delete saga reports it has finished.
      if (String(url).includes("/v1/operations/delete:p1")) {
        return Promise.resolve({
          ok: true,
          status: 200,
          json: async () => ({ kind: "delete_project", phase: "purge", done: 3, total: 3, status: "succeeded" }),
        });
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
      <ThemeProvider>
        <MemoryRouter>
          <VideoListPage />
        </MemoryRouter>
      </ThemeProvider>,
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

  it("keeps the row with a k/N progress card while the delete saga runs", async () => {
    global.fetch = vi.fn().mockImplementation((url: string, init?: RequestInit) => {
      if (init?.method === "DELETE") return Promise.resolve({ ok: true, status: 202 });
      if (String(url).includes("/v1/operations/delete:p1")) {
        return Promise.resolve({
          ok: true,
          status: 200,
          json: async () => ({ kind: "delete_project", phase: "purge", done: 1, total: 3, status: "running" }),
        });
      }
      return Promise.resolve({
        ok: true,
        status: 200,
        json: async () => ({ projects: [{ project_id: "p1", status: "published", updated_at: "2026-01-01T00:00:00Z" }] }),
      });
    }) as unknown as typeof fetch;
    vi.spyOn(window, "confirm").mockReturnValue(true);

    render(
      <ThemeProvider>
        <MemoryRouter>
          <VideoListPage />
        </MemoryRouter>
      </ThemeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("video-row-p1")).toBeInTheDocument());
    fireEvent.click(screen.getByTestId("delete-button-p1"));

    await waitFor(() => expect(screen.getByText("Đã xóa 1/3 mục")).toBeInTheDocument());
    expect(screen.getByTestId("video-row-p1")).toBeInTheDocument();
    expect(screen.getByRole("progressbar")).toHaveAttribute("aria-valuenow", "33");
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
      <ThemeProvider>
        <MemoryRouter>
          <VideoListPage />
        </MemoryRouter>
      </ThemeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("video-row-p1")).toBeInTheDocument());
    fireEvent.click(screen.getByTestId("delete-button-p1"));

    expect(screen.getByTestId("video-row-p1")).toBeInTheDocument();
    expect(global.fetch).toHaveBeenCalledTimes(1);
  });

  it("selects multiple videos via checkboxes and bulk-deletes them", async () => {
    const deleteCalls: string[] = [];
    global.fetch = vi.fn().mockImplementation((url: string, init?: RequestInit) => {
      if (init?.method === "DELETE") {
        deleteCalls.push(url);
        return Promise.resolve({ ok: true, status: 204 });
      }
      return Promise.resolve({
        ok: true,
        status: 200,
        json: async () => ({
          projects: [
            { project_id: "p1", status: "published", updated_at: "2026-01-01T00:00:00Z" },
            { project_id: "p2", status: "published", updated_at: "2026-01-02T00:00:00Z" },
            { project_id: "p3", status: "failed_at_render_scenes", updated_at: "2026-01-03T00:00:00Z" },
          ],
        }),
      });
    }) as unknown as typeof fetch;
    vi.spyOn(window, "confirm").mockReturnValue(true);

    render(
      <ThemeProvider>
        <MemoryRouter>
          <VideoListPage />
        </MemoryRouter>
      </ThemeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("video-row-p1")).toBeInTheDocument());

    const bulkDeleteButton = screen.getByTestId("bulk-delete-button");
    expect(bulkDeleteButton).toBeDisabled();

    fireEvent.click(screen.getByLabelText("Chọn video p1"));
    fireEvent.click(screen.getByLabelText("Chọn video p2"));

    expect(bulkDeleteButton).not.toBeDisabled();
    fireEvent.click(bulkDeleteButton);

    await waitFor(() => expect(screen.queryByTestId("video-row-p1")).not.toBeInTheDocument());
    expect(screen.queryByTestId("video-row-p2")).not.toBeInTheDocument();
    expect(screen.getByTestId("video-row-p3")).toBeInTheDocument();
    expect(deleteCalls).toEqual([
      expect.stringContaining("/v1/projects/p1"),
      expect.stringContaining("/v1/projects/p2"),
    ]);
  });

  it("marks which engine rendered each video", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        projects: [
          { project_id: "p1", status: "published", updated_at: "2026-01-01T00:00:00Z", render_engine: "manim" },
          { project_id: "p2", status: "published", updated_at: "2026-01-02T00:00:00Z", render_engine: "remotion" },
        ],
      }),
    }) as unknown as typeof fetch;

    render(
      <ThemeProvider>
        <MemoryRouter>
          <VideoListPage />
        </MemoryRouter>
      </ThemeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("video-row-p1")).toBeInTheDocument());
    const badges = screen.getAllByTestId("render-engine-badge");
    expect(badges[0]).toHaveTextContent("Manim");
    expect(badges[1]).toHaveTextContent("Remotion");
  });

  it("selects all videos via the header checkbox", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        projects: [
          { project_id: "p1", status: "published", updated_at: "2026-01-01T00:00:00Z" },
          { project_id: "p2", status: "published", updated_at: "2026-01-02T00:00:00Z" },
        ],
      }),
    }) as unknown as typeof fetch;

    render(
      <ThemeProvider>
        <MemoryRouter>
          <VideoListPage />
        </MemoryRouter>
      </ThemeProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("video-row-p1")).toBeInTheDocument());
    fireEvent.click(screen.getByLabelText("Chọn tất cả"));

    expect(screen.getByText("Đã chọn 2 video")).toBeInTheDocument();
    expect(screen.getByTestId("bulk-delete-button")).not.toBeDisabled();
  });

  describe("theo flow 13 bước", () => {
    function renderList(projects: unknown[]) {
      global.fetch = vi.fn().mockResolvedValue({ ok: true, status: 200, json: async () => ({ projects }) }) as unknown as typeof fetch;
      return render(
        <ThemeProvider>
          <MemoryRouter>
            <VideoListPage />
          </MemoryRouter>
        </ThemeProvider>,
      );
    }

    const base = { updated_at: "2026-01-01T00:00:00Z", render_engine: "manim" };
    const rows = [
      { ...base, project_id: "aaaaaaaa-1111", status: "rendering", topic: "thiên kiến sống sót", flow_step: 9, run_state: "running" },
      { ...base, project_id: "bbbbbbbb-2222", status: "failed_at_render_scenes", topic: "Vòng lặp for", flow_step: 9, run_state: "cancelled" },
      { ...base, project_id: "cccccccc-3333", status: "ready_to_publish", topic: "Cây nhị phân", flow_step: 12, run_state: "idle", forked_from: "aaaaaaaa-1111" },
      { ...base, project_id: "dddddddd-4444", status: "draft", flow_step: 2, run_state: "idle" },
    ];

    it("names a project by its topic, shows where it is in the 13 steps, and links a fork to its source", async () => {
      renderList(rows);
      await waitFor(() => expect(screen.getByTestId("video-row-aaaaaaaa-1111")).toBeInTheDocument());

      expect(screen.getByTestId("video-row-aaaaaaaa-1111")).toHaveTextContent("thiên kiến sống sót");
      expect(screen.getByTestId("video-row-aaaaaaaa-1111")).toHaveTextContent("Bước 9 — Render");
      expect(screen.getAllByTestId("flow-mini")).toHaveLength(4);
      expect(screen.getByTestId("video-row-cccccccc-3333")).toHaveTextContent("Bản mới từ “thiên kiến sống sót”");
      // No topic yet: say so instead of showing only a UUID.
      expect(screen.getByTestId("video-row-dddddddd-4444")).toHaveTextContent("chưa đặt chủ đề");
    });

    it("filters by what needs attention", async () => {
      renderList(rows);
      await waitFor(() => expect(screen.getByTestId("filter-problem")).toBeInTheDocument());

      fireEvent.click(screen.getByTestId("filter-problem"));
      expect(screen.getByTestId("video-row-bbbbbbbb-2222")).toBeInTheDocument();
      expect(screen.queryByTestId("video-row-aaaaaaaa-1111")).not.toBeInTheDocument();

      fireEvent.click(screen.getByTestId("filter-running"));
      expect(screen.getByTestId("video-row-aaaaaaaa-1111")).toBeInTheDocument();
      expect(screen.queryByTestId("video-row-bbbbbbbb-2222")).not.toBeInTheDocument();

      fireEvent.click(screen.getByTestId("filter-done"));
      expect(screen.getByTestId("video-row-cccccccc-3333")).toBeInTheDocument();
      expect(screen.queryByTestId("video-row-aaaaaaaa-1111")).not.toBeInTheDocument();
    });
  });
});
