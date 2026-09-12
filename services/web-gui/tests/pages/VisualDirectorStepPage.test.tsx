import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { VisualDirectorStepPage } from "../../src/pages/VisualDirectorStepPage";
import { ProjectDraftProvider } from "../../src/context/ProjectDraftContext";
import { ThemeProvider } from "../../src/context/ThemeContext";
import * as apiClient from "../../src/api/client";

// CR-025 step 2: fetches the visual_director template and rehydrates saved
// authoring state from the server — stub both so these tests don't need a
// live backend, mirroring ScriptStepPage.test.tsx's story_architect stub.
beforeEach(() => {
  vi.spyOn(apiClient, "getPromptTemplate").mockResolvedValue({
    role: "visual_director",
    language: "vi",
    version: 1,
    template_text: "DÀN Ý: {{previous_output}}",
  });
  vi.spyOn(apiClient, "getAuthoringState").mockResolvedValue({ story: "", storyboard: "", code: "", review: "" });
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
      expect(screen.getByText("Không lưu được storyboard, thử lại.")).toBeInTheDocument();
    });
  });
});
