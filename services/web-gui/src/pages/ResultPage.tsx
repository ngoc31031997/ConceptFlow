import { useCallback, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { VideoPlayer } from "../components/VideoPlayer";
import { AppShell } from "../components/AppShell";
import { RenderQualityPicker } from "../components/RenderQualityPicker";
import { ConfirmModal } from "../components/ConfirmModal";
import { useProject } from "../hooks/useProject";
import { ProjectInputPanel } from "../components/ProjectInputPanel";
import { ClipsPanel } from "../components/ClipsPanel";
import { MakeShortButton } from "../components/MakeShortButton";
import { CompanionProjectCard } from "../components/CompanionProjectCard";
import { DeleteProgressCard } from "../components/DeleteProgressCard";
import { Disclosure } from "../components/Disclosure";
import { startRenderSaga, deleteProject, getProjectVideoUrl, ApiError } from "../api/client";
import type { RenderQuality } from "../context/ProjectDraftContext";
import { Button, Card } from "../components/ui";
import glass from "../styles/glass.module.css";
import styles from "./ResultPage.module.css";
import { FLOW_RESULT, flowTitle } from "../utils/flow";

/**
 * Bước 13 — "Kết quả": trang này CHỈ xem lại
 * và các thao tác khác (render lại, tạo bản Shorts, xem input, xóa) — đăng
 * bài chuyển hẳn sang `PublishPage` (bước 14), tới đây bằng nút "Tiếp tục để
 * đăng" hoặc link "Xem chi tiết" khi đã đăng rồi.
 */
export function ResultPage() {
  const { id } = useParams<{ id: string }>();
  const projectId = id ?? "";
  const navigate = useNavigate();
  const { project } = useProject(projectId);
  const outputMode = project?.video_output_mode ?? "long";
  const [isDeleting, setIsDeleting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [rerenderQuality, setRerenderQuality] = useState<RenderQuality>("1080p60");
  const [isRerendering, setIsRerendering] = useState(false);
  const [rerenderError, setRerenderError] = useState<string | null>(null);

  /**
   * "Render lại ở chất lượng khác": dùng lại chính project_id
   * này — StartRenderSagaUseCase (orchestrator) upsert theo project_id, nên
   * đây là một saga render mới chạy lại từ parse_script, không phải một bước
   * vá riêng. Chấp nhận cái giá đó (tốn TTS lại) để đổi lấy việc không phải
   * xây một đường dựng lại-chỉ-hình riêng — thường dùng đúng một lần, khi
   * chốt bản final ở chất lượng cao hơn bản đã duyệt nội dung.
   */
  async function handleRerender() {
    if (!project) return;
    if (!project.script_content) {
      setRerenderError("Dự án này thiếu script gốc nên không thể dựng lại tự động.");
      return;
    }
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
        render_engine: project.render_engine,
        // The frame (16:9 or 9:16) was fixed when the code was written, so a
        // re-render keeps the project's own output mode.
        video_output_mode: outputMode,
        video_format_id: project.video_format_id,
        background_music_volume: project.background_music_volume,
      });
      // Render lại chạy lại saga TỪ ĐẦU, nên nó bắt đầu ở bước 7 (chạy thử, rồi
      // duyệt dàn ý ở 8), không phải bước 9.
      navigate(`/projects/${projectId}/validate`);
    } catch (err) {
      setRerenderError(err instanceof ApiError ? err.message : String(err));
    } finally {
      setIsRerendering(false);
    }
  }

  async function handleDelete() {
    setError(null);
    setIsDeleting(true);
    try {
      // 202: stay on the page with the progress card until the saga finishes;
      // DeleteProgressCard's onDone navigates away.
      await deleteProject(projectId);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : String(err));
      setIsDeleting(false);
    }
  }

  const handleDeleteDone = useCallback(() => navigate("/videos"), [navigate]);

  if (!project) return null;

  const isPublished = project.status === "published" || Boolean(project.youtube_video_url);

  // Clips are only cut from a "both" project, and only from `with
  // self.clip(...)` in its script; a short is already vertical.
  const wantsClips = outputMode === "both";

  return (
    <div data-testid="result-page">
      <AppShell
        currentStep={FLOW_RESULT}
        wide
        title={flowTitle(FLOW_RESULT)}
        subtitle="Xem lại video, cắt clip hoặc dựng lại. Đăng video ở bước tiếp theo."
      >
        {error && (
          <p role="alert" className={glass.helperText}>
            {error}
          </p>
        )}

        <div className={styles.layout}>
          <div className={styles.preview}>
            {project.video_path && (
              <VideoPlayer
                videoSrc={getProjectVideoUrl(projectId)}
                scenes={project.scenes}
                contentLanguage={project.voice_language}
              />
            )}
            {wantsClips && (
              <ClipsPanel projectId={projectId} clips={project.clips ?? []} videoOutputMode={outputMode} />
            )}
          </div>

          <div className={styles.publishColumn}>
            {isPublished ? (
              <Card title="Đã đăng thành công" data-testid="result-published-banner">
                <a href={project.youtube_video_url ?? undefined}>{project.youtube_video_url}</a>
                <div className={glass.mtSm}>
                  <Link
                    className={glass.ghostBtn}
                    style={{ textDecoration: "none", display: "inline-block" }}
                    to={`/projects/${projectId}/publish`}
                  >
                    Xem chi tiết đăng bài
                  </Link>
                </div>
              </Card>
            ) : (
              <Card
                title="Sẵn sàng đăng?"
                hint="Nếu video đã ổn, chuyển sang bước đăng lên YouTube."
                data-testid="result-continue-to-publish"
              >
                <div className={`${glass.ctaRow} ${glass.mtSm}`}>
                  <Link
                    className={glass.btnPrimary}
                    style={{ textDecoration: "none" }}
                    to={`/projects/${projectId}/publish`}
                    data-testid="result-continue-to-publish-link"
                  >
                    Tiếp tục để đăng
                  </Link>
                </div>
              </Card>
            )}
          </div>
        </div>

        {/*
          Secondary panels are grouped under one demoted heading, using the
          shared Disclosure affordance, so the page reads as "primary: preview &
          publish" then "secondary: everything else" instead of five stacked
          look-alikes.
        */}
        <div className={styles.moreActions}>
          <div className={styles.moreActionsHeading}>Thao tác khác</div>

          {project.companion_project_id && (
            <CompanionProjectCard companionProjectId={project.companion_project_id} />
          )}

          {project.video_path && (
            <Disclosure
              title="Dựng lại với chất lượng khác"
              hint="Video sẽ được tạo lại từ đầu, mất thời gian như lần đầu."
              testId="rerender"
            >
              <RenderQualityPicker value={rerenderQuality} onChange={setRerenderQuality} />
              {rerenderError && (
                <p role="alert" className={`${glass.helperText} ${glass.mtXs}`}>
                  {rerenderError}
                </p>
              )}
              <div className={`${glass.ctaRow} ${glass.mtSm}`}>
                <Button disabled={isRerendering} onClick={handleRerender} data-testid="rerender-submit">
                  {isRerendering ? "Đang bắt đầu..." : "Dựng lại"}
                </Button>
              </div>
            </Disclosure>
          )}

          {!project.companion_project_id && project.video_path && outputMode !== "short" && (
            <Disclosure
              title="Làm bản short dọc cho video này"
              hint="Một video dọc 9:16 cùng chủ đề, kịch bản riêng, không cắt từ video này."
              testId="short-companion"
            >
              <MakeShortButton projectId={projectId} contentLanguage={project.voice_language} />
            </Disclosure>
          )}

          <ProjectInputPanel project={project} />
        </div>

        {/*
          A destructive, rarely-used action does not belong at the top of the
          page, above the video it deletes. It sits after the work instead.
        */}
        <div className={styles.dangerZone}>
          <span className={glass.cardHint}>Xóa vĩnh viễn video này và dữ liệu liên quan.</span>
          <Button
            variant="dangerGhost"
            data-testid="result-delete-button"
            disabled={isDeleting}
            onClick={() => setShowDeleteModal(true)}
          >
            {isDeleting ? "Đang xóa..." : "Xóa video"}
          </Button>
          {isDeleting && (
            <div style={{ flexBasis: "100%" }}>
              <DeleteProgressCard projectId={projectId} onDone={handleDeleteDone} />
            </div>
          )}
        </div>
      </AppShell>

      <ConfirmModal
        isOpen={showDeleteModal}
        onClose={() => setShowDeleteModal(false)}
        onConfirm={handleDelete}
        title="Xác nhận xóa video"
        message="Bạn có chắc chắn muốn xóa video này và toàn bộ dữ liệu liên quan? Hành động này không thể hoàn tác."
        confirmLabel="Xóa video"
        cancelLabel="Hủy"
        isDangerous
      />
    </div>
  );
}
