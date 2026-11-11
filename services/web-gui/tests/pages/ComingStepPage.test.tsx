import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter, Routes, Route, useLocation } from "react-router-dom";
import { ComingStepPage } from "../../src/pages/ComingStepPage";
import { ProjectDraftProvider } from "../../src/context/ProjectDraftContext";
import { ProjectFlowProvider } from "../../src/context/ProjectFlowContext";
import { ThemeProvider } from "../../src/context/ThemeContext";
import * as apiClient from "../../src/api/client";
import type { Project } from "../../src/types";

function Where() {
  return <span data-testid="where">{useLocation().pathname}</span>;
}

function renderAt(path: string, project: Project | null = null) {
  if (project) vi.spyOn(apiClient, "getProject").mockResolvedValue(project);
  else vi.spyOn(apiClient, "getProject").mockRejectedValue(new apiClient.ApiError("nf"));
  vi.spyOn(apiClient, "listProjectErrors").mockResolvedValue([]);
  vi.spyOn(apiClient, "listProjects").mockResolvedValue([]);
  return render(
    <ThemeProvider>
      <MemoryRouter initialEntries={[path]}>
        <ProjectDraftProvider>
          <ProjectFlowProvider>
            <Routes>
              <Route path="/projects/:id/coming/:n" element={<ComingStepPage />} />
              <Route path="/create/coming/:n" element={<ComingStepPage />} />
              <Route path="*" element={<Where />} />
            </Routes>
          </ProjectFlowProvider>
        </ProjectDraftProvider>
      </MemoryRouter>
    </ThemeProvider>,
  );
}

const draft = { project_id: "p1", status: "draft", voice_language: "vi", scenes: [], flow_step: 2, run_state: "idle", video_output_mode: "long" } as Project;

describe("ComingStepPage", () => {
  afterEach(() => vi.restoreAllMocks());

  it("says the step is not developed, what it is for, and where it sits in the flow", async () => {
    renderAt("/projects/p1/coming/9", draft);
    expect(await screen.findByTestId("coming-step-page")).toBeInTheDocument();
    expect(screen.getByTestId("coming-step-notice")).toHaveTextContent("Bước này chưa phát triển");
    expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("Phương án hình ảnh");
    expect(screen.getByTestId("step-role")).toHaveTextContent(/Cách minh hoạ từng beat theo ngôn ngữ hình ảnh đã chốt/);
    await waitFor(() => expect(screen.getByTestId("phase-line")).toHaveTextContent("Tiền kỳ · Hình ảnh · Bước 9/23"));
    expect(screen.getByTestId("rail-step-9")).toHaveAttribute("aria-current", "step");
  });

  it("names the step before it", async () => {
    renderAt("/projects/p1/coming/9", draft);
    expect(await screen.findByTestId("coming-step-back")).toHaveTextContent("Bước trước: Ngôn ngữ hình ảnh");
  });

  it("offers another not-developed step as the step before, when that is what precedes it", async () => {
    renderAt("/projects/p1/coming/10", draft);
    expect(await screen.findByRole("heading", { level: 1 })).toHaveTextContent("Diễn tiến hình ảnh");
    fireEvent.click(screen.getByTestId("coming-step-back"));
    await waitFor(() => expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("Phương án hình ảnh"));
    expect(screen.getByTestId("phase-line")).toHaveTextContent("Bước 9/23");
  });

  it("works before a video exists", async () => {
    renderAt("/create/coming/9");
    expect(await screen.findByTestId("coming-step-notice")).toBeInTheDocument();
    expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("Phương án hình ảnh");
  });

  it("sends a number outside the flow home", async () => {
    renderAt("/create/coming/99");
    await waitFor(() => expect(screen.getByTestId("where")).toHaveTextContent("/"));
    expect(screen.queryByTestId("coming-step-page")).not.toBeInTheDocument();
  });

  it("opens the real screen of a step the app has", async () => {
    renderAt("/projects/p1/coming/3", draft);
    await waitFor(() => expect(screen.getByTestId("where")).toHaveTextContent("/projects/p1/research"));
    renderAt("/projects/p1/coming/7", draft);
    await waitFor(() => expect(screen.getAllByTestId("where").some((el) => el.textContent === "/projects/p1/lock")).toBe(true));
    renderAt("/projects/p1/coming/4", draft);
    await waitFor(() => expect(screen.getAllByTestId("where").some((el) => el.textContent === "/projects/p1/story")).toBe(true));
    renderAt("/projects/p1/coming/8", draft);
    await waitFor(() => expect(screen.getAllByTestId("where").some((el) => el.textContent === "/projects/p1/visual-concept")).toBe(true));
  });

  it("says a step whose screen was taken out is not developed, even though a running step stands behind it", async () => {
    for (const n of [11, 15, 16]) {
      renderAt(`/projects/p1/coming/${n}`, draft);
      await waitFor(() => expect(screen.getAllByTestId("coming-step-notice").length).toBeGreaterThan(0));
    }
  });
});
