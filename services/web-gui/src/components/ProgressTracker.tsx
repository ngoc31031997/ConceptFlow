import type { ProgressState } from "../hooks/useSSE";
import glass from "../styles/glass.module.css";
import styles from "./ProgressTracker.module.css";

interface ProgressTrackerProps {
  progressState: ProgressState;
}

export function ProgressTracker({ progressState }: ProgressTrackerProps) {
  const { currentStep, sceneIndex, sceneTotal, elapsedSeconds, animationIndex } = progressState;
  const hasSceneProgress = sceneIndex !== null && sceneTotal !== null && sceneTotal > 0;
  const percent = hasSceneProgress ? Math.round(((sceneIndex ?? 0) / (sceneTotal ?? 1)) * 100) : null;
  // A render reports elapsed time rather than a percentage — see ProgressMessage.
  const isRendering = !hasSceneProgress && elapsedSeconds !== null;

  return (
    <div className={glass.card}>
      <div className={styles.wrap}>
        <p className={styles.stepLabel} data-testid="progress-tracker-step-label">
          {currentStep ? `Đang xử lý: ${currentStep}` : "Đang khởi tạo..."}
        </p>
        {isRendering && (
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
      </div>
    </div>
  );
}

function formatElapsed(seconds: number): string {
  const total = Math.floor(seconds);
  const minutes = Math.floor(total / 60);
  return `${minutes}:${String(total % 60).padStart(2, "0")}`;
}
