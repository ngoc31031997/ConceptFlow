import { SelectableOption } from "./SelectableOption";
import { formatDuration } from "../utils/durationEstimate";
import type { VideoFormat } from "../types";
import selectable from "../styles/selectable.module.css";
import glass from "../styles/glass.module.css";

interface VideoFormatPickerProps {
  formats: VideoFormat[];
  value: string;
  onChange: (formatId: string) => void;
}

/**
 * Chọn hình dạng video cho project (CR-019 FR51.3).
 *
 * Bộ beat hiện ra ngay dưới mỗi lựa chọn chứ không giấu sau một nút: đó là phần
 * Creator thực sự cần biết khi viết script, và cũng là thứ hệ thống sẽ kiểm lại
 * ở bước validate — giấu nó đi thì cổng kiểm tra trở thành bất ngờ khó chịu.
 */
export function VideoFormatPicker({ formats, value, onChange }: VideoFormatPickerProps) {
  if (formats.length === 0) return null;

  return (
    <div className={glass.card} style={{ padding: 22 }} data-testid="video-format-picker">
      <div className={glass.cardTitle} style={{ marginBottom: 10 }}>
        Hình dạng video
      </div>
      <div className={selectable.stack} role="radiogroup" aria-label="Hình dạng video">
        {formats.map((format) => (
          <SelectableOption
            key={format.id}
            selected={format.id === value}
            onSelect={() => onChange(format.id)}
            label={format.name}
            hint={`${formatDuration(format.min_seconds)} – ${formatDuration(format.max_seconds)}`}
            testId={`video-format-${format.id}`}
          />
        ))}
      </div>

      {formats
        .filter((format) => format.id === value)
        .map((format) => (
          <p
            key={format.id}
            className={glass.helperText}
            style={{ marginRight: 0, marginTop: 10 }}
            data-testid="video-format-beats"
          >
            Cấu trúc:{" "}
            {format.beats
              .map((beat) => (beat.required ? `${beat.id}*` : beat.id))
              .join(" → ")}
            . Beat có dấu * là bắt buộc — thiếu thì hệ thống dừng trước khi tạo giọng đọc.
          </p>
        ))}
    </div>
  );
}
