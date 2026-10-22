import { useEffect } from "react";
import { Navigate, useNavigate, useParams } from "react-router-dom";
import { AppShell } from "../components/AppShell";
import { Button, Card } from "../components/ui";
import { useProjectFlow } from "../context/ProjectFlowContext";
import { useStepNav } from "../hooks/useStepNav";
import {
  AUTO_STEPS,
  FLOW_CONFIG,
  FLOW_LABELS,
  FLOW_STEP_PURPOSE,
  flowStepName,
} from "../utils/flow";
import { CreativeBriefStepPage } from "./CreativeBriefStepPage";
import styles from "./StepPreviewPage.module.css";

/**
 * Xem trước một bước video chưa tới: tên bước, bước
 * đó làm gì, và vì sao chưa mở được. Bước Cấu hình có màn thật nên được hiện
 * nguyên màn đó ở chế độ chỉ đọc. Màn chỉ để đọc — không có ô nhập hay nút
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
  const reached = valid && nav.hasProject && nav.isReached(step);

  useEffect(() => {
    if (reached) nav.open(step);
    // nav is rebuilt every render; reacting to `reached` is enough.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [reached, step]);

  if (!valid) return <Navigate to="/" replace />;

  const backToCurrent = () =>
    nav.hasProject && flow.flowStep ? nav.open(flow.flowStep) : navigate("/");

  const backLabel =
    nav.hasProject && flow.flowStep
      ? `Về bước đang làm: ${FLOW_LABELS[flow.flowStep - 1]}`
      : "Về bước Ý tưởng";

  // The brief step has a real screen: show it read-only instead of the placeholder.
  if (step === FLOW_CONFIG) {
    return <CreativeBriefStepPage readOnly backLabel={backLabel} onBack={backToCurrent} />;
  }

  return (
    <div data-testid="step-preview-page">
      <AppShell
        currentStep={step}
        preview
        title={flowStepName(step)}
        subtitle={FLOW_STEP_PURPOSE[step - 1]}
      >
        <p className={styles.notice} role="status" data-testid="step-preview-notice">
          <b>Chưa tới bước này.</b> Đây là xem trước; bước này mở khi video đi tới đây.
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
            {backLabel}
          </Button>
        </div>
      </AppShell>
    </div>
  );
}
