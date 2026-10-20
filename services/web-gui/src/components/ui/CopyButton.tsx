import { useCopy } from "../../hooks/useCopy";
import styles from "./CopyButton.module.css";

interface CopyButtonProps {
  /** The text that lands on the clipboard. */
  text: string;
  /** Idle label; the button still names what it copies for a screen reader. */
  label?: string;
  /** What is copied, for the accessible name ("Sao chép prompt nghiên cứu"). */
  what?: string;
  disabled?: boolean;
  testId?: string;
}

function CopyIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <rect x="9" y="9" width="12" height="12" rx="2" />
      <path d="M5 15H4a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1h10a1 1 0 0 1 1 1v1" />
    </svg>
  );
}

function CheckIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M5 12.5l4.5 4.5L19 7.5" />
    </svg>
  );
}

/**
 * The small quick-copy button that sits at the top-left of a block of text the
 * Creator may want to paste elsewhere (docs/ux-ui-design-rules.md, rule 8).
 * It copies in one click, without opening anything first.
 */
export function CopyButton({ text, label = "Sao chép", what, disabled, testId }: CopyButtonProps) {
  const { state, copy } = useCopy();
  const text_ = state === "copied" ? "Đã chép" : state === "failed" ? "Không chép được" : label;
  const cls = [styles.copyBtn, state === "copied" ? styles.done : "", state === "failed" ? styles.failed : ""]
    .filter(Boolean)
    .join(" ");
  return (
    <button
      type="button"
      className={cls}
      disabled={disabled || text === ""}
      onClick={() => void copy(text)}
      aria-label={what ? `${label} ${what}` : label}
      title={what ? `${label} ${what}` : label}
      data-testid={testId}
    >
      {state === "copied" ? <CheckIcon /> : <CopyIcon />}
      <span role="status">{text_}</span>
    </button>
  );
}
