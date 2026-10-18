import { screen, waitFor } from "@testing-library/react";

/** Waits for the step menu; every phase and step in it is always listed. */
export async function waitForStepRail(): Promise<void> {
  await waitFor(() => expect(screen.getByTestId("step-rail")).toBeInTheDocument());
}
