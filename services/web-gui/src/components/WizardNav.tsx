import type { ReactNode } from "react";
import { useKeyboardShortcuts, formatShortcut } from "../hooks/useKeyboardShortcuts";
import glass from "../styles/glass.module.css";
import styles from "./WizardNav.module.css";

interface WizardNavProps {
  /** Left-hand explanation of why the primary action is or is not available. */
  hint: ReactNode;
  isBlocked?: boolean;
  onBack?: () => void;
  backLabel?: string;
  onNext: () => void;
  nextLabel: string;
  nextDisabled?: boolean;
  nextTestId?: string;
  /** Rendered between the hint and the primary button — e.g. "Xem lỗi". */
  extraAction?: ReactNode;
}

function ArrowRight() {
  return (
    <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M5 12h14M13 6l6 6-6 6" />
    </svg>
  );
}

function ArrowLeft() {
  return (
    <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M19 12H5M11 18l-6-6 6-6" />
    </svg>
  );
}

/**
 * The bar every wizard step ends with, pinned to the bottom of the window so
 * the way forward is reachable no matter how long the step's content is.
 */
export function WizardNav({
  hint,
  isBlocked,
  onBack,
  backLabel = "Quay lại",
  onNext,
  nextLabel,
  nextDisabled,
  nextTestId,
  extraAction,
}: WizardNavProps) {
  // Keyboard shortcut for Next button (Ctrl+Enter or ⌘+Enter)
  useKeyboardShortcuts([
    {
      key: 'Enter',
      ctrlKey: true,
      action: () => {
        if (!nextDisabled) {
          onNext();
        }
      },
      description: 'Submit wizard step',
    },
  ], !nextDisabled);

  const nextTooltip = formatShortcut({ key: 'Enter', ctrlKey: true });

  return (
    <div className={styles.bar}>
      <div className={styles.inner}>
        {onBack && (
          <button type="button" className={glass.ghostBtn} onClick={onBack} data-testid="wizard-back">
            <ArrowLeft />
            {backLabel}
          </button>
        )}
        <p
          className={`${styles.hint} ${isBlocked ? styles.hintBlocked : ""}`}
          role={isBlocked ? "alert" : "status"}
        >
          {hint}
        </p>
        {extraAction}
        <button
          type="button"
          className={glass.btnPrimary}
          onClick={onNext}
          disabled={nextDisabled}
          data-testid={nextTestId}
          title={`${nextLabel} (${nextTooltip})`}
        >
          {nextLabel}
          <ArrowRight />
        </button>
      </div>
    </div>
  );
}
