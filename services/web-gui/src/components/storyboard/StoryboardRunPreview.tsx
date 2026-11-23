import styles from "../airun/AiRun.module.css";

/** The AI actions of step 11 whose result is previewed before it is stored. */
export type StoryboardRunKind = "assets" | "asset_detail" | "unit" | "review";

type Row = Record<string, unknown>;

function list(v: unknown): Row[] {
  return Array.isArray(v) ? (v.filter((x) => x && typeof x === "object") as Row[]) : [];
}

function text(v: unknown): string {
  if (Array.isArray(v)) return v.filter((x) => typeof x === "string" && x.trim() !== "").join(", ");
  return typeof v === "string" ? v : "";
}

/**
 * What applying a run would store, read from the run's preview: the shots of a unit, the asset
 * list, one asset's detail, or the review's findings. Only what the Creator needs to judge the
 * reply is shown; the whole reply stays one tab away.
 */
export function StoryboardRunPreview({ kind, preview }: { kind: StoryboardRunKind; preview: unknown }) {
  if (kind === "unit") {
    const shots = list(preview);
    return (
      <div className={styles.preview} data-testid="ai-run-preview">
        <p className={styles.previewHead}>{shots.length} shot sẽ được lưu cho unit này, rồi vẽ khung từng shot.</p>
        <ol className={styles.previewList}>
          {shots.map((s) => (
            <li key={String(s.shot_id)}>
              <b>{String(s.shot_id)}</b> · {text(s.shot_purpose) || "(chưa có mục đích)"}
              {text(s.action_summary) && <div className={styles.previewSub}>{text(s.action_summary)}</div>}
              {text(s.asset_refs) && <div className={styles.previewSub}>Tài sản: {text(s.asset_refs)}</div>}
            </li>
          ))}
        </ol>
      </div>
    );
  }
  if (kind === "assets") {
    const assets = list(preview);
    return (
      <div className={styles.preview} data-testid="ai-run-preview">
        <p className={styles.previewHead}>
          Danh sách gồm {assets.length} tài sản. Tài sản AI bỏ đi mà không shot nào dùng sẽ bị xoá khi áp dụng.
        </p>
        <ul className={styles.previewList}>
          {assets.map((a, i) => (
            <li key={`${String(a.name)}-${i}`}>
              <b>{String(a.name)}</b> · {text(a.kind)} · {text(a.tier)}
              {text(a.same_as) && <span className={styles.previewSub}> (giữ {text(a.same_as)})</span>}
              {text(a.units) && <div className={styles.previewSub}>Unit: {text(a.units)}</div>}
            </li>
          ))}
        </ul>
      </div>
    );
  }
  if (kind === "asset_detail") {
    const d = (preview ?? {}) as Row;
    return (
      <dl className={styles.preview} data-testid="ai-run-preview">
        <dt>Mô tả</dt>
        <dd>{text(d.portrayal)}</dd>
        <dt>Màu</dt>
        <dd>{text(d.palette_tokens) || "(không có)"}</dd>
        <dt>Không được gợi</dt>
        <dd>{text(d.must_not_imply) || "(không có)"}</dd>
        <dt>Biểu cảm</dt>
        <dd>{text(d.expressions) || "(không có)"}</dd>
      </dl>
    );
  }
  const r = (preview ?? {}) as Row;
  const findings = list(r.findings);
  const notes = text(r.notes);
  return (
    <div className={styles.preview} data-testid="ai-run-preview">
      <p className={styles.previewHead}>
        {findings.length === 0 ? "AI không báo vấn đề nào." : `AI báo ${findings.length} vấn đề.`} Vấn đề cũ AI không
        báo lại sẽ được đánh dấu đã sửa khi áp dụng.
      </p>
      <ul className={styles.previewList}>
        {findings.map((f, i) => (
          <li key={i}>
            <b>{text(f.unit_key)}</b> · {text(f.text)}
            {text(f.fix) && <div className={styles.previewSub}>Gợi ý: {text(f.fix)}</div>}
          </li>
        ))}
      </ul>
      {notes && <p className={styles.previewSub}>Ghi chú: {notes}</p>}
    </div>
  );
}
