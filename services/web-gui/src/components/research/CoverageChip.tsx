import type { QuestionCoverage } from "../../api/client";
import styles from "./research.module.css";

/**
 * How well a question is sourced: two independent sites whose quote was found
 * is enough, one more is asked for, none is said plainly.
 */
export function CoverageChip({ coverage, testId }: { coverage: QuestionCoverage | undefined; testId?: string }) {
  if (!coverage) return null;
  if (coverage.sources >= 2) {
    return (
      <span className={`${styles.chip} ${styles.chipOk}`} data-testid={testId}>
        ✓ {coverage.sources} nguồn khớp
      </span>
    );
  }
  if (coverage.sources === 1) {
    return (
      <span className={`${styles.chip} ${styles.chipWait}`} data-testid={testId}>
        1 nguồn khớp · cần thêm 1
      </span>
    );
  }
  return (
    <span className={`${styles.chip} ${styles.chipTodo}`} data-testid={testId}>
      chưa có nguồn khớp
    </span>
  );
}
