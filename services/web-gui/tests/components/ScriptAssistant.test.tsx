import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { ScriptAssistant } from "../../src/components/ScriptAssistant";

function renderAssistant(source: "blank" | "draft" | "ready" = "blank") {
  const onSourceChange = vi.fn();
  const onUseTemplate = vi.fn();
  render(
    <ScriptAssistant
      contentLanguage="vi"
      source={source}
      onSourceChange={onSourceChange}
      onUseTemplate={onUseTemplate}
    />,
  );
  return { onSourceChange, onUseTemplate };
}

describe("ScriptAssistant", () => {
  afterEach(() => vi.restoreAllMocks());

  it("marks exactly one situation as chosen", () => {
    // The whole point of the picker is that the active option is obvious, so
    // the selected state is asserted rather than left to the styling.
    renderAssistant("draft");

    expect(screen.getByTestId("script-source-draft")).toHaveAttribute("aria-checked", "true");
    expect(screen.getByTestId("script-source-blank")).toHaveAttribute("aria-checked", "false");
    expect(screen.getByTestId("script-source-ready")).toHaveAttribute("aria-checked", "false");
  });

  it("copies a prompt with the Creator's topic already substituted in", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText } });
    renderAssistant("blank");

    fireEvent.change(screen.getByTestId("script-assistant-topic"), {
      target: { value: "Vòng lặp for trong Java" },
    });
    fireEvent.click(screen.getByTestId("script-assistant-copy"));

    await waitFor(() => expect(writeText).toHaveBeenCalledTimes(1));
    const copied = writeText.mock.calls[0][0] as string;
    // Copying a prompt that still says "paste your topic here" was the most
    // common way the round trip failed.
    expect(copied).toContain("Vòng lặp for trong Java");
    expect(copied).not.toContain("[DÁN CHỦ ĐỀ CỦA BẠN VÀO ĐÂY]");
  });

  it("substitutes the existing script for the adjust prompt instead", async () => {
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

  it("says when the prompt is still missing its input", () => {
    renderAssistant("blank");
    expect(screen.getByText("Chưa nhập chủ đề")).toBeInTheDocument();

    fireEvent.change(screen.getByTestId("script-assistant-topic"), { target: { value: "Java" } });
    expect(screen.getByText("Đã gắn chủ đề của bạn")).toBeInTheDocument();
  });

  it("skips the AI round trip entirely for a ready script", () => {
    const { onUseTemplate } = renderAssistant("ready");

    expect(screen.queryByTestId("script-assistant-guide")).not.toBeInTheDocument();
    fireEvent.click(screen.getByTestId("script-assistant-template"));
    expect(onUseTemplate).toHaveBeenCalled();
  });
});
