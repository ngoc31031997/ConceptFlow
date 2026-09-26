import { useContext, useEffect } from "react";
import { getAuthoringState, saveAuthoringModels, type AuthoringStepModels } from "../api/client";
import { ProjectDraftContext, ProjectDraftDispatchContext } from "../context/ProjectDraftContext";

/**
 * Model-per-step picker — model Hive cho từng tab (1a/1b/1c), chọn ở bước 1
 * khi Creator đang ở chế độ "Gọi API trực tiếp". Cùng một cách làm với
 * useAuthoringMode: đọc từ project ở server lúc mount, ghi lên server mỗi
 * lần đổi, giữ trong draft để mọi trang đọc cùng một giá trị.
 *
 * Cùng race condition useAuthoringMode từng gặp (bug report: chọn ở bước 1
 * rồi sang tab kế bị đọc đè lại giá trị cũ). Chặn bằng bộ đếm version: một
 * presence-check trên "lượt lưu đang bay" không đủ, vì PUT có thể resolve
 * (tự xóa khỏi map) trước khi GET đang đua — đọc dữ liệu cũ — resolve xong,
 * nên giá trị cũ vẫn thắng. Đếm version theo project: chỉ áp kết quả đọc khi
 * không có lượt setModels nào xảy ra kể từ lúc đọc bắt đầu.
 */
const writeVersion = new Map<string, number>();

export function useAuthoringModels(projectId: string): {
  models: AuthoringStepModels;
  setModels: (models: AuthoringStepModels) => void;
} {
  const draft = useContext(ProjectDraftContext);
  const dispatch = useContext(ProjectDraftDispatchContext);

  useEffect(() => {
    if (!projectId) return;
    let cancelled = false;
    const versionAtStart = writeVersion.get(projectId) ?? 0;
    getAuthoringState(projectId)
      .then((state) => {
        if (cancelled || (writeVersion.get(projectId) ?? 0) !== versionAtStart) return;
        dispatch({
          type: "SET_AUTHORING_MODELS",
          payload: {
            story: state.story_model ?? "",
            storyboard: state.storyboard_model ?? "",
            code: state.code_model ?? "",
          },
        });
      })
      .catch(() => {
        /* best-effort — the draft's own value still drives the UI */
      });
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [projectId]);

  return {
    models: draft.authoringModels,
    setModels: (models) => {
      dispatch({ type: "SET_AUTHORING_MODELS", payload: models });
      if (!projectId) return;
      writeVersion.set(projectId, (writeVersion.get(projectId) ?? 0) + 1);
      void saveAuthoringModels(projectId, models).catch(() => {
        /* best-effort — see useAuthoringMode's docstring */
      });
    },
  };
}
