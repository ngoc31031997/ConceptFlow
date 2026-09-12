import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { NavigationLoader } from "../../src/components/NavigationLoader";

// Mock useNavigation hook
const mockUseNavigation = vi.fn();
vi.mock("react-router-dom", () => ({
  useNavigation: () => mockUseNavigation(),
}));

describe("NavigationLoader", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("does not show loader when navigation state is idle", () => {
    mockUseNavigation.mockReturnValue({ state: "idle" });
    render(<NavigationLoader />);
    expect(screen.queryByTestId("navigation-loader")).not.toBeInTheDocument();
  });

  it("shows loader after delay when navigation state is loading", () => {
    mockUseNavigation.mockReturnValue({ state: "loading" });
    render(<NavigationLoader />);

    // Should not show immediately
    expect(screen.queryByTestId("navigation-loader")).not.toBeInTheDocument();

    // Should show after 100ms delay
    vi.advanceTimersByTime(100);
    expect(screen.getByTestId("navigation-loader")).toBeInTheDocument();
    expect(screen.getByText("Đang tải...")).toBeInTheDocument();
  });

  it("does not show loader if navigation completes before delay", () => {
    mockUseNavigation.mockReturnValue({ state: "loading" });
    const { rerender } = render(<NavigationLoader />);

    // Navigation completes before 100ms
    vi.advanceTimersByTime(50);
    mockUseNavigation.mockReturnValue({ state: "idle" });
    rerender(<NavigationLoader />);

    vi.advanceTimersByTime(100);
    expect(screen.queryByTestId("navigation-loader")).not.toBeInTheDocument();
  });

  it("hides loader when navigation completes", () => {
    mockUseNavigation.mockReturnValue({ state: "loading" });
    const { rerender } = render(<NavigationLoader />);

    vi.advanceTimersByTime(100);
    expect(screen.getByTestId("navigation-loader")).toBeInTheDocument();

    // Navigation completes
    mockUseNavigation.mockReturnValue({ state: "idle" });
    rerender(<NavigationLoader />);
    expect(screen.queryByTestId("navigation-loader")).not.toBeInTheDocument();
  });

  it("has proper accessibility attributes", () => {
    mockUseNavigation.mockReturnValue({ state: "loading" });
    render(<NavigationLoader />);
    vi.advanceTimersByTime(100);

    const loader = screen.getByTestId("navigation-loader");
    expect(loader).toHaveAttribute("role", "status");
    expect(loader).toHaveAttribute("aria-live", "polite");
  });
});
