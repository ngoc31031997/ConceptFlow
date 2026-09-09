import { useRef, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { VideoPlayer } from "../components/VideoPlayer";
import { YoutubeChannels } from "../components/YoutubeChannels";
import { ThumbnailUpload } from "../components/ThumbnailUpload";
import { PublishForm } from "../components/PublishForm";
import { AppShell } from "../components/AppShell";
import { useProject } from "../hooks/useProject";
import {
  startPublishSaga,
  retryProject,
  deleteProject,
  getProjectVideoUrl,
  ApiError,
} from "../api/client";
import type { PublishMetadata } from "../types";
import glass from "../styles/glass.module.css";
import styles from "./ResultPage.module.css";

export function ResultPage() {
  const { id } = useParams<{ id: string }>();
  const projectId = id ?? "";
  const navigate = useNavigate();
  const { project, refetch } = useProject(projectId);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [thumbnailPath, setThumbnailPath] = useState<string | null>(null);
  const [channelId, setChannelId] = useState<string | null>(null);
  /*
    State lands a render behind the click, so two fast clicks can both read
    isSubmitting === false and fire two POSTs. The ref flips synchronously.
  */
  const inFlightRef = useRef(false);

  async function handlePublish(metadata: PublishMetadata) {
    if (inFlightRef.current) return;
    inFlightRef.current = true;
    setIsSubmitting(true);
    setError(null);
    try {
      await startPublishSaga(projectId, {
        ...metadata,
        thumbnail_path: thumbnailPath ?? undefined,
        // Left out when no channel is connected yet, so the Publisher
        // reports "not authenticated" rather than "channel '' not found".
        channel_id: channelId ?? undefined,
      });
      /*
        The POST only *starts* the saga; the project is now "publishing" and
        useProject's poll drives the rest of the UI. isSubmitting stays true
        until that refetch lands so the button never flickers back to enabled
        in between.
      */
      await refetch();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : String(err));
    } finally {
      inFlightRef.current = false;
      setIsSubmitting(false);
    }
  }

  async function handleRetryPublish() {
    if (inFlightRef.current) return;
    inFlightRef.current = true;
    setIsSubmitting(true);
    setError(null);
    try {
      await retryProject(projectId);
      await refetch();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : String(err));
    } finally {
      inFlightRef.current = false;
      setIsSubmitting(false);
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

  const isPublished = project.status === "published" || Boolean(project.youtube_video_url);
  const isPublishing = isSubmitting || project.status === "publishing";
  const hasPublishFailed = project.status === "failed_at_publish_video";

  const errorBanner = error && (
    <p role="alert" className={glass.helperText}>
      {error}
    </p>
  );

  return (
    <div data-testid="result-page">
      <AppShell
        currentStep={5}
        wide
        title="Xem kết quả & đăng video"
        subtitle="Xem trước video, kết nối YouTube và điền thông tin để xuất bản."
        headerAction={
          <Link to="/" className={glass.ghostBtn} style={{ textDecoration: "none" }}>
            Tạo video mới
          </Link>
        }
      >
        {isPublished ? (
          <div className={glass.card} style={{ textAlign: "center", padding: "44px 32px" }}>
            <p style={{ margin: "0 0 12px", fontSize: 19, fontWeight: 700 }}>Đã đăng thành công!</p>
            <a href={project.youtube_video_url ?? undefined}>{project.youtube_video_url}</a>
            {project.video_path && (
              <div style={{ marginTop: 24 }}>
                <VideoPlayer videoSrc={getProjectVideoUrl(projectId)} />
              </div>
            )}
          </div>
        ) : (
          /*
            Preview on the left, everything the upload needs on the right. The
            publish button used to sit at the bottom of a single stacked column
            — below the player, the connect button and the thumbnail uploader —
            so the action the page exists for was the last thing reachable.
          */
          <div className={styles.layout}>
            <div className={styles.preview}>
              {project.video_path && <VideoPlayer videoSrc={getProjectVideoUrl(projectId)} />}
            </div>

            <div className={styles.publishColumn}>
              {isPublishing ? (
                <div
                  className={`${glass.card} ${styles.publishStatus}`}
                  data-testid="result-publishing-status"
                  role="status"
                  aria-live="polite"
                >
                  <span className={styles.spinner} aria-hidden="true" />
                  <div>
                    <p className={styles.publishStatusText}>Đang tải video lên YouTube...</p>
                    <p className={styles.publishStatusHint}>
                      Quá trình này có thể mất vài phút. Bạn không cần bấm lại — trang sẽ tự cập nhật
                      khi đăng xong.
                    </p>
                  </div>
                </div>
              ) : (
                <>
                  {hasPublishFailed && (
                    <div
                      className={`${glass.card} ${styles.failedCard}`}
                      data-testid="result-publish-failed"
                      role="alert"
                    >
                      <p className={styles.publishStatusText}>Đăng lên YouTube thất bại</p>
                      <p className={styles.publishStatusHint}>
                        {project.error_message ?? "Không rõ nguyên nhân."}
                      </p>
                      <div className={glass.ctaRow} style={{ marginTop: 14 }}>
                        <button
                          type="button"
                          data-testid="result-retry-publish-button"
                          className={glass.btnPrimary}
                          disabled={isSubmitting}
                          onClick={handleRetryPublish}
                        >
                          Thử đăng lại
                        </button>
                      </div>
                    </div>
                  )}
                  <YoutubeChannels projectId={projectId} onSelectedChannelChange={setChannelId} />
                  <ThumbnailUpload
                    projectId={projectId}
                    onThumbnailPathChange={setThumbnailPath}
                    contentLanguage={project.voice_language}
                  />
                  {errorBanner}
                  {/*
                    Once the publish step has failed the saga is resumed with
                    POST /retry above — a fresh POST /v1/sagas/publish would
                    only 409, since it requires status ready_to_publish.
                  */}
                  {!hasPublishFailed && (
                    <PublishForm
                      projectId={projectId}
                      onSubmit={handlePublish}
                      isSubmitting={isSubmitting}
                    />
                  )}
                </>
              )}
              {isPublishing && errorBanner}
            </div>
          </div>
        )}

        {/*
          A destructive, rarely-used action does not belong at the top of the
          page, above the video it deletes. It sits after the work instead.
        */}
        <div className={styles.dangerZone}>
          <span className={glass.cardHint}>Xoá vĩnh viễn video này và toàn bộ dữ liệu liên quan.</span>
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
      </AppShell>
    </div>
  );
}
