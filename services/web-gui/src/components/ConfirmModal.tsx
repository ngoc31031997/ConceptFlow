import { useRef, type ReactNode } from "react";
import { formatShortcut } from "../hooks/useKeyboardShortcuts";
import styles from "./ConfirmModal.module.css";
import { Button, Dialog } from "./ui";

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
 * A confirmation in the shared dialog shell, in place of window.confirm().
 * Focus starts on the confirm button (Enter confirms, Esc cancels), and a
 * click on the backdrop cancels since there is nothing typed to lose.
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

  const handleConfirm = () => {
    onConfirm();
    onClose();
  };

  const escTooltip = formatShortcut({ key: "Escape" });
  const enterTooltip = formatShortcut({ key: "Enter" });

  return (
    <Dialog
      open={isOpen}
      onClose={onClose}
      role="alertdialog"
      size="sm"
      title={title}
      titleId="modal-title"
      describedBy="modal-message"
      testId="confirm-modal"
      backdropTestId="confirm-modal-backdrop"
      initialFocusRef={confirmButtonRef}
      icon={
        isDangerous ? (
          <div className={styles.iconDanger} aria-hidden="true">
            <AlertIcon />
          </div>
        ) : undefined
      }
      footer={
        <>
          <Button
            variant="ghost"
            onClick={onClose}
            data-testid="confirm-modal-cancel"
            title={`${cancelLabel} (${escTooltip})`}
          >
            {cancelLabel}
          </Button>
          <Button
            ref={confirmButtonRef}
            variant={isDangerous ? "danger" : "primary"}
            onClick={handleConfirm}
            data-testid="confirm-modal-confirm"
            title={`${confirmLabel} (${enterTooltip})`}
          >
            {confirmLabel}
          </Button>
        </>
      }
    >
      <p id="modal-message" className={styles.message}>
        {message}
      </p>
    </Dialog>
  );
}
