import { useEffect, useRef, useState } from "react";
import { MAX_ARC_BEATS, type ArcBeat } from "../../api/client";
import { sanitizeIdeaText } from "../../utils/sanitizeIdeaText";
import { Button, Dialog, FormField, TextArea, TextInput } from "../ui";
import unitStyles from "../units/units.module.css";
import styles from "./story.module.css";

interface StoryArcEditModalProps {
  open: boolean;
  onClose: () => void;
  arc: ArcBeat[];
  /** The story is approved: saving withdraws the approval. */
  approved: boolean;
  /** Saves the arc; a thrown error stays in the dialog. */
  onSave: (arc: ArcBeat[]) => Promise<void>;
}

interface Row {
  beat: string;
  text: string;
  /** Fact codes as typed, "F01, F02". */
  facts: string;
}

function rowsOf(arc: ArcBeat[]): Row[] {
  return arc.map((b) => ({ beat: b.beat, text: b.text, facts: b.facts.join(", ") }));
}

function codesOf(text: string): string[] {
  return text
    .split(/[\s,;]+/)
    .map((c) => c.trim().toUpperCase())
    .filter((c) => c !== "");
}

function sameArc(a: ArcBeat[], b: ArcBeat[]): boolean {
  return JSON.stringify(a) === JSON.stringify(b);
}

/**
 * "Sửa tay" for the narrative arc (docs/ux-ui-design-rules.md, rule 4): one row
 * per beat with its name, its text and the fact codes it rests on, in the order
 * the viewer meets them. Rows can be moved, added and removed; nothing is saved
 * until "Lưu bản sửa", so removing a row here discards nothing stored.
 */
export function StoryArcEditModal({ open, onClose, arc, approved, onSave }: StoryArcEditModalProps) {
  const [rows, setRows] = useState<Row[]>(() => rowsOf(arc));
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  // The rows to start from are read when the dialog opens, never again while it
  // is open: a reload of the page behind it must not overwrite what is typed.
  const latest = useRef(arc);
  useEffect(() => {
    latest.current = arc;
  });
  useEffect(() => {
    if (!open) return;
    setRows(rowsOf(latest.current));
    setError(null);
  }, [open]);

  function change(i: number, patch: Partial<Row>) {
    setRows((prev) => prev.map((r, j) => (j === i ? { ...r, ...patch } : r)));
  }
  function move(i: number, by: number) {
    setRows((prev) => {
      const j = i + by;
      if (j < 0 || j >= prev.length) return prev;
      const next = [...prev];
      [next[i], next[j]] = [next[j], next[i]];
      return next;
    });
  }

  async function handleSave() {
    const next: ArcBeat[] = rows
      .map((r) => ({ beat: r.beat.trim(), text: r.text.trim(), facts: codesOf(r.facts) }))
      .filter((b) => b.beat !== "" || b.text !== "" || b.facts.length > 0);
    if (sameArc(next, latest.current)) {
      onClose();
      return;
    }
    setSaving(true);
    setError(null);
    try {
      await onSave(next);
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
      title="Sửa tay · Mạch kể"
      hasInput
      size="xl"
      testId="story-arc-edit"
      footer={
        <>
          <Button variant="ghost" onClick={onClose} data-testid="story-arc-edit-cancel">
            Huỷ
          </Button>
          <Button variant="primary" onClick={() => void handleSave()} disabled={saving} data-testid="story-arc-edit-save">
            {saving ? "Đang lưu…" : "Lưu bản sửa"}
          </Button>
        </>
      }
    >
      {rows.map((row, i) => (
        <div key={i} className={styles.beatEdit} data-testid={`story-arc-row-${i}`}>
          <div className={styles.beatEditHead}>
            <span>Nhịp {i + 1}</span>
            <span className={styles.beatEditTools}>
              <Button variant="ghost" onClick={() => move(i, -1)} disabled={i === 0} aria-label={`Đưa nhịp ${i + 1} lên`} data-testid={`story-arc-up-${i}`}>
                ↑
              </Button>
              <Button variant="ghost" onClick={() => move(i, 1)} disabled={i === rows.length - 1} aria-label={`Đưa nhịp ${i + 1} xuống`} data-testid={`story-arc-down-${i}`}>
                ↓
              </Button>
              <Button variant="ghost" onClick={() => setRows((prev) => prev.filter((_, j) => j !== i))} data-testid={`story-arc-remove-${i}`}>
                Bỏ nhịp
              </Button>
            </span>
          </div>
          <FormField label="Tên nhịp">
            <TextInput value={row.beat} onChange={(e) => change(i, { beat: sanitizeIdeaText(e.target.value) })} data-testid={`story-arc-beat-${i}`} />
          </FormField>
          <FormField label="Nội dung">
            <TextArea value={row.text} onChange={(e) => change(i, { text: sanitizeIdeaText(e.target.value) })} rows={3} data-testid={`story-arc-text-${i}`} />
          </FormField>
          <FormField label="Dữ kiện dựa vào (mã cách nhau bằng dấu phẩy, có thể để trống)">
            <TextInput value={row.facts} onChange={(e) => change(i, { facts: e.target.value })} placeholder="F01, F02" data-testid={`story-arc-facts-${i}`} />
          </FormField>
        </div>
      ))}
      <div>
        <Button
          variant="ghost"
          onClick={() => setRows((prev) => [...prev, { beat: "", text: "", facts: "" }])}
          disabled={rows.length >= MAX_ARC_BEATS}
          data-testid="story-arc-add"
        >
          + Thêm nhịp
        </Button>
      </div>
      <p className={unitStyles.note}>
        Mạch kể có từ 3 đến {MAX_ARC_BEATS} nhịp; nhịp đầu dẫn vào câu hỏi, nhịp cuối trả lời nó. Mã dữ kiện phải có trong bảng dữ kiện.
        {approved ? " Câu chuyện đang được duyệt: lưu bản sửa sẽ đưa nó về chờ duyệt. " : " "}
        Mạch kể bạn sửa thành “Bạn viết”: AI giữ nguyên khi viết lại; xoá hết nhịp thì AI viết lại.
      </p>
      {error && (
        <p className={unitStyles.error} role="alert">
          {error}
        </p>
      )}
    </Dialog>
  );
}
