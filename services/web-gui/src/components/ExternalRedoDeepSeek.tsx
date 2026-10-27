import { useDeepSeekRun } from "../hooks/useDeepSeekRun";
import { DeepSeekRunButton } from "./DeepSeekRunButton";
import type { ExternalRedo } from "./ExternalAiDialog";

interface ExternalRedoDeepSeekProps {
  external: ExternalRedo;
  /** The note the redo sends along. */
  note: string;
  /** The redo form closes after the answer was read. */
  onClose: () => void;
  /** DeepSeek answered but the step could not read it: the form hands over to the outside-AI dialog. */
  onUnreadable: (reply: string, message: string) => void;
  disabled?: boolean;
  testId: string;
}

/**
 * "Chạy bằng DeepSeek" in the footer of a redo form: runs the redo with the
 * form's note on DeepSeek and closes the form once the answer was read.
 * Closing the form while it runs stops waiting for the run.
 */
export function ExternalRedoDeepSeek({ external, note, onClose, onUnreadable, disabled, testId }: ExternalRedoDeepSeekProps) {
  const deepSeek = useDeepSeekRun({
    run: (mode) => external.run(note, mode),
    onDone: async () => {
      await external.onDone();
      onClose();
    },
    onUnreadable,
  });
  return <DeepSeekRunButton run={deepSeek} disabled={disabled} testId={testId} />;
}
