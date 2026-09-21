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
    topic: "",
    story: "dan y",
    storyboard: "storyboard",
    code: "from conceptflow import *\nclass X(ConceptFlowScene):\n    def construct(self):\n        pass",
    review: "",
  });
  vi.spyOn(apiClient, "saveAuthoringReview").mockResolvedValue(undefined);
  vi.spyOn(apiClient, "saveAuthoringCode").mockResolvedValue(undefined);
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

    // A REVISE verdict is meant to block "Tiếp tục" (see the test below) — use
    // PASS here so this test still exercises the save-fails path.
    fireEvent.change(screen.getByTestId("script-reviewer-verdict-input"), {
      target: { value: "VERDICT: PASS" },
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

  it("blocks continuing and never saves when the AI verdict is REVISE", async () => {
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
      target: { value: "VERDICT: REVISE\n\nSửa lại đoạn mở đầu." },
    });

    await waitFor(() => {
      expect(screen.getByTestId("script-reviewer-step-next")).toBeDisabled();
    });
    // The back button stays enabled — REVISE is exactly the case where the
    // Creator needs to go fix the script, not get stuck.
    expect(screen.getByTestId("wizard-back")).not.toBeDisabled();

    fireEvent.click(screen.getByTestId("script-reviewer-step-next"));
    expect(apiClient.saveAuthoringReview).not.toHaveBeenCalled();
  });

  it("strips a pasted markdown code fence in the fix-code textarea too", async () => {
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
      target: { value: "VERDICT: REVISE\n\nThiếu class Scene." },
    });

    await waitFor(() => expect(screen.getByTestId("script-reviewer-fix-code-input")).toBeInTheDocument());
    const fixed = "from conceptflow import *\nclass FixedScene(ConceptFlowScene):\n    def construct(self):\n        pass";
    fireEvent.change(screen.getByTestId("script-reviewer-fix-code-input"), {
      target: { value: "```python\n" + fixed + "\n```" },
    });

    expect(screen.getByTestId("script-reviewer-fix-code-input")).toHaveValue(fixed);
  });

  it("offers a fix prompt and code-save loop for the REVISE case", async () => {
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
      target: { value: "VERDICT: REVISE\n\nThiếu class Scene." },
    });

    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText } });

    await waitFor(() => expect(screen.getByTestId("script-reviewer-fix-prompt")).toBeInTheDocument());
    fireEvent.click(screen.getByTestId("script-reviewer-fix-copy"));
    await waitFor(() => expect(writeText).toHaveBeenCalledTimes(1));
    // The fix prompt must carry the reviewer's own feedback forward, not just
    // regenerate a generic engineer prompt.
    expect(writeText.mock.calls[0][0]).toContain("Thiếu class Scene.");

    fireEvent.change(screen.getByTestId("script-reviewer-fix-code-input"), {
      target: { value: "from conceptflow import *\nclass FixedScene(ConceptFlowScene):\n    def construct(self):\n        pass" },
    });
    fireEvent.click(screen.getByTestId("script-reviewer-fix-save"));

    await waitFor(() => {
      expect(apiClient.saveAuthoringCode).toHaveBeenCalledWith(
        expect.any(String),
        expect.stringContaining("FixedScene"),
      );
    });
    // Saving the fix clears the stale REVISE verdict so the Creator pastes a
    // fresh review of the corrected code instead of getting stuck blocked.
    await waitFor(() => expect(screen.getByTestId("script-reviewer-verdict-input")).toHaveValue(""));
  });

  it("still offers the fix-prompt tool for a PASS verdict with NÊN SỬA notes, without blocking Tiếp tục", async () => {
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
      target: { value: "VERDICT: PASS\n\n### NỘI DUNG\n- [ ] Thiếu 1 phần tử ở beat 5 — NÊN SỬA" },
    });

    // PASS never blocks continuing — the Creator decides for themselves
    // whether a non-mandatory note is worth fixing before rendering.
    await waitFor(() => expect(screen.getByTestId("script-reviewer-step-next")).not.toBeDisabled());
    // But the fix tool should still be there for them to use if they want to.
    expect(await screen.findByTestId("script-reviewer-fix-prompt")).toBeInTheDocument();

    fireEvent.click(screen.getByTestId("script-reviewer-step-next"));
    await waitFor(() => expect(apiClient.saveAuthoringReview).toHaveBeenCalled());
  });

  it("shows the pipeline tab bar with 1d active", () => {
    render(
      <ThemeProvider>
        <MemoryRouter>
          <ProjectDraftProvider>
            <ScriptReviewerStepPage />
          </ProjectDraftProvider>
        </MemoryRouter>
      </ThemeProvider>,
    );

    expect(screen.getByTestId("script-tab-review")).toHaveAttribute("aria-selected", "true");
    expect(screen.getByTestId("script-tab-code")).not.toBeDisabled();
  });
});
