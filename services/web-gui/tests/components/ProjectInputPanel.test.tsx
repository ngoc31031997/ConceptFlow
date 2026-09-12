import { describe, it, expect } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { ProjectInputPanel } from "../../src/components/ProjectInputPanel";
import type { Project } from "../../src/types";

const BASE_PROJECT: Project = {
  project_id: "proj-1",
  status: "ready_to_publish",
  voice_language: "vi",
  scenes: [],
  script_content: "from conceptflow import *\n\nclass DemoScene(ConceptFlowScene):\n    pass\n",
  voice_id: "vi-VN-HoaiMyNeural",
  tts_enabled: true,
  subtitle_mode: "track",
  render_quality: "1080p60",
  video_format_id: "quick_explainer_3min",
  background_music_path: "assets/bgm/lofi.mp3",
  background_music_volume: 0.2,
};

describe("ProjectInputPanel", () => {
  it("ẩn chi tiết cho tới khi bấm mở, tránh chiếm chỗ trang kết quả", () => {
    render(<ProjectInputPanel project={BASE_PROJECT} />);
    expect(screen.queryByTestId("project-input-script")).not.toBeInTheDocument();
  });

  it("hiện đầy đủ script gốc và cấu hình sau khi mở", () => {
    render(<ProjectInputPanel project={BASE_PROJECT} />);
    fireEvent.click(screen.getByTestId("project-input-toggle"));

    expect(screen.getByTestId("project-input-script")).toHaveValue(BASE_PROJECT.script_content);
    expect(screen.getByText("vi-VN-HoaiMyNeural")).toBeInTheDocument();
    expect(screen.getByText(/Chuẩn \(1080p60\)/)).toBeInTheDocument();
    expect(screen.getByText(/lofi\.mp3/)).toBeInTheDocument();
  });

  it("báo rõ khi project cũ không còn lưu script gốc, thay vì im lặng bỏ trống", () => {
    const project: Project = { ...BASE_PROJECT, script_content: undefined };
    render(<ProjectInputPanel project={project} />);
    fireEvent.click(screen.getByTestId("project-input-toggle"));

    expect(screen.queryByTestId("project-input-script")).not.toBeInTheDocument();
    expect(screen.getByRole("alert")).toHaveTextContent(/không còn lưu script gốc/);
  });
});
