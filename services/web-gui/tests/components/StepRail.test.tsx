import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen, waitFor, fireEvent, within } from "@testing-library/react";
import { MemoryRouter, Routes, Route, useLocation } from "react-router-dom";
import { AppShell } from "../../src/components/AppShell";
import { ProjectDraftProvider } from "../../src/context/ProjectDraftContext";
import { ProjectFlowProvider } from "../../src/context/ProjectFlowContext";
import { ThemeProvider } from "../../src/context/ThemeContext";
import * as apiClient from "../../src/api/client";
import type { Project } from "../../src/types";
import { waitForStepRail } from "../helpers/stepRail";

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

function Where() {
  return <span data-testid="where">{useLocation().pathname}</span>;
}

/** The rail on a project's screen, with the screens it can open reduced to their address. */
function renderRailWithRoutes(p: Project, currentStep = 3) {
  vi.spyOn(apiClient, "getProject").mockResolvedValue(p);
  vi.spyOn(apiClient, "listProjectErrors").mockResolvedValue([]);
  return render(
    <ThemeProvider>
      <MemoryRouter initialEntries={["/projects/p1/render"]}>
        <ProjectDraftProvider>
          <ProjectFlowProvider>
            <Routes>
              <Route path="/projects/p1/render" element={<AppShell currentStep={currentStep} title="T" subtitle="S"><span /></AppShell>} />
              <Route path="*" element={<Where />} />
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

  it("lists the 21 steps of the four phases, each with its own state", async () => {
    renderRail(rendering);
    // Running steps 9 (voice) and 10 (render) both show as step 17.
    await waitFor(() => expect(screen.getByTestId("rail-step-17")).toHaveAttribute("data-status", "running"));

    for (let i = 1; i <= 21; i += 1) expect(screen.getByTestId(`rail-step-${i}`)).toBeInTheDocument();
    expect(screen.queryByTestId("rail-step-22")).not.toBeInTheDocument();
    const rail = within(screen.getByTestId("step-rail"));
    for (const phase of ["Tiền kỳ · Nội dung", "Tiền kỳ · Hình ảnh", "Sản xuất", "Hậu kỳ"]) expect(rail.getByText(phase)).toBeInTheDocument();

    expect(screen.getByTestId("rail-step-6")).toHaveAttribute("data-status", "done");
    expect(screen.getByTestId("rail-step-19")).toHaveAttribute("data-status", "pending");
    expect(screen.getByTestId("rail-step-20")).toHaveAttribute("data-status", "pending");
  });

  it("shows the steps the app does not have yet as “sắp có”, and opens a page saying so", async () => {
    renderRail(rendering);
    await waitForStepRail();
    for (const step of [8, 10, 11, 12, 18]) {
      const item = screen.getByTestId(`rail-step-${step}`);
      expect(item).toHaveAttribute("data-status", "coming");
      expect(item).toHaveTextContent("sắp có");
      expect(item).not.toBeDisabled();
    }
  });

  it("opens the not-developed page of a step the app does not have", async () => {
    renderRailWithRoutes(rendering);
    await waitFor(() => expect(screen.getByTestId("rail-step-17")).toHaveAttribute("data-status", "running"));

    fireEvent.click(screen.getByTestId("rail-step-8"));
    expect(screen.getByTestId("where")).toHaveTextContent("/projects/p1/coming/8");
  });

  it("opens the research screen from the step menu and shows where the project stands on it", async () => {
    renderRailWithRoutes({ ...rendering, status: "draft", flow_step: 3, run_state: "idle" } as Project);
    await waitFor(() => expect(screen.getByTestId("rail-step-2")).toHaveAttribute("data-status", "done"));
    const research = screen.getByTestId("rail-step-3");
    expect(research).toHaveAttribute("data-status", "waiting");
    expect(research).toHaveAttribute("data-reached", "true");
    expect(research).toHaveTextContent("tuỳ chọn");
    expect(research).not.toBeDisabled();
    fireEvent.click(research);
    expect(screen.getByTestId("where")).toHaveTextContent("/projects/p1/research");
  });

  it("calls the research step passed once the project has moved on to the script", async () => {
    renderRail(rendering);
    await waitFor(() => expect(screen.getByTestId("rail-step-3")).toHaveAttribute("data-status", "done"));
  });

  it("keeps the research step closed until there is a project", async () => {
    renderRail(null, 1);
    await waitFor(() => expect(screen.getByTestId("step-rail")).toBeInTheDocument());
    expect(screen.getByTestId("rail-step-3")).toHaveAttribute("data-status", "pending");
    expect(screen.getByTestId("rail-step-3")).toBeDisabled();
  });

  it("marks the research screen as the step being viewed, on a screen with no running step", async () => {
    vi.spyOn(apiClient, "getProject").mockResolvedValue({ ...rendering, status: "draft", flow_step: 3, run_state: "idle" } as Project);
    vi.spyOn(apiClient, "listProjectErrors").mockResolvedValue([]);
    render(
      <ThemeProvider>
        <MemoryRouter initialEntries={["/projects/p1/research"]}>
          <ProjectDraftProvider>
            <ProjectFlowProvider>
              <Routes>
                <Route path="/projects/:id/research" element={<AppShell viewStep={3} title="Nghiên cứu và kiểm chứng" subtitle="S"><span /></AppShell>} />
              </Routes>
            </ProjectFlowProvider>
          </ProjectDraftProvider>
        </MemoryRouter>
      </ThemeProvider>,
    );
    await waitFor(() => expect(screen.getByTestId("rail-step-3")).toHaveAttribute("aria-current", "step"));
    expect(screen.getByTestId("phase-line")).toHaveTextContent("Tiền kỳ · Nội dung · Bước 3/21");
    // A step the project has already passed can be opened from here.
    await waitFor(() => expect(screen.getByTestId("rail-step-2")).not.toBeDisabled());
    expect(screen.getByTestId("rail-step-3")).toBeDisabled();
  });

  it("counts the finished steps of each phase and of the video", async () => {
    renderRail(rendering, 10);
    await waitFor(() => expect(screen.getByTestId("rail-phase-count-3")).toHaveTextContent("4/4"));
    expect(screen.getByTestId("rail-phase-count-1")).toHaveTextContent("7/7");
    expect(screen.getByTestId("rail-phase-count-2")).toHaveTextContent("1/5");
    expect(screen.getByTestId("rail-phase-count-4")).toHaveTextContent("0/5");
    expect(screen.getByTestId("step-rail-meta")).toHaveTextContent("12/21 bước xong");
  });

  it("marks the built steps where the Creator approves with ◆", async () => {
    renderRail(rendering);
    await waitForStepRail();
    for (const step of [1, 2, 3, 4, 5, 6, 7, 9, 16]) expect(screen.getByTestId(`rail-gate-${step}`)).toHaveTextContent("◆");
    for (const step of [13, 14, 15, 17, 19, 20, 21]) {
      expect(screen.queryByTestId(`rail-gate-${step}`)).not.toBeInTheDocument();
    }
  });

  it("opens a built step only once the project has reached it", async () => {
    renderRail(rendering);
    await waitFor(() => expect(screen.getByTestId("rail-step-17")).toHaveAttribute("data-reached", "true"));
    for (const step of [2, 16, 17]) {
      expect(screen.getByTestId(`rail-step-${step}`)).toHaveAttribute("data-reached", "true");
    }
    for (const step of [19, 20, 21]) {
      expect(screen.getByTestId(`rail-step-${step}`)).toHaveAttribute("data-reached", "false");
      expect(screen.getByTestId(`rail-step-${step}`)).toBeDisabled();
    }
  });

  it("keeps the config step closed while the project is still at the idea step", async () => {
    renderRail({ ...rendering, status: "draft", flow_step: 1, run_state: "idle" } as Project, 1);
    await waitFor(() => expect(screen.getByTestId("rail-step-1")).toHaveAttribute("aria-current", "step"));
    for (const step of [2, 3, 4, 6]) expect(screen.getByTestId(`rail-step-${step}`)).toBeDisabled();
  });

  it("keeps the research step closed until the config is confirmed", async () => {
    renderRail({ ...rendering, status: "draft", flow_step: 2, run_state: "idle" } as Project, 2);
    await waitFor(() => expect(screen.getByTestId("rail-step-2")).toHaveAttribute("aria-current", "step"));
    expect(screen.getByTestId("rail-step-3")).toBeDisabled();
    expect(screen.getByTestId("rail-step-4")).toBeDisabled();
  });

  it("keeps the story step closed until research is approved or skipped", async () => {
    const atConfig = { ...rendering, status: "draft", flow_step: 3, run_state: "idle" } as Project;
    const research = vi.spyOn(apiClient, "getResearch");
    research.mockResolvedValue({ gate: { status: "idle", skipped: false, resolved: false } } as unknown as apiClient.ResearchState);
    const { unmount } = renderRail(atConfig, 2);
    await waitFor(() => expect(screen.getByTestId("rail-step-2")).toHaveAttribute("data-status", "done"));
    expect(screen.getByTestId("rail-step-3")).not.toBeDisabled();
    expect(screen.getByTestId("rail-step-4")).toBeDisabled();
    unmount();

    research.mockResolvedValue({ gate: { status: "idle", skipped: true, resolved: true } } as unknown as apiClient.ResearchState);
    renderRail(atConfig, 2);
    await waitFor(() => expect(screen.getByTestId("rail-step-4")).not.toBeDisabled());
  });

  it("marks a cancelled and a failed step differently", async () => {
    renderRail({ ...rendering, status: "failed_at_render_scenes", run_state: "cancelled" } as Project, 10);
    await waitFor(() => expect(screen.getByTestId("rail-step-17")).toHaveAttribute("data-status", "cancelled"));
  });

  it("collapses to a strip and remembers it", async () => {
    renderRail(rendering);
    await waitFor(() => expect(screen.getByTestId("step-rail")).toHaveAttribute("data-collapsed", "false"));
    fireEvent.click(screen.getByTestId("step-rail-toggle"));
    expect(screen.getByTestId("step-rail")).toHaveAttribute("data-collapsed", "true");
    expect(window.localStorage.getItem("conceptflow.stepRail.collapsed")).toBe("1");
    // Labels and phase headers go; every step's mark stays and still says what it is on hover.
    await waitFor(() =>
      expect(screen.getByTestId("rail-step-17")).toHaveAttribute(
        "title",
        expect.stringContaining("17. Render từng cảnh · Đang chạy · tự động"),
      ),
    );
    expect(within(screen.getByTestId("step-rail")).queryByText("Hậu kỳ")).not.toBeInTheDocument();
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
