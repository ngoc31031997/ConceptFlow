import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { ScriptEditor } from "../../src/components/ScriptEditor";

describe("ScriptEditor", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("calls onChange when typing", () => {
    const onChange = vi.fn();
    render(<ScriptEditor value="" onChange={onChange} contentLanguage="vi" />);
    fireEvent.change(screen.getByTestId("new-project-script-textarea"), {
      target: { value: "# Scene 1" },
    });
    expect(onChange).toHaveBeenCalledWith("# Scene 1");
  });

  it("toggles the AI prompt panel and copies the prompt to the clipboard", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText } });

    render(<ScriptEditor value="" onChange={vi.fn()} contentLanguage="vi" />);

    expect(screen.queryByTestId("script-editor-ai-prompt-panel")).not.toBeInTheDocument();

    fireEvent.click(screen.getByTestId("script-editor-ai-prompt-toggle"));
    expect(screen.getByTestId("script-editor-ai-prompt-panel")).toBeInTheDocument();

    fireEvent.click(screen.getByText("Copy"));

    await waitFor(() => expect(writeText).toHaveBeenCalledTimes(1));
    expect(writeText.mock.calls[0][0]).toContain("NARRATION");
    expect(await screen.findByText("Đã copy!")).toBeInTheDocument();
  });
});
