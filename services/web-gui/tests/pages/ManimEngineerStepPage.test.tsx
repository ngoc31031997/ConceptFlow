import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { ManimEngineerStepPage } from "../../src/pages/ManimEngineerStepPage";
import { ProjectDraftProvider } from "../../src/context/ProjectDraftContext";
import { ThemeProvider } from "../../src/context/ThemeContext";
import * as apiClient from "../../src/api/client";

const VALID_CODE = [
  "from conceptflow import *",
  "",
  "class DemoScene(ConceptFlowScene):",
  '    def construct(self):',
  '        self.narrate("Xin chao")',
].join("\n");

// CR-025 step 3: fetches the manim_engineer template and rehydrates saved
// authoring state from the server — stub both so these tests don't need a
// live backend, mirroring VisualDirectorStepPage.test.tsx's stub.
beforeEach(() => {
  vi.spyOn(apiClient, "getPromptTemplate").mockResolvedValue({
    role: "manim_engineer",
    language: "vi",
    version: 1,
    template_text: "TIEN DE: {{previous_output}}",
  });
  vi.spyOn(apiClient, "getAuthoringState").mockResolvedValue({
    topic: "",
    story: "",
    storyboard: "",
    code: "",
    review: "",
  });
  vi.spyOn(apiClient, "saveAuthoringCode").mockResolvedValue(undefined);
});

