import { useEffect } from "react";
import { useOperationProgress } from "../hooks/useOperationProgress";
import { OperationProgressCard } from "./OperationProgressCard";

/** How long to wait for the saga to report before giving up and letting the page re-check itself. */
const GIVE_UP_MS = 120_000;

interface DeleteProgressCardProps {
  projectId: string;
  /** Called once the delete saga has finished (or the card stopped hearing from it). */
  onDone: () => void;
}

/** FR116.2 — "Đang dọn · k/N service" while the delete saga runs. */
export function DeleteProgressCard({ projectId, onDone }: DeleteProgressCardProps) {
  const op = useOperationProgress(`delete:${projectId}`);
  const status = op?.status;

  useEffect(() => {
    if (status === "succeeded") onDone();
  }, [status, onDone]);

  useEffect(() => {
    if (status === "failed") return;
    const id = window.setTimeout(onDone, GIVE_UP_MS);
    return () => window.clearTimeout(id);
  }, [onDone, status]);

  return (
    <OperationProgressCard
      testId={`delete-progress-${projectId}`}
      title="Đang xóa"
      subtitle={`Đã xóa ${op?.done ?? 0}/${op?.total ?? "?"} mục`}
      done={op?.done}
      total={op?.total}
      error={status === "failed" ? `Không xóa được: ${op?.error ?? "lỗi không xác định"}. Bấm Xóa để thử lại.` : null}
    />
  );
}
