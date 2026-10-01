import { useEffect } from "react";
import { Navigate, useNavigate, useParams } from "react-router-dom";
import { AppShell } from "../components/AppShell";
import { Button, Card } from "../components/ui";
import { useProjectFlow } from "../context/ProjectFlowContext";
import { useStepNav } from "../hooks/useStepNav";
import {
  AUTO_STEPS,
  FLOW_LABELS,
  FLOW_STEP_PURPOSE,
  flowTitle,
  skippedReason,
} from "../utils/flow";
import styles from "./StepPreviewPage.module.css";

/**
 * Xem trước một bước video chưa tới, hoặc một bước "Không dùng": tên bước, bước
 * đó làm gì, và vì sao chưa mở được. Màn chỉ để đọc — không có ô nhập hay nút
 * hành động — nên không thể làm một bước khi các bước trước nó chưa xong.
 *
 * Mở từ menu bước (useStepNav). Khi project đã tới bước này (ví dụ mở lại một
 * đường dẫn cũ), màn tự chuyển sang màn thật của bước.
 */
export function StepPreviewPage() {
  const params = useParams<{ step: string }>();
  const step = Number(params.step);
  const valid = Number.isInteger(step) && step >= 1 && step <= FLOW_LABELS.length;
  const navigate = useNavigate();
  const flow = useProjectFlow();
  const nav = useStepNav(valid ? step : undefined, { preview: true });
  const status = valid ? nav.status(step) : "pending";
  const reached = valid && nav.hasProject && nav.isReached(step);

  useEffect(() => {
    if (reached) nav.open(step);
    // nav is rebuilt every render; reacting to `reached` is enough.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [reached, step]);

  if (!valid) return <Navigate to="/" replace />;

  const skipped = status === "skipped";
  const backToCurrent = () =>
    nav.hasProject && flow.flowStep ? nav.open(flow.flowStep) : navigate("/");

  return (
    <div data-testid="step-preview-page">
      <AppShell
        currentStep={step}
        preview
        title={flowTitle(step)}
        subtitle={FLOW_STEP_PURPOSE[step - 1]}
      >
        <p className={styles.notice} role="status" data-testid="step-preview-notice">
          {skipped ? (
            <>
              <b>Không dùng.</b> {skippedReason(step, flow.project?.video_output_mode)}
            </>
          ) : (
            <>
              <b>Chưa tới bước này.</b> Đây là xem trước; bước này mở khi video đi tới đây.
            </>
          )}
        </p>
        <Card title="Bước này làm gì">
          <p className={styles.purpose}>{FLOW_STEP_PURPOSE[step - 1]}</p>
          {AUTO_STEPS.has(step) && (
            <p className={styles.auto} data-testid="step-preview-auto">
              Bước này máy tự chạy, bạn không cần làm gì.
            </p>
          )}
        </Card>
        <div className={styles.actions}>
          <Button variant="ghost" onClick={backToCurrent} data-testid="step-preview-back">
            {nav.hasProject && flow.flowStep
              ? `Về bước đang làm: ${FLOW_LABELS[flow.flowStep - 1]}`
              : "Về bước Ý tưởng"}
          </Button>
        </div>
      </AppShell>
    </div>
  );
}
