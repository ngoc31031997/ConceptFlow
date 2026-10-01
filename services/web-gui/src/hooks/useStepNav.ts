import { useContext } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { ProjectDraftContext } from "../context/ProjectDraftContext";
import { useProjectFlow } from "../context/ProjectFlowContext";
import { useAuthoringRun } from "../context/AuthoringRunContext";
import { AUTHORING_STEP_FLOW, authoringRoute, flowRoute, previewRoute, stepStatus, type StepStatus } from "../utils/flow";

/** Bước bản nháp (chưa có project trên server) quay lại được, theo route. */
const DRAFT_ROUTES = ["/", "/create/script/settings", "/create/script/outline"];

export interface StepNav {
  /** Có project trên server để đọc vị trí hay không. */
  hasProject: boolean;
  /** Bước xa nhất đã tới (server) — hoặc bước đang mở, nếu lớn hơn. */
  reached: number;
  status: (step: number) => StepStatus;
  /** Mọi bước khác bước đang xem đều bấm được (bước chưa tới mở màn xem trước). */
  isClickable: (step: number) => boolean;
  /** Bước này mở màn thật (đã tới), không phải màn xem trước. */
  isReached: (step: number) => boolean;
  /** Mở bước `step` nếu nó bấm được từ màn này. */
  go: (step: number) => void;
  /** Mở màn của bước `step` (thật hoặc xem trước), kể cả khi đó là bước đang xem. */
  open: (step: number) => void;
}

export interface StepNavOptions {
  /**
   * Màn đang mở là màn xem trước của `currentStep`: bước đó chưa tới thật, nên
   * "đã tới" chỉ tính theo server, không theo bước đang xem.
   */
  preview?: boolean;
}

/**
 * Điều hướng theo 14 bước, dùng chung cho menu bước và thanh trạng thái: cùng
 * một luật "bước nào đã tới" và "bước này đang ở trạng thái gì", để hai chỗ
 * không bao giờ nói khác nhau. Bước đã tới mở màn thật; bước chưa tới hoặc
 * "Không dùng" mở màn xem trước (chỉ đọc).
 */
export function useStepNav(currentStep: number | undefined, options: StepNavOptions = {}): StepNav {
  const navigate = useNavigate();
  const draft = useContext(ProjectDraftContext);
  const flow = useProjectFlow();
  const routeProjectId = useParams().id;
  const projectId = routeProjectId || flow.projectId || draft.projectId;
  const hasProject = flow.project !== null;
  const reached = options.preview ? flow.flowStep : Math.max(currentStep ?? 0, flow.flowStep);
  const renderEngine = flow.project ? flow.project.render_engine : draft.renderEngine;
  // The server's flow step says nothing about an AI run on an authoring step
  // (the draft stays "idle"), so the step the AI works on right now comes
  // from the run in progress.
  const run = useAuthoringRun();
  const aiStep = run.running ? run.steps[run.currentIndex] : undefined;
  const aiFlowStep = aiStep ? AUTHORING_STEP_FLOW[aiStep] : 0;
  const status = (step: number): StepStatus =>
    step === aiFlowStep
      ? "running"
      : stepStatus(
      step,
      flow.flowStep,
      flow.runState,
      // Empty means "long" (a video with no vertical clips).
      flow.project ? flow.project.video_output_mode || "long" : undefined,
      renderEngine,
    );

  const isReached = (step: number) => {
    if (status(step) === "skipped") return false;
    if (hasProject) return step <= reached;
    // Chưa có project trên server: chỉ các bước nháp trước bước đang mở.
    const draftReached = options.preview ? 1 : (currentStep ?? 1);
    return step <= draftReached && step <= DRAFT_ROUTES.length;
  };

  const isClickable = (step: number) => !!currentStep && step !== currentStep;

  const open = (step: number) => {
    if (!isReached(step)) {
      navigate(previewRoute(step, hasProject ? projectId : ""));
      return;
    }
    if (!hasProject) {
      navigate(DRAFT_ROUTES[step - 1]);
      return;
    }
    // Bản nháp đang mở đúng project này và còn sửa được: đi thẳng, giữ những gì
    // đang gõ dở. Ngược lại (xem project khác / đã khoá) nạp lại từ server.
    if (step <= 6 && draft.projectId === projectId && flow.editable) {
      navigate(authoringRoute(step));
      return;
    }
    navigate(flowRoute(step, projectId, { view: step < flow.flowStep }));
  };

  const go = (step: number) => {
    if (isClickable(step)) open(step);
  };

  return {
    hasProject,
    reached,
    status,
    isClickable,
    isReached,
    go,
    open,
  };
}
