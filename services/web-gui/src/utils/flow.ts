import type { AuthoringStep } from "../api/client";

/**
 * The 14-step production flow, as the Creator sees it. The server decides where
 * a project stands (`flow_step` + `run_state` on GET /v1/projects/:id, derived
 * by domain.FlowStateFor); this file only names the steps, says which screen
 * shows each, and whether the inputs behind a screen may still be edited.
 *
 * Review (8) is a screen, not a saved state: validate (7) finishes at
 * `awaiting_review` and nothing runs until the Creator presses start.
 *
 * "Hình minh hoạ" is its own numbered step, shown disabled ("Không dùng")
 * when renderEngine !== "remotion". It sits at step 5, BEFORE
 * Code (6): illustrations run first and Code reads what they drew.
 */
export const FLOW_LABELS = [
  "Ý tưởng",
  "Cấu hình",
  "Kịch bản",
  "Hình ảnh",
  "Hình minh hoạ",
  "Code",
  "Kiểm tra tự động",
  "Duyệt nội dung",
  "Giọng đọc",
  "Dựng hình",
  "Ghép video",
  "Cắt short",
  "Kết quả",
  "Đăng video",
] as const;

/** What each step does, one sentence, for the preview of a step not reached yet. */
export const FLOW_STEP_PURPOSE = [
  "Nhập ý tưởng hoặc chủ đề của video. Đây là chỗ duy nhất sửa chủ đề.",
  "Chọn ngôn ngữ, kiểu video, format, giọng đọc và cách soạn (AI làm giúp hay tự làm).",
  "Dựng dàn ý cho video từ chủ đề: câu hỏi cốt lõi và các beat theo format đã chọn.",
  "Dựng storyboard: mỗi cảnh có lời đọc và những gì xuất hiện trên màn hình.",
  "Chuẩn bị hình minh hoạ cho video Remotion: dùng lại hình trong thư viện, AI vẽ hình còn thiếu, bạn duyệt.",
  "Viết code dựng video từ storyboard; hệ thống tự kiểm tra code trước khi gửi đi.",
  "Hệ thống tự phân tích và chạy thử kịch bản. Chưa tốn chi phí.",
  "Đọc và sửa lời thoại, chọn chất lượng, phụ đề, nhạc nền, rồi duyệt để bắt đầu tạo video.",
  "Hệ thống tự tạo giọng đọc cho từng đoạn lời thoại.",
  "Hệ thống tự dựng hình từng cảnh.",
  "Hệ thống tự ghép hình, giọng đọc, phụ đề và nhạc nền thành video, rồi chấm chất lượng.",
  "Hệ thống tự cắt các clip dọc cho Shorts/TikTok từ video dài.",
  "Xem lại video đã xong, cắt thêm clip hoặc dựng lại với cấu hình khác.",
  "Kết nối kênh YouTube, điền tiêu đề, mô tả và đăng video.",
] as const;

export const FLOW_INIT = 1;
export const FLOW_CONFIG = 2;
export const FLOW_STORY = 3;
export const FLOW_VISUAL = 4;
export const FLOW_ILLUSTRATIONS = 5;
export const FLOW_CODE = 6;
export const FLOW_VALIDATE = 7;
export const FLOW_REVIEW = 8;
export const FLOW_TTS = 9;
export const FLOW_RESULT = 13;

/** The flow step of each authoring step the AI can run. */
export const AUTHORING_STEP_FLOW: Record<AuthoringStep, number> = {
  story: 3,
  storyboard: 4,
  illustrations: 5,
  code: 6,
};
export const FLOW_PUBLISH = 14;

/**
 * "Bước 4 — Hình ảnh": the one way a step is named to the Creator, on a
 * screen title, in the AI chain's progress, anywhere. The name is the step
 * rail's, so a screen can never call a step something the rail does not.
 */
export function flowTitle(step: number): string {
  // A step outside the flow (a stale or hand-edited ?step=) keeps its number
  // rather than printing "undefined".
  if (!Number.isInteger(step)) return "Bước";
  const label = FLOW_LABELS[step - 1];
  return label ? `Bước ${step} — ${label}` : `Bước ${step}`;
}

/**
 * The name of a logged event's step as the step rail shows it. The server's
 * `step_label` is the fallback for a flow step the web app does not know.
 */
export function eventStepLabel(event: { flow_step: number; step_label: string }): string {
  return FLOW_LABELS[event.flow_step - 1] ?? event.step_label;
}

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

/** Wizard route for authoring steps 1-6 (the screens that hold a draft). */
export function authoringRoute(step: number): string {
  if (step <= 1) return "/";
  if (step === 2) return "/create/script/settings";
  if (step === 3) return "/create/script/outline";
  if (step === 4) return "/create/script/storyboard";
  if (step === 5) return "/create/script/illustrations";
  return "/create/script/code";
}

/**
 * The five phases the Creator sees, grouping the 14 server steps by boundary of
 * cost and editability: preparing, writing content, reviewing, producing (runs
 * on its own) and the finished video.
 */
