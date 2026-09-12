import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { ConfirmModal } from "../../src/components/ConfirmModal";

describe("ConfirmModal", () => {
  const defaultProps = {
    isOpen: true,
    onClose: vi.fn(),
    onConfirm: vi.fn(),
    title: "Test Modal",
    message: "This is a test message",
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    // Clean up any modals that might be left in the DOM
    document.body.style.overflow = "";
  });

  it("renders nothing when isOpen is false", () => {
    render(<ConfirmModal {...defaultProps} isOpen={false} />);
    expect(screen.queryByTestId("confirm-modal")).not.toBeInTheDocument();
  });

  it("renders modal with title and message when open", () => {
    render(<ConfirmModal {...defaultProps} />);
    expect(screen.getByText("Test Modal")).toBeInTheDocument();
    expect(screen.getByText("This is a test message")).toBeInTheDocument();
  });

  it("calls onConfirm and onClose when confirm button clicked", () => {
    render(<ConfirmModal {...defaultProps} />);
    fireEvent.click(screen.getByTestId("confirm-modal-confirm"));
    expect(defaultProps.onConfirm).toHaveBeenCalledTimes(1);
    expect(defaultProps.onClose).toHaveBeenCalledTimes(1);
  });

  it("calls onClose when cancel button clicked", () => {
    render(<ConfirmModal {...defaultProps} />);
    fireEvent.click(screen.getByTestId("confirm-modal-cancel"));
    expect(defaultProps.onClose).toHaveBeenCalledTimes(1);
    expect(defaultProps.onConfirm).not.toHaveBeenCalled();
  });

  it("calls onClose when backdrop clicked", () => {
    render(<ConfirmModal {...defaultProps} />);
    fireEvent.click(screen.getByTestId("confirm-modal-backdrop"));
    expect(defaultProps.onClose).toHaveBeenCalledTimes(1);
  });

  it("does not close when modal content clicked", () => {
    render(<ConfirmModal {...defaultProps} />);
    fireEvent.click(screen.getByTestId("confirm-modal"));
    expect(defaultProps.onClose).not.toHaveBeenCalled();
  });

  it("calls onClose when ESC key pressed", () => {
    render(<ConfirmModal {...defaultProps} />);
    fireEvent.keyDown(document, { key: "Escape" });
    expect(defaultProps.onClose).toHaveBeenCalledTimes(1);
  });

  it("shows custom button labels", () => {
    render(
      <ConfirmModal
        {...defaultProps}
        confirmLabel="Custom Confirm"
        cancelLabel="Custom Cancel"
      />
    );
    expect(screen.getByText("Custom Confirm")).toBeInTheDocument();
    expect(screen.getByText("Custom Cancel")).toBeInTheDocument();
  });

  it("applies danger styling when isDangerous is true", () => {
    render(<ConfirmModal {...defaultProps} isDangerous />);
    const confirmButton = screen.getByTestId("confirm-modal-confirm");
    expect(confirmButton.className).toContain("dangerBtn");
  });

  it("applies primary styling when isDangerous is false", () => {
    render(<ConfirmModal {...defaultProps} isDangerous={false} />);
    const confirmButton = screen.getByTestId("confirm-modal-confirm");
    expect(confirmButton.className).toContain("btnPrimary");
  });

  it("prevents body scroll when open", () => {
    render(<ConfirmModal {...defaultProps} />);
    expect(document.body.style.overflow).toBe("hidden");
  });

  it("restores body scroll when closed", () => {
    const { rerender } = render(<ConfirmModal {...defaultProps} />);
    expect(document.body.style.overflow).toBe("hidden");
    
    rerender(<ConfirmModal {...defaultProps} isOpen={false} />);
    expect(document.body.style.overflow).toBe("");
  });

  it("has proper ARIA attributes", () => {
    render(<ConfirmModal {...defaultProps} />);
    const modal = screen.getByTestId("confirm-modal");
    expect(modal).toHaveAttribute("role", "alertdialog");
    expect(modal).toHaveAttribute("aria-labelledby", "modal-title");
    expect(modal).toHaveAttribute("aria-describedby", "modal-message");
  });
});
