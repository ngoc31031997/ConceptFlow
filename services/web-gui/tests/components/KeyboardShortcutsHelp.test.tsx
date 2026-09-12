import { describe, it, expect } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { KeyboardShortcutsHelp } from "../../src/components/KeyboardShortcutsHelp";

describe("KeyboardShortcutsHelp", () => {
  it("renders trigger button initially", () => {
    render(<KeyboardShortcutsHelp />);
    expect(screen.getByTestId("keyboard-shortcuts-trigger")).toBeInTheDocument();
  });

  it("opens overlay when trigger button clicked", () => {
    render(<KeyboardShortcutsHelp />);

    fireEvent.click(screen.getByTestId("keyboard-shortcuts-trigger"));
    expect(screen.getByTestId("keyboard-shortcuts-overlay")).toBeInTheDocument();
  });

  it("displays shortcuts list when open", () => {
    render(<KeyboardShortcutsHelp />);

    fireEvent.click(screen.getByTestId("keyboard-shortcuts-trigger"));

    expect(screen.getByText(/Tiếp tục \/ Submit form/)).toBeInTheDocument();
    expect(screen.getByText(/Đóng modal \/ Dialog/)).toBeInTheDocument();
  });

  it("closes overlay when backdrop clicked", () => {
    render(<KeyboardShortcutsHelp />);

    fireEvent.click(screen.getByTestId("keyboard-shortcuts-trigger"));
    expect(screen.getByTestId("keyboard-shortcuts-overlay")).toBeInTheDocument();

    fireEvent.click(screen.getByTestId("keyboard-shortcuts-overlay"));
    expect(screen.queryByTestId("keyboard-shortcuts-overlay")).not.toBeInTheDocument();
  });

  it("closes overlay when close button clicked", () => {
    render(<KeyboardShortcutsHelp />);

    fireEvent.click(screen.getByTestId("keyboard-shortcuts-trigger"));
    const closeButton = screen.getByLabelText("Đóng");

    fireEvent.click(closeButton);
    expect(screen.queryByTestId("keyboard-shortcuts-overlay")).not.toBeInTheDocument();
  });

  it("does not close when modal content clicked", () => {
    render(<KeyboardShortcutsHelp />);

    fireEvent.click(screen.getByTestId("keyboard-shortcuts-trigger"));
    const modal = screen.getByRole("heading", { name: /Keyboard Shortcuts/ }).closest("div");

    if (modal) {
      fireEvent.click(modal);
      expect(screen.getByTestId("keyboard-shortcuts-overlay")).toBeInTheDocument();
    }
  });

  it("has proper accessibility structure", () => {
    render(<KeyboardShortcutsHelp />);

    fireEvent.click(screen.getByTestId("keyboard-shortcuts-trigger"));

    expect(screen.getByRole("heading", { name: /Keyboard Shortcuts/ })).toBeInTheDocument();
    expect(screen.getByLabelText("Đóng")).toBeInTheDocument();
  });
});
