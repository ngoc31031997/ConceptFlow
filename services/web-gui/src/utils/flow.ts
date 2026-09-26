/**
 * The 13-step production flow, as the Creator sees it. The server decides where
 * a project stands (`flow_step` + `run_state` on GET /v1/projects/:id, derived
 * by domain.FlowStateFor); this file only names the steps, says which screen
 * shows each, and whether the inputs behind a screen may still be edited.
 *
 * Review (7) is a screen, not a saved state: validate (6) finishes at
 * `awaiting_review` and nothing runs until the Creator presses start.
 */
export const FLOW_LABELS = [
  "Khởi tạo",
  "Cấu hình",
  "Kịch bản",
  "Visual",
  "Code",
  "Validate",
  "Review",
  "TTS",
  "Render",
  "Merge",
  "Cắt short",
  "Kết quả",
  "Publish",
] as const;

export const FLOW_INIT = 1;
export const FLOW_CODE = 5;
export const FLOW_VALIDATE = 6;
export const FLOW_REVIEW = 7;
export const FLOW_TTS = 8;
export const FLOW_RESULT = 12;
export const FLOW_PUBLISH = 13;

export type RunState = "idle" | "running" | "failed" | "done";

/** Screen that shows a step of an existing project. `view` = opened only to look. */
export function flowRoute(step: number, projectId: string, opts: { view?: boolean } = {}): string {
  // `step` tells a view-only screen which of its steps was asked for (validate
  // and review share a screen; so do TTS/render/merge/split).
  const q = opts.view ? `?view=1&step=${step}` : "";
  if (step <= FLOW_CODE) return `/projects/${projectId}/resume?step=${step}&view=1`;
  if (step === FLOW_VALIDATE || step === FLOW_REVIEW) return `/projects/${projectId}/validate${q}`;
  if (step >= FLOW_TTS && step < FLOW_RESULT) return `/projects/${projectId}/render${q}`;
  if (step === FLOW_RESULT) return `/projects/${projectId}/result`;
  return `/projects/${projectId}/publish`;
}

/** Wizard route for authoring steps 1-5 (the screens that hold a draft). */
export function authoringRoute(step: number): string {
  if (step <= 1) return "/";
  if (step === 2) return "/create/script/settings";
  if (step === 3) return "/create/script/outline";
  if (step === 4) return "/create/script/storyboard";
  return "/create/script/code";
}

/** Giai đoạn của từng bước: nhóm theo ranh giới chi phí và khả năng sửa. */
export const FLOW_PHASES = [
  { name: "Soạn", steps: [1, 2, 3, 4, 5] },
  { name: "Kiểm tra", steps: [6, 7] },
  { name: "Sản xuất", steps: [8, 9, 10, 11] },
  { name: "Đầu ra", steps: [12, 13] },
] as const;

/** Hiển thị của một bước trong thanh bước / menu dọc. */
export type StepStatus = "done" | "waiting" | "running" | "failed" | "cancelled" | "pending" | "skipped";

/**
 * Trạng thái của bước `step` cho dự án ở (flowStep, runState). Bước trước bước
 * hiện tại là xong, bước hiện tại mang trạng thái chạy của nó, các bước sau
 * chưa tới. Bước cắt short bị bỏ qua khi dự án không làm video dọc.
 */
export function stepStatus(
  step: number,
  flowStep: number,
  runState: string | undefined,
  outputMode?: string,
): StepStatus {
  if (step === 11 && outputMode === "long") return "skipped";
  if (!flowStep || step > flowStep) return "pending";
  if (step < flowStep) return "done";
  switch (runState) {
    case "running":
      return "running";
    case "failed":
      return "failed";
    case "cancelled":
      return "cancelled";
    case "done":
      return "done";
    default:
      return "waiting";
  }
}

/**
 * Whether the authoring inputs (steps 1-5) of a project can still be changed.
 * Mirrors the server's own lock (domain.IsAuthoringEditable): a draft, or a
 * project that failed at some step. Anything else — validating, waiting at
 * review, rendering, done — is view-only, so the UI says so up front instead of
 * letting an edit through to a refusal.
 */
export function isAuthoringEditable(status: string | undefined): boolean {
  if (!status) return true; // not loaded yet / no project yet: nothing to lock
  return status === "draft" || status.startsWith("failed_at_");
}

/** Why the inputs are read-only, in words for the banner. */
export function readOnlyReason(status: string): string {
  if (status === "awaiting_review") {
    return "Dự án đang chờ duyệt dàn ý — chỉ xem. Muốn sửa, chọn “Từ chối / quay lại sửa” ở bước Review.";
  }
  if (status === "ready_to_publish" || status === "publishing" || status === "published") {
    return "Video đã render xong — các bước trước chỉ để xem. Muốn thay đổi, tạo bản mới từ video này.";
  }
  return "Dự án đang chạy — các bước trước chỉ để xem, không sửa được cho tới khi chạy xong.";
}

/** Bước nào có worker để dừng (khớp runningStatusToStep của orchestrator). */
export function isCancellableStep(step: number): boolean {
  return step === FLOW_VALIDATE || step === FLOW_TTS || step === FLOW_TTS + 1 || step === FLOW_TTS + 2;
}

/** Điều gì được giữ / mất khi huỷ bước này — nói thật, không hứa hơn hệ thống làm. */
export function cancelExplain(step: number): string {
  switch (step) {
    case FLOW_VALIDATE:
      return "Kiểm tra kịch bản dừng ngay. Chưa tốn giọng đọc hay render nên không mất gì.";
    case FLOW_TTS:
      return "Tạo giọng đọc dừng ngay. Khi chạy tiếp, bước này làm lại từ đầu.";
    case FLOW_TTS + 1:
      return "Render dừng ngay. Phần hình đã dựng được lưu đệm nên lần chạy tiếp nhanh hơn.";
    default:
      return "Ghép video dừng ngay, phần ghép dở bị bỏ. Khi chạy tiếp, bước này ghép lại từ đầu.";
  }
}

/** Các bước có thể "làm lại từ đây" khi tạo bản mới (khớp ForkProjectUseCase). */
export const FORK_STEPS: { step: number; keeps: string }[] = [
  { step: 2, keeps: "Giữ chủ đề" },
  { step: 3, keeps: "Giữ chủ đề và cấu hình" },
  { step: 4, keeps: "Giữ chủ đề, cấu hình, kịch bản" },
  { step: 5, keeps: "Giữ chủ đề, cấu hình, kịch bản, visual" },
];
