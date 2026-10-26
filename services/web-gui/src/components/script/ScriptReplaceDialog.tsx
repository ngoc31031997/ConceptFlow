import { useEffect, useState } from "react";
import {
  ApiError,
  GENERIC_CONNECTION_ERROR,
  replaceInScript,
  type ScriptReplaceField,
  type ScriptReplaceResult,
} from "../../api/client";
import { Button, Dialog, FormField, TextInput } from "../ui";
import styles from "./ScriptReplaceDialog.module.css";

interface ScriptReplaceDialogProps {
  open: boolean;
  onClose: () => void;
  projectId: string;
  /** The lines were changed; the screen loads them again. */
  onApplied: () => void | Promise<void>;
}

function messageOf(err: unknown): string {
  return err instanceof ApiError && err.message ? err.message : GENERIC_CONNECTION_ERROR;
}

/**
 * "Tìm và thay" over the dialogue of the whole script (docs/ux-ui-design-rules.md,
 * rules 4 and 5): a word and what it becomes, where to write it (the wording the voice
 * reads, or the text on screen), then a preview of every line that changes before
 * anything is saved. Beats already approved are left as they are and counted; the
 * beats that change go back to review. Changing the word, the replacement or the
 * options clears the preview, so what is applied is always what was shown.
 */
export function ScriptReplaceDialog({
  open,
  onClose,
  projectId,
  onApplied,
}: ScriptReplaceDialogProps) {
  const [find, setFind] = useState("");
  const [replace, setReplace] = useState("");
  const [field, setField] = useState<ScriptReplaceField>("spoken");
  const [matchCase, setMatchCase] = useState(true);
  const [preview, setPreview] = useState<ScriptReplaceResult | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!open) return;
    setFind("");
    setReplace("");
    setField("spoken");
    setMatchCase(true);
    setPreview(null);
    setError(null);
  }, [open]);

  function change<T>(set: (value: T) => void) {
    return (value: T) => {
      set(value);
      setPreview(null);
      setError(null);
    };
  }

  async function run(apply: boolean) {
    setBusy(true);
    setError(null);
    try {
      const result = await replaceInScript(projectId, { find, replace, field, matchCase, apply });
      if (!apply) {
        setPreview(result);
        return;
      }
      await onApplied();
      onClose();
    } catch (err) {
      setError(messageOf(err));
    } finally {
      setBusy(false);
    }
  }

  const beatsChanged = new Set(preview?.matches.map((m) => m.beat)).size;
  const skipped = preview?.skippedApproved ?? [];

  return (
    <Dialog
      open={open}
      onClose={onClose}
      title="Tìm và thay trong lời thoại"
      size="lg"
      hasInput
      testId="script-replace"
      footer={
        <>
          <Button variant="ghost" onClick={onClose} data-testid="script-replace-cancel">
            Huỷ
          </Button>
          <Button
            variant="view"
            onClick={() => void run(false)}
            disabled={busy || find.trim() === ""}
            data-testid="script-replace-preview"
          >
            Xem trước
          </Button>
          <Button
            variant="impact"
            onClick={() => void run(true)}
            disabled={busy || !preview || preview.matches.length === 0}
            data-testid="script-replace-apply"
          >
            {preview && preview.matches.length > 0 ? `Đổi ${preview.matches.length} câu` : "Đổi"}
          </Button>
        </>
      }
    >
      <div className={styles.row}>
        <FormField label="Từ cần tìm">
          <TextInput
            value={find}
            onChange={(e) => change(setFind)(e.target.value)}
            placeholder="Ví dụ: TRNG"
            data-testid="script-replace-find"
          />
        </FormField>
        <FormField label="Thay bằng">
          <TextInput
            value={replace}
            onChange={(e) => change(setReplace)(e.target.value)}
            placeholder="Ví dụ: TI A EN GI"
            data-testid="script-replace-with"
          />
        </FormField>
      </div>
      <div className={styles.choice} role="radiogroup" aria-label="Đổi vào đâu">
        <label>
          <input
            type="radio"
            name="script-replace-field"
            checked={field === "spoken"}
            onChange={() => change(setField)("spoken")}
            data-testid="script-replace-field-spoken"
          />
          Lời đọc (chữ trên màn hình giữ nguyên)
        </label>
        <label>
          <input
            type="radio"
            name="script-replace-field"
            checked={field === "text"}
            onChange={() => change(setField)("text")}
            data-testid="script-replace-field-text"
          />
          Chữ hiển thị
        </label>
        <label>
          <input
            type="checkbox"
            checked={matchCase}
            onChange={(e) => change(setMatchCase)(e.target.checked)}
            data-testid="script-replace-case"
          />
          Phân biệt hoa thường
        </label>
      </div>
      <p className={styles.hint}>Chỉ đổi khi cả từ khớp: tìm “TRNG” không đổi “TRNGS”.</p>
      {preview && (
        <>
          {preview.matches.length === 0 ? (
            <p className={styles.hint} role="status" data-testid="script-replace-none">
              Không có câu nào cần đổi{skipped.length > 0 ? " ở các beat chưa duyệt" : ""}.
            </p>
          ) : (
            <>
              <p className={styles.hint} role="status" data-testid="script-replace-summary">
                {preview.matches.length} câu ở {beatsChanged} beat sẽ đổi. {beatsChanged} beat đó về
                chờ duyệt.
              </p>
              <ul className={styles.changes} data-testid="script-replace-matches">
                {preview.matches.map((m) => (
                  <li
                    key={m.line}
                    className={styles.change}
                    data-testid={`script-replace-match-${m.line}`}
                  >
                    <span className={styles.id}>{m.line}</span>{" "}
                    <span className={styles.before}>{m.before}</span> → {m.after}
                  </li>
                ))}
              </ul>
            </>
          )}
          {skipped.length > 0 && (
            <p className={styles.hint} data-testid="script-replace-skipped">
              Beat đã duyệt giữ nguyên:{" "}
              {skipped.map((s) => `${s.beat} (${s.lines} câu)`).join(", ")}. Bỏ duyệt beat đó nếu
              muốn đổi cả ở đấy.
            </p>
          )}
        </>
      )}
      {error && (
        <p className={styles.error} role="alert" data-testid="script-replace-error">
          {error}
        </p>
      )}
    </Dialog>
  );
}
