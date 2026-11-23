import styles from "../airun/AiRun.module.css";

/** The AI actions of step 7 whose result is previewed before it is stored. */
export type LockRunKind = "review" | "handoff";

type Row = Record<string, unknown>;

function list(v: unknown): Row[] {
  return Array.isArray(v) ? (v.filter((x) => x && typeof x === "object") as Row[]) : [];
}

function strings(v: unknown): string[] {
  return Array.isArray(v)
    ? v.filter((x): x is string => typeof x === "string" && x.trim() !== "")
    : [];
}

function text(v: unknown): string {
  return typeof v === "string" ? v : "";
}

interface LockRunPreviewProps {
  kind: LockRunKind;
  preview: unknown;
  /** The handoff sections in display order, with their labels. */
  sections?: { key: string; label: string }[];
}

/**
 * What applying a run would store: the script review with the AI's findings, or one beat's
 * handoff note with the glossary terms it adds. Only what the Creator needs to judge the reply is
 * shown; the whole reply stays one tab away.
 */
export function LockRunPreview({ kind, preview, sections = [] }: LockRunPreviewProps) {
  const p = (preview ?? {}) as Row;
  if (kind === "handoff") {
    const note = (p.note ?? {}) as Row;
    const terms = list(p.terms);
    return (
      <div className={styles.preview} data-testid="ai-run-preview">
        <p className={styles.previewHead}>
          Ghi chú bàn giao sẽ được lưu cho beat này, chờ bạn duyệt.
        </p>
        <dl>
          {sections.map((s) => {
            const items = strings(note[s.key]);
            return (
              <div key={s.key}>
                <dt>{s.label}</dt>
                <dd>{items.length === 0 ? "(không có)" : items.join(" · ")}</dd>
              </div>
            );
          })}
        </dl>
        {terms.length > 0 && (
          <>
            <p className={styles.previewSub}>Thuật ngữ mới gộp vào bảng chung:</p>
            <ul className={styles.previewList}>
              {terms.map((t) => (
                <li key={text(t.term)}>
                  <b>{text(t.term)}</b> · nói là “{text(t.say)}” · vẽ là “{text(t.draw)}”
                  {text(t.avoid) && (
                    <span className={styles.previewSub}> · tránh “{text(t.avoid)}”</span>
                  )}
                </li>
              ))}
            </ul>
          </>
        )}
      </div>
    );
  }
  const findings = list(p.findings);
  const notes = strings(p.ai_notes);
  return (
    <div className={styles.preview} data-testid="ai-run-preview">
      <p className={styles.previewHead}>
        Kết quả rà soát có {findings.length} mục của AI. Mục cũ AI không báo lại sẽ được đánh dấu đã
        sửa khi áp dụng.
      </p>
      <ul className={styles.previewList}>
        {findings.map((f) => (
          <li key={text(f.id)}>
            <b>{text(f.title)}</b>
            {text(f.line) && <span className={styles.previewSub}> · câu {text(f.line)}</span>}
            {text(f.detail) && <div className={styles.previewSub}>{text(f.detail)}</div>}
            {text(f.fix) && <div className={styles.previewSub}>Gợi ý: {text(f.fix)}</div>}
          </li>
        ))}
      </ul>
      {notes.length > 0 && <p className={styles.previewSub}>Ghi chú: {notes.join(" · ")}</p>}
    </div>
  );
}
