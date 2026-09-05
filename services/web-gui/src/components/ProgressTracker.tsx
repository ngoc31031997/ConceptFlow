import type { ProgressState } from "../hooks/useSSE";
import glass from "../styles/glass.module.css";
import styles from "./ProgressTracker.module.css";

interface ProgressTrackerProps {
  progressState: ProgressState;
}

export function ProgressTracker({ progressState }: ProgressTrackerProps) {
  const { currentStep, sceneIndex, sceneTotal } = progressState;
  const hasSceneProgress = sceneIndex !== null && sceneTotal !== null && sceneTotal > 0;
  const percent = hasSceneProgress ? Math.round(((sceneIndex ?? 0) / (sceneTotal ?? 1)) * 100) : null;

  return (
    <div className={glass.card}>
      <div className={styles.wrap}>
        <p className={styles.stepLabel} data-testid="progress-tracker-step-label">
          {currentStep ? `Đang xử lý: ${currentStep}` : "Đang khởi tạo..."}
        </p>
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
