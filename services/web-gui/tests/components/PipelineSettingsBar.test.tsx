import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { PipelineSettingsBar } from "../../src/components/PipelineSettingsBar";

const LLM_ENABLED = { enabled: true, provider: "hive" };

function renderBar(overrides: Partial<Parameters<typeof PipelineSettingsBar>[0]> = {}) {
  return render(
    <PipelineSettingsBar
      llm={LLM_ENABLED}
      mode="manual"
      onModeChange={vi.fn()}
      projectId="p1"
      {...overrides}
    />,
  );
}

// Cách làm đã chốt ở bước Cấu hình; hiện lại y nguyên bộ chọn đầy đủ ở mỗi
// tab 1a/1b/1c sẽ đọc như thể Creator chưa chọn gì. PipelineSettingsBar thay
// nó bằng một dòng tóm tắt, chỉ mở lại thành bộ chọn đầy đủ khi bấm "Đổi".
describe("PipelineSettingsBar", () => {
  it("mặc định thu gọn: không hiện công tắc chế độ", () => {
    renderBar();

    expect(screen.queryByTestId("authoring-mode-switch")).not.toBeInTheDocument();
  });

  it("bấm 'Đổi' mở ra công tắc chế độ; không có bộ chọn engine", () => {
    renderBar();

    fireEvent.click(screen.getByTestId("pipeline-settings-toggle"));

    expect(screen.getByTestId("authoring-mode-switch")).toBeInTheDocument();
    expect(screen.queryByTestId("render-engine-picker")).not.toBeInTheDocument();
  });

  it("dòng tóm tắt nói đúng chế độ đang chọn", () => {
    renderBar({ mode: "ai" });

    expect(screen.getByTestId("pipeline-settings-bar")).toHaveTextContent("Để AI làm giúp");
  });
});
