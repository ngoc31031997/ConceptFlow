import { useEffect, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { ProgressTracker } from "../components/ProgressTracker";
import { OutlineReview } from "../components/OutlineReview";
import { ErrorBanner } from "../components/ErrorBanner";
import { AppShell } from "../components/AppShell";
import { useSSE } from "../hooks/useSSE";
import { useProject } from "../hooks/useProject";
import { retryProject, ApiError } from "../api/client";
import glass from "../styles/glass.module.css";

// Steps whose input comes straight from the "Soạn nội dung" screen
// (script/plugin/category) — a failure here is most likely a bad input,
// so the user should go back and fix it rather than blindly retry the
// same input against the same failing step.
const INPUT_RELATED_STEPS = new Set(["parse_script", "classify_scenes"]);

export function RenderPage() {
  const { id } = useParams<{ id: string }>();
  const projectId = id ?? "";
  const navigate = useNavigate();
  const progressState = useSSE(projectId);
  const { project, refetch } = useProject(projectId);
  const [isRetrying, setIsRetrying] = useState(false);
  const [retryError, setRetryError] = useState<string | null>(null);

  // CR-024: Saga đang dừng chờ người, không phải đang chạy. Phân biệt hai thứ
  // này là cả điểm của FR69.5 — nếu không Creator sẽ ngồi đợi một tiến trình
  // đã dừng từ lâu.
  const isAwaitingReview = project?.status === "awaiting_review";

  const isFailed =
    progressState.status === "failed" || Boolean(project?.status.startsWith("failed_at_"));
  const errorMessage = progressState.errorMessage ?? project?.error_message ?? "";
  const failedStep = progressState.currentStep ?? project?.status.replace("failed_at_", "") ?? null;
  const isInputError = isFailed && failedStep !== null && INPUT_RELATED_STEPS.has(failedStep);

  useEffect(() => {
    if (project?.status === "ready_to_publish") {
      navigate(`/projects/${projectId}/result`);
    }
  }, [project?.status, projectId, navigate]);

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
        currentStep={4}
        headerAction={
          <Link to="/" className={glass.ghostBtn} style={{ textDecoration: "none" }}>
            Tạo video mới
          </Link>
        }
        title={isFailed ? "Đã xảy ra lỗi" : "Đang xử lý video"}
        subtitle={
          isFailed
            ? "Một bước trong quá trình xử lý không hoàn tất."
            : "Hệ thống đang tạo hoạt hình, giọng đọc và ghép video cho bạn."
        }
      >
        {/*
          The tracker stays visible after a failure. Replacing it outright hid
          how far the pipeline actually got, which is the first thing you want
          to know when deciding between retrying and going back to the script.
        */}
        {isFailed && (
          <ErrorBanner
            errorMessage={retryError ?? errorMessage}
            onRetry={handleRetry}
            isRetrying={isRetrying}
            onBack={isInputError ? () => navigate("/") : undefined}
          />
        )}
        {isAwaitingReview && project && (
          <OutlineReview project={project} onDecided={refetch} />
        )}

        <ProgressTracker
          progressState={failedStep ? { ...progressState, currentStep: failedStep } : progressState}
          isFailed={isFailed}
        />
      </AppShell>
    </div>
  );
}
