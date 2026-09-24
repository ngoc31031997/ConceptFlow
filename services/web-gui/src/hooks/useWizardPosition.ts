import { useContext, useEffect } from "react";
import { saveWizardPosition } from "../api/client";
import { ProjectDraftContext } from "../context/ProjectDraftContext";

/**
 * Ghi lại màn wizard đang mở của project lên server mỗi lần vào màn, để mở lại
 * project (nút "Chi tiết") về đúng chỗ dừng — kể cả tab 1b/1c còn trống hay
 * bước 2 chưa bấm "Tiếp tục". Best-effort: không chặn giao diện nếu lỗi.
 */
export function useWizardPosition(route: string): void {
  const projectId = useContext(ProjectDraftContext).projectId;
  useEffect(() => {
    if (!projectId) return;
    void saveWizardPosition(projectId, route).catch(() => {
      /* best-effort */
    });
  }, [projectId, route]);
}
