/**
 * Kiểm tra code Remotion ngay lúc soạn, trước khi gửi đi render.
 *
 * Kết luận "hợp lệ" là của `rendering` (`validate_script`, `script_locator`,
 * `dry_run`). Ở đây chỉ bắt những lỗi lặp lại mà một biểu thức chính quy bắt
 * được an toàn, để Creator sửa ngay trong ô nhập thay vì đọc log lỗi render.
 */

/**
 * Toàn bộ nội dung được bọc trong một khối markdown ```tsx ... ``` (hoặc
 * ``` trơn). Mọi prompt trong thư viện prompt (authoring-service) đều bảo AI trả lời đúng một
 * khối như vậy — Creator copy nguyên cả khối, kể cả hai dòng backtick, là
 * đường dán phổ biến nhất từ một cửa sổ chat. Kết quả là code không còn bắt
 * đầu bằng `import` mà bằng dòng "```tsx", nên esbuild phía Rendering chết
 * ngay ở dòng 1 với lỗi cú pháp — một thông báo không nói cho Creator biết
 * vấn đề thật là hai dòng thừa ở đầu/cuối.
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
 * Creator xóa mất dòng ``` đóng, hoặc dán thêm chữ phía trước). Ô code ở bước
 * Kỹ sư tự gỡ khối trọn vẹn qua `stripMarkdownCodeFence`; kiểm tra này chỉ để bắt
 * phần còn sót và nói đúng vấn đề thay vì để lỗi cú pháp mơ hồ của esbuild
 * (dòng 1) là thứ đầu tiên Creator nhìn thấy.
 */
const LEADING_FENCE_RE = /^```/;

/**
 * The recurring failure modes hit in production (missing `export const
 * narrations`, `<Composition>` missing `id="creator"`/`calculateMetadata`, a
 * pasted markdown fence, code truncated mid-paste) otherwise surface only
 * minutes later at real render time inside esbuild, with a stack trace
 * instead of a plain sentence. validateRemotionScript catches what a
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

