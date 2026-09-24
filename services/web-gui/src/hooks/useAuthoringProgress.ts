import { useEffect, useState } from "react";
import { getAuthoringProgress, type AuthoringProgress, type AuthoringStep } from "../api/client";

/**
 * Hỏi server mỗi giây xem lượt chạy AI của (project, step) đã tới đâu — pha
 * (chờ / suy luận / viết kết quả), số ký tự đã nhận, thời gian đã trôi. Chỉ
 * poll khi `step` khác null. Best-effort: lỗi mạng bỏ qua, thanh chạy chung
 * vẫn hiện.
 */
export function useAuthoringProgress(projectId: string, step: AuthoringStep | null): AuthoringProgress | null {
  const [progress, setProgress] = useState<AuthoringProgress | null>(null);

  useEffect(() => {
    if (!projectId || !step) {
      setProgress(null);
      return;
    }
    let cancelled = false;
    const tick = () => {
      getAuthoringProgress(projectId, step)
        .then((p) => {
          if (!cancelled) setProgress(p);
        })
        .catch(() => {
          /* best-effort */
        });
    };
    tick();
    const id = window.setInterval(tick, 1000);
    return () => {
      cancelled = true;
      window.clearInterval(id);
    };
  }, [projectId, step]);

  return progress;
}
