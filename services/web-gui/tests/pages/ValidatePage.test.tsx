import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { ValidatePage } from "../../src/pages/ValidatePage";
import { ProjectDraftProvider } from "../../src/context/ProjectDraftContext";
import { ThemeProvider } from "../../src/context/ThemeContext";

class FakeEventSource {
  static instances: FakeEventSource[] = [];
  onmessage: ((event: MessageEvent) => void) | null = null;
  closed = false;

  constructor(public url: string) {
    FakeEventSource.instances.push(this);
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

function renderValidatePage() {
  return render(
    <ThemeProvider>
      <MemoryRouter initialEntries={["/projects/p1/validate"]}>
        <ProjectDraftProvider>
          <Routes>
            <Route path="/projects/:id/validate" element={<ValidatePage />} />
            <Route path="/projects/:id/render" element={<div data-testid="render-page-stub" />} />
            <Route path="/projects/:id/result" element={<div data-testid="result-page-stub" />} />
            <Route path="/" element={<div data-testid="new-project-page-stub" />} />
            <Route path="/projects/:id/resume" element={<div data-testid="resume-page-stub" />} />
          </Routes>
        </ProjectDraftProvider>
      </MemoryRouter>
    </ThemeProvider>,
  );
}

describe("ValidatePage (bước 4 — chạy thử & duyệt)", () => {
  beforeEach(() => {
    FakeEventSource.instances = [];
    // @ts-expect-error test stub
    global.EventSource = FakeEventSource;
  });

  afterEach(() => {
    vi.restoreAllMocks();
    window.localStorage.clear();
  });

  it("chỉ hiện hai bước chạy thử, không hiện phần sản xuất của bước 5", async () => {
    stubProject({ project_id: "p1", status: "validating_script", scenes: [] });

    renderValidatePage();

    await waitFor(() => expect(screen.getByTestId("progress-tracker-steps")).toBeInTheDocument());
    const steps = screen.getByTestId("progress-tracker-steps");
    expect(steps).toHaveTextContent("Phân tích kịch bản");
    expect(steps).toHaveTextContent("Chạy thử & kiểm tra");
    expect(steps).not.toHaveTextContent("Render hoạt hình");
  });

  it("hiện dàn ý và tiến độ cùng lúc, cạnh nhau, khi đang chờ duyệt", async () => {
    // Bug report: OutlineReview (có thể dài hàng chục dòng) xếp chồng lên
    // ProgressTracker trong một cột duy nhất đẩy tiến độ xuống rất xa, làm cả
    // trang giống một bức tường chữ. Cả hai phải cùng hiện.
    stubProject({
      project_id: "p1",
      status: "awaiting_review",
      scenes: [{ scene_index: 0, narration_text: "Vì sao vòng lặp này chạy mãi" }],
      beats: [],
    });

    renderValidatePage();

    await waitFor(() => expect(screen.getByTestId("outline-review")).toBeInTheDocument());
    expect(screen.getByText(/Vì sao vòng lặp này chạy mãi/)).toBeInTheDocument();
    expect(screen.getByTestId("progress-tracker-steps")).toBeInTheDocument();
    expect(screen.getByTestId("outline-approve")).toBeInTheDocument();
  });

  it("chỉ có nút thử lại khi chạy thử hỏng — quay về sửa là việc Creator tự chọn ở thanh bước", async () => {
    stubProject({
      project_id: "p1",
      status: "failed_at_validate_script",
      scenes: [],
      error_message: "Kịch bản không hợp lệ",
    });

    renderValidatePage();

    await waitFor(() => expect(screen.getByTestId("error-banner-retry-button")).toBeInTheDocument());
    expect(screen.queryByTestId("error-banner-back-button")).not.toBeInTheDocument();
    expect(screen.getByTestId("error-banner-detail-toggle")).toBeInTheDocument();
  });

  it("đi tiếp sang bước 5 khi saga đã qua cổng duyệt", async () => {
    // Cùng một effect lo hai việc: duyệt xong thì đi tiếp, và cổng duyệt tắt
    // (ReviewEnabled=false) thì bước 4 không có gì để dừng.
    stubProject({ project_id: "p1", status: "synthesizing_speech", scenes: [] });

    renderValidatePage();

    await waitFor(() => expect(screen.getByTestId("render-page-stub")).toBeInTheDocument());
  });
});
