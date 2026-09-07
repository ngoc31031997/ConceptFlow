import { useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { VideoPlayer } from "../components/VideoPlayer";
import { YoutubeConnectButton } from "../components/YoutubeConnectButton";
import { ThumbnailUpload } from "../components/ThumbnailUpload";
import { PublishForm } from "../components/PublishForm";
import { AppShell } from "../components/AppShell";
import { useProject } from "../hooks/useProject";
import { startPublishSaga, deleteProject, getProjectVideoUrl, ApiError } from "../api/client";
import type { PublishMetadata } from "../types";
import glass from "../styles/glass.module.css";

export function ResultPage() {
  const { id } = useParams<{ id: string }>();
  const projectId = id ?? "";
  const navigate = useNavigate();
  const { project, refetch } = useProject(projectId);
  const [isPublishing, setIsPublishing] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [thumbnailPath, setThumbnailPath] = useState<string | null>(null);

  async function handlePublish(metadata: PublishMetadata) {
    setIsPublishing(true);
    setError(null);
    try {
      await startPublishSaga(projectId, {
        ...metadata,
        thumbnail_path: thumbnailPath ?? undefined,
      });
      await refetch();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : String(err));
    } finally {
      setIsPublishing(false);
    }
  }

  async function handleDelete() {
    if (!window.confirm("Xoá video này và toàn bộ dữ liệu liên quan? Hành động này không thể hoàn tác.")) {
      return;
    }
    setIsDeleting(true);
    setError(null);
    try {
      await deleteProject(projectId);
      navigate("/videos");
    } catch (err) {
      setError(err instanceof ApiError ? err.message : String(err));
      setIsDeleting(false);
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
        <div style={{ display: "flex", justifyContent: "flex-end" }}>
          <button
            type="button"
            data-testid="result-delete-button"
            className={glass.dangerGhostBtn}
            disabled={isDeleting}
            onClick={handleDelete}
          >
            {isDeleting ? "Đang xoá..." : "Xoá video"}
          </button>
        </div>

        {project.video_path && <VideoPlayer videoSrc={getProjectVideoUrl(projectId)} />}

        {project.youtube_video_url ? (
          <div className={glass.card} style={{ textAlign: "center", padding: "44px 32px" }}>
            <p style={{ margin: "0 0 12px", fontSize: 19, fontWeight: 700 }}>Đã đăng thành công!</p>
            <a href={project.youtube_video_url}>{project.youtube_video_url}</a>
          </div>
        ) : (
          <>
            <YoutubeConnectButton projectId={projectId} />
            <ThumbnailUpload
              projectId={projectId}
              onThumbnailPathChange={setThumbnailPath}
              contentLanguage={project.voice_language}
            />
            {error && (
              <p role="alert" className={glass.helperText}>
                {error}
              </p>
            )}
            <PublishForm projectId={projectId} onSubmit={handlePublish} isSubmitting={isPublishing} />
          </>
        )}
      </AppShell>
    </div>
  );
}
