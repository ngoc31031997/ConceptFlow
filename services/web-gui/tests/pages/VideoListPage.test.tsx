import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { VideoListPage } from "../../src/pages/VideoListPage";
import { ThemeProvider } from "../../src/context/ThemeContext";

type Row = { project_id: string; flow_step?: number; run_state?: string; [key: string]: unknown };
type Route = (url: string, init?: RequestInit) => Promise<unknown> | undefined;

const ok = (body: unknown) => Promise.resolve({ ok: true, status: 200, json: async () => body });

// The orchestrator's filter rules (domain.ListFilter.Matches).
function matches(p: Row, filter: string): boolean {
  const step = p.flow_step ?? 0;
  switch (filter) {
    case "running":
      return p.run_state === "running";
    case "problem":
      return p.run_state === "failed" || p.run_state === "cancelled";
    case "done":
      return step >= 13 && p.run_state !== "failed";
    case "waiting":
      return p.run_state === "idle" && step > 0 && step < 13;
    default:
      return true;
  }
}

/** A fake GET /v1/projects?page=… that filters, counts and pages `store`. */
function pageOf(store: Row[], url: string) {
  const q = new URL(url, "http://x").searchParams;
  const size = Number(q.get("page_size"));
  const filter = q.get("filter") ?? "all";
  const steps = (q.get("steps") ?? "").split(",").filter(Boolean).map(Number);
  const filtered = store.filter((p) => matches(p, filter) && (steps.length === 0 || steps.includes(p.flow_step ?? 0)));
  const pages = Math.max(1, Math.ceil(filtered.length / size));
  const page = Math.min(Math.max(Number(q.get("page")), 1), pages);
  const counts = Object.fromEntries(
    ["all", "running", "waiting", "problem", "done"].map((f) => [f, store.filter((p) => matches(p, f)).length]),
  );
  return { projects: filtered.slice((page - 1) * size, page * size), total: filtered.length, page, page_size: size, counts };
}

/**
 * Serves the list from `store`; a DELETE takes the project off it (the server
 * hides `deleting` projects). `route` answers anything else first.
 */
function serve(store: Row[], route?: Route) {
  const fetchMock = vi.fn().mockImplementation((url: string, init?: RequestInit) => {
    const routed = route?.(String(url), init);
    if (routed) return routed;
    if (init?.method === "DELETE") {
      const id = String(url).split("/").pop();
      store.splice(store.findIndex((p) => p.project_id === id), 1);
      return Promise.resolve({ ok: true, status: 204 });
    }
    return ok(pageOf(store, String(url)));
  });
  global.fetch = fetchMock as unknown as typeof fetch;
  return fetchMock;
}

function listUrls(fetchMock: ReturnType<typeof vi.fn>): string[] {
  return fetchMock.mock.calls.map(([url]) => String(url)).filter((u) => u.includes("/v1/projects?"));
}

function renderPage() {
  return render(
    <ThemeProvider>
      <MemoryRouter>
        <VideoListPage />
      </MemoryRouter>
    </ThemeProvider>,
  );
}

function many(n: number): Row[] {
  return Array.from({ length: n }, (_, i) => ({
    project_id: `p${String(i).padStart(2, "0")}`,
    status: "awaiting_review",
    topic: `Chủ đề ${i}`,
    updated_at: "2026-01-01T00:00:00Z",
    render_engine: "manim",
    flow_step: 8,
    run_state: i % 5 === 0 ? "failed" : "idle",
  }));
}

