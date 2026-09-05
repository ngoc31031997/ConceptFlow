import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { NewProjectPage } from "../../src/pages/NewProjectPage";
import { ProjectDraftProvider } from "../../src/context/ProjectDraftContext";

describe("NewProjectPage", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("disables submit until script and plugin are set", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ plugins: [{ plugin_id: "coding", name: "Lập trình" }] }),
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
      target: { value: "# Scene 1\n```\nprint(1)\n```" },
    });

    await waitFor(() =>
      expect(screen.getByRole("option", { name: "Lập trình" })).toBeInTheDocument(),
    );
    fireEvent.change(screen.getByTestId("new-project-plugin-select"), {
      target: { value: "coding" },
    });

    expect(screen.getByTestId("new-project-submit-button")).not.toBeDisabled();
  });
});
