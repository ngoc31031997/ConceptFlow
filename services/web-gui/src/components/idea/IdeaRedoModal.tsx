import { useEffect, useRef, useState, type ReactNode } from "react";
import { Button, Dialog, FormField, TextArea } from "../ui";
import unitStyles from "../units/units.module.css";

interface IdeaRedoModalProps {
  open: boolean;
  onClose: () => void;
  /** The idea is approved: redoing withdraws the approval. */
  approved: boolean;
  /** The note to start from: the one of a run that failed, so it is not typed twice. */
  initialNote: string;
  /** The model the redo runs on, chosen here before the AI is sent. */
  modelSelect?: ReactNode;
  /** Starts the redo; the dialog closes and the page follows the run. */
  onSend: (note: string) => void;
}

/**
 * "Làm lại kèm ghi chú" for the idea (docs/ux-ui-design-rules.md, rule 4): the
 * note for the AI, then what the redo does, then "Huỷ" and "Gửi cho AI làm lại".
 * The AI keeps the fields the Creator wrote. Nothing after the idea has been
 * built from it yet, so the button is purple (rule 6).
 */
export function IdeaRedoModal({ open, onClose, approved, initialNote, modelSelect, onSend }: IdeaRedoModalProps) {
  const [note, setNote] = useState(initialNote);
  const latestInitial = useRef(initialNote);
  useEffect(() => {
    latestInitial.current = initialNote;
  });
  useEffect(() => {
    if (open) setNote(latestInitial.current);
  }, [open]);

  return (
    <Dialog
      open={open}
      onClose={onClose}
      title="Làm lại kèm ghi chú · Ý tưởng"
      hasInput
      testId="idea-redo"
      footer={
        <>
          <Button variant="ghost" onClick={onClose} data-testid="idea-redo-cancel">
            Huỷ
          </Button>
          <Button
            variant="run"
            onClick={() => {
              onSend(note.trim());
              onClose();
            }}
            disabled={note.trim().length === 0}
            data-testid="idea-redo-send"
          >
            Gửi cho AI làm lại
          </Button>
        </>
      }
    >
      <FormField label="Ghi chú cho AI">
        <TextArea
          value={note}
          onChange={(e) => setNote(e.target.value)}
          rows={4}
          placeholder="Viết điều chưa ổn và muốn đổi thế nào"
          data-testid="idea-redo-note"
        />
      </FormField>
      {modelSelect}
      <p className={unitStyles.note}>
        AI viết lại các ô bạn chưa sửa; ô “Bạn viết” giữ nguyên.
        {approved ? " Ý tưởng đang được duyệt sẽ về chờ duyệt." : ""}
      </p>
    </Dialog>
  );
}
