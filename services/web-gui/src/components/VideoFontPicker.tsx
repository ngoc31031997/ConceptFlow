import { SelectableOption } from "./SelectableOption";
import glass from "../styles/glass.module.css";
import selectable from "../styles/selectable.module.css";
import styles from "./RenderEnginePicker.module.css";

interface VideoFontPickerProps {
  value: string;
  onChange: (font: string) => void;
}

// Must match the fonts the rendering image installs
// (services/rendering/remotion_project/src/conceptflow-mini/primitives.tsx
// INSTALLED_FONTS); anything else renders as Be Vietnam Pro.
const OPTIONS: { value: string; label: string }[] = [
  { value: "Be Vietnam Pro", label: "Be Vietnam Pro" },
  { value: "Montserrat", label: "Montserrat" },
  { value: "Cormorant Garamond", label: "Cormorant Garamond" },
];

/**
 * Font for text drawn inside a Remotion video — labels, numbers, titles.
 * Subtitles have their own font in the subtitle style panel. Manim has no
 * equivalent: its fonts come from the design-system theme.
 */
export function VideoFontPicker({ value, onChange }: VideoFontPickerProps) {
  return (
    <div className={`${glass.card} ${styles.card}`} data-testid="video-font-picker">
      <div className={styles.text}>
        <div className={glass.cardTitle}>Font chữ trong video</div>
        <p className={styles.hint} style={{ fontFamily: `'${value}', sans-serif`, fontSize: 14 }}>
          Nhãn, con số, tiêu đề hiện trên hình — Aa Ăă Ơơ Ưư 0123
        </p>
      </div>
      <div className={selectable.row} role="radiogroup" aria-label="Font chữ trong video">
        {OPTIONS.map((option) => (
          <SelectableOption
            key={option.value}
            selected={value === option.value}
            onSelect={() => onChange(option.value)}
            label={option.label}
            inline
            testId={`video-font-${option.value}`}
          />
        ))}
      </div>
    </div>
  );
}
