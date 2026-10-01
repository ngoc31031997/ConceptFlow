import { FLOW_VALIDATE } from "./flow";

/**
 * Vietnamese labels for the saga's step and status names.
 *
 * One table serves the progress tracker and the video list, so a new step is
 * translated once.
 */

/** Saga step ids, as they arrive on the SSE progress stream. */
export const STEP_LABELS: Record<string, string> = {
  // Hai bước này có nhãn riêng: chúng là toàn bộ nội dung của bước 7 (Kiểm tra tự động) — đó là màn hình
  // Creator ngồi đợi, nên biết đang phân tích hay đang chạy thử là khác biệt
  // thật: một cái tính bằng giây, một cái tính bằng phút.
  parse_script: "Phân tích kịch bản",
  validate_script: "Chạy thử & kiểm tra",
  classify_scenes: "Phân loại cảnh",
  synthesize_speech: "Tạo giọng đọc",
  // The rail's name — this is the only saga step of "Bước 10 — Dựng hình".
  render_scenes: "Dựng hình",
  assemble_video: "Ghép video hoàn chỉnh",
  // Không nằm trong luồng chính — nhãn giữ lại để hiển thị đúng cho project
  // đã chạy qua bước này.
  qc_video: "Chấm chất lượng video",
  generate_clips: "Cắt clip dọc Shorts/TikTok",
  publish_video: "Đăng lên YouTube",
};

/** Project statuses, as stored on the project record. */
const STATUS_LABELS: Record<string, string> = {
  draft: "Nháp",
  parsing_script: "Đang phân tích kịch bản",
  validating_script: "Đang chạy thử & kiểm tra",
  classifying_scenes: "Đang phân loại cảnh",
  awaiting_review: "Chờ duyệt dàn ý",
  synthesizing_speech: "Đang tổng hợp giọng đọc",
  rendering: "Đang render",
  assembling_video: "Đang ghép video",
  running_qc: "Đang chấm chất lượng video",
  generating_clips: "Đang cắt clip dọc",
  ready_to_publish: "Sẵn sàng đăng",
  publishing: "Đang đăng",
  published: "Đã đăng",
};

/** Falls back to the raw id so an unmapped step is still visible, not blank. */
export function stepLabel(step: string): string {
  return STEP_LABELS[step] ?? step;
}

/**
 * Maps a persisted project.status to the SSE step id it corresponds to.
 *
 * Bug report: a Creator who navigates away mid-render and comes back sees
 * "Đang khởi tạo..." no matter how far the saga actually got — useSSE's
 * state only fills in from the NEXT live progress.fanout message, and one
 * may not arrive for minutes (e.g. mid render_scenes). The render is still
 * running server-side the whole time; only the tracker looked stuck. This
 * lets the page seed the tracker from the project's own persisted status
 * (already fetched via useProject) until a live message replaces it.
 */
export function statusToStep(status: string): string | null {
  const map: Record<string, string> = {
    parsing_script: "parse_script",
    validating_script: "validate_script",
    // Cổng duyệt dừng NGAY SAU validate_script, nên ô sáng đúng là ô cuối của
    // bước 7 — không phải "chưa bắt đầu gì cả".
    awaiting_review: "validate_script",
    synthesizing_speech: "synthesize_speech",
    rendering: "render_scenes",
    assembling_video: "assemble_video",
    running_qc: "qc_video",
    generating_clips: "generate_clips",
  };
  return map[status] ?? null;
}

export function statusLabel(status: string): string {
  if (status.startsWith("failed_at_")) {
    const step = status.replace("failed_at_", "");
    return STEP_LABELS[step] ? `Thất bại — ${STEP_LABELS[step]}` : "Thất bại";
  }
  return STATUS_LABELS[status] ?? status;
}

/**
 * Bước 7 — "Kiểm tra tự động": phần rẻ của saga. Chạy xong hai việc này là đã biết
 * script có chạy được không và dàn ý ra sao, mà chưa tốn một giây TTS hay
 * render nào. Cổng duyệt dàn ý (bước 8 — Review) dừng đúng ở cuối danh sách này.
 */
export const VALIDATE_STEPS = ["parse_script", "validate_script"] as const;

/**
 * Hai việc con của bước 7 mang số con 7.1/7.2. Số trơn 1/2 từng làm
 * Creator đọc thành bước 1 và 2 của luồng.
 */
export const VALIDATE_SUBSTEP_NUMBERS: Record<(typeof VALIDATE_STEPS)[number], string> = {
  parse_script: `${FLOW_VALIDATE}.1`,
  validate_script: `${FLOW_VALIDATE}.2`,
};

/**
 * Bước 9–12 (TTS, Render, Merge, Cắt short): phần đắt, chỉ chạy sau khi
 * Creator duyệt ở bước 8. qc_video không có ở đây (không nằm trong luồng chính).
 */
export const PROCESS_STEPS = [
  "synthesize_speech",
  "render_scenes",
  "assemble_video",
  "generate_clips",
] as const;

/**
 * Màn hình nào đang sở hữu một project ở trạng thái này.
 *
 * Validate (7–8) và sản xuất (9–12) là hai màn riêng cùng theo dõi một saga, nên "project này
 * thuộc màn nào" phải trả lời được từ một chỗ duy nhất: nếu không, một Creator
 * mở lại bookmark cũ, hoặc bấm back sau khi duyệt, sẽ ngồi trên màn hình theo
 * dõi những bước đã chạy xong từ lâu mà không bao giờ thấy động tĩnh gì.
 */
export type ProjectPhase = "validate" | "process" | "result" | "publish";

export function projectPhase(status: string): ProjectPhase {
  const step = status.startsWith("failed_at_") ? status.replace("failed_at_", "") : null;
  if (step) {
    // Một bước hỏng thuộc về màn hình đang chạy nó — đó là nơi có nút thử lại
    // và câu giải thích đúng ngữ cảnh.
    // classify_scenes không còn trong saga, nhưng project cũ hỏng ở đó vẫn tồn
    // tại — và nó cũng là lỗi đầu vào, nên thuộc bước 7, nơi có đường quay về
    // sửa script.
    if ((VALIDATE_STEPS as readonly string[]).includes(step) || step === "classify_scenes") {
      return "validate";
    }
    if (step === "publish_video") return "publish";
    return "process";
  }
  switch (status) {
    case "draft":
    case "parsing_script":
    case "validating_script":
    case "classifying_scenes":
    case "awaiting_review":
      return "validate";
    case "ready_to_publish":
      return "result";
    case "publishing":
    case "published":
      return "publish";
    default:
      return "process";
  }
}

/** URL của màn hình sở hữu project ở trạng thái này. */
export function projectPath(projectId: string, status: string): string {
  // Draft chưa chạy saga: không có màn theo dõi nào, phải mở lại đúng bước wizard.
  if (status === "draft") return `/projects/${projectId}/resume`;
  const phase = projectPhase(status);
  if (phase === "validate") return `/projects/${projectId}/validate`;
  if (phase === "process") return `/projects/${projectId}/render`;
  // Bước 14 (đang đăng / đã đăng / đăng lỗi) có màn riêng.
  if (phase === "publish") return `/projects/${projectId}/publish`;
  return `/projects/${projectId}/result`;
}
