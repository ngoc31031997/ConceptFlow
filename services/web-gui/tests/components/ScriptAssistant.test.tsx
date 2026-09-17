import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { ScriptAssistant } from "../../src/components/ScriptAssistant";

// "Dựng từ đầu" (blank) moved out to its own 4-tab sub-wizard
// (ScriptOutlineStepPage etc. — see ScriptPipelineTabs); ScriptAssistant now
// only covers "draft" (adjust an existing script) and "ready" (paste
// directly, no AI round trip).
function renderAssistant(source: "draft" | "ready" = "draft", renderEngine: "manim" | "remotion" = "manim") {
  const onUseTemplate = vi.fn();
  render(
    <ScriptAssistant contentLanguage="vi" renderEngine={renderEngine} source={source} onUseTemplate={onUseTemplate} />,
  );
  return { onUseTemplate };
}

describe("ScriptAssistant", () => {
  afterEach(() => vi.restoreAllMocks());

  it("substitutes the existing script into the Manim adjust prompt", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText } });
    renderAssistant("draft");

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
    renderAssistant("draft", "remotion");

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
    renderAssistant("draft");
    expect(screen.getByText("Chưa dán script")).toBeInTheDocument();

    fireEvent.change(screen.getByTestId("script-assistant-existing"), {
      target: { value: "class A(Scene): pass" },
    });
    expect(screen.getByText("Đã gắn script của bạn")).toBeInTheDocument();
  });

  it("skips the AI round trip entirely for a ready script (Manim)", () => {
    const { onUseTemplate } = renderAssistant("ready");

    expect(screen.queryByTestId("script-assistant-guide")).not.toBeInTheDocument();
    fireEvent.click(screen.getByTestId("script-assistant-template"));
    expect(onUseTemplate).toHaveBeenCalled();
  });

  it("has no sample-template button for a ready Remotion script (no Remotion sample exists)", () => {
    renderAssistant("ready", "remotion");

    expect(screen.queryByTestId("script-assistant-guide")).not.toBeInTheDocument();
    expect(screen.queryByTestId("script-assistant-template")).not.toBeInTheDocument();
    expect(screen.getByText(/export const narrations/)).toBeInTheDocument();
  });
});
