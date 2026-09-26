import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { AppShell } from "../../src/components/AppShell";
import { ProjectDraftProvider } from "../../src/context/ProjectDraftContext";
import { ProjectFlowProvider } from "../../src/context/ProjectFlowContext";
import { ThemeProvider } from "../../src/context/ThemeContext";
import * as apiClient from "../../src/api/client";
import type { Project } from "../../src/types";

function renderRail(p: Project | null, currentStep = 3) {
  if (p) vi.spyOn(apiClient, "getProject").mockResolvedValue(p);
  else vi.spyOn(apiClient, "getProject").mockRejectedValue(new apiClient.ApiError("nf"));
  vi.spyOn(apiClient, "listProjectErrors").mockResolvedValue([]);
  return render(
    <ThemeProvider>
      <MemoryRouter initialEntries={["/projects/p1/render"]}>
        <ProjectDraftProvider>
          <ProjectFlowProvider>
            <Routes>
              <Route path="*" element={<AppShell currentStep={currentStep} title="T" subtitle="S"><span /></AppShell>} />
            </Routes>
          </ProjectFlowProvider>
        </ProjectDraftProvider>
      </MemoryRouter>
    </ThemeProvider>,
  );
}

const rendering = { project_id: "p1", status: "rendering", voice_language: "vi", scenes: [], flow_step: 9, run_state: "running", video_output_mode: "long" } as Project;

describe("StepRail — second-layer vertical menu", () => {
  afterEach(() => {
    vi.restoreAllMocks();
    window.localStorage.clear();
  });

  it("lists the 13 steps grouped in four phases, each with its own state", async () => {
    renderRail(rendering);
    await waitFor(() => expect(screen.getByTestId("step-rail")).toBeInTheDocument());

    for (let i = 1; i <= 13; i += 1) expect(screen.getByTestId(`rail-step-${i}`)).toBeInTheDocument();
    for (const phase of ["Soạn", "Kiểm tra", "Sản xuất", "Đầu ra"]) expect(screen.getByText(phase)).toBeInTheDocument();

    expect(screen.getByTestId("rail-step-3")).toHaveAttribute("data-status", "done");
    expect(screen.getByTestId("rail-step-9")).toHaveAttribute("data-status", "running");
    expect(screen.getByTestId("rail-step-10")).toHaveAttribute("data-status", "pending");
    // No vertical clips for this project: the split step is marked unused.
    expect(screen.getByTestId("rail-step-11")).toHaveAttribute("data-status", "skipped");
  });

  it("agrees with the horizontal bar about who can be opened", async () => {
    renderRail(rendering);
    await waitFor(() => expect(screen.getByTestId("step-rail")).toBeInTheDocument());
    for (const step of [2, 8, 9]) {
      expect(screen.getByTestId(`rail-step-${step}`)).not.toBeDisabled();
      expect(screen.getByTestId(`flow-step-${step}`)).not.toBeDisabled();
    }
    for (const step of [10, 13]) {
      expect(screen.getByTestId(`rail-step-${step}`)).toBeDisabled();
      expect(screen.getByTestId(`flow-step-${step}`)).toBeDisabled();
    }
  });

  it("marks a cancelled and a failed step differently", async () => {
    renderRail({ ...rendering, status: "failed_at_render_scenes", run_state: "cancelled" } as Project);
    await waitFor(() => expect(screen.getByTestId("rail-step-9")).toHaveAttribute("data-status", "cancelled"));
  });

  it("collapses to a strip and remembers it", async () => {
    renderRail(rendering);
    await waitFor(() => expect(screen.getByTestId("step-rail")).toHaveAttribute("data-collapsed", "false"));
    fireEvent.click(screen.getByTestId("step-rail-toggle"));
    expect(screen.getByTestId("step-rail")).toHaveAttribute("data-collapsed", "true");
    expect(window.localStorage.getItem("conceptflow.stepRail.collapsed")).toBe("1");
    // Labels go, the numbered marks stay and still say what they are on hover.
    expect(screen.getByTestId("rail-step-9")).toHaveAttribute("title", expect.stringContaining("Render"));
  });

  it("is absent before the server knows the project (a brand-new idea)", async () => {
    renderRail(null, 1);
    await waitFor(() => expect(screen.getByTestId("flow-step-1")).toBeInTheDocument());
    expect(screen.queryByTestId("step-rail")).not.toBeInTheDocument();
  });
});
