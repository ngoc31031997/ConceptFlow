import styles from "../airun/AiRun.module.css";

type Row = Record<string, unknown>;

function text(v: unknown): string {
  if (Array.isArray(v)) return v.filter((x) => typeof x === "string" && x.trim() !== "").join(", ");
  return typeof v === "string" ? v : "";
}

/**
 * What applying a beat run would store, read from the run's preview: the beat the AI wrote (its
 * title, goal, content, length and the line that joins it to the next beat) and where it goes.
 * `kind` "add" puts a new beat behind another; "write" writes one beat of the sheet again.
 */
export function BeatRunPreview({ kind, preview }: { kind: "add" | "write"; preview: unknown }) {
  const p = (preview ?? {}) as Row;
  const beat = (p.beat ?? {}) as Row;
  const after = text(p.after);
  const key = text(p.key);
  const head =
    kind === "add"
      ? after
        ? `Beat mới sẽ được thêm sau ${after}.`
        : "Beat mới sẽ được thêm vào bảng."
      : `Beat ${key} sẽ được viết lại như dưới đây; chức năng và dữ kiện của bộ khung giữ nguyên.`;
  return (
    <div className={styles.preview} data-testid="ai-run-preview">
      <p className={styles.previewHead}>{head}</p>
      <ul className={styles.previewList}>
        <li>
          <b>{text(beat.title) || "(chưa có tiêu đề)"}</b> · {text(beat.fn)} · {String(beat.dur ?? "")} s
          {text(beat.goal) && <div className={styles.previewSub}>Mục tiêu: {text(beat.goal)}</div>}
          {text(beat.content) && <div className={styles.previewSub}>Nội dung: {text(beat.content)}</div>}
          {text(beat.facts) && <div className={styles.previewSub}>Dữ kiện: {text(beat.facts)}</div>}
          {text(beat.next) && <div className={styles.previewSub}>Nối sang beat sau: {text(beat.next)}</div>}
        </li>
      </ul>
    </div>
  );
}
