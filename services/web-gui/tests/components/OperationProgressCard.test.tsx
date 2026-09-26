import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { OperationProgressCard } from "../../src/components/OperationProgressCard";
import { operationSubtitle, operationErrorLabel } from "../../src/lib/formatProgress";

describe("OperationProgressCard", () => {
  it("runs indeterminate, without a made-up percentage, when the total is unknown (FR116.4)", () => {
    render(<OperationProgressCard subtitle="AI đang phân tích · 1,2k ký tự · 5s" />);
    const bar = screen.getByRole("progressbar");
    expect(bar).not.toHaveAttribute("aria-valuenow");
    expect(screen.getByText(/AI đang phân tích/)).toBeInTheDocument();
  });

  it("shows a percentage only when total is known", () => {
    render(<OperationProgressCard done={1} total={4} />);
    expect(screen.getByRole("progressbar")).toHaveAttribute("aria-valuenow", "25");
  });

  it("keeps the subtitle and shows the classified error on failure (FR116.5)", () => {
    render(<OperationProgressCard subtitle="AI đang viết · 14,3k ký tự · 2m05s" error={operationErrorLabel("balance")} />);
    expect(screen.getByText(/14,3k ký tự/)).toBeInTheDocument();
    expect(screen.getByRole("alert")).toHaveTextContent("balance");
  });
});

describe("operationSubtitle", () => {
  const base = { reasoning_chars: 0, content_chars: 0, elapsed_ms: 125_000, done: null, total: null };
  it("formats phase, chars and clock like step 1b", () => {
    expect(operationSubtitle({ ...base, phase: "writing", content_chars: 14300 })).toBe("AI đang viết · 14,3k ký tự · 2m05s");
    expect(operationSubtitle({ ...base, phase: "reasoning", reasoning_chars: 900 })).toBe("AI đang phân tích · 900 ký tự · 2m05s");
  });
  it("prefers done/total when the operation knows its size", () => {
    expect(operationSubtitle({ ...base, phase: "purge", done: 1, total: 3 })).toBe("1/3 · 2m05s");
  });
});
