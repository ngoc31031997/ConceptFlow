import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen, waitFor, fireEvent, within } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { AppShell } from "../../src/components/AppShell";
import { ProjectDraftProvider } from "../../src/context/ProjectDraftContext";
import { ProjectFlowProvider } from "../../src/context/ProjectFlowContext";
import { ThemeProvider } from "../../src/context/ThemeContext";
import * as apiClient from "../../src/api/client";
import type { Project } from "../../src/types";
import { openAllPhases } from "../helpers/stepRail";

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

const rendering = { project_id: "p1", status: "rendering", voice_language: "vi", scenes: [], flow_step: 10, run_state: "running", video_output_mode: "long" } as Project;

describe("StepRail — second-layer vertical menu", () => {
  afterEach(() => {
    vi.restoreAllMocks();
    window.localStorage.clear();
  });

  it("lists the 13 steps grouped in five phases, each with its own state", async () => {
    renderRail(rendering);
    await openAllPhases();

    for (let i = 1; i <= 13; i += 1) expect(screen.getByTestId(`rail-step-${i}`)).toBeInTheDocument();
    expect(screen.queryByTestId("rail-step-14")).not.toBeInTheDocument();
    const rail = within(screen.getByTestId("step-rail"));
    for (const phase of ["Chuẩn bị", "Soạn nội dung", "Duyệt", "Sản xuất", "Hoàn tất"]) expect(rail.getByText(phase)).toBeInTheDocument();

    expect(screen.getByTestId("rail-step-3")).toHaveAttribute("data-status", "done");
    expect(screen.getByTestId("rail-step-10")).toHaveAttribute("data-status", "running");
    expect(screen.getByTestId("rail-step-11")).toHaveAttribute("data-status", "pending");
    expect(screen.getByTestId("rail-step-12")).toHaveAttribute("data-status", "pending");
  });

  it("opens only the phase of the step on screen; the others open and close on click", async () => {
    renderRail(rendering, 10);
    await waitFor(() => expect(screen.getByTestId("step-rail")).toBeInTheDocument());

    expect(screen.getByTestId("rail-phase-4")).toHaveAttribute("data-open", "true");
    expect(screen.getByTestId("rail-step-10")).toBeInTheDocument();
    expect(screen.getByTestId("rail-phase-2")).toHaveAttribute("data-open", "false");
    expect(screen.queryByTestId("rail-step-4")).not.toBeInTheDocument();

    fireEvent.click(screen.getByTestId("rail-phase-toggle-2"));
    expect(screen.getByTestId("rail-step-4")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("rail-phase-toggle-4"));
    expect(screen.getByTestId("rail-phase-4")).toHaveAttribute("data-open", "false");
  });

  it("counts the finished steps of each phase and of the video, leaving unused steps out", async () => {
    renderRail(rendering, 10);
    await waitFor(() => expect(screen.getByTestId("rail-phase-count-4")).toHaveTextContent("1/3"));
    expect(screen.getByTestId("rail-phase-count-1")).toHaveTextContent("2/2");
    expect(screen.getByTestId("rail-phase-count-3")).toHaveTextContent("2/2");
    expect(screen.getByTestId("rail-phase-count-5")).toHaveTextContent("0/2");
    expect(screen.getByTestId("step-rail-meta")).toHaveTextContent("Sản xuất · Dựng hình");
  });

  it("tags the steps a worker runs on its own as automatic", async () => {
    renderRail(rendering);
    await openAllPhases();
    for (const step of [7, 9, 10, 11]) {
      expect(screen.getByTestId(`rail-auto-${step}`)).toHaveTextContent("Tự động");
    }
    for (const step of [1, 2, 3, 4, 5, 6, 8, 12, 13]) {
      expect(screen.queryByTestId(`rail-auto-${step}`)).not.toBeInTheDocument();
    }
  });

  it("opens every step: the real screen of a reached one, a preview of the rest", async () => {
    renderRail(rendering);
    await openAllPhases();
    for (const step of [2, 9, 10]) {
      expect(screen.getByTestId(`rail-step-${step}`)).toHaveAttribute("data-reached", "true");
    }
    for (const step of [11, 12, 13]) {
      expect(screen.getByTestId(`rail-step-${step}`)).toHaveAttribute("data-reached", "false");
      expect(screen.getByTestId(`rail-step-${step}`)).not.toBeDisabled();
    }
  });

  it("marks a cancelled and a failed step differently", async () => {
    renderRail({ ...rendering, status: "failed_at_render_scenes", run_state: "cancelled" } as Project, 10);
    await waitFor(() => expect(screen.getByTestId("rail-step-10")).toHaveAttribute("data-status", "cancelled"));
  });

  it("collapses to a strip and remembers it", async () => {
    renderRail(rendering);
    await waitFor(() => expect(screen.getByTestId("step-rail")).toHaveAttribute("data-collapsed", "false"));
    fireEvent.click(screen.getByTestId("step-rail-toggle"));
    expect(screen.getByTestId("step-rail")).toHaveAttribute("data-collapsed", "true");
    expect(window.localStorage.getItem("conceptflow.stepRail.collapsed")).toBe("1");
    // Labels and phase headers go; every step's mark stays and still says what it is on hover.
    await waitFor(() =>
      expect(screen.getByTestId("rail-step-10")).toHaveAttribute("title", expect.stringContaining("Dựng hình (tự động)")),
    );
    expect(screen.queryByTestId("rail-phase-toggle-1")).not.toBeInTheDocument();
  });

  it("is there on step 1, before the server knows the project (a brand-new idea)", async () => {
    renderRail(null, 1);
    await waitFor(() => expect(screen.getByTestId("step-rail")).toBeInTheDocument());
    expect(screen.getByTestId("rail-step-1")).toHaveAttribute("aria-current", "step");
    // The old horizontal step bar is gone for good.
    expect(screen.queryByTestId("flow-step-1")).not.toBeInTheDocument();
  });

  it("stays fixed while the page scrolls: the rail is a fixed column", () => {
    // jsdom does not lay out, so check the CSS rule the layout relies on.
    const css = readFileSync(resolve(__dirname, "../../src/components/StepRail.module.css"), "utf8");
    expect(css).toMatch(/\.rail \{[^}]*position: fixed;/);
    const shell = readFileSync(resolve(__dirname, "../../src/components/AppShell.module.css"), "utf8");
    expect(shell).toMatch(/\.sidebar \{[^}]*position: fixed;/);
  });
});
