import type { RenderEngine } from "../context/ProjectDraftContext";
import { SelectableOption } from "./SelectableOption";
import glass from "../styles/glass.module.css";
import selectable from "../styles/selectable.module.css";

interface RenderEnginePickerProps {
  value: RenderEngine;
  onChange: (engine: RenderEngine) => void;
}

/**
 * feature/remotion-engine — which engine renders the pasted script. Manim
 * stays the default and the only engine with a design system, lint, and
 * overlap-detection so far; Remotion is a minimal first cut (plain centered
 * text primitives, script pasted directly, no wizard prompts yet).
 */
const OPTIONS: { value: RenderEngine; label: string; hint: string }[] = [
  { value: "manim", label: "Manim", hint: "Mặc định — có đầy đủ design system, lint, kiểm tra chồng lấn hình ảnh" },
  { value: "remotion", label: "Remotion", hint: "Mới — chỉ có component chữ tối giản, chưa có lint hay design system riêng" },
];

export function RenderEnginePicker({ value, onChange }: RenderEnginePickerProps) {
  return (
    <div data-testid="render-engine-picker">
      <div className={glass.cardTitle} style={{ marginBottom: 10 }}>
        Công cụ render
      </div>

      <div className={selectable.stack} role="radiogroup" aria-label="Công cụ render">
        {OPTIONS.map((option) => (
          <SelectableOption
            key={option.value}
            selected={value === option.value}
            onSelect={() => onChange(option.value)}
            label={option.label}
            hint={option.hint}
            testId={`render-engine-${option.value}`}
          />
        ))}
      </div>

      {value === "remotion" && (
        <p className={glass.helperText} style={{ marginRight: 0, marginTop: 10 }} role="status">
          Remotion còn ở giai đoạn đầu — script Manim dán sẵn sẽ không chạy được, cần viết code Remotion (.tsx) riêng.
        </p>
      )}
    </div>
  );
}