export const FLOW_PHASES = [
  { name: "Chuẩn bị", steps: [1, 2] },
  { name: "Soạn nội dung", steps: [3, 4, 5, 6] },
  { name: "Duyệt", steps: [7, 8] },
  { name: "Sản xuất", steps: [9, 10, 11, 12] },
  { name: "Hoàn tất", steps: [13, 14] },
] as const;

/** Steps a worker runs with nothing for the Creator to do. */
export const AUTO_STEPS: ReadonlySet<number> = new Set([FLOW_VALIDATE, FLOW_TTS, FLOW_TTS + 1, FLOW_TTS + 2, FLOW_TTS + 3]);

/** A phase of the flow with its position (0-based) in FLOW_PHASES. */
export interface FlowPhase {
  index: number;
  name: string;
  steps: readonly number[];
}

/** The phase that holds `step`, or null for a step outside the flow. */
export function phaseOf(step: number): FlowPhase | null {
  const index = FLOW_PHASES.findIndex((phase) => (phase.steps as readonly number[]).includes(step));
  if (index < 0) return null;
  return { index, name: FLOW_PHASES[index].name, steps: FLOW_PHASES[index].steps };
}

/** Hiển thị của một bước trong thanh bước / menu dọc. */
export type StepStatus = "done" | "waiting" | "running" | "failed" | "cancelled" | "pending" | "skipped";

/**
 * Trạng thái của bước `step` cho dự án ở (flowStep, runState). Bước trước bước
 * hiện tại là xong, bước hiện tại mang trạng thái chạy của nó, các bước sau
 * chưa tới. Bước cắt short bị bỏ qua khi dự án không làm video dọc. Bước Hình
 * minh hoạ bị bỏ qua ("Không dùng") khi dự án không dùng Remotion.
 */
export function stepStatus(
  step: number,
  flowStep: number,
  runState: string | undefined,
  outputMode?: string,
  renderEngine?: string,
): StepStatus {
  if (step === 12 && outputMode === "long") return "skipped";
  if (step === FLOW_ILLUSTRATIONS && renderEngine !== undefined && renderEngine !== "remotion") return "skipped";
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

/** Why a step marked "Không dùng" does not apply to this video. */
export function skippedReason(step: number): string {
  if (step === FLOW_ILLUSTRATIONS) return "Video dựng bằng Manim không có bước này; chỉ video Remotion mới có hình minh hoạ.";
  return "Video này chỉ làm bản dài, không cắt clip dọc.";
}

/**
 * The read-only preview of a step: for a step the project has not reached, or
 * one marked "Không dùng". Under /projects/:id when a project exists, so the
 * step menu keeps following that project.
 */
export function previewRoute(step: number, projectId: string): string {
  return projectId ? `/projects/${projectId}/preview/${step}` : `/create/preview/${step}`;
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
    return "Dự án đang chờ duyệt, chỉ có thể xem. Muốn sửa, chọn “Quay lại sửa script” ở bước Duyệt nội dung.";
  }
  if (status === "ready_to_publish" || status === "publishing" || status === "published") {
    return "Video đã hoàn tất, các bước trước chỉ để xem. Muốn thay đổi, hãy tạo bản mới.";
  }
  return "Dự án đang chạy, các bước trước chỉ để xem cho đến khi hoàn tất.";
}

/** Bước nào có worker để dừng (khớp runningStatusToStep của orchestrator). */
export function isCancellableStep(step: number): boolean {
  return step === FLOW_VALIDATE || step === FLOW_TTS || step === FLOW_TTS + 1 || step === FLOW_TTS + 2;
}

/** Điều gì được giữ / mất khi hủy bước này — nói thật, không hứa hơn hệ thống làm. */
export function cancelExplain(step: number): string {
  switch (step) {
    case FLOW_VALIDATE:
      return "Kiểm tra kịch bản sẽ dừng ngay, không mất chi phí.";
    case FLOW_TTS:
      return "Tạo giọng đọc sẽ dừng ngay. Khi chạy tiếp, bước này bắt đầu lại từ đầu.";
    case FLOW_TTS + 1:
      return "Dựng hình sẽ dừng ngay. Phần đã dựng được giữ lại nên lần chạy tiếp sẽ nhanh hơn.";
    default:
      return "Ghép video sẽ dừng ngay. Khi chạy tiếp, bước này bắt đầu lại từ đầu.";
  }
}

/**
 * Các bước có thể "làm lại từ đây" khi tạo bản mới (khớp ForkProjectUseCase).
 * Bước 5 (Hình minh hoạ) không vào danh sách này: ForkProjectUseCase chưa có
 * cách sao chép hình đã vẽ (file ảnh + trạng thái vẽ) sang dự án mới — làm
 * nửa vời (cho chọn rồi báo lỗi ở backend) còn tệ hơn không cho chọn. Dừng ở
 * bước 6 (Code).
 */
export const FORK_STEPS: { step: number; keeps: string }[] = [
  { step: 2, keeps: "Giữ chủ đề" },
  { step: 3, keeps: "Giữ chủ đề và cấu hình" },
  { step: 4, keeps: "Giữ chủ đề, cấu hình, kịch bản" },
  { step: 6, keeps: "Giữ chủ đề, cấu hình, kịch bản, visual" },
];
