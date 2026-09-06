import { useCallback, useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { AppShell } from "../components/AppShell";
import { deleteProject, getProjectVideoUrl, listProjects, ApiError } from "../api/client";
import type { ProjectSummary } from "../types";
import glass from "../styles/glass.module.css";

const STATUS_LABELS: Record<string, string> = {
  draft: "Nháp",
  parsing_script: "Đang xử lý kịch bản",
  classifying_scenes: "Đang phân loại cảnh",
  synthesizing_speech: "Đang tổng hợp giọng đọc",
  rendering: "Đang render",
  assembling_video: "Đang ghép video",
  ready_to_publish: "Sẵn sàng đăng",
  publishing: "Đang đăng",
  published: "Đã đăng",
};

function statusLabel(status: string): string {
  if (status.startsWith("failed_at_")) return "Thất bại";
  return STATUS_LABELS[status] ?? status;
}

export function VideoListPage() {
  const [projects, setProjects] = useState<ProjectSummary[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [deletingId, setDeletingId] = useState<string | null>(null);

  const refetch = useCallback(async () => {
    try {
      const result = await listProjects();
      setProjects(result);
      setError(null);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : String(err));
    }
  }, []);

  useEffect(() => {
    refetch();
  }, [refetch]);

  async function handleDelete(projectId: string) {
    if (!window.confirm("Xoá video này và toàn bộ dữ liệu liên quan? Hành động này không thể hoàn tác.")) {
      return;
    }
    setDeletingId(projectId);
    setError(null);
    try {
      await deleteProject(projectId);
      setProjects((current) => current?.filter((p) => p.project_id !== projectId) ?? null);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : String(err));
    } finally {
      setDeletingId(null);
    }
  }

  return (
    <div data-testid="video-list-page">
      <AppShell title="Danh sách video" subtitle="Tất cả video đã tạo, kể cả những video render thất bại. Xoá video không dùng nữa để giảm dung lượng.">
        {error && (
          <p role="alert" className={glass.helperText}>
            {error}
          </p>
        )}

        {projects === null ? (
          <p className={glass.helperText}>Đang tải danh sách...</p>
        ) : projects.length === 0 ? (
          <p className={glass.helperText}>Chưa có video nào.</p>
        ) : (
          <div className={glass.card}>
            {projects.map((project) => (
              <div
                key={project.project_id}
                data-testid={`video-row-${project.project_id}`}
                style={{
                  display: "flex",
                  alignItems: "center",
                  justifyContent: "space-between",
                  gap: 16,
                  padding: "14px 0",
                  borderBottom: "1px solid var(--surface-border)",
                }}
              >
                <div>
                  <div style={{ fontWeight: 600, fontSize: 14 }}>{project.project_id}</div>
                  <div className={glass.cardHint}>
                    {statusLabel(project.status)}
                    {project.error_message ? ` — ${project.error_message}` : ""}
                    {" · "}
                    {new Date(project.updated_at).toLocaleString("vi-VN")}
                  </div>
                </div>

                <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
                  {project.video_path && (
                    <a
                      className={glass.ghostBtn}
                      href={getProjectVideoUrl(project.project_id)}
                      target="_blank"
                      rel="noreferrer"
                    >
                      Xem video
                    </a>
                  )}
                  <Link className={glass.ghostBtn} to={`/projects/${project.project_id}/result`}>
                    Chi tiết
                  </Link>
                  <button
                    type="button"
                    data-testid={`delete-button-${project.project_id}`}
                    className={glass.ghostBtn}
                    disabled={deletingId === project.project_id}
                    onClick={() => handleDelete(project.project_id)}
                  >
                    {deletingId === project.project_id ? "Đang xoá..." : "Xoá"}
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </AppShell>
    </div>
  );
}
