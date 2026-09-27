import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { ProgressTracker } from "../../src/components/ProgressTracker";
import type { ProgressState } from "../../src/hooks/useSSE";

const base: ProgressState = {
  currentStep: "render_scenes",
  sceneIndex: null,
  sceneTotal: null,
  elapsedSeconds: 65,
  animationIndex: null,
  renderPercent: null,
  stageIndex: null,
  stageTotal: null,
  clipIndex: null,
  clipTotal: null,
  status: "in_progress",
  errorMessage: null,
};

describe("ProgressTracker — render progress", () => {
  it("draws a bar from a Remotion render's percentage", () => {
    render(<ProgressTracker progressState={{ ...base, renderPercent: 45 }} steps={["render_scenes"]} />);

    expect(screen.getByTestId("progress-tracker-bar")).toHaveStyle({ width: "45%" });
    expect(screen.getByText("Đã render 45% · 1:05")).toBeInTheDocument();
    expect(screen.queryByTestId("progress-tracker-elapsed")).not.toBeInTheDocument();
  });

  it("keeps the elapsed text for a Manim render, which sends no percentage", () => {
    render(<ProgressTracker progressState={{ ...base, animationIndex: 3 }} steps={["render_scenes"]} />);

    expect(screen.queryByTestId("progress-tracker-bar")).not.toBeInTheDocument();
    expect(screen.getByTestId("progress-tracker-elapsed")).toHaveTextContent("Đã render 1:05 — animation 3");
  });
});
