import { describe, it, expect, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import { AiOperationCard } from "../../src/components/AiOperationCard";
import type { OperationRun } from "../../src/hooks/useOperationRun";

function runWith(over: Partial<OperationRun> = {}): OperationRun {
  return { progress: null, begin: vi.fn(), end: vi.fn(), cancel: vi.fn().mockResolvedValue(undefined), cancelError: null, resumed: false, ...over };
}

describe("AiOperationCard", () => {
  it("draws nothing while no request is open", () => {
    render(<AiOperationCard run={runWith()} active={false} testId="op" />);
    expect(screen.queryByTestId("op")).toBeNull();
  });

  it("asks before it cancels, and cancels only on the confirmation", () => {
    const run = runWith();
    render(<AiOperationCard run={run} active testId="op" title="AI đang chạy" />);

    fireEvent.click(screen.getByTestId("op-cancel"));
    expect(run.cancel).not.toHaveBeenCalled();
    expect(screen.getByTestId("op-cancel-dialog")).toBeTruthy();

    fireEvent.click(screen.getByTestId("op-cancel-confirm"));
    expect(run.cancel).toHaveBeenCalledTimes(1);
  });

  it("keeps the run going when the Creator backs out of the confirmation", () => {
    const run = runWith();
    render(<AiOperationCard run={run} active testId="op" />);

    fireEvent.click(screen.getByTestId("op-cancel"));
    fireEvent.click(screen.getByTestId("op-cancel-dialog-cancel"));
    expect(run.cancel).not.toHaveBeenCalled();
  });

  it("offers no cancel on a card that only shows a failure", () => {
    const run = runWith({ progress: { status: "failed", error: "server" } as OperationRun["progress"] });
    render(<AiOperationCard run={run} active={false} testId="op" />);
    expect(screen.queryByTestId("op-cancel")).toBeNull();
  });
});
