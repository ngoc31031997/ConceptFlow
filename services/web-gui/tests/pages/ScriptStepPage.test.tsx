import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { ScriptStepPage } from "../../src/pages/ScriptStepPage";
import { ProjectDraftProvider } from "../../src/context/ProjectDraftContext";
import { ThemeProvider } from "../../src/context/ThemeContext";

function renderPage(fetchImpl: () => Promise<unknown>) {
  const fetchMock = vi.fn(fetchImpl);
  global.fetch = fetchMock as unknown as typeof fetch;

  render(
    <ThemeProvider>
      <MemoryRouter initialEntries={["/"]}>
        <ProjectDraftProvider>
          <Routes>
            <Route path="/" element={<ScriptStepPage />} />
            <Route path="/create/script/settings" element={<div data-testid="landed-on-settings" />} />
          </Routes>
        </ProjectDraftProvider>
      </MemoryRouter>
    </ThemeProvider>,
  );
  return fetchMock;
}

const createdOk = () =>
  Promise.resolve({
    ok: true,
    status: 201,
    json: async () => ({ project_id: "p1", similar_projects: [] }),
  });

function typeTopic(value: string) {
  fireEvent.change(screen.getByTestId("script-step-topic"), { target: { value } });
}

describe("ScriptStepPage (Bước 1 — Ý tưởng)", () => {
  afterEach(() => {
    vi.restoreAllMocks();
    window.localStorage.clear();
  });

  it("khoá Tiếp tục khi chủ đề còn trống hoặc chỉ có khoảng trắng", () => {
    renderPage(createdOk);

    expect(screen.getByTestId("script-step-next")).toBeDisabled();
    typeTopic("   ");
    expect(screen.getByTestId("script-step-next")).toBeDisabled();
    typeTopic("Vòng lặp for trong Java");
    expect(screen.getByTestId("script-step-next")).not.toBeDisabled();
  });

  it("tạo project bằng đúng một lệnh POST rồi sang Bước 2", async () => {
    const fetchMock = renderPage(createdOk);

    typeTopic("  Vòng lặp for trong Java  ");
    fireEvent.click(screen.getByTestId("script-step-next"));

    await waitFor(() => expect(screen.getByTestId("landed-on-settings")).toBeInTheDocument());

    // Ghi vị trí wizard (PUT .../wizard-position) là lệnh phụ, không tính vào
    // "đúng một lệnh POST tạo project".
    const creates = (fetchMock.mock.calls as unknown as [string, RequestInit][]).filter(
      ([u]) => !u.includes("/wizard-position"),
    );
    expect(creates).toHaveLength(1);
    const [url, init] = creates[0];
    expect(url).toMatch(/\/v1\/projects$/);
    expect(init.method).toBe("POST");
    expect(JSON.parse(init.body as string)).toMatchObject({ topic: "Vòng lặp for trong Java" });
  });

  it("báo lỗi và ở lại trang khi không tạo được project", async () => {
    renderPage(() => Promise.reject(new Error("network down")));

    typeTopic("Đệ quy");
    fireEvent.click(screen.getByTestId("script-step-next"));

    await waitFor(() => expect(screen.getByText(/Không tạo được dự án/)).toBeInTheDocument());
    expect(screen.queryByTestId("landed-on-settings")).not.toBeInTheDocument();
    // Lỗi xong thì Creator thử lại được, không bị kẹt ở trạng thái "Đang tạo".
    expect(screen.getByTestId("script-step-next")).not.toBeDisabled();
  });

  it("bắt đầu draft mới khi draft đã lưu từng chạy render", () => {
    // Không có việc này thì quay lại "/" sau một lần render sẽ dùng lại
    // project_id cũ và lần tạo tiếp theo ghi đè video trước.
    window.localStorage.setItem(
      "conceptflow.draft.v1",
      JSON.stringify({
        projectId: "spent-project",
        authoringTopic: "chủ đề cũ",
        hasSubmitted: true,
      }),
    );

    renderPage(createdOk);

    expect((screen.getByTestId("script-step-topic") as HTMLTextAreaElement).value).toBe("");
    expect(screen.getByTestId("script-step-next")).toBeDisabled();
  });
});
