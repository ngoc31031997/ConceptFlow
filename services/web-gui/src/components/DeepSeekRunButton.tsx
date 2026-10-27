import type { DeepSeekRun } from "../hooks/useDeepSeekRun";
import { AiBridgeRunStatus } from "./AiBridgeRunStatus";
import { Button } from "./ui";
import styles from "./ExternalAiDialog.module.css";

interface DeepSeekRunButtonProps {
  run: DeepSeekRun;
  /** Another request on the same card is running, so this one waits. */
  disabled?: boolean;
  /** The button text; defaults to "Chạy bằng DeepSeek". */
  label?: string;
  /** What a click does; defaults to starting the run, for a host that asks something first. */
  onStart?: () => void;
  /** Prefix of the test ids: `<testId>`, `<testId>-stage`, `<testId>-cancel`, `<testId>-error`. */
  testId: string;
}

/**
 * "Chạy bằng DeepSeek" next to an AI action's "Dùng AI ngoài": one click runs
 * the step's prompt on DeepSeek and has the step read the answer. While it
 * runs the button gives way to where the run stands and "Huỷ". Without the
 * AI Bridge extension it renders nothing; the dialog of the outside AI says
 * how to install it.
 */
export function DeepSeekRunButton({ run, disabled, label = "Chạy bằng DeepSeek", onStart, testId }: DeepSeekRunButtonProps) {
  if (!run.available) return null;
  return (
    <>
      {run.stage === null ? (
        <Button variant="run" onClick={onStart ?? (() => void run.start())} disabled={disabled || run.busy} data-testid={testId}>
          {run.busy ? "Đang chạy…" : label}
        </Button>
      ) : (
        <AiBridgeRunStatus stage={run.stage} modes={run.modes} onCancel={run.cancel} testId={testId} />
      )}
      {run.error && (
        <p className={styles.error} role="alert" data-testid={`${testId}-error`}>
          {run.error}
        </p>
      )}
    </>
  );
}
