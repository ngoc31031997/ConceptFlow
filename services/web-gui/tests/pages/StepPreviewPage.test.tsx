import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter, Routes, Route, useLocation } from "react-router-dom";
import { StepPreviewPage } from "../../src/pages/StepPreviewPage";
import { ProjectDraftProvider } from "../../src/context/ProjectDraftContext";
import { ProjectFlowProvider } from "../../src/context/ProjectFlowContext";
import { ThemeProvider } from "../../src/context/ThemeContext";
import * as apiClient from "../../src/api/client";
import type { Project } from "../../src/types";

function Where() {
  const { pathname, search } = useLocation();
  return <span data-testid="where">{pathname + search}</span>;
}

function renderPreview(path: string, p: Project | null) {
  if (p) vi.spyOn(apiClient, "getProject").mockResolvedValue(p);
  else vi.spyOn(apiClient, "getProject").mockRejectedValue(new apiClient.ApiError("not found"));
  vi.spyOn(apiClient, "listProjectErrors").mockResolvedValue([]);
  vi.spyOn(apiClient, "listProjects").mockResolvedValue([]);
  return render(
    <ThemeProvider>
      <MemoryRouter initialEntries={[path]}>
        <ProjectDraftProvider>
          <ProjectFlowProvider>
            <Routes>
              <Route path="/create/preview/:step" element={<StepPreviewPage />} />
              <Route path="/projects/:id/preview/:step" element={<StepPreviewPage />} />
              <Route path="*" element={<div data-testid="elsewhere" />} />
            </Routes>
            <Where />
          </ProjectFlowProvider>
        </ProjectDraftProvider>
      </MemoryRouter>
    </ThemeProvider>,
  );
}

const project = (over: Partial<Project>) =>
  ({ project_id: "p1", status: "draft", voice_language: "vi", scenes: [], ...over }) as Project;

describe("StepPreviewPage — xem trước bước chưa tới", () => {
  afterEach(() => {
    vi.restoreAllMocks();
    window.localStorage.clear();
  });

  it("nói bước làm gì và rằng nó chưa tới, không có ô nhập hay nút hành động", async () => {
    renderPreview("/create/preview/4", null);

    expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("Bước 4 — Hình ảnh");
    expect(screen.getByTestId("step-preview-notice")).toHaveTextContent("Chưa tới bước này");
    expect(screen.getByTestId("step-preview-page")).toHaveTextContent("Dựng storyboard");
    expect(document.querySelector("textarea, input, select")).toBeNull();
    // The step menu marks the previewed step as the one on screen.
    await waitFor(() =>
      expect(screen.getByTestId("rail-step-4")).toHaveAttribute("aria-current", "step"),
    );
  });

  it("nói rõ bước do máy tự chạy", () => {
    renderPreview("/create/preview/10", null);
    expect(screen.getByTestId("step-preview-auto")).toHaveTextContent("máy tự chạy");
  });

  it("giải thích vì sao một bước không dùng", async () => {
    renderPreview(
      "/projects/p1/preview/5",
      project({ flow_step: 4, run_state: "idle", render_engine: "manim" }),
    );
    await waitFor(() =>
      expect(screen.getByTestId("step-preview-notice")).toHaveTextContent("Không dùng"),
    );
    expect(screen.getByTestId("step-preview-notice")).toHaveTextContent("chỉ video Remotion");
  });

  it("“Về bước đang làm” mở bước hiện tại của project", async () => {
    renderPreview(
      "/projects/p1/preview/13",
      project({ status: "rendering", flow_step: 10, run_state: "running" }),
    );
    await waitFor(() =>
      expect(screen.getByTestId("step-preview-back")).toHaveTextContent(
        "Về bước đang làm: Dựng hình",
      ),
    );
    fireEvent.click(screen.getByTestId("step-preview-back"));
    expect(screen.getByTestId("where")).toHaveTextContent("/projects/p1/render");
  });

  it("chuyển sang màn thật khi project đã tới bước này", async () => {
    renderPreview(
      "/projects/p1/preview/9",
      project({ status: "rendering", flow_step: 10, run_state: "running" }),
    );
    await waitFor(() =>
      expect(screen.getByTestId("where").textContent).toMatch(/^\/projects\/p1\/render/),
    );
  });

  it("đưa về bước 1 khi số bước không hợp lệ", () => {
    renderPreview("/create/preview/99", null);
    expect(screen.getByTestId("where")).toHaveTextContent("/");
    expect(screen.queryByTestId("step-preview-page")).not.toBeInTheDocument();
  });
});
