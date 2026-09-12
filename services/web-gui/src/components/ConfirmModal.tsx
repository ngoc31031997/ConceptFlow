import { useEffect, useRef, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { formatShortcut } from "../hooks/useKeyboardShortcuts";
import glass from "../styles/glass.module.css";
import styles from "./ConfirmModal.module.css";

interface ConfirmModalProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: () => void;
  title: string;
  message: ReactNode;
  confirmLabel?: string;
  cancelLabel?: string;
  isDangerous?: boolean;
}

function AlertIcon() {
  return (
    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" />
      <line x1="12" y1="9" x2="12" y2="13" />
      <line x1="12" y1="17" x2="12.01" y2="17" />
    </svg>
  );
}

/**
 * Custom confirmation modal with glassmorphism design to replace native
 * window.confirm(). Supports keyboard shortcuts (ESC to cancel, Enter to
 * confirm) and focus trapping for accessibility.
 */
export function ConfirmModal({
  isOpen,
  onClose,
  onConfirm,
  title,
  message,
  confirmLabel = "Xác nhận",
  cancelLabel = "Hủy",
  isDangerous = false,
}: ConfirmModalProps) {
  const confirmButtonRef = useRef<HTMLButtonElement>(null);
  const modalRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!isOpen) return;

    // Focus the confirm button when modal opens
    confirmButtonRef.current?.focus();

    // Handle ESC key to cancel
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        onClose();
      }
    };

    // Trap focus within modal
    const handleFocusTrap = (e: KeyboardEvent) => {
      if (e.key !== "Tab" || !modalRef.current) return;

      const focusableElements = modalRef.current.querySelectorAll<HTMLElement>(
        'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
      );
      const firstElement = focusableElements[0];
      const lastElement = focusableElements[focusableElements.length - 1];

      if (e.shiftKey && document.activeElement === firstElement) {
        e.preventDefault();
        lastElement?.focus();
      } else if (!e.shiftKey && document.activeElement === lastElement) {
        e.preventDefault();
        firstElement?.focus();
      }
    };

    document.addEventListener("keydown", handleKeyDown);
    document.addEventListener("keydown", handleFocusTrap);

    // Prevent body scroll when modal is open
    document.body.style.overflow = "hidden";

    return () => {
      document.removeEventListener("keydown", handleKeyDown);
      document.removeEventListener("keydown", handleFocusTrap);
      document.body.style.overflow = "";
    };
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  const handleBackdropClick = (e: React.MouseEvent) => {
    if (e.target === e.currentTarget) {
      onClose();
    }
  };

  const handleConfirm = () => {
    onConfirm();
    onClose();
  };

  const escTooltip = formatShortcut({ key: 'Escape' });
  const enterTooltip = formatShortcut({ key: 'Enter' });

  return createPortal(
    <div
      className={styles.backdrop}
      onClick={handleBackdropClick}
      role="presentation"
      data-testid="confirm-modal-backdrop"
    >
      <div
        ref={modalRef}
        className={`${glass.card} ${styles.modal}`}
        role="alertdialog"
        aria-labelledby="modal-title"
        aria-describedby="modal-message"
        data-testid="confirm-modal"
      >
        <div className={styles.header}>
          {isDangerous && (
            <div className={styles.iconDanger} aria-hidden="true">
              <AlertIcon />
            </div>
          )}
          <h2 id="modal-title" className={styles.title}>
            {title}
          </h2>
        </div>

        <p id="modal-message" className={styles.message}>
          {message}
        </p>

        <div className={styles.actions}>
          <button
            type="button"
            className={glass.ghostBtn}
            onClick={onClose}
            data-testid="confirm-modal-cancel"
            title={`${cancelLabel} (${escTooltip})`}
          >
            {cancelLabel}
          </button>
          <button
            ref={confirmButtonRef}
            type="button"
            className={isDangerous ? glass.dangerBtn : glass.btnPrimary}
            onClick={handleConfirm}
            data-testid="confirm-modal-confirm"
            title={`${confirmLabel} (${enterTooltip})`}
          >
            {confirmLabel}
          </button>
        </div>
      </div>
    </div>,
    document.body
  );
}
