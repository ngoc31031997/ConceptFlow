import type { BeatState } from "../../api/client";
import researchStyles from "../research/research.module.css";
import { Card } from "../ui";
import styles from "./beats.module.css";

/**
 * The shape of the sheet: a strip with one segment per beat, as wide as its
 * target length, and the checks over the whole sheet. A check that fails is a
 * warning, never a block. While the skeleton is reviewed no length is decided
 * yet, so the strip gives way to the checks alone.
 */
export function StructureCard({ state }: { state: BeatState }) {
  const skeleton = state.phase === "skeleton";
  return (
    <Card data-testid="beats-structure-card">
      <div className={researchStyles.body}>
        <div className={researchStyles.head}>
          <h3>Cấu trúc</h3>
          {!skeleton && <span className={researchStyles.hint}>độ rộng theo thời lượng mục tiêu</span>}
        </div>
        {!skeleton && (
          <div className={styles.scroll}>
            <div className={styles.timeline} data-testid="beats-timeline">
              {state.beats.map((beat, i) => (
                <div key={beat.key} className={styles.segment} style={{ flex: beat.dur }} title={beat.title}>
                  B{i + 1} · {beat.fn}
                  <br />
                  {beat.dur} s
                </div>
              ))}
            </div>
          </div>
        )}
        {!skeleton && state.rebalanced > 0 && (
          <p className={researchStyles.hint} data-testid="beats-rebalanced">
            Đã cân lại thời lượng của {state.rebalanced} beat cho khớp mục tiêu {state.targetLabel}. Chỉnh bằng nút ± 5 s ở từng beat nếu thấy chưa hợp lý.
          </p>
        )}
        <div className={styles.checks} data-testid="beats-checks">
          {state.checks.map((check) => (
            <span key={check.text} className={`${styles.check} ${check.ok ? "" : styles.checkWarn}`} data-ok={check.ok}>
              {check.ok ? "✓" : "!"} {check.text}
            </span>
          ))}
        </div>
      </div>
    </Card>
  );
}
