import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook } from "@testing-library/react";
import { useKeyboardShortcuts, formatShortcut } from "../../src/hooks/useKeyboardShortcuts";

describe("useKeyboardShortcuts", () => {
  let mockAction: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    mockAction = vi.fn();
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it("triggers action on matching key", () => {
    renderHook(() =>
      useKeyboardShortcuts([
        { key: "Enter", action: mockAction, description: "Submit" },
      ])
    );

    window.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter" }));
    expect(mockAction).toHaveBeenCalledTimes(1);
  });

  it("triggers action with Ctrl modifier", () => {
    renderHook(() =>
      useKeyboardShortcuts([
        { key: "s", ctrlKey: true, action: mockAction, description: "Save" },
      ])
    );

    window.dispatchEvent(new KeyboardEvent("keydown", { key: "s", ctrlKey: true }));
    expect(mockAction).toHaveBeenCalledTimes(1);
  });

  it("triggers action with Meta modifier (Mac)", () => {
    renderHook(() =>
      useKeyboardShortcuts([
        { key: "s", metaKey: true, action: mockAction, description: "Save" },
      ])
    );

    window.dispatchEvent(new KeyboardEvent("keydown", { key: "s", metaKey: true }));
    expect(mockAction).toHaveBeenCalledTimes(1);
  });

  it("does not trigger without required modifier", () => {
    renderHook(() =>
      useKeyboardShortcuts([
        { key: "s", ctrlKey: true, action: mockAction, description: "Save" },
      ])
    );

    window.dispatchEvent(new KeyboardEvent("keydown", { key: "s" }));
    expect(mockAction).not.toHaveBeenCalled();
  });

  it("does not trigger in input fields", () => {
    const input = document.createElement("input");
    document.body.appendChild(input);

    renderHook(() =>
      useKeyboardShortcuts([
        { key: "Enter", ctrlKey: true, action: mockAction, description: "Submit" },
      ])
    );

    const event = new KeyboardEvent("keydown", { key: "Enter", ctrlKey: true, bubbles: true });
    Object.defineProperty(event, "target", { value: input, enumerable: true });
    window.dispatchEvent(event);

    expect(mockAction).not.toHaveBeenCalled();

    document.body.removeChild(input);
  });

  it("allows Escape to work in input fields", () => {
    const input = document.createElement("input");
    document.body.appendChild(input);

    renderHook(() =>
      useKeyboardShortcuts([
        { key: "Escape", action: mockAction, description: "Close" },
      ])
    );

    const event = new KeyboardEvent("keydown", { key: "Escape", bubbles: true });
    Object.defineProperty(event, "target", { value: input, enumerable: true });
    window.dispatchEvent(event);

    expect(mockAction).toHaveBeenCalledTimes(1);

    document.body.removeChild(input);
  });

  it("respects enabled flag", () => {
    const { rerender } = renderHook(
      ({ enabled }) =>
        useKeyboardShortcuts(
          [{ key: "Enter", action: mockAction, description: "Submit" }],
          enabled
        ),
      { initialProps: { enabled: true } }
    );

    window.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter" }));
    expect(mockAction).toHaveBeenCalledTimes(1);

    mockAction.mockClear();
    rerender({ enabled: false });

    window.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter" }));
    expect(mockAction).not.toHaveBeenCalled();
  });

  it("prevents default behavior by default", () => {
    renderHook(() =>
      useKeyboardShortcuts([
        { key: "s", ctrlKey: true, action: mockAction, description: "Save" },
      ])
    );

    const event = new KeyboardEvent("keydown", { key: "s", ctrlKey: true });
    const preventDefaultSpy = vi.spyOn(event, "preventDefault");
    
    window.dispatchEvent(event);

    expect(preventDefaultSpy).toHaveBeenCalled();
  });

  it("does not prevent default when preventDefault is false", () => {
    renderHook(() =>
      useKeyboardShortcuts([
        {
          key: "Enter",
          action: mockAction,
          description: "Submit",
          preventDefault: false,
        },
      ])
    );

    const event = new KeyboardEvent("keydown", { key: "Enter" });
    const preventDefaultSpy = vi.spyOn(event, "preventDefault");

    window.dispatchEvent(event);

    expect(preventDefaultSpy).not.toHaveBeenCalled();
    expect(mockAction).toHaveBeenCalled();
  });

  it("cleans up event listeners on unmount", () => {
    const removeEventListenerSpy = vi.spyOn(window, "removeEventListener");

    const { unmount } = renderHook(() =>
      useKeyboardShortcuts([
        { key: "Enter", action: mockAction, description: "Submit" },
      ])
    );

    unmount();

    expect(removeEventListenerSpy).toHaveBeenCalledWith("keydown", expect.any(Function));

    removeEventListenerSpy.mockRestore();
  });
});

describe("formatShortcut", () => {
  const originalPlatform = navigator.platform;

  afterEach(() => {
    Object.defineProperty(navigator, "platform", {
      value: originalPlatform,
      configurable: true,
    });
  });

  it("formats Ctrl key on Windows/Linux", () => {
    Object.defineProperty(navigator, "platform", {
      value: "Win32",
      configurable: true,
    });

    expect(formatShortcut({ key: "s", ctrlKey: true })).toBe("Ctrl+S");
  });

  it("formats ⌘ key on Mac", () => {
    Object.defineProperty(navigator, "platform", {
      value: "MacIntel",
      configurable: true,
    });

    expect(formatShortcut({ key: "s", ctrlKey: true })).toBe("⌘S");
  });

  it("formats Shift modifier", () => {
    expect(formatShortcut({ key: "Tab", shiftKey: true })).toContain("Shift");
  });

  it("formats Alt modifier", () => {
    expect(formatShortcut({ key: "F4", altKey: true })).toContain("Alt");
  });

  it("formats complex shortcuts", () => {
    Object.defineProperty(navigator, "platform", {
      value: "Win32",
      configurable: true,
    });

    expect(formatShortcut({ key: "s", ctrlKey: true, shiftKey: true })).toBe("Ctrl+Shift+S");
  });

  it("formats single letter keys in uppercase", () => {
    expect(formatShortcut({ key: "a" })).toBe("A");
  });

  it("formats special keys with proper casing", () => {
    expect(formatShortcut({ key: "enter" })).toBe("Enter");
    expect(formatShortcut({ key: "escape" })).toBe("Escape");
  });

  it("formats space key", () => {
    expect(formatShortcut({ key: " " })).toBe("Space");
  });
});
