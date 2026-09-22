import type { ProgressState } from "../hooks/useSSE";
import styles from "./ProgressTracker.module.css";
import { RENDER_STEPS, stepLabel, mergedStep } from "../utils/pipelineLabels";
import { Card } from "./ui";

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

/**
 * CR-029: render_scenes reports elapsed time (no reliable total — see
 * ProgressMessage), but synthesize_speech/assemble_video/generate_clips
 * each report a real (index, total) pair now, just under different field
 * names per step. This picks whichever one the current message actually
 * carries and gives it the right Vietnamese unit word for the bar's label.
 */
function unitProgress(
  progressState: ProgressState
): { index: number; total: number; word: string } | null {
  const { sceneIndex, sceneTotal, stageIndex, stageTotal, clipIndex, clipTotal } = progressState;
  if (sceneIndex !== null && sceneTotal !== null && sceneTotal > 0) {
    return { index: sceneIndex, total: sceneTotal, word: "Cảnh" };
  }
  if (stageIndex !== null && stageTotal !== null && stageTotal > 0) {
    return { index: stageIndex, total: stageTotal, word: "Giai đoạn" };
  }
  if (clipIndex !== null && clipTotal !== null && clipTotal > 0) {
    return { index: clipIndex, total: clipTotal, word: "Clip" };
  }
  return null;
}

export function ProgressTracker({ progressState, isFailed = false }: ProgressTrackerProps) {
  const { currentStep, elapsedSeconds, animationIndex } = progressState;
  const unit = unitProgress(progressState);
  const hasSceneProgress = unit !== null;
  const percent = hasSceneProgress ? Math.round((unit.index / unit.total) * 100) : null;
  // A render reports elapsed time rather than a percentage — see ProgressMessage.
  const isRendering = !hasSceneProgress && elapsedSeconds !== null;

  const activeIndex = currentStep ? RENDER_STEPS.indexOf(mergedStep(currentStep) as (typeof RENDER_STEPS)[number]) : -1;

  return (
    <Card>
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
        {unit && (
          <>
            <div className={styles.bar}>
              <div className={styles.barFill} style={{ width: `${percent}%` }} data-testid="progress-tracker-bar" />
            </div>
            <span className={styles.sceneLabel}>
              {unit.word} {unit.index}/{unit.total} ({percent}%)
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
    </Card>
  );
}

function formatElapsed(seconds: number): string {
  const total = Math.floor(seconds);
  const minutes = Math.floor(total / 60);
  return `${minutes}:${String(total % 60).padStart(2, "0")}`;
}
