import { useState } from "react";
import type { ExternalAi } from "../api/client";
import { useQueuedExternalRun } from "../hooks/useQueuedExternalRun";
import { QueuedRunButton } from "./QueuedRunButton";
import { ExternalAiDialog } from "./ExternalAiDialog";
import { Button } from "./ui";

interface ExternalAiButtonBaseProps<T> {
  /** Names the item and the task, shown as the dialog title and as the item's name in the extension's queue. */
  title: string;
  /** Makes the step's own AI request in the given external mode (see `ExternalAiDialog`). */
  run: (external: ExternalAi) => Promise<T>;
  /** The button text; defaults to "Dùng AI ngoài". */
  label?: string;
  disabled?: boolean;
  testId: string;
}

/**
 * What happens once the reply was read: with the queue (the default) the answer is read in the
 * extension and the screen loads what changed; a step whose result is shown rather than stored
 * (`queued: false`) gets the result of the pasted reply and has no queue button.
 */
type ExternalAiButtonProps<T> = ExternalAiButtonBaseProps<T> &
  ({ queued?: true; onDone: () => void | Promise<void> } | { queued: false; onDone: (result: T) => void | Promise<void> });

/**
 * The "Dùng AI ngoài" button of an AI action that has no form of its own, with its dialog,
 * and beside it "Thêm vào hàng đợi AI" when the AI Bridge extension is installed. An answer
 * the step cannot read opens the dialog with that answer and the reason.
 * An action with a form (a note) opens `ExternalAiDialog` from that form's footer instead.
 */
export function ExternalAiButton<T>(props: ExternalAiButtonProps<T>) {
  const { title, run, label = "Dùng AI ngoài", disabled, testId } = props;
  const [open, setOpen] = useState(false);
  const [unreadable, setUnreadable] = useState<{ reply: string; message: string } | null>(null);
  const queued = useQueuedExternalRun({
    run,
    label: title,
    onDone: async () => {
      if (props.queued !== false) await props.onDone();
    },
    onUnreadable: (reply, message) => {
      setUnreadable({ reply, message });
      setOpen(true);
    },
  });
  const dialog = {
    open,
    onClose: () => setOpen(false),
    title,
    run,
    initialReply: unreadable?.reply,
    initialError: unreadable?.message,
    testId: `${testId}-dialog`,
  };
  return (
    <>
      <Button
        variant="ghost"
        onClick={() => {
          setUnreadable(null);
          setOpen(true);
        }}
        disabled={disabled || queued.busy}
        data-testid={testId}
      >
        {label}
      </Button>
      {props.queued !== false && <QueuedRunButton run={queued} disabled={disabled} testId={`${testId}-queue`} />}
      {props.queued === false ? <ExternalAiDialog {...dialog} queued={false} onDone={props.onDone} /> : <ExternalAiDialog {...dialog} onDone={props.onDone} />}
    </>
  );
}
