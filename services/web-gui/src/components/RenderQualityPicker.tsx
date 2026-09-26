import type { RenderQuality } from "../context/ProjectDraftContext";
import { SelectableOption } from "./SelectableOption";
import glass from "../styles/glass.module.css";
import selectable from "../styles/selectable.module.css";

interface RenderQualityPickerProps {
  value: RenderQuality;
  onChange: (quality: RenderQuality) => void;
}

/**
 * CR-004 FR12.6. The presets are framed around what the Creator is doing
 * rather than the numbers: a draft pass exists to check that the content
 * works, and only the upload pass needs to be worth publishing.
 */
const OPTIONS: { value: RenderQuality; label: string; hint: string }[] = [
  { value: "480p15", label: "Test", hint: "480p15 · Nhanh nhất, chỉ để kiểm tra nội dung" },
  { value: "720p30", label: "Nháp", hint: "720p30 · Xem thử, chưa nên đăng" },
  { value: "1080p60", label: "Chuẩn", hint: "1080p60 · Khuyên dùng khi đăng YouTube" },
  { value: "4k60", label: "Cao", hint: "4K60 · Rất nặng, chỉ dùng khi thật sự cần" },
];

/**
 * No outer `glass.card` here (UX review #2 nesting fix) — both call sites
 * (SettingsStepPage, ResultPage's rerender section) now render this inside a
 * `Disclosure`, which already provides the card; wrapping again produced a
 * visible card-inside-a-card.
 */
export function RenderQualityPicker({ value, onChange }: RenderQualityPickerProps) {
  return (
    <div data-testid="render-quality-picker">
      <div className={glass.cardTitle} style={{ marginBottom: 10 }}>
        Chất lượng video
      </div>

      <div className={selectable.stack} role="radiogroup" aria-label="Chất lượng video">
        {OPTIONS.map((option) => (
          <SelectableOption
            key={option.value}
            selected={value === option.value}
            onSelect={() => onChange(option.value)}
            label={option.label}
            hint={option.hint}
            testId={`render-quality-${option.value}`}
          />
        ))}
      </div>

      {(value === "480p15" || value === "720p30") && (
        <p className={glass.helperText} style={{ marginRight: 0, marginTop: 10 }} role="status">
          Không nên đăng bản nháp vì YouTube sẽ nén lại, làm giảm chất lượng.
        </p>
      )}
      {value === "4k60" && (
        <p className={glass.helperText} style={{ marginRight: 0, marginTop: 10 }} role="status">
          4K mất nhiều thời gian hơn 1080p đáng kể và ít khi cần thiết.
        </p>
      )}
    </div>
  );
}
