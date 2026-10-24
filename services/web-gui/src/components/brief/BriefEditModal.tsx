import { useEffect, useRef, useState } from "react";
import type { ProjectBrief, ProjectBriefPatch } from "../../api/client";
import { formatClock, parseClock } from "../../utils/clockDuration";
import { sanitizeIdeaText } from "../../utils/sanitizeIdeaText";
import { Button, Dialog, FormField, TextArea, TextInput } from "../ui";
import unitStyles from "../units/units.module.css";

interface BriefEditModalProps {
  open: boolean;
  onClose: () => void;
  /** The brief as stored; null when the Creator writes it before the AI did. */
  brief: ProjectBrief | null;
  /** The lengths, in seconds, this video form allows. */
  minDurationSeconds: number;
  maxDurationSeconds: number;
  /** Saves the fields whose text changed; a thrown error stays in the dialog. */
  onSave: (patch: ProjectBriefPatch) => Promise<void>;
}

interface Form {
  audience: string;
  duration: string;
  tone: string;
  objectives: string;
  avoid: string;
}

function formOf(brief: ProjectBrief | null): Form {
  return {
    audience: brief?.audience ?? "",
    duration: brief && brief.durationSeconds > 0 ? formatClock(brief.durationSeconds) : "",
    tone: brief?.tone ?? "",
    objectives: (brief?.learningObjectives ?? []).join("\n"),
    avoid: (brief?.avoid ?? []).join("\n"),
  };
}

/** One list item per line, blank lines dropped. */
function linesOf(text: string): string[] {
  return text
    .split("\n")
    .map((line) => line.trim())
    .filter((line) => line !== "");
}

/**
 * "Sửa tay" for the brief (docs/ux-ui-design-rules.md, rule 4): the five fields
 * as a form, a line saying what saving does to the approval, then "Huỷ" and
 * "Lưu bản sửa". The length is typed as m:ss or h:mm:ss and saved as seconds.
 * Nothing after the brief reads it until it is approved, so saving only
 * withdraws an approval and the button stays yellow (rule 6).
 */
export function BriefEditModal({
  open,
  onClose,
  brief,
  minDurationSeconds,
  maxDurationSeconds,
  onSave,
}: BriefEditModalProps) {
  const [form, setForm] = useState<Form>(() => formOf(brief));
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  // The texts to start from are read when the dialog opens, never again while
  // it is open: a reload of the page behind it must not overwrite what is typed.
  const latest = useRef(brief);
  useEffect(() => {
    latest.current = brief;
  });
  useEffect(() => {
    if (!open) return;
    setForm(formOf(latest.current));
    setError(null);
  }, [open]);

  async function handleSave() {
    const start = formOf(latest.current);
    const patch: ProjectBriefPatch = {};
    if (form.audience.trim() !== start.audience.trim()) patch.audience = form.audience;
    if (form.tone.trim() !== start.tone.trim()) patch.tone = form.tone;
    if (linesOf(form.objectives).join("\n") !== linesOf(start.objectives).join("\n"))
      patch.learningObjectives = linesOf(form.objectives);
    if (linesOf(form.avoid).join("\n") !== linesOf(start.avoid).join("\n"))
      patch.avoid = linesOf(form.avoid);
    if (form.duration.trim() !== start.duration.trim()) {
      const seconds = parseClock(form.duration);
      if (seconds === null) {
        setError(
          "Thời lượng gõ theo dạng phút:giây, ví dụ 4:30, hoặc giờ:phút:giây, ví dụ 1:02:05.",
        );
        return;
      }
      patch.durationSeconds = seconds;
    }
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
      setError(
        err instanceof Error && err.message ? err.message : "Không lưu được bản sửa. Thử lại.",
      );
    } finally {
      setSaving(false);
    }
  }

  const approved = brief?.status === "approved";

  return (
    <Dialog
      open={open}
      onClose={onClose}
      title={brief ? "Sửa tay · Creative brief" : "Tự viết · Creative brief"}
      hasInput
      size="xl"
      testId="brief-edit"
      footer={
        <>
          <Button variant="ghost" onClick={onClose} data-testid="brief-edit-cancel">
            Huỷ
          </Button>
          <Button
            variant="primary"
            onClick={() => void handleSave()}
            disabled={saving}
            data-testid="brief-edit-save"
          >
            {saving ? "Đang lưu…" : "Lưu bản sửa"}
          </Button>
        </>
      }
    >
      <FormField label="Khán giả">
        <TextArea
          value={form.audience}
          onChange={(e) => setForm({ ...form, audience: sanitizeIdeaText(e.target.value) })}
          rows={2}
          data-testid="brief-edit-field-audience"
        />
      </FormField>
      <FormField
        label={`Thời lượng (phút:giây, từ ${formatClock(minDurationSeconds)} đến ${formatClock(maxDurationSeconds)})`}
      >
        <TextInput
          value={form.duration}
          onChange={(e) => setForm({ ...form, duration: e.target.value })}
          placeholder="4:30"
          data-testid="brief-edit-field-duration_seconds"
        />
      </FormField>
      <FormField label="Giọng điệu">
        <TextArea
          value={form.tone}
          onChange={(e) => setForm({ ...form, tone: sanitizeIdeaText(e.target.value) })}
          rows={2}
          data-testid="brief-edit-field-tone"
        />
      </FormField>
      <FormField label="Mục tiêu học tập (mỗi dòng một mục tiêu)">
        <TextArea
          value={form.objectives}
          onChange={(e) => setForm({ ...form, objectives: sanitizeIdeaText(e.target.value) })}
          rows={5}
          data-testid="brief-edit-field-learning_objectives"
        />
      </FormField>
      <FormField label="Cần tránh (mỗi dòng một điều)">
        <TextArea
          value={form.avoid}
          onChange={(e) => setForm({ ...form, avoid: sanitizeIdeaText(e.target.value) })}
          rows={5}
          data-testid="brief-edit-field-avoid"
        />
      </FormField>
      <p className={unitStyles.note}>
        {approved
          ? "Brief đang được duyệt: lưu bản sửa sẽ đưa nó về chờ duyệt, bạn duyệt lại sau. "
          : ""}
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
