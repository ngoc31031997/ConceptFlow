export function formatChars(n: number): string {
  return n >= 1000 ? `${(n / 1000).toFixed(1).replace(".", ",")}k` : String(n);
}

/** 125 → "2m 5s", 42 → "42s", 3725 → "1h 2m 5s": cùng dạng với màn Nhật ký. */
export function formatClock(totalSeconds: number): string {
  const s = Math.max(0, Math.round(totalSeconds));
  if (s < 60) return `${s}s`;
  const m = Math.floor(s / 60);
  if (m < 60) return `${m}m ${s % 60}s`;
  return `${Math.floor(m / 60)}h ${m % 60}m ${s % 60}s`;
}

const ERROR_LABELS: Record<string, string> = {
  balance: "Hết số dư tài khoản AI",
  budget: "Vượt ngân sách",
  auth: "Khóa API không hợp lệ",
  rate_limit: "Đã vượt giới hạn yêu cầu",
  timeout: "Quá thời gian chờ",
  malformed: "AI trả về kết quả không hợp lệ",
  empty: "AI trả về rỗng",
  server: "Dịch vụ AI đang gặp sự cố",
};

/** Lỗi đã phân loại của một lượt gọi → câu ngắn cho Creator. */
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
 * Dòng phụ `pha · số lượng · thời gian`: "AI đang viết · 14,3k ký tự ·
 * 2m05s". Có `done/total` thì hiện "k/N" thay cho số ký tự.
 */
export function operationSubtitle(op: SubtitleSource | null, waitingLabel = "Đang chờ AI phản hồi"): string {
  if (!op) return waitingLabel;
  const time = formatClock(op.elapsed_ms / 1000);
  if (op.total != null && op.total > 0) return `${op.done ?? 0}/${op.total} · ${time}`;
  if (op.phase === "writing") return `AI đang viết · ${formatChars(op.content_chars)} ký tự · ${time}`;
  if (op.phase === "reasoning") return `AI đang phân tích · ${formatChars(op.reasoning_chars)} ký tự · ${time}`;
  return `${waitingLabel} · ${time}`;
}
