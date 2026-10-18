import { Link } from "react-router-dom";
import { AppShell } from "../components/AppShell";
import { useRecentProject } from "../hooks/useRecentProject";
import {
  AUTO_STEPS,
  FLOW_VIEW,
  FLOW_VIEW_PHASES,
  comingRoute,
  isBuilt,
  previewRoute,
  screenRoute,
  type FlowViewStep,
} from "../utils/flow";
import styles from "./FlowMapPage.module.css";

const TONE_CLASS = {
  info: styles.toneInfo,
  accent: styles.toneAccent,
  run: styles.toneRun,
  accent2: styles.toneAccent2,
} as const;

function StepBody({ view }: { view: FlowViewStep }) {
  const coming = !isBuilt(view);
  const auto = !coming && view.server.length > 0 && view.server.every((step) => AUTO_STEPS.has(step));
  return (
    <>
      <span className={styles.n}>{view.n}</span>
      <span>
        <b>{view.title ?? view.label}</b>
        {coming && <span className={styles.coming}>sắp có</span>}
        <span className={styles.note}>{view.note}</span>
        {view.gate && <span className={styles.gate}>◆ bạn duyệt</span>}
        {view.optional && <span className={styles.auto}>tuỳ chọn</span>}
        {auto && <span className={styles.auto}>tự chạy</span>}
      </span>
    </>
  );
}

/**
 * The whole 21-step flow on one page (ADR-0042): a card per phase, each step
 * with its number, name, what it holds, and whether the Creator approves it
 * (◆) or a machine runs it. A step the app has opens its preview; a step with
 * a screen of its own opens that screen for the video last worked on; one not
 * built yet reads "sắp có" and opens a page that says it is not developed.
 */
export function FlowMapPage() {
  const recent = useRecentProject(null);
  const stepClass = styles.step;
  return (
    <AppShell
      title="Bản đồ luồng"
      subtitle={`${FLOW_VIEW.length} bước trong ${FLOW_VIEW_PHASES.length} chặng. ◆ là chỗ bạn duyệt. Bấm một bước để mở màn của nó; bước “sắp có” chưa phát triển nên chỉ mở một trang thông báo.`}
      wide
    >
      <div className={styles.map} data-testid="flow-map">
        {FLOW_VIEW_PHASES.map((phase, index) => (
          <section key={phase.name} className={styles.card} data-testid="flow-map-phase">
            <h3 className={styles.phaseName}>
              <i className={`${styles.swatch} ${TONE_CLASS[phase.tone]}`} aria-hidden="true" />
              {phase.name}
            </h3>
            {FLOW_VIEW.filter((v) => v.phase === index).map((view) => {
              const testId = `flow-map-step-${view.n}`;
              if (view.server.length > 0) {
                return (
                  <Link key={view.n} to={previewRoute(view.server[0], "")} className={stepClass} data-testid={testId}>
                    <StepBody view={view} />
                  </Link>
                );
              }
              if (view.screen) {
                // Its screen belongs to a video, so it opens only when there is one.
                return recent ? (
                  <Link key={view.n} to={screenRoute(view, recent.project_id)} className={stepClass} data-testid={testId}>
                    <StepBody view={view} />
                  </Link>
                ) : (
                  <div key={view.n} className={stepClass} data-testid={testId}>
                    <StepBody view={view} />
                  </div>
                );
              }
              return (
                <Link key={view.n} to={comingRoute(view.n, "")} className={`${stepClass} ${styles.stepComing}`} data-testid={testId}>
                  <StepBody view={view} />
                </Link>
              );
            })}
          </section>
        ))}
      </div>
    </AppShell>
  );
}
