import { useEffect, useMemo, useState } from "react";
import { AppShell } from "../components/AppShell";
import { Card } from "../components/ui";
import { listRecentEvents, listProjects, ApiError, type ProjectEvent } from "../api/client";
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
 *  (8) và Kết quả (13) là thời gian chờ người, không tính vào chi phí. */
const SAGA_WORK_STEPS = new Set([7, 9, 10, 11, 12]);

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
    // CR-046: the illustrations step is an authoring run too, now counted under its own
    // flow number (6) via e.flow_step; source "illustrations" still tells it apart from
    // "authoring" rows for historical events recorded before CR-046.
    if (e.source === "authoring" || e.source === "illustrations") {
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
  const [topics, setTopics] = useState<Record<string, string>>({});

  // Tên chủ đề để nhận ra dự án thay vì chỉ mã; lỗi thì vẫn hiện mã.
  useEffect(() => {
    let cancelled = false;
    listProjects()
      .then((rows) => {
        if (cancelled || !Array.isArray(rows)) return;
        setTopics(Object.fromEntries(rows.filter((r) => r.topic).map((r) => [r.project_id, r.topic as string])));
      })
      .catch(() => undefined);
    return () => {
      cancelled = true;
    };
  }, []);

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
  const maxMs = Math.max(1, ...overview.map((o) => o.maxMs));
  const slowestAvg = Math.max(1, ...overview.map((o) => (o.runs ? o.totalMs / o.runs : 0)));

  // Tổng thời gian và token theo bước: chỗ nào tốn nhất hiện ra ngay.
  const perStep = useMemo(() => {
    const by = new Map<string, { label: string; ms: number; tokens: number; failed: number }>();
    for (const e of timeline) {
      if (e.run_state === "running") continue; // dòng bắt đầu không có số đo
      // Saga: duration_ms là thời gian nằm ở from_status, tính cho bước của
      // trạng thái đó — nhưng dòng này ghi theo bước ĐÍCH nên chỉ cộng cho
      // authoring; thời gian saga hiện ở cột riêng trong bảng.
      if (e.source !== "authoring" && e.source !== "illustrations") continue;
      const label = e.source === "illustrations" ? "Hình minh hoạ" : e.step_label;
      const cur = by.get(label) ?? { label, ms: 0, tokens: 0, failed: 0 };
      cur.ms += e.duration_ms ?? 0;
      cur.tokens += (e.prompt_tokens ?? 0) + (e.completion_tokens ?? 0);
      if (e.run_state === "failed") cur.failed += 1;
      by.set(label, cur);
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
            <div className={styles.chart} data-testid="journal-overview">
              <div className={styles.legend}>
                <span><i className={styles.swAvg} /> Trung bình</span>
                <span><i className={styles.swMax} /> Lâu nhất</span>
              </div>
              {overview.map((o) => {
                const avg = o.runs ? o.totalMs / o.runs : 0;
                const pct = (ms: number) => `${Math.round((ms / maxMs) * 100)}%`;
                return (
                  <div key={o.step} className={styles.crow} data-testid={`overview-row-${o.step}`}>
                    <div className={styles.clabel}>{o.step}. {o.label}</div>
                    <div className={styles.ctrack}>
                      <div className={styles.cmax} style={{ width: pct(o.maxMs) }} />
                      <div className={`${styles.cavg} ${avg >= slowestAvg ? styles.hot : ""}`} style={{ width: pct(avg) }} />
                    </div>
                    <div className={styles.cval}>
                      <b>{formatDuration(avg)}</b>
                      <span>tối đa {formatDuration(o.maxMs)} · {o.runs} lượt</span>
                      <span>
                        {o.tokens ? `${o.tokens.toLocaleString("vi-VN")} token` : "— token"}
                        {o.failures > 0 && <em className={styles.fail}> · {o.failures} lỗi</em>}
                      </span>
                    </div>
                  </div>
                );
              })}
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
                    {topics[p.id] && <span className={styles.projectTopic}>{topics[p.id]}</span>}
                    <span className={styles.projectId}>{p.id.slice(0, 8)}</span>
                    <span className={styles.projectMeta}>
                      {p.last.step_label} · {STATE_LABEL[p.last.run_state]} · {p.count} sự kiện
                    </span>
                  </button>
                ))}
              </div>
            </Card>

            <Card
              title={`Dòng thời gian ${activeId ? (topics[activeId] ?? activeId.slice(0, 8)) : ""}`}
              hint={activeId && topics[activeId] ? activeId.slice(0, 8) : undefined}
            >
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
              <TimelineChart events={timeline} />
            </Card>
          </div>
        )}
      </AppShell>
    </div>
  );
}

