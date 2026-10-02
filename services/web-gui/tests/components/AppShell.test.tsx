import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { MemoryRouter, Routes, Route, useLocation } from "react-router-dom";
import { AppShell } from "../../src/components/AppShell";
import { WizardNav } from "../../src/components/WizardNav";
import { TextArea, TextInput, Select } from "../../src/components/ui";
import { ProjectDraftProvider } from "../../src/context/ProjectDraftContext";
import { ProjectFlowProvider } from "../../src/context/ProjectFlowContext";
import { ThemeProvider } from "../../src/context/ThemeContext";
import * as apiClient from "../../src/api/client";
import type { Project, ProjectSummary } from "../../src/types";
import { openAllPhases } from "../helpers/stepRail";

/** Where the router is, for the navigation assertions. */
function Where() {
  return <span data-testid="where">{useLocation().pathname}</span>;
}

function project(over: Partial<Project>): Project {
  return { project_id: "p1", status: "draft", voice_language: "vi", scenes: [], ...over } as Project;
}

function renderShell(p: Project | null, currentStep: number, recent: ProjectSummary[] = []) {
  window.localStorage.setItem("conceptflow.draft.v1", JSON.stringify({ projectId: "p1", voiceLanguage: "vi" }));
  if (p) vi.spyOn(apiClient, "getProject").mockResolvedValue(p);
  else vi.spyOn(apiClient, "getProject").mockRejectedValue(new apiClient.ApiError("not found"));
  vi.spyOn(apiClient, "listProjectErrors").mockResolvedValue([]);
  vi.spyOn(apiClient, "listProjects").mockResolvedValue(recent);
  return render(
    <ThemeProvider>
      <MemoryRouter initialEntries={["/create/script/outline"]}>
        <ProjectDraftProvider>
          <ProjectFlowProvider>
            <Routes>
              <Route
                path="*"
                element={
                  <>
                    <AppShell currentStep={currentStep} title="T" subtitle="S">
                      <input data-testid="an-input" />
                      <TextArea data-testid="story" defaultValue={"Dàn ý dòng 1\nDòng 2"} />
                      <TextInput data-testid="topic" defaultValue="thiên kiến sống sót" />
                      <Select data-testid="lang" defaultValue="vi">
                        <option value="vi">Tiếng Việt</option>
                        <option value="en">English</option>
                      </Select>
                    </AppShell>
                    <WizardNav hint="h" onNext={() => {}} nextLabel="Tiếp tục" nextTestId="next" />
                    <Where />
                  </>
                }
              />
            </Routes>
          </ProjectFlowProvider>
        </ProjectDraftProvider>
      </MemoryRouter>
    </ThemeProvider>,
  );
}

