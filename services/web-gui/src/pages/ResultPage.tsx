import { useState } from "react";
import { useParams } from "react-router-dom";
import { VideoPlayer } from "../components/VideoPlayer";
import { YoutubeConnectButton } from "../components/YoutubeConnectButton";
import { PublishForm } from "../components/PublishForm";
import { AppShell } from "../components/AppShell";
import { useProject } from "../hooks/useProject";
import { startPublishSaga, getProjectVideoUrl, ApiError } from "../api/client";
import type { PublishMetadata } from "../types";
import glass from "../styles/glass.module.css";

export function ResultPage() {
  const { id } = useParams<{ id: string }>();
  const projectId = id ?? "";
  const { project, refetch } = useProject(projectId);
  const [isPublishing, setIsPublishing] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handlePublish(metadata: PublishMetadata) {
    setIsPublishing(true);
    setError(null);
    try {
      await startPublishSaga(projectId, metadata);
      await refetch();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : String(err));
    } finally {
      setIsPublishing(false);
    }
  }

  if (!project) return null;

  return (
    <div data-testid="result-page">
      <AppShell
        currentStep={3}
        title="Xem kết quả & đăng video"
        subtitle="Xem trước video, kết nối YouTube và điền thông tin để xuất bản."
      >
        {project.video_path && <VideoPlayer videoSrc={getProjectVideoUrl(projectId)} />}

        {project.youtube_video_url ? (
          <div className={glass.card} style={{ textAlign: "center", padding: "44px 32px" }}>
            <p style={{ margin: "0 0 12px", fontSize: 19, fontWeight: 700 }}>Đã đăng thành công!</p>
            <a href={project.youtube_video_url}>{project.youtube_video_url}</a>
          </div>
        ) : (
          <>
            <YoutubeConnectButton projectId={projectId} />
            {error && (
              <p role="alert" className={glass.helperText}>
                {error}
              </p>
            )}
            <PublishForm onSubmit={handlePublish} isSubmitting={isPublishing} />
          </>
        )}
      </AppShell>
    </div>
  );
}
