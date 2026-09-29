import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { ResumeProjectPage } from "../../src/pages/ResumeProjectPage";
import { ProjectDraftProvider } from "../../src/context/ProjectDraftContext";
import { ThemeProvider } from "../../src/context/ThemeContext";
import * as apiClient from "../../src/api/client";
import type { Project } from "../../src/types";

function renderResume(flowStep: number, renderEngine: "manim" | "remotion") {
  const project = {
    project_id: "p1", status: "draft", voice_language: "vi", scenes: [], render_engine: renderEngine, flow_step: flowStep,
  } as unknown as Project;
  vi.spyOn(apiClient, "getProject").mockResolvedValue(project);
  vi.spyOn(apiClient, "getAuthoringState").mockResolvedValue({ mode: "manual", topic: "t", story: "s", storyboard: "{}", code: "" });
  return render(
    <ThemeProvider>
      <MemoryRouter initialEntries={["/projects/p1/resume"]}>
        <ProjectDraftProvider>
          <Routes>
            <Route path="/projects/:id/resume" element={<ResumeProjectPage />} />
            <Route path="/create/script/storyboard" element={<p>màn visual</p>} />
            <Route path="/create/script/illustrations" element={<p>màn hình minh hoạ</p>} />
            <Route path="/create/script/code" element={<p>màn code</p>} />
          </Routes>
        </ProjectDraftProvider>
      </MemoryRouter>
    </ThemeProvider>,
  );
}

// CR-051: the resume target was capped at 5 — Code in the 13-step flow — so
// after CR-046 a draft standing at Code (6) reopened on Hình minh hoạ.
describe("ResumeProjectPage mở lại đúng bước server báo", () => {
  afterEach(() => vi.restoreAllMocks());

  it("dự án ở bước 6 (Code) mở màn Code, kể cả video Manim", async () => {
    renderResume(6, "manim");
    expect(await screen.findByText("màn code")).toBeInTheDocument();
  });

  it("dự án Remotion ở bước 5 mở màn Hình minh hoạ", async () => {
    renderResume(5, "remotion");
    expect(await screen.findByText("màn hình minh hoạ")).toBeInTheDocument();
  });

  it("dự án ở bước 4 mở màn Visual", async () => {
    renderResume(4, "remotion");
    expect(await screen.findByText("màn visual")).toBeInTheDocument();
  });
});
