import { fireEvent, screen, waitFor } from "@testing-library/react";

/**
 * The step menu opens only the phase of the step on screen. Tests that look at
 * steps of other phases open every phase first, the way a Creator would.
 */
export async function openAllPhases(): Promise<void> {
  await waitFor(() => expect(screen.getByTestId("step-rail")).toBeInTheDocument());
  for (let phase = 1; phase <= 5; phase += 1) {
    if (screen.getByTestId(`rail-phase-${phase}`).getAttribute("data-open") === "false") {
      fireEvent.click(screen.getByTestId(`rail-phase-toggle-${phase}`));
    }
  }
}
