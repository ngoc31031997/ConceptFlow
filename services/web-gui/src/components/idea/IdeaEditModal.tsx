import { useEffect, useRef, useState } from "react";
import type { ProjectIdea, ProjectIdeaPatch } from "../../api/client";
import { sanitizeIdeaText } from "../../utils/sanitizeIdeaText";
import { Button, Dialog, FormField, TextArea } from "../ui";
import unitStyles from "../units/units.module.css";

interface IdeaEditModalProps {
  open: boolean;
  onClose: () => void;
  /** The idea as stored; null when the Creator writes it before the AI did. */
  idea: ProjectIdea | null;
  /** Saves the fields whose text changed; a thrown error stays in the dialog. */
  onSave: (patch: ProjectIdeaPatch) => Promise<void>;
}

interface Form {
  question: string;
  points: string;
  scope: string;
}

function formOf(idea: ProjectIdea | null): Form {
  return {
    question: idea?.centralQuestion ?? "",
    points: (idea?.curiosityPoints ?? []).join("\n"),
    scope: idea?.scope ?? "",
  };
}

/** One curiosity point per line, blank lines dropped. */
function pointsOf(text: string): string[] {
  return text
    .split("\n")
    .map((line) => line.trim())
    .filter((line) => line !== "");
}

/**
 * "Sửa tay" for the idea (docs/ux-ui-design-rules.md, rule 4): the three fields
 * as a form, a line saying what saving does to the approval, then "Huỷ" and
 * "Lưu bản sửa". Nothing after the idea reads it until it is approved, so saving
 * only withdraws an approval and the button stays yellow (rule 6).
 */
export function IdeaEditModal({ open, onClose, idea, onSave }: IdeaEditModalProps) {
  const [form, setForm] = useState<Form>(() => formOf(idea));
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  // The texts to start from are read when the dialog opens, never again while
  // it is open: a reload of the page behind it must not overwrite what is typed.
  const latest = useRef(idea);
  useEffect(() => {
    latest.current = idea;
  });
  useEffect(() => {
    if (!open) return;
    setForm(formOf(latest.current));
    setError(null);
  }, [open]);

  async function handleSave() {
    const start = formOf(latest.current);
    const patch: ProjectIdeaPatch = {};
    if (form.question.trim() !== start.question.trim()) patch.centralQuestion = form.question;
    if (pointsOf(form.points).join("\n") !== pointsOf(start.points).join("\n")) patch.curiosityPoints = pointsOf(form.points);
    if (form.scope.trim() !== start.scope.trim()) patch.scope = form.scope;
    if (Object.keys(patch).length === 0) {
      onClose();
      return;
    }
    setSaving(true);
    setError(null);
    try {
      await onSave(patch);
      onClose();
    } catch (err) {
      setError(err instanceof Error && err.message ? err.message : "Không lưu được bản sửa. Thử lại.");
    } finally {
      setSaving(false);
    }
  }

  const approved = idea?.status === "approved";

  return (
    <Dialog
      open={open}
      onClose={onClose}
      title={idea ? "Sửa tay · Ý tưởng" : "Tự viết · Ý tưởng"}
      hasInput
      size="lg"
      testId="idea-edit"
      footer={
        <>
          <Button variant="ghost" onClick={onClose} data-testid="idea-edit-cancel">
            Huỷ
          </Button>
          <Button variant="primary" onClick={() => void handleSave()} disabled={saving} data-testid="idea-edit-save">
            {saving ? "Đang lưu…" : "Lưu bản sửa"}
          </Button>
        </>
      }
    >
      <FormField label="Câu hỏi trung tâm">
        <TextArea value={form.question} onChange={(e) => setForm({ ...form, question: sanitizeIdeaText(e.target.value) })} rows={2} data-testid="idea-edit-field-central_question" />
      </FormField>
      <FormField label="Điểm gây tò mò (mỗi dòng một điểm)">
        <TextArea value={form.points} onChange={(e) => setForm({ ...form, points: sanitizeIdeaText(e.target.value) })} rows={5} data-testid="idea-edit-field-curiosity_points" />
      </FormField>
      <FormField label="Phạm vi">
        <TextArea value={form.scope} onChange={(e) => setForm({ ...form, scope: sanitizeIdeaText(e.target.value) })} rows={3} data-testid="idea-edit-field-scope" />
      </FormField>
      <p className={unitStyles.note}>
        {approved ? "Ý tưởng đang được duyệt: lưu bản sửa sẽ đưa nó về chờ duyệt, bạn duyệt lại sau. " : ""}
        Ô bạn sửa thành “Bạn viết”: AI giữ nguyên khi làm lại. Xoá trắng một ô thì AI viết lại ô đó.
      </p>
      {error && (
        <p className={unitStyles.error} role="alert">
          {error}
        </p>
      )}
    </Dialog>
  );
}
