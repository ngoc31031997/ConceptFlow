import styles from "../airun/AiRun.module.css";

/** The AI actions of step 6 whose result is previewed before it is stored. */
export type ScriptRunKind = "beat" | "suggestion";

type Row = Record<string, unknown>;

function lines(preview: unknown): Row[] {
  const l = (preview as { lines?: unknown } | null)?.lines;
  return Array.isArray(l) ? (l.filter((x) => x && typeof x === "object") as Row[]) : [];
}

function facts(v: unknown): string {
  return Array.isArray(v) ? v.filter((x) => typeof x === "string").join(", ") : "";
}

/**
 * What applying a run would store, read from the run's preview: the lines of a beat's dialogue
 * (with the wording the voice reads when it differs), or the one sentence the AI suggests for a line.
 */
export function ScriptRunPreview({ kind, preview }: { kind: ScriptRunKind; preview: unknown }) {
  if (kind === "suggestion") {
    const text = (preview as { text?: unknown } | null)?.text;
    return (
      <div className={styles.preview} data-testid="ai-run-preview">
        <p className={styles.previewHead}>
          Câu AI gợi ý thay. Áp dụng chỉ hiện gợi ý cạnh câu; câu chỉ đổi khi bạn bấm “Áp dụng” ở
          gợi ý đó.
        </p>
        <p>{typeof text === "string" ? text : ""}</p>
      </div>
    );
  }
  const rows = lines(preview);
  return (
    <div className={styles.preview} data-testid="ai-run-preview">
      <p className={styles.previewHead}>
        {rows.length} câu sẽ được lưu thành lời thoại của beat này; beat về “Chưa duyệt”.
      </p>
      <ol className={styles.previewList}>
        {rows.map((l) => (
          <li key={String(l.id)}>
            <b>{String(l.id)}</b> · {String(l.text ?? "")}
            {typeof l.spoken === "string" && l.spoken !== "" && (
              <div className={styles.previewSub}>Đọc: {l.spoken}</div>
            )}
            {facts(l.facts) && <div className={styles.previewSub}>Dữ kiện: {facts(l.facts)}</div>}
          </li>
        ))}
      </ol>
    </div>
  );
}
