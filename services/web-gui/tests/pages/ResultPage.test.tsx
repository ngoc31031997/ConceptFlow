import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { ResultPage } from "../../src/pages/ResultPage";

function renderResultPage() {
  return render(
    <MemoryRouter initialEntries={["/projects/p1/result"]}>
      <Routes>
        <Route path="/projects/:id/result" element={<ResultPage />} />
        <Route path="/projects/:id/render" element={<div data-testid="render-page-stub" />} />
        <Route path="/projects/:id/publish" element={<div data-testid="publish-page-stub" />} />
        <Route path="/videos" element={<div data-testid="video-list-page-stub" />} />
      </Routes>
    </MemoryRouter>,
  );
}

function mockFetch(overrides: {
  onDelete?: () => { ok: boolean; status: number };
  project?: Record<string, unknown>;
} = {}) {
  return vi.fn().mockImplementation((_url: string, init?: RequestInit) => {
    if (init?.method === "DELETE") {
      return Promise.resolve(overrides.onDelete ? overrides.onDelete() : { ok: true, status: 204 });
    }
    return Promise.resolve({
      ok: true,
      status: 200,
      json: async () => ({
        project_id: "p1",
        status: "ready_to_publish",
        scenes: [],
        video_path: "/shared/p1/video/final.mp4",
        ...overrides.project,
      }),
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

/*
  Bug report (2026-09-12): trước đây trang này vừa xem lại vừa đăng bài — giờ
  Bước 5 "Kết quả" chỉ dẫn sang Bước 6 "Đăng", không tự đăng gì ở đây.
*/
describe("ResultPage continue-to-publish handoff", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("chưa đăng thì hiện nút dẫn sang trang Đăng, không có form đăng nào ở đây", async () => {
    global.fetch = mockFetch({ project: { status: "ready_to_publish" } });

    renderResultPage();

    await waitFor(() => expect(screen.getByTestId("result-continue-to-publish")).toBeInTheDocument());
    expect(screen.queryByTestId("publish-form-submit-button")).not.toBeInTheDocument();

    fireEvent.click(screen.getByTestId("result-continue-to-publish-link"));
    await waitFor(() => expect(screen.getByTestId("publish-page-stub")).toBeInTheDocument());
  });

  it("đã đăng thì hiện banner gọn kèm link xem chi tiết, không hiện lại toàn bộ card thành công", async () => {
    global.fetch = mockFetch({
      project: { status: "published", youtube_video_url: "https://youtu.be/abc" },
    });

    renderResultPage();

    await waitFor(() => expect(screen.getByTestId("result-published-banner")).toBeInTheDocument());
    expect(screen.getByText("https://youtu.be/abc")).toBeInTheDocument();
    expect(screen.queryByTestId("result-continue-to-publish")).not.toBeInTheDocument();
  });
});

describe("render lại ở chất lượng khác", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  function mockRerenderFetch(bodies: Array<Record<string, unknown>>) {
    return vi.fn().mockImplementation((url: string, init?: RequestInit) => {
      if (init?.method === "POST" && url.includes("/v1/sagas/render")) {
        bodies.push(JSON.parse(init.body as string));
        return Promise.resolve({ ok: true, status: 201, json: async () => ({ saga_id: "s2", status: "draft" }) });
      }
      return Promise.resolve({
        ok: true,
        status: 200,
        json: async () => ({
          project_id: "p1",
          status: "ready_to_publish",
          scenes: [],
          video_path: "/shared/p1/video/final.mp4",
          script_content: "from conceptflow import *\n...",
          voice_language: "vi",
          tts_enabled: true,
          subtitle_mode: "track",
        }),
      });
    }) as unknown as typeof fetch;
  }

  it("hiện nút mở, ẩn form chọn chất lượng cho tới khi bấm", async () => {
    global.fetch = mockRerenderFetch([]);
    renderResultPage();

    await screen.findByTestId("rerender-toggle");
    expect(screen.queryByTestId("render-quality-picker")).not.toBeInTheDocument();

    fireEvent.click(screen.getByTestId("rerender-toggle"));

    expect(screen.getByTestId("render-quality-picker")).toBeInTheDocument();
  });

  it("gửi lại script gốc kèm chất lượng mới, rồi điều hướng sang trang render", async () => {
    const bodies: Array<Record<string, unknown>> = [];
    global.fetch = mockRerenderFetch(bodies);
    renderResultPage();

    fireEvent.click(await screen.findByTestId("rerender-toggle"));
    fireEvent.click(screen.getByTestId("render-quality-4k60"));
    fireEvent.click(screen.getByTestId("rerender-submit"));

    await waitFor(() => expect(bodies.length).toBe(1));
    expect(bodies[0]).toMatchObject({
      project_id: "p1",
      script_content: "from conceptflow import *\n...",
      render_quality: "4k60",
    });
  });

  it("báo lỗi rõ ràng thay vì gọi API khi project thiếu script_content", async () => {
    global.fetch = mockFetch();

    renderResultPage();
    fireEvent.click(await screen.findByTestId("rerender-toggle"));
    fireEvent.click(screen.getByTestId("rerender-submit"));

    await waitFor(() =>
      expect(screen.getByText(/Thiếu script gốc/)).toBeInTheDocument(),
    );
    expect(global.fetch).not.toHaveBeenCalledWith(
      expect.stringContaining("/v1/sagas/render"),
      expect.anything(),
    );
  });
});
