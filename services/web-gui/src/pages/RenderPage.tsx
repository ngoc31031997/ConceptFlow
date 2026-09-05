import { useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { ProgressTracker } from "../components/ProgressTracker";
import { ErrorBanner } from "../components/ErrorBanner";
import { useSSE } from "../hooks/useSSE";
import { useProject } from "../hooks/useProject";
import { retryProject, ApiError } from "../api/client";

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
      <h1>Đang Xử Lý Video</h1>
      {isFailed ? (
        <ErrorBanner
          errorMessage={retryError ?? errorMessage}
          onRetry={handleRetry}
          isRetrying={isRetrying}
        />
      ) : (
        <ProgressTracker progressState={progressState} />
      )}
    </div>
  );
}
