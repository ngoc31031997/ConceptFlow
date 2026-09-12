import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useDebounce } from "../../src/hooks/useDebounce";

describe("useDebounce", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("returns initial value immediately", () => {
    const { result } = renderHook(() => useDebounce("initial", 500));
    expect(result.current).toBe("initial");
  });

  it("does not update value before delay", () => {
    const { result, rerender } = renderHook(({ value }) => useDebounce(value, 500), {
      initialProps: { value: "initial" },
    });

    rerender({ value: "updated" });
    act(() => {
      vi.advanceTimersByTime(400);
    });
    expect(result.current).toBe("initial");
  });

  it("updates value after delay", () => {
    const { result, rerender } = renderHook(({ value }) => useDebounce(value, 500), {
      initialProps: { value: "initial" },
    });

    rerender({ value: "updated" });
    act(() => {
      vi.advanceTimersByTime(500);
    });
    expect(result.current).toBe("updated");
  });

  it("resets timer on rapid changes", () => {
    const { result, rerender } = renderHook(({ value }) => useDebounce(value, 500), {
      initialProps: { value: "initial" },
    });

    // First change
    rerender({ value: "change1" });
    act(() => {
      vi.advanceTimersByTime(300);
    });

    // Second change before first timer completes
    rerender({ value: "change2" });
    act(() => {
      vi.advanceTimersByTime(300);
    });

    // Still showing initial value because timer was reset
    expect(result.current).toBe("initial");

    // Wait remaining time
    act(() => {
      vi.advanceTimersByTime(200);
    });
    expect(result.current).toBe("change2");
  });

  it("handles multiple rapid changes correctly", () => {
    const { result, rerender } = renderHook(({ value }) => useDebounce(value, 500), {
      initialProps: { value: "v0" },
    });

    rerender({ value: "v1" });
    act(() => {
      vi.advanceTimersByTime(100);
    });

    rerender({ value: "v2" });
    act(() => {
      vi.advanceTimersByTime(100);
    });

    rerender({ value: "v3" });
    act(() => {
      vi.advanceTimersByTime(100);
    });

    rerender({ value: "v4" });
    act(() => {
      vi.advanceTimersByTime(100);
    });

    // Still at initial value
    expect(result.current).toBe("v0");

    // Final value after delay from last change
    act(() => {
      vi.advanceTimersByTime(400);
    });
    expect(result.current).toBe("v4");
  });

  it("uses custom delay", () => {
    const { result, rerender } = renderHook(({ value }) => useDebounce(value, 1000), {
      initialProps: { value: "initial" },
    });

    rerender({ value: "updated" });
    act(() => {
      vi.advanceTimersByTime(500);
    });
    expect(result.current).toBe("initial");

    act(() => {
      vi.advanceTimersByTime(500);
    });
    expect(result.current).toBe("updated");
  });

  it("works with different data types", () => {
    // Number
    const { result: numberResult, rerender: numberRerender } = renderHook(
      ({ value }) => useDebounce(value, 500),
      { initialProps: { value: 0 } }
    );
    numberRerender({ value: 42 });
    act(() => {
      vi.advanceTimersByTime(500);
    });
    expect(numberResult.current).toBe(42);

    // Boolean
    const { result: boolResult, rerender: boolRerender } = renderHook(
      ({ value }) => useDebounce(value, 500),
      { initialProps: { value: false } }
    );
    boolRerender({ value: true });
    act(() => {
      vi.advanceTimersByTime(500);
    });
    expect(boolResult.current).toBe(true);

    // Object
    const obj1 = { id: 1 };
    const obj2 = { id: 2 };
    const { result: objResult, rerender: objRerender } = renderHook(
      ({ value }) => useDebounce(value, 500),
      { initialProps: { value: obj1 } }
    );
    objRerender({ value: obj2 });
    act(() => {
      vi.advanceTimersByTime(500);
    });
    expect(objResult.current).toBe(obj2);
  });

  it("cleans up timeout on unmount", () => {
    const clearTimeoutSpy = vi.spyOn(global, "clearTimeout");
    const { unmount } = renderHook(() => useDebounce("value", 500));
    unmount();
    expect(clearTimeoutSpy).toHaveBeenCalled();
    clearTimeoutSpy.mockRestore();
  });
});
