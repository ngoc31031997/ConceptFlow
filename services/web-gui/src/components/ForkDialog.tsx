import { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { useNavigate } from "react-router-dom";
import { forkProject, ApiError } from "../api/client";
import { FLOW_LABELS, FORK_STEPS } from "../utils/flow";
import glass from "../styles/glass.module.css";
import styles from "./ForkDialog.module.css";

interface ForkDialogProps {
  projectId: string;
  onClose: () => void;
}

/**
 * Tạo project MỚI từ video này, làm lại từ một bước trong 1-5. Bản gốc không
 * đổi. Bước nào giữ, bước nào làm lại hiện ngay bên cạnh lựa chọn, để chọn
 * xong là biết mình sẽ có gì.
 */
export function ForkDialog({ projectId, onClose }: ForkDialogProps) {
  const navigate = useNavigate();
  const [from, setFrom] = useState(4);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const first = useRef<HTMLInputElement>(null);

  useEffect(() => {
    first.current?.focus();
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && !busy && onClose();
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [busy, onClose]);

  async function create() {
    setBusy(true);
    setError(null);
    try {
      const out = await forkProject(projectId, from);
      onClose();
      // Bản mới là một draft: mở nó ở đúng bước vừa chọn để làm lại.
      navigate(`/projects/${out.project_id}/resume?step=${from}`);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Không tạo được bản mới. Thử lại sau.");
      setBusy(false);
    }
  }

  return createPortal(
    <div className={styles.backdrop} role="presentation" data-testid="fork-dialog-backdrop">
      <div className={`${glass.card} ${styles.dialog}`} role="dialog" aria-labelledby="fork-title" data-testid="fork-dialog">
        <h2 id="fork-title" className={styles.title}>
          Tạo bản mới từ video này
        </h2>
        <p className={styles.lead}>
          Video gốc giữ nguyên. Bản mới bắt đầu làm lại từ bước bạn chọn; các bước trước nó được sao chép sang.
        </p>
        <fieldset className={styles.options}>
          <legend className={styles.legend}>Làm lại từ</legend>
          {FORK_STEPS.map((opt, i) => (
            <label key={opt.step} htmlFor={`fork-from-${opt.step}`} className={`${styles.option} ${from === opt.step ? styles.on : ""}`}>
              <input
                ref={i === 0 ? first : undefined}
                id={`fork-from-${opt.step}`}
                type="radio"
                name="fork-from"
                checked={from === opt.step}
                onChange={() => setFrom(opt.step)}
                data-testid={`fork-from-${opt.step}`}
              />
              <b>{FLOW_LABELS[opt.step - 1]}</b>
              <small>{opt.keeps}</small>
            </label>
          ))}
        </fieldset>
        <p className={styles.note}>
          Nhạc nền không được mang sang (file nằm trong thư mục của video gốc): chọn lại ở bước Cấu hình nếu cần.
        </p>
        {error && (
          <p role="alert" className={styles.error} data-testid="fork-error">
            {error}
          </p>
        )}
        <div className={styles.actions}>
          <button type="button" className={glass.ghostBtn} onClick={onClose} disabled={busy} data-testid="fork-cancel">
            Đóng
          </button>
          <button type="button" className={glass.btnPrimary} onClick={create} disabled={busy} data-testid="fork-confirm">
            {busy ? "Đang tạo..." : "Tạo bản mới"}
          </button>
        </div>
      </div>
    </div>,
    document.body,
  );
}
