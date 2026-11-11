import { useState } from "react";
import { generateVisualPlanBeat, type VisualPlanBeat, type VisualPlanState } from "../../api/client";
import type { VisualPlanControls } from "../../hooks/useVisualPlan";
import researchStyles from "../research/research.module.css";
import { Button, Card } from "../ui";
import { ConceptAiDialog } from "../visual/ConceptAiDialog";
import { UnitStatusChip } from "../units/UnitStatusChip";
import { PlanUnitCard } from "./PlanUnitCard";
import {
  FILTER_LABELS,
  beatOfLine,
  filterCounts,
  matchesFilter,
  seconds,
  type PlanFilter,
} from "./planModel";
import styles from "./plan.module.css";

interface PlanBeatsCardProps {
  projectId: string;
  state: VisualPlanState;
  plan: VisualPlanControls;
  readOnly: boolean;
}

/** Scrolls to the beat that holds a line, after the screen has settled. */
function goToBeat(key: string) {
  window.setTimeout(
    () => document.getElementById(`plan-beat-${key}`)?.scrollIntoView({ behavior: "smooth", block: "start" }),
    50,
  );
}

interface PlanBeatProps {
  projectId: string;
  beat: VisualPlanBeat;
  filter: PlanFilter;
  plan: VisualPlanControls;
  readOnly: boolean;
}

function PlanBeat({ projectId, beat, filter, plan, readOnly }: PlanBeatProps) {
  const [open, setOpen] = useState(false);
  const disabled = readOnly || plan.busy || plan.running !== null;
  const running =
    plan.running === "all" ? !beat.written : typeof plan.running === "object" && plan.running?.beat === beat.key;
  const approved = beat.units.length > 0 && beat.units.every((u) => u.status === "approved");
  const shown = beat.units.filter((u) => matchesFilter(u, filter));
  const risks = beat.units.filter((u) => u.risk_notes.length > 0).length;
  const t = beat.timing;

  return (
    <div className={styles.beat} id={`plan-beat-${beat.key}`} data-testid={`plan-beat-${beat.key}`}>
      <div className={styles.beatHead}>
        <span className={styles.beatTitle}>
          {beat.key} · {beat.title}
        </span>
        <span className={styles.beatMeta}>
          {t ? (
            <span data-testid={`plan-beat-${beat.key}-timing`}>
              {beat.units.length} unit · dự kiến {seconds(t.planned_s)} · beat-sheet {seconds(t.beat_sheet_s)} · lệch{" "}
              {t.delta_s > 0 ? "+" : ""}
              {seconds(t.delta_s)}
              {risks > 0 ? ` · ⚠ ${risks} rủi ro` : ""}
            </span>
          ) : (
            <span>{beat.lines_count} câu · chưa có kế hoạch</span>
          )}
          {running ? (
            <UnitStatusChip state="running" />
          ) : (
            beat.units.length > 0 && <UnitStatusChip state={approved ? "approved" : "pending"} />
          )}
          {beat.stale && (
            <span className={researchStyles.chip} data-testid={`plan-beat-${beat.key}-stale`}>
              cần lập lại: {beat.stale_sources.join(", ")} đã đổi
            </span>
          )}
          <Button
            variant="run"
            onClick={() => setOpen(true)}
            disabled={disabled}
            data-testid={`plan-beat-${beat.key}-generate`}
          >
            {beat.written ? "Lập lại beat này" : "Lập kế hoạch beat này"}
          </Button>
        </span>
      </div>
      {beat.units.length === 0 && (
        <p className={researchStyles.hint} data-testid={`plan-beat-${beat.key}-empty`}>
          Beat chưa có unit. Bấm “Lập kế hoạch beat này”.
        </p>
      )}
      {beat.units.length > 0 && shown.length === 0 && (
        <p className={researchStyles.hint}>Không có unit nào của beat này khớp bộ lọc.</p>
      )}
      {shown.map((u) => (
        <PlanUnitCard
          key={u.unit_id}
          projectId={projectId}
          beat={beat}
          unit={u}
          next={beat.units[beat.units.findIndex((x) => x.unit_id === u.unit_id) + 1] ?? null}
          plan={plan}
          readOnly={readOnly}
        />
      ))}
      <ConceptAiDialog
        open={open}
        onClose={() => setOpen(false)}
        title={`${beat.written ? "Lập lại" : "Lập kế hoạch"} beat ${beat.key}`}
        impact={
          <p className={researchStyles.hint}>
            AI đọc lời thoại của beat, beat trước và sau, ngôn ngữ hình ảnh đã chốt. Unit bạn đã sửa tay được giữ; mọi
            unit của beat về “Chờ duyệt”.
          </p>
        }
        onSystem={(note) => void plan.generateBeat(beat.key, note)}
        external={{
          run: (note, external) => generateVisualPlanBeat(projectId, beat.key, note, { external }),
          onDone: plan.reload,
        }}
        testId={`plan-beat-${beat.key}-dialog`}
      />
    </div>
  );
}

