import type { ProgressState } from "../hooks/useSSE";

interface ProgressTrackerProps {
  progressState: ProgressState;
}

export function ProgressTracker({ progressState }: ProgressTrackerProps) {
  const { currentStep, sceneIndex, sceneTotal } = progressState;
  const hasSceneProgress = sceneIndex !== null && sceneTotal !== null;

  return (
    <div>
      <p data-testid="progress-tracker-step-label">
        {currentStep ? `Đang xử lý: ${currentStep}` : "Đang khởi tạo..."}
      </p>
      {hasSceneProgress && (
        <progress data-testid="progress-tracker-bar" value={sceneIndex ?? 0} max={sceneTotal ?? 1} />
      )}
    </div>
  );
}
