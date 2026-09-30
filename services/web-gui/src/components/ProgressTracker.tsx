import type { ProgressState } from "../hooks/useSSE";
import styles from "./ProgressTracker.module.css";
import { stepLabel } from "../utils/pipelineLabels";
import { Card } from "./ui";

interface ProgressTrackerProps {
  progressState: ProgressState;
  /**
   * Các bước màn hình NÀY chịu trách nhiệm, theo thứ tự. Trước đây
   * tracker luôn vẽ cả saga từ một hằng số dùng chung; từ khi bước 4
   * (Validate) và bước 5 (Xử lý) là hai màn riêng, mỗi màn chỉ được vẽ phần
   * của mình — nếu không thì cả hai cùng hiện một danh sách giống hệt và
   * Creator không biết mình đang ở đâu.
   */
  steps: readonly string[];
  /**
   * Number shown in each row's dot, matching the sidebar — or a sub-step number
   * ("7.1") for work inside one flow step. Rows not listed fall back to their position.
   */
  stepNumbers?: Record<string, number | string>;
  /** Dims the tracker and drops the live wording once the saga has failed. */
  isFailed?: boolean;
  /** Xem lại một bước đã chạy xong: tất cả các ô là "xong", không có tiến độ sống. */
  allDone?: boolean;
  /** Set when these steps haven't started: replaces the live wording, which describes another step. */
  waitingLabel?: string;
}

function CheckIcon() {
  return (
    <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3.5" strokeLinecap="round" strokeLinejoin="round">
      <path d="M5 13l4 4L19 7" />
    </svg>
  );
}

/**
 * A Manim render_scenes reports elapsed time only (no reliable total);
 * a Remotion one carries render_percent, assemble_video merge_percent.
 * synthesize_speech/generate_clips
 * each report a real (index, total) pair now, just under different field
 * names per step. This picks whichever one the current message actually
 * carries and gives it the right Vietnamese unit word for the bar's label.
 */
function unitProgress(progressState: ProgressState): { percent: number; label: string } | null {
  const { sceneIndex, sceneTotal, clipIndex, clipTotal, renderPercent, mergePercent, elapsedSeconds } =
    progressState;
  const counted = (index: number, total: number, word: string) => ({
    percent: Math.round((index / total) * 100),
    label: `${word} ${index}/${total}`,
  });
  // Remotion only: renderMedia's own frame progress. Manim sends none and keeps the elapsed text.
  if (renderPercent !== null) {
    const elapsed = elapsedSeconds !== null ? ` · ${formatElapsed(elapsedSeconds)}` : "";
    return { percent: renderPercent, label: `Đã render ${renderPercent}%${elapsed}` };
  }
  if (sceneIndex !== null && sceneTotal !== null && sceneTotal > 0) return counted(sceneIndex, sceneTotal, "Cảnh");
  if (mergePercent !== null) return { percent: mergePercent, label: `Đã ghép ${mergePercent}%` };
  if (clipIndex !== null && clipTotal !== null && clipTotal > 0) return counted(clipIndex, clipTotal, "Clip");
  return null;
}

export function ProgressTracker({
  progressState,
  steps,
  stepNumbers,
  isFailed = false,
  allDone = false,
  waitingLabel,
}: ProgressTrackerProps) {
  const { currentStep, elapsedSeconds, animationIndex } = progressState;
  const unit = allDone || waitingLabel ? null : unitProgress(progressState);
  // A Manim render (or Remotion still bundling) reports elapsed time, not a percentage.
  const isRendering = !allDone && !waitingLabel && unit === null && elapsedSeconds !== null;

  // Một bước không thuộc màn này (saga đã chạy qua, hoặc chưa tới) cho -1 —
  // và -1 vẽ ra danh sách toàn "pending", đúng nghĩa "màn này chưa tới lượt".
  const activeIndex = currentStep ? steps.indexOf(currentStep) : -1;

  return (
    <Card>
      <div className={styles.wrap}>
        <p className={styles.stepLabel} data-testid="progress-tracker-step-label">
          {allDone
            ? "Đã chạy xong"
            : waitingLabel
            ? waitingLabel
            : currentStep
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
              <div className={styles.barFill} style={{ width: `${unit.percent}%` }} data-testid="progress-tracker-bar" />
            </div>
            <span className={styles.sceneLabel}>{unit.label}</span>
          </>
        )}

        {/*
          The whole pipeline, not just the current step. A render can sit on
          one step for many minutes, and without the surrounding steps there is
          no way to tell a slow step from a stuck one — or, after a failure, how
          far the project actually got.
        */}
        <ol className={styles.stepList} data-testid="progress-tracker-steps">
          {steps.map((step, index) => {
            const isDone = allDone || (activeIndex >= 0 && index < activeIndex);
            const isActive = !allDone && index === activeIndex;
            const state = isDone ? "done" : isActive ? (isFailed ? "failed" : "active") : "pending";
            return (
              <li key={step} className={styles.stepListItem} data-state={state}>
                <span className={styles.stepDot}>{isDone ? <CheckIcon /> : (stepNumbers?.[step] ?? index + 1)}</span>
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
