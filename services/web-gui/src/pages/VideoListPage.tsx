import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { AppShell } from "../components/AppShell";
import { StatusBadge } from "../components/StatusBadge";
import { ConfirmModal } from "../components/ConfirmModal";
import { DeleteProgressCard } from "../components/DeleteProgressCard";
import { RenderEngineBadge } from "../components/RenderEngineBadge";
import { deleteProject, getProjectVideoUrl, listProjectsPage, ApiError } from "../api/client";
import type { ProjectListFilter, ProjectPage, ProjectSummary } from "../types";

import { projectPath } from "../utils/pipelineLabels";
import { FLOW_LABELS, stepStatus } from "../utils/flow";
import { PAGE_SIZES, pageCount } from "../utils/pagination";
import { Card, Pagination } from "../components/ui";
import glass from "../styles/glass.module.css";
import styles from "./VideoListPage.module.css";

function TrashIcon() {
  return (
    <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M3 6h18M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2m3 0-1 14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2L4 6h16Z" />
    </svg>
  );
}

const FILTERS: { key: ProjectListFilter; label: string }[] = [
  { key: "all", label: "Tất cả" },
  { key: "running", label: "Đang chạy" },
  { key: "waiting", label: "Chờ bạn" },
  { key: "problem", label: "Lỗi / đã hủy" },
  { key: "done", label: "Xong" },
];

/** 14 ô nhỏ, một ô một bước: dự án đi tới đâu, đang chạy hay lỗi ở ô nào. */
function FlowMini({ project }: { project: ProjectSummary }) {
  const flowStep = project.flow_step ?? 0;
  if (!flowStep) return null;
  return (
    <span
      className={styles.mini}
      role="img"
      aria-label={`Bước ${flowStep}/${FLOW_LABELS.length}: ${FLOW_LABELS[flowStep - 1]}`}
      data-testid="flow-mini"
    >
      {FLOW_LABELS.map((label, i) => (
        <i key={label} className={styles[`mini_${stepStatus(i + 1, flowStep, project.run_state)}`]} title={label} />
      ))}
    </span>
  );
}

