import { useEffect, useState, type ReactNode } from "react";
import { ExternalAiDialog, type ExternalRedo } from "../ExternalAiDialog";
import { ExternalRedoDeepSeek } from "../ExternalRedoDeepSeek";
import { Button, Dialog, FormField, TextArea } from "../ui";

interface BeatAiDialogProps {
  open: boolean;
  onClose: () => void;
  /** Names the beat and the task ("Viết beat B4 · Ficin phân giải xác ong"). */
  title: string;
  /** The model the system's AI runs on, chosen here before it is sent. */
  modelSelect?: ReactNode;
  /** Runs the task on the system's AI; absent when no language model is configured. */
  onSystem?: (note: string) => void;
  /** Runs the task through an AI outside the system or DeepSeek, with the note. */
  external: ExternalRedo;
  /** Adds an empty beat instead of asking an AI; the dialog closes after it. */
  onBlank?: () => void;
  testId: string;
}

/**
 * The dialog that sends one beat's AI task (write it, or add a beat after it)
 * to the engine the Creator picks in the footer: the system's AI, an AI
 * outside, or DeepSeek (docs/ux-ui-design-rules.md, rule 4). The note is
 * optional, since nothing written exists yet that it would have to correct.
 */
export function BeatAiDialog({ open, onClose, title, modelSelect, onSystem, external, onBlank, testId }: BeatAiDialogProps) {
  const [note, setNote] = useState("");
  const [outside, setOutside] = useState(false);
  const [unreadable, setUnreadable] = useState<{ reply: string; message: string } | null>(null);
  useEffect(() => {
    if (!open) {
      setNote("");
      setOutside(false);
      setUnreadable(null);
    }
  }, [open]);

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
            {onBlank && (
              <Button
                variant="ghost"
                onClick={() => {
                  onBlank();
                  onClose();
                }}
                data-testid={`${testId}-blank`}
              >
                Thêm beat trống
              </Button>
            )}
            <Button
              variant="ghost"
              onClick={() => {
                setUnreadable(null);
                setOutside(true);
              }}
              data-testid={`${testId}-external`}
            >
              Dùng AI ngoài
            </Button>
            <ExternalRedoDeepSeek
              external={external}
              note={note.trim()}
              onClose={onClose}
              onUnreadable={(reply, message) => {
                setUnreadable({ reply, message });
                setOutside(true);
              }}
              testId={`${testId}-deepseek`}
            />
            {onSystem && (
              <Button
                variant="run"
                onClick={() => {
                  onSystem(note.trim());
                  onClose();
                }}
                data-testid={`${testId}-send`}
              >
                Gửi cho AI
              </Button>
            )}
          </>
        }
      >
        <FormField label="Ghi chú cho AI (không bắt buộc)">
          <TextArea
            value={note}
            onChange={(e) => setNote(e.target.value)}
            rows={4}
            placeholder="Điều muốn beat này nói hoặc tránh"
            data-testid={`${testId}-note`}
          />
        </FormField>
        {modelSelect}
      </Dialog>
      <ExternalAiDialog
        open={open && outside}
        onClose={() => setOutside(false)}
        title={title}
        run={(mode) => external.run(note.trim(), mode)}
        onDone={async () => {
          await external.onDone();
          onClose();
        }}
        initialReply={unreadable?.reply}
        initialError={unreadable?.message}
        testId={`${testId}-external-dialog`}
      />
    </>
  );
}
