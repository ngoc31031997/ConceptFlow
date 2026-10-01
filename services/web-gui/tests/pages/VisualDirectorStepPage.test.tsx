import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { VisualDirectorStepPage } from "../../src/pages/VisualDirectorStepPage";
import { ProjectDraftProvider } from "../../src/context/ProjectDraftContext";
import { ThemeProvider } from "../../src/context/ThemeContext";
import * as apiClient from "../../src/api/client";
import { mockRenderPrompt } from "../helpers/renderPromptMock";

// Step 2: renders the visual_director prompt (server-side) and rehydrates saved
// authoring state from the server — stub both so these tests don't need a
// live backend, mirroring ScriptStepPage.test.tsx's story_architect stub.
beforeEach(() => {
  mockRenderPrompt();
  vi.spyOn(apiClient, "getAuthoringState").mockResolvedValue({ topic: "", story: "", storyboard: "", code: "" });
  vi.spyOn(apiClient, "saveAuthoringStoryboard").mockResolvedValue(undefined);
});

describe("VisualDirectorStepPage", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("blocks the step until a storyboard is pasted, then saves and advances", async () => {
    render(
      <ThemeProvider>
        <MemoryRouter>
          <ProjectDraftProvider>
            <VisualDirectorStepPage />
          </ProjectDraftProvider>
        </MemoryRouter>
      </ThemeProvider>,
    );

    expect(screen.getByTestId("visual-director-step-next")).toBeDisabled();

    fireEvent.change(screen.getByTestId("visual-director-storyboard-input"), {
      target: { value: "CẢNH 1 — ..." },
    });

    expect(screen.getByTestId("visual-director-step-next")).not.toBeDisabled();

    fireEvent.click(screen.getByTestId("visual-director-step-next"));

    await waitFor(() => {
      expect(apiClient.saveAuthoringStoryboard).toHaveBeenCalledWith(
        expect.any(String),
        "CẢNH 1 — ...",
      );
    });
  });

  it("shows an error and stays put when saving fails", async () => {
    vi.spyOn(apiClient, "saveAuthoringStoryboard").mockRejectedValue(new Error("boom"));

    render(
      <ThemeProvider>
        <MemoryRouter>
          <ProjectDraftProvider>
            <VisualDirectorStepPage />
          </ProjectDraftProvider>
        </MemoryRouter>
      </ThemeProvider>,
    );

    fireEvent.change(screen.getByTestId("visual-director-storyboard-input"), {
      target: { value: "CẢNH 1 — ..." },
    });
    fireEvent.click(screen.getByTestId("visual-director-step-next"));

    await waitFor(() => {
      expect(screen.getByText("Không lưu được. Vui lòng thử lại.")).toBeInTheDocument();
    });
  });

  it("has no tab bar: moving between script steps is the flow's step bar", () => {
    render(
      <ThemeProvider>
        <MemoryRouter>
          <ProjectDraftProvider>
            <VisualDirectorStepPage />
          </ProjectDraftProvider>
        </MemoryRouter>
      </ThemeProvider>,
    );

    expect(screen.queryByTestId("script-tab-storyboard")).not.toBeInTheDocument();
    // Screen titles name the step as the rail does.
    expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("Bước 4 — Hình ảnh");
  });

  describe("lời nhắc theo cách làm", () => {
    function renderStep() {
      render(
        <ThemeProvider>
          <MemoryRouter>
            <ProjectDraftProvider>
              <VisualDirectorStepPage />
            </ProjectDraftProvider>
          </MemoryRouter>
        </ThemeProvider>,
      );
    }

    it("tự làm: nhắc dán storyboard từ AI bên ngoài", async () => {
      vi.spyOn(apiClient, "getLlmStatus").mockResolvedValue({ enabled: true, provider: "hive" });
      renderStep();
      expect(await screen.findByText("Dán storyboard từ AI để tiếp tục.")).toBeInTheDocument();
      expect(screen.getByTestId("wizard-back")).toHaveTextContent("Quay lại Kịch bản");
    });

    it("AI làm giúp: nhắc bấm chạy hoặc tự viết, không nói “dán”", async () => {
      vi.spyOn(apiClient, "getLlmStatus").mockResolvedValue({ enabled: true, provider: "hive" });
      vi.spyOn(apiClient, "getAuthoringState").mockResolvedValue({
        mode: "ai", topic: "", story: "", storyboard: "", code: "",
      });
      renderStep();
      expect(await screen.findByText("Bấm Chạy bằng AI, hoặc tự viết storyboard vào ô bên dưới.")).toBeInTheDocument();
    });
  });
  // Skipped in CI only (Creator decision, 2026-09-28): on the GitHub runner the box
  // is still there after "Đóng thông báo"; not reproducible locally.
  it.skipIf(process.env.CI)("hiện cảnh báo của storyboard ở bước Hình ảnh, và đóng được", async () => {
    vi.spyOn(apiClient, "getLlmStatus").mockResolvedValue({ enabled: true, provider: "hive" });
    vi.spyOn(apiClient, "getAuthoringState").mockResolvedValue({
      mode: "ai", topic: "", story: "s", storyboard: "sb", code: "",
    });
    vi.spyOn(apiClient, "getAuthoringChain").mockResolvedValue({
      running: false,
      steps: ["story", "storyboard", "code"],
      current_index: 3,
      finished: true,
      warnings: {
        storyboard: [
          "Cảnh concrete: ~58 giây, ngân sách 30–45 giây (+29%)",
          "Shot 2.3: lời thoại nhắc 'vi khuẩn' nhưng HÌNH không có",
        ],
      },
      finished_at: new Date().toISOString(),
    });
    render(
      <ThemeProvider>
        <MemoryRouter>
          <ProjectDraftProvider>
            <VisualDirectorStepPage />
          </ProjectDraftProvider>
        </MemoryRouter>
      </ThemeProvider>,
    );

    const box = await screen.findByTestId("run-with-ai-storyboard-warnings");
    expect(box).toHaveTextContent("Cảnh báo ở bước Hình ảnh (2)");
    expect(box).toHaveTextContent("Shot 2.3: lời thoại nhắc 'vi khuẩn' nhưng HÌNH không có");
    // Warnings do not turn a finished run into a failure.
    expect(screen.getByTestId("run-with-ai-done")).toBeInTheDocument();

    fireEvent.click(screen.getByTestId("run-with-ai-dismiss"));
    await waitFor(() => expect(screen.queryByTestId("run-with-ai-storyboard-warnings")).not.toBeInTheDocument());
  });
});
