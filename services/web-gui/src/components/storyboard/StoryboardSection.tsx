import type { ReactNode } from "react";
import researchStyles from "../research/research.module.css";
import { Button } from "../ui";
import { Foldable } from "./Foldable";
import styles from "./storyboard.module.css";

/**
 * Where a section of step 11 stands: done (folds to its head), current (the
 * first one with work left), open (has content while a section above still
 * waits) or locked (nothing to do until the sections above are done).
 */
export type SectionState = "done" | "current" | "open" | "locked";

interface StoryboardSectionProps {
  number: number;
  title: string;
  /** One or two sentences on what the section does. */
  intro: ReactNode;
  state: SectionState;
  /** The short progress chip at the right of the head. */
  chip: ReactNode;
  /** Why a locked section waits, or what an open section needs from a section above. */
  note?: string;
  /** A done section shows its body only while the Creator opened it. */
  expanded: boolean;
  onToggle: (open: boolean) => void;
  testId: string;
  children: ReactNode;
}

/**
 * One numbered section of the storyboard page. The page reads top to bottom:
 * each section opens once the one above is done, a done section folds to its
 * head with a "Mở" button, and a locked one shows only why it waits.
 */
export function StoryboardSection({
  number,
  title,
  intro,
  state,
  chip,
  note,
  expanded,
  onToggle,
  testId,
  children,
}: StoryboardSectionProps) {
  const done = state === "done";
  return (
    <section className={styles.section} data-state={state} data-testid={testId} aria-label={title}>
      <div className={styles.sectionHead}>
        <span className={styles.sectionNum} aria-hidden="true">
          {done ? "✓" : number}
        </span>
        <div className={styles.sectionText}>
          <h3>
            {number} · {title}
          </h3>
          <span className={researchStyles.hint}>{intro}</span>
        </div>
        {chip}
        {done && (
          <Button
            variant="view"
            onClick={() => onToggle(!expanded)}
            data-testid={`${testId}-toggle`}
          >
            {expanded ? "Thu gọn" : "Mở"}
          </Button>
        )}
      </div>
      {note && (
        <p className={researchStyles.hint} data-testid={`${testId}-note`}>
          {state === "locked" ? "🔒 " : "⚠ "}
          {note}
        </p>
      )}
      {state !== "locked" && (
        <Foldable folded={done && !expanded} summary={null}>
          <div className={researchStyles.body}>{children}</div>
        </Foldable>
      )}
    </section>
  );
}
