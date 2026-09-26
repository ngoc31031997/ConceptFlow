import { FLOW_LABELS, FLOW_PHASES, type StepStatus } from "../utils/flow";
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
  /** Tên dự án hiển thị ở đầu menu. */
  title: string;
  /** Thu gọn / mở rộng do AppShell giữ, vì bề rộng của nó quyết định lề nội dung. */
  collapsed: boolean;
  onToggle: () => void;
}

/**
 * Menu dọc thứ hai — lớp nằm cạnh menu chính (Tạo video / Danh sách / Nhật ký),
 * xếp chồng lên nó thay vì trộn vào cùng một danh sách. Menu chính nói "đang ở
 * đâu trong ứng dụng"; lớp này nói "dự án này đang ở đâu trong 13 bước", nhóm
 * theo giai đoạn để thấy ranh giới sửa được / tốn tiền / đầu ra.
 *
 * Cùng luật bấm và cùng trạng thái với thanh bước ngang (useStepNav).
 */
export function StepRail({ currentStep, title, collapsed, onToggle }: StepRailProps) {
  const nav = useStepNav(currentStep);

  return (
    <nav
      className={`${styles.rail} ${collapsed ? styles.railCollapsed : ""}`}
      aria-label="Các bước của dự án"
      data-testid="step-rail"
      data-collapsed={collapsed}
    >
      <div className={styles.head}>
        {!collapsed && (
          <div className={styles.title} title={title}>
            <span className={styles.titleLabel}>Dự án</span>
            <span className={styles.titleName}>{title}</span>
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
        {FLOW_PHASES.map((phase) => (
          <section key={phase.name} className={styles.phase}>
            {!collapsed && <h2 className={styles.phaseName}>{phase.name}</h2>}
            <ol className={styles.list}>
              {phase.steps.map((step) => {
                const status = nav.status(step);
                const label = FLOW_LABELS[step - 1];
                const clickable = nav.isClickable(step);
                const viewing = step === currentStep;
                return (
                  <li key={step}>
                    <button
                      type="button"
                      className={[
                        styles.item,
                        styles[`s_${status}`],
                        viewing ? styles.viewing : "",
                      ]
                        .filter(Boolean)
                        .join(" ")}
                      disabled={!clickable}
                      onClick={() => nav.go(step)}
                      aria-current={viewing ? "step" : undefined}
                      title={collapsed ? `${step}. ${label} · ${STATUS_TEXT[status]}` : undefined}
                      data-testid={`rail-step-${step}`}
                      data-status={status}
                    >
                      <span className={styles.mark}>
                        <Mark status={status} n={step} />
                      </span>
                      {!collapsed && (
                        <span className={styles.text}>
                          <span className={styles.label}>{label}</span>
                          <span className={styles.state}>{STATUS_TEXT[status]}</span>
                        </span>
                      )}
                    </button>
                  </li>
                );
              })}
            </ol>
          </section>
        ))}
      </div>
    </nav>
  );
}
