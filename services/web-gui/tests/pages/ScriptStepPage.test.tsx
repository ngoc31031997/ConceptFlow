import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { ScriptStepPage } from "../../src/pages/ScriptStepPage";
import { ProjectDraftProvider } from "../../src/context/ProjectDraftContext";
import { ThemeProvider } from "../../src/context/ThemeContext";

function renderPage() {
  global.fetch = vi.fn().mockResolvedValue({
    ok: true,
    json: async () => ({ connected: false }),
  }) as unknown as typeof fetch;

  return render(
    <ThemeProvider>
      <MemoryRouter initialEntries={["/"]}>
        <ProjectDraftProvider>
          <Routes>
            <Route path="/" element={<ScriptStepPage />} />
            <Route path="/create/script/outline" element={<div data-testid="landed-on-outline" />} />
            <Route path="/create/script/storyboard" element={<div data-testid="landed-on-storyboard" />} />
            <Route path="/create/script/code" element={<div data-testid="landed-on-code" />} />
          </Routes>
        </ProjectDraftProvider>
      </MemoryRouter>
    </ThemeProvider>,
  );
}

describe("ScriptStepPage", () => {
  afterEach(() => {
    vi.restoreAllMocks();
    window.localStorage.clear();
  });

  // CR-031 — bốn tình huống, mỗi cái là một điểm vào khác nhau của chuỗi
  // 1a → 1b → 1c. Đây là cả nội dung của bước 1: chọn sai điểm vào thì Creator
  // hoặc phải bỏ qua một tab bằng tay, hoặc mất luôn đường dán artefact sẵn có.
  it.each([
    ["idea", "landed-on-outline"],
    ["outline", "landed-on-outline"],
    ["storyboard", "landed-on-storyboard"],
    ["code", "landed-on-code"],
  ])("đưa tình huống '%s' vào đúng tab của chuỗi script", (source, landingTestId) => {
    renderPage();

    fireEvent.click(screen.getByTestId(`script-source-${source}`));
    fireEvent.click(screen.getByTestId("script-step-next"));

    expect(screen.getByTestId(landingTestId)).toBeInTheDocument();
  });

  it("mặc định vào 'chỉ có ý tưởng' nên Tiếp tục luôn đi được, không chặn", () => {
    // Trang này không còn ô soạn thảo nào để chặn: mọi việc nhập liệu đã sang
    // các tab 1a–1c, nên một nút Tiếp tục bị khoá ở đây chỉ là ngõ cụt.
    renderPage();

    expect(screen.getByTestId("script-step-next")).not.toBeDisabled();
    fireEvent.click(screen.getByTestId("script-step-next"));
    expect(screen.getByTestId("landed-on-outline")).toBeInTheDocument();
  });

  it("chốt ngôn ngữ, engine và cách làm ngay tại đây vì cả ba chi phối mọi tab sau", async () => {
    renderPage();

    expect(screen.getByTestId("render-engine-remotion")).toBeInTheDocument();
    // AuthoringModeBar chờ biết máy chủ có API key hay không rồi mới vẽ, nên
    // nó xuất hiện sau một vòng fetch.
    await waitFor(() => expect(screen.getByTestId("authoring-mode-bar")).toBeInTheDocument());

    fireEvent.click(screen.getByTestId("render-engine-remotion"));

    // Chữ của tình huống "đã có code" đi theo engine — Creator phải biết mình
    // sắp dán Remotion hay Manim trước khi bấm vào.
    expect(screen.getByTestId("script-source-code")).toHaveTextContent("code Remotion");
  });
});

describe("ScriptStepPage draft lifecycle", () => {
  afterEach(() => {
    vi.restoreAllMocks();
    window.localStorage.clear();
  });

  it("starts a fresh draft when the stored one already began a render", () => {
    // Without this, going back to "/" after a render reused the same
    // project_id and the next submit overwrote the previous video.
    window.localStorage.setItem(
      "conceptflow.draft.v1",
      JSON.stringify({ projectId: "spent-project", scriptContent: "old script", hasSubmitted: true }),
    );

    renderPage();

    fireEvent.click(screen.getByTestId("script-step-next"));
    // Draft mới quay về tình huống mặc định "idea", nên Tiếp tục đi vào 1a —
    // không phải vào 1c như script đã tiêu ở draft cũ.
    expect(screen.getByTestId("landed-on-outline")).toBeInTheDocument();
  });

  it("dịch tên tình huống cũ trong localStorage sang tên mới thay vì bỏ trắng lựa chọn", () => {
    // Draft lưu trước CR-031 mang "ready" — nghĩa là đã có code đúng chuẩn.
    window.localStorage.setItem(
      "conceptflow.draft.v1",
      JSON.stringify({ projectId: "old-project", scriptSource: "ready", hasSubmitted: false }),
    );

    renderPage();

    fireEvent.click(screen.getByTestId("script-step-next"));
    expect(screen.getByTestId("landed-on-code")).toBeInTheDocument();
  });
});
