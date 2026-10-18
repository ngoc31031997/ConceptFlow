import type { UnitStatus } from "../../api/client";
import styles from "./units.module.css";

/** A unit's review status, or "running" while the AI writes it. */
export type UnitChipState = UnitStatus | "running";

const CHIPS: Record<UnitChipState, { label: string; className: string }> = {
  pending: { label: "◆ Chờ duyệt", className: styles.chipPending },
  approved: { label: "✓ Đã duyệt", className: styles.chipApproved },
  running: { label: "● AI đang viết", className: styles.chipRunning },
};

/** The status chip in a unit card's header. */
export function UnitStatusChip({ state, testId }: { state: UnitChipState; testId?: string }) {
  const chip = CHIPS[state];
  return (
    <span className={`${styles.chip} ${chip.className}`} data-testid={testId} data-state={state}>
      {chip.label}
    </span>
  );
}
