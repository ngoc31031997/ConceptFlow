import { useState } from "react";
import type { ExternalAi } from "../api/client";
import { useDeepSeekRun } from "../hooks/useDeepSeekRun";
import { DeepSeekRunButton } from "./DeepSeekRunButton";
import { ExternalAiDialog } from "./ExternalAiDialog";
import { Button } from "./ui";

interface ExternalAiButtonProps<T> {
  /** Names the item and the task, shown as the dialog title. */
  title: string;
  /** Makes the step's own AI request in the given external mode (see `ExternalAiDialog`). */
  run: (external: ExternalAi) => Promise<T>;
  /** The reply was read and the step ran on it; gets what the request returned. The screen loads what changed. */
  onDone: (result: T) => void | Promise<void>;
  /** The button text; defaults to "Dùng AI ngoài". */
  label?: string;
  disabled?: boolean;
  testId: string;
}

/**
 * The "Dùng AI ngoài" button of an AI action that has no form of its own, with its dialog,
 * and beside it "Chạy bằng DeepSeek" when the AI Bridge extension is installed. A DeepSeek
 * answer the step cannot read opens the dialog with that answer and the reason.
 * An action with a form (a note) opens `ExternalAiDialog` from that form's footer instead.
 */
export function ExternalAiButton<T>({ title, run, onDone, label = "Dùng AI ngoài", disabled, testId }: ExternalAiButtonProps<T>) {
  const [open, setOpen] = useState(false);
  const [unreadable, setUnreadable] = useState<{ reply: string; message: string } | null>(null);
  const deepSeek = useDeepSeekRun({
    run,
    onDone,
    onUnreadable: (reply, message) => {
      setUnreadable({ reply, message });
      setOpen(true);
    },
  });
  return (
    <>
      <Button
        variant="ghost"
        onClick={() => {
          setUnreadable(null);
          setOpen(true);
        }}
        disabled={disabled || deepSeek.busy}
        data-testid={testId}
      >
        {label}
      </Button>
      <DeepSeekRunButton run={deepSeek} disabled={disabled} testId={`${testId}-deepseek`} />
      <ExternalAiDialog
        open={open}
        onClose={() => setOpen(false)}
        title={title}
        run={run}
        onDone={onDone}
        initialReply={unreadable?.reply}
        initialError={unreadable?.message}
        testId={`${testId}-dialog`}
      />
    </>
  );
}
