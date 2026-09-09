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
  { value: "720p30", label: "Nháp nhanh", hint: "720p30 — render nhanh nhất, để duyệt nội dung" },
  { value: "1080p60", label: "Chuẩn", hint: "1080p60 — mức nên dùng khi đăng YouTube" },
  { value: "4k60", label: "Cao", hint: "4K60 — rất nặng, chỉ dùng khi thật sự cần" },
];

export function RenderQualityPicker({ value, onChange }: RenderQualityPickerProps) {
  return (
    <div className={glass.card} style={{ padding: 22 }} data-testid="render-quality-picker">
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

      {value === "720p30" && (
        <p className={glass.helperText} style={{ marginRight: 0, marginTop: 10 }} role="status">
          Bản nháp không nên dùng để đăng — YouTube sẽ nén lại một lần nữa, nên nguồn cần dư chất lượng.
        </p>
      )}
      {value === "4k60" && (
        <p className={glass.helperText} style={{ marginRight: 0, marginTop: 10 }} role="status">
          Render 4K nặng hơn 1080p nhiều lần và hiếm khi tăng lượt xem — cân nhắc kỹ trước khi dùng.
        </p>
      )}
    </div>
  );
}
