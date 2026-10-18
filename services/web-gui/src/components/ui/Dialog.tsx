import { useEffect, useId, useRef, type MouseEvent, type ReactNode, type RefObject } from "react";
import { createPortal } from "react-dom";
import { usePresence } from "../../hooks/usePresence";
import styles from "./Dialog.module.css";

const FOCUSABLE =
  'button:not(:disabled), [href], input:not(:disabled), select:not(:disabled), textarea:not(:disabled), [tabindex]:not([tabindex="-1"])';
const EXIT_MS = 150;

export interface DialogProps {
  open: boolean;
  /** Called by ×, Esc and (for a dialog without input) a click on the backdrop. */
  onClose: () => void;
  /** Names the item and the task ("Sửa tay · Creative brief"). */
  title: ReactNode;
  /** Shown before the title (the warning sign of a dangerous confirmation). */
  icon?: ReactNode;
  /** Fields first, then the impact of the action. */
  children: ReactNode;
  /** The buttons: "Huỷ" first, the main action last. */
  footer: ReactNode;
  /**
   * The dialog holds a form: a click on the backdrop then does not close it,
   * so a stray click cannot lose typed text. ×, "Huỷ" and Esc still do.
   */
  hasInput?: boolean;
  /** Takes focus on open. Default: the first control in the body, else the main action. */
  initialFocusRef?: RefObject<HTMLElement>;
  role?: "dialog" | "alertdialog";
  size?: "sm" | "md" | "lg";
  titleId?: string;
  describedBy?: string;
  testId?: string;
  backdropTestId?: string;
}

/**
 * The one modal shell in the app (docs/ux-ui-design-rules.md, rule 4): a
 * portal over the page, focus moved inside and kept there with Tab, Esc and ×
 * to close, the page behind locked from scrolling, and focus handed back to
 * the control that opened it. It fades and rises in, and fades out while
 * `usePresence` keeps it mounted.
 */
export function Dialog({
  open,
  onClose,
  title,
  icon,
  children,
  footer,
  hasInput = false,
  initialFocusRef,
  role = "dialog",
  size = "md",
  titleId,
  describedBy,
  testId,
  backdropTestId,
}: DialogProps) {
  const { mounted, closing } = usePresence(open, EXIT_MS);
  const generatedId = useId();
  const headingId = titleId ?? `${generatedId}-title`;
  const dialogRef = useRef<HTMLDivElement>(null);
  const bodyRef = useRef<HTMLDivElement>(null);
  const footerRef = useRef<HTMLDivElement>(null);
  // A ref, so a re-render of the page behind (typing, a poll landing) does not
  // re-run the open effect and pull focus back to the first field.
  const onCloseRef = useRef(onClose);
  useEffect(() => {
    onCloseRef.current = onClose;
  }, [onClose]);

  useEffect(() => {
    if (!open) return;
    const opener = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    const footerButtons = footerRef.current?.querySelectorAll<HTMLElement>(FOCUSABLE);
    const target =
      initialFocusRef?.current ??
      bodyRef.current?.querySelector<HTMLElement>(FOCUSABLE) ??
      (footerButtons && footerButtons.length > 0 ? footerButtons[footerButtons.length - 1] : null);
    target?.focus();

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        onCloseRef.current();
        return;
      }
      if (e.key !== "Tab" || !dialogRef.current) return;
      const items = dialogRef.current.querySelectorAll<HTMLElement>(FOCUSABLE);
      if (items.length === 0) return;
      const first = items[0];
      const last = items[items.length - 1];
      if (e.shiftKey && document.activeElement === first) {
        e.preventDefault();
        last.focus();
      } else if (!e.shiftKey && document.activeElement === last) {
        e.preventDefault();
        first.focus();
      }
    };
    document.addEventListener("keydown", handleKeyDown);
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    return () => {
      document.removeEventListener("keydown", handleKeyDown);
      document.body.style.overflow = previousOverflow;
      if (opener && document.contains(opener)) opener.focus();
    };
  }, [open, initialFocusRef]);

  if (!mounted) return null;

  const handleBackdropClick = (e: MouseEvent) => {
    if (!hasInput && e.target === e.currentTarget) onClose();
  };

  return createPortal(
    <div
      className={closing ? `${styles.backdrop} ${styles.closing}` : styles.backdrop}
      onClick={handleBackdropClick}
      role="presentation"
      data-testid={backdropTestId}
    >
      <div
        ref={dialogRef}
        className={`${styles.dialog} ${styles[size]}`}
        role={role}
        aria-modal="true"
        aria-labelledby={headingId}
        aria-describedby={describedBy}
        data-testid={testId}
      >
        <div className={styles.header}>
          {icon}
          <h2 id={headingId} className={styles.title}>
            {title}
          </h2>
          <button
            type="button"
            className={styles.close}
            onClick={onClose}
            aria-label="Đóng"
            data-testid={testId ? `${testId}-close` : undefined}
          >
            ×
          </button>
        </div>
        <div ref={bodyRef} className={styles.body}>
          {children}
        </div>
        <div ref={footerRef} className={styles.footer}>
          {footer}
        </div>
      </div>
    </div>,
    document.body,
  );
}
