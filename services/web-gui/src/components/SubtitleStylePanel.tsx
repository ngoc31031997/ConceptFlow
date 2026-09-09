import type { SubtitleStyle } from "../context/ProjectDraftContext";
import styles from "./SubtitleStylePanel.module.css";
import { SelectableOption } from "./SelectableOption";
import selectable from "../styles/selectable.module.css";

interface SubtitleStyleFieldsProps {
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

// The most common subtitle colors, so picking one is a click instead of a
// trip through the native color wheel every time.
const COLOR_PRESETS = ["#FFFFFF", "#FFE066", "#000000", "#FF4D4F", "#4DD4FF"];

/**
 * The subtitle style controls, without a card of their own: they now render
 * nested under the subtitle toggle in NarrationPanel rather than as a sibling
 * card that popped in and out of the sidebar.
 */
export function SubtitleStyleFields({ value, onChange }: SubtitleStyleFieldsProps) {
  return (
    <>
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

      <div className={styles.fieldsGrid}>
        <div className={styles.field}>
          <span className={styles.fieldLabel}>Cỡ chữ</span>
          <div className={selectable.row} role="radiogroup" aria-label="Cỡ chữ">
            {FONT_SIZES.map((option) => (
              <SelectableOption
                key={option.value}
                selected={value.fontSize === option.value}
                onSelect={() => onChange({ fontSize: option.value })}
                label={option.label}
                inline
                compact
              />
            ))}
          </div>
        </div>

        <div className={styles.field}>
          <span className={styles.fieldLabel}>Vị trí</span>
          <div className={selectable.row} role="radiogroup" aria-label="Vị trí">
            {POSITIONS.map((option) => (
              <SelectableOption
                key={option.value}
                selected={value.position === option.value}
                onSelect={() => onChange({ position: option.value })}
                label={option.label}
                inline
                compact
              />
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
            <div className={styles.colorPresets} role="group" aria-label="Màu có sẵn">
              {COLOR_PRESETS.map((color) => (
                <button
                  key={color}
                  type="button"
                  className={styles.colorSwatch}
                  style={{ background: color }}
                  data-active={value.textColor.toUpperCase() === color}
                  aria-label={color}
                  onClick={() => onChange({ textColor: color })}
                />
              ))}
            </div>
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
      </div>
    </>
  );
}
