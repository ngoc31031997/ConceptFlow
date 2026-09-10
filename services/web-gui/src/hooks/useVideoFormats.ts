import { useEffect, useState } from "react";
import { fetchVideoFormats } from "../api/client";
import type { VideoFormat } from "../types";

/**
 * Danh sách hình dạng video (CR-019 FR51.3).
 *
 * Lỗi mạng trả về danh sách rỗng thay vì ném ra: chọn format là một tiện ích
 * định hướng, không phải điều kiện để soạn được script. Hỏng thì Creator vẫn
 * làm việc bình thường, chỉ là không có gợi ý cấu trúc.
 */
export function useVideoFormats(): VideoFormat[] {
  const [formats, setFormats] = useState<VideoFormat[]>([]);

  useEffect(() => {
    let cancelled = false;
    fetchVideoFormats()
      .then((rows) => {
        if (!cancelled) setFormats(rows);
      })
      .catch(() => {
        /* xem docstring */
      });
    return () => {
      cancelled = true;
    };
  }, []);

  return formats;
}

/** Tổng ngân sách tối thiểu/tối đa của các beat, tính bằng giây. */
export function formatBudget(format: VideoFormat): { min: number; max: number } {
  return format.beats.reduce(
    (sum, beat) => {
      const repeats = Math.max(1, beat.max_repeat);
      return {
        min: sum.min + (beat.required ? beat.min_seconds : 0),
        max: sum.max + beat.max_seconds * repeats,
      };
    },
    { min: 0, max: 0 },
  );
}
