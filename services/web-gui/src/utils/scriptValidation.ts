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
 * CR-040 FR113.3: kết luận "hợp lệ" là của `rendering` (`validate_script`,
 * `script_locator`, `dry_run`). Những gì server đã bắt đúng — thiếu class Scene,
 * thiếu `<Composition id>` / `narrations` — đã bị xóa khỏi đây; chuẩn cũ
 * `# NARRATION` + `self.wait(AUTO)` đã bị gỡ khỏi hệ thống nên cũng không còn
 * được nhắc tới. Ở lại là những gì server không báo trước lúc render.
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

/**
 * Toàn bộ nội dung được bọc trong một khối markdown ```python ... ``` (hoặc
 * ``` trơn). Mọi prompt trong thư viện prompt (authoring-service) đều bảo AI trả lời đúng một
 * khối như vậy — Creator copy nguyên cả khối, kể cả hai dòng backtick, là
 * đường dán phổ biến nhất từ một cửa sổ chat. Kết quả là script không còn bắt
 * đầu bằng `from conceptflow import *` mà bằng dòng "```python", nên
 * `ast.parse` phía Rendering chết ngay ở dòng 1 với "invalid syntax" — một
 * thông báo không nói cho Creator biết vấn đề thật là hai dòng thừa ở đầu/cuối.
 */
const MARKDOWN_FENCE_RE = /^```[a-zA-Z0-9]*\r?\n([\s\S]*?)\r?\n?```\s*$/;

/**
 * Cùng một khối fence, nhưng KHÔNG neo đầu/cuối chuỗi. Bắt trường hợp AI
 * (đặc biệt Gemini/ChatGPT ở chế độ mặc định) phá lệnh "chỉ trả lời bằng đúng
 * một khối code" và vẫn thêm câu mở đầu ("Đây là script đã chỉnh sửa:") hay
 * lời chào cuối ("Nếu cần chỉnh gì thêm...") quanh khối — khi đó
 * MARKDOWN_FENCE_RE không khớp toàn chuỗi nên không gỡ được gì, và Creator
 * dán luôn cả câu văn lẫn hai dòng backtick vào script.
 */
const MARKDOWN_FENCE_ANYWHERE_RE = /```[a-zA-Z0-9]*\r?\n([\s\S]*?)\r?\n```/;

/** Gỡ khối markdown bọc ngoài nếu có; trả nguyên văn nếu không khớp. */
export function stripMarkdownCodeFence(script: string): string {
  const trimmed = script.trim();
  const fullMatch = MARKDOWN_FENCE_RE.exec(trimmed);
  if (fullMatch) return fullMatch[1];

  const anywhereMatch = MARKDOWN_FENCE_ANYWHERE_RE.exec(trimmed);
  if (anywhereMatch) return anywhereMatch[1];

  return script;
}

/**
 * Dấu hiệu còn sót dòng backtick mở đầu dù không khớp trọn khối (ví dụ
 * Creator xóa mất dòng ``` đóng, hoặc dán thêm chữ phía trước). ScriptEditor
 * tự gỡ khối trọn vẹn qua `stripMarkdownCodeFence`; kiểm tra này chỉ để bắt
 * phần còn sót và nói đúng vấn đề thay vì để lỗi cú pháp Python mơ hồ ở
 * `ast.parse` (dòng 1: invalid syntax) là thứ đầu tiên Creator nhìn thấy.
 */
const LEADING_FENCE_RE = /^```/;

export interface NarrationEstimate {
  text: string;
  words: number;
  seconds: number;
}

export interface ScriptValidation {
  narrationCount: number;
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

  const base = {
    narrationCount: narrations.length,
    narrations,
    totalWords,
    estimatedNarrationSeconds,
  };

  if (script.trim().length === 0) {
    return { ...base, isValid: true, message: null };
  }

  if (LEADING_FENCE_RE.test(script.trim())) {
    return {
      ...base,
      isValid: false,
      message:
        'Script còn dính dòng markdown ``` ở đầu (thường sót lại khi copy nguyên khối code từ AI). Xóa dòng ``` (và dòng ``` đóng ở cuối nếu có) — script phải bắt đầu ngay bằng from conceptflow import *.',
    };
  }

  return { ...base, isValid: true, message: null };
}

/**
 * feature/remotion-engine — Remotion had NO client-side check at all (any
 * non-empty text was accepted), so every one of the recurring failure modes
 * we've actually hit in production (missing `export const narrations`,
 * `<Composition>` missing `id="creator"`/`calculateMetadata`, a pasted
 * markdown fence, code truncated mid-paste) only ever surfaced minutes later
 * at real render time inside esbuild, with a stack trace instead of a plain
 * sentence. This mirrors validateScript's role for Manim: catch what a
 * regex safely can BEFORE the round trip to render, so a Creator fixes it in
 * the textarea instead of an error log. It intentionally does NOT try to
 * catch everything the prompt's self-check asks for (e.g. two overlapping
 * full-frame elements) — that needs real JSX layout, not text matching, and
 * a wrong flag there would be worse than not checking it at all.
 */
