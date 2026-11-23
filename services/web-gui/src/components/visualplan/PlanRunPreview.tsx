import styles from "../airun/AiRun.module.css";

/** The AI actions of step 9 whose result is previewed before it is stored. */
export type PlanRunKind = "beat" | "review";

type Row = Record<string, unknown>;

function list(v: unknown): Row[] {
  return Array.isArray(v) ? (v.filter((x) => x && typeof x === "object") as Row[]) : [];
}

function text(v: unknown): string {
  return typeof v === "string" ? v : "";
}

/**
 * What applying a run would store, read from the run's preview: the units of a beat, or the
 * review's findings. Only what the Creator needs to judge the reply is shown; the whole reply
 * stays one tab away.
 */
export function PlanRunPreview({ kind, preview }: { kind: PlanRunKind; preview: unknown }) {
  if (kind === "beat") {
    const units = list(preview);
    return (
      <div className={styles.preview} data-testid="ai-run-preview">
        <p className={styles.previewHead}>
          {units.length} unit sẽ được lưu cho beat này. Unit bạn đã sửa tay giữ các trường đó; mọi unit về “Chờ duyệt”.
        </p>
        <ol className={styles.previewList}>
          {units.map((u, i) => {
            const approach = (u.visual_approach ?? {}) as Row;
            const refs = Array.isArray(u.script_refs) ? u.script_refs.join(", ") : "";
            return (
              <li key={`${text(u.unit_id)}-${i}`}>
                <b>{text(u.unit_id) || "Unit mới"}</b> · {text(u.narration_summary) || "(chưa có tóm tắt)"}
                {refs && <div className={styles.previewSub}>Các câu: {refs}</div>}
                {text(u.communication_goal) && (
                  <div className={styles.previewSub}>Mục tiêu: {text(u.communication_goal)}</div>
                )}
                {text(approach.description) && (
                  <div className={styles.previewSub}>
                    Hình ({text(approach.method)}): {text(approach.description)}
                  </div>
                )}
              </li>
            );
          })}
        </ol>
      </div>
    );
  }
  const r = (preview ?? {}) as Row;
  const findings = list(r.findings);
  const open = findings.filter((f) => text(f.state) !== "fixed" && text(f.state) !== "accepted");
  const fixed = findings.filter((f) => text(f.state) === "fixed");
  return (
    <div className={styles.preview} data-testid="ai-run-preview">
      <p className={styles.previewHead}>
        Sau khi áp dụng có {open.length} vấn đề đang mở
        {fixed.length > 0 ? `, ${fixed.length} vấn đề AI không báo lại được đánh dấu đã sửa` : ""}.
      </p>
      <ul className={styles.previewList}>
        {open.map((f, i) => (
          <li key={`${text(f.id)}-${i}`}>
            <b>{text(f.unit_key)}</b> · {text(f.text)}
            {text(f.fix) && <div className={styles.previewSub}>Gợi ý: {text(f.fix)}</div>}
          </li>
        ))}
      </ul>
    </div>
  );
}
