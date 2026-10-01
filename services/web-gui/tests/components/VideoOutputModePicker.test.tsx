import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { VideoOutputModePicker } from "../../src/components/VideoOutputModePicker";

describe("VideoOutputModePicker", () => {
  it("defaults to long-only", () => {
    render(<VideoOutputModePicker value="long" onChange={vi.fn()} />);

    expect(screen.getByTestId("video-output-mode-long")).toHaveAttribute("aria-checked", "true");
    expect(screen.getByTestId("video-output-mode-short")).toHaveAttribute("aria-checked", "false");
    expect(screen.queryByTestId("video-output-mode-both")).toBeNull();
  });

  it("reports the chosen mode", () => {
    const onChange = vi.fn();
    render(<VideoOutputModePicker value="long" onChange={onChange} />);

    fireEvent.click(screen.getByTestId("video-output-mode-short"));

    expect(onChange).toHaveBeenCalledWith("short");
  });

  it("says what a vertical short is built with when short is picked", () => {
    render(<VideoOutputModePicker value="short" onChange={vi.fn()} />);
    const note = screen.getByRole("status").textContent ?? "";
    expect(note).toContain("Remotion");
    expect(note).not.toContain("self.clip");
  });

  it("shows no warning for long-only", () => {
    render(<VideoOutputModePicker value="long" onChange={vi.fn()} />);
    expect(screen.queryByRole("status")).toBeNull();
  });
});