// Anchored on `export const narrations` — the type annotation is optional
// (mirrors remotion_renderer.py's dry_run(), which doesn't require it
// either) — so a `// narrations = [...]` left in a comment, or an unrelated
// `const myNarrations = [...]`, can never be picked up in place of the real
// array.
const REMOTION_NARRATIONS_HEADER_RE = /export\s+const\s+narrations\s*(?::\s*string\s*\[\s*\]\s*)?=\s*\[/;
const REMOTION_STRING_LITERAL_RE = /(["'`])(?:(?!\1)[^\\]|\\.)*\1/g;
const REMOTION_CALCULATE_METADATA_RE = /calculateMetadata\s*=\s*\{\s*calculateMetadataFromSegments\s*\}/;
const REMOTION_SEGMENTS_USAGE_RE = /<Segments\b/;
const REMOTION_IMPORT_RE = /import\s*\{[^}]*\bregisterRoot\b[^}]*\}\s*from\s*['"]remotion['"]/;

export interface RemotionScriptValidation {
  narrationCount: number;
  isValid: boolean;
  message: string | null;
}

function countChar(text: string, char: string): number {
  let count = 0;
  for (const c of text) if (c === char) count += 1;
  return count;
}

/** Removes line and block comments, respecting string literals, so a
 * comment can never be mistaken for the real narrations array. */
function stripComments(script: string): string {
  let out = "";
  let inString: string | null = null;
  let i = 0;
  while (i < script.length) {
    const ch = script[i];
    if (inString) {
      out += ch;
      if (ch === "\\" && i + 1 < script.length) {
        out += script[i + 1];
        i += 2;
        continue;
      }
      if (ch === inString) inString = null;
      i += 1;
      continue;
    }
    if (ch === "'" || ch === '"' || ch === "`") {
      inString = ch;
      out += ch;
      i += 1;
      continue;
    }
    if (ch === "/" && script[i + 1] === "/") {
      while (i < script.length && script[i] !== "\n") i += 1;
      continue;
    }
    if (ch === "/" && script[i + 1] === "*") {
      i += 2;
      while (i + 1 < script.length && !(script[i] === "*" && script[i + 1] === "/")) i += 1;
      i += 2;
      continue;
    }
    out += ch;
    i += 1;
  }
  return out;
}

/** Finds the `export const narrations = [...]` array and returns its body —
 * the raw text between the brackets — or null if absent. Scans
 * bracket/string-aware instead of a `[^\]]*` regex, which would cut the
 * array short at the first `]` even when it's inside a narration string
 * (e.g. `"Xem mục [1] nhé."`). */
function extractNarrationsBody(script: string): string | null {
  const stripped = stripComments(script);
  const header = REMOTION_NARRATIONS_HEADER_RE.exec(stripped);
  if (!header) return null;

  let i = header.index + header[0].length;
  const start = i;
  let depth = 1;
  let inString: string | null = null;
  while (i < stripped.length && depth > 0) {
    const ch = stripped[i];
    if (inString) {
      if (ch === "\\" && i + 1 < stripped.length) {
        i += 2;
        continue;
      }
      if (ch === inString) inString = null;
      i += 1;
      continue;
    }
    if (ch === "'" || ch === '"' || ch === "`") {
      inString = ch;
    } else if (ch === "[") {
      depth += 1;
    } else if (ch === "]") {
      depth -= 1;
    }
    i += 1;
  }
  return stripped.slice(start, i - 1);
}

function extractRemotionNarrationCount(script: string): number {
  const body = extractNarrationsBody(script);
  if (body === null) return 0;
  const literals = body.match(REMOTION_STRING_LITERAL_RE);
  return literals ? literals.length : 0;
}

export function validateRemotionScript(script: string): RemotionScriptValidation {
  const narrationCount = extractRemotionNarrationCount(script);
  const base = { narrationCount };

  if (script.trim().length === 0) {
    return { ...base, isValid: true, message: null };
  }

  if (LEADING_FENCE_RE.test(script.trim())) {
    return {
      ...base,
      isValid: false,
      message:
        "Code còn dính dòng markdown ``` ở đầu (thường sót lại khi copy nguyên khối code từ AI). Xóa dòng ``` (và dòng ``` đóng ở cuối nếu có).",
    };
  }

  if (!REMOTION_CALCULATE_METADATA_RE.test(script)) {
    return {
      ...base,
      isValid: false,
      message: "`<Composition>` thiếu `calculateMetadata={calculateMetadataFromSegments}` — thời lượng video sẽ tính sai.",
    };
  }

  if (!REMOTION_SEGMENTS_USAGE_RE.test(script)) {
    return {
      ...base,
      isValid: false,
      message: "Không thấy `<Segments>` — component chính phải dùng nó để hiển thị đúng hình theo từng đoạn lời thoại.",
    };
  }

  if (!REMOTION_IMPORT_RE.test(script)) {
    return {
      ...base,
      isValid: false,
      message: "Thiếu `import {registerRoot, Composition} from 'remotion';` ở đầu file.",
    };
  }

  const braceBalance = countChar(script, "{") - countChar(script, "}");
  const parenBalance = countChar(script, "(") - countChar(script, ")");
  if (braceBalance !== 0 || parenBalance !== 0) {
    return {
      ...base,
      isValid: false,
      message: "Dấu ngoặc { } hoặc ( ) không cân — code có thể đã bị cắt cụt lúc dán, thử copy và dán lại toàn bộ.",
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
