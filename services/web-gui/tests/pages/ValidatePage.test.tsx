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

describe("ValidatePage (bước 7 — chạy thử, bước 8 — duyệt)", () => {
  beforeEach(() => {
    FakeEventSource.instances = [];
    // @ts-expect-error test stub
    global.EventSource = FakeEventSource;
  });

  afterEach(() => {
    vi.restoreAllMocks();
    window.localStorage.clear();
  });

  it("chỉ hiện hai việc chạy thử, không hiện phần sản xuất của bước 9–12", async () => {
    stubProject({ project_id: "p1", status: "validating_script", scenes: [] });

    renderValidatePage();

    await waitFor(() => expect(screen.getByTestId("progress-tracker-steps")).toBeInTheDocument());
    const steps = screen.getByTestId("progress-tracker-steps");
    expect(steps).toHaveTextContent("Phân tích kịch bản");
    expect(steps).toHaveTextContent("Chạy thử & kiểm tra");
    expect(steps).not.toHaveTextContent("Tạo giọng đọc");
  });

  // The two jobs inside step 7 are numbered 7.1/7.2 so they do not read as
  // flow steps 1 and 2; the title names the step as the rail does.
  it("đánh số hai việc là 7.1/7.2 và tiêu đề là “Bước 7 — Validate”", async () => {
    // parsing: 7.1 is active and 7.2 pending, so both dots show their number.
    stubProject({ project_id: "p1", status: "parsing_script", scenes: [] });

    renderValidatePage();

    await waitFor(() => expect(screen.getByTestId("progress-tracker-steps")).toBeInTheDocument());
    const items = screen.getByTestId("progress-tracker-steps").querySelectorAll("li");
    expect(Array.from(items).map((li) => li.textContent)).toEqual(["7.1Phân tích kịch bản", "7.2Chạy thử & kiểm tra"]);
    expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("Bước 7 — Validate");
  });

  it("khi chờ duyệt, tiêu đề là “Bước 8 — Review”", async () => {
    stubProject({ project_id: "p1", status: "awaiting_review", scenes: [], beats: [] });

    renderValidatePage();

    await waitFor(() => expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("Bước 8 — Review"));
    expect(screen.getByTestId("outline-review")).toHaveTextContent("Duyệt dàn ý trước khi sản xuất");
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

  it("chọn cấu hình dựng & ghép ở màn duyệt, phía trên nút duyệt", async () => {
    stubProject({ project_id: "p1", status: "awaiting_review", scenes: [], beats: [] });

    renderValidatePage();

    await waitFor(() => expect(screen.getByTestId("production-settings")).toBeInTheDocument());
    expect(screen.getByTestId("production-render-quality")).toBeInTheDocument();
    expect(screen.getByTestId("production-subtitles")).toBeInTheDocument();
    const settings = screen.getByTestId("production-settings");
    const approve = screen.getByTestId("outline-approve");
    expect(settings.compareDocumentPosition(approve) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
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
