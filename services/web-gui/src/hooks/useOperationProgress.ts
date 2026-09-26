import { useEffect, useState } from "react";
import { getOperation, type OperationProgress } from "../api/client";

/**
 * Poll `GET /v1/operations/{id}` mỗi giây tới khi lượt gọi kết thúc. Best-effort:
 * lỗi mạng bỏ qua, không làm hỏng lượt gọi (FR116.5).
 */
export function useOperationProgress(operationId: string | null): OperationProgress | null {
  const [progress, setProgress] = useState<OperationProgress | null>(null);

  useEffect(() => {
    if (!operationId) {
      setProgress(null);
      return;
    }
    let cancelled = false;
    let timer: number | undefined;
    const tick = () => {
      Promise.resolve()
        .then(() => getOperation(operationId))
        .then((p) => {
          if (cancelled) return;
          setProgress(p);
          if (p.status === "running" || p.status === "pending") timer = window.setTimeout(tick, 1000);
        })
        .catch(() => {
          if (!cancelled) timer = window.setTimeout(tick, 1000);
        });
    };
    tick();
    return () => {
      cancelled = true;
      window.clearTimeout(timer);
    };
  }, [operationId]);

  return progress;
}