export function VideoListPage() {
  // CR-054: the server filters, counts and pages the list; this screen holds
  // only the page on show.
  const [data, setData] = useState<ProjectPage | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [isBulkDeleting, setIsBulkDeleting] = useState(false);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [filter, setFilter] = useState<ProjectListFilter>("all");
  const [stepFilter, setStepFilter] = useState<Set<number>>(new Set());
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [confirmDeleteId, setConfirmDeleteId] = useState<string | null>(null);
  const [confirmBulkDelete, setConfirmBulkDelete] = useState(false);
  const requestSeq = useRef(0);
  const listTopRef = useRef<HTMLDivElement>(null);

  const refetch = useCallback(async () => {
    // Clicking through pages fast: only the latest request may land.
    const seq = ++requestSeq.current;
    try {
      const result = await listProjectsPage({
        page,
        pageSize,
        filter,
        steps: Array.from(stepFilter).sort((a, b) => a - b),
      });
      if (seq !== requestSeq.current) return;
      setData(result);
      // Past the last page (e.g. after deleting its rows) the server answers
      // with the last page.
      if (result.page !== page) setPage(result.page);
      setError(null);
    } catch (err) {
      if (seq !== requestSeq.current) return;
      setError(err instanceof ApiError ? err.message : String(err));
    }
  }, [page, pageSize, filter, stepFilter]);

  useEffect(() => {
    refetch();
  }, [refetch]);

  // DeleteProgressCard keeps onDone in an effect's deps, so handleDeleteDone
  // must stay stable while refetch changes with the page.
  const refetchRef = useRef(refetch);
  refetchRef.current = refetch;

  const rows = data?.projects ?? [];
  const pages = data ? pageCount(data.total, data.page_size) : 1;

  const allSelected = useMemo(
    () => rows.length > 0 && rows.every((p) => selected.has(p.project_id)),
    [rows, selected],
  );

  function chooseFilter(next: ProjectListFilter) {
    setFilter(next);
    setPage(1);
  }

  function toggleStepFilter(step: number) {
    setStepFilter((current) => {
      const next = new Set(current);
      if (next.has(step)) next.delete(step);
      else next.add(step);
      return next;
    });
    setPage(1);
  }

  function clearStepFilter() {
    setStepFilter(new Set());
    setPage(1);
  }

  function goToPage(next: number) {
    setPage(next);
    listTopRef.current?.scrollIntoView?.({ block: "start", behavior: "smooth" });
  }

  function choosePageSize(size: number) {
    setPageSize(size);
    setPage(1);
  }

  // "Chọn tất cả" covers the page on show; picks on other pages stay.
  function toggleSelectAll() {
    setSelected((current) => {
      const next = new Set(current);
      for (const p of rows) {
        if (allSelected) next.delete(p.project_id);
        else next.add(p.project_id);
      }
      return next;
    });
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
        setSelected((current) => {
          const next = new Set(current);
          next.delete(id);
          return next;
        });
      }
      return null;
    });
    // Rows from the next page move up into the gap.
    refetchRef.current();
  }, []);

  async function handleBulkDelete() {
    const ids = Array.from(selected);
    if (ids.length === 0) return;
    setError(null); // Clear previous errors
    setIsBulkDeleting(true);
    const results = await Promise.allSettled(ids.map((id) => deleteProject(id)));
    const failedIds = ids.filter((_, index) => results[index].status === "rejected");

    setSelected(new Set(failedIds));
    if (failedIds.length > 0) {
      setError(`Không thể xóa ${failedIds.length}/${ids.length} video. Vui lòng thử lại.`);
    }
    setIsBulkDeleting(false);
    await refetchRef.current();
  }

  return (
    <div data-testid="video-list-page">
      <AppShell
        wide
        title="Danh sách video"
        subtitle="Tất cả video đã tạo, kể cả video bị lỗi. Xóa video không dùng để giải phóng dung lượng."
      >
        {error && (
          <p role="alert" className={glass.helperText}>
            {error}
          </p>
        )}

        {data === null ? (
          !error && <p className={glass.helperText}>Đang tải danh sách...</p>
        ) : data.counts.all === 0 ? (
          <p className={glass.helperText}>Chưa có video nào.</p>
        ) : (
          <>
            <div ref={listTopRef} className={styles.filters} role="tablist" aria-label="Lọc theo trạng thái">
              {FILTERS.map((f) => (
                <button
                  key={f.key}
                  type="button"
                  role="tab"
                  aria-selected={filter === f.key}
                  className={`${styles.chip} ${filter === f.key ? styles.chipOn : ""}`}
                  onClick={() => chooseFilter(f.key)}
                  data-testid={`filter-${f.key}`}
                >
                  {f.label}
                  <span className={styles.chipCount}>{data.counts[f.key]}</span>
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
                    <button type="button" className={styles.chip} onClick={clearStepFilter}>
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

              <button
                type="button"
                data-testid="bulk-delete-button"
                className={glass.dangerBtn}
                disabled={selected.size === 0 || isBulkDeleting}
                onClick={() => setConfirmBulkDelete(true)}
              >
                <TrashIcon />
                {isBulkDeleting ? "Đang xóa..." : `Xóa đã chọn${selected.size > 0 ? ` (${selected.size})` : ""}`}
              </button>
              {selected.size > 0 && (
                <span className={`${styles.selectedCount} ${glass.reveal}`}>Đã chọn {selected.size} video</span>
              )}
            </div>

            <Card>
              {rows.length === 0 && <p className={glass.helperText}>Không có video nào ở nhóm này.</p>}
              {rows.map((project) => (
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
                        Bản mới từ “{project.forked_from_topic || project.forked_from.slice(0, 8)}”
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
                      onClick={() => setConfirmDeleteId(project.project_id)}
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

            {pages > 1 && (
              <Pagination
                page={data.page}
                pageCount={pages}
                total={data.total}
                pageSize={data.page_size}
                pageSizes={PAGE_SIZES}
                label="video"
                onPageChange={goToPage}
                onPageSizeChange={choosePageSize}
              />
            )}
          </>
        )}
      </AppShell>

      <ConfirmModal
        isOpen={confirmDeleteId !== null}
        onClose={() => setConfirmDeleteId(null)}
        onConfirm={() => {
          if (confirmDeleteId) handleDelete(confirmDeleteId);
        }}
        title="Xác nhận xóa video"
        message="Xóa video này và toàn bộ dữ liệu liên quan? Hành động này không thể hoàn tác."
        confirmLabel="Xóa video"
        cancelLabel="Hủy"
        isDangerous
      />

      <ConfirmModal
        isOpen={confirmBulkDelete}
        onClose={() => setConfirmBulkDelete(false)}
        onConfirm={handleBulkDelete}
        title="Xác nhận xóa nhiều video"
        message={`Xóa ${selected.size} video đã chọn và toàn bộ dữ liệu liên quan? Hành động này không thể hoàn tác.`}
        confirmLabel="Xóa đã chọn"
        cancelLabel="Hủy"
        isDangerous
      />
    </div>
  );
}
