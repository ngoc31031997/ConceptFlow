import { describe, it, expect, vi, afterEach, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { ScriptAssistant } from "../../src/components/ScriptAssistant";
import * as apiClient from "../../src/api/client";
import { mockRenderPrompt } from "../helpers/renderPromptMock";

// CR-031 — ScriptAssistant chỉ còn một việc: biến code sẵn có thành một
// prompt yêu cầu AI chuẩn hoá nó. Nhánh "ready" cũ (hướng dẫn + nút script
// mẫu) đã về tab 1c, nơi ô soạn thảo thật sự nằm.
function renderAssistant(renderEngine: "manim" | "remotion" = "manim") {
  render(<ScriptAssistant contentLanguage="vi" renderEngine={renderEngine} />);
}

describe("ScriptAssistant", () => {
  afterEach(() => vi.restoreAllMocks());
  beforeEach(() => {
    mockRenderPrompt();
  });

  it("asks the server for the Manim adjust prompt with the pasted script, and copies its answer", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText } });
    renderAssistant();

    fireEvent.change(screen.getByTestId("script-assistant-existing"), {
      target: { value: "class OldScene(Scene): pass" },
    });

    await waitFor(() =>
      expect(apiClient.renderPrompt).toHaveBeenCalledWith(
        expect.objectContaining({ role: "manim_adjust", language: "vi", script: "class OldScene(Scene): pass" }),
      ),
    );
    await waitFor(() => expect(screen.getByTestId("script-assistant-copy")).toBeEnabled());
    fireEvent.click(screen.getByTestId("script-assistant-copy"));
    await waitFor(() => expect(writeText).toHaveBeenCalledTimes(1));
    expect(writeText.mock.calls[0][0]).toContain("RENDERED[manim_adjust]");
    expect(writeText.mock.calls[0][0]).toContain("class OldScene(Scene): pass");
  });

  it("uses the remotion_adjust role instead when renderEngine is remotion", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText } });
    renderAssistant("remotion");

    fireEvent.change(screen.getByTestId("script-assistant-existing"), {
      target: { value: "export const narrations = ['x'];" },
    });

    await waitFor(() =>
      expect(apiClient.renderPrompt).toHaveBeenCalledWith(
        expect.objectContaining({ role: "remotion_adjust", script: "export const narrations = ['x'];" }),
      ),
    );
    await waitFor(() => expect(screen.getByTestId("script-assistant-copy")).toBeEnabled());
    fireEvent.click(screen.getByTestId("script-assistant-copy"));
    await waitFor(() => expect(writeText).toHaveBeenCalledTimes(1));
    expect(writeText.mock.calls[0][0]).toContain("RENDERED[remotion_adjust]");
  });

  it("says when the prompt is still missing its input", () => {
    renderAssistant();
    expect(screen.getByText("Chưa dán script")).toBeInTheDocument();

    fireEvent.change(screen.getByTestId("script-assistant-existing"), {
      target: { value: "class A(Scene): pass" },
    });
    expect(screen.getByText("Đã gắn script của bạn")).toBeInTheDocument();
  });

});
