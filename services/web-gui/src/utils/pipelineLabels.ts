/**
 * Vietnamese labels for the saga's step and status names.
 *
 * The progress tracker used to print the raw step id ("Đang xử lý:
 * assemble_video") while the video list already had its own Vietnamese
 * mapping. One table now serves both, so a new step is translated once.
 */

/** Saga step ids, as they arrive on the SSE progress stream. */
export const STEP_LABELS: Record<string, string> = {
  parse_script: "Đọc kịch bản",
  // CR-020: cổng kiểm tra trước khi tốn TTS — thay cho classify_scenes cũ (đã
  // gỡ khỏi saga, giữ nhãn dưới đây chỉ để hiển thị project cũ nếu còn sót).
  validate_script: "Kiểm tra kịch bản",
  classify_scenes: "Phân loại cảnh",
  synthesize_speech: "Tạo giọng đọc",
  render_scenes: "Render hoạt hình",
  assemble_video: "Ghép video hoàn chỉnh",
  // CR-021
  qc_video: "Chấm chất lượng video",
  // CR-007
  generate_clips: "Cắt clip dọc Shorts/TikTok",
  publish_video: "Đăng lên YouTube",
};

/** Project statuses, as stored on the project record. */
const STATUS_LABELS: Record<string, string> = {
  draft: "Nháp",
  parsing_script: "Đang đọc kịch bản",
  validating_script: "Đang kiểm tra kịch bản",
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

export function statusLabel(status: string): string {
  if (status.startsWith("failed_at_")) {
    const step = status.replace("failed_at_", "");
    return STEP_LABELS[step] ? `Thất bại — ${STEP_LABELS[step]}` : "Thất bại";
  }
  return STATUS_LABELS[status] ?? status;
}

/** The ordered pipeline, so the tracker can show how far along a render is. */
export const RENDER_STEPS = [
  "parse_script",
  "validate_script",
  "synthesize_speech",
  "render_scenes",
  "assemble_video",
  "qc_video",
  "generate_clips",
] as const;
