import { useEffect, useState } from "react";
import { getLlmStatus, type LlmStatus } from "../api/client";

// Bước 1 có 4 tab (ScriptStepPage, ScriptOutlineStepPage,
// VisualDirectorStepPage, ManimEngineerStepPage) và mỗi tab tự mount hook
// này. Không cache, mỗi lần chuyển tab component remount, status lại rơi về
// null trong lúc chờ getLlmStatus() trả lời — mọi nơi đọc `llm?.enabled` (kể
// cả PipelineSettingsBar's dòng tóm tắt) đọc thấy "chưa biết" và coi như
// false, nên một project đã chọn "Gọi API trực tiếp" lại chớp qua nhãn "Copy
// prompt ra ngoài" mỗi khi đổi tab. Cache module-scope này giữ giá trị đã
// biết qua lần mount sau: hiện ngay giá trị cũ (không có bản build nào bật
// tắt LLM provider giữa chừng một phiên làm việc), vẫn fetch lại nền để tự
// sửa nếu có gì đổi thật.
let cachedStatus: LlmStatus | null = null;

/**
 * CR-027 FR79.4 — whether the "Gọi API trực tiếp" mode exists on this
 * deployment at all.
 *
 * `null` means "not known yet", which callers must treat as neither: drawing
 * the AI option and then taking it away a tick later is worse than waiting a
 * tick. A failed request counts as disabled — if the GUI cannot even ask, the
 * copy-out path is the honest thing to offer.
 */
export function useLlmStatus(): LlmStatus | null {
  const [status, setStatus] = useState<LlmStatus | null>(cachedStatus);

  useEffect(() => {
    let cancelled = false;
    getLlmStatus()
      .then((s) => {
        cachedStatus = s;
        if (!cancelled) setStatus(s);
      })
      .catch(() => {
        // Một request lỗi tạm thời không nên xóa cache đúng đã có (ngắt mạng
        // giữa hai tab không có nghĩa là LLM vừa bị tắt) — chỉ hạ về
        // "disabled" khi chưa từng biết gì.
        if (cancelled) return;
        const fallback = cachedStatus ?? { enabled: false, provider: "", reason: "" };
        setStatus(fallback);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  return status;
}
