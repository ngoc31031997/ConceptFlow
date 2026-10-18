import type { PasteLog } from "../../api/client";

/** What a pasted answer gave, in one line: what was added, updated and left out. */
export function describePaste(log: PasteLog): string {
  const read = log.added.length + log.updated.length;
  if (read === 0 && log.rejected.length === 0) return "Không đọc được dữ kiện nào.";
  let text = read === 0 ? "Không đọc được dữ kiện nào" : `Đọc được ${read} dữ kiện`;
  const parts: string[] = [];
  if (log.added.length) parts.push(`thêm ${log.added.join(", ")}`);
  if (log.updated.length) parts.push(`cập nhật ${log.updated.join(", ")}`);
  if (parts.length) text += `: ${parts.join("; ")}`;
  text += ".";
  if (log.rejected.length) {
    const rows = log.rejected.map((r) => (r.row ? `dòng ${r.row} (${r.reason})` : r.reason));
    text += ` Bỏ ${log.rejected.length} dòng: ${rows.join("; ")}.`;
  }
  return text;
}

/** The time of day a paste was read, as "14:05"; empty when the time is unreadable. */
export function pasteTime(iso: string): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return "";
  return date.toLocaleTimeString("vi-VN", { hour: "2-digit", minute: "2-digit" });
}
