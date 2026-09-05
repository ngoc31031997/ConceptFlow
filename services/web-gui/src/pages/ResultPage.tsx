import { useState } from "react";
import { useParams } from "react-router-dom";
import { VideoPlayer } from "../components/VideoPlayer";
import { YoutubeConnectButton } from "../components/YoutubeConnectButton";
import { PublishForm } from "../components/PublishForm";
import { useProject } from "../hooks/useProject";
import { startPublishSaga, ApiError } from "../api/client";
import type { PublishMetadata } from "../types";

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
      <h1>Kết Quả</h1>
      {project.video_path && <VideoPlayer videoSrc={project.video_path} />}
      {project.youtube_video_url ? (
        <p>
          Đã đăng: <a href={project.youtube_video_url}>{project.youtube_video_url}</a>
        </p>
      ) : (
        <>
          <YoutubeConnectButton />
          {error && <p role="alert">{error}</p>}
          <PublishForm onSubmit={handlePublish} isSubmitting={isPublishing} />
        </>
      )}
    </div>
  );
}
