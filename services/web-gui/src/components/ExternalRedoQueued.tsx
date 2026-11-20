import { useQueuedExternalRun } from "../hooks/useQueuedExternalRun";
import { QueuedRunButton } from "./QueuedRunButton";
import type { ExternalRedo } from "./ExternalAiDialog";

interface ExternalRedoQueuedProps {
  external: ExternalRedo;
  /** The note the redo sends along. */
  note: string;
  /** Names the item in the extension's popup. */
  label: string;
  /** The redo form closes after the answer was read. */
  onClose: () => void;
  /** The provider answered but the step could not read it: the form hands over to the outside-AI dialog. */
  onUnreadable: (reply: string, message: string) => void;
  disabled?: boolean;
  testId: string;
}

/**
 * "Thêm vào hàng đợi AI" in the footer of a redo form: queues the redo with the form's note,
 * which the extension runs on an outside AI, and closes the form once the answer was read.
 * Closing the form while the item is queued leaves it in the queue; the screen is not told when it ends.
 */
export function ExternalRedoQueued({ external, note, label, onClose, onUnreadable, disabled, testId }: ExternalRedoQueuedProps) {
  const queued = useQueuedExternalRun({
    run: (mode) => external.run(note, mode),
    label,
    onDone: async () => {
      await external.onDone();
      onClose();
    },
    onUnreadable,
  });
  return <QueuedRunButton run={queued} disabled={disabled} testId={testId} />;
}
