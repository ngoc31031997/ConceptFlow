import { useRef, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { VideoPlayer } from "../components/VideoPlayer";
import { YoutubeChannels } from "../components/YoutubeChannels";
import { ThumbnailUpload } from "../components/ThumbnailUpload";
import { PublishForm } from "../components/PublishForm";
import { AppShell } from "../components/AppShell";
import { useProject } from "../hooks/useProject";
import { QCReportPanel } from "../components/QCReportPanel";
import { ClipsPanel } from "../components/ClipsPanel";
import {
  startPublishSaga,
  retryProject,
  getProjectVideoUrl,
  ApiError,
  ERROR_CODE_QC_BLOCKED,
} from "../api/client";
import type { PublishMetadata } from "../types";
import glass from "../styles/glass.module.css";
import styles from "./PublishPage.module.css";

/**
 * Bước 6 — "Đăng" (bug report, 2026-09-12): tách khỏi ResultPage (Bước 5,
 * "Kết quả"), để trang này CHỈ làm một việc — kết nối YouTube, điền
 * tiêu đề/mô tả, xem báo cáo QC, và đăng. Mọi thứ khác (render lại, tạo bản
 * Shorts, xem input, xoá) ở lại ResultPage.
 */
export function PublishPage() {
  const { id } = useParams<{ id: string }>();
  const projectId = id ?? "";
  const { project, refetch } = useProject(projectId);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [thumbnailPath, setThumbnailPath] = useState<string | null>(null);
  const [channelId, setChannelId] = useState<string | null>(null);
  // CR-007 follow-up: collapsed by default only for "short" — a Creator who
  // picked "chỉ video ngắn" is here for the clip, not the YouTube form. null
  // means "not touched yet" so the default can depend on `project`, which is
  // not loaded yet on the render that mounts this state.
  const [youtubePublishOverride, setYoutubePublishOverride] = useState<boolean | null>(null);
  /*
    State lands a render behind the click, so two fast clicks can both read
    isSubmitting === false and fire two POSTs. The ref flips synchronously.
  */
  const inFlightRef = useRef(false);
  const videoRef = useRef<HTMLVideoElement>(null);

  /** FR61.2 — bấm mốc thời gian trong báo cáo QC thì tua player tới đúng giây. */
  function handleSeek(seconds: number) {
    const video = videoRef.current;
    if (!video) return;
    video.currentTime = seconds;
  }

  async function handlePublish(metadata: PublishMetadata, acknowledgeQC = false) {
    if (inFlightRef.current) return;
    inFlightRef.current = true;
    setIsSubmitting(true);
    setError(null);
    let qcBlocked = false;
    try {
      await startPublishSaga(
        projectId,
        {
          ...metadata,
          thumbnail_path: thumbnailPath ?? undefined,
          // Left out when no channel is connected yet, so the Publisher
          // reports "not authenticated" rather than "channel '' not found".
          channel_id: channelId ?? undefined,
        },
        acknowledgeQC,
      );
      /*
        The POST only *starts* the saga; the project is now "publishing" and
        useProject's poll drives the rest of the UI. isSubmitting stays true
        until that refetch lands so the button never flickers back to enabled
        in between.
      */
      await refetch();
    } catch (err) {
      /*
        CR-021 FR61.3: QC chặn là loại 409 duy nhất có đường đi tiếp. Xin đồng ý
        SAU khi đã nhả cờ in-flight ở finally, nếu không lần gọi lại sẽ bị chính
        cái chốt chống bấm hai lần chặn mất.
      */
      if (err instanceof ApiError && err.code === ERROR_CODE_QC_BLOCKED && !acknowledgeQC) {
        qcBlocked = true;
      } else {
        setError(err instanceof ApiError ? err.message : String(err));
      }
    } finally {
      inFlightRef.current = false;
      setIsSubmitting(false);
    }

    /*
      Không tự gửi lại kèm acknowledge_qc — Creator phải chủ động đồng ý, và
      máy chủ ghi lại lần bỏ qua đó. Hỏi đúng một lần: nếu lần gửi có cờ vẫn bị
      chặn thì đó là lỗi khác, và nó đi vào nhánh hiện nguyên văn ở trên.
    */
    if (
      qcBlocked &&
      window.confirm(
        "Kiểm tra chất lượng phát hiện lỗi nghiêm trọng. Vẫn đăng video này? " +
          "Lần bỏ qua sẽ được ghi lại.",
      )
    ) {
      await handlePublish(metadata, true);
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

  if (!project) return null;

  const isPublished = project.status === "published" || Boolean(project.youtube_video_url);
  const isPublishing = isSubmitting || project.status === "publishing";
  const hasPublishFailed = project.status === "failed_at_publish_video";

  // CR-007 follow-up: "short" still renders the full long-form pipeline as
  // clip source (CR-007 D1), but the Creator picked this project to only care
  // about the vertical clip — the YouTube publish UI would just be clutter in
  // front of the thing they actually want, so it collapses behind a toggle.
  const outputMode = project.video_output_mode ?? "long";
  const wantsClips = outputMode === "short" || outputMode === "both";
  const showYoutubePublish = youtubePublishOverride ?? outputMode !== "short";
  const clipsPanel = wantsClips && (
    <ClipsPanel projectId={projectId} clips={project.clips ?? []} videoOutputMode={outputMode} />
  );

  const errorBanner = error && (
    <p role="alert" className={glass.helperText}>
      {error}
    </p>
  );

  return (
    <div data-testid="publish-page">
      <AppShell
        currentStep={6}
        wide
        title="Đăng video"
        subtitle="Kết nối YouTube và điền thông tin để xuất bản."
        headerAction={
          <Link to={`/projects/${projectId}/result`} className={glass.ghostBtn} style={{ textDecoration: "none" }}>
            Quay lại xem kết quả
          </Link>
        }
      >
        {isPublished ? (
          <div className={`${glass.card} ${styles.publishedCard}`}>
            <p className={styles.publishedTitle}>Đã đăng thành công!</p>
            <a href={project.youtube_video_url ?? undefined}>{project.youtube_video_url}</a>
            {project.caption_status === "skipped_no_scope" && (
              <p role="alert" className={`${glass.helperText} ${glass.mtSm}`}>
                Video không có phụ đề YouTube: kênh này cần được nối lại để cấp thêm quyền. Vào mục
                Kênh YouTube ở lần đăng sau, ngắt rồi nối lại kênh này.
              </p>
            )}
            {project.caption_status === "failed" && (
              <p role="alert" className={`${glass.helperText} ${glass.mtSm}`}>
                Video đã đăng nhưng tải phụ đề lên YouTube thất bại — thử đăng lại, hoặc tải phụ đề
                lên thủ công trong YouTube Studio.
              </p>
            )}
            {project.video_path && (
              <div className={glass.mtMd}>
                <VideoPlayer videoSrc={getProjectVideoUrl(projectId)} />
              </div>
            )}
            {clipsPanel}
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
              {project.video_path && (
                <VideoPlayer videoSrc={getProjectVideoUrl(projectId)} videoRef={videoRef} />
              )}
              {clipsPanel}
            </div>

            <div className={styles.publishColumn}>
              {outputMode === "short" && !showYoutubePublish ? (
                <button
                  type="button"
                  className={glass.ghostBtn}
                  onClick={() => setYoutubePublishOverride(true)}
                  data-testid="result-show-youtube-publish"
                >
                  Cũng muốn đăng bản dài này lên YouTube?
                </button>
              ) : isPublishing ? (
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
                      <div className={`${glass.ctaRow} ${glass.mtSm}`}>
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
                    Trước nút đăng, không sau (FR61.2): báo cáo chỉ có tác dụng
                    nếu Creator đọc nó trước khi quyết định đăng.
                  */}
                  <QCReportPanel projectId={projectId} onSeek={handleSeek} />
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
      </AppShell>
    </div>
  );
}
