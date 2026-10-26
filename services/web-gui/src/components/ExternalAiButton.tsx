import { useState } from "react";
import type { ExternalAi } from "../api/client";
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
 * The "Dùng AI ngoài" button of an AI action that has no form of its own, with its dialog.
 * An action with a form (a note) opens `ExternalAiDialog` from that form's footer instead.
 */
export function ExternalAiButton<T>({ title, run, onDone, label = "Dùng AI ngoài", disabled, testId }: ExternalAiButtonProps<T>) {
  const [open, setOpen] = useState(false);
  return (
    <>
      <Button
        variant="ghost"
        onClick={() => setOpen(true)}
        disabled={disabled}
        data-testid={testId}
      >
        {label}
      </Button>
      <ExternalAiDialog
        open={open}
        onClose={() => setOpen(false)}
        title={title}
        run={run}
        onDone={onDone}
        testId={`${testId}-dialog`}
      />
    </>
  );
}
