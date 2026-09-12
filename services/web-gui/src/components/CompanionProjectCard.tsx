import { Link } from "react-router-dom";
import { useProject } from "../hooks/useProject";
import { getProjectVideoUrl } from "../api/client";
import { VideoPlayer } from "./VideoPlayer";
import { ClipsPanel } from "./ClipsPanel";
import { StatusBadge } from "./StatusBadge";
import glass from "../styles/glass.module.css";

interface CompanionProjectCardProps {
  companionProjectId: string;
}

/**
 * CR-026 FR73.3 — project cùng chủ đề (bản dài/bản ngắn kia), hiện tóm tắt
 * ngay trên Result page thay vì bắt Creator tự đi tìm trong danh sách video.
 *
 * Cố ý KHÔNG nhân đôi toàn bộ PublishForm/YoutubeChannels lên đây (D6) — chỉ
 * đủ để biết trạng thái và xem/tải nhanh; muốn đăng/thao tác đầy đủ thì bấm
 * "Xem đầy đủ" sang trang riêng của nó (mỗi project vẫn có trang đầy đủ của
 * mình, y như trước CR-026).
 */
export function CompanionProjectCard({ companionProjectId }: CompanionProjectCardProps) {
  const { project, error } = useProject(companionProjectId);

  return (
    <div className={glass.card} data-testid="companion-project-card">
      <div className={glass.cardHeader}>
        <span className={glass.cardTitle}>
          {project?.video_output_mode === "short" ? "Bản Shorts/TikTok riêng" : "Video cùng chủ đề"}
        </span>
        {project && <StatusBadge status={project.status} />}
      </div>

      {error && (
        <p role="alert" className={glass.helperText}>
          Không tải được project liên kết ({error}).
        </p>
      )}

      {!error && !project && <p className={glass.cardHint}>Đang tải...</p>}

      {project && (
        <>
          {project.video_path && (
            <div className={glass.mtSm}>
              <VideoPlayer videoSrc={getProjectVideoUrl(companionProjectId)} />
            </div>
          )}

          {(project.video_output_mode === "short" || project.video_output_mode === "both") && (
            <ClipsPanel
              projectId={companionProjectId}
              clips={project.clips ?? []}
              videoOutputMode={project.video_output_mode}
            />
          )}

          <Link
            className={`${glass.ghostBtn} ${glass.mtSm}`}
            style={{ textDecoration: "none", display: "inline-block" }}
            to={`/projects/${companionProjectId}/result`}
          >
            Xem đầy đủ
          </Link>
        </>
      )}
    </div>
  );
}
