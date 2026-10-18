import { Link } from "react-router-dom";
import { AppShell } from "../components/AppShell";
import { AUTO_STEPS, FLOW_VIEW, FLOW_VIEW_PHASES, previewRoute, type FlowViewStep } from "../utils/flow";
import styles from "./FlowMapPage.module.css";

const TONE_CLASS = {
  info: styles.toneInfo,
  accent: styles.toneAccent,
  run: styles.toneRun,
  accent2: styles.toneAccent2,
} as const;

function StepBody({ view }: { view: FlowViewStep }) {
  const coming = view.server.length === 0;
  const auto = !coming && view.server.every((step) => AUTO_STEPS.has(step));
  return (
    <>
      <span className={styles.n}>{view.n}</span>
      <span>
        <b>{view.title ?? view.label}</b>
        {coming && <span className={styles.coming}>sắp có</span>}
        <span className={styles.note}>{view.note}</span>
        {view.gate && <span className={styles.gate}>◆ bạn duyệt</span>}
        {auto && <span className={styles.auto}>tự chạy</span>}
      </span>
    </>
  );
}

/**
 * The whole 21-step flow on one page (ADR-0042): a card per phase, each step
 * with its number, name, what it holds, and whether the Creator approves it
 * (◆) or a machine runs it. A step the app has opens its preview; one not
 * built yet reads "sắp có".
 */
export function FlowMapPage() {
  return (
    <AppShell
      title="Bản đồ luồng"
      subtitle={`${FLOW_VIEW.length} bước trong ${FLOW_VIEW_PHASES.length} chặng. ◆ là chỗ bạn duyệt. Bấm một bước để mở màn của nó; bước “sắp có” chưa có trong app.`}
      wide
    >
      <div className={styles.map} data-testid="flow-map">
        {FLOW_VIEW_PHASES.map((phase, index) => (
          <section key={phase.name} className={styles.card} data-testid="flow-map-phase">
            <h3 className={styles.phaseName}>
              <i className={`${styles.swatch} ${TONE_CLASS[phase.tone]}`} aria-hidden="true" />
              {phase.name}
            </h3>
            {FLOW_VIEW.filter((v) => v.phase === index).map((view) =>
              view.server.length > 0 ? (
                <Link
                  key={view.n}
                  to={previewRoute(view.server[0], "")}
                  className={styles.step}
                  data-testid={`flow-map-step-${view.n}`}
                >
                  <StepBody view={view} />
                </Link>
              ) : (
                <div key={view.n} className={`${styles.step} ${styles.stepComing}`} data-testid={`flow-map-step-${view.n}`}>
                  <StepBody view={view} />
                </div>
              ),
            )}
          </section>
        ))}
      </div>
    </AppShell>
  );
}
