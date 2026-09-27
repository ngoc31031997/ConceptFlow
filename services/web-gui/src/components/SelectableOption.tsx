import { useLayoutEffect, useRef, type ReactNode } from "react";
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

/** Shrinks the font until the text fits its box instead of overflowing it. */
function useShrinkToFit<T extends HTMLElement>(dep: unknown, enabled: boolean) {
  const ref = useRef<T>(null);
  useLayoutEffect(() => {
    const el = ref.current;
    if (!el || !enabled) return;
    const fit = () => {
      el.style.fontSize = "";
      const base = parseFloat(getComputedStyle(el).fontSize);
      let size = base;
      while (el.scrollWidth > el.clientWidth && size > 8) {
        size -= 0.5;
        el.style.fontSize = `${size}px`;
      }
    };
    fit();
    const ro = new ResizeObserver(fit);
    ro.observe(el);
    return () => ro.disconnect();
  }, [dep, enabled]);
  return ref;
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
  const labelRef = useShrinkToFit<HTMLSpanElement>(label, !!inline);
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
      // aria-checked is the one a radio takes; aria-pressed alongside it made
      // the same element announce as both a radio and a toggle button.
      aria-checked={selected}
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
        <span ref={labelRef} className={selectable.label}>{label}</span>
      ) : (
        <span className={selectable.body}>
          <span className={selectable.label}>{label}</span>
          {hint && <span className={selectable.hint}>{hint}</span>}
        </span>
      )}
    </button>
  );
}
