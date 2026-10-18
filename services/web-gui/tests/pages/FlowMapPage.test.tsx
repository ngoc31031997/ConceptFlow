import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
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
    expect(screen.getByTestId("flow-map-step-3")).toHaveTextContent("sắp có");
    expect(screen.getByTestId("flow-map-step-3").tagName).toBe("DIV");
    expect(screen.getByTestId("flow-map-step-6")).not.toHaveTextContent("sắp có");
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
