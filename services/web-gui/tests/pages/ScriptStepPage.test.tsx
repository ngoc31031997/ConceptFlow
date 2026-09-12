import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { ScriptStepPage } from "../../src/pages/ScriptStepPage";
import { ProjectDraftProvider } from "../../src/context/ProjectDraftContext";
import * as apiClient from "../../src/api/client";

// CR-025: the default "blank" source now fetches its prompt from the DB and
// asks for a pasted story outline instead of Manim code — stub the fetch so
// these tests don't need a live backend. Re-armed in beforeEach because
// afterEach below calls restoreAllMocks().
beforeEach(() => {
  vi.spyOn(apiClient, "getPromptTemplate").mockResolvedValue({
    role: "story_architect",
    language: "vi",
    version: 1,
    template_text: "CHỦ ĐỀ VIDEO: {{topic}}\n{{format_beats}}\n{{narration_language_rule}}",
  });
});

describe("ScriptStepPage", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("blocks the step until a story outline is pasted (source: blank)", () => {
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

    fireEvent.change(screen.getByTestId("script-assistant-story-outline"), {
      target: { value: "CÂU HỎI CỐT LÕI: ...\nBEAT 1 — ..." },
    });

    expect(screen.getByTestId("script-step-next")).not.toBeDisabled();
  });

  it("blocks the step until the script is valid (source: draft)", () => {
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

    expect(screen.getByTestId("script-assistant-story-outline")).toHaveValue("");
    expect(screen.getByTestId("script-step-next")).toBeDisabled();
  });
});
