import type { ReactNode } from "react";
import glass from "../../styles/glass.module.css";

interface CardProps {
  title?: ReactNode;
  hint?: ReactNode;
  headerAction?: ReactNode;
  children: ReactNode;
  className?: string;
  "data-testid"?: string;
}

/**
 * The one "glass card" shape (UX review #1/#3 pattern, see glass.module.css)
 * — every page that groups form fields, a panel, or a status block should
 * reach for this instead of hand-rolling `glass.card` + `glass.cardHeader`,
 * so a future visual tweak to the card shell only has to change one place.
 */
export function Card({ title, hint, headerAction, children, className, ...rest }: CardProps) {
  return (
    <div className={className ? `${glass.card} ${className}` : glass.card} {...rest}>
      {(title || headerAction) && (
        <div className={glass.cardHeader}>
          <div>
            {title && <span className={glass.cardTitle}>{title}</span>}
            {hint && <p className={glass.cardHint}>{hint}</p>}
          </div>
          {headerAction}
        </div>
      )}
      {children}
    </div>
  );
}
