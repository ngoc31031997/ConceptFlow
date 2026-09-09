import { useEffect, useState } from "react";
import { fetchVoiceCalibration } from "../api/client";

/**
 * Tốc độ đọc **đo được** của từng giọng (CR-016 FR43.2).
 *
 * Hằng số words-per-minute trong `durationEstimate.ts` là một con số phỏng đoán
 * chưa từng được đối chiếu với gì. Orchestrator thì đo được nó miễn phí sau mỗi
 * lần tổng hợp giọng — nó biết cả văn bản đã gửi lẫn thời lượng audio thật trả
 * về. Hook này mang số đo đó về để ước lượng lúc soạn khớp với giọng Creator
 * thực sự dùng.
 *
 * Giọng chưa đủ mẫu **không** có mặt trong map: gọi `wordsPerMinuteFor` sẽ trả
 * `undefined` và bên ước lượng tự rơi về hằng số theo ngôn ngữ. Đó là câu trả
 * lời trung thực hơn một con số độ tin cậy thấp.
 *
 * Lỗi mạng được nuốt có chủ ý: đây là thứ làm ước lượng chính xác hơn, không
 * phải thứ ước lượng phụ thuộc vào. Hỏng thì Creator vẫn thấy con số, chỉ là
 * con số mặc định.
 */
export function useVoiceCalibration(): Record<string, number> {
  const [wpm, setWpm] = useState<Record<string, number>>({});

  useEffect(() => {
    let cancelled = false;
    fetchVoiceCalibration()
      .then((measured) => {
        if (!cancelled) setWpm(measured);
      })
      .catch(() => {
        /* xem docstring: ước lượng vẫn chạy bằng hằng số */
      });
    return () => {
      cancelled = true;
    };
  }, []);

  return wpm;
}

export function wordsPerMinuteFor(
  calibration: Record<string, number>,
  voiceId: string | null | undefined,
): number | undefined {
  if (!voiceId) return undefined;
  return calibration[voiceId];
}
