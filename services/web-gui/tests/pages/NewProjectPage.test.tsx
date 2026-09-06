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
