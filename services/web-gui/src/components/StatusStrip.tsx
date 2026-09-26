import { useState } from "react";
import { useProjectFlow } from "../context/ProjectFlowContext";
import { useStepNav } from "../hooks/useStepNav";
import { cancelProject, retryProject, ApiError } from "../api/client";
import { FLOW_LABELS, FLOW_RESULT, FLOW_REVIEW, cancelExplain, isCancellableStep } from "../utils/flow";
import { ConfirmModal } from "./ConfirmModal";
import { ForkDialog } from "./ForkDialog";
import glass from "../styles/glass.module.css";
import styles from "./StatusStrip.module.css";

interface StatusStripProps {
  /** Bước màn này đang hiển thị — để biết người dùng đang xem ngoài bước đang chạy. */
  currentStep: number;
}

/**
 * Một dòng luôn trả lời "dự án đang làm gì", trên mọi màn của một project: chạy,
 * lỗi, đã hủy, chờ duyệt, xong. Đọc từ `run_state` của server nên không thể im
 * lặng khi server đã làm xong (hoặc đã dừng).
 *
 * Hành động ở đây chỉ là những gì không thuộc về màn đang mở: hủy bước đang
 * chạy (màn tiến độ không có nút đó), quay về bước đang chạy khi đang xem một
 * bước khác, chạy tiếp một bước đã hủy, và tạo bản mới khi đã có kết quả. Thử
 * lại sau lỗi vẫn ở ErrorBanner của màn sở hữu bước đó.
 */
export function StatusStrip({ currentStep }: StatusStripProps) {
  const flow = useProjectFlow();
  const nav = useStepNav(currentStep);
  const [confirming, setConfirming] = useState(false);
  const [forking, setForking] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  if (!flow.project) return null;
  const step = flow.flowStep;
  const state = flow.runState;
  const label = FLOW_LABELS[step - 1] ?? "";
  const elsewhere = currentStep !== step;

  async function run(action: () => Promise<unknown>) {
    setBusy(true);
    setError(null);
    try {
      await action();
      flow.refetch();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Không thực hiện được. Thử lại sau.");
    } finally {
      setBusy(false);
    }
  }

  const back = elsewhere && (
    <button type="button" className={glass.ghostBtn} onClick={() => nav.go(step)} data-testid="strip-goto">
      Về bước {label}
    </button>
  );

  let pill: { cls: string; text: string } | null = null;
  let message = "";
  let actions: React.ReactNode = null;

  if (state === "running") {
    pill = { cls: styles.run, text: "Đang chạy" };
    message = `${label} đang chạy. Bạn vẫn có thể xem các bước trước.`;
    actions = (
      <>
        {back}
        {isCancellableStep(step) && (
          <button type="button" className={glass.dangerGhostBtn} onClick={() => setConfirming(true)} disabled={busy} data-testid="strip-cancel">
            Hủy…
          </button>
        )}
      </>
    );
  } else if (state === "cancelled") {
    pill = { cls: styles.warn, text: "Đã hủy" };
    message = `${label} đã dừng theo yêu cầu của bạn. Các bước trước giữ nguyên.`;
    actions = (
      <>
        {back}
        <button type="button" className={glass.btnPrimary} onClick={() => run(() => retryProject(flow.projectId))} disabled={busy} data-testid="strip-resume">
          Chạy tiếp {label}
        </button>
      </>
    );
  } else if (state === "failed") {
    pill = { cls: styles.bad, text: "Lỗi" };
    message = `${label} dừng vì lỗi. Xem chi tiết ở dấu “?” của bước đó.`;
    actions = back;
  } else if (step === FLOW_REVIEW) {
    pill = { cls: styles.idle, text: "Chờ bạn" };
    message = "Kiểm tra xong. Duyệt để bắt đầu tạo video.";
    actions = back;
  } else if (step >= FLOW_RESULT) {
    pill = { cls: styles.ok, text: step === FLOW_RESULT ? "Xong" : state === "done" ? "Đã đăng" : "Xong" };
    message = "Video đã hoàn tất. Muốn thay đổi, hãy tạo bản mới.";
    actions = (
      <>
        {back}
        <button type="button" className={glass.ghostBtn} onClick={() => setForking(true)} data-testid="strip-fork">
          Tạo bản mới từ video này
        </button>
      </>
    );
  }

  if (!pill) return null;

  return (
    <>
      <div className={styles.strip} role="status" data-testid="status-strip" data-state={state}>
        <span className={`${styles.pill} ${pill.cls}`} data-testid="strip-pill">
          {pill.cls === styles.run && <span className={styles.dot} aria-hidden="true" />}
          {pill.text}
        </span>
        <span className={styles.message}>{error ?? message}</span>
        <span className={styles.actions}>{actions}</span>
      </div>

      <ConfirmModal
        isOpen={confirming}
        onClose={() => setConfirming(false)}
        onConfirm={() => run(() => cancelProject(flow.projectId))}
        title={`Hủy bước ${label}?`}
        message={
          <>
            {cancelExplain(step)} Dự án ở lại bước này với trạng thái “đã hủy”, bấm Chạy tiếp khi muốn làm tiếp.
          </>
        }
        confirmLabel={`Hủy ${label}`}
        cancelLabel="Tiếp tục chạy"
        isDangerous
      />
      {forking && <ForkDialog projectId={flow.projectId} onClose={() => setForking(false)} />}
    </>
  );
}