/**
 * Sections 3 and 4 of step 9: how much of the script the units cover with the
 * filters, then the units grouped by beat.
 */
export function PlanBeatsCard({ projectId, state, plan, readOnly }: PlanBeatsCardProps) {
  const [filter, setFilter] = useState<PlanFilter>("all");
  const counts = filterCounts(state);
  const { lines_total: total, lines_covered: covered, uncovered, duplicated } = state.coverage;
  const pct = total > 0 ? (covered / total) * 100 : 0;

  return (
    <>
      <Card data-testid="plan-coverage">
        <div className={researchStyles.body}>
          <div className={researchStyles.head}>
            <h3>Độ phủ kịch bản</h3>
            <span className={researchStyles.hint} data-testid="plan-coverage-count">
              {covered} / {total} câu
            </span>
          </div>
          <div
            className={styles.cov}
            role="img"
            aria-label={`${covered} trên ${total} câu đã có unit`}
          >
            <i className={styles.covOk} style={{ width: `${pct}%` }} />
            <i className={styles.covMissing} style={{ width: `${100 - pct}%` }} />
          </div>
          {uncovered.length > 0 && (
            <div className={researchStyles.actions}>
              <span data-testid="plan-uncovered">
                {uncovered.length} câu chưa thuộc unit nào: <b>{uncovered.join(", ")}</b>
              </span>
              <span className={researchStyles.grow} />
              <Button variant="view" onClick={() => goToBeat(beatOfLine(uncovered[0]))} data-testid="plan-go-uncovered">
                Tới câu chưa phủ ({uncovered[0]})
              </Button>
            </div>
          )}
          {duplicated.length > 0 && (
            <p className={researchStyles.error} role="status" data-testid="plan-duplicated">
              {duplicated.length} câu thuộc nhiều hơn một unit: {duplicated.join(", ")}
            </p>
          )}
          <div className={styles.filters} role="group" aria-label="Lọc unit">
            {(Object.keys(FILTER_LABELS) as PlanFilter[]).map((f) => (
              <button
                key={f}
                type="button"
                className={styles.filter}
                aria-pressed={filter === f}
                onClick={() => setFilter(f)}
                data-testid={`plan-filter-${f}`}
              >
                {FILTER_LABELS[f]} · {counts[f]}
              </button>
            ))}
          </div>
        </div>
      </Card>
      <Card data-testid="plan-beats">
        <div className={researchStyles.body}>
          <div className={researchStyles.head}>
            <h3>Kế hoạch hình</h3>
            <span className={researchStyles.hint}>Gom theo beat · trái là lời thoại, phải là hình</span>
          </div>
          {state.beats.map((b) => (
            <PlanBeat key={b.key} projectId={projectId} beat={b} filter={filter} plan={plan} readOnly={readOnly} />
          ))}
        </div>
      </Card>
    </>
  );
}
