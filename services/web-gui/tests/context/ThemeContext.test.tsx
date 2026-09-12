import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { ThemeProvider, useTheme } from "../../src/context/ThemeContext";

describe("ThemeContext", () => {
  beforeEach(() => {
    localStorage.clear();
    document.documentElement.removeAttribute("data-theme");
  });

  afterEach(() => {
    localStorage.clear();
  });

  it("provides theme context", () => {
    const { result } = renderHook(() => useTheme(), {
      wrapper: ThemeProvider,
    });

    expect(result.current.theme).toBeDefined();
    expect(result.current.toggleTheme).toBeDefined();
  });

  it("throws error when used outside provider", () => {
    // Suppress console.error for this test
    const spy = vi.spyOn(console, "error").mockImplementation(() => {});

    expect(() => {
      renderHook(() => useTheme());
    }).toThrow("useTheme must be used within ThemeProvider");

    spy.mockRestore();
  });

  it("defaults to light theme", () => {
    const { result } = renderHook(() => useTheme(), {
      wrapper: ThemeProvider,
    });

    expect(result.current.theme).toBe("light");
  });

  it("respects stored theme preference", () => {
    localStorage.setItem("conceptflow-theme", "dark");

    const { result } = renderHook(() => useTheme(), {
      wrapper: ThemeProvider,
    });

    expect(result.current.theme).toBe("dark");
  });

  it("toggles theme", () => {
    const { result } = renderHook(() => useTheme(), {
      wrapper: ThemeProvider,
    });

    expect(result.current.theme).toBe("light");

    act(() => {
      result.current.toggleTheme();
    });

    expect(result.current.theme).toBe("dark");

    act(() => {
      result.current.toggleTheme();
    });

    expect(result.current.theme).toBe("light");
  });

  it("persists theme to localStorage", () => {
    const { result } = renderHook(() => useTheme(), {
      wrapper: ThemeProvider,
    });

    act(() => {
      result.current.toggleTheme();
    });

    expect(localStorage.getItem("conceptflow-theme")).toBe("dark");
  });

  it("sets data-theme attribute on document", () => {
    const { result } = renderHook(() => useTheme(), {
      wrapper: ThemeProvider,
    });

    expect(document.documentElement.getAttribute("data-theme")).toBe("light");

    act(() => {
      result.current.toggleTheme();
    });

    expect(document.documentElement.getAttribute("data-theme")).toBe("dark");
  });

  it("respects system dark mode preference when no stored value", () => {
    Object.defineProperty(window, "matchMedia", {
      writable: true,
      value: vi.fn().mockImplementation((query) => ({
        matches: query === "(prefers-color-scheme: dark)",
        media: query,
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
      })),
    });

    const { result } = renderHook(() => useTheme(), {
      wrapper: ThemeProvider,
    });

    expect(result.current.theme).toBe("dark");
  });
});
