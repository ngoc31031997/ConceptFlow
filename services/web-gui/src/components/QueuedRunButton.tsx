import type { QueuedExternalRun } from "../hooks/useQueuedExternalRun";
import { AiBridgeRunStatus } from "./AiBridgeRunStatus";
import { Button } from "./ui";
import styles from "./ExternalAiDialog.module.css";

/** Said where the extension is installed but too old for the queue. */
export const UPDATE_EXTENSION_TEXT = "Cập nhật extension ConceptFlow AI Bridge lên bản mới để chạy bằng hàng đợi AI.";

interface QueuedRunButtonProps {
  run: QueuedExternalRun;
  /** Another request on the same card is running, so this one waits. */
  disabled?: boolean;
  /** The button text; defaults to "Thêm vào hàng đợi AI". */
  label?: string;
  /** What a click does; defaults to starting the run, for a host that asks something first. */
  onStart?: () => void;
  /** Prefix of the test ids: `<testId>`, `<testId>-stage`, `<testId>-cancel`, `<testId>-error`. */
  testId: string;
}

/**
 * "Thêm vào hàng đợi AI" next to an AI action's "Dùng AI ngoài": one click queues the step's
 * request, which the extension runs on an outside AI and has the step read. While the item is on
 * its way the button gives way to where it stands and "Huỷ". Without the AI Bridge extension it
 * renders nothing (the dialog of the outside AI says how to install it); with an old one it says to update.
 */
export function QueuedRunButton({ run, disabled, label = "Thêm vào hàng đợi AI", onStart, testId }: QueuedRunButtonProps) {
  if (run.outdated && !run.available) {
    return (
      <p className={styles.hint} data-testid={`${testId}-outdated`}>
        {UPDATE_EXTENSION_TEXT}
      </p>
    );
  }
  if (!run.available) return null;
  return (
    <>
      {!run.busy ? (
        <Button variant="run" onClick={onStart ?? (() => void run.start())} disabled={disabled} data-testid={testId}>
          {label}
        </Button>
      ) : (
        <AiBridgeRunStatus item={run.item} ahead={run.ahead} onCancel={run.cancel} testId={testId} />
      )}
      {run.error && (
        <p className={styles.error} role="alert" data-testid={`${testId}-error`}>
          {run.error}
        </p>
      )}
    </>
  );
}
