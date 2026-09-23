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
 * rồi sang tab kế bị đọc đè lại giá trị cũ) nên cùng cách chặn: theo dõi lượt
 * lưu đang bay theo project, bỏ qua kết quả đọc khi một lượt lưu chưa xong.
 */
const pendingSaves = new Map<string, Promise<unknown>>();

export function useAuthoringModels(projectId: string): {
  models: AuthoringStepModels;
  setModels: (models: AuthoringStepModels) => void;
} {
  const draft = useContext(ProjectDraftContext);
  const dispatch = useContext(ProjectDraftDispatchContext);

  useEffect(() => {
    if (!projectId) return;
    let cancelled = false;
    getAuthoringState(projectId)
      .then((state) => {
        if (cancelled || pendingSaves.has(projectId)) return;
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
      const save = saveAuthoringModels(projectId, models).catch(() => {
        /* best-effort — see useAuthoringMode's docstring */
      });
      pendingSaves.set(projectId, save);
      void save.finally(() => {
        if (pendingSaves.get(projectId) === save) pendingSaves.delete(projectId);
      });
    },
  };
}