describe("ManimEngineerStepPage", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("blocks the step until valid Manim code is pasted, then saves and advances", async () => {
    render(
      <ThemeProvider>
        <MemoryRouter>
          <ProjectDraftProvider>
            <ManimEngineerStepPage />
          </ProjectDraftProvider>
        </MemoryRouter>
      </ThemeProvider>,
    );

    expect(screen.getByTestId("manim-engineer-step-next")).toBeDisabled();

    fireEvent.change(screen.getByTestId("manim-engineer-code-input"), {
      target: { value: "not valid python at all" },
    });
    expect(screen.getByTestId("manim-engineer-step-next")).toBeDisabled();

    fireEvent.change(screen.getByTestId("manim-engineer-code-input"), {
      target: { value: VALID_CODE },
    });
    expect(screen.getByTestId("manim-engineer-step-next")).not.toBeDisabled();

    fireEvent.click(screen.getByTestId("manim-engineer-step-next"));

    await waitFor(() => {
      expect(apiClient.saveAuthoringCode).toHaveBeenCalledWith(expect.any(String), VALID_CODE);
    });
  });

  it("shows an error and stays put when saving fails", async () => {
    vi.spyOn(apiClient, "saveAuthoringCode").mockRejectedValue(new Error("boom"));

    render(
      <ThemeProvider>
        <MemoryRouter>
          <ProjectDraftProvider>
            <ManimEngineerStepPage />
          </ProjectDraftProvider>
        </MemoryRouter>
      </ThemeProvider>,
    );

    fireEvent.change(screen.getByTestId("manim-engineer-code-input"), {
      target: { value: VALID_CODE },
    });
    fireEvent.click(screen.getByTestId("manim-engineer-step-next"));

    await waitFor(() => {
      expect(screen.getByText("Không lưu được code, thử lại.")).toBeInTheDocument();
    });
  });

  it("strips a pasted markdown code fence instead of saving the ``` markers as part of the script", () => {
    // The engineer prompt asks the AI to answer with exactly one ```python
    // (or ```tsx) fenced block. Pasting that whole block, fence included, is
    // the single most common way this round trip fails: esbuild/ast.parse
    // chokes on line 1 with an opaque syntax error that says nothing about
    // the real cause.
    render(
      <ThemeProvider>
        <MemoryRouter>
          <ProjectDraftProvider>
            <ManimEngineerStepPage />
          </ProjectDraftProvider>
        </MemoryRouter>
      </ThemeProvider>,
    );

    fireEvent.change(screen.getByTestId("manim-engineer-code-input"), {
      target: { value: "```python\n" + VALID_CODE + "\n```" },
    });

    expect(screen.getByTestId("manim-engineer-code-input")).toHaveValue(VALID_CODE);
  });

  // feature/remotion-engine: the remotion_engineer prompt is topic -> code,
  // so it carries {{topic}}. Leaving it unsubstituted hands the Creator a
  // prompt that still says "paste your topic here".
  it("fills {{topic}} in the engineer prompt instead of leaving the raw token", async () => {
    vi.mocked(apiClient.getPromptTemplate).mockResolvedValue({
      role: "remotion_engineer",
      language: "vi",
      version: 1,
      template_text: "CHU DE VIDEO: {{topic}}",
    });

    render(
      <ThemeProvider>
        <MemoryRouter>
          <ProjectDraftProvider>
            <ManimEngineerStepPage />
          </ProjectDraftProvider>
        </MemoryRouter>
      </ThemeProvider>,
    );

    await waitFor(() => {
      expect(screen.getByTestId("manim-engineer-prompt")).toHaveValue(
        "CHU DE VIDEO: [DÁN CHỦ ĐỀ CỦA BẠN VÀO ĐÂY]",
      );
    });
  });

  it("shows the pipeline tab bar (1c active) and the render engine picker", () => {
    render(
      <ThemeProvider>
        <MemoryRouter>
          <ProjectDraftProvider>
            <ManimEngineerStepPage />
          </ProjectDraftProvider>
        </MemoryRouter>
      </ThemeProvider>,
    );

    expect(screen.getByTestId("script-tab-code")).toHaveAttribute("aria-selected", "true");
    // feature/remotion-engine: the engine choice lives on THIS tab now, not
    // on the situation-chooser page — switching it must fetch the matching
    // prompt role (remotion_engineer instead of manim_engineer).
    expect(screen.getByTestId("render-engine-picker")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("render-engine-remotion"));
    expect(apiClient.getPromptTemplate).toHaveBeenCalledWith("remotion_engineer", "vi");
  });

  it("keeps typed code across a re-render instead of a local buffer that a tab switch would lose", () => {
    const { unmount } = render(
      <ThemeProvider>
        <MemoryRouter>
          <ProjectDraftProvider>
            <ManimEngineerStepPage />
          </ProjectDraftProvider>
        </MemoryRouter>
      </ThemeProvider>,
    );

    fireEvent.change(screen.getByTestId("manim-engineer-code-input"), {
      target: { value: VALID_CODE },
    });
    unmount();

    render(
      <ThemeProvider>
        <MemoryRouter>
          <ProjectDraftProvider>
            <ManimEngineerStepPage />
          </ProjectDraftProvider>
        </MemoryRouter>
      </ThemeProvider>,
    );
    // draft.scriptContent (persisted to localStorage by ProjectDraftProvider)
    // is the source of truth now, not a local useState buffer that would
    // reset to draft.scriptContent-at-mount-time and drop unsaved typing.
    expect(screen.getByTestId("manim-engineer-code-input")).toHaveValue(VALID_CODE);
  });

  // CR-027 FR79 — chế độ đã chọn ở tab 1a áp cho cả bước 1: tab này đọc nó từ
  // draft (đã persist sang localStorage) chứ không hỏi lại.
  describe("chế độ làm bước 1 (CR-027 FR79)", () => {
    afterEach(() => {
      window.localStorage.clear();
    });

    function renderWithAiMode() {
      window.localStorage.setItem(
        "conceptflow.draft.v1",
        JSON.stringify({
          projectId: "p-123",
          voiceLanguage: "vi",
          authoringMode: "ai",
          authoringStory: "dàn ý",
          authoringStoryboard: "storyboard",
        }),
      );
      return render(
        <ThemeProvider>
          <MemoryRouter>
            <ProjectDraftProvider>
              <ManimEngineerStepPage />
            </ProjectDraftProvider>
          </MemoryRouter>
        </ThemeProvider>,
      );
    }

    it("mang chế độ AI sang tab 1c: ẩn prompt copy tay, hiện nút chạy", async () => {
      vi.spyOn(apiClient, "getLlmStatus").mockResolvedValue({ enabled: true, provider: "hive" });
      renderWithAiMode();

      await waitFor(() => expect(screen.getByTestId("run-with-ai-code")).toBeInTheDocument());
      expect(screen.queryByTestId("manim-engineer-prompt")).not.toBeInTheDocument();
      expect(screen.queryByTestId("manim-engineer-copy")).not.toBeInTheDocument();
    });

    it("điền code AI sinh ra vào ô soạn thảo, lint vẫn chạy như khi dán tay", async () => {
      vi.spyOn(apiClient, "getLlmStatus").mockResolvedValue({ enabled: true, provider: "hive" });
      vi.spyOn(apiClient, "generateAuthoringStep").mockResolvedValue({
        step: "code",
        role: "manim_engineer",
        content: VALID_CODE,
        provider: "hive",
        usage: { model: "deepseek" },
      });
      renderWithAiMode();

      await waitFor(() => expect(screen.getByTestId("run-with-ai-code")).not.toBeDisabled());
      fireEvent.click(screen.getByTestId("run-with-ai-code"));

      await waitFor(() =>
        expect(screen.getByTestId("manim-engineer-code-input")).toHaveValue(VALID_CODE),
      );
      // Code hợp lệ theo lint client → bước mở ra, đúng như khi Creator dán tay.
      expect(screen.getByTestId("manim-engineer-step-next")).not.toBeDisabled();
    });

    it("quay về copy tay khi máy chủ chưa có API key, dù draft chọn AI", async () => {
      vi.spyOn(apiClient, "getLlmStatus").mockResolvedValue({
        enabled: false,
        provider: "",
        reason: "Chưa cấu hình HIVE_API_KEY",
      });
      renderWithAiMode();

      await waitFor(() => expect(screen.getByTestId("manim-engineer-prompt")).toBeInTheDocument());
      expect(screen.queryByTestId("run-with-ai-code")).not.toBeInTheDocument();
      expect(screen.getByTestId("manim-engineer-copy")).toBeInTheDocument();
    });
  });
});
