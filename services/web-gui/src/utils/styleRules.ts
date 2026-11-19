import type { CodeDiagnostic } from "../api/client";

export interface StyleWarning {
  /** Mã luật, vd "S3"; rỗng nếu cảnh báo không mang mã. */
  code: string;
  line: number | null;
  /** Chi tiết bộ kiểm tra báo, đã bỏ phần "[S3] " ở đầu. */
  detail: string;
}

/** Tách mã luật "[S3]" ở đầu thông báo của bộ kiểm tra khỏi phần chi tiết. */
export function describeWarning(w: CodeDiagnostic): StyleWarning {
  const m = /^\[(S\d+)\]\s*([\s\S]*)$/.exec(w.message);
  const code = m ? m[1] : "";
  return { code, line: w.line ?? null, detail: m ? m[2] : w.message };
}

/** "S3 — dòng 14: nét thiếu strokeLinecap="round"" */
export function warningText(w: StyleWarning): string {
  const head = w.code;
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
