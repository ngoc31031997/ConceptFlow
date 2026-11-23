import type { VisualConceptUnitKey } from "../../api/client";
import styles from "../airun/AiRun.module.css";
import { PART_SCHEMAS } from "./conceptSchema";
import { PartView } from "./PartView";

/** The AI actions of step 8 whose result is previewed before it is stored. */
export type ConceptRunKind = "unit" | "review";

type Row = Record<string, unknown>;

function rows(v: unknown): Row[] {
  return Array.isArray(v) ? (v.filter((x) => x && typeof x === "object") as Row[]) : [];
}

function text(v: unknown): string {
  return typeof v === "string" ? v : "";
}

/**
 * What applying a run of step 8 would store, read from the run's preview: the content of the
 * rewritten part, or the review's findings merged with the earlier ones.
 */
export function ConceptRunPreview({ kind, preview }: { kind: ConceptRunKind; preview: unknown }) {
  const p = (preview ?? {}) as Row;
  if (kind === "unit") {
    const schema = PART_SCHEMAS[text(p.key) as VisualConceptUnitKey];
    const content = (p.content ?? {}) as Record<string, unknown>;
    return (
      <div className={styles.preview} data-testid="ai-run-preview">
        <p className={styles.previewHead}>Nội dung mới của phần này; sau khi áp dụng phần về “chờ duyệt”.</p>
        {schema ? <PartView schema={schema} content={content} mine={[]} /> : <pre>{JSON.stringify(content, null, 2)}</pre>}
      </div>
    );
  }
  const findings = rows(p.findings);
  const open = findings.filter((f) => f.state === "open");
  const fixed = findings.filter((f) => f.state === "fixed");
  const notes = Array.isArray(p.notes) ? p.notes.filter((n): n is string => typeof n === "string") : [];
  return (
    <div className={styles.preview} data-testid="ai-run-preview">
      <p className={styles.previewHead}>
        {open.length === 0 ? "Không còn vấn đề nào mở." : `${open.length} vấn đề đang mở.`}
        {fixed.length > 0 ? ` ${fixed.length} vấn đề cũ AI không báo lại được đánh dấu đã sửa.` : ""}
      </p>
      <ul className={styles.previewList}>
        {open.map((f, i) => (
          <li key={i}>
            <b>{text(f.unit_key)}</b> · {text(f.text)}
            {text(f.fix) && <div className={styles.previewSub}>Gợi ý: {text(f.fix)}</div>}
          </li>
        ))}
      </ul>
      {notes.length > 0 && <p className={styles.previewSub}>Nhận xét: {notes.join(" · ")}</p>}
    </div>
  );
}
