import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { RenderPage } from "../../src/pages/RenderPage";
import { ThemeProvider } from "../../src/context/ThemeContext";

class FakeEventSource {
  static instances: FakeEventSource[] = [];
  onmessage: ((event: MessageEvent) => void) | null = null;
  closed = false;

  constructor(public url: string) {
    FakeEventSource.instances.push(this);
  }

  emit(data: unknown) {
    this.onmessage?.({ data: JSON.stringify(data) } as MessageEvent);
  }

  close() {
    this.closed = true;
  }
}

function stubProject(project: Record<string, unknown>) {
  global.fetch = vi.fn().mockResolvedValue({
    ok: true,
    json: async () => project,
  }) as unknown as typeof fetch;
}

function renderRenderPage() {
  return render(
    <ThemeProvider>
      <MemoryRouter initialEntries={["/projects/p1/render"]}>
        <Routes>
          <Route path="/projects/:id/render" element={<RenderPage />} />
          <Route path="/projects/:id/validate" element={<div data-testid="validate-page-stub" />} />
          <Route path="/projects/:id/result" element={<div data-testid="result-page-stub" />} />
          <Route path="/" element={<div data-testid="new-project-page-stub" />} />
        </Routes>
      </MemoryRouter>
    </ThemeProvider>,
  );
}

describe("RenderPage (bước 5 — sản xuất)", () => {
  beforeEach(() => {
    FakeEventSource.instances = [];
    // @ts-expect-error test stub
    global.EventSource = FakeEventSource;
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("mỗi bước 9-12 là một màn riêng: đang render thì chỉ hiện bước Render", async () => {
    stubProject({ project_id: "p1", status: "rendering", scenes: [] });

    renderRenderPage();

    await waitFor(() => expect(screen.getByTestId("progress-tracker-steps").querySelectorAll("li")).toHaveLength(1));
    const steps = screen.getByTestId("progress-tracker-steps");
    expect(steps).toHaveTextContent("10Render");
    expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("Bước 10 — Render");
    expect(steps).not.toHaveTextContent("Tạo giọng đọc");
    expect(steps).not.toHaveTextContent("Ghép video hoàn chỉnh");
    expect(steps).not.toHaveTextContent("Phân tích kịch bản");
  });

  it("cho thử lại khi một bước sản xuất hỏng, và không rủ quay về sửa script", async () => {
    // Script này đã qua lượt chạy thử và đã được duyệt ở bước 4, nên một lần
    // hỏng ở đây gần như luôn là hạ tầng — retry mới là việc đúng.
    stubProject({
      project_id: "p1",
      status: "failed_at_render_scenes",
      scenes: [],
      error_message: "Render timeout",
    });

    renderRenderPage();

    await waitFor(() => expect(screen.getByTestId("error-banner-retry-button")).toBeInTheDocument());
    expect(screen.queryByTestId("error-banner-back-button")).not.toBeInTheDocument();
  });

  it("đẩy về bước 4 khi project vẫn đang ở giai đoạn chạy thử/chờ duyệt", async () => {
    // Bookmark cũ, hay nút back sau khi duyệt: nếu không đẩy đi thì Creator
    // nhìn bốn ô "pending" bất động và tưởng saga đã chết.
    stubProject({ project_id: "p1", status: "awaiting_review", scenes: [], beats: [] });

    renderRenderPage();

    await waitFor(() => expect(screen.getByTestId("validate-page-stub")).toBeInTheDocument());
  });

  it("đi tiếp sang màn kết quả khi đã sản xuất xong", async () => {
    stubProject({ project_id: "p1", status: "ready_to_publish", scenes: [] });

    renderRenderPage();

    await waitFor(() => expect(screen.getByTestId("result-page-stub")).toBeInTheDocument());
  });

  it("hiện cấu hình dựng & ghép ngay trên nút thử lại khi Render lỗi", async () => {
    stubProject({ project_id: "p1", status: "failed_at_render_scenes", scenes: [], error_message: "Render timeout" });

    renderRenderPage();

    await waitFor(() => expect(screen.getByTestId("production-settings")).toBeInTheDocument());
    expect(screen.getByTestId("production-render-quality")).toBeInTheDocument();
    expect(screen.getByTestId("production-subtitles")).toBeInTheDocument();
    expect(screen.getByTestId("production-background-music")).toBeInTheDocument();
    // Cấu hình đứng trước hành động dùng nó (docs/ux-ui-design-rules.md §1).
    const settings = screen.getByTestId("production-settings");
    const retry = screen.getByRole("alert");
    expect(settings.compareDocumentPosition(retry) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  });

  it("khi Merge lỗi chỉ cho sửa phụ đề và nhạc nền — Render đã chạy xong", async () => {
    stubProject({ project_id: "p1", status: "failed_at_assemble_video", scenes: [], error_message: "ffmpeg" });

    renderRenderPage();

    await waitFor(() => expect(screen.getByTestId("production-settings")).toBeInTheDocument());
    expect(screen.queryByTestId("production-render-quality")).not.toBeInTheDocument();
    expect(screen.getByTestId("production-subtitles")).toBeInTheDocument();
  });

  it("không hiện cấu hình khi đang chạy", async () => {
    stubProject({ project_id: "p1", status: "rendering", scenes: [] });

    renderRenderPage();

    await waitFor(() => expect(screen.getByTestId("progress-tracker-steps")).toBeInTheDocument());
    expect(screen.queryByTestId("production-settings")).not.toBeInTheDocument();
  });
});
