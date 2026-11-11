import { useEffect, useState } from "react";
import { listVideoFonts } from "../api/client";
import { SelectableOption } from "./SelectableOption";
import glass from "../styles/glass.module.css";
import selectable from "../styles/selectable.module.css";
import styles from "./VideoFontPicker.module.css";

interface VideoFontPickerProps {
  value: string;
  onChange: (font: string) => void;
}

type FontState =
  | { status: "loading" }
  | { status: "error" }
  | { status: "ready"; fonts: string[] };

/**
 * Font for text drawn inside a Remotion video — labels, numbers, titles.
 * Subtitles have their own font in the subtitle style panel. The choices come
 * from the rendering service, which owns the list of installed fonts.
 */
export function VideoFontPicker({ value, onChange }: VideoFontPickerProps) {
  const [state, setState] = useState<FontState>({ status: "loading" });

  useEffect(() => {
    let cancelled = false;
    listVideoFonts()
      .then((list) => {
        if (!cancelled) setState({ status: "ready", fonts: list.fonts });
      })
      .catch(() => {
        if (!cancelled) setState({ status: "error" });
      });
    return () => {
      cancelled = true;
    };
  }, []);

  const fonts = state.status === "ready" ? state.fonts : [];
  // A font saved on the video that rendering no longer offers stays visible.
  const orphan = state.status === "ready" && value && !fonts.includes(value) ? value : null;

  return (
    <div className={`${glass.card} ${styles.card}`} data-testid="video-font-picker">
      <div className={styles.text}>
        <div className={glass.cardTitle}>Font chữ trong video</div>
        <p className={styles.hint} style={{ fontFamily: `'${value}', sans-serif`, fontSize: 14 }}>
          Nhãn, con số, tiêu đề hiện trên hình — Aa Ăă Ơơ Ưư 0123
        </p>
        {state.status === "error" && (
          <p className={styles.hint} role="alert" data-testid="video-font-error">
            Không lấy được danh sách font từ rendering
          </p>
        )}
        {state.status === "loading" && (
          <p className={styles.hint} data-testid="video-font-loading">
            Đang tải danh sách font…
          </p>
        )}
      </div>
      <div className={selectable.row} role="radiogroup" aria-label="Font chữ trong video">
        {fonts.map((font) => (
          <SelectableOption
            key={font}
            selected={value === font}
            onSelect={() => onChange(font)}
            label={font}
            inline
            testId={`video-font-${font}`}
          />
        ))}
        {orphan && (
          <SelectableOption
            key={orphan}
            selected
            onSelect={() => onChange(orphan)}
            label={`${orphan} (không có trong rendering)`}
            inline
            testId={`video-font-${orphan}`}
          />
        )}
      </div>
    </div>
  );
}
