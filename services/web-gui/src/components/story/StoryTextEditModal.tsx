import { useEffect, useRef, useState } from "react";
import { sanitizeIdeaText } from "../../utils/sanitizeIdeaText";
import { Button, Dialog, FormField, TextArea } from "../ui";
import unitStyles from "../units/units.module.css";

interface StoryTextEditModalProps {
  open: boolean;
  onClose: () => void;
  /** "Câu hỏi trung tâm". */
  label: string;
  value: string;
  rows: number;
  /** The story is approved: saving withdraws the approval. */
  approved: boolean;
  /** Saves the text; a thrown error stays in the dialog. */
  onSave: (text: string) => Promise<void>;
  testId: string;
}

/**
 * "Sửa tay" for one text field of the story (docs/ux-ui-design-rules.md, rule
 * 4). Nothing after the story reads it until it is approved, so saving only
 * withdraws an approval and the button stays yellow (rule 6).
 */
export function StoryTextEditModal({ open, onClose, label, value, rows, approved, onSave, testId }: StoryTextEditModalProps) {
  const [text, setText] = useState(value);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  // The text to start from is read when the dialog opens, never again while it
  // is open: a reload of the page behind it must not overwrite what is typed.
  const latest = useRef(value);
  useEffect(() => {
    latest.current = value;
  });
  useEffect(() => {
    if (!open) return;
    setText(latest.current);
    setError(null);
  }, [open]);

  async function handleSave() {
    if (text.trim() === latest.current.trim()) {
      onClose();
      return;
    }
    setSaving(true);
    setError(null);
    try {
      await onSave(text);
      onClose();
    } catch (err) {
      setError(err instanceof Error && err.message ? err.message : "Không lưu được bản sửa. Thử lại.");
    } finally {
      setSaving(false);
    }
  }

  return (
    <Dialog
      open={open}
      onClose={onClose}
      title={`Sửa tay · ${label}`}
      hasInput
      testId={testId}
      footer={
        <>
          <Button variant="ghost" onClick={onClose} data-testid={`${testId}-cancel`}>
            Huỷ
          </Button>
          <Button variant="primary" onClick={() => void handleSave()} disabled={saving} data-testid={`${testId}-save`}>
            {saving ? "Đang lưu…" : "Lưu bản sửa"}
          </Button>
        </>
      }
    >
      <FormField label={label}>
        <TextArea value={text} onChange={(e) => setText(sanitizeIdeaText(e.target.value))} rows={rows} data-testid={`${testId}-field`} />
      </FormField>
      <p className={unitStyles.note}>
        {approved ? "Câu chuyện đang được duyệt: lưu bản sửa sẽ đưa nó về chờ duyệt, bạn duyệt lại sau. " : ""}
        Ô bạn sửa thành “Bạn viết”: AI giữ nguyên khi viết lại. Xoá trắng thì AI viết lại ô này.
      </p>
      {error && (
        <p className={unitStyles.error} role="alert">
          {error}
        </p>
      )}
    </Dialog>
  );
}