describe("AppShell — 13-step flow", () => {
  afterEach(() => {
    vi.restoreAllMocks();
    window.localStorage.clear();
  });

  it("shows all 13 steps, with the current one marked", async () => {
    renderShell(project({ status: "draft", flow_step: 3, run_state: "idle" }), 3);
    await openAllPhases();
    for (let i = 1; i <= 13; i += 1) expect(screen.getByTestId(`rail-step-${i}`)).toBeInTheDocument();
    expect(screen.queryByTestId("rail-step-14")).not.toBeInTheDocument();
    expect(screen.getByTestId("rail-step-3")).toHaveAttribute("aria-current", "step");
  });

  it("opens a step the project has reached, and only a preview of one it has not", async () => {
    renderShell(project({ status: "rendering", flow_step: 10, run_state: "running" }), 5);
    await openAllPhases();
    await waitFor(() => expect(screen.getByTestId("rail-step-10")).toHaveAttribute("data-reached", "true"));
    expect(screen.getByTestId("rail-step-9")).toHaveAttribute("data-reached", "true");
    expect(screen.getByTestId("rail-step-11")).toHaveAttribute("data-reached", "false");
    expect(screen.getByTestId("rail-step-13")).not.toBeDisabled();

    fireEvent.click(screen.getByTestId("rail-step-13"));
    expect(screen.getByTestId("where").textContent).toMatch(/^\/projects\/[^/]+\/preview\/13$/);
    fireEvent.click(screen.getByTestId("rail-step-10"));
    expect(screen.getByTestId("where").textContent).toMatch(/^\/projects\/[^/]+\/render$/);
  });

  it("shows the phase and step above the title", async () => {
    renderShell(project({ status: "draft", flow_step: 4, run_state: "idle" }), 4);
    expect(screen.getByTestId("phase-line")).toHaveTextContent("Soạn nội dung · Bước 4/13");
  });

  it("shows the step menu on step 1, before the server has a project", async () => {
    renderShell(null, 1);
    await waitFor(() => expect(screen.getByTestId("step-rail")).toBeInTheDocument());
    expect(screen.getByTestId("step-rail")).toHaveTextContent("Video mới");
    expect(screen.getByTestId("step-rail-meta")).toHaveTextContent("Nhập ý tưởng để bắt đầu");
    // Nothing runs yet, so there is no status strip.
    expect(screen.queryByTestId("status-strip")).not.toBeInTheDocument();
    await openAllPhases();
    fireEvent.click(screen.getByTestId("rail-step-4"));
    expect(screen.getByTestId("where")).toHaveTextContent("/create/preview/4");
  });

  it("names the phase and the step on the sidebar's resume link, not a bare step count", async () => {
    renderShell(null, 1, [
      { project_id: "p9", topic: "Cây nhị phân", status: "rendering", flow_step: 10, run_state: "running" } as ProjectSummary,
    ]);

    const link = await screen.findByTestId("resume-recent-project");
    expect(link).toHaveTextContent("Tiếp tục · đang chạy · Sản xuất · Dựng hình");
    expect(link).not.toHaveTextContent("/13");
  });

  it("locks the authoring screens read-only while the project is rendering, and says why", async () => {
    renderShell(project({ status: "rendering", flow_step: 10, run_state: "running" }), 3);

    await waitFor(() => expect(screen.getByTestId("read-only-banner")).toBeInTheDocument());
    expect(screen.getByTestId("read-only-banner")).toHaveTextContent("đang chạy");
    expect(screen.getByTestId("an-input")).toBeDisabled();
    // The bar that would save or start a render is locked too.
    expect(screen.getByTestId("next")).toBeDisabled();
    // Navigation stays usable: looking around is the point.
    await openAllPhases();
    expect(screen.getByTestId("rail-step-2")).not.toBeDisabled();
  });

  it("shows a locked project's fields as plain text you can read and copy, not greyed-out boxes", async () => {
    renderShell(project({ status: "ready_to_publish", flow_step: 12, run_state: "idle" }), 3);

    await waitFor(() => expect(screen.getByTestId("read-only-banner")).toBeInTheDocument());
    const story = screen.getByTestId("story");
    expect(story.tagName).toBe("DIV");
    expect(story).toHaveAttribute("data-readonly", "true");
    expect(story).toHaveTextContent("Dàn ý dòng 1");
    expect(screen.getByTestId("topic")).toHaveTextContent("thiên kiến sống sót");
    // A select shows the chosen option's label, not its value.
    expect(screen.getByTestId("lang")).toHaveTextContent("Tiếng Việt");
    expect(document.querySelector("textarea")).toBeNull();
  });

  it("keeps real inputs for a draft", async () => {
    renderShell(project({ status: "draft", flow_step: 3, run_state: "idle" }), 3);
    await openAllPhases();
    await waitFor(() => expect(screen.getByTestId("rail-step-2")).toHaveAttribute("data-reached", "true"));
    expect(screen.getByTestId("story").tagName).toBe("TEXTAREA");
    expect(screen.getByTestId("lang").tagName).toBe("SELECT");
  });

  it("leaves a draft fully editable", async () => {
    renderShell(project({ status: "draft", flow_step: 3, run_state: "idle" }), 3);
    await openAllPhases();
    await waitFor(() => expect(screen.getByTestId("rail-step-2")).toHaveAttribute("data-reached", "true"));
    expect(screen.queryByTestId("read-only-banner")).not.toBeInTheDocument();
    expect(screen.getByTestId("an-input")).not.toBeDisabled();
    expect(screen.getByTestId("next")).not.toBeDisabled();
  });

  it("keeps a project that failed at render editable, so the Creator can go fix an earlier step", async () => {
    renderShell(project({ status: "failed_at_render_scenes", flow_step: 10, run_state: "failed" }), 5);
    await openAllPhases();
    await waitFor(() => expect(screen.getByTestId("rail-step-10")).toHaveAttribute("data-reached", "true"));
    expect(screen.queryByTestId("read-only-banner")).not.toBeInTheDocument();
    expect(screen.getByTestId("an-input")).not.toBeDisabled();
  });

  it("opens a past step of a locked project through resume (no API write on the way)", async () => {
    const put = vi.spyOn(globalThis, "fetch");
    renderShell(project({ status: "ready_to_publish", flow_step: 12, run_state: "idle" }), 12);
    await openAllPhases();
    await waitFor(() => expect(screen.getByTestId("rail-step-3")).toHaveAttribute("data-reached", "true"));
    put.mockClear();
    fireEvent.click(screen.getByTestId("rail-step-3"));
    // Navigating is a client-side route change; nothing is sent to the server.
    expect(put).not.toHaveBeenCalled();
  });
});
