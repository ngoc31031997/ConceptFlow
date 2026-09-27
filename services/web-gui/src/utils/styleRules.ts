import type { CodeDiagnostic } from "../api/client";

/**
 * CR-045 — tên các luật style, đọc thẳng từ file luật server trả về
 * (GET /v1/illustration-style), để cảnh báo "[S3] ..." hiện kèm tên luật mà
 * không phải chép danh sách luật sang một chỗ thứ hai.
 *
 * Dòng luật có dạng "- [S3] BO TRÒN: góc chữ nhật...": tên là phần trước dấu
 * ":" (hoặc trước " (", ", ", ". " nếu tới trước), chữ viết HOA để nhấn mạnh
 * được hạ về chữ thường, chữ đầu viết hoa.
 */
export function parseRuleNames(rules: string): Record<string, string> {
  const out: Record<string, string> = {};
  for (const m of rules.matchAll(/\[(S\d+)\]\s+([^\n]+)/g)) {
    out[m[1]] = ruleTitle(m[2]);
  }
  return out;
}

function ruleTitle(text: string): string {
  let end = text.length;
  for (const stop of [":", " (", ", ", ". ", " → "]) {
    const i = text.indexOf(stop);
    if (i > 0 && i < end) end = i;
  }
  const words = text
    .slice(0, end)
    .trim()
    .split(/\s+/)
    .map((w) => (w.length > 1 && w === w.toLocaleUpperCase("vi") && w !== w.toLocaleLowerCase("vi") ? w.toLocaleLowerCase("vi") : w));
  const title = words.join(" ");
  return title.charAt(0).toLocaleUpperCase("vi") + title.slice(1);
}

export interface StyleWarning {
  /** Mã luật, vd "S3"; rỗng nếu cảnh báo không mang mã. */
  code: string;
  /** Tên luật, vd "Bo tròn"; rỗng nếu chưa tải được file luật. */
  rule: string;
  line: number | null;
  /** Chi tiết bộ kiểm tra báo, đã bỏ phần "[S3] " ở đầu. */
  detail: string;
}

export function describeWarning(w: CodeDiagnostic, names: Record<string, string>): StyleWarning {
  const m = /^\[(S\d+)\]\s*([\s\S]*)$/.exec(w.message);
  const code = m ? m[1] : "";
  return { code, rule: names[code] ?? "", line: w.line ?? null, detail: m ? m[2] : w.message };
}

/** "S3 · Bo tròn — dòng 14: nét thiếu strokeLinecap="round"" */
export function warningText(w: StyleWarning): string {
  const head = [w.code, w.rule].filter(Boolean).join(" · ");
  const body = `${w.line != null ? `dòng ${w.line}: ` : ""}${w.detail}`;
  return head ? `${head} — ${body}` : body;
}

/** Ghi chú cho "Vẽ lại bằng AI": đủ để AI biết sửa gì mà không vẽ lại từ đầu. */
export function fixWarningsNote(warnings: StyleWarning[]): string {
  return [
    "Sửa các cảnh báo style sau, giữ nguyên ý hình, tên component và các prop:",
    ...warnings.map((w) => `- ${warningText(w)}`),
  ].join("\n");
}
