import { useCallback, useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { AppShell } from "../components/AppShell";
import { deleteProject, getProjectVideoUrl, listProjects, ApiError } from "../api/client";
import type { ProjectSummary } from "../types";
import glass from "../styles/glass.module.css";
import styles from "./VideoListPage.module.css";
import { statusLabel } from "../utils/pipelineLabels";

function statusBadgeClass(status: string): string {
  if (status.startsWith("failed_at_")) return styles.badgeFailed;
  if (status === "published" || status === "ready_to_publish") return styles.badgeSuccess;
  if (status === "draft") return styles.badgeNeutral;
  return styles.badgeProgress;
}

function TrashIcon() {
  return (
    <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M3 6h18M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2m3 0-1 14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2L4 6h16Z" />
    </svg>
  );
}

export function VideoListPage() {
  const [projects, setProjects] = useState<ProjectSummary[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [isBulkDeleting, setIsBulkDeleting] = useState(false);
  const [selected, setSelected] = useState<Set<string>>(new Set());

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

  const allSelected = useMemo(
    () => !!projects && projects.length > 0 && projects.every((p) => selected.has(p.project_id)),
    [projects, selected],
  );

  function toggleSelectAll() {
    if (!projects) return;
    setSelected(allSelected ? new Set() : new Set(projects.map((p) => p.project_id)));
  }

  function toggleSelect(projectId: string) {
    setSelected((current) => {
      const next = new Set(current);
      if (next.has(projectId)) next.delete(projectId);
      else next.add(projectId);
      return next;
    });
  }

  async function handleDelete(projectId: string) {
    if (!window.confirm("Xoá video này và toàn bộ dữ liệu liên quan? Hành động này không thể hoàn tác.")) {
      return;
    }
    setDeletingId(projectId);
    setError(null);
    try {
      await deleteProject(projectId);
      setProjects((current) => current?.filter((p) => p.project_id !== projectId) ?? null);
      setSelected((current) => {
        const next = new Set(current);
        next.delete(projectId);
        return next;
      });
    } catch (err) {
      setError(err instanceof ApiError ? err.message : String(err));
    } finally {
      setDeletingId(null);
    }
  }

  async function handleBulkDelete() {
    const ids = Array.from(selected);
    if (ids.length === 0) return;
    if (
      !window.confirm(
        `Xoá ${ids.length} video đã chọn và toàn bộ dữ liệu liên quan? Hành động này không thể hoàn tác.`,
      )
    ) {
      return;
    }
    setIsBulkDeleting(true);
    setError(null);
    const results = await Promise.allSettled(ids.map((id) => deleteProject(id)));
    const failedIds = ids.filter((_, index) => results[index].status === "rejected");

    setProjects((current) => current?.filter((p) => !ids.includes(p.project_id) || failedIds.includes(p.project_id)) ?? null);
    setSelected(new Set(failedIds));
    if (failedIds.length > 0) {
      setError(`Không thể xoá ${failedIds.length}/${ids.length} video. Vui lòng thử lại.`);
    }
    setIsBulkDeleting(false);
  }

  return (
    <div data-testid="video-list-page">
      <AppShell
        wide
        title="Danh sách video"
        subtitle="Tất cả video đã tạo, kể cả những video render thất bại. Xoá video không dùng nữa để giảm dung lượng."
        headerAction={
          <Link to="/" className={glass.btnPrimary} style={{ textDecoration: "none", padding: "9px 16px", fontSize: 13 }}>
            Tạo video mới
          </Link>
        }
      >
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
          <>
            <div className={styles.toolbar}>
              <label className={styles.selectAllLabel}>
                <input
                  type="checkbox"
                  className={styles.checkbox}
                  checked={allSelected}
                  onChange={toggleSelectAll}
                  aria-label="Chọn tất cả"
                />
                Chọn tất cả
              </label>

              {selected.size > 0 && (
                <span className={styles.selectedCount}>Đã chọn {selected.size} video</span>
              )}

              <button
                type="button"
                data-testid="bulk-delete-button"
                className={glass.dangerBtn}
                disabled={selected.size === 0 || isBulkDeleting}
                onClick={handleBulkDelete}
              >
                <TrashIcon />
                {isBulkDeleting ? "Đang xoá..." : `Xoá đã chọn${selected.size > 0 ? ` (${selected.size})` : ""}`}
              </button>
            </div>

            <div className={glass.card}>
              {projects.map((project) => (
                <div
                  key={project.project_id}
                  data-testid={`video-row-${project.project_id}`}
                  className={`${styles.row} ${selected.has(project.project_id) ? styles.rowSelected : ""}`}
                >
                  <input
                    type="checkbox"
                    className={styles.checkbox}
                    checked={selected.has(project.project_id)}
                    onChange={() => toggleSelect(project.project_id)}
                    aria-label={`Chọn video ${project.project_id}`}
                  />

                  <div className={styles.rowMain}>
                    <div className={styles.projectId}>{project.project_id}</div>
                    <div className={styles.meta}>
                      <span className={`${styles.badge} ${statusBadgeClass(project.status)}`}>
                        <span className={styles.badgeDot} />
                        {statusLabel(project.status)}
                      </span>
                      <span className={glass.cardHint}>
                        {project.error_message ? `${project.error_message} · ` : ""}
                        {new Date(project.updated_at).toLocaleString("vi-VN")}
                      </span>
                    </div>
                  </div>

                  <div className={styles.actions}>
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
                      className={glass.dangerGhostBtn}
                      disabled={deletingId === project.project_id || isBulkDeleting}
                      onClick={() => handleDelete(project.project_id)}
                    >
                      <TrashIcon />
                      {deletingId === project.project_id ? "Đang xoá..." : "Xoá"}
                    </button>
                  </div>
                </div>
              ))}
            </div>
          </>
        )}
      </AppShell>
    </div>
  );
}
