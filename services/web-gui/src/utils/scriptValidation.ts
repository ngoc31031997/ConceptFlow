/**
 * Kiểm tra script ngay lúc soạn, trước khi gửi đi render.
 *
 * Sau CR-018 đây **không còn là bài toán đếm khớp**. Trước kia lời thoại là
 * comment `# NARRATION:` còn điểm chờ là `self.wait(AUTO)`, hai thứ tách rời mà
 * số lượng phải bằng nhau tuyệt đối — nên file này tồn tại chủ yếu để bắt lỗi
 * lệch đếm, thứ mà chính prompt mô tả là lỗi thường gặp nhất. `self.narrate()`
 * gộp hai thứ làm một, nên lớp lỗi đó biến mất theo cấu trúc và không còn gì
 * để đếm khớp.
 *
 * Việc còn lại của file này là **cho Creator thấy trước video sẽ dài bao nhiêu**
 * (CR-016 FR42). Con số đó vốn chỉ lộ ra ở bước 3 của Saga, sau khi TTS đã chạy.
 *
 * Lưu ý: phân tích tĩnh ở đây chỉ thấy các lời gọi `self.narrate("…")` viết
 * thẳng trong script. Lời thoại sinh ra trong vòng lặp hoặc trong hàm helper —
 * hoàn toàn hợp lệ sau CR-018 — không đếm được từ text. Vì vậy đây là **ước
 * lượng tối thiểu**, và chỉ lượt dry bên Rendering mới biết con số thật.
 */

import {
  type ContentLanguage,
  countWords,
  estimateNarrationDuration,
} from "./durationEstimate";

/** Khớp `self.narrate("...")` và `self.narrate('...')` trên một dòng. */
const NARRATE_RE = /self\.narrate\(\s*(["'])((?:(?!\1)[^\\]|\\.)*)\1\s*\)/g;
const SCENE_CLASS_RE = /^class\s+(\w+)\s*\([^)]*Scene[^)]*\)\s*:/;

/** Dấu hiệu script còn viết theo chuẩn trước CR-018. */
const LEGACY_NARRATION_RE = /^\s*#\s*NARRATION:/m;
const LEGACY_AUTO_WAIT_RE = /self\.wait\(\s*AUTO\s*\)/;

export interface NarrationEstimate {
  text: string;
  words: number;
  seconds: number;
}

export interface ScriptValidation {
  narrationCount: number;
  hasSceneClass: boolean;
  isValid: boolean;
  message: string | null;
  /** Từng đoạn lời thoại tìm thấy, kèm ước lượng thời lượng (FR42.4). */
  narrations: NarrationEstimate[];
  totalWords: number;
  /** Tổng thời lượng lời thoại. CHƯA gồm thời gian animation (FR42.3). */
  estimatedNarrationSeconds: number;
}

export function validateScript(
  script: string,
  language: ContentLanguage = "vi",
  wordsPerMinute?: number,
): ScriptValidation {
  const narrations = extractNarrations(script, language, wordsPerMinute);
  const totalWords = narrations.reduce((sum, n) => sum + n.words, 0);
  const estimatedNarrationSeconds = narrations.reduce((sum, n) => sum + n.seconds, 0);
  const hasSceneClass = script
    .split("\n")
    .some((line) => SCENE_CLASS_RE.test(line.trim()));

  const base = {
    narrationCount: narrations.length,
    hasSceneClass,
    narrations,
    totalWords,
    estimatedNarrationSeconds,
  };

  if (script.trim().length === 0) {
    return { ...base, isValid: true, message: null };
  }

  if (LEGACY_NARRATION_RE.test(script) || LEGACY_AUTO_WAIT_RE.test(script)) {
    return {
      ...base,
      isValid: false,
      message:
        'Script đang dùng chuẩn cũ (# NARRATION và self.wait(AUTO)). Chuẩn hiện tại là self.narrate("..."), và script phải bắt đầu bằng from conceptflow import *.',
    };
  }

  if (!hasSceneClass) {
    return {
      ...base,
      isValid: false,
      message: "Chưa tìm thấy class Scene (cần dạng class TenScene(ConceptFlowScene):).",
    };
  }

  return { ...base, isValid: true, message: null };
}

function extractNarrations(
  script: string,
  language: ContentLanguage,
  wordsPerMinute?: number,
): NarrationEstimate[] {
  const found: NarrationEstimate[] = [];
  // `matchAll` cần cờ /g và regex là hằng module, nên lastIndex phải được đặt
  // lại — nếu không, lần gọi thứ hai sẽ bắt đầu từ giữa chuỗi trước.
  NARRATE_RE.lastIndex = 0;
  for (const match of script.matchAll(NARRATE_RE)) {
    const text = match[2];
    if (text.trim().length === 0) continue;
    found.push({
      text,
      words: countWords(text),
      seconds: estimateNarrationDuration(text, language, wordsPerMinute),
    });
  }
  return found;
}
