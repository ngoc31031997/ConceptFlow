import { useState } from "react";
import type { OperationRun } from "../hooks/useOperationRun";
import { operationErrorLabel, operationSubtitle } from "../lib/formatProgress";
import { OperationProgressCard } from "./OperationProgressCard";
import { ImpactConfirm } from "./research/ImpactConfirm";

interface AiOperationCardProps {
  run: OperationRun;
  /** The request is still open. */
  active: boolean;
  testId: string;
  title?: string;
}

/**
 * OperationProgressCard wired to a useOperationRun: live while
 * the request is open, and — after a failure — kept on screen with the chars and
 * time already spent plus the classified error. While the request is open the
 * Creator can cancel it, after a confirmation: the request is aborted, the
 * server drops the model call and nothing is stored.
 */
export function AiOperationCard({ run, active, testId, title }: AiOperationCardProps) {
  const [confirming, setConfirming] = useState(false);
  const failed = run.progress?.status === "failed";
  if (!active && !failed) return null;
  return (
    <>
      <OperationProgressCard
        testId={testId}
        title={title}
        subtitle={operationSubtitle(run.progress)}
        done={run.progress?.done}
        total={run.progress?.total}
        error={failed ? operationErrorLabel(run.progress?.error ?? "server") : run.cancelError}
        onCancel={active ? () => setConfirming(true) : undefined}
      />
      <ImpactConfirm
        open={confirming}
        onClose={() => setConfirming(false)}
        title="Huỷ lượt chạy AI này?"
        list={[
          "AI dừng ngay, phần đang viết dở bị bỏ",
          "Dữ liệu hiện có giữ nguyên, không có gì được lưu từ lượt này",
          "Phần AI đã xử lý có thể vẫn tính vào mức dùng",
        ]}
        undo="Muốn có kết quả thì bấm chạy lại từ đầu."
        choices={[{ label: "Dừng AI", variant: "impact", onChoose: () => void run.cancel(), testId: `${testId}-cancel-confirm` }]}
        testId={`${testId}-cancel-dialog`}
      />
    </>
  );
}
