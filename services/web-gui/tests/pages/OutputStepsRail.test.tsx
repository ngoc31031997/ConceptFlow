import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { ResultPage } from "../../src/pages/ResultPage";
import { PublishPage } from "../../src/pages/PublishPage";
import { ProjectDraftProvider } from "../../src/context/ProjectDraftContext";
import { ProjectFlowProvider } from "../../src/context/ProjectFlowContext";
import { ThemeProvider } from "../../src/context/ThemeContext";
import * as apiClient from "../../src/api/client";
import type { Project } from "../../src/types";

function renderAt(path: string, status: string, flowStep: number) {
  const project = {
    project_id: "p1", status, voice_language: "vi", scenes: [], flow_step: flowStep, run_state: "idle",
  } as unknown as Project;
  window.localStorage.setItem("conceptflow.draft.v1", JSON.stringify({ projectId: "p1", voiceLanguage: "vi" }));
  vi.spyOn(apiClient, "getProject").mockResolvedValue(project);
  vi.spyOn(apiClient, "listProjectErrors").mockResolvedValue([]);
  global.fetch = vi.fn().mockResolvedValue({ ok: true, status: 200, json: async () => project }) as unknown as typeof fetch;
  return render(
    <ThemeProvider>
      <MemoryRouter initialEntries={[path]}>
        <ProjectDraftProvider>
          <ProjectFlowProvider>
            <Routes>
              <Route path="/projects/:id/result" element={<ResultPage />} />
              <Route path="/projects/:id/publish" element={<PublishPage />} />
            </Routes>
          </ProjectFlowProvider>
        </ProjectDraftProvider>
      </MemoryRouter>
    </ThemeProvider>,
  );
}

// CR-051 review: Result and Publish still told the rail they were steps 12 and
// 13 (the 13-step flow), so it highlighted "Cắt short" under "Bước 13 — Kết quả".
describe("the rail highlights the step the title names on the output screens", () => {
  afterEach(() => {
    vi.restoreAllMocks();
    window.localStorage.clear();
  });

  it("Kết quả is step 13", async () => {
    renderAt("/projects/p1/result", "ready_to_publish", 13);
    expect(await screen.findByTestId("rail-step-13")).toHaveAttribute("aria-current", "step");
    expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("Bước 13 — Kết quả");
  });

  it("Publish is step 14", async () => {
    renderAt("/projects/p1/publish", "published", 14);
    expect(await screen.findByTestId("rail-step-14")).toHaveAttribute("aria-current", "step");
    expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("Bước 14 — Publish");
  });
});
