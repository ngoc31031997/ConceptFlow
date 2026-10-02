import { useEffect, useState, type ReactNode } from "react";
import { AUTO_STEPS, FLOW_LABELS, FLOW_PHASES, phaseOf, type StepStatus } from "../utils/flow";
import { useProjectFlow } from "../context/ProjectFlowContext";
import { usePresence } from "../hooks/usePresence";
import glass from "../styles/glass.module.css";
import { useStepNav } from "../hooks/useStepNav";
import styles from "./StepRail.module.css";

const COLLAPSE_KEY = "conceptflow.stepRail.collapsed";

/** Remembered per viewer; a broken or blocked storage just means "expanded". */
export function readRailCollapsed(): boolean {
  try {
    return window.localStorage.getItem(COLLAPSE_KEY) === "1";
  } catch {
    return false;
  }
}

export function writeRailCollapsed(collapsed: boolean): void {
  try {
    window.localStorage.setItem(COLLAPSE_KEY, collapsed ? "1" : "0");
  } catch {
    /* per-viewer convenience only */
  }
}

const STATUS_TEXT: Record<StepStatus, string> = {
  done: "Xong",
  waiting: "Đang chờ bạn",
  running: "Đang chạy",
  failed: "Lỗi",
  cancelled: "Đã hủy",
  pending: "Chưa tới",
  skipped: "Không dùng",
};

function Mark({ status, n }: { status: StepStatus; n: number }) {
  if (status === "done") return <span aria-hidden="true">✓</span>;
  if (status === "running") return <span className={styles.dot} aria-hidden="true" />;
  if (status === "failed") return <span aria-hidden="true">!</span>;
  if (status === "cancelled") return <span aria-hidden="true">■</span>;
  return <span aria-hidden="true">{n}</span>;
}

interface StepRailProps {
  currentStep: number;
  /** Tên video hiển thị ở đầu menu ("Video mới" khi chưa có chủ đề). */
  title: string;
  /** Thu gọn / mở rộng do AppShell giữ, vì bề rộng của nó quyết định lề nội dung. */
  collapsed: boolean;
  onToggle: () => void;
  /** Màn đang mở là màn xem trước của `currentStep` (xem useStepNav). */
  preview?: boolean;
}

/** A phase's step list, sliding open and closed with the shared reveal classes. */
function PhaseSteps({ open, children }: { open: boolean; children: ReactNode }) {
  const { mounted, closing } = usePresence(open);
  if (!mounted) return null;
  return <ol className={`${styles.list} ${closing ? glass.revealOut : glass.reveal}`}>{children}</ol>;
}

/** Index in FLOW_PHASES of the phase holding `step`; 0 for a step outside the flow. */
function phaseIndex(step: number): number {
  return Math.max(0, phaseOf(step)?.index ?? 0);
}

/**
 * Menu dọc thứ hai — lớp nằm cạnh menu chính (Tạo video / Danh sách / Nhật ký),
 * xếp chồng lên nó thay vì trộn vào cùng một danh sách. Menu chính nói "đang ở
 * đâu trong ứng dụng"; lớp này nói "video này đang ở đâu trong 13 bước", từ
 * bước 1 (trước khi có project) tới lúc đăng.
 *
 * Đầu menu là tên video, giai đoạn và bước hiện tại, số bước đã xong và thanh
 * tiến độ. Bên dưới là 5 giai đoạn thu gọn được: giai đoạn chứa bước đang xem
 * tự mở, Creator mở/đóng các giai đoạn khác. Mỗi giai đoạn ghi số bước đã xong
 * trên số bước dùng tới (bước "Không dùng" không tính). Bước máy tự chạy mang
 * nhãn "Tự động". Mọi bước đều bấm được: bước đã tới mở màn thật, bước khác mở
 * màn xem trước (useStepNav).
 */
