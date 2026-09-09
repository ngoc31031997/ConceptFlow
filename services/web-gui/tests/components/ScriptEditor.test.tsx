import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { ScriptEditor } from "../../src/components/ScriptEditor";
import { HOOK_SNIPPETS } from "../../src/components/scriptTemplates";

const VALID = 'class DemoScene(Scene):\n    def construct(self):\n        # NARRATION: "hi"\n        self.wait(AUTO)';

describe("ScriptEditor", () => {
  it("calls onChange when typing", () => {
    const onChange = vi.fn();
    render(<ScriptEditor value="" onChange={onChange} contentLanguage="vi" />);
    fireEvent.change(screen.getByTestId("new-project-script-textarea"), {
      target: { value: "# Scene 1" },
    });
    expect(onChange).toHaveBeenCalledWith("# Scene 1");
  });

  it("reports whether the script's markers and waits line up", () => {
    const { rerender } = render(<ScriptEditor value={VALID} onChange={vi.fn()} contentLanguage="vi" />);
    expect(screen.getByTestId("script-editor-validation")).toHaveTextContent("Hợp lệ");

    rerender(
      <ScriptEditor
        value={'class A(Scene):\n    def construct(self):\n        # NARRATION: "x"\n        self.wait(1)'}
        onChange={vi.fn()}
        contentLanguage="vi"
      />,
    );
    expect(screen.getByTestId("script-editor-validation")).toHaveTextContent("Lệch số lượng");
  });

  it("offers the snippets only once there is a script to append them to", () => {
    const onChange = vi.fn();
    const { rerender } = render(<ScriptEditor value="" onChange={onChange} contentLanguage="vi" />);
    expect(screen.queryByTestId("script-editor-insert-hook")).not.toBeInTheDocument();

    rerender(<ScriptEditor value={VALID} onChange={onChange} contentLanguage="vi" />);
    fireEvent.click(screen.getByTestId("script-editor-insert-hook"));
    expect(onChange).toHaveBeenCalledWith(`${VALID}\n${HOOK_SNIPPETS.vi}`);
  });
});
