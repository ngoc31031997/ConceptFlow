import type { QuestionCoverage } from "../../api/client";
import styles from "./research.module.css";

/**
 * How well a question is sourced: two independent sites whose quote was found
 * is enough, one more is asked for, none is said plainly. While a link is being
 * checked it says so instead of showing a count that may still change.
 */
export function CoverageChip({
  coverage,
  checking = false,
  testId,
}: {
  coverage: QuestionCoverage | undefined;
  /** A link of one of the question's facts is still being checked, so the count is not final. */
  checking?: boolean;
  testId?: string;
}) {
  if (checking) {
    return (
      <span className={`${styles.chip} ${styles.chipRun}`} data-testid={testId} data-checking="true">
        <span className={styles.spinner} aria-hidden="true" />
        đang tính…
      </span>
    );
  }
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
