import { useEffect, useState } from "react";
import { Link, useNavigate, useParams, useSearchParams } from "react-router-dom";
import { ProgressTracker } from "../components/ProgressTracker";
import { ErrorBanner } from "../components/ErrorBanner";
import { AppShell } from "../components/AppShell";
import { useSSE } from "../hooks/useSSE";
import { useProject } from "../hooks/useProject";
import { retryProject, ApiError } from "../api/client";
import { statusToStep, projectPhase, projectPath, PROCESS_STEPS } from "../utils/pipelineLabels";
import { FLOW_TTS } from "../utils/flow";
import glass from "../styles/glass.module.css";

/**
 * Bước 5 của 7 — "Xử lý": phần đắt, chạy sau khi Creator duyệt dàn ý ở bước 4.
 *
 * CR-031 — màn này từng ôm cả lượt chạy thử kịch bản lẫn cổng duyệt dàn ý.
 * Cả hai đã sang bước 4 (ValidatePage), nên ở đây không còn nhánh nào dừng
 * chờ người: mọi thứ từ lúc này tới `ready_to_publish` đều tự chạy, và việc
 * duy nhất của trang là cho thấy nó chạy tới đâu.
 *
 * Lỗi ở đây khác hẳn lỗi ở bước 4. TTS/render/ghép hỏng thường vì hạ tầng —
 * hết quota, worker chết, hết đĩa — nên "Thử lại" là việc đúng, và không có
 * nút quay về sửa script: script này đã qua lượt chạy thử và đã được duyệt.
 */
export function RenderPage() {
  const { id } = useParams<{ id: string }>();
  const projectId = id ?? "";
  const navigate = useNavigate();
  // ?view=1&step=N: mở chỉ để XEM lại bước 8-11 của dự án đã chạy xong.
  const [search] = useSearchParams();
  const viewOnly = search.get("view") === "1";
  const viewStep = Number(search.get("step")) || FLOW_TTS;
  const progressState = useSSE(projectId);
  const { project } = useProject(projectId);
  const [isRetrying, setIsRetrying] = useState(false);
  const [retryError, setRetryError] = useState<string | null>(null);

  // Một project chưa qua bước 4 (hoặc đã xong hẳn) không thuộc màn này. Bookmark
  // cũ, nút back sau khi duyệt, hay một saga bị đẩy lùi vì Creator từ chối dàn
  // ý — cả ba đều dẫn tới đây với một trạng thái mà trang này không có gì để
  // hiển thị ngoài bốn ô "pending" bất động.
  const phase = project ? projectPhase(project.status) : null;
  useEffect(() => {
    if (project && phase !== "process" && !viewOnly) {
      navigate(projectPath(projectId, project.status), { replace: true });
    }
  }, [project, phase, viewOnly, projectId, navigate]);
  const reviewingPast = viewOnly && phase !== "process";

  const isFailed =
    progressState.status === "failed" || Boolean(project?.status.startsWith("failed_at_"));
  const errorMessage = progressState.errorMessage ?? project?.error_message ?? "";
  // Creator stopped it on purpose: not an error to explain, and the strip above
  // carries the one action ("Chạy tiếp"). Only a real failure gets the banner.
  const isCancelled = project?.run_state === "cancelled";

  // Bug report: navigating away mid-render and back showed "Đang khởi
  // tạo..." with no sign of progress, or of whether it was even still
  // running. useSSE only knows what arrived on THIS tab's SSE connection —
  // reopening the page starts that at null, and the next progress.fanout
  // message can be minutes away. The render itself never paused (nothing
  // client-side can pause a saga already running server-side); only the
  // tracker looked stuck. Seed it from the project's own persisted status
  // until a live message replaces it with real scene/elapsed detail.
  const displayStep =
    (isFailed
      ? (progressState.currentStep ?? project?.status.replace("failed_at_", ""))
      : progressState.currentStep) ?? (project ? statusToStep(project.status) : null);
  const displayProgressState =
    displayStep && displayStep !== progressState.currentStep
      ? {
          ...progressState,
          currentStep: displayStep,
          status: isFailed ? progressState.status : ("in_progress" as const),
        }
      : progressState;

  // Bước 8-11 theo bước saga đang chạy: TTS, render, merge (kèm QC), cắt short.
  const activeFlowStep =
    { synthesize_speech: 8, render_scenes: 9, assemble_video: 10, qc_video: 10, generate_clips: 11 }[
      displayStep ?? ""
    ] ?? FLOW_TTS;

  async function handleRetry() {
    setIsRetrying(true);
    setRetryError(null);
    try {
      await retryProject(projectId);
    } catch (err) {
      setRetryError(err instanceof ApiError ? err.message : String(err));
    } finally {
      setIsRetrying(false);
    }
  }

  return (
    <div data-testid="render-page">
      <AppShell
        currentStep={viewOnly ? viewStep : activeFlowStep}
        headerAction={
          <Link to="/" className={glass.ghostBtn} style={{ textDecoration: "none" }}>
            Tạo video mới
          </Link>
        }
        title={isCancelled ? "Đã huỷ" : isFailed ? "Đã xảy ra lỗi" : "Đang xử lý video"}
        subtitle={
          isCancelled
            ? "Bạn đã dừng bước này. Chạy tiếp từ dải trạng thái phía trên."
            : isFailed
            ? "Một bước trong quá trình sản xuất không hoàn tất."
            : "Dàn ý đã duyệt — hệ thống đang tạo giọng đọc, render hoạt hình và ghép video."
        }
      >
        {/*
          The tracker stays visible after a failure. Replacing it outright
          hid how far the pipeline actually got, which is the first thing
          you want to know when deciding whether to retry.
        */}
        {isFailed && !isCancelled && (
          <ErrorBanner
            errorMessage={retryError ?? errorMessage}
            onRetry={handleRetry}
            isRetrying={isRetrying}
            projectId={projectId}
            step={displayStep ?? undefined}
          />
        )}
        <ProgressTracker
          progressState={displayProgressState}
          steps={PROCESS_STEPS}
          isFailed={isFailed}
          allDone={reviewingPast}
        />
      </AppShell>
    </div>
  );
}
