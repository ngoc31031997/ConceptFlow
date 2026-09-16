import type { RenderEngine } from "../context/ProjectDraftContext";
import { SelectableOption } from "./SelectableOption";
import glass from "../styles/glass.module.css";
import selectable from "../styles/selectable.module.css";
import styles from "./RenderEnginePicker.module.css";

interface RenderEnginePickerProps {
  value: RenderEngine;
  onChange: (engine: RenderEngine) => void;
}

/**
 * feature/remotion-engine — which engine renders the pasted script. Manim
 * stays the default and the only engine with a design system, lint, and
 * overlap-detection so far; Remotion is a minimal first cut (plain centered
 * text primitives, script pasted directly, no wizard prompts yet).
 *
 * Shaped like ContentLanguagePicker (compact inline row, not a stack of
 * hint-heavy cards) — both live side by side atop Step 1, so a matching
 * shape reads as one pair of quick settings instead of two differently-sized
 * widgets fighting for the same row.
 */
const OPTIONS: { value: RenderEngine; label: string }[] = [
  { value: "manim", label: "Manim" },
  { value: "remotion", label: "Remotion" },
];

export function RenderEnginePicker({ value, onChange }: RenderEnginePickerProps) {
  return (
    <div className={`${glass.card} ${styles.card}`} data-testid="render-engine-picker">
      <div className={styles.text}>
        <div className={glass.cardTitle}>Công cụ render</div>
        <p className={styles.hint}>
          {value === "remotion"
            ? "Remotion còn ở giai đoạn đầu — chưa có design system/lint riêng, cần viết code Remotion (.tsx)."
            : "Mặc định — có đầy đủ design system, lint, kiểm tra chồng lấn hình ảnh."}
        </p>
      </div>

      <div className={selectable.row} role="radiogroup" aria-label="Công cụ render">
        {OPTIONS.map((option) => (
          <SelectableOption
            key={option.value}
            selected={value === option.value}
            onSelect={() => onChange(option.value)}
            label={option.label}
            inline
            testId={`render-engine-${option.value}`}
          />
        ))}
      </div>
    </div>
  );
}
