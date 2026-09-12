import { describe, it, expect } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { ThemeToggle } from "../../src/components/ThemeToggle";
import { ThemeProvider } from "../../src/context/ThemeContext";

describe("ThemeToggle", () => {
  it("renders theme toggle button", () => {
    render(
      <ThemeProvider>
        <ThemeToggle />
      </ThemeProvider>
    );

    expect(screen.getByTestId("theme-toggle")).toBeInTheDocument();
  });

  it("shows moon icon in light mode", () => {
    render(
      <ThemeProvider>
        <ThemeToggle />
      </ThemeProvider>
    );

    const button = screen.getByTestId("theme-toggle");
    expect(button).toHaveAttribute("aria-label", "Chuyển sang chế độ tối");
  });

  it("toggles theme on click", () => {
    render(
      <ThemeProvider>
        <ThemeToggle />
      </ThemeProvider>
    );

    const button = screen.getByTestId("theme-toggle");

    // Initially light mode
    expect(button).toHaveAttribute("aria-label", "Chuyển sang chế độ tối");

    // Click to switch to dark
    fireEvent.click(button);
    expect(button).toHaveAttribute("aria-label", "Chuyển sang chế độ sáng");

    // Click again to switch back to light
    fireEvent.click(button);
    expect(button).toHaveAttribute("aria-label", "Chuyển sang chế độ tối");
  });

  it("has proper accessibility attributes", () => {
    render(
      <ThemeProvider>
        <ThemeToggle />
      </ThemeProvider>
    );

    const button = screen.getByTestId("theme-toggle");
    expect(button).toHaveAttribute("aria-label");
    expect(button).toHaveAttribute("title");
    expect(button.tagName).toBe("BUTTON");
    expect(button).toHaveAttribute("type", "button");
  });

  it("updates document data-theme attribute", () => {
    render(
      <ThemeProvider>
        <ThemeToggle />
      </ThemeProvider>
    );

    const button = screen.getByTestId("theme-toggle");

    expect(document.documentElement.getAttribute("data-theme")).toBe("light");

    fireEvent.click(button);
    expect(document.documentElement.getAttribute("data-theme")).toBe("dark");

    fireEvent.click(button);
    expect(document.documentElement.getAttribute("data-theme")).toBe("light");
  });
});
