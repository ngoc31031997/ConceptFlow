import type { SubtitleStyle } from "../context/ProjectDraftContext";
import glass from "../styles/glass.module.css";
import styles from "./SubtitleStylePanel.module.css";

interface SubtitleStylePanelProps {
  value: SubtitleStyle;
  onChange: (patch: Partial<SubtitleStyle>) => void;
}

const SAMPLE_TEXT = "Đây là phụ đề mẫu hiển thị trên video";

// Mirrors the ASS font sizes the Video Assembly Service burns in, scaled down
// to the preview box so what the Creator sees here matches the output.
const PREVIEW_FONT_SIZE: Record<SubtitleStyle["fontSize"], number> = {
  small: 10,
  medium: 13,
  large: 17,
};

const FONT_SIZES: { value: SubtitleStyle["fontSize"]; label: string }[] = [
  { value: "small", label: "Nhỏ" },
  { value: "medium", label: "Vừa" },
  { value: "large", label: "Lớn" },
];

const POSITIONS: { value: SubtitleStyle["position"]; label: string }[] = [
  { value: "bottom", label: "Dưới" },
  { value: "top", label: "Trên" },
];

export function SubtitleStylePanel({ value, onChange }: SubtitleStylePanelProps) {
  return (
    <div className={glass.card} style={{ padding: 22 }} data-testid="subtitle-style-panel">
      <div className={glass.cardTitle} style={{ marginBottom: 14 }}>
        Kiểu phụ đề
      </div>

      <div className={styles.preview} data-position={value.position} data-testid="subtitle-preview">
        <span
          className={styles.previewText}
          style={{
            color: value.textColor,
            fontSize: PREVIEW_FONT_SIZE[value.fontSize],
            backgroundColor: `rgba(0, 0, 0, ${value.backgroundOpacity})`,
          }}
        >
          {SAMPLE_TEXT}
        </span>
      </div>

      <div className={styles.field}>
        <span className={styles.fieldLabel}>Cỡ chữ</span>
        <div className={styles.segmented}>
          {FONT_SIZES.map((option) => (
            <button
              key={option.value}
              type="button"
              className={`${styles.segment} ${value.fontSize === option.value ? styles.active : ""}`}
              aria-pressed={value.fontSize === option.value}
              onClick={() => onChange({ fontSize: option.value })}
            >
              {option.label}
            </button>
          ))}
        </div>
      </div>

      <div className={styles.field}>
        <label className={styles.fieldLabel} htmlFor="subtitle-color">
          Màu chữ
        </label>
        <div className={styles.inlineRow}>
          <input
            id="subtitle-color"
            type="color"
            className={styles.colorInput}
            value={value.textColor}
            onChange={(event) => onChange({ textColor: event.target.value.toUpperCase() })}
          />
          <span className={styles.value}>{value.textColor}</span>
        </div>
      </div>

      <div className={styles.field}>
        <label className={styles.fieldLabel} htmlFor="subtitle-opacity">
          Độ mờ nền
        </label>
        <div className={styles.inlineRow}>
          <input
            id="subtitle-opacity"
            type="range"
            className={styles.slider}
            min={0}
            max={1}
            step={0.1}
            value={value.backgroundOpacity}
            onChange={(event) => onChange({ backgroundOpacity: Number(event.target.value) })}
          />
          <span className={styles.value}>{Math.round(value.backgroundOpacity * 100)}%</span>
        </div>
      </div>

      <div className={styles.field} style={{ marginBottom: 0 }}>
        <span className={styles.fieldLabel}>Vị trí</span>
        <div className={styles.segmented}>
          {POSITIONS.map((option) => (
            <button
              key={option.value}
              type="button"
              className={`${styles.segment} ${value.position === option.value ? styles.active : ""}`}
              aria-pressed={value.position === option.value}
              onClick={() => onChange({ position: option.value })}
            >
              {option.label}
            </button>
          ))}
        </div>
      </div>
    </div>
  );
}
