import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, act } from "@testing-library/react";
import { NavigationLoader } from "../../src/components/NavigationLoader";

// `useNavigation()` only works under a data router (`createBrowserRouter`);
// this app uses plain `<BrowserRouter>`, so the component tracks route
// changes via `useLocation()` instead — mock that one hook the same way the
// old test mocked `useNavigation`.
const mockUseLocation = vi.fn();
vi.mock("react-router-dom", () => ({
  useLocation: () => mockUseLocation(),
}));

describe("NavigationLoader", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.useFakeTimers();
    mockUseLocation.mockReturnValue({ pathname: "/" });
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("does not show loader when the route hasn't changed", () => {
    render(<NavigationLoader />);
    expect(screen.queryByTestId("navigation-loader")).not.toBeInTheDocument();
  });

  it("shows loader briefly when the route changes", () => {
    const { rerender } = render(<NavigationLoader />);
    expect(screen.queryByTestId("navigation-loader")).not.toBeInTheDocument();

    mockUseLocation.mockReturnValue({ pathname: "/create/settings" });
    rerender(<NavigationLoader />);

    expect(screen.getByTestId("navigation-loader")).toBeInTheDocument();
    expect(screen.getByText("Đang tải...")).toBeInTheDocument();
  });

  it("hides the loader again after the delay", () => {
    const { rerender } = render(<NavigationLoader />);

    mockUseLocation.mockReturnValue({ pathname: "/create/settings" });
    rerender(<NavigationLoader />);
    expect(screen.getByTestId("navigation-loader")).toBeInTheDocument();

    act(() => {
      vi.advanceTimersByTime(250);
    });

    expect(screen.queryByTestId("navigation-loader")).not.toBeInTheDocument();
  });

  it("has proper accessibility attributes", () => {
    const { rerender } = render(<NavigationLoader />);
    mockUseLocation.mockReturnValue({ pathname: "/create/settings" });
    rerender(<NavigationLoader />);

    const loader = screen.getByTestId("navigation-loader");
    expect(loader).toHaveAttribute("role", "status");
    expect(loader).toHaveAttribute("aria-live", "polite");
  });
});
