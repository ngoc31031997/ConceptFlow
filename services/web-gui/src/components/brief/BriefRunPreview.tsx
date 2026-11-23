import styles from "../airun/AiRun.module.css";

/** The AI actions of step 2 whose result is previewed before it is stored. */
export type BriefRunKind = "brief" | "review";

type Row = Record<string, unknown>;

const RATING: Record<string, string> = { good: "Tốt", fair: "Tạm ổn", weak: "Yếu" };

const ASPECTS: { key: string; label: string }[] = [
  { key: "audience_fit", label: "Khán giả" },
  { key: "duration_fit", label: "Thời lượng" },
  { key: "objectives", label: "Mục tiêu học tập" },
  { key: "avoid_list", label: "Cần tránh" },
];

function strings(v: unknown): string[] {
  return Array.isArray(v) ? v.filter((x): x is string => typeof x === "string" && x.trim() !== "") : [];
}

function Lines({ title, items, ordered }: { title: string; items: string[]; ordered?: boolean }) {
  const Tag = ordered ? "ol" : "ul";
  return (
    <>
      <p className={styles.previewSub}>{title}</p>
      <Tag className={styles.previewList}>
        {items.length === 0 ? <li>(trống)</li> : items.map((x) => <li key={x}>{x}</li>)}
      </Tag>
    </>
  );
}

/**
 * What applying a run would store, read from the run's preview: the brief's five fields (the ones
 * the Creator wrote are already kept in it), or the AI's review of the brief.
 */
export function BriefRunPreview({ kind, preview }: { kind: BriefRunKind; preview: unknown }) {
  const p = (preview && typeof preview === "object" ? preview : {}) as Row;
  if (kind === "review") {
    return (
      <div className={styles.preview} data-testid="ai-run-preview">
        <p className={styles.previewHead}>Đánh giá này sẽ được lưu; brief không đổi.</p>
        <ul className={styles.previewList}>
          {ASPECTS.map((a) => {
            const aspect = (p[a.key] ?? {}) as Row;
            return (
              <li key={a.key}>
                <b>{a.label}</b> · {RATING[String(aspect.rating)] ?? String(aspect.rating ?? "")}
                <div className={styles.previewSub}>{String(aspect.comment ?? "")}</div>
              </li>
            );
          })}
        </ul>
        {strings(p.unclear).length > 0 && <Lines title="Chỗ chưa rõ" items={strings(p.unclear)} />}
        {strings(p.suggestions).length > 0 && <Lines title="Gợi ý cải thiện" items={strings(p.suggestions)} />}
      </div>
    );
  }
  const seconds = typeof p.duration_seconds === "number" ? p.duration_seconds : 0;
  return (
    <div className={styles.preview} data-testid="ai-run-preview">
      <p className={styles.previewHead}>Brief này sẽ được lưu và chờ duyệt.</p>
      <p className={styles.previewSub}>Khán giả</p>
      <p>{String(p.audience ?? "") || "(trống)"}</p>
      <p className={styles.previewSub}>Thời lượng</p>
      <p>{seconds > 0 ? `${seconds} giây` : "(trống)"}</p>
      <p className={styles.previewSub}>Giọng điệu</p>
      <p>{String(p.tone ?? "") || "(trống)"}</p>
      <Lines title="Mục tiêu học tập" items={strings(p.learning_objectives)} ordered />
      <Lines title="Cần tránh" items={strings(p.avoid)} />
    </div>
  );
}
