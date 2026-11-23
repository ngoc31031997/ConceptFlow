import styles from "../airun/AiRun.module.css";

/** The AI actions of the idea step whose result is previewed before it is stored. */
export type IdeaRunKind = "idea" | "review";

type Row = Record<string, unknown>;

const RATING_TEXT: Record<string, string> = { good: "Tốt", fair: "Tạm ổn", weak: "Yếu" };

function strings(v: unknown): string[] {
  return Array.isArray(v)
    ? v.filter((x): x is string => typeof x === "string" && x.trim() !== "")
    : [];
}

function aspect(label: string, v: unknown) {
  const a = (v ?? {}) as Row;
  const rating = typeof a.rating === "string" ? (RATING_TEXT[a.rating] ?? a.rating) : "";
  return (
    <li key={label}>
      <b>{label}</b> · {rating}
      {typeof a.comment === "string" && a.comment && (
        <div className={styles.previewSub}>{a.comment}</div>
      )}
    </li>
  );
}

/**
 * What applying a run would store, read from the run's preview: the three fields of the idea
 * (a field the Creator wrote already holds their own text), or the AI's review of it.
 */
export function IdeaRunPreview({ kind, preview }: { kind: IdeaRunKind; preview: unknown }) {
  const p = (preview ?? {}) as Row;
  if (kind === "idea") {
    const points = strings(p.curiosity_points);
    return (
      <dl className={styles.preview} data-testid="ai-run-preview">
        <dt>Câu hỏi trung tâm</dt>
        <dd>{typeof p.central_question === "string" ? p.central_question : ""}</dd>
        <dt>Điểm gây tò mò</dt>
        <dd>
          {points.length > 0 ? (
            <ul className={styles.previewList}>
              {points.map((x) => (
                <li key={x}>{x}</li>
              ))}
            </ul>
          ) : (
            "(trống)"
          )}
        </dd>
        <dt>Phạm vi</dt>
        <dd>{typeof p.scope === "string" ? p.scope : ""}</dd>
      </dl>
    );
  }
  const unclear = strings(p.unclear);
  const suggestions = strings(p.suggestions);
  return (
    <div className={styles.preview} data-testid="ai-run-preview">
      <ul className={styles.previewList}>
        {aspect("Độ rõ", p.clarity)}
        {aspect("Khả năng nghiên cứu", p.researchability)}
        {aspect("Phạm vi", p.scope_fit)}
      </ul>
      {unclear.length > 0 && (
        <>
          <p className={styles.previewHead}>Chỗ chưa rõ</p>
          <ul className={styles.previewList}>
            {unclear.map((x) => (
              <li key={x}>{x}</li>
            ))}
          </ul>
        </>
      )}
      {suggestions.length > 0 && (
        <>
          <p className={styles.previewHead}>Gợi ý cải thiện</p>
          <ul className={styles.previewList}>
            {suggestions.map((x) => (
              <li key={x}>{x}</li>
            ))}
          </ul>
        </>
      )}
    </div>
  );
}
