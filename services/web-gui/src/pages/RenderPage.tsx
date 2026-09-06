import { useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { ProgressTracker } from "../components/ProgressTracker";
import { ErrorBanner } from "../components/ErrorBanner";
import { AppShell } from "../components/AppShell";
import { useSSE } from "../hooks/useSSE";
import { useProject } from "../hooks/useProject";
import { retryProject, ApiError } from "../api/client";

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
  const { project } = useProject(projectId);
  const [isRetrying, setIsRetrying] = useState(false);
  const [retryError, setRetryError] = useState<string | null>(null);

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
        currentStep={2}
        title={isFailed ? "Đã xảy ra lỗi" : "Đang xử lý video"}
        subtitle={
          isFailed
            ? "Một bước trong quá trình xử lý không hoàn tất."
            : "Hệ thống đang tạo hoạt hình, giọng đọc và ghép video cho bạn."
        }
      >
        {isFailed ? (
          <ErrorBanner
            errorMessage={retryError ?? errorMessage}
            onRetry={handleRetry}
            isRetrying={isRetrying}
            onBack={isInputError ? () => navigate("/") : undefined}
          />
        ) : (
          <ProgressTracker progressState={progressState} />
        )}
      </AppShell>
    </div>
  );
}
