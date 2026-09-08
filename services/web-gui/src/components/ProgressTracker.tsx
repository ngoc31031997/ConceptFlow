import type { ProgressState } from "../hooks/useSSE";
import glass from "../styles/glass.module.css";
import styles from "./ProgressTracker.module.css";
import { RENDER_STEPS, stepLabel } from "../utils/pipelineLabels";

interface ProgressTrackerProps {
  progressState: ProgressState;
  /** Dims the tracker and drops the live wording once the saga has failed. */
  isFailed?: boolean;
}

function CheckIcon() {
  return (
    <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3.5" strokeLinecap="round" strokeLinejoin="round">
      <path d="M5 13l4 4L19 7" />
    </svg>
  );
}

export function ProgressTracker({ progressState, isFailed = false }: ProgressTrackerProps) {
  const { currentStep, sceneIndex, sceneTotal, elapsedSeconds, animationIndex } = progressState;
  const hasSceneProgress = sceneIndex !== null && sceneTotal !== null && sceneTotal > 0;
  const percent = hasSceneProgress ? Math.round(((sceneIndex ?? 0) / (sceneTotal ?? 1)) * 100) : null;
  // A render reports elapsed time rather than a percentage — see ProgressMessage.
  const isRendering = !hasSceneProgress && elapsedSeconds !== null;

  const activeIndex = currentStep ? RENDER_STEPS.indexOf(currentStep as (typeof RENDER_STEPS)[number]) : -1;

  return (
    <div className={glass.card}>
      <div className={styles.wrap}>
        <p className={styles.stepLabel} data-testid="progress-tracker-step-label">
          {currentStep
            ? isFailed
              ? `Dừng ở bước: ${stepLabel(currentStep)}`
              : `Đang xử lý: ${stepLabel(currentStep)}`
            : "Đang khởi tạo..."}
        </p>
        {isRendering && !isFailed && (
          <span className={styles.sceneLabel} data-testid="progress-tracker-elapsed">
            Đã render {formatElapsed(elapsedSeconds ?? 0)}
            {animationIndex !== null ? ` — animation ${animationIndex}` : ""}
          </span>
        )}
        {hasSceneProgress && (
          <>
            <div className={styles.bar}>
              <div className={styles.barFill} style={{ width: `${percent}%` }} data-testid="progress-tracker-bar" />
            </div>
            <span className={styles.sceneLabel}>
              Cảnh {sceneIndex}/{sceneTotal}
            </span>
          </>
        )}

        {/*
          The whole pipeline, not just the current step. A render can sit on
          one step for many minutes, and without the surrounding steps there is
          no way to tell a slow step from a stuck one — or, after a failure, how
          far the project actually got.
        */}
        <ol className={styles.stepList} data-testid="progress-tracker-steps">
          {RENDER_STEPS.map((step, index) => {
            const isDone = activeIndex >= 0 && index < activeIndex;
            const isActive = index === activeIndex;
            const state = isDone ? "done" : isActive ? (isFailed ? "failed" : "active") : "pending";
            return (
              <li key={step} className={styles.stepListItem} data-state={state}>
                <span className={styles.stepDot}>{isDone ? <CheckIcon /> : index + 1}</span>
                {stepLabel(step)}
              </li>
            );
          })}
        </ol>
      </div>
    </div>
  );
}

function formatElapsed(seconds: number): string {
  const total = Math.floor(seconds);
  const minutes = Math.floor(total / 60);
  return `${minutes}:${String(total % 60).padStart(2, "0")}`;
}
