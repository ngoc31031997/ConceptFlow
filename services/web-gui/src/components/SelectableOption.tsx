import type { ReactNode } from "react";
import selectable from "../styles/selectable.module.css";

interface SelectableOptionProps {
  selected: boolean;
  onSelect: () => void;
  label: ReactNode;
  hint?: ReactNode;
  /** Rendered before the label — a flag, an icon. */
  leading?: ReactNode;
  compact?: boolean;
  /** Centred single-line choice (a row) rather than a labelled card. */
  inline?: boolean;
  testId?: string;
  ariaLabel?: string;
}

function CheckIcon() {
  return (
    <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="4" strokeLinecap="round" strokeLinejoin="round">
      <path d="M5 13l4 4L19 7" />
    </svg>
  );
}

/**
 * One choice among several, with the selected state spelled out rather than
 * implied: a tinted fill, an accent ring and a check badge, plus aria-checked
 * for anyone not looking at the colours at all.
 */
export function SelectableOption({
  selected,
  onSelect,
  label,
  hint,
  leading,
  compact,
  inline,
  testId,
  ariaLabel,
}: SelectableOptionProps) {
  const className = [
    selectable.option,
    selected ? selectable.selected : "",
    compact ? selectable.compact : "",
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <button
      type="button"
      role="radio"
      aria-checked={selected}
      aria-pressed={selected}
      aria-label={ariaLabel}
      data-testid={testId}
      className={className}
      onClick={onSelect}
    >
      <span className={selectable.check} aria-hidden="true">
        {selected && <CheckIcon />}
      </span>
      {leading}
      {inline ? (
        <span className={selectable.label}>{label}</span>
      ) : (
        <span className={selectable.body}>
          <span className={selectable.label}>{label}</span>
          {hint && <span className={selectable.hint}>{hint}</span>}
        </span>
      )}
    </button>
  );
}
