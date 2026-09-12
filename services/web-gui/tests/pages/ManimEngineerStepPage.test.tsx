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
});
