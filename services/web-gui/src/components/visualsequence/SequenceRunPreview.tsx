import styles from "../airun/AiRun.module.css";

/** The AI actions of step 10 whose result is previewed before it is stored. */
export type SequenceRunKind = "motifs" | "unit" | "review";

type Row = Record<string, unknown>;

function list(v: unknown): Row[] {
  return Array.isArray(v) ? (v.filter((x) => x && typeof x === "object") as Row[]) : [];
}

function text(v: unknown): string {
  if (Array.isArray(v)) return v.filter((x) => typeof x === "string" && x.trim() !== "").join(", ");
  return typeof v === "string" ? v : "";
}

/**
 * What applying a run would store, read from the run's preview: the motifs of the video, the
 * item of one unit, or the review's findings. Only what the Creator needs to judge the reply is
 * shown; the whole reply stays one tab away.
 */
export function SequenceRunPreview({ kind, preview }: { kind: SequenceRunKind; preview: unknown }) {
  if (kind === "motifs") {
    const motifs = list((preview as Row | null)?.motifs);
    return (
      <div className={styles.preview} data-testid="ai-run-preview">
        <p className={styles.previewHead}>
          Danh sách gồm {motifs.length} motif, về “Chờ duyệt” khi áp dụng.
        </p>
        <ul className={styles.previewList}>
          {motifs.map((m, i) => (
            <li key={`${String(m.motif_id)}-${i}`}>
              <b>{text(m.motif_id)}</b> · {text(m.name) || "(chưa có tên)"}
              {text(m.role) && <span className={styles.previewSub}> · {text(m.role)}</span>}
              {text(m.description) && (
                <div className={styles.previewSub}>{text(m.description)}</div>
              )}
              {text(m.planned_units) && (
                <div className={styles.previewSub}>Unit: {text(m.planned_units)}</div>
              )}
            </li>
          ))}
        </ul>
      </div>
    );
  }
  if (kind === "unit") {
    const it = (preview ?? {}) as Row;
    const pacing = (it.pacing ?? {}) as Row;
    const tr = it.transition_from_previous as Row | null | undefined;
    return (
      <dl className={styles.preview} data-testid="ai-run-preview">
        <dt>Vai trò trong mạch</dt>
        <dd>{text(it.purpose_in_flow)}</dd>
        <dt>Nhịp</dt>
        <dd>
          {[text(pacing.name), text(pacing.tempo)].filter(Boolean).join(" · ")}
          {text(pacing.description) && (
            <div className={styles.previewSub}>{text(pacing.description)}</div>
          )}
        </dd>
        <dt>Chuyển cảnh vào</dt>
        <dd>
          {tr ? (
            <>
              {text(tr.type)}
              {text(tr.description) && (
                <div className={styles.previewSub}>{text(tr.description)}</div>
              )}
              {text(tr.via_motif) && (
                <div className={styles.previewSub}>Nối qua {text(tr.via_motif)}</div>
              )}
            </>
          ) : (
            "(unit mở đầu video)"
          )}
        </dd>
        <dt>Motif</dt>
        <dd>{text(it.motif_refs) || "(không có)"}</dd>
      </dl>
    );
  }
  const r = (preview ?? {}) as Row;
  const findings = list(r.findings);
  return (
    <div className={styles.preview} data-testid="ai-run-preview">
      <p className={styles.previewHead}>
        Sau khi gộp với các vấn đề cũ có {findings.length} vấn đề. Vấn đề cũ AI không báo lại sẽ
        được đánh dấu đã sửa khi áp dụng.
      </p>
      <ul className={styles.previewList}>
        {findings.map((f, i) => (
          <li key={i}>
            <b>{text(f.unit_key)}</b> · {text(f.text)}
            {text(f.state) && text(f.state) !== "open" && (
              <span className={styles.previewSub}> ({text(f.state)})</span>
            )}
            {text(f.fix) && <div className={styles.previewSub}>Gợi ý: {text(f.fix)}</div>}
          </li>
        ))}
      </ul>
    </div>
  );
}
