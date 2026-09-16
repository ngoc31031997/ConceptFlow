import type { ReactNode } from "react";
import glass from "../../styles/glass.module.css";

interface CtaRowProps {
  helperText?: ReactNode;
  children: ReactNode;
}

/** Right-aligned action row (helper text left, buttons right) shared across forms. */
export function CtaRow({ helperText, children }: CtaRowProps) {
  return (
    <div className={glass.ctaRow}>
      {helperText && <span className={glass.helperText}>{helperText}</span>}
      {children}
    </div>
  );
}
