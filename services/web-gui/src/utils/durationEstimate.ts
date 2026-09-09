/**
 * Ước lượng thời lượng đọc một đoạn lời thoại (CR-016 FR42).
 *
 * **Đây là bản sao của `EstimateNarrationDuration` bên Go**
 * (`services/orchestrator/internal/domain/narration.go`). Hai bản phải cho cùng
 * kết quả trên cùng input, vì cùng một con số được dùng ở hai nơi khác nhau:
 * ở đây để Creator thấy trước lúc soạn, và ở Orchestrator để định thời lượng
 * thật khi tắt TTS (CR-001). Lệch nhau thì con số hiển thị lúc soạn khác con số
 * hệ thống thực sự dùng — loại sai lệch không ai truy ra được.
 *
 * `tests/fixtures/narration-duration-vectors.json` ở gốc repo là bộ dữ liệu
 * dùng chung khoá hai bản với nhau.
 */

export type ContentLanguage = "vi" | "en";

/**
 * Tiếng Việt đọc chậm hơn tiếng Anh một chút vì từ ngắn hơn và nhiều hơn cho
 * cùng một lượng nội dung.
 *
 * Đây là giá trị mặc định khi chưa hiệu chỉnh. Sau đủ số mẫu đo thật,
 * `voice_calibration` bên TTS cho ra WPM riêng cho từng giọng (FR43).
 */
export const WORDS_PER_MINUTE: Record<ContentLanguage, number> = {
  vi: 140,
  en: 150,
};

/** Giữ một câu rất ngắn trên màn hình đủ lâu để đọc kịp. */
export const MIN_NARRATION_SECONDS = 1.5;

export function countWords(text: string): number {
  const trimmed = text.trim();
  return trimmed.length === 0 ? 0 : trimmed.split(/\s+/).length;
}

/** Thời lượng ước tính của MỘT đoạn lời thoại, tính bằng giây. */
export function estimateNarrationDuration(
  text: string,
  language: ContentLanguage,
  wordsPerMinute?: number,
): number {
  const words = countWords(text);
  if (words === 0) return MIN_NARRATION_SECONDS;

  const wpm = wordsPerMinute ?? WORDS_PER_MINUTE[language] ?? WORDS_PER_MINUTE.en;
  const seconds = (words / wpm) * 60;
  return seconds < MIN_NARRATION_SECONDS ? MIN_NARRATION_SECONDS : seconds;
}

/** Định dạng giây thành "7 phút 20 giây" / "45 giây". */
export function formatDuration(seconds: number): string {
  const total = Math.round(seconds);
  const minutes = Math.floor(total / 60);
  const rest = total % 60;
  if (minutes === 0) return `${rest} giây`;
  if (rest === 0) return `${minutes} phút`;
  return `${minutes} phút ${rest} giây`;
}
