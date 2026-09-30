import { useEffect, useState } from "react";
import { listVideoArchetypes, type VideoArchetype } from "../api/client";
import type { VideoFormat } from "../types";
import { Dropdown, type DropdownOption } from "./ui";
import styles from "./VideoArchetypePicker.module.css";

// "kiểu: B" ở cuối chủ đề — đúng cú pháp prompt Biên kịch đọc để ép kiểu.
const FORCE_RE = /\s*[—–-]?\s*kiểu\s*:\s*([^\s:—]+)\s*$/iu;

/** Mã kiểu đang ép trong chủ đề, "" nếu để AI tự chọn. */
export function forcedArchetypeCode(topic: string): string {
  return topic.match(FORCE_RE)?.[1]?.toUpperCase() ?? "";
}

/** Chủ đề với phần "kiểu: X" được thay bằng mã mới (hoặc bỏ đi khi code = ""). */
export function withForcedArchetype(topic: string, code: string): string {
  const base = topic.replace(FORCE_RE, "").trimEnd();
  if (!code) return base;
  return base ? `${base} — kiểu: ${code}` : `kiểu: ${code}`;
}

interface VideoArchetypePickerProps {
  topic: string;
  onTopicChange: (topic: string) => void;
  formats: VideoFormat[];
  formatId: string;
  onFormatChange: (formatId: string) => void;
}

/**
 * Chọn kiểu video ngay cạnh chủ đề. Server không lưu kiểu riêng: lựa
 * chọn được ghi thành "kiểu: X" ở cuối chủ đề, đúng cú pháp prompt Biên kịch
 * vốn đã hiểu, nên đường copy prompt tay và đường "Chạy bằng AI" đều thấy nó
 * mà không cần thêm cột nào. Gõ tay "kiểu: X" vào chủ đề vẫn được, và dropdown
 * sẽ tự hiện đúng lựa chọn.
 */
export function VideoArchetypePicker({ topic, onTopicChange, formats, formatId, onFormatChange }: VideoArchetypePickerProps) {
  const [archetypes, setArchetypes] = useState<VideoArchetype[]>([]);

  useEffect(() => {
    let cancelled = false;
    listVideoArchetypes()
      .then((rows) => {
        if (!cancelled) setArchetypes(rows);
      })
      .catch(() => {
        /* Không tải được thì ẩn bộ chọn — AI vẫn tự chọn kiểu như trước. */
      });
    return () => {
      cancelled = true;
    };
  }, []);

  if (archetypes.length === 0) return null;

  const code = forcedArchetypeCode(topic);
  const chosen = archetypes.find((a) => a.code.toUpperCase() === code) ?? null;
  const formatName = (id: string) => formats.find((f) => f.id === id)?.name;

  const options: DropdownOption[] = [
    { value: "", label: "Để AI tự chọn", hint: "AI đọc chủ đề rồi chọn kiểu hợp nhất." },
    ...archetypes.map((a) => ({
      value: a.code.toUpperCase(),
      label: `${a.code} — ${a.name}`,
      hint: a.when_to_use,
      badge: a.is_system ? undefined : "Của bạn",
    })),
  ];

  const recommended = chosen?.recommended_format_id ?? "";
  const recommendedName = recommended ? formatName(recommended) : undefined;
  const mismatch = recommendedName !== undefined && recommended !== formatId;

  return (
    <div className={styles.picker} data-testid="video-archetype-picker">
      <span className={styles.label}>Kiểu video</span>
      <Dropdown
        aria-label="Kiểu video"
        value={chosen ? code : ""}
        options={options}
        onChange={(next) => onTopicChange(withForcedArchetype(topic, next))}
        data-testid="video-archetype-select"
      />
      {code && !chosen && (
        <p className={styles.warn} data-testid="video-archetype-unknown">
          Không có kiểu "{code}" trong bảng Kiểu video — AI sẽ tự chọn.
        </p>
      )}
      {mismatch && (
        <div className={styles.suggest} data-testid="video-archetype-format-suggestion">
          <span>
            Kiểu này hợp nhất với format <strong>{recommendedName}</strong>.
          </span>
          <button
            type="button"
            className={styles.suggestBtn}
            onClick={() => onFormatChange(recommended)}
            data-testid="video-archetype-use-format"
          >
            Dùng format này
          </button>
        </div>
      )}
    </div>
  );
}