export function StepRail({ currentStep, title, collapsed, onToggle, preview }: StepRailProps) {
  const nav = useStepNav(currentStep, { preview });
  const flow = useProjectFlow();
  const viewingPhase = phaseIndex(currentStep);
  const [open, setOpen] = useState<Set<number>>(() => new Set([viewingPhase]));
  // Moving to a step of another phase opens that phase too.
  useEffect(() => {
    setOpen((prev) => (prev.has(viewingPhase) ? prev : new Set(prev).add(viewingPhase)));
  }, [viewingPhase]);

  const togglePhase = (index: number) =>
    setOpen((prev) => {
      const next = new Set(prev);
      if (next.has(index)) next.delete(index);
      else next.add(index);
      return next;
    });

  const counted = FLOW_PHASES.flatMap((phase) => phase.steps.filter((step) => nav.status(step) !== "skipped"));
  const doneCount = counted.filter((step) => nav.status(step) === "done").length;
  const atStep = flow.flowStep || 1;
  const atPhase = phaseOf(atStep);

  return (
    <nav
      className={`${styles.rail} ${collapsed ? styles.railCollapsed : ""}`}
      aria-label="Các bước của video"
      data-testid="step-rail"
      data-collapsed={collapsed}
    >
      <div className={styles.head}>
        {!collapsed && (
          <div className={styles.title} title={title}>
            <span className={styles.titleLabel}>Video đang làm</span>
            <span className={styles.titleName}>{title}</span>
            <span className={styles.meta} data-testid="step-rail-meta">
              {nav.hasProject && atPhase
                ? `${atPhase.name} · ${FLOW_LABELS[atStep - 1]} · ${doneCount}/${counted.length} bước`
                : "Nhập ý tưởng để bắt đầu"}
            </span>
            <span className={styles.progress} aria-hidden="true">
              <span style={{ width: `${counted.length ? Math.round((doneCount / counted.length) * 100) : 0}%` }} />
            </span>
          </div>
        )}
        <button
          type="button"
          className={styles.toggle}
          onClick={onToggle}
          aria-label={collapsed ? "Mở rộng menu bước" : "Thu gọn menu bước"}
          aria-expanded={!collapsed}
          data-testid="step-rail-toggle"
        >
          {collapsed ? "›" : "‹"}
        </button>
      </div>

      <div className={styles.scroll}>
        {FLOW_PHASES.map((phase, index) => {
          const active = phase.steps.filter((step) => nav.status(step) !== "skipped");
          const done = active.filter((step) => nav.status(step) === "done").length;
          const phaseDone = active.length > 0 && done === active.length;
          // The narrow rail has no phase headers to open, so it lists every step.
          const expanded = collapsed || open.has(index);
          return (
            <section
              key={phase.name}
              className={`${styles.phase} ${index === viewingPhase ? styles.phaseViewing : ""}`}
              data-testid={`rail-phase-${index + 1}`}
              data-open={expanded}
            >
              {!collapsed && (
                <button
                  type="button"
                  className={`${styles.phaseHead} ${phaseDone ? styles.phaseDone : ""}`}
                  onClick={() => togglePhase(index)}
                  aria-expanded={expanded}
                  data-testid={`rail-phase-toggle-${index + 1}`}
                >
                  <span className={styles.phaseNo} aria-hidden="true">
                    {phaseDone ? "✓" : index + 1}
                  </span>
                  <span className={styles.phaseName}>{phase.name}</span>
                  <span className={styles.phaseCount} data-testid={`rail-phase-count-${index + 1}`}>
                    {done}/{active.length}
                  </span>
                  <span className={styles.chevron} aria-hidden="true">
                    ›
                  </span>
                </button>
              )}
              <PhaseSteps open={expanded}>
                  {phase.steps.map((step) => {
                    const status = nav.status(step);
                    const label = FLOW_LABELS[step - 1];
                    const viewing = step === currentStep;
                    const auto = AUTO_STEPS.has(step);
                    return (
                      <li key={step}>
                        <button
                          type="button"
                          className={[
                            styles.item,
                            styles[`s_${status}`],
                            viewing ? styles.viewing : "",
                            auto ? styles.auto : "",
                          ]
                            .filter(Boolean)
                            .join(" ")}
                          disabled={!nav.isClickable(step)}
                          onClick={() => nav.go(step)}
                          aria-current={viewing ? "step" : undefined}
                          title={collapsed ? `${step}. ${label}${auto ? " (tự động)" : ""} · ${STATUS_TEXT[status]}` : undefined}
                          data-testid={`rail-step-${step}`}
                          data-status={status}
                          data-reached={nav.isReached(step)}
                        >
                          <span className={styles.mark}>
                            <Mark status={status} n={step} />
                          </span>
                          {!collapsed && (
                            <span className={styles.text}>
                              <span className={styles.label}>
                                {label}
                                {auto && (
                                  <span className={styles.autoTag} data-testid={`rail-auto-${step}`}>
                                    Tự động
                                  </span>
                                )}
                              </span>
                              <span className={styles.state}>{STATUS_TEXT[status]}</span>
                            </span>
                          )}
                        </button>
                      </li>
                    );
                  })}
              </PhaseSteps>
            </section>
          );
        })}
      </div>
    </nav>
  );
}
