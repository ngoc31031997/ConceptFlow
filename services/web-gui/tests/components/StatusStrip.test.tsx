import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { MemoryRouter, Routes, Route, useLocation } from "react-router-dom";
import { AppShell } from "../../src/components/AppShell";
import { ProjectDraftProvider } from "../../src/context/ProjectDraftContext";
import { ProjectFlowProvider } from "../../src/context/ProjectFlowContext";
import { ThemeProvider } from "../../src/context/ThemeContext";
import * as apiClient from "../../src/api/client";
import type { Project } from "../../src/types";

function project(over: Partial<Project>): Project {
  return { project_id: "p1", status: "draft", voice_language: "vi", scenes: [], ...over } as Project;
}

function Where() {
  return <div data-testid="where">{useLocation().pathname + useLocation().search}</div>;
}

function renderStrip(p: Project, currentStep: number) {
  vi.spyOn(apiClient, "getProject").mockResolvedValue(p);
  vi.spyOn(apiClient, "listProjectErrors").mockResolvedValue([]);
  return render(
    <ThemeProvider>
      <MemoryRouter initialEntries={["/projects/p1/render"]}>
        <ProjectDraftProvider>
          <ProjectFlowProvider>
            <Routes>
              <Route
                path="*"
                element={
                  <>
                    <AppShell currentStep={currentStep} title="T" subtitle="S">
                      <span />
                    </AppShell>
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

describe("StatusStrip", () => {
  afterEach(() => {
    vi.restoreAllMocks();
    window.localStorage.clear();
  });

  it("says a step is running even while the Creator looks at an earlier one, and offers the way back", async () => {
    renderStrip(project({ status: "rendering", flow_step: 9, run_state: "running" }), 3);

    await waitFor(() => expect(screen.getByTestId("strip-pill")).toHaveTextContent("Đang chạy"));
    expect(screen.getByTestId("status-strip")).toHaveTextContent("Render");
    fireEvent.click(screen.getByTestId("strip-goto"));
    expect(screen.getByTestId("where")).toHaveTextContent("/projects/p1/render");
  });

  it("cancels only after the Creator confirms, and says what is kept", async () => {
    const cancel = vi.spyOn(apiClient, "cancelProject").mockResolvedValue({ step: "render_scenes", status: "failed_at_render_scenes" });
    renderStrip(project({ status: "rendering", flow_step: 9, run_state: "running" }), 9);

    await waitFor(() => expect(screen.getByTestId("strip-cancel")).toBeInTheDocument());
    fireEvent.click(screen.getByTestId("strip-cancel"));
    expect(cancel).not.toHaveBeenCalled();
    expect(screen.getByTestId("confirm-modal")).toHaveTextContent("lưu đệm");

    fireEvent.click(screen.getByTestId("confirm-modal-confirm"));
    await waitFor(() => expect(cancel).toHaveBeenCalledWith("p1"));
  });

  it("does not cancel when the Creator backs out", async () => {
    const cancel = vi.spyOn(apiClient, "cancelProject").mockResolvedValue({ step: "", status: "" });
    renderStrip(project({ status: "rendering", flow_step: 9, run_state: "running" }), 9);

    await waitFor(() => expect(screen.getByTestId("strip-cancel")).toBeInTheDocument());
    fireEvent.click(screen.getByTestId("strip-cancel"));
    fireEvent.click(screen.getByTestId("confirm-modal-cancel"));
    expect(cancel).not.toHaveBeenCalled();
  });

  it("offers no cancel for a step with no worker to stop (cắt short, publish)", async () => {
    renderStrip(project({ status: "generating_clips", flow_step: 11, run_state: "running" }), 11);
    await waitFor(() => expect(screen.getByTestId("strip-pill")).toHaveTextContent("Đang chạy"));
    expect(screen.queryByTestId("strip-cancel")).not.toBeInTheDocument();
  });

  it("a cancelled step is shown as cancelled, not failed, and can be resumed", async () => {
    const retry = vi.spyOn(apiClient, "retryProject").mockResolvedValue({ saga_id: "s", status: "rendering" } as never);
    renderStrip(project({ status: "failed_at_render_scenes", flow_step: 9, run_state: "cancelled" }), 9);

    await waitFor(() => expect(screen.getByTestId("strip-pill")).toHaveTextContent("Đã huỷ"));
    fireEvent.click(screen.getByTestId("strip-resume"));
    await waitFor(() => expect(retry).toHaveBeenCalledWith("p1"));
  });

  it("a failed step points to the error detail instead of duplicating the retry button", async () => {
    renderStrip(project({ status: "failed_at_merge", flow_step: 10, run_state: "failed" }), 3);

    await waitFor(() => expect(screen.getByTestId("strip-pill")).toHaveTextContent("Lỗi"));
    expect(screen.queryByTestId("strip-resume")).not.toBeInTheDocument();
    expect(screen.getByTestId("strip-goto")).toBeInTheDocument();
  });

  it("forks from the chosen step and opens the new project there", async () => {
    const fork = vi.spyOn(apiClient, "forkProject").mockResolvedValue({ project_id: "new-1", from_step: 4, needs_music_reselect: false });
    renderStrip(project({ status: "ready_to_publish", flow_step: 12, run_state: "idle" }), 12);

    await waitFor(() => expect(screen.getByTestId("strip-fork")).toBeInTheDocument());
    fireEvent.click(screen.getByTestId("strip-fork"));
    expect(screen.getByTestId("fork-dialog")).toBeInTheDocument();
    // Default: redo from Visual (keeps topic, config, script).
    expect(screen.getByTestId("fork-from-4")).toBeChecked();
    fireEvent.click(screen.getByTestId("fork-from-5"));
    fireEvent.click(screen.getByTestId("fork-confirm"));

    await waitFor(() => expect(fork).toHaveBeenCalledWith("p1", 5));
    await waitFor(() => expect(screen.getByTestId("where")).toHaveTextContent("/projects/new-1/resume?step=5"));
  });

  it("stays quiet for a draft: nothing running, nothing to say", async () => {
    renderStrip(project({ status: "draft", flow_step: 3, run_state: "idle" }), 3);
    await waitFor(() => expect(screen.getByTestId("rail-step-3")).toBeInTheDocument());
    await new Promise((r) => setTimeout(r, 20));
    expect(screen.queryByTestId("status-strip")).not.toBeInTheDocument();
  });
});
