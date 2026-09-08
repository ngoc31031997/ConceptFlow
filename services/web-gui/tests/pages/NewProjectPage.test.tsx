import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { NewProjectPage } from "../../src/pages/NewProjectPage";
import { ProjectDraftProvider } from "../../src/context/ProjectDraftContext";

describe("NewProjectPage", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("disables submit until a script is entered", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ connected: false }),
    }) as unknown as typeof fetch;

    render(
      <MemoryRouter>
        <ProjectDraftProvider>
          <NewProjectPage />
        </ProjectDraftProvider>
      </MemoryRouter>,
    );

    expect(screen.getByTestId("new-project-submit-button")).toBeDisabled();

    fireEvent.change(screen.getByTestId("new-project-script-textarea"), {
      target: { value: 'class DemoScene(Scene):\n    def construct(self):\n        # NARRATION: "hi"\n        self.wait(AUTO)' },
    });

    expect(screen.getByTestId("new-project-submit-button")).not.toBeDisabled();
  });
});

describe("NewProjectPage draft lifecycle", () => {
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
          <NewProjectPage />
        </ProjectDraftProvider>
      </MemoryRouter>,
    );

    expect(screen.getByTestId("new-project-script-textarea")).toHaveValue("");
    expect(screen.getByTestId("new-project-submit-button")).toBeDisabled();
  });
});
