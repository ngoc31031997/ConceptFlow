import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { ResumeProjectPage } from "../../src/pages/ResumeProjectPage";
import { ProjectDraftProvider } from "../../src/context/ProjectDraftContext";
import { ThemeProvider } from "../../src/context/ThemeContext";
import * as apiClient from "../../src/api/client";
import type { Project } from "../../src/types";

function researchState(resolved: boolean) {
  vi.spyOn(apiClient, "getResearch").mockResolvedValue({ gate: { resolved } } as unknown as apiClient.ResearchState);
}

function storyState(status: string) {
  vi.spyOn(apiClient, "getStory").mockResolvedValue({ status } as unknown as apiClient.StoryState);
}

function renderResume(flowStep: number) {
  const project = {
    project_id: "p1", status: "draft", voice_language: "vi", scenes: [], render_engine: "remotion", flow_step: flowStep,
  } as unknown as Project;
  vi.spyOn(apiClient, "getProject").mockResolvedValue(project);
  vi.spyOn(apiClient, "getAuthoringState").mockResolvedValue({ mode: "manual", topic: "t", story: "s", storyboard: "{}", code: "" });
  return render(
    <ThemeProvider>
      <MemoryRouter initialEntries={["/projects/p1/resume"]}>
        <ProjectDraftProvider>
          <Routes>
            <Route path="/projects/:id/resume" element={<ResumeProjectPage />} />
            <Route path="/" element={<p>màn ý tưởng</p>} />
            <Route path="/projects/:id/research" element={<p>màn nghiên cứu</p>} />
            <Route path="/projects/:id/story" element={<p>màn câu chuyện</p>} />
            <Route path="/projects/:id/beats" element={<p>màn beat sheet</p>} />
            <Route path="/projects/:id/script" element={<p>màn kịch bản</p>} />
            <Route path="/create/script/storyboard" element={<p>màn visual</p>} />
            <Route path="/create/script/illustrations" element={<p>màn hình minh hoạ</p>} />
            <Route path="/create/script/code" element={<p>màn code</p>} />
          </Routes>
        </ProjectDraftProvider>
      </MemoryRouter>
    </ThemeProvider>,
  );
}

// A draft standing at Code (6) must reopen on Code, not on Hình minh hoạ.
describe("ResumeProjectPage mở lại đúng bước server báo", () => {
  afterEach(() => vi.restoreAllMocks());

  it("dự án ở bước 6 (Code) mở màn Code", async () => {
    renderResume(6);
    expect(await screen.findByText("màn code")).toBeInTheDocument();
  });

  it("dự án ở bước 5 mở màn Hình minh hoạ", async () => {
    renderResume(5);
    expect(await screen.findByText("màn hình minh hoạ")).toBeInTheDocument();
  });

  it("dự án ở bước 4 mở màn Visual", async () => {
    renderResume(4);
    expect(await screen.findByText("màn visual")).toBeInTheDocument();
  });

  it("dự án ở bước 1 (brief chưa duyệt) mở lại màn Ý tưởng", async () => {
    renderResume(1);
    expect(await screen.findByText("màn ý tưởng")).toBeInTheDocument();
  });

  it("dự án ở bước 3 mà nghiên cứu chưa duyệt/bỏ qua mở màn Nghiên cứu", async () => {
    researchState(false);
    renderResume(3);
    expect(await screen.findByText("màn nghiên cứu")).toBeInTheDocument();
  });

  it("dự án ở bước 3 với nghiên cứu đã xong mà câu chuyện chưa duyệt mở màn Câu chuyện", async () => {
    researchState(true);
    storyState("pending");
    renderResume(3);
    expect(await screen.findByText("màn câu chuyện")).toBeInTheDocument();
  });

  it("dự án ở bước 3 với câu chuyện đã duyệt mà beat chưa duyệt mở màn Beat sheet", async () => {
    researchState(true);
    storyState("approved");
    vi.spyOn(apiClient, "getBeats").mockResolvedValue({ status: "pending" } as never);
    renderResume(3);
    expect(await screen.findByText("màn beat sheet")).toBeInTheDocument();
  });

  it("dự án ở bước 3 với beat đã duyệt mở màn Viết kịch bản mới", async () => {
    researchState(true);
    storyState("approved");
    vi.spyOn(apiClient, "getBeats").mockResolvedValue({ status: "approved" } as never);
    renderResume(3);
    expect(await screen.findByText("màn kịch bản")).toBeInTheDocument();
  });
});
