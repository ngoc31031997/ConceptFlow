import { useEffect, useState } from "react";
import { listProjectIllustrations } from "../api/client";
import { useAuthoringRun } from "../context/AuthoringRunContext";
import { isProjectIllustrationReady } from "../utils/illustrationLabels";
import { Button } from "./ui";
import styles from "./IllustrationsGateNote.module.css";

/**
 * On the Code tab of a Remotion video: where its drawings stand, and a
 * way to the illustrations step. Running Code with AI waits for every drawing
 * to be approved or skipped, so the Creator sees that here before pressing it.
 */
export function IllustrationsGateNote({ projectId, onOpen }: { projectId: string; onOpen: () => void }) {
  const [state, setState] = useState<{ total: number; ready: number } | null>(null);
  const run = useAuthoringRun();
  useEffect(() => {
    let cancelled = false;
    listProjectIllustrations(projectId)
      .then((r) => {
        if (!cancelled) setState({ total: r.illustrations.length, ready: r.illustrations.filter(isProjectIllustrationReady).length });
      })
      .catch(() => {
        /* the note is a hint; the code step's own gate still holds */
      });
    return () => {
      cancelled = true;
    };
  }, [projectId, run.running]);

  if (!state) return null;
  const done = state.total > 0 && state.ready === state.total;
  const text =
    state.total === 0
      ? "Hình minh hoạ: chưa có danh sách. Chạy bước Hình minh hoạ trước khi chạy Code bằng AI."
      : done
        ? `Hình minh hoạ: đủ ${state.total} hình đã duyệt hoặc bỏ qua.`
        : `Hình minh hoạ: ${state.ready}/${state.total} hình sẵn sàng — Code bằng AI sẽ chờ bạn duyệt hoặc bỏ qua phần còn lại.`;
  return (
    <div className={`${styles.note} ${done ? styles.ok : styles.waiting}`} data-testid="illustrations-gate-note">
      <span>{text}</span>
      <Button variant="ghost" onClick={onOpen} data-testid="illustrations-gate-open">
        Mở bước Hình minh hoạ
      </Button>
    </div>
  );
}
