import { useRef, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { VideoPlayer } from "../components/VideoPlayer";
import { YoutubeChannels } from "../components/YoutubeChannels";
import { ThumbnailUpload } from "../components/ThumbnailUpload";
import { PublishForm } from "../components/PublishForm";
import { AppShell } from "../components/AppShell";
import { RenderQualityPicker } from "../components/RenderQualityPicker";
import { useProject } from "../hooks/useProject";
import { QCReportPanel } from "../components/QCReportPanel";
import {
  startPublishSaga,
  startRenderSaga,
  retryProject,
  deleteProject,
  getProjectVideoUrl,
  ApiError,
  ERROR_CODE_QC_BLOCKED,
} from "../api/client";
import type { PublishMetadata } from "../types";
import type { RenderQuality } from "../context/ProjectDraftContext";
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
  const [showRerender, setShowRerender] = useState(false);
  const [rerenderQuality, setRerenderQuality] = useState<RenderQuality>("1080p60");
  const [isRerendering, setIsRerendering] = useState(false);
  const [rerenderError, setRerenderError] = useState<string | null>(null);
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

  /**
   * "Render lại ở chất lượng khác" (bug report): dùng lại chính project_id
   * này — StartRenderSagaUseCase (orchestrator) upsert theo project_id, nên
   * đây là một saga render mới chạy lại từ parse_script, không phải một bước
   * vá riêng. Chấp nhận cái giá đó (tốn TTS lại) để đổi lấy việc không phải
   * xây một đường dựng lại-chỉ-hình riêng — thường dùng đúng một lần, khi
   * chốt bản final ở chất lượng cao hơn bản đã duyệt nội dung.
   */
  async function handleRerender() {
    if (inFlightRef.current || !project) return;
    if (!project.script_content) {
      setRerenderError("Thiếu script gốc của project này — không render lại tự động được.");
      return;
    }
    inFlightRef.current = true;
    setIsRerendering(true);
    setRerenderError(null);
    try {
      await startRenderSaga({
        project_id: projectId,
        script_content: project.script_content,
        voice_language: project.voice_language,
        background_music_path: project.background_music_path,
        tts_enabled: project.tts_enabled ?? true,
        voice_id: project.voice_id,
        subtitle_mode: project.subtitle_mode ?? "track",
        subtitle_style: project.subtitle_style,
        render_quality: rerenderQuality,
        video_format_id: project.video_format_id,
        background_music_volume: project.background_music_volume,
      });
      navigate(`/projects/${projectId}/render`);
    } catch (err) {
      setRerenderError(err instanceof ApiError ? err.message : String(err));
    } finally {
      inFlightRef.current = false;
      setIsRerendering(false);
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
            {project.caption_status === "skipped_no_scope" && (
              <p role="alert" className={glass.helperText} style={{ marginTop: 12 }}>
                Video không có phụ đề YouTube: kênh này cần được nối lại để cấp thêm quyền. Vào mục
                Kênh YouTube ở lần đăng sau, ngắt rồi nối lại kênh này.
              </p>
            )}
            {project.caption_status === "failed" && (
              <p role="alert" className={glass.helperText} style={{ marginTop: 12 }}>
                Video đã đăng nhưng tải phụ đề lên YouTube thất bại — thử đăng lại, hoặc tải phụ đề
                lên thủ công trong YouTube Studio.
              </p>
            )}
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
              {project.video_path && (
                <VideoPlayer videoSrc={getProjectVideoUrl(projectId)} videoRef={videoRef} />
              )}
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

        {project.video_path && (
          <div className={glass.card} style={{ marginTop: 20, padding: 20 }} data-testid="rerender-section">
            {!showRerender ? (
              <button
                type="button"
                className={glass.ghostBtn}
                onClick={() => setShowRerender(true)}
                data-testid="rerender-toggle"
              >
                Render lại ở chất lượng khác
              </button>
            ) : (
              <>
                <div className={glass.cardTitle} style={{ marginBottom: 10 }}>
                  Render lại ở chất lượng khác
                </div>
                <p className={glass.cardHint} style={{ marginBottom: 14 }}>
                  Chạy lại toàn bộ pipeline cho video này ở chất lượng mới — tốn thời gian và (nếu có
                  giọng đọc) tốn quota TTS lại như một lần render mới. Dùng khi đã duyệt nội dung ở bản
                  nháp và muốn chốt bản cuối ở chất lượng cao hơn.
                </p>
                <RenderQualityPicker value={rerenderQuality} onChange={setRerenderQuality} />
                {rerenderError && (
                  <p role="alert" className={glass.helperText} style={{ marginTop: 10 }}>
                    {rerenderError}
                  </p>
                )}
                <div className={glass.ctaRow} style={{ marginTop: 14 }}>
                  <button
                    type="button"
                    className={glass.btnPrimary}
                    disabled={isRerendering}
                    onClick={handleRerender}
                    data-testid="rerender-submit"
                  >
                    {isRerendering ? "Đang bắt đầu..." : "Render lại"}
                  </button>
                  <button
                    type="button"
                    className={glass.ghostBtn}
                    disabled={isRerendering}
                    onClick={() => setShowRerender(false)}
                  >
                    Huỷ
                  </button>
                </div>
              </>
            )}
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
