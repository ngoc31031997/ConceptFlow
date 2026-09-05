import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { ScriptEditor } from "../../src/components/ScriptEditor";

describe("ScriptEditor", () => {
  it("calls onChange when typing", () => {
    const onChange = vi.fn();
    render(<ScriptEditor value="" onChange={onChange} />);
    fireEvent.change(screen.getByTestId("new-project-script-textarea"), {
      target: { value: "# Scene 1" },
    });
    expect(onChange).toHaveBeenCalledWith("# Scene 1");
  });
});
