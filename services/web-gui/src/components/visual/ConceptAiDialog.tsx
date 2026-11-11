import { useEffect, useState, type ReactNode } from "react";
import { ExternalAiDialog, type ExternalRedo } from "../ExternalAiDialog";
import { ExternalRedoDeepSeek } from "../ExternalRedoDeepSeek";
import { Button, Dialog, FormField, TextArea } from "../ui";

interface ConceptAiDialogProps {
  open: boolean;
  onClose: () => void;
  /** Names the part and the task ("Làm lại phần 2 · Bảng màu và tương phản"). */
  title: string;
  /** What happens to the rest when this runs, shown above the buttons. */
  impact?: ReactNode;
  /** Runs the task on the system's AI with the note. */
  onSystem: (note: string) => void;
  /** Runs the task through an AI outside the system or DeepSeek, with the note. */
  external: ExternalRedo;
  testId: string;
}

/**
 * The dialog that sends one AI task of step 8 (write or rewrite a part, review
 * the whole concept) to the engine the Creator picks in the footer: the
 * system's AI, an AI outside, or DeepSeek (docs/ux-ui-design-rules.md, rule 4).
 * The note is optional.
 */
export function ConceptAiDialog({
  open,
  onClose,
  title,
  impact,
  onSystem,
  external,
  testId,
}: ConceptAiDialogProps) {
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
          </>
        }
      >
        <FormField label="Ghi chú cho AI (không bắt buộc)">
          <TextArea
            value={note}
            onChange={(e) => setNote(e.target.value)}
            rows={4}
            placeholder="Viết điều chưa ổn và muốn đổi thế nào"
            data-testid={`${testId}-note`}
          />
        </FormField>
        {impact}
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
