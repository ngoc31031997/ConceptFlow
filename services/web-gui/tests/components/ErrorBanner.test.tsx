import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { ErrorBanner } from "../../src/components/ErrorBanner";

describe("ErrorBanner", () => {
  it("shows message and triggers retry", () => {
    const onRetry = vi.fn();
    render(<ErrorBanner errorMessage="Lỗi render scene 3" onRetry={onRetry} isRetrying={false} />);

    expect(screen.getByTestId("error-banner-message").textContent).toBe("Lỗi render scene 3");
    fireEvent.click(screen.getByTestId("error-banner-retry-button"));
    expect(onRetry).toHaveBeenCalled();
  });

  it("disables retry button while retrying", () => {
    render(<ErrorBanner errorMessage="err" onRetry={vi.fn()} isRetrying={true} />);
    expect(screen.getByTestId("error-banner-retry-button")).toBeDisabled();
  });

  it("does not show a back button when onBack is not provided", () => {
    render(<ErrorBanner errorMessage="err" onRetry={vi.fn()} isRetrying={false} />);
    expect(screen.queryByTestId("error-banner-back-button")).not.toBeInTheDocument();
  });

  it("shows a back button and triggers it when onBack is provided", () => {
    const onBack = vi.fn();
    render(<ErrorBanner errorMessage="err" onRetry={vi.fn()} isRetrying={false} onBack={onBack} />);

    fireEvent.click(screen.getByTestId("error-banner-back-button"));
    expect(onBack).toHaveBeenCalled();
  });
});
