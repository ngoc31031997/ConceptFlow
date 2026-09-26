export function formatChars(n: number): string {
  return n >= 1000 ? `${(n / 1000).toFixed(1).replace(".", ",")}k` : String(n);
}

/** 125 → "2m05s", 42 → "42s": dạng gọn cho dòng phụ của thẻ tiến độ. */
export function formatClock(totalSeconds: number): string {
  const s = Math.max(0, Math.round(totalSeconds));
  return s < 60 ? `${s}s` : `${Math.floor(s / 60)}m${String(s % 60).padStart(2, "0")}s`;
}

const ERROR_LABELS: Record<string, string> = {
  balance: "Hết số dư tài khoản AI",
  budget: "Vượt ngân sách",
  auth: "Sai hoặc thiếu API key",
  rate_limit: "Bị giới hạn tốc độ",
  timeout: "Quá thời gian chờ",
  malformed: "AI trả kết quả không đọc được",
  empty: "AI trả về rỗng",
  server: "Lỗi phía máy chủ AI",
};

/** Lỗi đã phân loại của một lượt gọi → câu ngắn cho Creator (FR116.5). */
export function operationErrorLabel(kind: string): string {
  return ERROR_LABELS[kind] ? `${ERROR_LABELS[kind]} (${kind})` : kind;
}

interface SubtitleSource {
  phase: string;
  reasoning_chars: number;
  content_chars: number;
  elapsed_ms: number;
  done: number | null;
  total: number | null;
}

/**
 * Dòng phụ `pha · số lượng · thời gian` (FR116.1): "AI đang viết · 14,3k ký tự ·
 * 2m05s". Có `done/total` thì hiện "k/N" thay cho số ký tự.
 */
export function operationSubtitle(op: SubtitleSource | null, waitingLabel = "Đang chờ AI phản hồi"): string {
  if (!op) return waitingLabel;
  const time = formatClock(op.elapsed_ms / 1000);
  if (op.total != null && op.total > 0) return `${op.done ?? 0}/${op.total} · ${time}`;
  if (op.phase === "writing") return `AI đang viết · ${formatChars(op.content_chars)} ký tự · ${time}`;
  if (op.phase === "reasoning") return `AI đang suy luận · ${formatChars(op.reasoning_chars)} ký tự · ${time}`;
  return `${waitingLabel} · ${time}`;
}
