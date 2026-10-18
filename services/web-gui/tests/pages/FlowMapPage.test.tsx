import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor, cleanup } from "@testing-library/react";
import { MemoryRouter, Routes, Route, useLocation } from "react-router-dom";
import { FlowMapPage } from "../../src/pages/FlowMapPage";
import { ProjectDraftProvider } from "../../src/context/ProjectDraftContext";
import { ThemeProvider } from "../../src/context/ThemeContext";
import * as apiClient from "../../src/api/client";

function Where() {
  return <span data-testid="where">{useLocation().pathname}</span>;
}

function renderMap() {
  vi.spyOn(apiClient, "listProjects").mockResolvedValue([]);
  return render(
    <ThemeProvider>
      <MemoryRouter initialEntries={["/flow"]}>
        <ProjectDraftProvider>
          <Routes>
            <Route path="/flow" element={<FlowMapPage />} />
            <Route path="*" element={<Where />} />
          </Routes>
        </ProjectDraftProvider>
      </MemoryRouter>
    </ThemeProvider>,
  );
}

describe("FlowMapPage", () => {
  afterEach(() => vi.restoreAllMocks());

  it("shows the 21 steps in four phases, ◆ on the steps the Creator approves, “sắp có” on the ones not built", () => {
    renderMap();
    expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("Bản đồ luồng");
    const phases = screen.getAllByTestId("flow-map-phase");
    expect(phases.map((p) => p.querySelector("h3")?.textContent)).toEqual([
      "Tiền kỳ · Nội dung",
      "Tiền kỳ · Hình ảnh",
      "Sản xuất",
      "Hậu kỳ",
    ]);
    for (let step = 1; step <= 21; step += 1) expect(screen.getByTestId(`flow-map-step-${step}`)).toBeInTheDocument();
    expect(screen.getByTestId("flow-map-step-9")).toHaveTextContent("◆ bạn duyệt");
    expect(screen.getByTestId("flow-map-step-17")).toHaveTextContent("tự chạy");
    expect(screen.getByTestId("flow-map-step-2")).not.toHaveTextContent("◆");
    expect(screen.getByTestId("flow-map-step-4")).toHaveTextContent("sắp có");
    expect(screen.getByTestId("flow-map-step-6")).not.toHaveTextContent("sắp có");
    // The research step has a screen of its own, so it is not “sắp có”; it is optional.
    expect(screen.getByTestId("flow-map-step-3")).not.toHaveTextContent("sắp có");
    expect(screen.getByTestId("flow-map-step-3")).toHaveTextContent("tuỳ chọn");
  });

  it("opens the not-developed page of a step that is not built", () => {
    renderMap();
    fireEvent.click(screen.getByTestId("flow-map-step-4"));
    expect(screen.getByTestId("where")).toHaveTextContent("/create/coming/4");
  });

  it("opens the research screen of the video last worked on, and only when there is one", async () => {
    renderMap();
    // No video yet: the step is listed but there is nothing to open.
    expect(screen.getByTestId("flow-map-step-3").tagName).toBe("DIV");
    cleanup();

    vi.restoreAllMocks();
    vi.spyOn(apiClient, "listProjects").mockResolvedValue([
      { project_id: "p9", topic: "Sung", status: "draft", run_state: "idle", flow_step: 2, updated_at: "2026-10-18T00:00:00Z" } as never,
    ]);
    render(
      <ThemeProvider>
        <MemoryRouter initialEntries={["/flow"]}>
          <ProjectDraftProvider>
            <Routes>
              <Route path="/flow" element={<FlowMapPage />} />
              <Route path="*" element={<Where />} />
            </Routes>
          </ProjectDraftProvider>
        </MemoryRouter>
      </ThemeProvider>,
    );
    await waitFor(() => expect(screen.getByTestId("flow-map-step-3").tagName).toBe("A"));
    fireEvent.click(screen.getByTestId("flow-map-step-3"));
    expect(screen.getByTestId("where")).toHaveTextContent("/projects/p9/research");
  });

  it("opens a step's preview", () => {
    renderMap();
    fireEvent.click(screen.getByTestId("flow-map-step-13"));
    expect(screen.getByTestId("where")).toHaveTextContent("/create/preview/5");
  });

  it("is in the main menu with how many steps it shows", () => {
    renderMap();
    expect(screen.getByTestId("nav-flow-map")).toHaveTextContent("Bản đồ luồngxem cả 21 bước");
  });
});
