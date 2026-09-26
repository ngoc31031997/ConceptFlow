import { useEffect, useMemo, useState } from "react";
import { AppShell } from "../components/AppShell";
import { Card } from "../components/ui";
import { listRecentEvents, ApiError, type ProjectEvent } from "../api/client";
import { FLOW_LABELS } from "../utils/flow";
import glass from "../styles/glass.module.css";
import styles from "./JournalPage.module.css";

const STATE_LABEL: Record<ProjectEvent["run_state"], string> = {
  idle: "Chờ",
  running: "Đang chạy",
  done: "Xong",
  cancelled: "Đã hủy",
  failed: "Lỗi",
};

function formatDuration(ms?: number): string {
  if (!ms || ms <= 0) return "—";
  const s = Math.round(ms / 1000);
  if (s < 60) return `${s}s`;
  const m = Math.floor(s / 60);
  return `${m}m ${s % 60}s`;
}

function formatTime(iso: string): string {
  return new Date(iso).toLocaleString("vi-VN", { hour12: false });
}

export interface StepStat {
  step: number;
  label: string;
  runs: number;
  totalMs: number;
  maxMs: number;
  failures: number;
  tokens: number;
}

/** Bước saga chạy thật (có worker): thời gian ở đó là chi phí của bước. Review
 *  (7) và Kết quả (12) là thời gian chờ người, không tính vào chi phí. */
const SAGA_WORK_STEPS = new Set([6, 8, 9, 10, 11]);

/**
 * Gom cả nhật ký theo BƯỚC để thấy bước nào chậm/tốn/hay lỗi nhất. Dòng
 * authoring: thời gian của chính lượt chạy đó. Dòng saga: duration_ms là thời
 * gian đã ở bước TRƯỚC khi chuyển (from_flow_step), nên tính cho bước đó chứ
 * không cho bước vừa vào.
 */
export function aggregateByStep(events: ProjectEvent[]): StepStat[] {
  const stats = new Map<number, StepStat>();
  const at = (step: number): StepStat => {
    let cur = stats.get(step);
    if (!cur) {
      cur = { step, label: FLOW_LABELS[step - 1] ?? String(step), runs: 0, totalMs: 0, maxMs: 0, failures: 0, tokens: 0 };
      stats.set(step, cur);
    }
    return cur;
  };
  for (const e of events) {
    if (e.source === "authoring") {
      if (e.run_state === "running") continue; // dòng bắt đầu không có số đo
      const st = at(e.flow_step);
      if (e.run_state === "failed") st.failures += 1;
      st.runs += 1;
      st.totalMs += e.duration_ms ?? 0;
      st.maxMs = Math.max(st.maxMs, e.duration_ms ?? 0);
      st.tokens += (e.prompt_tokens ?? 0) + (e.completion_tokens ?? 0);
    } else {
      if (e.run_state === "failed" || e.run_state === "cancelled") at(e.flow_step).failures += e.run_state === "failed" ? 1 : 0;
      const from = e.from_flow_step ?? 0;
      if (SAGA_WORK_STEPS.has(from) && (e.duration_ms ?? 0) > 0) {
        const st = at(from);
        st.runs += 1;
        st.totalMs += e.duration_ms ?? 0;
        st.maxMs = Math.max(st.maxMs, e.duration_ms ?? 0);
      }
    }
  }
  return Array.from(stats.values()).sort((a, b) => a.step - b.step);
}

/**
 * Nhật ký sản xuất: hành trình của từng dự án qua 13 bước (project_events),
 * để biết một dự án đã qua bước nào, mất bao lâu, tốn bao nhiêu token, lỗi ở
 * đâu — làm căn cứ tối ưu về sau. Chỉ đọc.
 */