interface Segment {
  event: ProjectEvent;
  start: number;
  end: number;
}

/**
 * Biểu đồ Gantt của một dự án: mỗi bước một hàng, mỗi sự kiện là một đoạn kéo
 * dài tới sự kiện kế tiếp (sự kiện cuối đang chạy thì kéo tới hiện tại). Màu
 * theo trạng thái; di chuột để xem chi tiết.
 */
function TimelineChart({ events }: { events: ProjectEvent[] }) {
  if (events.length === 0) return null;
  const times = events.map((e) => new Date(e.at).getTime());
  const last = events[events.length - 1];
  const t0 = times[0];
  const tEnd = Math.max(times[times.length - 1] + 1000, last.run_state === "running" ? Date.now() : 0);
  const span = Math.max(1, tEnd - t0);
  const segs: Segment[] = events.map((e, i) => ({
    event: e,
    start: times[i],
    end: i + 1 < events.length ? times[i + 1] : e.run_state === "running" ? tEnd : times[i],
  }));

  const rows: { label: string; segs: Segment[]; tokens: number }[] = [];
  for (const sg of segs) {
    let row = rows.find((r) => r.label === sg.event.step_label);
    if (!row) {
      row = { label: sg.event.step_label, segs: [], tokens: 0 };
      rows.push(row);
    }
    row.segs.push(sg);
    row.tokens += (sg.event.prompt_tokens ?? 0) + (sg.event.completion_tokens ?? 0);
  }

  const pct = (t: number) => `${((t - t0) / span) * 100}%`;
  const ticks = [0, 0.25, 0.5, 0.75, 1].map((f) => t0 + f * span);
  const failures = events.filter((e) => e.run_state === "failed" && e.detail);

  return (
    <div className={styles.gantt} data-testid="journal-timeline">
      <div className={styles.legend}>
        {(["running", "done", "idle", "failed"] as const).map((st) => (
          <span key={st}>
            <i className={`${styles.seg} ${styles[`seg_${st}`]}`} /> {STATE_LABEL[st]}
          </span>
        ))}
      </div>
      <div className={styles.grow}>
        <div />
        <div className={styles.axis}>
          {ticks.map((t, i) => (
            <span key={i} style={{ left: pct(t) }}>
              {new Date(t).toLocaleTimeString("vi-VN", { hour12: false, hour: "2-digit", minute: "2-digit" })}
            </span>
          ))}
        </div>
        <div />
      </div>
      {rows.map((r) => (
        <div key={r.label} className={styles.grow}>
          <div className={styles.clabel}>{r.label}</div>
          <div className={styles.gtrack}>
            {r.segs.map((sg) => {
              const e = sg.event;
              const tokens = (e.prompt_tokens ?? 0) + (e.completion_tokens ?? 0);
              const tip = [
                `${e.step_label} · ${STATE_LABEL[e.run_state]}`,
                formatTime(e.at),
                e.duration_ms ? `Thời gian: ${formatDuration(e.duration_ms)}${e.source === "saga" ? " (ở bước trước)" : ""}` : "",
                e.content_chars ? `Ký tự: ${e.content_chars.toLocaleString("vi-VN")}` : "",
                tokens ? `Token: ${tokens.toLocaleString("vi-VN")}` : "",
                e.detail ?? "",
              ].filter(Boolean).join("\n");
              return (
                <div
                  key={e.id}
                  data-testid="journal-row"
                  title={tip}
                  className={`${styles.seg} ${styles[`seg_${e.run_state}`] ?? ""}`}
                  style={{ left: pct(sg.start), width: `max(4px, ${((sg.end - sg.start) / span) * 100}%)` }}
                />
              );
            })}
          </div>
          <div className={styles.cval}>
            <b>{formatDuration(r.segs.reduce((a, sg) => a + (sg.end - sg.start), 0))}</b>
            <span>{r.tokens ? `${r.tokens.toLocaleString("vi-VN")} token` : "— token"}</span>
          </div>
        </div>
      ))}
      {failures.length > 0 && (
        <ul className={styles.failList}>
          {failures.map((e) => (
            <li key={e.id} className={styles.detail}>
              {formatTime(e.at)} · {e.step_label}: {e.detail}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
