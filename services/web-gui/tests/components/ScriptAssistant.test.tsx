import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { ScriptAssistant } from "../../src/components/ScriptAssistant";

// CR-031 — ScriptAssistant chỉ còn một việc: biến code sẵn có thành một
// prompt yêu cầu AI chuẩn hoá nó. Nhánh "ready" cũ (hướng dẫn + nút script
// mẫu) đã về tab 1c, nơi ô soạn thảo thật sự nằm.
function renderAssistant(renderEngine: "manim" | "remotion" = "manim") {
  render(<ScriptAssistant contentLanguage="vi" renderEngine={renderEngine} />);
}

describe("ScriptAssistant", () => {
  afterEach(() => vi.restoreAllMocks());

  it("substitutes the existing script into the Manim adjust prompt", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText } });
    renderAssistant();

    fireEvent.change(screen.getByTestId("script-assistant-existing"), {
      target: { value: "class OldScene(Scene): pass" },
    });
    fireEvent.click(screen.getByTestId("script-assistant-copy"));

    await waitFor(() => expect(writeText).toHaveBeenCalledTimes(1));
    const copied = writeText.mock.calls[0][0] as string;
    expect(copied).toContain("class OldScene(Scene): pass");
    expect(copied).not.toContain("<dán script Manim của bạn vào đây>");
  });

  it("substitutes the existing script into the Remotion adjust prompt instead, when renderEngine is remotion", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText } });
    renderAssistant("remotion");

    fireEvent.change(screen.getByTestId("script-assistant-existing"), {
      target: { value: "export const narrations = ['x'];" },
    });
    fireEvent.click(screen.getByTestId("script-assistant-copy"));

    await waitFor(() => expect(writeText).toHaveBeenCalledTimes(1));
    const copied = writeText.mock.calls[0][0] as string;
    // The Remotion adjust prompt talks about narrations/Composition
    // id="creator", not self.narrate/ConceptFlowScene.
    expect(copied).toContain("export const narrations = ['x'];");
    expect(copied).toContain('Composition id="creator"');
    expect(copied).not.toContain("self.narrate");
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
