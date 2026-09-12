import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { ScriptReviewerStepPage } from "../../src/pages/ScriptReviewerStepPage";
import { ProjectDraftProvider } from "../../src/context/ProjectDraftContext";
import { ThemeProvider } from "../../src/context/ThemeContext";
import * as apiClient from "../../src/api/client";

// CR-025 step 4: fetches the script_reviewer template and rehydrates saved
// authoring state from the server — stub both so these tests don't need a
// live backend, mirroring ManimEngineerStepPage.test.tsx's stub.
beforeEach(() => {
  vi.spyOn(apiClient, "getPromptTemplate").mockResolvedValue({
    role: "script_reviewer",
    language: "vi",
    version: 1,
    template_text: "NOI DUNG: {{previous_output}} LINT: {{lint_results}}",
  });
  vi.spyOn(apiClient, "getAuthoringState").mockResolvedValue({
    story: "dan y",
    storyboard: "storyboard",
    code: "from conceptflow import *\nclass X(ConceptFlowScene):\n    def construct(self):\n        pass",
    review: "",
  });
  vi.spyOn(apiClient, "saveAuthoringReview").mockResolvedValue(undefined);
});

describe("ScriptReviewerStepPage", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("blocks the step until a verdict is pasted, then saves and advances", async () => {
    render(
      <ThemeProvider>
        <MemoryRouter>
          <ProjectDraftProvider>
            <ScriptReviewerStepPage />
          </ProjectDraftProvider>
        </MemoryRouter>
      </ThemeProvider>,
    );

    expect(screen.getByTestId("script-reviewer-step-next")).toBeDisabled();

    fireEvent.change(screen.getByTestId("script-reviewer-verdict-input"), {
      target: { value: "VERDICT: PASS" },
    });

    expect(screen.getByTestId("script-reviewer-step-next")).not.toBeDisabled();

    fireEvent.click(screen.getByTestId("script-reviewer-step-next"));

    await waitFor(() => {
      expect(apiClient.saveAuthoringReview).toHaveBeenCalledWith(expect.any(String), "VERDICT: PASS");
    });
  });

  it("shows an error and stays put when saving fails", async () => {
    vi.spyOn(apiClient, "saveAuthoringReview").mockRejectedValue(new Error("boom"));

    render(
      <ThemeProvider>
        <MemoryRouter>
          <ProjectDraftProvider>
            <ScriptReviewerStepPage />
          </ProjectDraftProvider>
        </MemoryRouter>
      </ThemeProvider>,
    );

    fireEvent.change(screen.getByTestId("script-reviewer-verdict-input"), {
      target: { value: "VERDICT: REVISE" },
    });
    fireEvent.click(screen.getByTestId("script-reviewer-step-next"));

    await waitFor(() => {
      expect(screen.getByText("Không lưu được kết quả đánh giá, thử lại.")).toBeInTheDocument();
    });
  });

  it("offers a back link to Manim Engineer for the REVISE case", async () => {
    render(
      <ThemeProvider>
        <MemoryRouter>
          <ProjectDraftProvider>
            <ScriptReviewerStepPage />
          </ProjectDraftProvider>
        </MemoryRouter>
      </ThemeProvider>,
    );

    await waitFor(() => {
      expect(screen.getByTestId("wizard-back")).toBeInTheDocument();
    });
  });
});
