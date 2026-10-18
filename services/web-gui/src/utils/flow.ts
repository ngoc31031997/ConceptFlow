import type { AuthoringStep } from "../api/client";

/**
 * The 13-step production flow, as the Creator sees it. The server decides where
 * a project stands (`flow_step` + `run_state` on GET /v1/projects/:id, derived
 * by domain.FlowStateFor); this file only names the steps, says which screen
 * shows each, and whether the inputs behind a screen may still be edited.
 *
 * Review (8) is a screen, not a saved state: validate (7) finishes at
 * `awaiting_review` and nothing runs until the Creator presses start.
 *
 * "Hình minh hoạ" is its own numbered step at 5, BEFORE Code (6):
 * illustrations run first and Code reads what they drew.
 */
export const FLOW_LABELS = [
  "Ý tưởng",
  "Cấu hình",
  "Viết kịch bản",
  "Storyboard",
  "Hình minh hoạ",
  "Code",
  "Validate",
  "Review",
  "Giọng đọc",
  "Render từng cảnh",
  "Merge + QC",
  "Kết quả",
  "Publish",
] as const;

/**
 * What each step is for, one sentence: the "Bước này:" line under a step's
 * title, and the preview of a step not reached yet.
 */
export const FLOW_STEP_PURPOSE = [
  "Biến ý tưởng ban đầu thành một đề tài đủ cụ thể để phát triển: câu hỏi trung tâm, khán giả, điều người xem mang về. Chưa viết chuyện.",
  "Chọn kiểu video, format, giọng đọc và cách soạn (AI làm giúp hay tự làm) cho video này.",
  "Dựng dàn ý và lời thoại theo các beat của format, bám theo brief đã duyệt.",
  "Chia chuyện thành cảnh và shot: mỗi cảnh có lời đọc và những gì xuất hiện trên màn hình.",
  "Chuẩn bị hình cho từng shot: dùng lại hình trong thư viện, AI vẽ hình còn thiếu, bạn duyệt.",
  "Viết code Remotion dựng từng cảnh từ storyboard và hình; hệ thống tự kiểm code trước khi gửi đi.",
  "Máy kiểm code và luật trước khi render. Chưa tốn chi phí.",
  "Nhìn lại toàn bộ một lần: lời thoại, chất lượng, phụ đề, nhạc nền, rồi bắt đầu tạo video.",
  "Đọc giọng chính thức cho từng câu thoại.",
  "Render từng cảnh thành video.",
  "Ghép cảnh, giọng, phụ đề và nhạc; kiểm chất lượng.",
  "Xem, tải video, dựng lại với cấu hình khác hoặc làm bản Shorts.",
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
export const FLOW_RESULT = 12;

/** The flow step of each authoring step the AI can run. */
export const AUTHORING_STEP_FLOW: Record<AuthoringStep, number> = {
  story: 3,
  storyboard: 4,
  illustrations: 5,
  code: 6,
};
export const FLOW_PUBLISH = 13;

/** The theme colour that marks a phase (a token in theme.css). */
export type PhaseTone = "info" | "accent" | "run" | "accent2";

/** The four phases of the flow (see ADR-0042), in order. */
export const FLOW_VIEW_PHASES = [
  { name: "Tiền kỳ · Nội dung", tone: "info" },
  { name: "Tiền kỳ · Hình ảnh", tone: "accent" },
  { name: "Sản xuất", tone: "run" },
  { name: "Hậu kỳ", tone: "accent2" },
] as const satisfies readonly { name: string; tone: PhaseTone }[];

/**
 * One step of the 21-step flow the Creator sees (ADR-0042, the mockup). The
 * project still runs the 13 steps above; `server` lists which of them this
 * step shows, and is empty for a step not built yet. Render (17) shows both
 * the voice and the render, which run as one stretch after Review.
 */
export interface FlowViewStep {
  n: number;
  label: string;
  /** Screen title when it differs from the label ("Ý tưởng · creative brief"). */
  title?: string;
  /** Index in FLOW_VIEW_PHASES. */
  phase: 0 | 1 | 2 | 3;
  /** The flow waits here for the Creator to approve (◆). */
  gate?: boolean;
  /** The Creator may skip the step; it never holds the flow back. */
  optional?: boolean;
  server: readonly number[];
  /**
   * The step has a screen of its own that no running step stands behind (it
   * is not one of the 13 steps the server derives a project's position from).
   */
  screen?: "research";
  /** What the step holds, in a few words, for the flow map. */
  note: string;
}

export const FLOW_VIEW: readonly FlowViewStep[] = [
  { n: 1, phase: 0, label: "Ý tưởng", title: "Ý tưởng · creative brief", gate: true, server: [1], note: "Đề tài, câu hỏi trung tâm, khán giả, điều người xem mang về" },
  { n: 2, phase: 0, label: "Cấu hình", server: [2], note: "Kiểu video, format, giọng đọc, cách soạn" },
  { n: 3, phase: 0, label: "Nghiên cứu", title: "Nghiên cứu và kiểm chứng", gate: true, optional: true, server: [], screen: "research", note: "Câu hỏi cần tra, prompt chép ra ngoài, dán kết quả, kiểm link và đoạn trích" },
  { n: 4, phase: 0, label: "Câu chuyện", title: "Xây dựng câu chuyện", gate: true, server: [], note: "Góc tiếp cận, câu hỏi mở đầu, lời hứa, chuỗi nhân quả, payoff, dữ kiện dùng" },
  { n: 5, phase: 0, label: "Beat sheet", title: "Dàn ý · beat sheet", gate: true, server: [], note: "Beat, chức năng, mục tiêu, dữ kiện, thời lượng" },
  { n: 6, phase: 0, label: "Viết kịch bản", gate: true, server: [3], note: "Dàn ý và lời thoại theo beat" },
  { n: 7, phase: 0, label: "Rà soát và chốt", title: "Rà soát và chốt kịch bản", gate: true, server: [], note: "Rà soát, ghi chú bàn giao, chốt phiên bản" },
  { n: 8, phase: 1, label: "Mỹ thuật · lượt 1", title: "Thiết kế mỹ thuật · lượt 1", gate: true, server: [], note: "Diễn viên và phim trường dùng chung cả video" },
  { n: 9, phase: 1, label: "Storyboard", gate: true, server: [4], note: "Cảnh, shot, lời đọc, hình trên màn" },
  { n: 10, phase: 1, label: "Mỹ thuật · lượt 2", title: "Thiết kế mỹ thuật · lượt 2", gate: true, server: [], note: "Màu và ánh sáng từng phần của chuyện; khung mẫu" },
  { n: 11, phase: 1, label: "Giọng sơ bộ", server: [], note: "Giọng chính thức từng câu, tự chạy" },
  { n: 12, phase: 1, label: "Animatic", gate: true, server: [], note: "Khung tĩnh và giọng, chốt nhịp" },
  { n: 13, phase: 2, label: "Hình minh hoạ", server: [5], note: "Hình riêng của từng shot" },
  { n: 14, phase: 2, label: "Code", server: [6], note: "Code Remotion theo cảnh, tự kiểm lỗi" },
  { n: 15, phase: 2, label: "Validate", server: [7], note: "Kiểm code và luật" },
  { n: 16, phase: 2, label: "Review", gate: true, server: [8], note: "Nhìn lại, bắt đầu tạo video" },
  { n: 17, phase: 3, label: "Render từng cảnh", server: [9, 10], note: "Giọng đọc rồi render từng cảnh, tự chạy" },
  { n: 18, phase: 3, label: "Dựng", server: [], note: "Ghi chú theo mốc, làm lại một cảnh" },
  { n: 19, phase: 3, label: "Merge + QC", server: [11], note: "Ghép, âm lượng, QC" },
  { n: 20, phase: 3, label: "Kết quả", server: [12], note: "Xem, tải, làm Shorts" },
  { n: 21, phase: 3, label: "Publish", server: [13], note: "Đăng YouTube" },
];

/** The step of the 21-step flow that shows running step `step`, if any. */
export function viewStepOf(step: number): FlowViewStep | undefined {
  return FLOW_VIEW.find((v) => v.server.includes(step));
}

/** The number, in the 21-step flow, of the research step. */
export const VIEW_RESEARCH = 3;

/** Whether the app has a screen for the step: a running step behind it, or a screen of its own. */
export function isBuilt(view: FlowViewStep): boolean {
  return view.server.length > 0 || !!view.screen;
}

/** The screen that tells the Creator step `n` of the 21-step flow is not developed yet. */
export function comingRoute(n: number, projectId: string): string {
  return projectId ? `/projects/${projectId}/coming/${n}` : `/create/coming/${n}`;
}

/** The screen of a step of the 21-step flow that has one of its own (not a running step's). */
export function screenRoute(view: FlowViewStep, projectId: string): string {
  return `/projects/${projectId}/${view.screen}`;
}

/**
 * Where a step of a screen of its own stands for a project at running step
 * `flowStep`. The research step is open to the Creator from the config step on
 * and counts as passed once the project has moved on to the script.
 */
export function screenStatus(flowStep: number): StepStatus {
  if (flowStep < FLOW_CONFIG) return "pending";
  return flowStep === FLOW_CONFIG ? "waiting" : "done";
}

/** The phase that holds step `n` of the 21-step flow, or null for a number outside it. */
export function phaseOfView(n: number): FlowPhase | null {
  const view = FLOW_VIEW.find((v) => v.n === n);
  if (!view) return null;
  const phase = FLOW_VIEW_PHASES[view.phase];
  return { index: view.phase, name: phase.name, tone: phase.tone };
}

/**
 * "Bước 9 — Storyboard": a step named with its number in the 21-step flow,
 * for text that stands alone (the AI chain's progress, the video list). A
 * screen's own title is just the name (flowStepName): the line above it
 * already says "<nhóm> · Bước n/21". Both use the step rail's names.
 */
export function flowTitle(step: number): string {
  // A step outside the flow (a stale or hand-edited ?step=) keeps its number
  // rather than printing "undefined".
  if (!Number.isInteger(step)) return "Bước";
  const view = viewStepOf(step);
  return view ? `Bước ${view.n} — ${view.label}` : `Bước ${step}`;
}

/** "Storyboard": the name a screen of running step `step` carries as its title. */
export function flowStepName(step: number): string {
  const view = viewStepOf(step);
  return view ? (view.title ?? view.label) : `Bước ${step}`;
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
  // and review share a screen; so do TTS/render/merge).
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


/** Steps a worker runs with nothing for the Creator to do. */
export const AUTO_STEPS: ReadonlySet<number> = new Set([FLOW_VALIDATE, FLOW_TTS, FLOW_TTS + 1, FLOW_TTS + 2]);

/** A phase of the flow with its position (0-based) in FLOW_VIEW_PHASES. */
export interface FlowPhase {
  index: number;
  name: string;
  tone: PhaseTone;
}

/** The phase that holds running step `step`, or null for a step outside the flow. */
export function phaseOf(step: number): FlowPhase | null {
  const view = viewStepOf(step);
  return view ? phaseOfView(view.n) : null;
}

/** Hiển thị của một bước trong thanh bước / menu dọc. */
export type StepStatus = "done" | "waiting" | "running" | "failed" | "cancelled" | "pending";

/**
 * Trạng thái của bước `step` cho dự án ở (flowStep, runState). Bước trước bước
 * hiện tại là xong, bước hiện tại mang trạng thái chạy của nó, các bước sau
 * chưa tới.
 */
export function stepStatus(step: number, flowStep: number, runState: string | undefined): StepStatus {
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
 * The read-only preview of a step the project has not reached. Under /projects/:id when a project exists, so the
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
    return "Dự án đang chờ duyệt, chỉ có thể xem. Muốn sửa, chọn “Quay lại sửa script” ở bước Review.";
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
      return "Render sẽ dừng ngay. Phần đã dựng được giữ lại nên lần chạy tiếp sẽ nhanh hơn.";
    default:
      return "Merge + QC sẽ dừng ngay. Khi chạy tiếp, bước này bắt đầu lại từ đầu.";
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