export function JournalPage() {
  const [events, setEvents] = useState<ProjectEvent[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [selected, setSelected] = useState<string | null>(null);
  const [view, setView] = useState<"overview" | "project">("overview");

  useEffect(() => {
    let cancelled = false;
    listRecentEvents(1000)
      .then((rows) => {
        if (!cancelled) setEvents(rows);
      })
      .catch((err) => {
        if (!cancelled) setError(err instanceof ApiError ? err.message : String(err));
      });
    return () => {
      cancelled = true;
    };
  }, []);

  // Dự án theo lần hoạt động gần nhất (events đã sắp mới → cũ).
  const projects = useMemo(() => {
    const seen = new Map<string, { id: string; last: ProjectEvent; count: number }>();
    for (const e of events ?? []) {
      const cur = seen.get(e.project_id);
      if (cur) cur.count += 1;
      else seen.set(e.project_id, { id: e.project_id, last: e, count: 1 });
    }
    return Array.from(seen.values());
  }, [events]);

  const activeId = selected ?? projects[0]?.id ?? null;
  const timeline = useMemo(
    () => (events ?? []).filter((e) => e.project_id === activeId).sort((a, b) => a.id - b.id),
    [events, activeId],
  );

  const overview = useMemo(() => aggregateByStep(events ?? []), [events]);
  const slowestAvg = Math.max(1, ...overview.map((o) => (o.runs ? o.totalMs / o.runs : 0)));

  // Tổng thời gian và token theo bước: chỗ nào tốn nhất hiện ra ngay.
  const perStep = useMemo(() => {
    const by = new Map<string, { label: string; ms: number; tokens: number; failed: number }>();
    for (const e of timeline) {
      if (e.run_state === "running") continue; // dòng bắt đầu không có số đo
      // Saga: duration_ms là thời gian nằm ở from_status, tính cho bước của
      // trạng thái đó — nhưng dòng này ghi theo bước ĐÍCH nên chỉ cộng cho
      // authoring; thời gian saga hiện ở cột riêng trong bảng.
      if (e.source !== "authoring") continue;
      const cur = by.get(e.step_label) ?? { label: e.step_label, ms: 0, tokens: 0, failed: 0 };
      cur.ms += e.duration_ms ?? 0;
      cur.tokens += (e.prompt_tokens ?? 0) + (e.completion_tokens ?? 0);
      if (e.run_state === "failed") cur.failed += 1;
      by.set(e.step_label, cur);
    }
    return Array.from(by.values());
  }, [timeline]);

  return (
    <div data-testid="journal-page">
      <AppShell
        wide
        title="Nhật ký sản xuất"
        subtitle="Theo dõi từng dự án qua các bước: thời gian, chi phí và lỗi phát sinh."
      >
        {error && (
          <p role="alert" className={glass.helperText}>
            {error}
          </p>
        )}
        {events === null && !error && <p className={glass.helperText}>Đang tải nhật ký...</p>}
        {events !== null && projects.length === 0 && (
          <p className={glass.helperText}>Chưa có hoạt động nào được ghi lại.</p>
        )}

        {projects.length > 0 && (
          <div className={styles.tabs} role="tablist" aria-label="Kiểu xem nhật ký">
            <button type="button" role="tab" aria-selected={view === "overview"} className={`${styles.tab} ${view === "overview" ? styles.tabOn : ""}`} onClick={() => setView("overview")} data-testid="journal-tab-overview">
              Tổng quan theo bước
            </button>
            <button type="button" role="tab" aria-selected={view === "project"} className={`${styles.tab} ${view === "project" ? styles.tabOn : ""}`} onClick={() => setView("project")} data-testid="journal-tab-project">
              Từng dự án
            </button>
          </div>
        )}

        {projects.length > 0 && view === "overview" && (
          <Card title="Bước nào chậm, tốn, hay lỗi nhất" hint={`Tính trên ${projects.length} dự án, ${events?.length ?? 0} sự kiện gần nhất. Thời gian chờ bạn thao tác không được tính.`}>
            <div className={styles.tbl}>
              <table className={styles.table} data-testid="journal-overview">
                <thead>
                  <tr>
                    <th>Bước</th>
                    <th className={styles.num}>Lượt</th>
                    <th className={styles.num}>Trung bình</th>
                    <th className={styles.barcell} />
                    <th className={styles.num}>Lâu nhất</th>
                    <th className={styles.num}>Token</th>
                    <th className={styles.num}>Lỗi</th>
                  </tr>
                </thead>
                <tbody>
                  {overview.map((o) => {
                    const avg = o.runs ? o.totalMs / o.runs : 0;
                    return (
                      <tr key={o.step} data-testid={`overview-row-${o.step}`}>
                        <td>{o.step}. {o.label}</td>
                        <td className={styles.num}>{o.runs}</td>
                        <td className={styles.num}>{formatDuration(avg)}</td>
                        <td className={styles.barcell}>
                          <div className={`${styles.hb} ${avg >= slowestAvg ? styles.hot : ""}`} style={{ width: `${Math.round((avg / slowestAvg) * 100)}%` }} />
                        </td>
                        <td className={styles.num}>{formatDuration(o.maxMs)}</td>
                        <td className={styles.num}>{o.tokens ? o.tokens.toLocaleString("vi-VN") : "—"}</td>
                        <td className={styles.num}>{o.failures}</td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          </Card>
        )}

        {projects.length > 0 && view === "project" && (
          <div className={styles.layout}>
            <Card title="Dự án" hint="Mới hoạt động nhất ở trên.">
              <div className={styles.projectList}>
                {projects.map((p) => (
                  <button
                    key={p.id}
                    type="button"
                    className={`${styles.projectBtn} ${p.id === activeId ? styles.projectBtnOn : ""}`}
                    onClick={() => setSelected(p.id)}
                    data-testid={`journal-project-${p.id}`}
                  >
                    <span className={styles.projectId}>{p.id.slice(0, 8)}</span>
                    <span className={styles.projectMeta}>
                      {p.last.step_label} · {STATE_LABEL[p.last.run_state]} · {p.count} sự kiện
                    </span>
                  </button>
                ))}
              </div>
            </Card>

            <Card title={`Dòng thời gian ${activeId ? activeId.slice(0, 8) : ""}`}>
              {perStep.length > 0 && (
                <div className={styles.summary} data-testid="journal-summary">
                  {perStep.map((s) => (
                    <span key={s.label}>
                      <b>{s.label}</b>: {formatDuration(s.ms)}, {s.tokens.toLocaleString("vi-VN")} token
                      {s.failed > 0 ? `, ${s.failed} lần lỗi` : ""}
                    </span>
                  ))}
                </div>
              )}
              <table className={styles.table}>
                <thead>
                  <tr>
                    <th>Thời điểm</th>
                    <th>Bước</th>
                    <th>Trạng thái</th>
                    <th className={styles.num}>Thời gian</th>
                    <th className={styles.num}>Ký tự</th>
                    <th className={styles.num}>Token</th>
                    <th>Ghi chú</th>
                  </tr>
                </thead>
                <tbody>
                  {timeline.map((e) => (
                    <tr key={e.id} data-testid="journal-row">
                      <td>{formatTime(e.at)}</td>
                      <td>{e.step_label}</td>
                      <td>
                        <span className={`${styles.state} ${styles[`state_${e.run_state}`] ?? ""}`}>
                          {STATE_LABEL[e.run_state]}
                        </span>
                      </td>
                      <td className={styles.num}>
                        {formatDuration(e.duration_ms)}
                        {e.source === "saga" && e.duration_ms ? " (ở bước trước)" : ""}
                      </td>
                      <td className={styles.num}>{e.content_chars ? e.content_chars.toLocaleString("vi-VN") : "—"}</td>
                      <td className={styles.num}>
                        {e.prompt_tokens || e.completion_tokens
                          ? ((e.prompt_tokens ?? 0) + (e.completion_tokens ?? 0)).toLocaleString("vi-VN")
                          : "—"}
                      </td>
                      <td className={e.run_state === "failed" ? styles.detail : undefined}>{e.detail || ""}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </Card>
          </div>
        )}
      </AppShell>
    </div>
  );
}
