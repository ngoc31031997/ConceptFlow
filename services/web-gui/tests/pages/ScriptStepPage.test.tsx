import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
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
          </Routes>
        </ProjectDraftProvider>
      </MemoryRouter>
    </ThemeProvider>,
  );
}

describe("ScriptStepPage", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("hands off to the outline sub-wizard tab as soon as 'blank' is picked", () => {
    // "Dựng từ đầu" no longer has any inline UI on this page — it moved to
    // its own 4-tab sub-wizard (ScriptPipelineTabs), reached via routing.
    renderPage();

    expect(screen.getByTestId("script-step-next")).toBeDisabled();
    fireEvent.click(screen.getByTestId("script-source-blank"));

    expect(screen.getByTestId("landed-on-outline")).toBeInTheDocument();
  });

  it("blocks the step until the script is valid (source: draft)", () => {
    renderPage();

    fireEvent.click(screen.getByTestId("script-source-draft"));
    expect(screen.getByTestId("script-step-next")).toBeDisabled();

    fireEvent.change(screen.getByTestId("new-project-script-textarea"), {
      target: {
        value:
          'from conceptflow import *\n\nclass DemoScene(ConceptFlowScene):\n    def construct(self):\n        self.narrate("xin chào")',
      },
    });

    expect(screen.getByTestId("script-step-next")).not.toBeDisabled();
  });

  it("skips the Manim lint entirely for a Remotion draft/ready script", () => {
    renderPage();

    fireEvent.click(screen.getByTestId("render-engine-remotion"));
    fireEvent.click(screen.getByTestId("script-source-ready"));

    fireEvent.change(screen.getByTestId("new-project-script-textarea"), {
      target: { value: 'export const narrations: string[] = ["xin chào"];' },
    });

    expect(screen.getByTestId("script-step-next")).not.toBeDisabled();
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

    // A fresh draft defaults back to scriptSource "blank" — no draft/ready
    // panel, and Next stays disabled until a situation with inline content
    // is picked (or "blank" is picked, which navigates away instead).
    expect(screen.queryByTestId("new-project-script-textarea")).not.toBeInTheDocument();
    expect(screen.getByTestId("script-step-next")).toBeDisabled();
  });
});
