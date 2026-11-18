import { Fragment } from "react";
import type { VisualSequenceState } from "../../api/client";
import researchStyles from "../research/research.module.css";
import { Card } from "../ui";
import { seconds } from "../visualplan/planModel";
import styles from "./sequence.module.css";
import { TEMPOS, TEMPO_LABELS, allUnits, clock, motifName, motifSlot } from "./sequenceModel";

/** Width of a segment per planned second, and its bounds, so short and long units stay readable. */
const PX_PER_SECOND = 8;
const MIN_WIDTH = 72;
const MAX_WIDTH = 220;

interface SequenceFlowStripProps {
  state: VisualSequenceState;
  /** Scrolls to a unit's card. */
  onOpenUnit: (unitId: string) => void;
}

/**
 * Section 4 of step 10: every unit of the plan in order as one strip. A
 * segment is as wide as the unit is long, tinted by its tempo, named by its
 * pacing and dotted with its motifs; between two segments stands the
 * transition into the second.
 */
export function SequenceFlowStrip({ state, onOpenUnit }: SequenceFlowStripProps) {
  const units = allUnits(state);
  const total = units.reduce((s, u) => s + u.planned_s, 0);

  return (
    <Card data-testid="seq-flow">
      <div className={researchStyles.body}>
        <div className={researchStyles.head}>
          <h3>4. Dòng chảy</h3>
          <span className={researchStyles.hint} data-testid="seq-flow-count">
            {units.length} unit · {clock(total)} · {state.coverage.units_sequenced}/
            {state.coverage.units_total} đã có chuỗi
          </span>
        </div>
        <div className={styles.strip} aria-label="Dòng chảy hình ảnh">
          {units.map((u, i) => {
            const it = u.item;
            const width = Math.min(MAX_WIDTH, Math.max(MIN_WIDTH, u.planned_s * PX_PER_SECOND));
            const tempo = it?.pacing.tempo ? styles[it.pacing.tempo] : "";
            return (
              <Fragment key={u.unit_id}>
                {i > 0 && (
                  <span className={styles.tr} data-testid={`seq-flow-tr-${u.unit_id}`}>
                    {it?.transition_from_previous?.type || "…"}
                  </span>
                )}
                <button
                  type="button"
                  className={`${styles.seg} ${it ? tempo : styles.segEmpty}`}
                  style={{ width }}
                  onClick={() => onOpenUnit(u.unit_id)}
                  title={`${u.unit_id} · ${seconds(u.planned_s)}`}
                  data-testid={`seq-flow-${u.unit_id}`}
                >
                  <span className={styles.segId}>{u.unit_id}</span>
                  <span>{it ? it.pacing.name || "chưa đặt nhịp" : "chưa viết"}</span>
                  <span className={styles.dots}>
                    {(it?.motif_refs ?? []).map((m) => (
                      <i
                        key={m}
                        className={`${styles.dot} ${styles[`c${motifSlot(state, m)}`] ?? ""}`}
                        title={`${m} · ${motifName(state, m)}`}
                      />
                    ))}
                  </span>
                </button>
              </Fragment>
            );
          })}
        </div>
        <div className={styles.legend}>
          {TEMPOS.map((t) => (
            <span key={t}>
              <i className={`${styles.swatch} ${styles[t]}`} />
              {TEMPO_LABELS[t]}
            </span>
          ))}
          <span>
            Độ rộng ô theo thời lượng dự kiến · chấm tròn là motif · đọc từ trái sang phải, hết hàng
            xuống hàng · bấm ô để tới unit
          </span>
        </div>
      </div>
    </Card>
  );
}
