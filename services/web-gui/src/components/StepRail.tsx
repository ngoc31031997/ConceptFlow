import {
  AUTO_STEPS,
  FLOW_VIEW,
  FLOW_VIEW_PHASES,
  isBuilt,
  screenStatus,
  viewStepOf,
  type FlowViewStep,
  type StepStatus,
} from "../utils/flow";
import { useProjectFlow } from "../context/ProjectFlowContext";
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
};

const TONE_CLASS = {
  info: styles.toneInfo,
  accent: styles.toneAccent,
  run: styles.toneRun,
  accent2: styles.toneAccent2,
} as const;

function Mark({ status, n }: { status: StepStatus; n: number }) {
  if (status === "done") return <span aria-hidden="true">✓</span>;
  if (status === "running") return <span className={styles.dot} aria-hidden="true" />;
  if (status === "failed") return <span aria-hidden="true">!</span>;
  if (status === "cancelled") return <span aria-hidden="true">■</span>;
  return <span aria-hidden="true">{n}</span>;
}

interface StepRailProps {
  /** Bước chạy mà màn đang mở hiển thị; trống ở màn của một bước không có bước chạy (xem `viewStep`). */
  currentStep?: number;
  /** Số của bước trong 21 bước mà màn đang mở hiển thị, khi nó không có bước chạy phía sau. */
  viewStep?: number;
  /** Tên video hiển thị ở đầu menu ("Video mới" khi chưa có chủ đề). */
  title: string;
  /** Thu gọn / mở rộng do AppShell giữ, vì bề rộng của nó quyết định lề nội dung. */
  collapsed: boolean;
  onToggle: () => void;
  /** Màn đang mở là màn xem trước của `currentStep` (xem useStepNav). */
  preview?: boolean;
}

/**
 * The state of a step of the 21-step view from the running steps it shows:
 * done when all are done, otherwise the state of the first one not done.
 */
function viewStatus(view: FlowViewStep, status: (step: number) => StepStatus): StepStatus {
  const states = view.server.map(status);
  if (states.length === 0) return "pending";
  return states.find((s) => s !== "done") ?? "done";
}

/**
 * Menu dọc thứ hai, cạnh menu chính: "video này đang ở đâu trong luồng", 21
 * bước của luồng mục tiêu (ADR-0042), từ bước 1 (trước khi có project) tới lúc
 * đăng.
 *
 * Đầu menu là tên video và số bước đã xong. Bên dưới là bốn nhóm, mỗi nhóm có
 * ô màu riêng và số bước xong trên tổng; mọi bước luôn hiện. Bước đang xem tô
 * vàng, bước đã xong có dấu ✓, bước Creator duyệt mang dấu ◆. Bước app đã có
 * bấm được (bước đã tới mở màn thật, bước khác mở màn xem trước, xem
 * useStepNav); bước chưa làm mang nhãn "sắp có" và mở màn "chưa phát triển".
 * Bước có màn riêng (Nghiên cứu) không có bước chạy: trạng thái của nó suy từ
 * vị trí của project.
 */
export function StepRail({ currentStep, viewStep, title, collapsed, onToggle, preview }: StepRailProps) {
  const nav = useStepNav(currentStep, { preview, webScreen: !currentStep && !!viewStep });
  const flow = useProjectFlow();
  const viewing = viewStep ?? (currentStep ? viewStepOf(currentStep)?.n : undefined);
  const statusOf = (view: FlowViewStep): StepStatus =>
    view.screen ? (nav.hasProject ? screenStatus(view, flow.flowStep, flow.researchResolved) : "pending") : viewStatus(view, nav.status);
  const doneCount = FLOW_VIEW.filter((v) => isBuilt(v) && statusOf(v) === "done").length;

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
            <span className={styles.titleName}>{title}</span>
            <span className={styles.meta} data-testid="step-rail-meta">
              {nav.hasProject ? `${doneCount}/${FLOW_VIEW.length} bước xong` : "Nhập ý tưởng để bắt đầu"}
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
        {FLOW_VIEW_PHASES.map((phase, index) => {
          const steps = FLOW_VIEW.filter((v) => v.phase === index);
          const done = steps.filter((v) => statusOf(v) === "done").length;
          return (
            <section key={phase.name} className={styles.phase} data-testid={`rail-phase-${index + 1}`}>
              {!collapsed && (
                <div className={styles.phaseHead}>
                  <span className={styles.phaseName}>
                    <i className={`${styles.swatch} ${TONE_CLASS[phase.tone]}`} aria-hidden="true" />
                    {phase.name}
                  </span>
                  <span className={styles.phaseCount} data-testid={`rail-phase-count-${index + 1}`}>
                    {done}/{steps.length}
                  </span>
                </div>
              )}
              <ol className={styles.list}>
                {steps.map((view) => {
                  const coming = !isBuilt(view);
                  const ownScreen = !!view.screen;
                  const status = statusOf(view);
                  const isViewing = view.n === viewing;
                  const target = view.server[0];
                  const auto = !ownScreen && !coming && view.server.every((step) => AUTO_STEPS.has(step));
                  const hint = coming
                    ? `${view.n}. ${view.label} · sắp có, chưa phát triển`
                    : `${view.n}. ${view.label} · ${STATUS_TEXT[status]}${auto ? " · tự động" : ""}${view.optional ? " · tuỳ chọn" : ""}${view.gate ? " · bạn duyệt ở bước này" : ""}`;
                  // A step opens only once the one before it is done or skipped. A step
                  // with a screen of its own, and one not developed yet, open without a
                  // running step behind them; the first needs a project that got that far.
                  const disabled = coming
                    ? isViewing
                    : ownScreen
                      ? isViewing || !nav.hasProject || status === "pending"
                      : !nav.isClickable(target) || !nav.isReached(target);
                  const open = () => (coming || ownScreen ? nav.openView(view) : nav.go(target));
                  return (
                    <li key={view.n}>
                      <button
                        type="button"
                        className={[
                          styles.item,
                          styles[`s_${status}`],
                          isViewing ? styles.viewing : "",
                          coming ? styles.coming : "",
                        ]
                          .filter(Boolean)
                          .join(" ")}
                        disabled={disabled}
                        onClick={open}
                        aria-current={isViewing ? "step" : undefined}
                        title={hint}
                        data-testid={`rail-step-${view.n}`}
                        data-status={coming ? "coming" : status}
                        data-reached={ownScreen ? nav.hasProject : !coming && nav.isReached(target)}
                      >
                        <span className={styles.mark}>
                          <Mark status={status} n={view.n} />
                        </span>
                        {!collapsed && <span className={styles.label}>{view.label}</span>}
                        {!collapsed && coming && <span className={styles.comingTag}>sắp có</span>}
                        {!collapsed && view.optional && !coming && <span className={styles.comingTag}>tuỳ chọn</span>}
                        {!collapsed && !coming && view.gate && (
                          <span className={styles.gate} data-testid={`rail-gate-${view.n}`}>
                            ◆
                          </span>
                        )}
                      </button>
                    </li>
                  );
                })}
              </ol>
            </section>
          );
        })}
      </div>
    </nav>
  );
}
