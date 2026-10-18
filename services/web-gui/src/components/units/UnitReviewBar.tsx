import type { UnitStatus } from "../../api/client";
import { Button } from "../ui";
import styles from "./units.module.css";

interface UnitReviewBarProps {
  status: UnitStatus;
  /** "Duyệt brief". */
  approveLabel: string;
  /** The unit is not complete enough to approve yet. */
  approveDisabled?: boolean;
  /** An action on the unit is in flight, or the AI is rewriting it. */
  busy: boolean;
  /** A language model is configured, so the AI can redo the unit. */
  canRedo: boolean;
  onEdit: () => void;
  onRedo: () => void;
  onApprove: () => void;
  onUnapprove: () => void;
  testId: string;
}

/**
 * The actions under a unit, left to right: "Sửa tay" always; while it waits
 * for review, "Làm lại kèm ghi chú" and the approve button; once approved,
 * "Bỏ duyệt". Edit and redo open their dialogs; approving moves forward and is
 * easy to undo, so it asks nothing.
 */
export function UnitReviewBar({
  status,
  approveLabel,
  approveDisabled = false,
  busy,
  canRedo,
  onEdit,
  onRedo,
  onApprove,
  onUnapprove,
  testId,
}: UnitReviewBarProps) {
  return (
    <div className={styles.bar} data-testid={testId}>
      <Button variant="ghost" onClick={onEdit} disabled={busy} data-testid={`${testId}-edit`}>
        Sửa tay
      </Button>
      {status === "pending" ? (
        <>
          {canRedo && (
            <Button variant="ghost" onClick={onRedo} disabled={busy} data-testid={`${testId}-redo`}>
              Làm lại kèm ghi chú
            </Button>
          )}
          <Button
            variant="primary"
            onClick={onApprove}
            disabled={busy || approveDisabled}
            data-testid={`${testId}-approve`}
          >
            {approveLabel}
          </Button>
        </>
      ) : (
        <Button variant="ghost" onClick={onUnapprove} disabled={busy} data-testid={`${testId}-unapprove`}>
          Bỏ duyệt
        </Button>
      )}
    </div>
  );
}