describe("VideoListPage", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("lists both successful and failed videos, and deletes one on confirm", async () => {
    const fetchMock = serve(
      [
        { project_id: "p1", status: "published", video_path: "/shared/p1/video/final.mp4", updated_at: "2026-01-01T00:00:00Z" },
        { project_id: "p2", status: "failed_at_render_scenes", error_message: "boom", updated_at: "2026-01-02T00:00:00Z" },
      ],
      // The row goes once the delete saga reports it has finished.
      (url) =>
        url.includes("/v1/operations/delete:p1")
          ? ok({ kind: "delete_project", phase: "purge", done: 3, total: 3, status: "succeeded" })
          : undefined,
    );
    renderPage();

    await waitFor(() => expect(screen.getByTestId("video-row-p1")).toBeInTheDocument());
    expect(screen.getByTestId("video-row-p2")).toBeInTheDocument();
    expect(screen.getByText(/Thất bại/)).toBeInTheDocument();

    fireEvent.click(screen.getByTestId("delete-button-p1"));
    fireEvent.click(screen.getByTestId("confirm-modal-confirm"));

    await waitFor(() => expect(screen.queryByTestId("video-row-p1")).not.toBeInTheDocument());
    expect(screen.getByTestId("video-row-p2")).toBeInTheDocument();
    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining("/v1/projects/p1"),
      expect.objectContaining({ method: "DELETE" }),
    );
  });

  it("keeps the row with a k/N progress card while the delete saga runs", async () => {
    // The row is still listed: the saga has not finished, nothing refetches.
    serve([{ project_id: "p1", status: "published", updated_at: "2026-01-01T00:00:00Z" }], (url, init) => {
      if (init?.method === "DELETE") return Promise.resolve({ ok: true, status: 202 });
      if (url.includes("/v1/operations/delete:p1")) {
        return ok({ kind: "delete_project", phase: "purge", done: 1, total: 3, status: "running" });
      }
      return undefined;
    });
    renderPage();

    await waitFor(() => expect(screen.getByTestId("video-row-p1")).toBeInTheDocument());
    fireEvent.click(screen.getByTestId("delete-button-p1"));
    fireEvent.click(screen.getByTestId("confirm-modal-confirm"));

    await waitFor(() => expect(screen.getByText("Đã xóa 1/3 mục")).toBeInTheDocument());
    expect(screen.getByTestId("video-row-p1")).toBeInTheDocument();
    expect(screen.getByRole("progressbar")).toHaveAttribute("aria-valuenow", "33");
  });

  it("does not delete when the user cancels the confirmation", async () => {
    const fetchMock = serve([{ project_id: "p1", status: "published", updated_at: "2026-01-01T00:00:00Z" }]);
    renderPage();

    await waitFor(() => expect(screen.getByTestId("video-row-p1")).toBeInTheDocument());
    fireEvent.click(screen.getByTestId("delete-button-p1"));
    fireEvent.click(screen.getByTestId("confirm-modal-cancel"));

    expect(screen.queryByTestId("confirm-modal")).not.toBeInTheDocument();
    expect(screen.getByTestId("video-row-p1")).toBeInTheDocument();
    const calls = fetchMock.mock.calls as [string, RequestInit?][];
    expect(calls.some(([, init]) => init?.method === "DELETE")).toBe(false);
  });

  it("selects multiple videos via checkboxes and bulk-deletes them", async () => {
    const fetchMock = serve([
      { project_id: "p1", status: "published", updated_at: "2026-01-01T00:00:00Z" },
      { project_id: "p2", status: "published", updated_at: "2026-01-02T00:00:00Z" },
      { project_id: "p3", status: "failed_at_render_scenes", updated_at: "2026-01-03T00:00:00Z" },
    ]);
    renderPage();

    await waitFor(() => expect(screen.getByTestId("video-row-p1")).toBeInTheDocument());

    const bulkDeleteButton = screen.getByTestId("bulk-delete-button");
    expect(bulkDeleteButton).toBeDisabled();

    fireEvent.click(screen.getByLabelText("Chọn video p1"));
    fireEvent.click(screen.getByLabelText("Chọn video p2"));

    expect(bulkDeleteButton).not.toBeDisabled();
    fireEvent.click(bulkDeleteButton);
    fireEvent.click(screen.getByTestId("confirm-modal-confirm"));

    await waitFor(() => expect(screen.queryByTestId("video-row-p1")).not.toBeInTheDocument());
    expect(screen.queryByTestId("video-row-p2")).not.toBeInTheDocument();
    expect(screen.getByTestId("video-row-p3")).toBeInTheDocument();
    const deleteCalls = (fetchMock.mock.calls as [string, RequestInit?][])
      .filter(([, init]) => init?.method === "DELETE")
      .map(([url]) => url);
    expect(deleteCalls).toEqual([
      expect.stringContaining("/v1/projects/p1"),
      expect.stringContaining("/v1/projects/p2"),
    ]);
  });

  it("marks which engine rendered each video", async () => {
    serve([
      { project_id: "p1", status: "published", updated_at: "2026-01-01T00:00:00Z", render_engine: "manim" },
      { project_id: "p2", status: "published", updated_at: "2026-01-02T00:00:00Z", render_engine: "remotion" },
    ]);
    renderPage();

    await waitFor(() => expect(screen.getByTestId("video-row-p1")).toBeInTheDocument());
    const badges = screen.getAllByTestId("render-engine-badge");
    expect(badges[0]).toHaveTextContent("Manim");
    expect(badges[1]).toHaveTextContent("Remotion");
  });

  it("selects all videos via the header checkbox", async () => {
    serve([
      { project_id: "p1", status: "published", updated_at: "2026-01-01T00:00:00Z" },
      { project_id: "p2", status: "published", updated_at: "2026-01-02T00:00:00Z" },
    ]);
    renderPage();

    await waitFor(() => expect(screen.getByTestId("video-row-p1")).toBeInTheDocument());
    fireEvent.click(screen.getByLabelText("Chọn tất cả"));

    expect(screen.getByText("Đã chọn 2 video")).toBeInTheDocument();
    expect(screen.getByTestId("bulk-delete-button")).not.toBeDisabled();
  });

  describe("theo flow 14 bước", () => {
    const base = { updated_at: "2026-01-01T00:00:00Z", render_engine: "manim" };
    const rows = (): Row[] => [
      { ...base, project_id: "aaaaaaaa-1111", status: "rendering", topic: "thiên kiến sống sót", flow_step: 10, run_state: "running" },
      { ...base, project_id: "bbbbbbbb-2222", status: "failed_at_render_scenes", topic: "Vòng lặp for", flow_step: 10, run_state: "cancelled" },
      {
        ...base, project_id: "cccccccc-3333", status: "ready_to_publish", topic: "Cây nhị phân", flow_step: 13, run_state: "idle",
        forked_from: "aaaaaaaa-1111", forked_from_topic: "thiên kiến sống sót",
      },
      { ...base, project_id: "dddddddd-4444", status: "draft", flow_step: 2, run_state: "idle" },
    ];

    it("names a project by its topic, shows where it is in the 14 steps, and links a fork to its source", async () => {
      serve(rows());
      renderPage();
      await waitFor(() => expect(screen.getByTestId("video-row-aaaaaaaa-1111")).toBeInTheDocument());

      expect(screen.getByTestId("video-row-aaaaaaaa-1111")).toHaveTextContent("thiên kiến sống sót");
      expect(screen.getByTestId("video-row-aaaaaaaa-1111")).toHaveTextContent("Bước 10 — Dựng hình");
      expect(screen.getAllByTestId("flow-mini")).toHaveLength(4);
      expect(screen.getByTestId("video-row-cccccccc-3333")).toHaveTextContent("Bản mới từ “thiên kiến sống sót”");
      // No topic yet: say so instead of showing only a UUID.
      expect(screen.getByTestId("video-row-dddddddd-4444")).toHaveTextContent("chưa đặt chủ đề");
      // One page only: no pager.
      expect(screen.queryByTestId("pagination")).not.toBeInTheDocument();
    });

    it("names a fork's source from the server even when the source is on another page", async () => {
      serve([{ ...base, project_id: "eeeeeeee-5555", status: "draft", flow_step: 3, run_state: "idle", forked_from: "ffffffff-6666", forked_from_topic: "Đệ quy" }]);
      renderPage();
      await waitFor(() => expect(screen.getByTestId("lineage")).toHaveTextContent("Bản mới từ “Đệ quy”"));
    });

    it("asks the server for each chip and shows what it sends back", async () => {
      const fetchMock = serve(rows());
      renderPage();
      await waitFor(() => expect(screen.getByTestId("filter-problem")).toBeInTheDocument());
      // Chip counts come from the server's counts, over the whole list.
      expect(screen.getByTestId("filter-problem")).toHaveTextContent("1");
      expect(screen.getByTestId("filter-all")).toHaveTextContent("4");

      fireEvent.click(screen.getByTestId("filter-problem"));
      await waitFor(() => expect(screen.queryByTestId("video-row-aaaaaaaa-1111")).not.toBeInTheDocument());
      expect(screen.getByTestId("video-row-bbbbbbbb-2222")).toBeInTheDocument();
      expect(listUrls(fetchMock).at(-1)).toContain("filter=problem");

      fireEvent.click(screen.getByTestId("filter-running"));
      await waitFor(() => expect(screen.getByTestId("video-row-aaaaaaaa-1111")).toBeInTheDocument());
      expect(screen.queryByTestId("video-row-bbbbbbbb-2222")).not.toBeInTheDocument();

      fireEvent.click(screen.getByTestId("filter-done"));
      await waitFor(() => expect(screen.getByTestId("video-row-cccccccc-3333")).toBeInTheDocument());
      expect(screen.queryByTestId("video-row-aaaaaaaa-1111")).not.toBeInTheDocument();

      fireEvent.click(screen.getByTestId("filter-all"));
      fireEvent.click(screen.getByTestId("step-filter-10"));
      await waitFor(() => expect(listUrls(fetchMock).at(-1)).toContain("steps=10"));
    });
  });

  describe("phân trang", () => {
    it("shows 20 videos a page and moves between pages", async () => {
      const fetchMock = serve(many(45));
      renderPage();
      await waitFor(() => expect(screen.getByTestId("video-row-p00")).toBeInTheDocument());

      expect(screen.getAllByTestId("flow-mini")).toHaveLength(20);
      expect(screen.getByTestId("pagination-summary")).toHaveTextContent("Hiển thị 1–20 / 45 video");
      expect(listUrls(fetchMock)[0]).toContain("page=1&page_size=20&filter=all");

      fireEvent.click(screen.getByTestId("pagination-next"));
      await waitFor(() => expect(screen.getByTestId("video-row-p20")).toBeInTheDocument());
      expect(screen.queryByTestId("video-row-p00")).not.toBeInTheDocument();

      fireEvent.click(screen.getByTestId("pagination-page-3"));
      await waitFor(() => expect(screen.getByTestId("pagination-summary")).toHaveTextContent("Hiển thị 41–45 / 45 video"));
      expect(screen.getAllByTestId("flow-mini")).toHaveLength(5);
    });

    it("goes back to page 1 when the Creator picks another chip", async () => {
      const fetchMock = serve(many(45));
      renderPage();
      await waitFor(() => expect(screen.getByTestId("pagination-page-3")).toBeInTheDocument());
      fireEvent.click(screen.getByTestId("pagination-page-3"));
      await waitFor(() => expect(screen.getByTestId("video-row-p40")).toBeInTheDocument());

      fireEvent.click(screen.getByTestId("filter-waiting"));
      await waitFor(() => expect(screen.getByTestId("video-row-p01")).toBeInTheDocument());
      expect(listUrls(fetchMock).at(-1)).toContain("page=1&page_size=20&filter=waiting");
    });

    it("lets the Creator show 50 a page, and hides the pager when one page is enough", async () => {
      const fetchMock = serve(many(45));
      renderPage();
      await waitFor(() => expect(screen.getByTestId("pagination-page-2")).toBeInTheDocument());
      fireEvent.click(screen.getByTestId("pagination-page-2"));
      await waitFor(() => expect(screen.getByTestId("video-row-p20")).toBeInTheDocument());

      fireEvent.change(screen.getByTestId("pagination-size"), { target: { value: "50" } });
      await waitFor(() => expect(screen.getAllByTestId("flow-mini")).toHaveLength(45));
      expect(listUrls(fetchMock).at(-1)).toContain("page=1&page_size=50");
      expect(screen.queryByTestId("pagination")).not.toBeInTheDocument();
    });

    it("selects all on the page on show only, keeping picks from other pages", async () => {
      serve(many(45));
      renderPage();
      await waitFor(() => expect(screen.getByTestId("video-row-p00")).toBeInTheDocument());
      fireEvent.click(screen.getByLabelText("Chọn video p00"));

      fireEvent.click(screen.getByTestId("pagination-next"));
      await waitFor(() => expect(screen.getByTestId("video-row-p20")).toBeInTheDocument());
      fireEvent.click(screen.getByLabelText("Chọn tất cả"));
      expect(screen.getByTestId("bulk-delete-button")).toHaveTextContent("Xóa đã chọn (21)");

      fireEvent.click(screen.getByTestId("pagination-prev"));
      await waitFor(() => expect(screen.getByTestId("video-row-p00")).toBeInTheDocument());
      expect(screen.getByLabelText("Chọn tất cả")).not.toBeChecked();
      expect(screen.getByLabelText("Chọn video p00")).toBeChecked();
    });

    it("ignores an older page that answers after a newer one", async () => {
      const store = many(45);
      let releasePage2: () => void = () => {};
      serve(store, (url) => {
        if (!url.includes("page=2&")) return undefined;
        return new Promise((resolve) => {
          releasePage2 = () => resolve({ ok: true, status: 200, json: async () => pageOf(store, url) });
        });
      });
      renderPage();
      await waitFor(() => expect(screen.getByTestId("pagination-next")).toBeInTheDocument());

      fireEvent.click(screen.getByTestId("pagination-next")); // page 2 hangs
      fireEvent.click(screen.getByTestId("pagination-page-3"));
      await waitFor(() => expect(screen.getByTestId("video-row-p40")).toBeInTheDocument());

      releasePage2();
      await new Promise((r) => setTimeout(r, 0));
      expect(screen.getByTestId("video-row-p40")).toBeInTheDocument();
      expect(screen.queryByTestId("video-row-p20")).not.toBeInTheDocument();
    });

    it("refills the page from the server after a bulk delete", async () => {
      const fetchMock = serve(many(25));
      renderPage();
      await waitFor(() => expect(screen.getByTestId("video-row-p00")).toBeInTheDocument());
      fireEvent.click(screen.getByLabelText("Chọn video p00"));
      fireEvent.click(screen.getByTestId("bulk-delete-button"));
      fireEvent.click(screen.getByTestId("confirm-modal-confirm"));

      // p20 moves up from page 2 into the gap.
      await waitFor(() => expect(screen.getByTestId("video-row-p20")).toBeInTheDocument());
      expect(screen.queryByTestId("video-row-p00")).not.toBeInTheDocument();
      expect(listUrls(fetchMock).length).toBeGreaterThanOrEqual(2);
    });
  });
});
