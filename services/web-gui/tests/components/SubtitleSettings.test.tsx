import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { SubtitleSettings } from "../../src/components/SubtitleSettings";
import { defaultSubtitleStyle } from "../../src/context/ProjectDraftContext";

function renderSettings(overrides: Partial<React.ComponentProps<typeof SubtitleSettings>> = {}) {
  const props = {
    mode: "off" as const,
    onModeChange: vi.fn(),
    style: defaultSubtitleStyle,
    onStyleChange: vi.fn(),
    ttsEnabled: true,
    ...overrides,
  };
  render(<SubtitleSettings {...props} />);
  return props;
}

describe("SubtitleSettings", () => {
  it("warns when the video would have neither narration nor subtitles", () => {
    renderSettings({ ttsEnabled: false, mode: "off" });

    expect(screen.getByRole("status")).toHaveTextContent("không có lời thoại và phụ đề");
  });

  it("offers all four subtitle delivery modes", () => {
    renderSettings();

    expect(screen.getByTestId("subtitle-mode-off")).toBeInTheDocument();
    expect(screen.getByTestId("subtitle-mode-track")).toBeInTheDocument();
    expect(screen.getByTestId("subtitle-mode-burn_in")).toBeInTheDocument();
    expect(screen.getByTestId("subtitle-mode-both")).toBeInTheDocument();
  });

  it("selecting a mode reports it to the caller", () => {
    const props = renderSettings();

    fireEvent.click(screen.getByTestId("subtitle-mode-track"));

    expect(props.onModeChange).toHaveBeenCalledWith("track");
  });

  it("hides the subtitle style panel for track-only (SRT carries no styling)", () => {
    renderSettings({ mode: "track" });

    expect(screen.queryByTestId("subtitle-style-panel")).not.toBeInTheDocument();
  });

  it("shows the subtitle style panel for burn_in and both", () => {
    renderSettings({ mode: "burn_in" });
    expect(screen.getByTestId("subtitle-style-panel")).toBeInTheDocument();
  });

  it("warns about doubled text only when both is selected", () => {
    renderSettings({ mode: "both" });

    expect(screen.getByRole("status")).toHaveTextContent("chữ bị trùng lặp");
  });
});
