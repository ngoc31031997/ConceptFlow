import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { VideoOutputModePicker } from "../../src/components/VideoOutputModePicker";

describe("VideoOutputModePicker", () => {
  it("defaults to long-only, reproducing the pre-CR-007-follow-up behaviour", () => {
    render(<VideoOutputModePicker value="long" onChange={vi.fn()} />);

    expect(screen.getByTestId("video-output-mode-long")).toHaveAttribute("aria-checked", "true");
    expect(screen.getByTestId("video-output-mode-short")).toHaveAttribute("aria-checked", "false");
    expect(screen.getByTestId("video-output-mode-both")).toHaveAttribute("aria-checked", "false");
  });

  it("reports the chosen mode", () => {
    const onChange = vi.fn();
    render(<VideoOutputModePicker value="long" onChange={onChange} />);

    fireEvent.click(screen.getByTestId("video-output-mode-short"));

    expect(onChange).toHaveBeenCalledWith("short");
  });

  it("warns that a clip needs self.clip(...) markers when short or both is picked", () => {
    render(<VideoOutputModePicker value="short" onChange={vi.fn()} />);
    expect(screen.getByRole("status").textContent).toContain("self.clip");
  });

  it("shows no warning for long-only", () => {
    render(<VideoOutputModePicker value="long" onChange={vi.fn()} />);
    expect(screen.queryByRole("status")).toBeNull();
  });
});
