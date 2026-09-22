import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { ProgressTracker } from "../../src/components/ProgressTracker";
import type { ProgressState } from "../../src/hooks/useSSE";

const base: ProgressState = {
  currentStep: null,
  sceneIndex: null,
  sceneTotal: null,
  elapsedSeconds: null,
  animationIndex: null,
  stageIndex: null,
  stageTotal: null,
  clipIndex: null,
  clipTotal: null,
  status: "in_progress",
  errorMessage: null,
};

describe("ProgressTracker", () => {
  it("shows the real unit progress reported by the backend", () => {
    render(
      <ProgressTracker
        progressState={{ ...base, currentStep: "synthesize_speech", sceneIndex: 5, sceneTotal: 10 }}
      />
    );
    expect(screen.getByTestId("progress-tracker-bar")).toHaveStyle({ width: "50%" });
  });

  it("switches to failed wording once the saga has failed", () => {
    render(<ProgressTracker progressState={{ ...base, currentStep: "render_scenes" }} isFailed />);
    expect(screen.getByTestId("progress-tracker-step-label")).toHaveTextContent("Dừng ở bước");
  });
});
