import type { UnitImpact } from "../../hooks/useUnitImpact";
import styles from "./units.module.css";

interface ImpactListProps {
  /** What the list is the consequence of ("Lưu bản sửa thì phần phụ thuộc sẽ làm lại:"). */
  head: string;
  impact: UnitImpact;
  testId?: string;
}

/**
 * The later work an edit or redo sends back, inside the dialog that does it
 * (docs/ux-ui-design-rules.md, rule 5). Nothing is drawn when nothing depends
 * on the unit yet.
 */
export function ImpactList({ head, impact, testId }: ImpactListProps) {
  if (impact.items === null) {
    return (
      <p className={styles.note} role="status">
        Đang kiểm tra phần phụ thuộc…
      </p>
    );
  }
  if (impact.failed) {
    return (
      <p className={styles.error} role="alert" data-testid={testId}>
        Không tải được danh sách phần phụ thuộc. Những bước đã làm sau mục này có thể phải làm lại.
      </p>
    );
  }
  if (impact.items.length === 0) return null;
  return (
    <div className={styles.impact} data-testid={testId}>
      <b>{head}</b>
      <ul>
        {impact.items.map((item) => (
          <li key={item}>{item}</li>
        ))}
      </ul>
    </div>
  );
}
