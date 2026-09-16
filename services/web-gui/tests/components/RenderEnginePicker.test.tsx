import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { RenderEnginePicker } from "../../src/components/RenderEnginePicker";

describe("RenderEnginePicker", () => {
  it("defaults the project to Manim", () => {
    render(<RenderEnginePicker value="manim" onChange={vi.fn()} />);

    expect(screen.getByTestId("render-engine-manim")).toHaveAttribute("aria-checked", "true");
    expect(screen.getByTestId("render-engine-remotion")).toHaveAttribute("aria-checked", "false");
  });

  it("reports the chosen engine", () => {
    const onChange = vi.fn();
    render(<RenderEnginePicker value="manim" onChange={onChange} />);

    fireEvent.click(screen.getByTestId("render-engine-remotion"));

    expect(onChange).toHaveBeenCalledWith("remotion");
  });

  it("warns that Remotion has no design system yet", () => {
    render(<RenderEnginePicker value="remotion" onChange={vi.fn()} />);

    expect(screen.getByTestId("render-engine-picker").textContent).toContain("giai đoạn đầu");
  });

  it("describes Manim as the full-featured default", () => {
    render(<RenderEnginePicker value="manim" onChange={vi.fn()} />);

    expect(screen.getByTestId("render-engine-picker").textContent).toContain("Mặc định");
  });
});
