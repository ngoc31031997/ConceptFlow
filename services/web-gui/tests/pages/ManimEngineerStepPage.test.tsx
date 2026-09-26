import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { ManimEngineerStepPage } from "../../src/pages/ManimEngineerStepPage";
import { ProjectDraftProvider } from "../../src/context/ProjectDraftContext";
import { ThemeProvider } from "../../src/context/ThemeContext";
import * as apiClient from "../../src/api/client";
import { mockRenderPrompt } from "../helpers/renderPromptMock";

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
  mockRenderPrompt();
  vi.spyOn(apiClient, "getAuthoringState").mockResolvedValue({
    topic: "",
    story: "",
    storyboard: "",
    code: "",
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

  // CR-040 FR113: the {{topic}} substitution lives on the server now (and is held
  // to the old TypeScript output by its golden tests). This proves the wiring: the
  // page sends the draft's values and shows what the server answers.
  it("asks the server to render the engineer prompt and shows its answer", async () => {
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
      expect((screen.getByTestId("manim-engineer-prompt") as HTMLTextAreaElement).value).toContain(
        "RENDERED[manim_engineer]",
      );
    });
    expect(apiClient.renderPrompt).toHaveBeenCalledWith(
      expect.objectContaining({ role: "manim_engineer", language: expect.stringMatching(/^(vi|en)$/) }),
    );
  });

  it("shows the render engine picker", async () => {
    render(
      <ThemeProvider>
        <MemoryRouter>
          <ProjectDraftProvider>
            <ManimEngineerStepPage />
          </ProjectDraftProvider>
        </MemoryRouter>
      </ThemeProvider>,
    );

    // feature/remotion-engine: the engine choice lives on THIS tab now, not
    // on the situation-chooser page — switching it must fetch the matching
    // prompt role (remotion_engineer instead of manim_engineer).
    // CR-031 — thu gọn sau PipelineSettingsBar's "Đổi".
    fireEvent.click(screen.getByTestId("pipeline-settings-toggle"));
    expect(screen.getByTestId("render-engine-picker")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("render-engine-remotion"));
    await waitFor(() =>
      expect(apiClient.renderPrompt).toHaveBeenCalledWith(expect.objectContaining({ role: "remotion_engineer" })),
    );
  });
});
