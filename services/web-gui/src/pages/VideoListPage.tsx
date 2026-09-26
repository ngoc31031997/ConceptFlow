import { useCallback, useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { AppShell } from "../components/AppShell";
import { StatusBadge } from "../components/StatusBadge";
import { DeleteProgressCard } from "../components/DeleteProgressCard";
import { RenderEngineBadge } from "../components/RenderEngineBadge";
import { deleteProject, getProjectVideoUrl, listProjects, ApiError } from "../api/client";
import type { ProjectSummary } from "../types";

import { projectPath } from "../utils/pipelineLabels";
import { FLOW_LABELS, stepStatus } from "../utils/flow";
import { Card } from "../components/ui";
import glass from "../styles/glass.module.css";
import styles from "./VideoListPage.module.css";

function TrashIcon() {
  return (
    <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M3 6h18M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2m3 0-1 14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2L4 6h16Z" />
    </svg>
  );
}

type Filter = "all" | "running" | "waiting" | "problem" | "done";

const FILTERS: { key: Filter; label: string }[] = [
  { key: "all", label: "Tất cả" },
  { key: "running", label: "Đang chạy" },
  { key: "waiting", label: "Chờ bạn" },
  { key: "problem", label: "Lỗi / đã hủy" },
  { key: "done", label: "Xong" },
];

function matches(p: ProjectSummary, filter: Filter): boolean {
  const step = p.flow_step ?? 0;
  switch (filter) {
    case "running":
      return p.run_state === "running";
    case "problem":
      return p.run_state === "failed" || p.run_state === "cancelled";
    case "done":
      return step >= 12 && p.run_state !== "failed";
    case "waiting":
      return p.run_state === "idle" && step > 0 && step < 12;
    default:
      return true;
  }
}

/** 13 ô nhỏ, một ô một bước: dự án đi tới đâu, đang chạy hay lỗi ở ô nào. */
function FlowMini({ project }: { project: ProjectSummary }) {
  const flowStep = project.flow_step ?? 0;
  if (!flowStep) return null;
  return (
    <span
      className={styles.mini}
      role="img"
      aria-label={`Bước ${flowStep}/13: ${FLOW_LABELS[flowStep - 1]}`}
      data-testid="flow-mini"
    >
      {FLOW_LABELS.map((label, i) => (
        <i key={label} className={styles[`mini_${stepStatus(i + 1, flowStep, project.run_state)}`]} title={label} />
      ))}
    </span>
  );
}

export function VideoListPage() {
  const [projects, setProjects] = useState<ProjectSummary[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [isBulkDeleting, setIsBulkDeleting] = useState(false);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [filter, setFilter] = useState<Filter>("all");
  const [stepFilter, setStepFilter] = useState<Set<number>>(new Set());

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

  const visible = useMemo(
    () => (projects ?? []).filter(
        (p) => matches(p, filter) && (stepFilter.size === 0 || stepFilter.has(p.flow_step ?? 0)),
      ),
    [projects, filter, stepFilter],
  );
  const nameOf = useMemo(() => {
    const byId = new Map((projects ?? []).map((p) => [p.project_id, p.topic || p.project_id.slice(0, 8)]));
    return (id: string) => byId.get(id) ?? id.slice(0, 8);
  }, [projects]);

  const allSelected = useMemo(
    () => visible.length > 0 && visible.every((p) => selected.has(p.project_id)),
    [visible, selected],
  );

  function toggleStepFilter(step: number) {
    setStepFilter((current) => {
      const next = new Set(current);
      if (next.has(step)) next.delete(step);
      else next.add(step);
      return next;
    });
  }

  function toggleSelectAll() {
    setSelected(allSelected ? new Set() : new Set(visible.map((p) => p.project_id)));
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
    if (!window.confirm("Xóa video này và toàn bộ dữ liệu liên quan? Hành động này không thể hoàn tác.")) {
      return;
    }
    setError(null); // Clear previous errors
    setDeletingId(projectId);
    try {
      // 202: the delete saga runs on; the row goes when DeleteProgressCard
      // reports every service has cleaned up (FR116.2).
      await deleteProject(projectId);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : String(err));
      setDeletingId(null);
    }
  }

  const handleDeleteDone = useCallback(() => {
    setDeletingId((id) => {
      if (id) {
        setProjects((current) => current?.filter((p) => p.project_id !== id) ?? null);
        setSelected((current) => {
          const next = new Set(current);
          next.delete(id);
          return next;
        });
      }
      return null;
    });
  }, []);

  async function handleBulkDelete() {
    const ids = Array.from(selected);
    if (ids.length === 0) return;
    if (
      !window.confirm(
        `Xóa ${ids.length} video đã chọn và toàn bộ dữ liệu liên quan? Hành động này không thể hoàn tác.`,
      )
    ) {
      return;
    }
    setError(null); // Clear previous errors
    setIsBulkDeleting(true);
    const results = await Promise.allSettled(ids.map((id) => deleteProject(id)));
    const failedIds = ids.filter((_, index) => results[index].status === "rejected");

    setProjects((current) => current?.filter((p) => !ids.includes(p.project_id) || failedIds.includes(p.project_id)) ?? null);
    setSelected(new Set(failedIds));
    if (failedIds.length > 0) {
      setError(`Không thể xóa ${failedIds.length}/${ids.length} video. Vui lòng thử lại.`);
    }
    setIsBulkDeleting(false);
  }

  return (
    <div data-testid="video-list-page">
      <AppShell
        wide
        title="Danh sách video"
        subtitle="Tất cả video đã tạo, kể cả video bị lỗi. Xóa video không dùng để giải phóng dung lượng."
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
            <div className={styles.filters} role="tablist" aria-label="Lọc theo trạng thái">
              {FILTERS.map((f) => (
                <button
                  key={f.key}
                  type="button"
                  role="tab"
                  aria-selected={filter === f.key}
                  className={`${styles.chip} ${filter === f.key ? styles.chipOn : ""}`}
                  onClick={() => setFilter(f.key)}
                  data-testid={`filter-${f.key}`}
                >
                  {f.label}
                  <span className={styles.chipCount}>{(projects ?? []).filter((p) => matches(p, f.key)).length}</span>
                </button>
              ))}
              <details className={styles.stepFilter} data-testid="step-filter">
                <summary className={`${styles.chip} ${stepFilter.size > 0 ? styles.chipOn : ""}`}>
                  Bước{stepFilter.size > 0 ? ` (${stepFilter.size})` : ""}
                </summary>
                <div className={styles.stepMenu}>
                  {FLOW_LABELS.map((label, i) => (
                    <label key={label} className={styles.stepOption}>
                      <input
                        type="checkbox"
                        checked={stepFilter.has(i + 1)}
                        onChange={() => toggleStepFilter(i + 1)}
                        data-testid={`step-filter-${i + 1}`}
                      />
                      {i + 1}. {label}
                    </label>
                  ))}
                  {stepFilter.size > 0 && (
                    <button type="button" className={styles.chip} onClick={() => setStepFilter(new Set())}>
                      Bỏ chọn
                    </button>
                  )}
                </div>
              </details>
            </div>

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
                {isBulkDeleting ? "Đang xóa..." : `Xóa đã chọn${selected.size > 0 ? ` (${selected.size})` : ""}`}
              </button>
            </div>

            <Card>
              {visible.length === 0 && <p className={glass.helperText}>Không có video nào ở nhóm này.</p>}
              {visible.map((project) => (
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
                    <div className={styles.name} data-testid="project-name">
                      {project.topic || <span className={styles.unnamed}>(chưa đặt chủ đề)</span>}
                      <span className={styles.projectId}>{project.project_id.slice(0, 8)}</span>
                    </div>
                    {project.forked_from && (
                      <div className={styles.lineage} data-testid="lineage">
                        Bản mới từ “{nameOf(project.forked_from)}”
                      </div>
                    )}
                    <FlowMini project={project} />
                    <div className={styles.meta}>
                      {project.flow_step ? (
                        <span className={glass.cardHint} data-testid="wizard-step">
                          Bước {project.flow_step} — {FLOW_LABELS[project.flow_step - 1]}
                        </span>
                      ) : null}
                      <StatusBadge status={project.status} />
                      <RenderEngineBadge renderEngine={project.render_engine} />
                      {project.error_message && (
                        <span className={styles.errorText}>{project.error_message}</span>
                      )}
                      <span className={glass.cardHint}>
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
                    {/* Bug report: "Chi tiết" từng luôn trỏ vào /result, vốn
                        chẳng hiển thị gì cho một project chưa xong — Creator
                        rời đi giữa chừng rồi quay lại qua danh sách này thì
                        không có đường về màn theo dõi. projectPath trả lời
                        "project ở trạng thái này thuộc màn nào", và từ CR-031
                        câu trả lời đó có thêm bước 4 (Validate). */}
                    <Link className={glass.ghostBtn} to={projectPath(project.project_id, project.status)}>
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
                      {deletingId === project.project_id ? "Đang xóa..." : "Xóa"}
                    </button>
                  </div>
                  {deletingId === project.project_id && (
                    <div style={{ gridColumn: "1 / -1", width: "100%" }}>
                      <DeleteProgressCard projectId={project.project_id} onDone={handleDeleteDone} />
                    </div>
                  )}
                </div>
              ))}
            </Card>
          </>
        )}
      </AppShell>
    </div>
  );
}
