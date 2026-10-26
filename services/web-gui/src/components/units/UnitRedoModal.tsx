import { useEffect, useRef, useState, type ReactNode } from "react";
import { ExternalAiDialog, type ExternalRedo } from "../ExternalAiDialog";
import { Button, Dialog, FormField, TextArea } from "../ui";
import { reachesLaterSteps, type UnitImpact } from "../../hooks/useUnitImpact";
import { ImpactList } from "./ImpactList";

interface UnitRedoModalProps {
  open: boolean;
  onClose: () => void;
  /** "Làm lại kèm ghi chú · Creative brief". */
  title: string;
  impact: UnitImpact;
  /** The note to start from: the one of a run that failed, so it is not typed twice. */
  initialNote: string;
  /** The model the redo runs on, chosen here before the AI is sent. */
  modelSelect?: ReactNode;
  /** Starts the redo; the dialog closes and the page follows the run. Absent when the system's own AI is not configured. */
  onSend?: (note: string) => void;
  /** Runs the redo through an AI outside the system; the button shows only when the step supports it. */
  external?: ExternalRedo;
  testId: string;
}

/**
 * "Làm lại kèm ghi chú" for one unit (docs/ux-ui-design-rules.md, rule 4):
 * the note for the AI, then what the redo sends back in later steps, then
 * "Huỷ" and "Gửi cho AI làm lại" — orange when later work depends on the unit,
 * purple when nothing does yet. Being the action's own dialog, it is also its
 * confirmation.
 */
export function UnitRedoModal({
  open,
  onClose,
  title,
  impact,
  initialNote,
  modelSelect,
  onSend,
  external,
  testId,
}: UnitRedoModalProps) {
  const [note, setNote] = useState(initialNote);
  const [outside, setOutside] = useState(false);
  const latestInitial = useRef(initialNote);
  useEffect(() => {
    latestInitial.current = initialNote;
  });
  useEffect(() => {
    if (open) setNote(latestInitial.current);
    else setOutside(false);
  }, [open]);

  const canSend = note.trim().length > 0 && impact.items !== null;

  return (
    <>
      <Dialog
        open={open && !outside}
        onClose={onClose}
        title={title}
        hasInput
        testId={testId}
        footer={
          <>
            <Button variant="ghost" onClick={onClose} data-testid={`${testId}-cancel`}>
              Huỷ
            </Button>
            {external && (
              <Button
                variant="ghost"
                onClick={() => setOutside(true)}
                disabled={!canSend}
                data-testid={`${testId}-external`}
              >
                Dùng AI ngoài
              </Button>
            )}
            {onSend && (
              <Button
                variant={reachesLaterSteps(impact) ? "impact" : "run"}
                onClick={() => {
                  onSend(note.trim());
                  onClose();
                }}
                disabled={!canSend}
                data-testid={`${testId}-send`}
              >
                Gửi cho AI làm lại
              </Button>
            )}
          </>
        }
      >
        <FormField label="Ghi chú cho AI">
          <TextArea
            value={note}
            onChange={(e) => setNote(e.target.value)}
            rows={4}
            placeholder="Viết điều chưa ổn và muốn đổi thế nào"
            data-testid={`${testId}-note`}
          />
        </FormField>
        {modelSelect}
        <ImpactList
          head="AI làm lại mục này thì phần phụ thuộc sẽ làm lại:"
          impact={impact}
          testId={`${testId}-impact`}
        />
      </Dialog>
      {external && (
        <ExternalAiDialog
          open={open && outside}
          onClose={() => setOutside(false)}
          title={title}
          run={(mode) => external.run(note.trim(), mode)}
          onDone={async () => {
            await external.onDone();
            onClose();
          }}
          testId={`${testId}-external-dialog`}
        />
      )}
    </>
  );
}
