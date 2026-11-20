import type { ReactNode } from "react";
import { usePresence } from "../../hooks/usePresence";
import glass from "../../styles/glass.module.css";

interface FoldableProps {
  /** Show the one-line summary instead of the full content. */
  folded: boolean;
  /** The one-line summary shown while folded. */
  summary: ReactNode;
  /** The full content shown while not folded. */
  children: ReactNode;
}

/**
 * An item that folds to a one-line summary and opens to its full content. The
 * content slides in and out with the shared reveal classes
 * (docs/ux-ui-design-rules.md, rule 3); the summary shows while it is folded.
 */
export function Foldable({ folded, summary, children }: FoldableProps) {
  const { mounted, closing } = usePresence(!folded);
  return (
    <>
      {folded && summary}
      {mounted && <div className={closing ? glass.revealOut : glass.reveal}>{children}</div>}
    </>
  );
}
