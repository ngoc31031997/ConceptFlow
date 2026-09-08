import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { RenderQualityPicker } from "../../src/components/RenderQualityPicker";

describe("RenderQualityPicker", () => {
  it("defaults the project to 1080p60, not the old hardcoded 720p30", () => {
    // CR-004 FR12.1: 720p30 is below what a monetized channel should publish.
    render(<RenderQualityPicker value="1080p60" onChange={vi.fn()} />);

    expect(screen.getByTestId("render-quality-1080p60")).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByTestId("render-quality-720p30")).toHaveAttribute("aria-pressed", "false");
  });

  it("reports the chosen preset", () => {
    const onChange = vi.fn();
    render(<RenderQualityPicker value="1080p60" onChange={onChange} />);

    fireEvent.click(screen.getByTestId("render-quality-720p30"));

    expect(onChange).toHaveBeenCalledWith("720p30");
  });

  it("warns that a draft render is not meant for publishing", () => {
    render(<RenderQualityPicker value="720p30" onChange={vi.fn()} />);

    expect(screen.getByRole("status").textContent).toContain("không nên dùng để đăng");
  });

  it("warns that 4K is rarely worth its cost", () => {
    render(<RenderQualityPicker value="4k60" onChange={vi.fn()} />);

    expect(screen.getByRole("status").textContent).toContain("nặng hơn");
  });

  it("shows no warning for the recommended preset", () => {
    render(<RenderQualityPicker value="1080p60" onChange={vi.fn()} />);

    expect(screen.queryByRole("status")).toBeNull();
  });
});
