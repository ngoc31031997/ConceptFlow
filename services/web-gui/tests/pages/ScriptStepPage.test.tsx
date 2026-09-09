import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { ScriptStepPage } from "../../src/pages/ScriptStepPage";
import { ProjectDraftProvider } from "../../src/context/ProjectDraftContext";

describe("ScriptStepPage", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("blocks the step until the script is valid", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ connected: false }),
    }) as unknown as typeof fetch;

    render(
      <MemoryRouter>
        <ProjectDraftProvider>
          <ScriptStepPage />
        </ProjectDraftProvider>
      </MemoryRouter>,
    );

    expect(screen.getByTestId("script-step-next")).toBeDisabled();

    fireEvent.change(screen.getByTestId("new-project-script-textarea"), {
      target: {
        value:
          'from conceptflow import *\n\nclass DemoScene(ConceptFlowScene):\n    def construct(self):\n        self.narrate("xin chào")',
      },
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
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ connected: false }),
    }) as unknown as typeof fetch;

    window.localStorage.setItem(
      "conceptflow.draft.v1",
      JSON.stringify({ projectId: "spent-project", scriptContent: "old script", hasSubmitted: true }),
    );

    render(
      <MemoryRouter>
        <ProjectDraftProvider>
          <ScriptStepPage />
        </ProjectDraftProvider>
      </MemoryRouter>,
    );

    expect(screen.getByTestId("new-project-script-textarea")).toHaveValue("");
    expect(screen.getByTestId("script-step-next")).toBeDisabled();
  });
});
