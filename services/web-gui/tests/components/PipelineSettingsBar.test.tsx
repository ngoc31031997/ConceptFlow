import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { PipelineSettingsBar } from "../../src/components/PipelineSettingsBar";

const LLM_ENABLED = { enabled: true, provider: "hive" };

function renderBar(overrides: Partial<Parameters<typeof PipelineSettingsBar>[0]> = {}) {
  return render(
    <PipelineSettingsBar
      renderEngine="manim"
      onEngineChange={vi.fn()}
      llm={LLM_ENABLED}
      mode="manual"
      onModeChange={vi.fn()}
      projectId="p1"
      {...overrides}
    />,
  );
}

// CR-031 bug report — engine và cách làm đã chốt ở màn chọn tình huống
// (ScriptStepPage); hiện lại y nguyên hai bộ chọn đầy đủ ở mỗi tab 1a/1b/1c
// đọc như thể Creator chưa chọn gì. PipelineSettingsBar thay chúng bằng một
// dòng tóm tắt, chỉ mở lại thành bộ chọn đầy đủ khi bấm "Đổi".
describe("PipelineSettingsBar", () => {
  it("mặc định thu gọn: không hiện bộ chọn engine hay công tắc chế độ", () => {
    renderBar();

    expect(screen.getByText("Manim", { exact: false })).toBeInTheDocument();
    expect(screen.queryByTestId("render-engine-picker")).not.toBeInTheDocument();
    expect(screen.queryByTestId("authoring-mode-switch")).not.toBeInTheDocument();
  });

  it("bấm 'Đổi' mở ra cả hai bộ chọn đầy đủ", () => {
    renderBar();

    fireEvent.click(screen.getByTestId("pipeline-settings-toggle"));

    expect(screen.getByTestId("render-engine-picker")).toBeInTheDocument();
    expect(screen.getByTestId("authoring-mode-switch")).toBeInTheDocument();
  });

  it("không cho đổi engine khi onEngineChange không được truyền (1b)", () => {
    renderBar({ onEngineChange: undefined });

    fireEvent.click(screen.getByTestId("pipeline-settings-toggle"));

    expect(screen.queryByTestId("render-engine-picker")).not.toBeInTheDocument();
    expect(screen.getByTestId("authoring-mode-switch")).toBeInTheDocument();
  });

  it("dòng tóm tắt nói đúng engine và chế độ đang chọn", () => {
    renderBar({ renderEngine: "remotion", mode: "ai" });

    expect(screen.getByTestId("pipeline-settings-bar")).toHaveTextContent("Remotion");
    expect(screen.getByTestId("pipeline-settings-bar")).toHaveTextContent("Gọi API trực tiếp");
  });
});
