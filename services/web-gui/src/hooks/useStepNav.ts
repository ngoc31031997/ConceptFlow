import { useContext } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { ProjectDraftContext } from "../context/ProjectDraftContext";
import { useProjectFlow } from "../context/ProjectFlowContext";
import { authoringRoute, flowRoute, stepStatus, type StepStatus } from "../utils/flow";

/** Bước bản nháp (chưa có project trên server) quay lại được, theo route. */
const DRAFT_ROUTES = ["/", "/create/script/settings", "/create/script/outline"];

export interface StepNav {
  /** Có project trên server để đọc vị trí hay không. */
  hasProject: boolean;
  /** Bước xa nhất đã tới (server) — hoặc bước đang mở, nếu lớn hơn. */
  reached: number;
  status: (step: number) => StepStatus;
  isClickable: (step: number) => boolean;
  go: (step: number) => void;
}

/**
 * Điều hướng theo 13 bước, dùng chung cho thanh bước ngang và menu dọc: cùng
 * một luật "bấm được bước nào" và "bước này đang ở trạng thái gì", để hai chỗ
 * không bao giờ nói khác nhau.
 */
export function useStepNav(currentStep: number | undefined): StepNav {
  const navigate = useNavigate();
  const draft = useContext(ProjectDraftContext);
  const flow = useProjectFlow();
  const routeProjectId = useParams().id;
  const projectId = routeProjectId || flow.projectId || draft.projectId;
  const hasProject = flow.project !== null;
  const reached = Math.max(currentStep ?? 0, flow.flowStep);

  const isClickable = (step: number) => {
    if (!currentStep || step === currentStep) return false;
    if (hasProject) return step <= reached;
    return step < currentStep && step <= DRAFT_ROUTES.length;
  };

  const go = (step: number) => {
    if (!isClickable(step)) return;
    if (!hasProject) {
      navigate(DRAFT_ROUTES[step - 1]);
      return;
    }
    // Bản nháp đang mở đúng project này và còn sửa được: đi thẳng, giữ những gì
    // đang gõ dở. Ngược lại (xem project khác / đã khoá) nạp lại từ server.
    if (step <= 5 && draft.projectId === projectId && flow.editable) {
      navigate(authoringRoute(step));
      return;
    }
    navigate(flowRoute(step, projectId, { view: step < flow.flowStep }));
  };

  return {
    hasProject,
    reached,
    status: (step) =>
      stepStatus(
        step,
        flow.flowStep,
        flow.runState,
        // Empty means "long" (a video with no vertical clips).
        flow.project ? flow.project.video_output_mode || "long" : undefined,
      ),
    isClickable,
    go,
  };
}
