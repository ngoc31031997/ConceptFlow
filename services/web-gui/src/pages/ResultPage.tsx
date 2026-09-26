import { useCallback, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { NEW_VIDEO_STATE } from "../context/ProjectDraftContext";
import { VideoPlayer } from "../components/VideoPlayer";
import { AppShell } from "../components/AppShell";
import { RenderQualityPicker } from "../components/RenderQualityPicker";
import { VideoOutputModePicker } from "../components/VideoOutputModePicker";
import { ConfirmModal } from "../components/ConfirmModal";
import { useProject } from "../hooks/useProject";
import { ProjectInputPanel } from "../components/ProjectInputPanel";
import { ClipsPanel } from "../components/ClipsPanel";
import { ShortScriptAssistant } from "../components/ShortScriptAssistant";
import { CompanionProjectCard } from "../components/CompanionProjectCard";
import { DeleteProgressCard } from "../components/DeleteProgressCard";
import { Disclosure } from "../components/Disclosure";
import { startRenderSaga, deleteProject, getProjectVideoUrl, ApiError } from "../api/client";
import type { RenderQuality, VideoOutputMode } from "../context/ProjectDraftContext";
import { Button, Card } from "../components/ui";
import glass from "../styles/glass.module.css";
import styles from "./ResultPage.module.css";

/**
 * Bước 5 — "Kết quả" (bug report, 2026-09-12): trước đây trang này vừa xem
 * lại video vừa đăng bài cùng lúc, nên "chỉ muốn xem/chỉnh sửa" và "chỉ muốn
 * đăng" luôn phải đi qua chung một trang dài. Tách ra: trang này CHỈ xem lại
 * và các thao tác khác (render lại, tạo bản Shorts, xem input, xóa) — đăng
 * bài chuyển hẳn sang `PublishPage` (Bước 6), tới đây bằng nút "Tiếp tục để
 * đăng" hoặc link "Xem chi tiết" khi đã đăng rồi.
 */
export function ResultPage() {
  const { id } = useParams<{ id: string }>();
  const projectId = id ?? "";
  const navigate = useNavigate();
  const { project } = useProject(projectId);
  const [isDeleting, setIsDeleting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [rerenderQuality, setRerenderQuality] = useState<RenderQuality>("1080p60");
  // null = "not touched yet": the sensible default is "carry over this
  // project's current mode", which is not known until `project` loads.
  const [rerenderOutputModeOverride, setRerenderOutputModeOverride] = useState<VideoOutputMode | null>(
    null,
  );
  const [isRerendering, setIsRerendering] = useState(false);
  const [rerenderError, setRerenderError] = useState<string | null>(null);

  /**
   * "Render lại ở chất lượng khác" (bug report): dùng lại chính project_id
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
        video_output_mode: rerenderOutputMode,
        video_format_id: project.video_format_id,
        background_music_volume: project.background_music_volume,
      });
      // Render lại chạy lại saga TỪ ĐẦU, nên nó bắt đầu ở bước 4 (chạy thử +
      // duyệt dàn ý), không phải bước 5.
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
      // 202: stay on the page with the progress card until the saga finishes
      // (FR116.2); DeleteProgressCard's onDone navigates away.
      await deleteProject(projectId);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : String(err));
      setIsDeleting(false);
    }
  }

  const handleDeleteDone = useCallback(() => navigate("/videos"), [navigate]);

  if (!project) return null;

  const isPublished = project.status === "published" || Boolean(project.youtube_video_url);

  // CR-007 follow-up: a clip only ever comes from `with self.clip(...)` in
  // the script — picking "short"/"both" alone never produces one.
  const outputMode = project.video_output_mode ?? "long";
  const wantsClips = outputMode === "short" || outputMode === "both";
  const rerenderOutputMode = rerenderOutputModeOverride ?? outputMode;

  return (
    <div data-testid="result-page">
      <AppShell
        currentStep={12}
        wide
        title="Xem kết quả"
        subtitle="Xem lại video, cắt clip hoặc dựng lại. Đăng video ở bước tiếp theo."
        headerAction={
          <Link to="/" state={NEW_VIDEO_STATE} className={glass.ghostBtn} style={{ textDecoration: "none" }}>
            Tạo video mới
          </Link>
        }
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
          UX review #2 — the highest-impact fix on this page: these used to be
          3-4 full-weight glass cards stacked as peers of the primary preview
          +publish layout above, each with its own bespoke toggle. Grouped
          under one demoted heading, using the one shared Disclosure
          affordance (review #5), so the page reads as "primary: preview &
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
              title="Dựng lại với chất lượng hoặc loại video khác"
              hint="Video sẽ được tạo lại từ đầu, mất thời gian như lần đầu."
              testId="rerender"
            >
              <RenderQualityPicker value={rerenderQuality} onChange={setRerenderQuality} />
              <div className={glass.mtSm}>
                <div className={glass.cardTitle} style={{ marginBottom: 10 }}>
                  Loại video
                </div>
                <VideoOutputModePicker value={rerenderOutputMode} onChange={setRerenderOutputModeOverride} bare />
              </div>
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

          {!project.companion_project_id && project.video_path && (
            <Disclosure
              title="Tạo bản Shorts/TikTok riêng cho video này"
              hint="Một kịch bản riêng, ngắn gọn và tự hoàn chỉnh, không cắt từ video này."
              testId="short-companion"
            >
              <ShortScriptAssistant
                sourceProjectId={projectId}
                sourceScriptContent={project.script_content}
                contentLanguage={project.voice_language}
                onCreated={(newProjectId) => navigate(`/projects/${newProjectId}/validate`)}
              />
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
